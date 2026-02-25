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

func (s *FormStore) List() ([]*models.Form, error) {
	rows, err := s.db.Query(
		"SELECT id, name, description, playbook_id, server_id, created_at, updated_at FROM forms ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forms []*models.Form
	for rows.Next() {
		f := &models.Form{}
		if err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.PlaybookID, &f.ServerID, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		forms = append(forms, f)
	}
	return forms, rows.Err()
}

func (s *FormStore) Get(id string) (*models.Form, error) {
	f := &models.Form{}
	err := s.db.QueryRow(
		"SELECT id, name, description, playbook_id, server_id, created_at, updated_at FROM forms WHERE id = ?", id,
	).Scan(&f.ID, &f.Name, &f.Description, &f.PlaybookID, &f.ServerID, &f.CreatedAt, &f.UpdatedAt)
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

func (s *FormStore) Create(name, description, playbookID, serverID string, fields []models.FormField) (*models.Form, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now()
	f := &models.Form{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		PlaybookID:  playbookID,
		ServerID:    serverID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = tx.Exec(
		"INSERT INTO forms (id, name, description, playbook_id, server_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		f.ID, f.Name, f.Description, f.PlaybookID, f.ServerID, f.CreatedAt, f.UpdatedAt,
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

func (s *FormStore) Update(id, name, description, playbookID, serverID string, fields []models.FormField) (*models.Form, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"UPDATE forms SET name=?, description=?, playbook_id=?, server_id=?, updated_at=? WHERE id=?",
		name, description, playbookID, serverID, time.Now(), id,
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
