package persistence

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func GetBcryptCost() int {
	costStr := os.Getenv("BCRYPT_COST")
	cost, err := strconv.Atoi(costStr)
	if err != nil || cost < 4 || cost > 31 {
		return bcrypt.DefaultCost
	}
	return cost
}

func EnsureSchema(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(64) PRIMARY KEY,
			username VARCHAR(64) NOT NULL UNIQUE,
			password_hash VARBINARY(255) NOT NULL,
			institution_id VARCHAR(64) NOT NULL,
			department_id VARCHAR(64) NOT NULL,
			team_id VARCHAR(64) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS roles (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(128) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS user_roles (
			user_id VARCHAR(64) NOT NULL,
			role_id VARCHAR(64) NOT NULL,
			PRIMARY KEY (user_id, role_id)
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token VARCHAR(64) PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			revoked_at TIMESTAMP NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX (user_id),
			INDEX (expires_at)
		)`,
		`CREATE TABLE IF NOT EXISTS audit_records (
			id VARCHAR(64) PRIMARY KEY,
			trace_id VARCHAR(64) NOT NULL,
			actor_id VARCHAR(64) NOT NULL,
			action VARCHAR(64) NOT NULL,
			entity VARCHAR(64) NOT NULL,
			entity_id VARCHAR(64) NOT NULL,
			before_snapshot JSON NULL,
			after_snapshot JSON NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX (actor_id),
			INDEX (action),
			INDEX (created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS candidates (
			id VARCHAR(64) PRIMARY KEY,
			institution_id VARCHAR(64) NOT NULL,
			name VARCHAR(128) NOT NULL,
			phone VARCHAR(32) NOT NULL DEFAULT '',
			phone_masked VARCHAR(32) NOT NULL,
			id_number VARCHAR(64) NOT NULL DEFAULT '',
			education VARCHAR(64) NOT NULL,
			experience_years INT NOT NULL,
			skills_json JSON NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NULL,
			deleted_at TIMESTAMP NULL,
			INDEX (institution_id),
			INDEX (phone),
			INDEX (id_number),
			INDEX (created_at),
			INDEX (deleted_at)
		)`,
		`CREATE TABLE IF NOT EXISTS qualifications (
			id VARCHAR(64) PRIMARY KEY,
			candidate_id VARCHAR(64) NOT NULL,
			institution_id VARCHAR(64) NOT NULL,
			name VARCHAR(128) NOT NULL,
			issued_date DATE NOT NULL,
			expiry_date DATE NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NULL,
			deleted_at TIMESTAMP NULL,
			INDEX (candidate_id),
			INDEX (institution_id),
			INDEX (expiry_date),
			INDEX (status),
			INDEX (deleted_at)
		)`,
		`CREATE TABLE IF NOT EXISTS restrictions (
			id VARCHAR(64) PRIMARY KEY,
			institution_id VARCHAR(64) NOT NULL,
			candidate_id VARCHAR(64) NOT NULL,
			restriction_type VARCHAR(64) NOT NULL,
			reason VARCHAR(256) NOT NULL DEFAULT '',
			purchase_window_start TIMESTAMP NULL,
			purchase_window_end TIMESTAMP NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX (institution_id),
			INDEX (candidate_id),
			INDEX (is_active),
			INDEX (purchase_window_end)
		)`,
		`CREATE TABLE IF NOT EXISTS cases (
			id VARCHAR(64) PRIMARY KEY,
			case_number VARCHAR(64) NOT NULL,
			institution_id VARCHAR(64) NOT NULL,
			candidate_id VARCHAR(64) NOT NULL,
			case_type VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			subject VARCHAR(256) NOT NULL,
			description TEXT NOT NULL,
			created_by VARCHAR(64) NOT NULL,
			assigned_to VARCHAR(64) NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NULL,
			closed_at TIMESTAMP NULL,
			INDEX (institution_id),
			INDEX (case_number),
			INDEX (candidate_id),
			INDEX (status),
			INDEX (created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS attachments (
			id VARCHAR(64) PRIMARY KEY,
			case_id VARCHAR(64) NOT NULL,
			file_name VARCHAR(256) NOT NULL,
			file_size BIGINT NOT NULL,
			mime_type VARCHAR(128) NOT NULL,
			sha256_hash VARCHAR(64) NOT NULL,
			storage_path VARCHAR(512) NOT NULL,
			uploaded_by VARCHAR(64) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX (case_id),
			INDEX (sha256_hash)
		)`,
		`CREATE TABLE IF NOT EXISTS attachment_chunks (
			id VARCHAR(64) PRIMARY KEY,
			upload_id VARCHAR(64) NOT NULL,
			case_id VARCHAR(64) NOT NULL,
			chunk_index INT NOT NULL,
			chunk_size INT NOT NULL,
			sha256_hash VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX (upload_id),
			INDEX (case_id),
			INDEX (status)
		)`,
		`CREATE TABLE IF NOT EXISTS positions (
			id VARCHAR(64) PRIMARY KEY,
			institution_id VARCHAR(64) NOT NULL,
			title VARCHAR(256) NOT NULL,
			department VARCHAR(128) NOT NULL,
			description TEXT NOT NULL,
			requirements TEXT NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'open',
			created_by VARCHAR(64) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NULL,
			closed_at TIMESTAMP NULL,
			INDEX (institution_id),
			INDEX (status),
			INDEX (created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS qualification_profiles (
			id VARCHAR(64) PRIMARY KEY,
			institution_id VARCHAR(64) NOT NULL,
			name VARCHAR(128) NOT NULL,
			description TEXT NOT NULL,
			required_skills JSON NOT NULL,
			min_experience_years INT NOT NULL DEFAULT 0,
			required_education VARCHAR(64) NOT NULL,
			validity_months INT NOT NULL DEFAULT 12,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NULL,
			INDEX (institution_id),
			INDEX (is_active)
		)`,
		`CREATE TABLE IF NOT EXISTS idempotency_keys (
			id VARCHAR(64) PRIMARY KEY,
			key_hash VARCHAR(64) NOT NULL,
			entity_type VARCHAR(64) NOT NULL,
			response_status INT NOT NULL,
			response_body TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX (key_hash),
			INDEX (expires_at)
		)`,
		`CREATE TABLE IF NOT EXISTS case_history (
			id VARCHAR(64) PRIMARY KEY,
			case_id VARCHAR(64) NOT NULL,
			action VARCHAR(64) NOT NULL,
			actor_id VARCHAR(64) NOT NULL,
			details JSON NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX (case_id),
			INDEX (created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			id VARCHAR(64) PRIMARY KEY,
			institution_id VARCHAR(64) NOT NULL,
			name VARCHAR(128) NOT NULL,
			color VARCHAR(32) NOT NULL DEFAULT '#6b7280',
			description VARCHAR(256) NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX (institution_id),
			UNIQUE KEY unique_tag_name (institution_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS candidate_tags (
			candidate_id VARCHAR(64) NOT NULL,
			tag_id VARCHAR(64) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (candidate_id, tag_id),
			INDEX (candidate_id),
			INDEX (tag_id)
		)`,
		`CREATE TABLE IF NOT EXISTS case_tags (
			case_id VARCHAR(64) NOT NULL,
			tag_id VARCHAR(64) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (case_id, tag_id),
			INDEX (case_id),
			INDEX (tag_id)
		)`,
	}

	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return err
		}
	}

	// Migration: add missing columns to existing candidates table (ignore errors if columns exist).
	migrationAddColumn := func(colName string, colDef string) {
		_, _ = db.ExecContext(ctx, "ALTER TABLE candidates ADD COLUMN "+colName+" "+colDef)
	}
	migrationAddColumn("phone", "VARCHAR(32) NOT NULL DEFAULT ''")
	migrationAddColumn("id_number", "VARCHAR(64) NOT NULL DEFAULT ''")
	migrationAddColumn("updated_at", "TIMESTAMP NULL")

	// Seed admin role and admin user if absent.
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO roles (id, name) VALUES ('role_admin', 'Administrator')`); err != nil {
		return err
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE username='admin'`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("password123"), GetBcryptCost())
		if err != nil {
			return err
		}
		_, err = db.ExecContext(ctx, `
			INSERT INTO users (id, username, password_hash, institution_id, department_id, team_id)
			VALUES ('u_admin', 'admin', ?, 'inst_demo', 'dept_demo', 'team_demo')
		`, hash)
		if err != nil {
			return err
		}
		_, err = db.ExecContext(ctx, `INSERT IGNORE INTO user_roles (user_id, role_id) VALUES ('u_admin', 'role_admin')`)
		if err != nil {
			return err
		}
	}

	// Cleanup expired sessions opportunistically.
	cleanupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, _ = db.ExecContext(cleanupCtx, `DELETE FROM sessions WHERE expires_at < ?`, time.Now().UTC())

	return nil
}
