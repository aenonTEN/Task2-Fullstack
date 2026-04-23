package httpserver

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var allowedMimeTypes = map[string]bool{
	"application/pdf":    true,
	"image/jpeg":         true,
	"image/png":          true,
	"image/gif":          true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"text/plain": true,
}

type attachmentUploadInitRequest struct {
	CaseID      string `json:"caseId"`
	FileName    string `json:"fileName"`
	FileSize    int64  `json:"fileSize,omitempty"`
	MimeType    string `json:"mimeType"`
	TotalChunks int    `json:"totalChunks,omitempty"`
}

type attachmentUploadInitResponse struct {
	UploadID    string `json:"uploadId"`
	ChunkSize   int    `json:"chunkSize"`
	TotalChunks int    `json:"totalChunks"`
	UploadURL   string `json:"uploadUrl,omitempty"`
}

type attachmentChunkUploadRequest struct {
	UploadID   string `json:"uploadId"`
	ChunkData  string `json:"chunkData"`
	ChunkIndex int    `json:"chunkIndex"`
}

type attachmentCompleteRequest struct {
	UploadID string `json:"uploadId"`
}

type attachmentItem struct {
	ID         string `json:"id"`
	CaseID     string `json:"caseId"`
	FileName   string `json:"fileName"`
	FileSize   int64  `json:"fileSize"`
	MimeType   string `json:"mimeType"`
	SHA256     string `json:"sha256"`
	UploadedBy string `json:"uploadedBy"`
	CreatedAt  string `json:"createdAt"`
}

const (
	maxChunkSize    = 5 * 1024 * 1024  // 5MB per chunk
	maxFileSize     = 50 * 1024 * 1024 // 50MB max file size
	storageBasePath = "/app/storage"
	dedupeWindow    = 5 * time.Minute
)

func attachmentInitUpload(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		var req attachmentUploadInitRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}

		if req.FileName == "" || req.FileSize <= 0 {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "FileName and FileSize are required.", TraceID: c.GetString("traceId")})
			return
		}

		if req.FileSize > maxFileSize {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: fmt.Sprintf("File size exceeds maximum of %d MB.", maxFileSize/1024/1024), TraceID: c.GetString("traceId")})
			return
		}

		req.MimeType = strings.ToLower(strings.TrimSpace(req.MimeType))
		if !allowedMimeTypes[req.MimeType] {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "File type not allowed.", TraceID: c.GetString("traceId")})
			return
		}

		uploadID := uuid.NewString()
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		chunkSize := maxChunkSize
		if req.FileSize < int64(chunkSize) {
			chunkSize = int(req.FileSize)
		}
		if req.TotalChunks <= 0 {
			req.TotalChunks = 1
		}

		uploadDir := filepath.Join(storageBasePath, "uploads", uploadID)
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to create upload directory.", TraceID: c.GetString("traceId")})
			return
		}

		_, err := db.ExecContext(ctx, `
			INSERT INTO attachment_chunks (id, upload_id, case_id, chunk_index, chunk_size, sha256_hash, status)
			VALUES (?, ?, ?, ?, ?, '', 'pending')
		`, uuid.NewString(), uploadID, req.CaseID, 0, chunkSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to initialize upload.", TraceID: c.GetString("traceId")})
			return
		}

		c.JSON(http.StatusOK, attachmentUploadInitResponse{
			UploadID:    uploadID,
			ChunkSize:   chunkSize,
			TotalChunks: req.TotalChunks,
			UploadURL:   fmt.Sprintf("/api/v1/attachments/%s/chunk", uploadID),
		})
	}
}

func attachmentUploadChunk(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uploadID := c.Param("uploadId")
		if uploadID == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Upload ID required.", TraceID: c.GetString("traceId")})
			return
		}

		var req attachmentChunkUploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}

		req.UploadID = uploadID

		chunkData, err := hex.DecodeString(req.ChunkData)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid chunk data encoding.", TraceID: c.GetString("traceId")})
			return
		}

		if len(chunkData) > maxChunkSize {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: fmt.Sprintf("Chunk size exceeds maximum of %d MB.", maxChunkSize/1024/1024), TraceID: c.GetString("traceId")})
			return
		}

		hash := sha256.Sum256(chunkData)
		hashStr := hex.EncodeToString(hash[:])

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		chunkPath := filepath.Join(storageBasePath, "uploads", uploadID, fmt.Sprintf("chunk_%d", req.ChunkIndex))
		if err := os.WriteFile(chunkPath, chunkData, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to save chunk.", TraceID: c.GetString("traceId")})
			return
		}

		_, err = db.ExecContext(ctx, `
			UPDATE attachment_chunks 
			SET sha256_hash = ?, status = 'uploaded'
			WHERE upload_id = ? AND chunk_index = ?
		`, hashStr, uploadID, req.ChunkIndex)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to record chunk.", TraceID: c.GetString("traceId")})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":     "uploaded",
			"sha256":     hashStr,
			"chunkIndex": req.ChunkIndex,
		})
	}
}

func attachmentCompleteUpload(db *sql.DB, store auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req attachmentCompleteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Invalid request body.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		var caseID string
		rows, err := db.QueryContext(ctx, `
			SELECT case_id, chunk_index, sha256_hash 
			FROM attachment_chunks 
			WHERE upload_id = ? ORDER BY chunk_index
		`, req.UploadID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to retrieve chunks.", TraceID: c.GetString("traceId")})
			return
		}
		defer rows.Close()

		var chunks []struct {
			Hash  string
			Index int
		}
		for rows.Next() {
			var idx int
			var h string
			if err := rows.Scan(&caseID, &idx, &h); err != nil {
				continue
			}
			chunks = append(chunks, struct {
				Hash  string
				Index int
			}{h, idx})
		}

		if len(chunks) == 0 {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "No chunks found.", TraceID: c.GetString("traceId")})
			return
		}

		tempFilePath := filepath.Join(storageBasePath, "uploads", req.UploadID, "reassembled.dat")
		out, err := os.Create(tempFilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to reassemble chunks.", TraceID: c.GetString("traceId")})
			return
		}

		h256 := sha256.New()
		totalSize := int64(0)

		for _, chunk := range chunks {
			chunkPath := filepath.Join(storageBasePath, "uploads", req.UploadID, fmt.Sprintf("chunk_%d", chunk.Index))
			data, err := os.ReadFile(chunkPath)
			if err != nil {
				out.Close()
				c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to read chunk.", TraceID: c.GetString("traceId")})
				return
			}
			out.Write(data)
			h256.Write(data)
			totalSize += int64(len(data))
		}
		out.Close()
		finalHash := hex.EncodeToString(h256.Sum(nil))

		storagePath := filepath.Join(storageBasePath, "attachments", finalHash[:2], finalHash)
		if err := os.MkdirAll(filepath.Dir(storagePath), 0755); err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to create storage path.", TraceID: c.GetString("traceId")})
			return
		}

		var existingID string
		err = db.QueryRowContext(ctx, `
			SELECT id FROM attachments WHERE sha256_hash = ? LIMIT 1
		`, finalHash).Scan(&existingID)
		if err == nil && existingID != "" {
			os.Remove(tempFilePath)
			c.JSON(http.StatusOK, gin.H{
				"status":       "deduplicated",
				"attachmentId": existingID,
				"sha256":       finalHash,
			})
			return
		}

		err = os.Rename(tempFilePath, storagePath)
		if err != nil {
			data, _ := os.ReadFile(tempFilePath)
			os.WriteFile(storagePath, data, 0644)
			os.Remove(tempFilePath)
		}

		var fileName, mimeType string
		err = db.QueryRowContext(ctx, `
			SELECT ac.case_id, 'file.dat', 'application/octet-stream'
			FROM attachment_chunks ac WHERE ac.upload_id = ? LIMIT 1
		`, req.UploadID).Scan(&caseID, &fileName, &mimeType)
		if err != nil {
			caseID = "unknown"
		}

		userID := c.GetString("userId")
		attID := uuid.NewString()
		now := time.Now().UTC()

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to start transaction.", TraceID: c.GetString("traceId")})
			return
		}
		defer tx.Rollback()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO attachments (id, case_id, file_name, file_size, mime_type, sha256_hash, storage_path, uploaded_by, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, attID, caseID, fileName, totalSize, mimeType, finalHash, storagePath, userID, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to create attachment record.", TraceID: c.GetString("traceId")})
			return
		}

		auditRec := auditRecord{
			ID:        uuid.NewString(),
			TraceID:   c.GetString("traceId"),
			ActorID:   userID,
			Action:    "UPLOAD",
			Entity:    "Attachment",
			EntityID:  attID,
			After:     gin.H{"caseId": caseID, "sha256": finalHash, "size": totalSize},
			CreatedAt: now.Format(time.RFC3339),
		}
		_ = store.AppendAudit(auditRec)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to commit transaction.", TraceID: c.GetString("traceId")})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":       "completed",
			"attachmentId": attID,
			"sha256":       finalHash,
		})
	}
}

func attachmentListByCase(db *sql.DB) gin.HandlerFunc {
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
			SELECT id, case_id, file_name, file_size, mime_type, sha256_hash, uploaded_by, created_at
			FROM attachments
			WHERE case_id = ?
			ORDER BY created_at DESC
		`, caseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "Failed to load attachments.", TraceID: c.GetString("traceId")})
			return
		}
		defer rows.Close()

		out := make([]attachmentItem, 0, 16)
		for rows.Next() {
			var item attachmentItem
			var created time.Time
			if err := rows.Scan(&item.ID, &item.CaseID, &item.FileName, &item.FileSize, &item.MimeType, &item.SHA256, &item.UploadedBy, &created); err != nil {
				continue
			}
			item.CreatedAt = created.UTC().Format(time.RFC3339)
			out = append(out, item)
		}

		c.JSON(http.StatusOK, out)
	}
}

func attachmentGetDownload(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		attachmentID := c.Param("id")
		if attachmentID == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "validation_error", Message: "Attachment ID required.", TraceID: c.GetString("traceId")})
			return
		}

		scope, ok := getScope(c)
		if !ok || scope.InstitutionID == "" {
			c.JSON(http.StatusForbidden, errorResponse{Code: "authorization_error", Message: "Missing scope.", TraceID: c.GetString("traceId")})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		var fileName, storagePath, mimeType string
		var fileSize int64
		err := db.QueryRowContext(ctx, `
			SELECT a.file_name, a.storage_path, a.mime_type, a.file_size
			FROM attachments a
			JOIN cases c ON a.case_id = c.id
			WHERE a.id = ? AND c.institution_id = ?
		`, attachmentID, scope.InstitutionID).Scan(&fileName, &storagePath, &mimeType, &fileSize)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse{Code: "not_found", Message: "Attachment not found.", TraceID: c.GetString("traceId")})
			return
		}

		data, err := os.ReadFile(storagePath)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse{Code: "not_found", Message: "File not found on storage.", TraceID: c.GetString("traceId")})
			return
		}

		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		c.Header("Content-Type", mimeType)
		c.Header("Content-Length", strconv.FormatInt(fileSize, 10))
		c.Data(http.StatusOK, mimeType, data)
	}
}

func init() {
	os.MkdirAll(storageBasePath, 0755)
	os.MkdirAll(filepath.Join(storageBasePath, "uploads"), 0755)
	os.MkdirAll(filepath.Join(storageBasePath, "attachments"), 0755)
}
