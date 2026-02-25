package api

import (
	"net/http"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type FormsHandler struct {
	forms *store.FormStore
}

func newFormsHandler(forms *store.FormStore) *FormsHandler {
	return &FormsHandler{forms: forms}
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

type formRequest struct {
	Name        string             `json:"name" binding:"required"`
	Description string             `json:"description"`
	PlaybookID  string             `json:"playbook_id" binding:"required"`
	ServerID    string             `json:"server_id" binding:"required"`
	Fields      []models.FormField `json:"fields"`
}

func (h *FormsHandler) Create(c *gin.Context) {
	var req formRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := h.forms.Create(req.Name, req.Description, req.PlaybookID, req.ServerID, req.Fields)
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

	f, err := h.forms.Update(id, req.Name, req.Description, req.PlaybookID, req.ServerID, req.Fields)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "form not found"})
		return
	}
	c.JSON(http.StatusOK, f)
}

func (h *FormsHandler) Delete(c *gin.Context) {
	if err := h.forms.Delete(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
