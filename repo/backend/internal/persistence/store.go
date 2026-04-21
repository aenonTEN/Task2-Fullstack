package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type DataScope struct {
	InstitutionID string
	DepartmentID  string
	TeamID        string
}

type UserInfo struct {
	UserID       string
	Username     string
	PasswordHash []byte
	RoleIDs      []string
	Scope        DataScope
}

type SessionInfo struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type AuditRecord struct {
	ID        string
	TraceID   string
	ActorID   string
	Action    string
	Entity    string
	EntityID  string
	Before    any
	After     any
	CreatedAt time.Time
}

type Store struct {
	DB *sql.DB
}

func (s *Store) FindUserByUsername(ctx context.Context, username string) (UserInfo, bool, error) {
	var u UserInfo
	var scope DataScope
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, username, password_hash, institution_id, department_id, team_id
		FROM users
		WHERE username = ?
	`, username).Scan(&u.UserID, &u.Username, &u.PasswordHash, &scope.InstitutionID, &scope.DepartmentID, &scope.TeamID)
	if errors.Is(err, sql.ErrNoRows) {
		return UserInfo{}, false, nil
	}
	if err != nil {
		return UserInfo{}, false, err
	}
	u.Scope = scope

	rows, err := s.DB.QueryContext(ctx, `SELECT role_id FROM user_roles WHERE user_id = ?`, u.UserID)
	if err != nil {
		return UserInfo{}, false, err
	}
	defer rows.Close()
	roles := make([]string, 0, 4)
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return UserInfo{}, false, err
		}
		roles = append(roles, r)
	}
	u.RoleIDs = roles
	return u, true, nil
}

func (s *Store) FindUserByID(ctx context.Context, userID string) (UserInfo, bool, error) {
	var u UserInfo
	var scope DataScope
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, username, password_hash, institution_id, department_id, team_id
		FROM users
		WHERE id = ?
	`, userID).Scan(&u.UserID, &u.Username, &u.PasswordHash, &scope.InstitutionID, &scope.DepartmentID, &scope.TeamID)
	if errors.Is(err, sql.ErrNoRows) {
		return UserInfo{}, false, nil
	}
	if err != nil {
		return UserInfo{}, false, err
	}
	u.Scope = scope

	rows, err := s.DB.QueryContext(ctx, `SELECT role_id FROM user_roles WHERE user_id = ?`, u.UserID)
	if err != nil {
		return UserInfo{}, false, err
	}
	defer rows.Close()
	roles := make([]string, 0, 4)
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return UserInfo{}, false, err
		}
		roles = append(roles, r)
	}
	u.RoleIDs = roles
	return u, true, nil
}

func (s *Store) CreateSession(ctx context.Context, token string, userID string, expiresAt time.Time) error {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`, token, userID, expiresAt)
	return err
}

func (s *Store) GetSession(ctx context.Context, token string) (SessionInfo, bool, error) {
	var si SessionInfo
	err := s.DB.QueryRowContext(ctx, `
		SELECT token, user_id, expires_at, revoked_at
		FROM sessions
		WHERE token = ?
	`, token).Scan(&si.Token, &si.UserID, &si.ExpiresAt, &si.RevokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return SessionInfo{}, false, nil
	}
	if err != nil {
		return SessionInfo{}, false, err
	}
	return si, true, nil
}

func (s *Store) RevokeSession(ctx context.Context, token string, revokedAt time.Time) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE sessions SET revoked_at = ? WHERE token = ?`, revokedAt, token)
	return err
}

func (s *Store) AppendAudit(ctx context.Context, rec AuditRecord) error {
	var beforeJSON any
	var afterJSON any
	if rec.Before != nil {
		b, _ := json.Marshal(rec.Before)
		beforeJSON = string(b)
	}
	if rec.After != nil {
		a, _ := json.Marshal(rec.After)
		afterJSON = string(a)
	}
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO audit_records (id, trace_id, actor_id, action, entity, entity_id, before_snapshot, after_snapshot, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rec.ID, rec.TraceID, rec.ActorID, rec.Action, rec.Entity, rec.EntityID, beforeJSON, afterJSON, rec.CreatedAt)
	return err
}

func (s *Store) ListAudit(ctx context.Context, limit int) ([]AuditRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, trace_id, actor_id, action, entity, entity_id, before_snapshot, after_snapshot, created_at
		FROM audit_records
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AuditRecord, 0, 32)
	for rows.Next() {
		var r AuditRecord
		var before sql.NullString
		var after sql.NullString
		if err := rows.Scan(&r.ID, &r.TraceID, &r.ActorID, &r.Action, &r.Entity, &r.EntityID, &before, &after, &r.CreatedAt); err != nil {
			return nil, err
		}
		if before.Valid {
			_ = json.Unmarshal([]byte(before.String), &r.Before)
		}
		if after.Valid {
			_ = json.Unmarshal([]byte(after.String), &r.After)
		}
		out = append(out, r)
	}
	return out, nil
}

