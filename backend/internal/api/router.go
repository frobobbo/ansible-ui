package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/brettjrea/ansible-frontend/internal/auth"
	"github.com/brettjrea/ansible-frontend/internal/scheduler"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

func NewRouter(db *store.DB, jwtSvc *auth.JWTService, uploadDir string, vaultUploadDir string, formImageDir string, jwtSecret string, runsH *RunsHandler, sched *scheduler.Scheduler) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Handlers
	auditStore := db.Audit()
	vaultStore := db.Vaults(jwtSecret)
	authH := newAuthHandler(db.Users(), jwtSvc)
	auditH := newAuditHandler(auditStore)
	usersH := newUsersHandler(db.Users(), auditStore)
	serversH := newServersHandler(db.Servers(), auditStore)
	serverGroupsH := newServerGroupsHandler(db.ServerGroups(), auditStore)
	playbooksH := newPlaybooksHandler(db.Playbooks(), auditStore, uploadDir)
	formsH := newFormsHandler(db.Forms(), auditStore, formImageDir, sched)
	vaultsH := newVaultsHandler(vaultStore, auditStore, vaultUploadDir)

	// Health check — no auth required
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes
	api := r.Group("/api")
	{
		api.POST("/auth/login", authH.Login)
		api.POST("/auth/logout", authH.Logout)

		protected := api.Group("/")
		protected.Use(auth.Middleware(jwtSvc))
		{
			// Users (admin-only writes)
			protected.GET("/users", usersH.List)
			protected.POST("/users", auth.RequireAdmin, usersH.Create)
			protected.PUT("/users/:id", auth.RequireAdmin, usersH.Update)
			protected.DELETE("/users/:id", auth.RequireAdmin, usersH.Delete)

			// Servers
			protected.GET("/servers", serversH.List)
			protected.GET("/servers/:id", serversH.Get)
			protected.POST("/servers", auth.RequireAdmin, serversH.Create)
			protected.PUT("/servers/:id", auth.RequireAdmin, serversH.Update)
			protected.DELETE("/servers/:id", auth.RequireAdmin, serversH.Delete)
			protected.POST("/servers/:id/test", serversH.Test)

			// Server groups
			protected.GET("/server-groups", serverGroupsH.List)
			protected.GET("/server-groups/:id", serverGroupsH.Get)
			protected.POST("/server-groups", auth.RequireAdmin, serverGroupsH.Create)
			protected.PUT("/server-groups/:id", auth.RequireAdmin, serverGroupsH.Update)
			protected.DELETE("/server-groups/:id", auth.RequireAdmin, serverGroupsH.Delete)
			protected.GET("/server-groups/:id/members", serverGroupsH.GetMembers)
			protected.PUT("/server-groups/:id/members", auth.RequireAdmin, serverGroupsH.SetMembers)

			// Playbooks
			protected.GET("/playbooks", playbooksH.List)
			protected.GET("/playbooks/:id", playbooksH.Get)
			protected.POST("/playbooks", auth.RequireAdmin, playbooksH.Upload)
			protected.DELETE("/playbooks/:id", auth.RequireAdmin, playbooksH.Delete)

			// Forms
			protected.GET("/forms", formsH.List)
			protected.GET("/forms/:id", formsH.Get)
			protected.GET("/forms/:id/fields", formsH.GetFields)
			protected.POST("/forms", auth.RequireEditor, formsH.Create)
			protected.PUT("/forms/:id", auth.RequireEditor, formsH.Update)
			protected.DELETE("/forms/:id", auth.RequireEditor, formsH.Delete)
			protected.POST("/forms/:id/image", auth.RequireEditor, formsH.UploadImage)
			protected.DELETE("/forms/:id/image", auth.RequireEditor, formsH.DeleteImage)
			protected.POST("/forms/:id/webhook-token", auth.RequireEditor, formsH.RegenerateWebhookToken)
			protected.DELETE("/forms/:id/webhook-token", auth.RequireEditor, formsH.RevokeWebhookToken)

			// Quick actions — accessible to all authenticated users (including viewers)
			protected.GET("/quick-actions", formsH.GetQuickActions)

			// Vaults (admin-only writes)
			protected.GET("/vaults", vaultsH.List)
			protected.GET("/vaults/:id", vaultsH.Get)
			protected.POST("/vaults", auth.RequireAdmin, vaultsH.Create)
			protected.PUT("/vaults/:id", auth.RequireAdmin, vaultsH.Update)
			protected.DELETE("/vaults/:id", auth.RequireAdmin, vaultsH.Delete)
			protected.POST("/vaults/:id/upload", auth.RequireAdmin, vaultsH.UploadFile)
			protected.DELETE("/vaults/:id/file", auth.RequireAdmin, vaultsH.DeleteFile)

			// Runs
			protected.GET("/runs", runsH.List)
			protected.GET("/runs/:id", runsH.Get)
			protected.POST("/runs", runsH.Create)
			protected.POST("/runs/:id/cancel", runsH.Cancel)

			// Audit log (admin only)
			protected.GET("/audit", auth.RequireAdmin, auditH.List)
		}
	}

	// Webhook trigger — no auth, self-authenticating via token in URL.
	api.POST("/webhook/forms/:token", runsH.TriggerWebhook)

	// Form images — served without auth so browser <img> tags work.
	r.GET("/api/forms/:id/image", formsH.GetImage)

	// Run SSE stream — auth via ?token= query param since EventSource can't send headers.
	api.GET("/runs/:id/stream", runsH.Stream)

	// Serve static SvelteKit SPA.
	// Try to serve the real file from frontend/dist first; fall back to index.html
	// for SPA client-side routes. This correctly handles /_app/, /favicon.*, etc.
	r.NoRoute(func(c *gin.Context) {
		urlPath := c.Request.URL.Path

		if strings.HasPrefix(urlPath, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		staticPath := filepath.Join("./frontend/dist", filepath.Clean("/"+urlPath))
		info, err := os.Stat(staticPath)
		if err == nil && !info.IsDir() {
			c.File(staticPath)
			return
		}

		c.File("./frontend/dist/index.html")
	})

	return r
}
