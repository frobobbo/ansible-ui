package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type FormsHandler struct {
	forms    *store.FormStore
	imageDir string
}

func newFormsHandler(forms *store.FormStore, imageDir string) *FormsHandler {
	return &FormsHandler{forms: forms, imageDir: imageDir}
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
	c.JSON(http.StatusOK, list)
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
	c.JSON(http.StatusOK, list)
}

func (h *FormsHandler) Get(c *gin.Context) {
	f, err := h.forms.Get(c.Param("id"))
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "form not found"})
		return
	}
	c.JSON(http.StatusOK, f)
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
	c.JSON(http.StatusOK, f)
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
	c.JSON(http.StatusOK, f)
}

type formRequest struct {
	Name          string             `json:"name" binding:"required"`
	Description   string             `json:"description"`
	PlaybookID    string             `json:"playbook_id" binding:"required"`
	ServerID      string             `json:"server_id" binding:"required"`
	VaultID       *string            `json:"vault_id"`
	IsQuickAction bool               `json:"is_quick_action"`
	Fields        []models.FormField `json:"fields"`
}

func (h *FormsHandler) Create(c *gin.Context) {
	var req formRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vaultID := req.VaultID
	if vaultID != nil && *vaultID == "" {
		vaultID = nil
	}

	f, err := h.forms.Create(req.Name, req.Description, req.PlaybookID, req.ServerID, vaultID, req.IsQuickAction, req.Fields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, f)
}

func (h *FormsHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req formRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vaultID := req.VaultID
	if vaultID != nil && *vaultID == "" {
		vaultID = nil
	}

	f, err := h.forms.Update(id, req.Name, req.Description, req.PlaybookID, req.ServerID, vaultID, req.IsQuickAction, req.Fields)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "form not found"})
		return
	}
	c.JSON(http.StatusOK, f)
}

func (h *FormsHandler) Delete(c *gin.Context) {
	// Clean up image file if present
	if oldPath, err := h.forms.ClearImage(c.Param("id")); err == nil && oldPath != "" {
		os.Remove(oldPath)
	}
	if err := h.forms.Delete(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
