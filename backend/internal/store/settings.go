package store

import "database/sql"

type SettingsStore struct {
	db *sql.DB
}

func (s *SettingsStore) GetAll() (map[string]string, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}

func (s *SettingsStore) SetMany(m map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	stmt, err := tx.Prepare(
		"INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for k, v := range m {
		if _, err := stmt.Exec(k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}
