package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type caseCreateRequest struct {
	CandidateID string `json:"candidateId"`
	CaseType    string `json:"caseType"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
}

type caseItem struct {
	ID          string `json:"id"`
	CaseNumber  string `json:"caseNumber"`
	CandidateID string `json:"candidateId"`
	CaseType    string `json:"caseType"`
	Status      string `json:"status"`
	Subject     string `json:"subject"`
	CreatedBy   string `json:"createdBy"`
	AssignedTo  string `json:"assignedTo,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

func generateCaseNumber(db *sql.DB, institutionID string) (string, error) {
	now := time.Now().UTC()
	datePrefix := now.Format("20060102")

	var maxSeq int
	err := db.QueryRowContext(context.Background(), `
		SELECT COALESCE(MAX(CAST(RIGHT(case_number, 6) AS UNSIGNED)), 0)
		FROM cases
		WHERE institution_id = ? AND DATE(created_at) = ?
	`, institutionID, now.Format("2006-01-02")).Scan(&maxSeq)
	if err != nil {
		return "", err
	}

	seq := maxSeq + 1
	return fmt.Sprintf("%s-%s-%06d", datePrefix, institutionID, seq), nil
}

func checkCaseDuplicateWindow(db *sql.DB, scope dataScope, candidateID, caseType, subject string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	windowStart := time.Now().UTC().Add(-5 * time.Minute)
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM cases
		WHERE candidate_id = ? AND case_type = ? AND subject = ? 
		AND institution_id = ? AND created_at > ?
	`, candidateID, caseType, subject, scope.InstitutionID, windowStart).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func caseLedgerListCases(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		rows, err := db.QueryContext(ctx, `
			SELECT id, case_number, candidate_id, case_type, status, subject, created_by, assigned_to, created_at, updated_at
			FROM cases
			WHERE institution_id = ? AND closed_at IS NULL
			ORDER BY created_at DESC
			LIMIT 100
		`, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to load cases.", TraceID: c.GetString("traceId")})
			return
		}
		defer rows.Close()

		out := make([]caseItem, 0, 32)
		for rows.Next() {
			var item caseItem
			var created, updated sql.NullTime
			var assignedTo sql.NullString
			if err := rows.Scan(&item.ID, &item.CaseNumber, &item.CandidateID, &item.CaseType, &item.Status, &item.Subject, &item.CreatedBy, &assignedTo, &created, &updated); err != nil {
				continue
			}
			if created.Valid {
				item.CreatedAt = created.Time.UTC().Format(time.RFC3339)
			}
			if updated.Valid {
				item.UpdatedAt = updated.Time.UTC().Format(time.RFC3339)
			}
			if assignedTo.Valid {
				item.AssignedTo = assignedTo.String
			}
			out = append(out, item)
		}

		c.JSON(http.StatusOK, out)
	}
}

func caseLedgerCreateCase(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req caseCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}
		req.CaseType = strings.TrimSpace(req.CaseType)
		req.Subject = strings.TrimSpace(req.Subject)
		if req.CaseType == "" || req.Subject == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "CaseType and Subject are required.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		isDup, err := checkCaseDuplicateWindow(db, scope, req.CandidateID, req.CaseType, req.Subject)
		if err == nil && isDup {
			c.JSON(http.StatusConflict, errorResponse{Code: "duplicate_case", Message: "A similar case was created within the last 5 minutes.", TraceID: c.GetString("traceId")})
			return
		}

		caseNumber, err := generateCaseNumber(db, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to generate case number.", TraceID: c.GetString("traceId")})
			return
		}

		userID := c.GetString("userId")
		id := uuid.NewString()
		now := time.Now().UTC()

		_, err = db.ExecContext(ctx, `
			INSERT INTO cases (id, case_number, institution_id, candidate_id, case_type, status, subject, description, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?)
		`, id, caseNumber, scope.InstitutionID, req.CandidateID, req.CaseType, req.Subject, req.Description, userID, now, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to create case.", TraceID: c.GetString("traceId")})
			return
		}

		auditRec := auditRecord{
			ID:       uuid.NewString(),
			TraceID:  c.GetString("traceId"),
			ActorID:  userID,
			Action:   "CREATE",
			Entity:   "Case",
			EntityID: id,
			After: caseItem{
				ID:          id,
				CaseNumber:  caseNumber,
				CandidateID: req.CandidateID,
				CaseType:    req.CaseType,
				Status:      "pending",
				Subject:     req.Subject,
				CreatedBy:   userID,
				CreatedAt:   now.Format(time.RFC3339),
			},
			CreatedAt: now.Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		c.JSON(http.StatusCreated, caseItem{
			ID:          id,
			CaseNumber:  caseNumber,
			CandidateID: req.CandidateID,
			CaseType:    req.CaseType,
			Status:      "pending",
			Subject:     req.Subject,
			CreatedBy:   userID,
			CreatedAt:   now.Format(time.RFC3339),
		})
	}
}

func caseLedgerUpdateStatus(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		caseID := c.Param("id")
		var req struct {
			Status string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}

		validStatuses := map[string]bool{"pending": true, "in_progress": true, "resolved": true, "closed": true}
		if !validStatuses[req.Status] {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid status.", TraceID: c.GetString("traceId")})
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
		var closedAt interface{}
		if req.Status == "closed" {
			closedAt = now
		}

		result, err := db.ExecContext(ctx, `
			UPDATE cases SET status = ?, updated_at = ?, closed_at = ? 
			WHERE id = ? AND institution_id = ?
		`, req.Status, now, closedAt, caseID, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to update case.", TraceID: c.GetString("traceId")})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, errorResponse{Code: "not_found", Message: "Case not found.", TraceID: c.GetString("traceId")})
			return
		}

		userID := c.GetString("userId")
		auditRec := auditRecord{
			ID:        uuid.NewString(),
			TraceID:   c.GetString("traceId"),
			ActorID:   userID,
			Action:    "UPDATE_STATUS",
			Entity:    "Case",
			EntityID:  caseID,
			After:     gin.H{"status": req.Status},
			CreatedAt: now.Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		historyRec := auditRecord{
			ID:        uuid.NewString(),
			TraceID:   c.GetString("traceId"),
			ActorID:   userID,
			Action:    "STATUS_CHANGE",
			Entity:    "CaseHistory",
			EntityID:  uuid.NewString(),
			After:     gin.H{"caseId": caseID, "newStatus": req.Status},
			CreatedAt: now.Format(time.RFC3339),
		}
		_ = store.AppendAudit(historyRec)

		c.JSON(http.StatusOK, gin.H{"id": caseID, "status": req.Status})
	}
}

type caseAssignRequest struct {
	AssignedTo string `json:"assignedTo"`
	Note       string `json:"note,omitempty"`
}

func caseLedgerAssign(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		caseID := c.Param("id")
		var req caseAssignRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}
		req.AssignedTo = strings.TrimSpace(req.AssignedTo)
		if req.AssignedTo == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "AssignedTo is required.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		var currentAssignedTo sql.NullString
		err := db.QueryRowContext(ctx, `SELECT assigned_to FROM cases WHERE id = ? AND institution_id = ?`, caseID, scope.InstitutionID).Scan(&currentAssignedTo)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse{Code: "not_found", Message: "Case not found.", TraceID: c.GetString("traceId")})
			return
		}

		now := time.Now().UTC()
		userID := c.GetString("userId")

		_, err = db.ExecContext(ctx, `
			UPDATE cases SET assigned_to = ?, updated_at = ?
			WHERE id = ? AND institution_id = ?
		`, req.AssignedTo, now, caseID, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to assign case.", TraceID: c.GetString("traceId")})
			return
		}

		details := gin.H{
			"caseId":           caseID,
			"previousAssignee": currentAssignedTo.String,
			"newAssignee":      req.AssignedTo,
		}
		if req.Note != "" {
			details["note"] = req.Note
		}

		auditRec := auditRecord{
			ID:        uuid.NewString(),
			TraceID:   c.GetString("traceId"),
			ActorID:   userID,
			Action:    "ASSIGN",
			Entity:    "Case",
			EntityID:  caseID,
			After:     details,
			CreatedAt: now.Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		c.JSON(http.StatusOK, gin.H{"id": caseID, "assignedTo": req.AssignedTo})
	}
}

type caseHistoryItem struct {
	ID        string `json:"id"`
	CaseID    string `json:"caseId"`
	Action    string `json:"action"`
	ActorID   string `json:"actorId"`
	Details   gin.H  `json:"details"`
	CreatedAt string `json:"createdAt"`
}

func caseLedgerHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		caseID := c.Param("id")
		if caseID == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Case ID required.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		rows, err := db.QueryContext(ctx, `
			SELECT id, case_id, action, actor_id, details, created_at
			FROM case_history
			WHERE case_id = ?
			ORDER BY created_at DESC
		`, caseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to load history.", TraceID: c.GetString("traceId")})
			return
		}
		defer rows.Close()

		out := make([]caseHistoryItem, 0, 32)
		for rows.Next() {
			var item caseHistoryItem
			var detailsJSON string
			var created time.Time
			if err := rows.Scan(&item.ID, &item.CaseID, &item.Action, &item.ActorID, &detailsJSON, &created); err != nil {
				continue
			}
			_ = json.Unmarshal([]byte(detailsJSON), &item.Details)
			item.CreatedAt = created.UTC().Format(time.RFC3339)
			out = append(out, item)
		}

		c.JSON(http.StatusOK, out)
	}
}
