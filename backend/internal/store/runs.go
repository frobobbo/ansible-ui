package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/google/uuid"
)

type RunStore struct {
	db *sql.DB
}

func (s *RunStore) Create(formID *string, playbookID, serverID, variables string) (*models.Run, error) {
	r := &models.Run{
		ID:         uuid.New().String(),
		FormID:     formID,
		PlaybookID: playbookID,
		ServerID:   serverID,
		Variables:  variables,
		Status:     "pending",
		Output:     "",
	}
	_, err := s.db.Exec(
		"INSERT INTO runs (id, form_id, playbook_id, server_id, variables, status, output) VALUES (?, ?, ?, ?, ?, 'pending', '')",
		r.ID, r.FormID, r.PlaybookID, r.ServerID, r.Variables,
	)
	return r, err
}

func (s *RunStore) Get(id string) (*models.Run, error) {
	r := &models.Run{}
	err := s.db.QueryRow(
		"SELECT id, form_id, playbook_id, server_id, variables, status, output, started_at, finished_at FROM runs WHERE id = ?", id,
	).Scan(&r.ID, &r.FormID, &r.PlaybookID, &r.ServerID, &r.Variables, &r.Status, &r.Output, &r.StartedAt, &r.FinishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

func (s *RunStore) List() ([]*models.Run, error) {
	rows, err := s.db.Query(
		"SELECT id, form_id, playbook_id, server_id, variables, status, output, started_at, finished_at FROM runs ORDER BY rowid DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*models.Run
	for rows.Next() {
		r := &models.Run{}
		if err := rows.Scan(&r.ID, &r.FormID, &r.PlaybookID, &r.ServerID, &r.Variables, &r.Status, &r.Output, &r.StartedAt, &r.FinishedAt); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

func (s *RunStore) SetRunning(id string) error {
	t := time.Now()
	_, err := s.db.Exec("UPDATE runs SET status='running', started_at=? WHERE id=?", t, id)
	return err
}

func (s *RunStore) AppendOutput(id, chunk string) error {
	_, err := s.db.Exec("UPDATE runs SET output = output || ? WHERE id = ?", chunk, id)
	return err
}

func (s *RunStore) Finish(id, status, output string) error {
	t := time.Now()
	_, err := s.db.Exec(
		"UPDATE runs SET status=?, output=?, finished_at=? WHERE id=?",
		status, output, t, id,
	)
	return err
}
