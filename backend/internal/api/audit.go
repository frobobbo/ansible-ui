package api

import (
	"net/http"
	"strconv"

	"github.com/brettjrea/ansible-frontend/internal/auth"
	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	audit *store.AuditStore
}

func newAuditHandler(audit *store.AuditStore) *AuditHandler {
	return &AuditHandler{audit: audit}
}

func (h *AuditHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, total, err := h.audit.List(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if logs == nil {
		logs = []*models.AuditLog{}
	}
	c.Header("X-Total-Count", strconv.Itoa(total))
	c.JSON(http.StatusOK, logs)
}

// auditUser extracts the user ID and username from the JWT claims in the context.
func auditUser(c *gin.Context) (userID, username string) {
	if cl := auth.GetClaims(c); cl != nil {
		return cl.UserID, cl.Username
	}
	return "", ""
}
