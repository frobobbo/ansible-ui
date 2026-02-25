package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/brettjrea/ansible-frontend/internal/auth"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

func NewRouter(db *store.DB, jwtSvc *auth.JWTService, uploadDir string, jwtSecret string) *gin.Engine {
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
	vaultStore := db.Vaults(jwtSecret)
	authH := newAuthHandler(db.Users(), jwtSvc)
	usersH := newUsersHandler(db.Users())
	serversH := newServersHandler(db.Servers())
	playbooksH := newPlaybooksHandler(db.Playbooks(), uploadDir)
	formsH := newFormsHandler(db.Forms())
	vaultsH := newVaultsHandler(vaultStore)
	runsH := newRunsHandler(db.Runs(), db.Forms(), db.Servers(), db.Playbooks(), vaultStore)

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

			// Playbooks
			protected.GET("/playbooks", playbooksH.List)
			protected.GET("/playbooks/:id", playbooksH.Get)
			protected.POST("/playbooks", auth.RequireAdmin, playbooksH.Upload)
			protected.DELETE("/playbooks/:id", auth.RequireAdmin, playbooksH.Delete)

			// Forms
			protected.GET("/forms", formsH.List)
			protected.GET("/forms/:id", formsH.Get)
			protected.GET("/forms/:id/fields", formsH.GetFields)
			protected.POST("/forms", formsH.Create)
			protected.PUT("/forms/:id", formsH.Update)
			protected.DELETE("/forms/:id", formsH.Delete)

			// Vaults (admin-only writes)
			protected.GET("/vaults", vaultsH.List)
			protected.GET("/vaults/:id", vaultsH.Get)
			protected.POST("/vaults", auth.RequireAdmin, vaultsH.Create)
			protected.PUT("/vaults/:id", auth.RequireAdmin, vaultsH.Update)
			protected.DELETE("/vaults/:id", auth.RequireAdmin, vaultsH.Delete)

			// Runs
			protected.GET("/runs", runsH.List)
			protected.GET("/runs/:id", runsH.Get)
			protected.POST("/runs", runsH.Create)
		}
	}

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
