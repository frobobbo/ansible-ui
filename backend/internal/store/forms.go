package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/google/uuid"
)

type FormStore struct {
	db *sql.DB
}

const formSelect = "SELECT id, name, description, playbook_id, server_id, server_group_id, vault_id, is_quick_action, image_name, schedule_cron, schedule_enabled, webhook_token, notify_webhook, notify_email, created_at, updated_at FROM forms"

func scanForm(row interface {
	Scan(...any) error
}) (*models.Form, error) {
	f := &models.Form{}
	var isQuickAction, scheduleEnabled int
	var serverID, serverGroupID sql.NullString
	err := row.Scan(&f.ID, &f.Name, &f.Description, &f.PlaybookID, &serverID, &serverGroupID, &f.VaultID, &isQuickAction, &f.ImageName, &f.ScheduleCron, &scheduleEnabled, &f.WebhookToken, &f.NotifyWebhook, &f.NotifyEmail, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if serverID.Valid {
		f.ServerID = &serverID.String
	}
	if serverGroupID.Valid {
		f.ServerGroupID = &serverGroupID.String
	}
	f.IsQuickAction = isQuickAction == 1
	f.ScheduleEnabled = scheduleEnabled == 1
	return f, nil
}

func (s *FormStore) List() ([]*models.Form, error) {
	rows, err := s.db.Query(formSelect + " ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forms []*models.Form
	for rows.Next() {
		f, err := scanForm(rows)
		if err != nil {
			return nil, err
		}
		forms = append(forms, f)
	}
	return forms, rows.Err()
}

func (s *FormStore) GetQuickActions() ([]*models.Form, error) {
	rows, err := s.db.Query(formSelect + " WHERE is_quick_action = 1 ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forms []*models.Form
	for rows.Next() {
		f, err := scanForm(rows)
		if err != nil {
			return nil, err
		}
		forms = append(forms, f)
	}
	return forms, rows.Err()
}

func (s *FormStore) Get(id string) (*models.Form, error) {
	f, err := scanForm(s.db.QueryRow(formSelect+" WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	fields, err := s.GetFields(id)
	if err != nil {
		return nil, err
	}
	f.Fields = fields
	return f, nil
}

func (s *FormStore) GetByWebhookToken(token string) (*models.Form, error) {
	if token == "" {
		return nil, nil
	}
	f, err := scanForm(s.db.QueryRow(formSelect+" WHERE webhook_token = ?", token))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.Fields, err = s.GetFields(f.ID)
	return f, err
}

func (s *FormStore) SetWebhookToken(id, token string) error {
	_, err := s.db.Exec("UPDATE forms SET webhook_token=? WHERE id=?", token, id)
	return err
}

func (s *FormStore) GetFields(formID string) ([]models.FormField, error) {
	rows, err := s.db.Query(
		"SELECT id, form_id, name, label, field_type, default_value, options, required, sort_order FROM form_fields WHERE form_id = ? ORDER BY sort_order",
		formID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []models.FormField
	for rows.Next() {
		var ff models.FormField
		var required int
		if err := rows.Scan(&ff.ID, &ff.FormID, &ff.Name, &ff.Label, &ff.FieldType, &ff.DefaultValue, &ff.Options, &required, &ff.SortOrder); err != nil {
			return nil, err
		}
		ff.Required = required == 1
		fields = append(fields, ff)
	}
	return fields, rows.Err()
}

func (s *FormStore) Create(name, description, playbookID string, serverID *string, serverGroupID *string, vaultID *string, isQuickAction bool, scheduleCron string, scheduleEnabled bool, notifyWebhook, notifyEmail string, fields []models.FormField) (*models.Form, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now()
	f := &models.Form{
		ID:              uuid.New().String(),
		Name:            name,
		Description:     description,
		PlaybookID:      playbookID,
		ServerID:        serverID,
		ServerGroupID:   serverGroupID,
		VaultID:         vaultID,
		IsQuickAction:   isQuickAction,
		ScheduleCron:    scheduleCron,
		ScheduleEnabled: scheduleEnabled,
		NotifyWebhook:   notifyWebhook,
		NotifyEmail:     notifyEmail,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = tx.Exec(
		"INSERT INTO forms (id, name, description, playbook_id, server_id, server_group_id, vault_id, is_quick_action, image_path, image_name, schedule_cron, schedule_enabled, webhook_token, notify_webhook, notify_email, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '', '', ?, ?, '', ?, ?, ?, ?)",
		f.ID, f.Name, f.Description, f.PlaybookID, f.ServerID, f.ServerGroupID, f.VaultID, boolToInt(f.IsQuickAction), f.ScheduleCron, boolToInt(f.ScheduleEnabled), f.NotifyWebhook, f.NotifyEmail, f.CreatedAt, f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	for i, ff := range fields {
		ff.ID = uuid.New().String()
		ff.FormID = f.ID
		ff.SortOrder = i
		_, err = tx.Exec(
			"INSERT INTO form_fields (id, form_id, name, label, field_type, default_value, options, required, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			ff.ID, ff.FormID, ff.Name, ff.Label, ff.FieldType, ff.DefaultValue, ff.Options, boolToInt(ff.Required), ff.SortOrder,
		)
		if err != nil {
			return nil, err
		}
		f.Fields = append(f.Fields, ff)
	}

	return f, tx.Commit()
}

func (s *FormStore) Update(id, name, description, playbookID string, serverID *string, serverGroupID *string, vaultID *string, isQuickAction bool, scheduleCron string, scheduleEnabled bool, notifyWebhook, notifyEmail string, fields []models.FormField) (*models.Form, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"UPDATE forms SET name=?, description=?, playbook_id=?, server_id=?, server_group_id=?, vault_id=?, is_quick_action=?, schedule_cron=?, schedule_enabled=?, notify_webhook=?, notify_email=?, updated_at=? WHERE id=?",
		name, description, playbookID, serverID, serverGroupID, vaultID, boolToInt(isQuickAction), scheduleCron, boolToInt(scheduleEnabled), notifyWebhook, notifyEmail, time.Now(), id,
	)
	if err != nil {
		return nil, err
	}

	// Replace all fields
	if _, err = tx.Exec("DELETE FROM form_fields WHERE form_id = ?", id); err != nil {
		return nil, err
	}
	for i, ff := range fields {
		ff.ID = uuid.New().String()
		ff.FormID = id
		ff.SortOrder = i
		_, err = tx.Exec(
			"INSERT INTO form_fields (id, form_id, name, label, field_type, default_value, options, required, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			ff.ID, ff.FormID, ff.Name, ff.Label, ff.FieldType, ff.DefaultValue, ff.Options, boolToInt(ff.Required), ff.SortOrder,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *FormStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM forms WHERE id = ?", id)
	return err
}

// ListScheduled returns all forms that have a schedule enabled, with their
// fields populated (so the scheduler can build default variables on startup).
func (s *FormStore) ListScheduled() ([]*models.Form, error) {
	rows, err := s.db.Query(formSelect + " WHERE schedule_enabled = 1 AND schedule_cron != '' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forms []*models.Form
	for rows.Next() {
		f, err := scanForm(rows)
		if err != nil {
			return nil, err
		}
		f.Fields, err = s.GetFields(f.ID)
		if err != nil {
			return nil, err
		}
		forms = append(forms, f)
	}
	return forms, rows.Err()
}

// SetImage stores the local image path and original filename for a form.
func (s *FormStore) SetImage(id, filePath, fileName string) error {
	_, err := s.db.Exec("UPDATE forms SET image_path=?, image_name=? WHERE id=?", filePath, fileName, id)
	return err
}

// ClearImage removes the image reference and returns the old local path for deletion.
func (s *FormStore) ClearImage(id string) (string, error) {
	var oldPath string
	err := s.db.QueryRow("SELECT image_path FROM forms WHERE id = ?", id).Scan(&oldPath)
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec("UPDATE forms SET image_path='', image_name='' WHERE id=?", id)
	return oldPath, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
