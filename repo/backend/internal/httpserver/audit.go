package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type auditRecord struct {
	ID        string      `json:"id"`
	TraceID   string      `json:"traceId"`
	ActorID   string      `json:"actorId"`
	Action    string      `json:"action"`
	Entity    string      `json:"entity"`
	EntityID  string      `json:"entityId"`
	Before    interface{} `json:"beforeSnapshot,omitempty"`
	After     interface{} `json:"afterSnapshot,omitempty"`
	CreatedAt string      `json:"createdAt"`
}

func auditAppendMiddleware(writer auditWriter, action string, entity string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only append on successful mutations.
		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 {
			return
		}

		_ = writer.AppendAudit(auditRecord{
			ID:        uuid.NewString(),
			TraceID:   c.GetString("traceId"),
			ActorID:   c.GetString("userId"),
			Action:    action,
			Entity:    entity,
			EntityID:  "",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func auditListHandler(writer auditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		records, err := writer.ListAudit()
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{
				Code:    "internal_error",
				Message: "Failed to load audit records.",
				TraceID: c.GetString("traceId"),
			})
			return
		}
		// Preserve compatibility with existing output of Invoke-RestMethod (which wraps arrays sometimes).
		raw, _ := json.Marshal(records)
		c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
	}
}

