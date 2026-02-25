package models

import "time"

type User struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         string    `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Server struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Host          string    `json:"host" db:"host"`
	Port          int       `json:"port" db:"port"`
	Username      string    `json:"username" db:"username"`
	SSHPrivateKey string    `json:"ssh_private_key,omitempty" db:"ssh_private_key"`
	PreCommand    string    `json:"pre_command" db:"pre_command"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type Playbook struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	FilePath    string    `json:"file_path" db:"file_path"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type FormField struct {
	ID           string `json:"id" db:"id"`
	FormID       string `json:"form_id" db:"form_id"`
	Name         string `json:"name" db:"name"`
	Label        string `json:"label" db:"label"`
	FieldType    string `json:"field_type" db:"field_type"`
	DefaultValue string `json:"default_value" db:"default_value"`
	Options      string `json:"options" db:"options"`
	Required     bool   `json:"required" db:"required"`
	SortOrder    int    `json:"sort_order" db:"sort_order"`
}

type Vault struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	VaultFileName string   `json:"vault_file_name"` // original filename; empty if no file uploaded
	CreatedAt    time.Time `json:"created_at"`
}

type Form struct {
	ID          string      `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description string      `json:"description" db:"description"`
	PlaybookID  string      `json:"playbook_id" db:"playbook_id"`
	ServerID    string      `json:"server_id" db:"server_id"`
	VaultID     *string     `json:"vault_id" db:"vault_id"`
	Fields      []FormField `json:"fields,omitempty" db:"-"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

type Run struct {
	ID         string     `json:"id" db:"id"`
	FormID     *string    `json:"form_id" db:"form_id"`
	PlaybookID string     `json:"playbook_id" db:"playbook_id"`
	ServerID   string     `json:"server_id" db:"server_id"`
	Variables  string     `json:"variables" db:"variables"`
	Status     string     `json:"status" db:"status"`
	Output     string     `json:"output" db:"output"`
	StartedAt  *time.Time `json:"started_at" db:"started_at"`
	FinishedAt *time.Time `json:"finished_at" db:"finished_at"`
}
