package api

import (
	"net/http"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/runner"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type ServersHandler struct {
	servers *store.ServerStore
}

func newServersHandler(servers *store.ServerStore) *ServersHandler {
	return &ServersHandler{servers: servers}
}

func (h *ServersHandler) List(c *gin.Context) {
	list, err := h.servers.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*models.Server{}
	}
	c.JSON(http.StatusOK, list)
}

func (h *ServersHandler) Get(c *gin.Context) {
	sv, err := h.servers.Get(c.Param("id"))
	if err != nil || sv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	sv.SSHPrivateKey = "" // never expose key in GET
	c.JSON(http.StatusOK, sv)
}

func (h *ServersHandler) Create(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		Host          string `json:"host" binding:"required"`
		Port          int    `json:"port"`
		Username      string `json:"username" binding:"required"`
		SSHPrivateKey string `json:"ssh_private_key" binding:"required"`
		PreCommand    string `json:"pre_command"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Port == 0 {
		req.Port = 22
	}

	sv, err := h.servers.Create(req.Name, req.Host, req.Port, req.Username, req.SSHPrivateKey, req.PreCommand)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sv)
}

func (h *ServersHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name          string `json:"name" binding:"required"`
		Host          string `json:"host" binding:"required"`
		Port          int    `json:"port"`
		Username      string `json:"username" binding:"required"`
		SSHPrivateKey string `json:"ssh_private_key"`
		PreCommand    string `json:"pre_command"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Port == 0 {
		req.Port = 22
	}

	sv, err := h.servers.Update(id, req.Name, req.Host, req.Port, req.Username, req.SSHPrivateKey, req.PreCommand)
	if err != nil || sv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	sv.SSHPrivateKey = ""
	c.JSON(http.StatusOK, sv)
}

func (h *ServersHandler) Delete(c *gin.Context) {
	if err := h.servers.Delete(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ServersHandler) Test(c *gin.Context) {
	sv, err := h.servers.Get(c.Param("id"))
	if err != nil || sv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	client, err := runner.Connect(sv.Host, sv.Port, sv.Username, sv.SSHPrivateKey)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer client.Close()

	out, err := client.RunCommand("echo ok")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "SSH connection successful: " + out})
}
