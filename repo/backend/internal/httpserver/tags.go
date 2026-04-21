package httpserver

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type tagItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type tagCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

func tagList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse{Code: "unauthorized", Message: "Not authenticated"})
			return
		}

		rows, err := db.QueryContext(c.Request.Context(), `
			SELECT id, name, color, description FROM tags 
			WHERE institution_id = ? 
			ORDER BY name
		`, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to load tags", TraceID: c.GetString("traceId")})
			return
		}
		defer rows.Close()

		tags := make([]tagItem, 0, 16)
		for rows.Next() {
			var t tagItem
			if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.Description); err != nil {
				continue
			}
			tags = append(tags, t)
		}
		c.JSON(http.StatusOK, tags)
	}
}

func tagCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse{Code: "unauthorized", Message: "Not authenticated"})
			return
		}

		var req tagCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "invalid_request", Message: err.Error()})
			return
		}

		color := req.Color
		if color == "" {
			color = "#6b7280"
		}
		desc := req.Description
		if desc == "" {
			desc = ""
		}

		id := uuid.New().String()
		_, err := db.ExecContext(c.Request.Context(), `
			INSERT INTO tags (id, institution_id, name, color, description)
			VALUES (?, ?, ?, ?, ?)
		`, id, scope.InstitutionID, req.Name, color, desc)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to create tag", TraceID: c.GetString("traceId")})
			return
		}

		c.JSON(http.StatusCreated, tagItem{ID: id, Name: req.Name, Color: color, Description: desc})
	}
}

func tagDelete(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse{Code: "unauthorized", Message: "Not authenticated"})
			return
		}

		tagID := c.Param("id")
		if tagID == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "invalid_request", Message: "Tag ID required"})
			return
		}

		_, err := db.ExecContext(c.Request.Context(), `
			DELETE FROM tags WHERE id = ? AND institution_id = ?
		`, tagID, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to delete tag", TraceID: c.GetString("traceId")})
			return
		}

		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

type tagAssignRequest struct {
	EntityType string   `json:"entityType" binding:"required,oneof=candidate case"`
	EntityID   string   `json:"entityId" binding:"required"`
	TagIDs     []string `json:"tagIds" binding:"required"`
}

func tagAssign(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := getScope(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse{Code: "unauthorized", Message: "Not authenticated"})
			return
		}

		var req tagAssignRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "invalid_request", Message: err.Error()})
			return
		}

		tx, err := db.BeginTx(c.Request.Context(), nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to assign tags"})
			return
		}
		defer tx.Rollback()

		var table string
		if req.EntityType == "candidate" {
			table = "candidate_tags"
		} else {
			table = "case_tags"
		}

		_, err = tx.ExecContext(c.Request.Context(), "DELETE FROM "+table+" WHERE "+req.EntityType+"_id = ?", req.EntityID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to clear tags"})
			return
		}

		for _, tagID := range req.TagIDs {
			_, err = tx.ExecContext(c.Request.Context(), "INSERT INTO "+table+" ("+req.EntityType+"_id, tag_id) VALUES (?, ?)", req.EntityID, tagID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to assign tag"})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to assign tags"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"assigned": true})
	}
}

func tagGetByEntity(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse{Code: "unauthorized", Message: "Not authenticated"})
			return
		}

		entityType := c.Query("entityType")
		entityID := c.Query("entityId")
		if entityType == "" || entityID == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "invalid_request", Message: "entityType and entityId required"})
			return
		}

		var table string
		if entityType == "candidate" {
			table = "candidate_tags"
		} else if entityType == "case" {
			table = "case_tags"
		} else {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "invalid_request", Message: "Invalid entityType"})
			return
		}

		rows, err := db.QueryContext(c.Request.Context(), `
			SELECT t.id, t.name, t.color, t.description 
			FROM tags t
			JOIN `+table+` ct ON ct.tag_id = t.id
			WHERE ct.`+entityType+`_id = ? AND t.institution_id = ?
		`, entityID, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to load tags"})
			return
		}
		defer rows.Close()

		tags := make([]tagItem, 0, 8)
		for rows.Next() {
			var t tagItem
			if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.Description); err != nil {
				continue
			}
			tags = append(tags, t)
		}
		c.JSON(http.StatusOK, tags)
	}
}
