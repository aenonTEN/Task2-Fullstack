package httpserver

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"eaglepoint/backend/internal/persistence"
)

type RouterDeps struct {
	DB *sql.DB
}

func NewRouterWithDeps(deps *RouterDeps) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	if deps == nil || deps.DB == nil {
		router.GET("/api/v1/health/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		return router
	}

	store := &persistence.Store{DB: deps.DB}
	audit := &sqlAuditWriter{store: store}

	v1 := router.Group("/api/v1")
	{
		v1.Use(complianceCheckExpiryOnArrival(store.DB))

		v1.GET("/health/ready", func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if err := store.DB.PingContext(ctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db_unavailable"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		auth := newAuthHandler(store)
		v1.POST("/auth/login", auditAppendMiddleware(audit, "LOGIN", "Session"), auth.Login)
		v1.POST("/auth/logout", auth.RequireAuth, auditAppendMiddleware(audit, "LOGOUT", "Session"), auth.Logout)

		v1.GET("/me", auth.RequireAuth, whoAmI)
		// Example protected route requiring admin role.
		v1.GET("/admin/ping", auth.RequireAuth, requireRole("role_admin"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "admin_ok"})
		})

		v1.GET("/audit/records", auth.RequireAuth, requireRole("role_admin"), auditListHandler(audit))

		// Recruitment: institution-scoped candidates. Write operations require role_admin.
		recruit := v1.Group("/recruitment")
		{
			recruit.POST("/candidates", auth.RequireAuth, requireRole("role_admin"), recruitmentCreateCandidate(store.DB, audit))
			recruit.POST("/bulk", auth.RequireAuth, requireRole("role_admin"), recruitmentBulkImport(store.DB, audit))
			recruit.GET("/candidates", auth.RequireAuth, recruitmentListCandidates(store.DB))
			recruit.GET("/search", auth.RequireAuth, recruitmentSearch(store.DB))
		}

		// Compliance: qualifications and restrictions. Write operations require role_admin.
		comp := v1.Group("/compliance")
		{
			comp.GET("/qualifications", auth.RequireAuth, complianceListQualifications(store.DB))
			comp.POST("/qualifications", auth.RequireAuth, requireRole("role_admin"), complianceCreateQualification(store.DB, audit))
			comp.POST("/qualifications/expire", auth.RequireAuth, requireRole("role_admin"), complianceExpireQualifications(store.DB, audit))
			comp.POST("/qualifications/reactivate", auth.RequireAuth, requireRole("role_admin"), complianceReactivateQualification(store.DB, audit))
			comp.POST("/restrictions/check", auth.RequireAuth, complianceCheckRestriction(store.DB))
			comp.POST("/restrictions", auth.RequireAuth, requireRole("role_admin"), complianceApplyRestriction(store.DB, audit))
		}

		// Case Ledger: numbering, dedupe, workflow. Write operations require role_admin.
		cases := v1.Group("/cases")
		{
			cases.GET("", auth.RequireAuth, caseLedgerListCases(store.DB))
			cases.POST("", auth.RequireAuth, requireRole("role_admin"), caseLedgerCreateCase(store.DB, audit))
			cases.PATCH("/:id/status", auth.RequireAuth, requireRole("role_admin"), caseLedgerUpdateStatus(store.DB, audit))
			cases.POST("/:id/assign", auth.RequireAuth, requireRole("role_admin"), caseLedgerAssign(store.DB, audit))
			cases.GET("/:id/history", auth.RequireAuth, caseLedgerHistory(store.DB))
			cases.GET("/:id/attachments", auth.RequireAuth, attachmentListByCase(store.DB))
		}

		// Attachments: chunk upload, SHA256 dedup. Write operations require role_admin.
		attachments := v1.Group("/attachments")
		{
			attachments.POST("/init", auth.RequireAuth, requireRole("role_admin"), attachmentInitUpload(store.DB))
			attachments.POST("/:uploadId/chunk", auth.RequireAuth, requireRole("role_admin"), attachmentUploadChunk(store.DB))
			attachments.POST("/complete", auth.RequireAuth, requireRole("role_admin"), attachmentCompleteUpload(store.DB, audit))
			attachments.GET("/:id/download", auth.RequireAuth, attachmentGetDownload(store.DB))
		}

		// Positions and Qualification Profiles. Write operations require role_admin.
		positions := v1.Group("/positions")
		{
			positions.GET("", auth.RequireAuth, positionList(store.DB))
			positions.POST("", auth.RequireAuth, requireRole("role_admin"), positionCreate(store.DB, audit))
			positions.POST("/:id/close", auth.RequireAuth, requireRole("role_admin"), positionClose(store.DB, audit))
		}

		// Qualification Profiles. Write operations require role_admin.
		profiles := v1.Group("/profiles")
		{
			profiles.GET("/qualifications", auth.RequireAuth, qualificationProfileList(store.DB))
			profiles.POST("/qualifications", auth.RequireAuth, requireRole("role_admin"), qualificationProfileCreate(store.DB, audit))
		}

		// Tags: configurable tags for candidates and cases. Write operations require role_admin.
		tags := v1.Group("/tags")
		{
			tags.GET("", auth.RequireAuth, tagList(store.DB))
			tags.POST("", auth.RequireAuth, requireRole("role_admin"), tagCreate(store.DB))
			tags.DELETE("/:id", auth.RequireAuth, requireRole("role_admin"), tagDelete(store.DB))
			tags.POST("/assign", auth.RequireAuth, requireRole("role_admin"), tagAssign(store.DB))
			tags.GET("/entity", auth.RequireAuth, tagGetByEntity(store.DB))
		}
	}

	return router
}

func NewRouter() *gin.Engine {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:root@tcp(db:3306)/eaglepoint?parseTime=true"
	}
	db, err := persistence.OpenMySQL(dsn)
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := persistence.EnsureSchema(ctx, db); err != nil {
		panic("failed to ensure schema: " + err.Error())
	}
	return NewRouterWithDeps(&RouterDeps{DB: db})
}
