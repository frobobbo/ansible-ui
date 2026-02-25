package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

//go:embed schema.sql
var schema string

type DB struct {
	conn *sql.DB
}

func New(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Single connection to prevent write contention with SQLite
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("pragma foreign_keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("pragma journal_mode: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	// Migrations: ALTER TABLE is idempotent-ish — SQLite returns an error if the
	// column already exists, which we intentionally ignore.
	db.Exec("ALTER TABLE servers ADD COLUMN pre_command TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE forms ADD COLUMN vault_id TEXT REFERENCES vaults(id) ON DELETE SET NULL")
	db.Exec("ALTER TABLE vaults ADD COLUMN vault_file_path TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE vaults ADD COLUMN vault_file_name TEXT NOT NULL DEFAULT ''")

	return &DB{conn: db}, nil
}

func (db *DB) EnsureDefaultAdmin() error {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = db.conn.Exec(
		"INSERT INTO users (id, username, password_hash, role, created_at) VALUES (?, ?, ?, 'admin', ?)",
		uuid.New().String(), "admin", string(hash), time.Now(),
	)
	return err
}

func (db *DB) Users() *UserStore         { return &UserStore{db: db.conn} }
func (db *DB) Servers() *ServerStore     { return &ServerStore{db: db.conn} }
func (db *DB) Playbooks() *PlaybookStore { return &PlaybookStore{db: db.conn} }
func (db *DB) Forms() *FormStore         { return &FormStore{db: db.conn} }
func (db *DB) Runs() *RunStore           { return &RunStore{db: db.conn} }
func (db *DB) Vaults(secret string) *VaultStore {
	return newVaultStore(db.conn, secret)
}

// scanUser scans a user row (without password_hash by default)
func scanUser(row *sql.Row) (*models.User, error) {
	u := &models.User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}
