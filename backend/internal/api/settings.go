package api

import (
	"net/http"

	"github.com/brettjrea/ansible-frontend/internal/notify"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	settings *store.SettingsStore
	users    *store.UserStore
}

func newSettingsHandler(settings *store.SettingsStore, users *store.UserStore) *SettingsHandler {
	return &SettingsHandler{settings: settings, users: users}
}

var githubSettingKeys = []string{
	"github_token", "github_repo", "github_branch",
}

var emailSettingKeys = []string{
	"email_provider",
	"smtp_host", "smtp_port", "smtp_username", "smtp_password", "smtp_from",
	"mailgun_api_key", "mailgun_domain", "mailgun_from", "mailgun_region",
}

func (h *SettingsHandler) GetEmail(c *gin.Context) {
	all, err := h.settings.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	result := map[string]string{}
	for _, k := range emailSettingKeys {
		result[k] = all[k]
	}
	c.JSON(http.StatusOK, result)
}

func (h *SettingsHandler) UpdateEmail(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Only persist known email keys
	toSave := map[string]string{}
	for _, k := range emailSettingKeys {
		if v, ok := req[k]; ok {
			toSave[k] = v
		}
	}

	if err := h.settings.SetMany(toSave); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload the active notify config from the updated settings
	all, _ := h.settings.GetAll()
	notify.SetConfig(notify.ConfigFromSettings(all))

	c.JSON(http.StatusOK, toSave)
}

func (h *SettingsHandler) GetGitHub(c *gin.Context) {
	all, err := h.settings.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	result := map[string]string{}
	for _, k := range githubSettingKeys {
		result[k] = all[k]
	}
	c.JSON(http.StatusOK, result)
}

func (h *SettingsHandler) UpdateGitHub(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	toSave := map[string]string{}
	for _, k := range githubSettingKeys {
		if v, ok := req[k]; ok {
			toSave[k] = v
		}
	}
	if err := h.settings.SetMany(toSave); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toSave)
}

func (h *SettingsHandler) TestEmail(c *gin.Context) {
	var req struct {
		To     string            `json:"to" binding:"required"`
		Config map[string]string `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If the caller provides an inline config (before saving), use that; otherwise use active config.
	var cfg notify.Config
	if len(req.Config) > 0 {
		cfg = notify.ConfigFromSettings(req.Config)
	} else {
		cfg = notify.GetConfig()
	}

	if err := notify.SendTest(cfg, req.To); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test email sent to " + req.To})
}
