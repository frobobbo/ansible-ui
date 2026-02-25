package api

import (
	"net/http"

	"github.com/brettjrea/ansible-frontend/internal/auth"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	users  *store.UserStore
	jwtSvc *auth.JWTService
}

func newAuthHandler(users *store.UserStore, jwtSvc *auth.JWTService) *AuthHandler {
	return &AuthHandler{users: users, jwtSvc: jwtSvc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := h.jwtSvc.Sign(user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT is stateless; logout is handled client-side by discarding the token
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
