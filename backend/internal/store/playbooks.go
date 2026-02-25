package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/google/uuid"
)

type PlaybookStore struct {
	db *sql.DB
}

func (s *PlaybookStore) List() ([]*models.Playbook, error) {
	rows, err := s.db.Query("SELECT id, name, description, file_path, created_at FROM playbooks ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playbooks []*models.Playbook
	for rows.Next() {
		p := &models.Playbook{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.FilePath, &p.CreatedAt); err != nil {
			return nil, err
		}
		playbooks = append(playbooks, p)
	}
	return playbooks, rows.Err()
}

func (s *PlaybookStore) Get(id string) (*models.Playbook, error) {
	p := &models.Playbook{}
	err := s.db.QueryRow(
		"SELECT id, name, description, file_path, created_at FROM playbooks WHERE id = ?", id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.FilePath, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

func (s *PlaybookStore) Create(name, description, filePath string) (*models.Playbook, error) {
	p := &models.Playbook{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		FilePath:    filePath,
		CreatedAt:   time.Now(),
	}
	_, err := s.db.Exec(
		"INSERT INTO playbooks (id, name, description, file_path, created_at) VALUES (?, ?, ?, ?, ?)",
		p.ID, p.Name, p.Description, p.FilePath, p.CreatedAt,
	)
	return p, err
}

func (s *PlaybookStore) Delete(id string) (string, error) {
	var filePath string
	err := s.db.QueryRow("SELECT file_path FROM playbooks WHERE id = ?", id).Scan(&filePath)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec("DELETE FROM playbooks WHERE id = ?", id)
	return filePath, err
}
