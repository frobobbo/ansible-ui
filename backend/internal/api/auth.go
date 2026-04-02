package api

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/auth"
	"github.com/brettjrea/ansible-frontend/internal/notify"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// loginLimiter allows at most 10 login attempts per IP per minute.
var loginLimiter = &rateLimiter{max: 10, window: time.Minute, entries: map[string]*rlEntry{}}

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rlEntry
	window  time.Duration
	max     int
}

type rlEntry struct {
	count     int
	windowEnd time.Time
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	e, ok := rl.entries[ip]
	if !ok || now.After(e.windowEnd) {
		rl.entries[ip] = &rlEntry{count: 1, windowEnd: now.Add(rl.window)}
		return true
	}
	if e.count >= rl.max {
		return false
	}
	e.count++
	return true
}

type AuthHandler struct {
	users  *store.UserStore
	jwtSvc *auth.JWTService
}

func newAuthHandler(users *store.UserStore, jwtSvc *auth.JWTService) *AuthHandler {
	return &AuthHandler{users: users, jwtSvc: jwtSvc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	if !loginLimiter.allow(c.ClientIP()) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts, try again later"})
		return
	}
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

// ForgotPassword accepts {"username":"..."} and sends a reset email if the
// account has an email address. Always returns 200 to prevent username enumeration.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
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

	// Always respond 200 regardless of whether the user or email exists
	const msg = "If an account with that username and a configured email address exists, a reset link has been sent."

	if user == nil || user.Email == "" {
		c.JSON(http.StatusOK, gin.H{"message": msg})
		return
	}

	token, err := h.users.CreateResetToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create reset token"})
		return
	}

	baseURL := os.Getenv("APP_URL")
	if baseURL == "" {
		scheme := "https"
		if c.Request.TLS == nil && c.GetHeader("X-Forwarded-Proto") != "https" {
			scheme = "http"
		}
		baseURL = scheme + "://" + c.Request.Host
	}
	resetURL := baseURL + "/reset-password?token=" + token

	go notify.SendPasswordReset(user.Email, resetURL)

	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// ResetPassword accepts {"token":"...","password":"..."} and updates the password.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token    string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := h.users.ConsumeResetToken(req.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired reset token"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	if err := h.users.UpdatePassword(userID, string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}
