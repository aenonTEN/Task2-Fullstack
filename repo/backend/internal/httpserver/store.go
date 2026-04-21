package httpserver

import "time"

type userInfo struct {
	UserID       string
	Username     string
	PasswordHash []byte
	RoleIDs      []string
	Scope        dataScope
}

type sessionInfo struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type authStore interface {
	FindUserByUsername(username string) (userInfo, bool, error)
	FindUserByID(userID string) (userInfo, bool, error)
	CreateSession(token string, userID string, expiresAt time.Time) error
	GetSession(token string) (sessionInfo, bool, error)
	RevokeSession(token string, revokedAt time.Time) error
}

type auditWriter interface {
	AppendAudit(rec auditRecord) error
	ListAudit() ([]auditRecord, error)
}

