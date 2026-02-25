package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/google/uuid"
)

type ServerStore struct {
	db *sql.DB
}

func (s *ServerStore) List() ([]*models.Server, error) {
	rows, err := s.db.Query("SELECT id, name, host, port, username, pre_command, created_at FROM servers ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*models.Server
	for rows.Next() {
		sv := &models.Server{}
		if err := rows.Scan(&sv.ID, &sv.Name, &sv.Host, &sv.Port, &sv.Username, &sv.PreCommand, &sv.CreatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, sv)
	}
	return servers, rows.Err()
}

func (s *ServerStore) Get(id string) (*models.Server, error) {
	sv := &models.Server{}
	err := s.db.QueryRow(
		"SELECT id, name, host, port, username, ssh_private_key, pre_command, created_at FROM servers WHERE id = ?", id,
	).Scan(&sv.ID, &sv.Name, &sv.Host, &sv.Port, &sv.Username, &sv.SSHPrivateKey, &sv.PreCommand, &sv.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return sv, err
}

func (s *ServerStore) Create(name, host string, port int, username, sshKey, preCommand string) (*models.Server, error) {
	sv := &models.Server{
		ID:            uuid.New().String(),
		Name:          name,
		Host:          host,
		Port:          port,
		Username:      username,
		SSHPrivateKey: sshKey,
		PreCommand:    preCommand,
		CreatedAt:     time.Now(),
	}
	_, err := s.db.Exec(
		"INSERT INTO servers (id, name, host, port, username, ssh_private_key, pre_command, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		sv.ID, sv.Name, sv.Host, sv.Port, sv.Username, sv.SSHPrivateKey, sv.PreCommand, sv.CreatedAt,
	)
	sv.SSHPrivateKey = "" // don't return key
	return sv, err
}

func (s *ServerStore) Update(id, name, host string, port int, username, sshKey, preCommand string) (*models.Server, error) {
	if sshKey != "" {
		_, err := s.db.Exec(
			"UPDATE servers SET name=?, host=?, port=?, username=?, ssh_private_key=?, pre_command=? WHERE id=?",
			name, host, port, username, sshKey, preCommand, id,
		)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := s.db.Exec(
			"UPDATE servers SET name=?, host=?, port=?, username=?, pre_command=? WHERE id=?",
			name, host, port, username, preCommand, id,
		)
		if err != nil {
			return nil, err
		}
	}
	return s.Get(id)
}

func (s *ServerStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM servers WHERE id = ?", id)
	return err
}
