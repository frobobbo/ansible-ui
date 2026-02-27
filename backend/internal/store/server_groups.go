package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/google/uuid"
)

type ServerGroupStore struct {
	db *sql.DB
}

func (s *ServerGroupStore) List() ([]*models.ServerGroup, error) {
	rows, err := s.db.Query("SELECT id, name, description, created_at FROM server_groups ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*models.ServerGroup
	for rows.Next() {
		g := &models.ServerGroup{}
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (s *ServerGroupStore) Get(id string) (*models.ServerGroup, error) {
	g := &models.ServerGroup{}
	err := s.db.QueryRow("SELECT id, name, description, created_at FROM server_groups WHERE id = ?", id).
		Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return g, err
}

func (s *ServerGroupStore) Create(name, description string) (*models.ServerGroup, error) {
	g := &models.ServerGroup{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}
	_, err := s.db.Exec(
		"INSERT INTO server_groups (id, name, description, created_at) VALUES (?, ?, ?, ?)",
		g.ID, g.Name, g.Description, g.CreatedAt,
	)
	return g, err
}

func (s *ServerGroupStore) Update(id, name, description string) (*models.ServerGroup, error) {
	_, err := s.db.Exec("UPDATE server_groups SET name=?, description=? WHERE id=?", name, description, id)
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *ServerGroupStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM server_groups WHERE id = ?", id)
	return err
}

// GetMembers returns the servers that belong to a server group.
func (s *ServerGroupStore) GetMembers(groupID string) ([]*models.Server, error) {
	rows, err := s.db.Query(`
		SELECT s.id, s.name, s.host, s.port, s.username, s.ssh_private_key, s.pre_command, s.created_at
		FROM servers s
		JOIN server_group_members m ON s.id = m.server_id
		WHERE m.group_id = ?
		ORDER BY s.name`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*models.Server
	for rows.Next() {
		sv := &models.Server{}
		if err := rows.Scan(&sv.ID, &sv.Name, &sv.Host, &sv.Port, &sv.Username, &sv.SSHPrivateKey, &sv.PreCommand, &sv.CreatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, sv)
	}
	return servers, rows.Err()
}

// SetMembers replaces the server group's member list.
func (s *ServerGroupStore) SetMembers(groupID string, serverIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM server_group_members WHERE group_id = ?", groupID); err != nil {
		return err
	}
	for _, sid := range serverIDs {
		if _, err := tx.Exec("INSERT INTO server_group_members (group_id, server_id) VALUES (?, ?)", groupID, sid); err != nil {
			return err
		}
	}
	return tx.Commit()
}
