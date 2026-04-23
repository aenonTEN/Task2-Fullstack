package httpserver

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type qualificationCreateRequest struct {
	CandidateID string `json:"candidateId"`
	Name        string `json:"name"`
	IssuedDate  string `json:"issuedDate"`
	ExpiryDate  string `json:"expiryDate,omitempty"`
}

type qualificationItem struct {
	ID          string `json:"id"`
	CandidateID string `json:"candidateId"`
	Name        string `json:"name"`
	IssuedDate  string `json:"issuedDate"`
	ExpiryDate  string `json:"expiryDate,omitempty"`
	Status      string `json:"status"`
}

type restrictionCheckRequest struct {
	CandidateID string `json:"candidateId"`
}

type restrictionCheckResponse struct {
	Allowed        bool   `json:"allowed"`
	Reason         string `json:"reason,omitempty"`
	NextEligibleAt string `json:"nextEligibleAt,omitempty"`
}

func checkQualificationExpiry(expiryDate string) (isActive bool) {
	if expiryDate == "" {
		return true
	}
	t, err := time.Parse("2006-01-02", expiryDate)
	if err != nil {
		return true
	}
	return time.Now().UTC().Before(t)
}

func complianceListQualifications(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		rows, err := db.QueryContext(ctx, `
			SELECT q.id, q.candidate_id, q.name, q.issued_date, q.expiry_date, q.status
			FROM qualifications q
			WHERE q.institution_id = ? AND q.deleted_at IS NULL
			ORDER BY q.created_at DESC
			LIMIT 200
		`, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to load qualifications.", TraceID: c.GetString("traceId")})
			return
		}
		defer rows.Close()

		out := make([]qualificationItem, 0, 32)
		for rows.Next() {
			var item qualificationItem
			var issued, expiry sql.NullString
			if err := rows.Scan(&item.ID, &item.CandidateID, &item.Name, &issued, &expiry, &item.Status); err != nil {
				continue
			}
			if issued.Valid {
				item.IssuedDate = issued.String
			}
			if expiry.Valid {
				item.ExpiryDate = expiry.String
				if !checkQualificationExpiry(expiry.String) {
					item.Status = "expired"
				}
			}
			out = append(out, item)
		}

		c.JSON(http.StatusOK, out)
	}
}

func complianceCreateQualification(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req qualificationCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Name is required.", TraceID: c.GetString("traceId")})
			return
		}
		if req.IssuedDate == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "IssuedDate is required.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		var candidateExists int
		err := db.QueryRowContext(ctx, `
			SELECT 1 FROM candidates
			WHERE id = ? AND institution_id = ? AND deleted_at IS NULL
			LIMIT 1
		`, req.CandidateID, scope.InstitutionID).Scan(&candidateExists)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Candidate not found.", TraceID: c.GetString("traceId")})
			return
		}

		expiryDate := strings.TrimSpace(req.ExpiryDate)
		status := "active"
		if expiryDate != "" && !checkQualificationExpiry(expiryDate) {
			status = "expired"
		}

		id := uuid.NewString()
		now := time.Now().UTC()

		_, err = db.ExecContext(ctx, `
			INSERT INTO qualifications (id, candidate_id, institution_id, name, issued_date, expiry_date, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, req.CandidateID, scope.InstitutionID, req.Name, req.IssuedDate, expiryDate, status, now, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to create qualification.", TraceID: c.GetString("traceId")})
			return
		}

		auditRec := auditRecord{
			ID:       uuid.NewString(),
			TraceID:  c.GetString("traceId"),
			ActorID:  c.GetString("userId"),
			Action:   "CREATE",
			Entity:   "Qualification",
			EntityID: id,
			Before:   nil,
			After: qualificationItem{
				ID:          id,
				CandidateID: req.CandidateID,
				Name:        req.Name,
				IssuedDate:  req.IssuedDate,
				ExpiryDate:  expiryDate,
				Status:      status,
			},
			CreatedAt: now.Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func check168HourRestriction(db *sql.DB, scope dataScope, candidateID string) restrictionCheckResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var windowEnd sql.NullTime
	err := db.QueryRowContext(ctx, `
		SELECT purchase_window_end
		FROM restrictions
		WHERE candidate_id = ? AND institution_id = ? AND restriction_type = 'purchase_168h' AND is_active = TRUE
		AND purchase_window_end > ?
		ORDER BY purchase_window_end DESC
		LIMIT 1
	`, candidateID, scope.InstitutionID, time.Now().UTC()).Scan(&windowEnd)
	if err == nil && windowEnd.Valid {
		return restrictionCheckResponse{
			Allowed:        false,
			Reason:         "Purchase restriction active (rolling 168h window)",
			NextEligibleAt: windowEnd.Time.Format(time.RFC3339),
		}
	}

	return restrictionCheckResponse{Allowed: true}
}

func complianceCheckRestriction(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req restrictionCheckRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		result := check168HourRestriction(db, scope, req.CandidateID)
		c.JSON(http.StatusOK, result)
	}
}

func complianceApplyRestriction(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			CandidateID     string `json:"candidateId"`
			RestrictionType string `json:"restrictionType"`
			Reason          string `json:"reason"`
			WindowHours     int    `json:"windowHours"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		windowHours := req.WindowHours
		if windowHours <= 0 {
			windowHours = 168
		}

		windowStart := time.Now().UTC()
		windowEnd := windowStart.Add(time.Duration(windowHours) * time.Hour)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		id := uuid.NewString()
		_, err := db.ExecContext(ctx, `
			INSERT INTO restrictions (id, institution_id, candidate_id, restriction_type, reason, purchase_window_start, purchase_window_end, is_active)
			VALUES (?, ?, ?, ?, ?, ?, ?, TRUE)
		`, id, scope.InstitutionID, req.CandidateID, req.RestrictionType, req.Reason, windowStart, windowEnd)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to apply restriction.", TraceID: c.GetString("traceId")})
			return
		}

		auditRec := auditRecord{
			ID:       uuid.NewString(),
			TraceID:  c.GetString("traceId"),
			ActorID:  c.GetString("userId"),
			Action:   "CREATE",
			Entity:   "Restriction",
			EntityID: id,
			After: gin.H{
				"candidateId":     req.CandidateID,
				"restrictionType": req.RestrictionType,
				"reason":          req.Reason,
				"windowEnd":       windowEnd.Format(time.RFC3339),
			},
			CreatedAt: windowStart.Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func complianceExpireQualifications(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		now := time.Now().UTC().Format("2006-01-02")

		result, err := db.ExecContext(ctx, `
			UPDATE qualifications
			SET status = 'expired', updated_at = ?
			WHERE institution_id = ? AND expiry_date < ? AND status = 'active' AND deleted_at IS NULL
		`, now, scope.InstitutionID, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to expire qualifications.", TraceID: c.GetString("traceId")})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		userID := c.GetString("userId")
		auditRec := auditRecord{
			ID:        uuid.NewString(),
			TraceID:   c.GetString("traceId"),
			ActorID:   userID,
			Action:    "AUTO_EXPIRE",
			Entity:    "Qualification",
			After:     gin.H{"expiredCount": rowsAffected},
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		c.JSON(http.StatusOK, gin.H{"expiredCount": rowsAffected})
	}
}

func complianceCheckExpiryOnArrival(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		now := time.Now().UTC().Format("2006-01-02")

		result, err := db.ExecContext(ctx, `
			UPDATE qualifications
			SET status = 'expired', updated_at = ?
			WHERE institution_id = ? AND expiry_date < ? AND status = 'active' AND deleted_at IS NULL
		`, now, scope.InstitutionID, now)
		if err == nil {
			rowsAffected, _ := result.RowsAffected()
			if rowsAffected > 0 {
				c.Header("X-Qualifications-Expired", fmt.Sprintf("%d", rowsAffected))
			}
		}

		c.Next()
	}
}

type qualificationReactivateRequest struct {
	QualificationID string `json:"qualificationId"`
	NewExpiryDate   string `json:"newExpiryDate"`
	ApprovalNote    string `json:"approvalNote"`
}

func complianceReactivateQualification(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req qualificationReactivateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		var currentStatus string
		err := db.QueryRowContext(ctx, `
			SELECT status FROM qualifications
			WHERE id = ? AND institution_id = ? AND deleted_at IS NULL
		`, req.QualificationID, scope.InstitutionID).Scan(&currentStatus)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse{Code: "not_found", Message: "Qualification not found.", TraceID: c.GetString("traceId")})
			return
		}

		if currentStatus != "expired" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Only expired qualifications can be reactivated.", TraceID: c.GetString("traceId")})
			return
		}

		req.NewExpiryDate = strings.TrimSpace(req.NewExpiryDate)
		if req.NewExpiryDate == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "NewExpiryDate is required for reactivation.", TraceID: c.GetString("traceId")})
			return
		}

		now := time.Now().UTC()
		userID := c.GetString("userId")

		_, err = db.ExecContext(ctx, `
			UPDATE qualifications
			SET status = 'active', expiry_date = ?, updated_at = ?
			WHERE id = ? AND institution_id = ?
		`, req.NewExpiryDate, now, req.QualificationID, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to reactivate qualification.", TraceID: c.GetString("traceId")})
			return
		}

		auditRec := auditRecord{
			ID:        uuid.NewString(),
			TraceID:   c.GetString("traceId"),
			ActorID:   userID,
			Action:    "REACTIVATE",
			Entity:    "Qualification",
			EntityID:  req.QualificationID,
			After:     gin.H{"newExpiryDate": req.NewExpiryDate, "approvalNote": req.ApprovalNote},
			CreatedAt: now.Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		c.JSON(http.StatusOK, gin.H{"id": req.QualificationID, "status": "active", "newExpiryDate": req.NewExpiryDate})
	}
}
