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

const playbookCols = "id, name, description, repo_url, branch, playbook_path, token, created_at"

func scanPlaybook(row interface {
	Scan(...any) error
}) (*models.Playbook, error) {
	p := &models.Playbook{}
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.RepoURL, &p.Branch, &p.PlaybookPath, &p.Token, &p.CreatedAt)
	return p, err
}

func (s *PlaybookStore) List() ([]*models.Playbook, error) {
	rows, err := s.db.Query("SELECT " + playbookCols + " FROM playbooks ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Playbook
	for rows.Next() {
		p, err := scanPlaybook(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (s *PlaybookStore) Get(id string) (*models.Playbook, error) {
	p, err := scanPlaybook(s.db.QueryRow("SELECT "+playbookCols+" FROM playbooks WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

func (s *PlaybookStore) Create(name, description, repoURL, branch, playbookPath, token string) (*models.Playbook, error) {
	p := &models.Playbook{
		ID:           uuid.New().String(),
		Name:         name,
		Description:  description,
		RepoURL:      repoURL,
		Branch:       branch,
		PlaybookPath: playbookPath,
		Token:        token,
		CreatedAt:    time.Now(),
	}
	_, err := s.db.Exec(
		"INSERT INTO playbooks (id, name, description, repo_url, branch, playbook_path, token, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		p.ID, p.Name, p.Description, p.RepoURL, p.Branch, p.PlaybookPath, p.Token, p.CreatedAt,
	)
	return p, err
}

func (s *PlaybookStore) Update(id, name, description, repoURL, branch, playbookPath, token string) (*models.Playbook, error) {
	_, err := s.db.Exec(
		"UPDATE playbooks SET name=?, description=?, repo_url=?, branch=?, playbook_path=?, token=? WHERE id=?",
		name, description, repoURL, branch, playbookPath, token, id,
	)
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *PlaybookStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM playbooks WHERE id = ?", id)
	return err
}
