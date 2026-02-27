package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PlaybooksHandler struct {
	playbooks *store.PlaybookStore
	audit     *store.AuditStore
	uploadDir string
}

func newPlaybooksHandler(playbooks *store.PlaybookStore, audit *store.AuditStore, uploadDir string) *PlaybooksHandler {
	return &PlaybooksHandler{playbooks: playbooks, audit: audit, uploadDir: uploadDir}
}

func (h *PlaybooksHandler) List(c *gin.Context) {
	list, err := h.playbooks.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*models.Playbook{}
	}
	c.JSON(http.StatusOK, list)
}

func (h *PlaybooksHandler) Get(c *gin.Context) {
	p, err := h.playbooks.Get(c.Param("id"))
	if err != nil || p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playbook not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *PlaybooksHandler) Upload(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// Use a UUID as the filename to avoid path traversal
	id := uuid.New().String()
	filePath := fmt.Sprintf("%s/%s.yml", h.uploadDir, id)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	p, err := h.playbooks.Create(name, description, filePath)
	if err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "upload", "playbook", p.ID, "", c.ClientIP())
	c.JSON(http.StatusCreated, p)
}

func (h *PlaybooksHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	filePath, err := h.playbooks.Delete(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if filePath != "" {
		os.Remove(filePath)
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "delete", "playbook", id, "", c.ClientIP())
	c.Status(http.StatusNoContent)
}
