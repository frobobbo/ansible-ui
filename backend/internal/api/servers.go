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
	audit   *store.AuditStore
}

func newServersHandler(servers *store.ServerStore, audit *store.AuditStore) *ServersHandler {
	return &ServersHandler{servers: servers, audit: audit}
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
		Name                 string `json:"name" binding:"required"`
		Host                 string `json:"host"`
		Port                 int    `json:"port"`
		Username             string `json:"username"`
		SSHPrivateKey        string `json:"ssh_private_key"`
		PreCommand           string `json:"pre_command"`
		ExecutionEnvironment string `json:"execution_environment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// SSH fields required when not using an execution environment.
	if req.ExecutionEnvironment == "" {
		if req.Host == "" || req.Username == "" || req.SSHPrivateKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "host, username, and ssh_private_key are required for SSH servers"})
			return
		}
		if req.Port == 0 {
			req.Port = 22
		}
	}

	sv, err := h.servers.Create(req.Name, req.Host, req.Port, req.Username, req.SSHPrivateKey, req.PreCommand, req.ExecutionEnvironment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "create", "server", sv.ID, "", c.ClientIP())
	c.JSON(http.StatusCreated, sv)
}

func (h *ServersHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name                 string `json:"name" binding:"required"`
		Host                 string `json:"host"`
		Port                 int    `json:"port"`
		Username             string `json:"username"`
		SSHPrivateKey        string `json:"ssh_private_key"`
		PreCommand           string `json:"pre_command"`
		ExecutionEnvironment string `json:"execution_environment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ExecutionEnvironment == "" {
		if req.Host == "" || req.Username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "host and username are required for SSH servers"})
			return
		}
		if req.Port == 0 {
			req.Port = 22
		}
	}

	sv, err := h.servers.Update(id, req.Name, req.Host, req.Port, req.Username, req.SSHPrivateKey, req.PreCommand, req.ExecutionEnvironment)
	if err != nil || sv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	sv.SSHPrivateKey = ""
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "update", "server", id, "", c.ClientIP())
	c.JSON(http.StatusOK, sv)
}

func (h *ServersHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.servers.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "delete", "server", id, "", c.ClientIP())
	c.Status(http.StatusNoContent)
}

func (h *ServersHandler) Test(c *gin.Context) {
	sv, err := h.servers.Get(c.Param("id"))
	if err != nil || sv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	if sv.ExecutionEnvironment != "" {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Execution Environment: " + sv.ExecutionEnvironment + " (connection test not applicable)"})
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
