package main

import (
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jiin/weeky/internal/auth"
	"github.com/jiin/weeky/internal/config"
	"github.com/jiin/weeky/internal/crypto"
	"github.com/jiin/weeky/internal/handler"
	"github.com/jiin/weeky/internal/middleware"
	"github.com/jiin/weeky/internal/repository"
)

func main() {
	// Load .env file (ignore error if file doesn't exist)
	if err := config.LoadEnv(".env"); err != nil {
		slog.Info("No .env file found, using environment variables")
	}

	// Validate encryption key
	if err := crypto.InitDefault(); err != nil {
		log.Fatalf("ENCRYPTION_KEY is required: %v", err)
	}

	// Initialize JWT secret from env (if set)
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		auth.SetSecret(secret)
	}

	// Database path
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./jugan.db"
	}

	// Initialize repository
	repo, err := repository.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer repo.Close()

	// Initialize handler
	h := handler.New(repo)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName: "jugan",
	})

	// Middleware
	app.Use(logger.New())

	// CORS - read allowed origins from env, default to localhost dev server
	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000,http://localhost:3004,http://localhost:5173"
	}
	allowHeaders := "Origin, Content-Type, Accept, Authorization"
	app.Use(cors.New(cors.Config{
		AllowOrigins: corsOrigins,
		AllowHeaders: allowHeaders,
	}))

	// Rate limiting - 300 requests per minute per IP
	app.Use(limiter.New(limiter.Config{
		Max:        300,
		Expiration: 1 * time.Minute,
	}))

	// Health check (public)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// API routes
	api := app.Group("/api/v1")

	// Auth rate limiter - stricter for login/register (10 per minute)
	authLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
	})

	// Public auth routes (no authentication required)
	authRoutes := api.Group("/auth")
	authRoutes.Get("/setup", h.CheckSetup)
	authRoutes.Post("/register", authLimiter, h.Register)
	authRoutes.Post("/login", authLimiter, h.Login)
	authRoutes.Post("/refresh", h.RefreshToken)

	// Protected routes (authentication required)
	protected := api.Group("", middleware.RequireAuth())

	// Auth - current user
	protected.Get("/auth/me", h.GetMe)

	// Admin routes
	admin := protected.Group("/admin", middleware.RequireAdmin())
	admin.Post("/invite-codes", h.CreateInviteCode)
	admin.Get("/invite-codes", h.GetInviteCodes)

	// Template routes
	protected.Get("/templates", h.GetTemplates)
	protected.Post("/templates", h.CreateTemplate)
	protected.Put("/templates/:id", h.UpdateTemplate)
	protected.Delete("/templates/:id", h.DeleteTemplate)

	// User list (for team member selection)
	protected.Get("/users", h.GetUsers)

	// Report routes
	protected.Get("/reports", h.GetReports)
	protected.Get("/reports/:id", h.GetReport)
	protected.Post("/reports", h.CreateReport)
	protected.Put("/reports/:id", h.UpdateReport)
	protected.Post("/reports/save", h.SaveReport)

	// Config routes
	protected.Get("/config", h.GetConfig)
	protected.Put("/config", h.UpdateConfig)

	// Sync routes
	protected.Post("/sync/github", h.SyncGitHub)
	protected.Post("/sync/gitlab", h.SyncGitLab)
	protected.Post("/sync/jira", h.SyncJira)
	protected.Post("/sync/hiworks", h.SyncHiworks)
	protected.Post("/sync/hiworks/test", h.TestHiworks)

	// GitLab project discovery
	protected.Get("/gitlab/projects", h.ListGitLabProjects)

	// AI routes
	protected.Post("/ai/generate", h.GenerateAIReport)

	// Team routes
	protected.Post("/teams", h.CreateTeam)
	protected.Get("/teams", h.GetMyTeams)
	protected.Get("/teams/:id", h.GetTeam)
	protected.Put("/teams/:id", h.UpdateTeam)
	protected.Delete("/teams/:id", h.DeleteTeam)

	// Team member routes
	protected.Post("/teams/:id/members", h.AddTeamMember)
	protected.Get("/teams/:id/members", h.GetTeamMembers)
	protected.Put("/teams/:id/members/:memberId", h.UpdateTeamMember)
	protected.Delete("/teams/:id/members/:memberId", h.RemoveTeamMember)

	// Team submission routes
	protected.Post("/teams/:id/submit", h.SubmitReport)
	protected.Delete("/teams/:id/submit/:reportId", h.UnsubmitReport)
	protected.Get("/teams/:id/submissions", h.GetTeamSubmissions)
	protected.Get("/teams/:id/my-submission", h.GetMySubmission)
	protected.Get("/teams/:id/my-submissions", h.GetMySubmissions)

	// Team report routes (leader/group_leader)
	protected.Get("/teams/:id/reports/:reportId", h.GetTeamMemberReport)
	protected.Put("/teams/:id/reports/:reportId", h.UpdateTeamMemberReport)
	protected.Get("/teams/:id/consolidated", h.GetConsolidatedReport)
	protected.Post("/teams/:id/ai/summarize", h.SummarizeConsolidatedReport)

	// Team project routes
	protected.Get("/teams/:id/projects", h.GetTeamProjects)
	protected.Post("/teams/:id/projects", h.CreateTeamProject)
	protected.Post("/teams/:id/projects/auto", h.AutoCreateTeamProject)
	protected.Put("/teams/:id/projects/reorder", h.ReorderTeamProjects)
	protected.Put("/teams/:id/projects/:pid", h.UpdateTeamProject)
	protected.Delete("/teams/:id/projects/:pid", h.DeleteTeamProject)

	// Consolidated edit routes
	protected.Put("/teams/:id/consolidated-edit", h.SaveConsolidatedEdit)
	protected.Get("/teams/:id/consolidated-edit", h.GetConsolidatedEdit)
	protected.Delete("/teams/:id/consolidated-edit", h.DeleteConsolidatedEdit)

	// Team history route
	protected.Get("/teams/:id/history", h.GetTeamHistory)

	// Backward-compatible /api routes (redirect to /api/v1)
	app.Use("/api", func(c *fiber.Ctx) error {
		path := c.Path()
		if !strings.HasPrefix(path, "/api/v1") {
			newPath := strings.Replace(path, "/api", "/api/v1", 1)
			return c.Redirect(newPath, fiber.StatusTemporaryRedirect)
		}
		return c.Next()
	})

	// Static files (for production)
	app.Static("/", "./dist")

	// SPA catch-all: serve index.html for all non-API/non-static routes
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./dist/index.html")
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "28080"
	}

	slog.Info("Server starting", "port", port)
	log.Fatal(app.Listen(":" + port))
}
