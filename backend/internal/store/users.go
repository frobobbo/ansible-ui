package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/google/uuid"
)

type UserStore struct {
	db *sql.DB
}

const userCols = "id, username, password_hash, COALESCE(email,''), role, created_at"

func scanUser(row interface{ Scan(...any) error }) (*models.User, error) {
	u := &models.User{}
	return u, row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Email, &u.Role, &u.CreatedAt)
}

func (s *UserStore) List() ([]*models.User, error) {
	rows, err := s.db.Query("SELECT " + userCols + " FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *UserStore) GetByUsername(username string) (*models.User, error) {
	u, err := scanUser(s.db.QueryRow(
		"SELECT "+userCols+" FROM users WHERE username = ?", username,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (s *UserStore) GetByID(id string) (*models.User, error) {
	u, err := scanUser(s.db.QueryRow(
		"SELECT "+userCols+" FROM users WHERE id = ?", id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (s *UserStore) Create(username, passwordHash, role, email string) (*models.User, error) {
	u := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: passwordHash,
		Email:        email,
		Role:         role,
		CreatedAt:    time.Now(),
	}
	_, err := s.db.Exec(
		"INSERT INTO users (id, username, password_hash, email, role, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		u.ID, u.Username, u.PasswordHash, u.Email, u.Role, u.CreatedAt,
	)
	return u, err
}

func (s *UserStore) Update(id, username, passwordHash, role, email string) (*models.User, error) {
	if passwordHash != "" {
		_, err := s.db.Exec(
			"UPDATE users SET username=?, password_hash=?, email=?, role=? WHERE id=?",
			username, passwordHash, email, role, id,
		)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := s.db.Exec(
			"UPDATE users SET username=?, email=?, role=? WHERE id=?",
			username, email, role, id,
		)
		if err != nil {
			return nil, err
		}
	}
	return s.GetByID(id)
}

func (s *UserStore) UpdatePassword(id, passwordHash string) error {
	_, err := s.db.Exec("UPDATE users SET password_hash=? WHERE id=?", passwordHash, id)
	return err
}

func (s *UserStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

// ── Password reset tokens ────────────────────────────────────────────────────

// CreateResetToken generates a secure random token, stores its SHA-256 hash,
// and returns the raw token to be sent to the user.
func (s *UserStore) CreateResetToken(userID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	hash := sha256Token(token)

	_, err := s.db.Exec(
		"INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at) VALUES (?, ?, ?, ?)",
		uuid.New().String(), userID, hash, time.Now().Add(time.Hour),
	)
	return token, err
}

// ConsumeResetToken validates a raw token and returns the associated user ID.
// Returns ("", nil) if the token is not found, expired, or already used.
// On success the token is marked used.
func (s *UserStore) ConsumeResetToken(rawToken string) (string, error) {
	hash := sha256Token(rawToken)

	var id, userID string
	var expiresAt time.Time
	var usedAt sql.NullTime

	err := s.db.QueryRow(
		"SELECT id, user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash=?", hash,
	).Scan(&id, &userID, &expiresAt, &usedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if usedAt.Valid || time.Now().After(expiresAt) {
		return "", nil
	}

	_, err = s.db.Exec("UPDATE password_reset_tokens SET used_at=? WHERE id=?", time.Now(), id)
	return userID, err
}

func sha256Token(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
