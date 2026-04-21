package httpserver

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type idempotencyStore struct {
	DB *sql.DB
}

func (s *idempotencyStore) Get(keyHash string) (int, string, bool, error) {
	var status int
	var body string
	var expiresAt time.Time
	err := s.DB.QueryRowContext(context.Background(), `
		SELECT response_status, response_body, expires_at 
		FROM idempotency_keys 
		WHERE key_hash = ? AND expires_at > ?
	`, keyHash, time.Now().UTC()).Scan(&status, &body, &expiresAt)
	if err == sql.ErrNoRows {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, err
	}
	return status, body, true, nil
}

func (s *idempotencyStore) Set(keyHash string, entityType string, status int, body string, window time.Duration) error {
	expiresAt := time.Now().UTC().Add(window)
	_, err := s.DB.ExecContext(context.Background(), `
		INSERT INTO idempotency_keys (id, key_hash, entity_type, response_status, response_body, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, uuid.New().String(), keyHash, entityType, status, body, expiresAt)
	return err
}

func IdempotencyMiddleware(store *idempotencyStore, entityType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		hash := sha256.Sum256([]byte(key))
		keyHash := hex.EncodeToString(hash[:])

		status, body, found, err := store.Get(keyHash)
		if err == nil && found {
			c.JSON(status, body)
			c.Abort()
			return
		}

		c.Set("idempotencyKey", keyHash)
		c.Set("idempotencyEntityType", entityType)
		c.Next()

		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			body := c.GetString("idempotencyResponse")
			if body != "" {
				_ = store.Set(keyHash, entityType, c.Writer.Status(), body, 24*time.Hour)
			}
		}
	}
}
