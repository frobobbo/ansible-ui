package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/google/uuid"
)

type UserStore struct {
	db *sql.DB
}

func (s *UserStore) List() ([]*models.User, error) {
	rows, err := s.db.Query("SELECT id, username, password_hash, role, created_at FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *UserStore) GetByUsername(username string) (*models.User, error) {
	u := &models.User{}
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (s *UserStore) GetByID(id string) (*models.User, error) {
	u := &models.User{}
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (s *UserStore) Create(username, passwordHash, role string) (*models.User, error) {
	u := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    time.Now(),
	}
	_, err := s.db.Exec(
		"INSERT INTO users (id, username, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)",
		u.ID, u.Username, u.PasswordHash, u.Role, u.CreatedAt,
	)
	return u, err
}

func (s *UserStore) Update(id, username, passwordHash, role string) (*models.User, error) {
	if passwordHash != "" {
		_, err := s.db.Exec(
			"UPDATE users SET username = ?, password_hash = ?, role = ? WHERE id = ?",
			username, passwordHash, role, id,
		)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := s.db.Exec(
			"UPDATE users SET username = ?, role = ? WHERE id = ?",
			username, role, id,
		)
		if err != nil {
			return nil, err
		}
	}
	return s.GetByID(id)
}

func (s *UserStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}
