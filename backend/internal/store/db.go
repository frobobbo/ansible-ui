package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"strings"
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
	db.Exec("ALTER TABLE forms ADD COLUMN is_quick_action INTEGER NOT NULL DEFAULT 0")
	db.Exec("ALTER TABLE forms ADD COLUMN image_path TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE forms ADD COLUMN image_name TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE forms ADD COLUMN schedule_cron TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE forms ADD COLUMN schedule_enabled INTEGER NOT NULL DEFAULT 0")
	db.Exec("ALTER TABLE forms ADD COLUMN webhook_token TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE forms ADD COLUMN notify_webhook TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE forms ADD COLUMN notify_email TEXT NOT NULL DEFAULT ''")

	// Server groups: rebuild forms to make server_id nullable and add server_group_id.
	// PRAGMA legacy_alter_table=ON keeps FK refs in form_fields/runs pointing to the
	// name "forms" (not rewritten to "_forms_bak"), so they resolve to the new table.
	var sgColExists int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('forms') WHERE name='server_group_id'").Scan(&sgColExists)
	if sgColExists == 0 {
		db.Exec("PRAGMA legacy_alter_table = ON")
		db.Exec("PRAGMA foreign_keys = OFF")
		db.Exec("ALTER TABLE forms RENAME TO _forms_bak")
		db.Exec(`CREATE TABLE forms (
			id               TEXT PRIMARY KEY,
			name             TEXT NOT NULL,
			description      TEXT NOT NULL DEFAULT '',
			playbook_id      TEXT NOT NULL REFERENCES playbooks(id) ON DELETE CASCADE,
			server_id        TEXT REFERENCES servers(id) ON DELETE CASCADE,
			server_group_id  TEXT REFERENCES server_groups(id) ON DELETE SET NULL,
			vault_id         TEXT REFERENCES vaults(id) ON DELETE SET NULL,
			is_quick_action  INTEGER NOT NULL DEFAULT 0,
			image_path       TEXT NOT NULL DEFAULT '',
			image_name       TEXT NOT NULL DEFAULT '',
			schedule_cron    TEXT NOT NULL DEFAULT '',
			schedule_enabled INTEGER NOT NULL DEFAULT 0,
			webhook_token    TEXT NOT NULL DEFAULT '',
			notify_webhook   TEXT NOT NULL DEFAULT '',
			notify_email     TEXT NOT NULL DEFAULT '',
			created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
		db.Exec(`INSERT INTO forms (id, name, description, playbook_id, server_id, vault_id, is_quick_action, image_path, image_name, schedule_cron, schedule_enabled, webhook_token, notify_webhook, notify_email, created_at, updated_at)
			SELECT id, name, description, playbook_id, server_id, vault_id, is_quick_action, image_path, image_name, schedule_cron, schedule_enabled, webhook_token, notify_webhook, notify_email, created_at, updated_at
			FROM _forms_bak`)
		db.Exec("DROP TABLE _forms_bak")
		db.Exec("PRAGMA foreign_keys = ON")
		db.Exec("PRAGMA legacy_alter_table = OFF")
	}
	db.Exec("ALTER TABLE runs ADD COLUMN batch_id TEXT")

	// Migrate users table to add 'editor' role — SQLite CHECK constraints require
	// a full table rebuild; check sqlite_master to avoid re-running on every start.
	var userSchema string
	db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='users'").Scan(&userSchema)
	if !strings.Contains(userSchema, "'editor'") {
		db.Exec(`ALTER TABLE users RENAME TO _users_old`)
		db.Exec(`CREATE TABLE users (
			id            TEXT PRIMARY KEY,
			username      TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role          TEXT NOT NULL CHECK(role IN ('admin','editor','viewer')) DEFAULT 'viewer',
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
		db.Exec(`INSERT INTO users SELECT * FROM _users_old`)
		db.Exec(`DROP TABLE _users_old`)
	}

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

func (db *DB) Users() *UserStore               { return &UserStore{db: db.conn} }
func (db *DB) Servers() *ServerStore           { return &ServerStore{db: db.conn} }
func (db *DB) Playbooks() *PlaybookStore       { return &PlaybookStore{db: db.conn} }
func (db *DB) Forms() *FormStore               { return &FormStore{db: db.conn} }
func (db *DB) Runs() *RunStore                 { return &RunStore{db: db.conn} }
func (db *DB) Audit() *AuditStore              { return &AuditStore{db: db.conn} }
func (db *DB) ServerGroups() *ServerGroupStore { return &ServerGroupStore{db: db.conn} }
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
