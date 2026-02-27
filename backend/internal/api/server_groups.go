package api

import (
	"net/http"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type ServerGroupsHandler struct {
	groups *store.ServerGroupStore
	audit  *store.AuditStore
}

func newServerGroupsHandler(groups *store.ServerGroupStore, audit *store.AuditStore) *ServerGroupsHandler {
	return &ServerGroupsHandler{groups: groups, audit: audit}
}

func (h *ServerGroupsHandler) List(c *gin.Context) {
	list, err := h.groups.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*models.ServerGroup{}
	}
	c.JSON(http.StatusOK, list)
}

func (h *ServerGroupsHandler) Get(c *gin.Context) {
	g, err := h.groups.Get(c.Param("id"))
	if err != nil || g == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server group not found"})
		return
	}
	c.JSON(http.StatusOK, g)
}

func (h *ServerGroupsHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g, err := h.groups.Create(req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "create", "server-group", g.ID, "", c.ClientIP())
	c.JSON(http.StatusCreated, g)
}

func (h *ServerGroupsHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g, err := h.groups.Update(id, req.Name, req.Description)
	if err != nil || g == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server group not found"})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "update", "server-group", id, "", c.ClientIP())
	c.JSON(http.StatusOK, g)
}

func (h *ServerGroupsHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.groups.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "delete", "server-group", id, "", c.ClientIP())
	c.Status(http.StatusNoContent)
}

func (h *ServerGroupsHandler) GetMembers(c *gin.Context) {
	members, err := h.groups.GetMembers(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if members == nil {
		members = []*models.Server{}
	}
	// Blank out SSH private keys before returning
	for _, s := range members {
		s.SSHPrivateKey = ""
	}
	c.JSON(http.StatusOK, members)
}

func (h *ServerGroupsHandler) SetMembers(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		ServerIDs []string `json:"server_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.groups.SetMembers(id, req.ServerIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "update", "server-group-members", id, "", c.ClientIP())
	c.Status(http.StatusNoContent)
}
