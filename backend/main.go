package main

import (
	"log"
	"os"

	"github.com/brettjrea/ansible-frontend/internal/api"
	"github.com/brettjrea/ansible-frontend/internal/auth"
	"github.com/brettjrea/ansible-frontend/internal/scheduler"
	"github.com/brettjrea/ansible-frontend/internal/store"
)

func main() {
	// Ensure data directories exist
	for _, dir := range []string{"./data/playbooks", "./data/vaults", "./data/form-images"} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			log.Fatal("create data dir:", err)
		}
	}

	// Database
	dsn := "file:./data/ansible.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
	db, err := store.New(dsn)
	if err != nil {
		log.Fatal("init db:", err)
	}

	// Seed default admin if no users exist
	if err := db.EnsureDefaultAdmin(); err != nil {
		log.Fatal("ensure default admin:", err)
	}

	// JWT
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change-me-in-production-use-a-long-random-string"
		log.Println("WARNING: JWT_SECRET not set, using insecure default")
	}
	jwtSvc := auth.NewJWTService(jwtSecret)

	// Runs handler + scheduler (created before NewRouter to avoid circular deps)
	vaultStoreForRuns := db.Vaults(jwtSecret)
	runsH := api.NewRunsHandler(db.Runs(), db.Forms(), db.Servers(), db.ServerGroups(), db.Playbooks(), vaultStoreForRuns, db.Audit(), jwtSvc)

	sched := scheduler.New(runsH.TriggerScheduledRun)
	defer sched.Stop()

	if forms, err := db.Forms().ListScheduled(); err == nil {
		for _, f := range forms {
			sched.Upsert(f)
		}
	}

	// Router
	router := api.NewRouter(db, jwtSvc, "./data/playbooks", "./data/vaults", "./data/form-images", jwtSecret, runsH, sched)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Ansible Frontend starting on :%s", port)
	log.Printf("Default credentials: admin / %s", func() string {
		if p := os.Getenv("ADMIN_PASSWORD"); p != "" {
			return "(see ADMIN_PASSWORD env var)"
		}
		return "admin"
	}())

	if err := router.Run(":" + port); err != nil {
		log.Fatal("server:", err)
	}
}
