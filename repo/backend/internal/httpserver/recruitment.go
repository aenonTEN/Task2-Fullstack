package httpserver

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func getAESKey() []byte {
	keyHex := os.Getenv("AES_KEY_HEX")
	if keyHex == "" {
		key := make([]byte, 32)
		rand.Read(key)
		return key
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		key := make([]byte, 32)
		rand.Read(key)
		return key
	}
	return key
}

func encryptData(text string) string {
	if text == "" {
		return ""
	}
	key := getAESKey()
	c, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(c)
	nonce := make([]byte, 12)
	rand.Read(nonce)
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(text), nil))
}

func decryptData(cryptoText string) string {
	if cryptoText == "" {
		return ""
	}
	data, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return cryptoText
	}
	key := getAESKey()
	c, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(c)
	nonceSize := 12
	if len(data) < nonceSize {
		return cryptoText
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return cryptoText
	}
	return string(plaintext)
}

type candidateCreateRequest struct {
	Name            string   `json:"name"`
	Phone           string   `json:"phone"`
	IDNumber        string   `json:"idNumber"`
	Education       string   `json:"education"`
	ExperienceYears int      `json:"experienceYears"`
	Skills          []string `json:"skills"`
}

type candidateListItem struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	PhoneMasked     string   `json:"phoneMasked"`
	Education       string   `json:"education"`
	ExperienceYears int      `json:"experienceYears"`
	Skills          []string `json:"skills"`
	CreatedAt       string   `json:"createdAt"`
}

type candidateDetailItem struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	PhoneMasked     string   `json:"phoneMasked"`
	IDNumber        string   `json:"idNumber"`
	Education       string   `json:"education"`
	ExperienceYears int      `json:"experienceYears"`
	Skills          []string `json:"skills"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt,omitempty"`
}

type candidateMergeResult struct {
	IsNew        bool     `json:"isNew"`
	CandidateID  string   `json:"candidateId"`
	MergedFromID string   `json:"mergedFromId,omitempty"`
	Conflicts    []string `json:"conflicts,omitempty"`
}

func maskPhone(phone string) string {
	p := strings.TrimSpace(phone)
	if len(p) < 4 {
		return "***"
	}
	return "***" + p[len(p)-4:]
}

func getScope(c *gin.Context) (dataScope, bool) {
	scopeAny, ok := c.Get("scope")
	if !ok {
		return dataScope{}, false
	}
	scope, ok := scopeAny.(dataScope)
	return scope, ok
}

func recruitmentCreateCandidate(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req candidateCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Name is required.", TraceID: c.GetString("traceId")})
			return
		}
		if req.ExperienceYears < 0 || req.ExperienceYears > 80 {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid experienceYears.", TraceID: c.GetString("traceId")})
			return
		}
		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		phone := strings.TrimSpace(req.Phone)
		idNumber := strings.TrimSpace(req.IDNumber)
		encryptedPhone := encryptData(phone)
		encryptedID := encryptData(idNumber)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to start transaction.", TraceID: c.GetString("traceId")})
			return
		}
		defer tx.Rollback()

		duplicateFound := false
		mergeCandidateID := ""
		mergeConflicts := make([]string, 0)

		if phone != "" {
			err := tx.QueryRowContext(ctx, `
				SELECT id FROM candidates
				WHERE institution_id = ? AND phone = ? AND deleted_at IS NULL
				LIMIT 1
			`, scope.InstitutionID, encryptedPhone).Scan(&mergeCandidateID)
			if err == nil {
				duplicateFound = true
				mergeConflicts = append(mergeConflicts, "phone")
			}
		}

		if !duplicateFound && idNumber != "" {
			err := tx.QueryRowContext(ctx, `
				SELECT id FROM candidates
				WHERE institution_id = ? AND id_number = ? AND deleted_at IS NULL
				LIMIT 1
			`, scope.InstitutionID, encryptedID).Scan(&mergeCandidateID)
			if err == nil {
				duplicateFound = true
				mergeConflicts = append(mergeConflicts, "id_number")
			}
		}

		if duplicateFound {
			var existingSkillsJSON string
			err := tx.QueryRowContext(ctx, "SELECT skills_json FROM candidates WHERE id = ?", mergeCandidateID).Scan(&existingSkillsJSON)
			if err == nil {
				var existingSkills []string
				_ = json.Unmarshal([]byte(existingSkillsJSON), &existingSkills)
				skillSet := make(map[string]bool)
				for _, s := range existingSkills {
					skillSet[s] = true
				}
				for _, s := range req.Skills {
					if !skillSet[s] {
						existingSkills = append(existingSkills, s)
						skillSet[s] = true
					}
				}
				newSkillsJSON, _ := json.Marshal(existingSkills)
				_, _ = tx.ExecContext(ctx, `
					UPDATE candidates SET education = ?, experience_years = ?, skills_json = ?, updated_at = ?
					WHERE id = ?
				`, strings.TrimSpace(req.Education), req.ExperienceYears, string(newSkillsJSON), time.Now().UTC(), mergeCandidateID)

				auditRec := auditRecord{
					ID:        uuid.NewString(),
					TraceID:   c.GetString("traceId"),
					ActorID:   c.GetString("userId"),
					Action:    "MERGE_UPDATE",
					Entity:    "Candidate",
					EntityID:  mergeCandidateID,
					After:     gin.H{"mergedFields": mergeConflicts},
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
				}
				_ = store.AppendAudit(auditRec)
			}

			if err := tx.Commit(); err != nil {
				c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to commit transaction.", TraceID: c.GetString("traceId")})
				return
			}

			result := candidateMergeResult{
				IsNew:        false,
				CandidateID:  mergeCandidateID,
				MergedFromID: "",
				Conflicts:    mergeConflicts,
			}
			c.JSON(http.StatusOK, result)
			return
		}

		skillsJSON, _ := json.Marshal(req.Skills)
		id := uuid.NewString()
		now := time.Now().UTC()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO candidates (id, institution_id, name, phone, phone_masked, id_number, education, experience_years, skills_json, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, scope.InstitutionID, req.Name, encryptedPhone, maskPhone(phone), encryptedID, strings.TrimSpace(req.Education), req.ExperienceYears, string(skillsJSON), now, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to create candidate.", TraceID: c.GetString("traceId")})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to commit transaction.", TraceID: c.GetString("traceId")})
			return
		}

		auditRec := auditRecord{
			ID:       uuid.NewString(),
			TraceID:  c.GetString("traceId"),
			ActorID:  c.GetString("userId"),
			Action:   "CREATE",
			Entity:   "Candidate",
			EntityID: id,
			Before:   nil,
			After: candidateDetailItem{
				ID:              id,
				Name:            req.Name,
				PhoneMasked:     maskPhone(phone),
				IDNumber:        "***ENCRYPTED***",
				Education:       strings.TrimSpace(req.Education),
				ExperienceYears: req.ExperienceYears,
				Skills:          req.Skills,
				CreatedAt:       now.Format(time.RFC3339),
			},
			CreatedAt: now.Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		c.JSON(http.StatusCreated, candidateMergeResult{IsNew: true, CandidateID: id})
	}
}

func recruitmentListCandidates(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		rows, err := db.QueryContext(ctx, `
			SELECT id, name, phone_masked, education, experience_years, skills_json, created_at
			FROM candidates
			WHERE institution_id = ? AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT 100
		`, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to load candidates.", TraceID: c.GetString("traceId")})
			return
		}
		defer rows.Close()

		out := make([]candidateListItem, 0, 32)
		for rows.Next() {
			var item candidateListItem
			var skillsRaw string
			var created time.Time
			if err := rows.Scan(&item.ID, &item.Name, &item.PhoneMasked, &item.Education, &item.ExperienceYears, &skillsRaw, &created); err != nil {
				c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to parse candidates.", TraceID: c.GetString("traceId")})
				return
			}
			_ = json.Unmarshal([]byte(skillsRaw), &item.Skills)
			item.CreatedAt = created.UTC().Format(time.RFC3339)
			out = append(out, item)
		}

		c.JSON(http.StatusOK, out)
	}
}

type matchResult struct {
	CandidateID string   `json:"candidateId"`
	Score       int      `json:"score"`
	Reasons     []string `json:"reasons"`
}

func recruitmentSearch(db *sql.DB) gin.HandlerFunc {
	// Explainable scoring baseline (from questions.md): skills 50, experience 30, education 20.
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
		skill := strings.ToLower(strings.TrimSpace(c.Query("skill")))
		education := strings.ToLower(strings.TrimSpace(c.Query("education")))
		// For scaffold: scan recent candidates and compute a simple score.
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		rows, err := db.QueryContext(ctx, `
			SELECT id, name, education, experience_years, skills_json
			FROM candidates
			WHERE institution_id = ? AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT 200
		`, scope.InstitutionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Search failed.", TraceID: c.GetString("traceId")})
			return
		}
		defer rows.Close()

		results := make([]matchResult, 0, 50)
		scoringWarning := ""
		for rows.Next() {
			var id, name, edu string
			var exp int
			var skillsRaw string
			if err := rows.Scan(&id, &name, &edu, &exp, &skillsRaw); err != nil {
				continue
			}
			// keyword filter
			if keyword != "" && !strings.Contains(strings.ToLower(name), keyword) {
				continue
			}
			var skills []string
			_ = json.Unmarshal([]byte(skillsRaw), &skills)

			score, reasons := CalculateMatchScore(exp, edu, skills, skill, education)

			results = append(results, matchResult{
				CandidateID: id,
				Score:       score,
				Reasons:     reasons,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"results": results,
			"warning": scoringWarning,
		})
	}
}

type bulkImportRequest struct {
	Candidates     []candidateCreateRequest `json:"candidates"`
	IdempotencyKey string                   `json:"idempotencyKey,omitempty"`
}

type bulkImportResult struct {
	Total      int               `json:"total"`
	Created    int               `json:"created"`
	Duplicates int               `json:"duplicates"`
	Errors     []bulkImportError `json:"errors,omitempty"`
}

type bulkImportError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

func recruitmentBulkImport(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req bulkImportRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}

		if len(req.Candidates) == 0 {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "No candidates provided.", TraceID: c.GetString("traceId")})
			return
		}

		if len(req.Candidates) > 1000 {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Maximum 1000 candidates per import.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
		defer cancel()

		result := bulkImportResult{
			Total:   len(req.Candidates),
			Created: 0,
			Errors:  make([]bulkImportError, 0),
		}

		for i, cand := range req.Candidates {
			cand.Name = strings.TrimSpace(cand.Name)
			if cand.Name == "" {
				result.Errors = append(result.Errors, bulkImportError{Index: i, Error: "Name is required"})
				continue
			}

			phone := strings.TrimSpace(cand.Phone)
			idNumber := strings.TrimSpace(cand.IDNumber)

			encryptedPhone := encryptData(phone)
			encryptedID := encryptData(idNumber)

			var existingID string
			if phone != "" {
				_ = db.QueryRowContext(ctx, `
					SELECT id FROM candidates
					WHERE institution_id = ? AND phone = ? AND deleted_at IS NULL
					LIMIT 1
				`, scope.InstitutionID, encryptedPhone).Scan(&existingID)
			}
			if existingID == "" && idNumber != "" {
				_ = db.QueryRowContext(ctx, `
					SELECT id FROM candidates
					WHERE institution_id = ? AND id_number = ? AND deleted_at IS NULL
					LIMIT 1
				`, scope.InstitutionID, encryptedID).Scan(&existingID)
			}

			if existingID != "" {
				result.Duplicates++
				continue
			}

			skillsJSON, _ := json.Marshal(cand.Skills)
			id := uuid.NewString()
			now := time.Now().UTC()

			_, err := db.ExecContext(ctx, `
				INSERT INTO candidates (id, institution_id, name, phone, phone_masked, id_number, education, experience_years, skills_json, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, id, scope.InstitutionID, cand.Name, encryptedPhone, maskPhone(phone), encryptedID, strings.TrimSpace(cand.Education), cand.ExperienceYears, string(skillsJSON), now, now)
			if err != nil {
				result.Errors = append(result.Errors, bulkImportError{Index: i, Error: err.Error()})
				continue
			}

			result.Created++
		}

		if result.Created > 0 {
			auditRec := auditRecord{
				ID:        uuid.NewString(),
				TraceID:   c.GetString("traceId"),
				ActorID:   c.GetString("userId"),
				Action:    "BULK_IMPORT",
				Entity:    "Candidate",
				After:     gin.H{"total": result.Total, "created": result.Created, "duplicates": result.Duplicates},
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			}
			_ = store.AppendAudit(auditRec)
		}

		c.JSON(http.StatusOK, result)
	}
}
