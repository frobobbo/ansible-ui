package store

import (
	"database/sql"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/google/uuid"
)

type AuditStore struct {
	db *sql.DB
}

// Log records an audit event. Safe to call with empty fields — they are stored as-is.
func (s *AuditStore) Log(userID, username, action, resource, resourceID, details, ip string) {
	if details == "" {
		details = "{}"
	}
	s.db.Exec(
		"INSERT INTO audit_logs (id, user_id, username, action, resource, resource_id, details, ip, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		uuid.New().String(), userID, username, action, resource, resourceID, details, ip, time.Now(),
	)
}

// List returns audit log entries in reverse-chronological order.
func (s *AuditStore) List(limit, offset int) ([]*models.AuditLog, int, error) {
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&total); err != nil {
		return nil, 0, err
	}

	q := "SELECT id, user_id, username, action, resource, resource_id, details, ip, created_at FROM audit_logs ORDER BY created_at DESC"
	if limit > 0 {
		q += " LIMIT ? OFFSET ?"
	}

	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = s.db.Query(q, limit, offset)
	} else {
		rows, err = s.db.Query(q)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		l := &models.AuditLog{}
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.Action, &l.Resource, &l.ResourceID, &l.Details, &l.IP, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}
