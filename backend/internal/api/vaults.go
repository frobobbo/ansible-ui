package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type VaultsHandler struct {
	vaults    *store.VaultStore
	audit     *store.AuditStore
	uploadDir string
}

func newVaultsHandler(vaults *store.VaultStore, audit *store.AuditStore, uploadDir string) *VaultsHandler {
	return &VaultsHandler{vaults: vaults, audit: audit, uploadDir: uploadDir}
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
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "create", "vault", v.ID, "", c.ClientIP())
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
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "update", "vault", id, "", c.ClientIP())
	c.JSON(http.StatusOK, v)
}

func (h *VaultsHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	// Clean up vault file if one exists
	oldPath, _ := h.vaults.ClearVaultFile(id)
	if oldPath != "" {
		os.Remove(oldPath)
	}
	if err := h.vaults.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "delete", "vault", id, "", c.ClientIP())
	c.Status(http.StatusNoContent)
}

// UploadFile handles multipart upload of the ansible-vault encrypted YAML file.
func (h *VaultsHandler) UploadFile(c *gin.Context) {
	id := c.Param("id")

	v, err := h.vaults.Get(id)
	if err != nil || v == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vault not found"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	filePath := fmt.Sprintf("%s/%s.yml", h.uploadDir, id)

	// Remove old file if present (ignore error — may not exist)
	os.Remove(filePath)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	if err := h.vaults.SetVaultFile(id, filePath, file.Filename); err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	v, _ = h.vaults.Get(id)
	c.JSON(http.StatusOK, v)
}

// DeleteFile removes the uploaded vault file from a vault.
func (h *VaultsHandler) DeleteFile(c *gin.Context) {
	id := c.Param("id")

	oldPath, err := h.vaults.ClearVaultFile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vault not found"})
		return
	}
	if oldPath != "" {
		os.Remove(oldPath)
	}

	v, _ := h.vaults.Get(id)
	c.JSON(http.StatusOK, v)
}
