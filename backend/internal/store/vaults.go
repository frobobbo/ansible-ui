package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/google/uuid"
)

type VaultStore struct {
	db  *sql.DB
	key [32]byte
}

func newVaultStore(db *sql.DB, secret string) *VaultStore {
	return &VaultStore{
		db:  db,
		key: sha256.Sum256([]byte(secret)),
	}
}

func (s *VaultStore) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *VaultStore) decrypt(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	block, err := aes.NewCipher(s.key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

func (s *VaultStore) List() ([]*models.Vault, error) {
	rows, err := s.db.Query("SELECT id, name, description, vault_file_name, created_at FROM vaults ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vaults []*models.Vault
	for rows.Next() {
		v := &models.Vault{}
		if err := rows.Scan(&v.ID, &v.Name, &v.Description, &v.VaultFileName, &v.CreatedAt); err != nil {
			return nil, err
		}
		vaults = append(vaults, v)
	}
	return vaults, rows.Err()
}

func (s *VaultStore) Get(id string) (*models.Vault, error) {
	v := &models.Vault{}
	err := s.db.QueryRow(
		"SELECT id, name, description, vault_file_name, created_at FROM vaults WHERE id = ?", id,
	).Scan(&v.ID, &v.Name, &v.Description, &v.VaultFileName, &v.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return v, err
}

// GetDecryptedPassword returns the plaintext vault password. Used only at run time.
func (s *VaultStore) GetDecryptedPassword(id string) (string, error) {
	var enc string
	err := s.db.QueryRow("SELECT password_enc FROM vaults WHERE id = ?", id).Scan(&enc)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("vault not found")
	}
	if err != nil {
		return "", err
	}
	return s.decrypt(enc)
}

// GetVaultFilePath returns the local file path of the vault file (empty if none uploaded).
func (s *VaultStore) GetVaultFilePath(id string) (string, error) {
	var filePath string
	err := s.db.QueryRow("SELECT vault_file_path FROM vaults WHERE id = ?", id).Scan(&filePath)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("vault not found")
	}
	return filePath, err
}

func (s *VaultStore) Create(name, description, password string) (*models.Vault, error) {
	enc, err := s.encrypt(password)
	if err != nil {
		return nil, fmt.Errorf("encrypt password: %w", err)
	}
	v := &models.Vault{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}
	_, err = s.db.Exec(
		"INSERT INTO vaults (id, name, description, password_enc, vault_file_path, vault_file_name, created_at) VALUES (?, ?, ?, ?, '', '', ?)",
		v.ID, v.Name, v.Description, enc, v.CreatedAt,
	)
	return v, err
}

func (s *VaultStore) Update(id, name, description, password string) (*models.Vault, error) {
	if password != "" {
		enc, err := s.encrypt(password)
		if err != nil {
			return nil, fmt.Errorf("encrypt password: %w", err)
		}
		_, err = s.db.Exec(
			"UPDATE vaults SET name=?, description=?, password_enc=? WHERE id=?",
			name, description, enc, id,
		)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := s.db.Exec(
			"UPDATE vaults SET name=?, description=? WHERE id=?",
			name, description, id,
		)
		if err != nil {
			return nil, err
		}
	}
	return s.Get(id)
}

// SetVaultFile stores the local file path and original filename for a vault.
func (s *VaultStore) SetVaultFile(id, filePath, fileName string) error {
	_, err := s.db.Exec(
		"UPDATE vaults SET vault_file_path=?, vault_file_name=? WHERE id=?",
		filePath, fileName, id,
	)
	return err
}

// ClearVaultFile removes the file reference and returns the old local path for deletion.
func (s *VaultStore) ClearVaultFile(id string) (string, error) {
	var oldPath string
	err := s.db.QueryRow("SELECT vault_file_path FROM vaults WHERE id = ?", id).Scan(&oldPath)
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec("UPDATE vaults SET vault_file_path='', vault_file_name='' WHERE id=?", id)
	return oldPath, err
}

func (s *VaultStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM vaults WHERE id = ?", id)
	return err
}
