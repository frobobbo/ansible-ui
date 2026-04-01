package api

import (
	"net/http"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type PlaybooksHandler struct {
	playbooks *store.PlaybookStore
	audit     *store.AuditStore
}

func newPlaybooksHandler(playbooks *store.PlaybookStore, audit *store.AuditStore) *PlaybooksHandler {
	return &PlaybooksHandler{playbooks: playbooks, audit: audit}
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
		c.JSON(http.StatusNotFound, gin.H{"error": "playbook source not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

type playbookBody struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	RepoURL      string `json:"repo_url"`
	Branch       string `json:"branch"`
	PlaybookPath string `json:"playbook_path"`
	Token        string `json:"token"`
}

func (h *PlaybooksHandler) Create(c *gin.Context) {
	var body playbookBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Name == "" || body.RepoURL == "" || body.PlaybookPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, repo_url, and playbook_path are required"})
		return
	}
	if body.Branch == "" {
		body.Branch = "main"
	}

	p, err := h.playbooks.Create(body.Name, body.Description, body.RepoURL, body.Branch, body.PlaybookPath, body.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "create", "playbook_source", p.ID, "", c.ClientIP())
	c.JSON(http.StatusCreated, p)
}

func (h *PlaybooksHandler) Update(c *gin.Context) {
	var body playbookBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Name == "" || body.RepoURL == "" || body.PlaybookPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, repo_url, and playbook_path are required"})
		return
	}
	if body.Branch == "" {
		body.Branch = "main"
	}

	p, err := h.playbooks.Update(c.Param("id"), body.Name, body.Description, body.RepoURL, body.Branch, body.PlaybookPath, body.Token)
	if err != nil || p == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "update", "playbook_source", p.ID, "", c.ClientIP())
	c.JSON(http.StatusOK, p)
}

func (h *PlaybooksHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.playbooks.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "delete", "playbook_source", id, "", c.ClientIP())
	c.Status(http.StatusNoContent)
}
