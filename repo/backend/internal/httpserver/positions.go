package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func parseJSONArray(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}

type positionCreateRequest struct {
	Title        string `json:"title"`
	Department   string `json:"department"`
	Description  string `json:"description"`
	Requirements string `json:"requirements"`
}

type positionItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Department   string `json:"department"`
	Description  string `json:"description"`
	Requirements string `json:"requirements"`
	Status       string `json:"status"`
	CreatedBy    string `json:"createdBy"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
	ClosedAt     string `json:"closedAt,omitempty"`
}

type qualificationProfileCreateRequest struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	RequiredSkills     []string `json:"requiredSkills"`
	MinExperienceYears int      `json:"minExperienceYears"`
	RequiredEducation  string   `json:"requiredEducation"`
	ValidityMonths     int      `json:"validityMonths"`
}

type qualificationProfileItem struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	RequiredSkills     []string `json:"requiredSkills"`
	MinExperienceYears int      `json:"minExperienceYears"`
	RequiredEducation  string   `json:"requiredEducation"`
	ValidityMonths     int      `json:"validityMonths"`
	IsActive           bool     `json:"isActive"`
	CreatedAt          string   `json:"createdAt"`
}

func positionList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		status := c.Query("status")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		query := `
			SELECT id, title, department, description, requirements, status, created_by, created_at, updated_at, closed_at
			FROM positions
			WHERE institution_id = ?`
		args := []interface{}{scope.InstitutionID}

		if status != "" {
			query += " AND status = ?"
			args = append(args, status)
		} else {
			query += " AND status != 'closed'"
		}
		query += " ORDER BY created_at DESC LIMIT 100"

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to load positions.", TraceID: c.GetString("traceId")})
			return
		}
		defer rows.Close()

		out := make([]positionItem, 0, 32)
		for rows.Next() {
			var item positionItem
			var created, updated, closed sql.NullTime
			var createdBy string
			if err := rows.Scan(&item.ID, &item.Title, &item.Department, &item.Description, &item.Requirements, &item.Status, &createdBy, &created, &updated, &closed); err != nil {
				continue
			}
			item.CreatedBy = createdBy
			if created.Valid {
				item.CreatedAt = created.Time.UTC().Format(time.RFC3339)
			}
			if updated.Valid {
				item.UpdatedAt = updated.Time.UTC().Format(time.RFC3339)
			}
			if closed.Valid {
				item.ClosedAt = closed.Time.UTC().Format(time.RFC3339)
			}
			out = append(out, item)
		}

		c.JSON(http.StatusOK, out)
	}
}

func positionCreate(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req positionCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}
		req.Title = strings.TrimSpace(req.Title)
		if req.Title == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Title is required.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		id := uuid.NewString()
		userID := c.GetString("userId")
		now := time.Now().UTC()

		_, err := db.ExecContext(ctx, `
			INSERT INTO positions (id, institution_id, title, department, description, requirements, status, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 'open', ?, ?, ?)
		`, id, scope.InstitutionID, req.Title, req.Department, req.Description, req.Requirements, userID, now, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to create position.", TraceID: c.GetString("traceId")})
			return
		}

		auditRec := auditRecord{
			ID:        uuid.NewString(),
			TraceID:   c.GetString("traceId"),
			ActorID:   userID,
			Action:    "CREATE",
			Entity:    "Position",
			EntityID:  id,
			After:     positionItem{ID: id, Title: req.Title, Department: req.Department, Status: "open", CreatedBy: userID, CreatedAt: now.Format(time.RFC3339)},
			CreatedAt: now.Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		c.JSON(http.StatusCreated, positionItem{ID: id, Title: req.Title, Department: req.Department, Description: req.Description, Requirements: req.Requirements, Status: "open", CreatedBy: userID, CreatedAt: now.Format(time.RFC3339)})
	}
}

func positionClose(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		positionID := c.Param("id")
		if positionID == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Position ID required.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		now := time.Now().UTC()
		result, err := db.ExecContext(ctx, `
			UPDATE positions SET status = 'closed', closed_at = ?, updated_at = ?
			WHERE id = ? AND institution_id = ?
		`, now, now, positionID, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to close position.", TraceID: c.GetString("traceId")})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, errorResponse{Code: "not_found", Message: "Position not found.", TraceID: c.GetString("traceId")})
			return
		}

		userID := c.GetString("userId")
		auditRec := auditRecord{
			ID:        uuid.NewString(),
			TraceID:   c.GetString("traceId"),
			ActorID:   userID,
			Action:    "CLOSE",
			Entity:    "Position",
			EntityID:  positionID,
			After:     gin.H{"status": "closed", "closedAt": now.Format(time.RFC3339)},
			CreatedAt: now.Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		c.JSON(http.StatusOK, gin.H{"id": positionID, "status": "closed"})
	}
}

func qualificationProfileList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		rows, err := db.QueryContext(ctx, `
			SELECT id, name, description, required_skills, min_experience_years, required_education, validity_months, is_active, created_at
			FROM qualification_profiles
			WHERE institution_id = ? AND is_active = TRUE
			ORDER BY created_at DESC
		`, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to load qualification profiles.", TraceID: c.GetString("traceId")})
			return
		}
		defer rows.Close()

		out := make([]qualificationProfileItem, 0, 32)
		for rows.Next() {
			var item qualificationProfileItem
			var skillsJSON string
			var created time.Time
			if err := rows.Scan(&item.ID, &item.Name, &item.Description, &skillsJSON, &item.MinExperienceYears, &item.RequiredEducation, &item.ValidityMonths, &item.IsActive, &created); err != nil {
				continue
			}
			_ = parseJSONArray(skillsJSON, &item.RequiredSkills)
			item.CreatedAt = created.UTC().Format(time.RFC3339)
			out = append(out, item)
		}

		c.JSON(http.StatusOK, out)
	}
}

func qualificationProfileCreate(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req qualificationProfileCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Name is required.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		validityMonths := req.ValidityMonths
		if validityMonths <= 0 {
			validityMonths = 12
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		skillsJSON, _ := jsonMarshal(req.RequiredSkills)
		id := uuid.NewString()
		now := time.Now().UTC()

		_, err := db.ExecContext(ctx, `
			INSERT INTO qualification_profiles (id, institution_id, name, description, required_skills, min_experience_years, required_education, validity_months, is_active, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE, ?)
		`, id, scope.InstitutionID, req.Name, req.Description, string(skillsJSON), req.MinExperienceYears, req.RequiredEducation, validityMonths, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to create qualification profile.", TraceID: c.GetString("traceId")})
			return
		}

		userID := c.GetString("userId")
		auditRec := auditRecord{
			ID:        uuid.NewString(),
			TraceID:   c.GetString("traceId"),
			ActorID:   userID,
			Action:    "CREATE",
			Entity:    "QualificationProfile",
			EntityID:  id,
			After:     qualificationProfileItem{ID: id, Name: req.Name, Description: req.Description, RequiredSkills: req.RequiredSkills, MinExperienceYears: req.MinExperienceYears, RequiredEducation: req.RequiredEducation, ValidityMonths: validityMonths, IsActive: true, CreatedAt: now.Format(time.RFC3339)},
			CreatedAt: now.Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		c.JSON(http.StatusCreated, qualificationProfileItem{ID: id, Name: req.Name, Description: req.Description, RequiredSkills: req.RequiredSkills, MinExperienceYears: req.MinExperienceYears, RequiredEducation: req.RequiredEducation, ValidityMonths: validityMonths, IsActive: true, CreatedAt: now.Format(time.RFC3339)})
	}
}
