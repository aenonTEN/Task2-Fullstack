package httpserver

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"eaglepoint/backend/internal/persistence"
)

func getBcryptCost() int {
	return persistence.GetBcryptCost()
}

func getTokenTTL() time.Duration {
	ttlStr := os.Getenv("TOKEN_TTL_HOURS")
	ttl, err := strconv.Atoi(ttlStr)
	if err != nil || ttl <= 0 {
		return 8 * time.Hour
	}
	return time.Duration(ttl) * time.Hour
}

type authHandler struct {
	store *persistence.Store
}

type dataScope struct {
	InstitutionID string `json:"institutionId"`
	DepartmentID  string `json:"departmentId"`
	TeamID        string `json:"teamId"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
}

type errorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	TraceID string      `json:"traceId"`
}

func newAuthHandler(store *persistence.Store) *authHandler {
	return &authHandler{store: store}
}

func (h *authHandler) writeError(c *gin.Context, status int, code string, message string, details interface{}) {
	traceID := c.GetString("traceId")
	c.JSON(status, errorResponse{
		Code:    code,
		Message: message,
		Details: details,
		TraceID: traceID,
	})
}

func (h *authHandler) ensureTraceID(c *gin.Context) string {
	if existing := c.GetString("traceId"); existing != "" {
		return existing
	}
	id := uuid.NewString()
	c.Set("traceId", id)
	return id
}

func (h *authHandler) Login(c *gin.Context) {
	h.ensureTraceID(c)

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeError(c, http.StatusBadRequest, "validation_error", "Invalid request body.", gin.H{"field": "body"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Password) < 8 {
		h.writeError(c, http.StatusBadRequest, "validation_error", "Invalid username or password format.", gin.H{"minPasswordLength": 8})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	user, ok, err := h.store.FindUserByUsername(ctx, req.Username)
	if err != nil {
		h.writeError(c, http.StatusInternalServerError, "internal_error", "Failed to authenticate.", nil)
		return
	}
	if !ok || bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(req.Password)) != nil {
		h.writeError(c, http.StatusUnauthorized, "authentication_error", "Invalid credentials.", nil)
		return
	}

	token := uuid.NewString()
	expiresAt := time.Now().UTC().Add(getTokenTTL())
	if err := h.store.CreateSession(ctx, token, user.UserID, expiresAt); err != nil {
		h.writeError(c, http.StatusInternalServerError, "internal_error", "Failed to create session.", nil)
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		AccessToken: token,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	})

	// Make actor available for audit middleware (login is unauthenticated at start).
	c.Set("userId", user.UserID)
}

func (h *authHandler) RequireAuth(c *gin.Context) {
	h.ensureTraceID(c)

	authz := c.GetHeader("Authorization")
	if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
		h.writeError(c, http.StatusUnauthorized, "authentication_error", "Missing bearer token.", nil)
		c.Abort()
		return
	}

	token := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	sess, ok, err := h.store.GetSession(ctx, token)
	if err != nil {
		h.writeError(c, http.StatusInternalServerError, "internal_error", "Failed to validate session.", nil)
		c.Abort()
		return
	}
	if !ok || sess.RevokedAt != nil || time.Now().UTC().After(sess.ExpiresAt) {
		h.writeError(c, http.StatusUnauthorized, "authentication_error", "Invalid or expired session.", nil)
		c.Abort()
		return
	}

	c.Set("userId", sess.UserID)
	u, ok, err := h.store.FindUserByID(ctx, sess.UserID)
	if err != nil || !ok {
		h.writeError(c, http.StatusUnauthorized, "authentication_error", "Invalid session user.", nil)
		c.Abort()
		return
	}
	c.Set("roleIds", u.RoleIDs)
	c.Set("scope", dataScope{InstitutionID: u.Scope.InstitutionID, DepartmentID: u.Scope.DepartmentID, TeamID: u.Scope.TeamID})
	c.Next()
}

func (h *authHandler) Logout(c *gin.Context) {
	h.ensureTraceID(c)

	authz := c.GetHeader("Authorization")
	token := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	_ = h.store.RevokeSession(ctx, token, time.Now().UTC())

	c.Status(http.StatusNoContent)
}
