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

// SSHCertStore stores SSH certificates encrypted at rest using AES-256-GCM.
// The encryption key is derived from the application's JWT secret so no
// separate secret management is required.
type SSHCertStore struct {
	db  *sql.DB
	key [32]byte
}

func newSSHCertStore(db *sql.DB, secret string) *SSHCertStore {
	return &SSHCertStore{
		db:  db,
		key: sha256.Sum256([]byte(secret)),
	}
}

func (s *SSHCertStore) encryptBytes(plaintext []byte) (string, error) {
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
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *SSHCertStore) decryptBytes(encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	block, err := aes.NewCipher(s.key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

func (s *SSHCertStore) List() ([]*models.SSHCert, error) {
	rows, err := s.db.Query(
		"SELECT id, name, description, file_name, created_at FROM ssh_certs ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []*models.SSHCert
	for rows.Next() {
		c := &models.SSHCert{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.FileName, &c.CreatedAt); err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return certs, rows.Err()
}

func (s *SSHCertStore) Get(id string) (*models.SSHCert, error) {
	c := &models.SSHCert{}
	err := s.db.QueryRow(
		"SELECT id, name, description, file_name, created_at FROM ssh_certs WHERE id = ?", id,
	).Scan(&c.ID, &c.Name, &c.Description, &c.FileName, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// GetDecryptedCert returns the plaintext certificate bytes. Used at run time
// to pass the cert to a runner (SSH or Execution Environment).
func (s *SSHCertStore) GetDecryptedCert(id string) ([]byte, error) {
	var enc string
	err := s.db.QueryRow("SELECT cert_enc FROM ssh_certs WHERE id = ?", id).Scan(&enc)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("ssh cert not found")
	}
	if err != nil {
		return nil, err
	}
	if enc == "" {
		return nil, fmt.Errorf("no certificate uploaded for this record")
	}
	return s.decryptBytes(enc)
}

func (s *SSHCertStore) Create(name, description string) (*models.SSHCert, error) {
	c := &models.SSHCert{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}
	_, err := s.db.Exec(
		"INSERT INTO ssh_certs (id, name, description, file_name, cert_enc, created_at) VALUES (?, ?, ?, '', '', ?)",
		c.ID, c.Name, c.Description, c.CreatedAt,
	)
	return c, err
}

func (s *SSHCertStore) Update(id, name, description string) (*models.SSHCert, error) {
	_, err := s.db.Exec(
		"UPDATE ssh_certs SET name=?, description=? WHERE id=?",
		name, description, id,
	)
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

// SetCert encrypts certContent and stores it alongside the original filename.
func (s *SSHCertStore) SetCert(id, fileName string, certContent []byte) error {
	enc, err := s.encryptBytes(certContent)
	if err != nil {
		return fmt.Errorf("encrypt cert: %w", err)
	}
	_, err = s.db.Exec(
		"UPDATE ssh_certs SET file_name=?, cert_enc=? WHERE id=?",
		fileName, enc, id,
	)
	return err
}

// ClearCert removes the stored certificate from a record.
func (s *SSHCertStore) ClearCert(id string) error {
	_, err := s.db.Exec(
		"UPDATE ssh_certs SET file_name='', cert_enc='' WHERE id=?", id,
	)
	return err
}

func (s *SSHCertStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM ssh_certs WHERE id = ?", id)
	return err
}
