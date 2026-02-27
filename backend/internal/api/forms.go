package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/scheduler"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type FormsHandler struct {
	forms    *store.FormStore
	audit    *store.AuditStore
	imageDir string
	sched    *scheduler.Scheduler
}

func newFormsHandler(forms *store.FormStore, audit *store.AuditStore, imageDir string, sched *scheduler.Scheduler) *FormsHandler {
	return &FormsHandler{forms: forms, audit: audit, imageDir: imageDir, sched: sched}
}

// formResponse wraps a Form and adds the computed next_run_at field.
type formResponse struct {
	*models.Form
	NextRunAt *time.Time `json:"next_run_at"`
}

func (h *FormsHandler) withNextRun(f *models.Form) formResponse {
	var next *time.Time
	if h.sched != nil {
		next = h.sched.NextRunAt(f.ID)
	}
	return formResponse{Form: f, NextRunAt: next}
}

func (h *FormsHandler) List(c *gin.Context) {
	list, err := h.forms.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*models.Form{}
	}
	resp := make([]formResponse, len(list))
	for i, f := range list {
		resp[i] = h.withNextRun(f)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *FormsHandler) GetQuickActions(c *gin.Context) {
	list, err := h.forms.GetQuickActions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*models.Form{}
	}
	resp := make([]formResponse, len(list))
	for i, f := range list {
		resp[i] = h.withNextRun(f)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *FormsHandler) Get(c *gin.Context) {
	f, err := h.forms.Get(c.Param("id"))
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "form not found"})
		return
	}
	c.JSON(http.StatusOK, h.withNextRun(f))
}

func (h *FormsHandler) GetFields(c *gin.Context) {
	fields, err := h.forms.GetFields(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if fields == nil {
		fields = []models.FormField{}
	}
	c.JSON(http.StatusOK, fields)
}

// GetImage serves the form's image file. Registered outside the auth group so
// browser <img> tags can load it without Bearer token plumbing.
func (h *FormsHandler) GetImage(c *gin.Context) {
	f, err := h.forms.Get(c.Param("id"))
	if err != nil || f == nil || f.ImageName == "" {
		c.Status(http.StatusNotFound)
		return
	}
	imagePath := filepath.Join(h.imageDir, fmt.Sprintf("%s%s", c.Param("id"), filepath.Ext(f.ImageName)))
	c.File(imagePath)
}

func (h *FormsHandler) UploadImage(c *gin.Context) {
	id := c.Param("id")
	f, err := h.forms.Get(id)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "form not found"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	ext := filepath.Ext(file.Filename)
	filePath := filepath.Join(h.imageDir, id+ext)
	// Remove any previous image with a different extension
	os.Remove(filepath.Join(h.imageDir, id+".png"))
	os.Remove(filepath.Join(h.imageDir, id+".jpg"))
	os.Remove(filepath.Join(h.imageDir, id+".jpeg"))
	os.Remove(filepath.Join(h.imageDir, id+".gif"))
	os.Remove(filepath.Join(h.imageDir, id+".webp"))
	os.Remove(filepath.Join(h.imageDir, id+".svg"))

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save image"})
		return
	}

	if err := h.forms.SetImage(id, filePath, file.Filename); err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	f, _ = h.forms.Get(id)
	c.JSON(http.StatusOK, h.withNextRun(f))
}

func (h *FormsHandler) DeleteImage(c *gin.Context) {
	id := c.Param("id")
	oldPath, err := h.forms.ClearImage(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "form not found"})
		return
	}
	if oldPath != "" {
		os.Remove(oldPath)
	}
	f, _ := h.forms.Get(id)
	c.JSON(http.StatusOK, h.withNextRun(f))
}

// RegenerateWebhookToken creates a new random webhook token for the form.
func (h *FormsHandler) RegenerateWebhookToken(c *gin.Context) {
	id := c.Param("id")
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	token := hex.EncodeToString(b)
	if err := h.forms.SetWebhookToken(id, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	f, _ := h.forms.Get(id)
	c.JSON(http.StatusOK, h.withNextRun(f))
}

// RevokeWebhookToken clears the webhook token, disabling webhook triggers.
func (h *FormsHandler) RevokeWebhookToken(c *gin.Context) {
	id := c.Param("id")
	if err := h.forms.SetWebhookToken(id, ""); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	f, _ := h.forms.Get(id)
	c.JSON(http.StatusOK, h.withNextRun(f))
}

type formRequest struct {
	Name            string             `json:"name" binding:"required"`
	Description     string             `json:"description"`
	PlaybookID      string             `json:"playbook_id" binding:"required"`
	ServerID        string             `json:"server_id" binding:"required"`
	VaultID         *string            `json:"vault_id"`
	IsQuickAction   bool               `json:"is_quick_action"`
	ScheduleCron    string             `json:"schedule_cron"`
	ScheduleEnabled bool               `json:"schedule_enabled"`
	NotifyWebhook   string             `json:"notify_webhook"`
	NotifyEmail     string             `json:"notify_email"`
	Fields          []models.FormField `json:"fields"`
}

func (h *FormsHandler) Create(c *gin.Context) {
	var req formRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := scheduler.ValidateCron(req.ScheduleCron); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron expression: " + err.Error()})
		return
	}

	vaultID := req.VaultID
	if vaultID != nil && *vaultID == "" {
		vaultID = nil
	}

	f, err := h.forms.Create(req.Name, req.Description, req.PlaybookID, req.ServerID, vaultID, req.IsQuickAction, req.ScheduleCron, req.ScheduleEnabled, req.NotifyWebhook, req.NotifyEmail, req.Fields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.sched != nil {
		h.sched.Upsert(f)
	}

	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "create", "form", f.ID, "", c.ClientIP())
	c.JSON(http.StatusCreated, h.withNextRun(f))
}

func (h *FormsHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req formRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := scheduler.ValidateCron(req.ScheduleCron); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron expression: " + err.Error()})
		return
	}

	vaultID := req.VaultID
	if vaultID != nil && *vaultID == "" {
		vaultID = nil
	}

	f, err := h.forms.Update(id, req.Name, req.Description, req.PlaybookID, req.ServerID, vaultID, req.IsQuickAction, req.ScheduleCron, req.ScheduleEnabled, req.NotifyWebhook, req.NotifyEmail, req.Fields)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "form not found"})
		return
	}

	if h.sched != nil {
		h.sched.Upsert(f)
	}

	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "update", "form", id, "", c.ClientIP())
	c.JSON(http.StatusOK, h.withNextRun(f))
}

func (h *FormsHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if h.sched != nil {
		h.sched.Remove(id)
	}

	// Clean up image file if present
	if oldPath, err := h.forms.ClearImage(id); err == nil && oldPath != "" {
		os.Remove(oldPath)
	}
	if err := h.forms.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "delete", "form", id, "", c.ClientIP())
	c.Status(http.StatusNoContent)
}
