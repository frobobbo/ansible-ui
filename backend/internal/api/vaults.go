package api

import (
	"net/http"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type VaultsHandler struct {
	vaults *store.VaultStore
}

func newVaultsHandler(vaults *store.VaultStore) *VaultsHandler {
	return &VaultsHandler{vaults: vaults}
}

func (h *VaultsHandler) List(c *gin.Context) {
	list, err := h.vaults.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*models.Vault{}
	}
	c.JSON(http.StatusOK, list)
}

func (h *VaultsHandler) Get(c *gin.Context) {
	v, err := h.vaults.Get(c.Param("id"))
	if err != nil || v == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vault not found"})
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *VaultsHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Password    string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	v, err := h.vaults.Create(req.Name, req.Description, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *VaultsHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Password    string `json:"password"` // optional on update
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	v, err := h.vaults.Update(id, req.Name, req.Description, req.Password)
	if err != nil || v == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vault not found"})
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *VaultsHandler) Delete(c *gin.Context) {
	if err := h.vaults.Delete(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
