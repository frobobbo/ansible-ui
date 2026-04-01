package api

import (
	"net/http"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type HostsHandler struct {
	hosts *store.HostStore
	audit *store.AuditStore
}

func newHostsHandler(hosts *store.HostStore, audit *store.AuditStore) *HostsHandler {
	return &HostsHandler{hosts: hosts, audit: audit}
}

func NewHostsHandler(hosts *store.HostStore, audit *store.AuditStore) *HostsHandler {
	return newHostsHandler(hosts, audit)
}

func (h *HostsHandler) List(c *gin.Context) {
	list, err := h.hosts.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*models.Host{}
	}
	c.JSON(http.StatusOK, list)
}

func (h *HostsHandler) Get(c *gin.Context) {
	host, err := h.hosts.Get(c.Param("id"))
	if err != nil || host == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}
	c.JSON(http.StatusOK, host)
}

func (h *HostsHandler) Create(c *gin.Context) {
	var req struct {
		Name        string            `json:"name" binding:"required"`
		Address     string            `json:"address" binding:"required"`
		Description string            `json:"description"`
		SSHCertID   *string           `json:"ssh_cert_id"`
		Vars        map[string]string `json:"vars"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	host, err := h.hosts.Create(req.Name, req.Address, req.Description, req.SSHCertID, req.Vars)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "create", "host", host.ID, "", c.ClientIP())
	c.JSON(http.StatusCreated, host)
}

func (h *HostsHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string            `json:"name" binding:"required"`
		Address     string            `json:"address" binding:"required"`
		Description string            `json:"description"`
		SSHCertID   *string           `json:"ssh_cert_id"`
		Vars        map[string]string `json:"vars"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	host, err := h.hosts.Update(id, req.Name, req.Address, req.Description, req.SSHCertID, req.Vars)
	if err != nil || host == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "update", "host", id, "", c.ClientIP())
	c.JSON(http.StatusOK, host)
}

func (h *HostsHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.hosts.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "delete", "host", id, "", c.ClientIP())
	c.Status(http.StatusNoContent)
}
