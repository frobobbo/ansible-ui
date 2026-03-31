package api

import (
	"io"
	"net/http"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type SSHCertsHandler struct {
	certs *store.SSHCertStore
	audit *store.AuditStore
}

func newSSHCertsHandler(certs *store.SSHCertStore, audit *store.AuditStore) *SSHCertsHandler {
	return &SSHCertsHandler{certs: certs, audit: audit}
}

// NewSSHCertsHandler is the exported constructor used by main.go.
func NewSSHCertsHandler(certs *store.SSHCertStore, audit *store.AuditStore) *SSHCertsHandler {
	return newSSHCertsHandler(certs, audit)
}

func (h *SSHCertsHandler) List(c *gin.Context) {
	list, err := h.certs.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*models.SSHCert{}
	}
	c.JSON(http.StatusOK, list)
}

func (h *SSHCertsHandler) Get(c *gin.Context) {
	cert, err := h.certs.Get(c.Param("id"))
	if err != nil || cert == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ssh cert not found"})
		return
	}
	c.JSON(http.StatusOK, cert)
}

func (h *SSHCertsHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cert, err := h.certs.Create(req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "create", "ssh-cert", cert.ID, "", c.ClientIP())
	c.JSON(http.StatusCreated, cert)
}

func (h *SSHCertsHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cert, err := h.certs.Update(id, req.Name, req.Description)
	if err != nil || cert == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ssh cert not found"})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "update", "ssh-cert", id, "", c.ClientIP())
	c.JSON(http.StatusOK, cert)
}

func (h *SSHCertsHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.certs.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "delete", "ssh-cert", id, "", c.ClientIP())
	c.Status(http.StatusNoContent)
}

// Upload accepts a multipart file upload, reads the cert bytes, encrypts them,
// and stores them in the database — no plaintext cert ever touches the filesystem.
func (h *SSHCertsHandler) Upload(c *gin.Context) {
	id := c.Param("id")

	cert, err := h.certs.Get(id)
	if err != nil || cert == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ssh cert not found"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
		return
	}

	if err := h.certs.SetCert(id, file.Filename, content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "upload", "ssh-cert", id, "", c.ClientIP())

	updated, _ := h.certs.Get(id)
	c.JSON(http.StatusOK, updated)
}

// DeleteFile removes the stored certificate from a record without deleting the record itself.
func (h *SSHCertsHandler) DeleteFile(c *gin.Context) {
	id := c.Param("id")
	if err := h.certs.ClearCert(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ssh cert not found"})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "delete-file", "ssh-cert", id, "", c.ClientIP())
	cert, _ := h.certs.Get(id)
	c.JSON(http.StatusOK, cert)
}
