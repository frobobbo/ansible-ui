package api

import (
	"net/http"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	users *store.UserStore
	audit *store.AuditStore
}

func newUsersHandler(users *store.UserStore, audit *store.AuditStore) *UsersHandler {
	return &UsersHandler{users: users, audit: audit}
}

func (h *UsersHandler) List(c *gin.Context) {
	users, err := h.users.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if users == nil {
		users = []*models.User{}
	}
	c.JSON(http.StatusOK, users)
}

func (h *UsersHandler) Create(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required,oneof=admin editor viewer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user, err := h.users.Create(req.Username, string(hash), req.Role)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "create", "user", user.ID, "", c.ClientIP())
	c.JSON(http.StatusCreated, user)
}

func (h *UsersHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password"`
		Role     string `json:"role" binding:"required,oneof=admin editor viewer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var hash string
	if req.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		hash = string(h)
	}

	user, err := h.users.Update(id, req.Username, hash, req.Role)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "update", "user", id, "", c.ClientIP())
	c.JSON(http.StatusOK, user)
}

func (h *UsersHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.users.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "delete", "user", id, "", c.ClientIP())
	c.Status(http.StatusNoContent)
}
