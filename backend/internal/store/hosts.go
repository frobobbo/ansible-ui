package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/google/uuid"
)

type HostStore struct {
	db *sql.DB
}

func (s *HostStore) List() ([]*models.Host, error) {
	rows, err := s.db.Query(
		"SELECT id, name, address, description, vars, created_at FROM hosts ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []*models.Host
	for rows.Next() {
		h, err := scanHost(rows.Scan)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}
	return hosts, rows.Err()
}

func (s *HostStore) Get(id string) (*models.Host, error) {
	row := s.db.QueryRow(
		"SELECT id, name, address, description, vars, created_at FROM hosts WHERE id = ?", id,
	)
	h, err := scanHost(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return h, err
}

func (s *HostStore) Create(name, address, description string, vars map[string]string) (*models.Host, error) {
	if vars == nil {
		vars = map[string]string{}
	}
	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return nil, err
	}
	h := &models.Host{
		ID:          uuid.New().String(),
		Name:        name,
		Address:     address,
		Description: description,
		Vars:        vars,
		CreatedAt:   time.Now(),
	}
	_, err = s.db.Exec(
		"INSERT INTO hosts (id, name, address, description, vars, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		h.ID, h.Name, h.Address, h.Description, string(varsJSON), h.CreatedAt,
	)
	return h, err
}

func (s *HostStore) Update(id, name, address, description string, vars map[string]string) (*models.Host, error) {
	if vars == nil {
		vars = map[string]string{}
	}
	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec(
		"UPDATE hosts SET name=?, address=?, description=?, vars=? WHERE id=?",
		name, address, description, string(varsJSON), id,
	)
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *HostStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM hosts WHERE id = ?", id)
	return err
}

// scanHost decodes a row using the provided scan function (handles both *sql.Row and *sql.Rows).
func scanHost(scan func(...any) error) (*models.Host, error) {
	h := &models.Host{}
	var varsJSON string
	if err := scan(&h.ID, &h.Name, &h.Address, &h.Description, &varsJSON, &h.CreatedAt); err != nil {
		return nil, err
	}
	h.Vars = map[string]string{}
	if varsJSON != "" && varsJSON != "null" {
		if err := json.Unmarshal([]byte(varsJSON), &h.Vars); err != nil {
			return nil, err
		}
	}
	return h, nil
}
