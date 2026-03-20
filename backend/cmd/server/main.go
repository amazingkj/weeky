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
	if err := config.LoadEnv(".env"); err != nil {
		slog.Info("No .env file found, using environment variables")
	}

	if err := crypto.InitDefault(); err != nil {
		log.Fatalf("ENCRYPTION_KEY is required: %v", err)
	}

	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		auth.SetSecret(secret)
	}

	repo, err := repository.NewFromEnv()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer repo.Close()

	h := handler.New(repo)

	app := fiber.New(fiber.Config{
		AppName: "jugan",
	})

	app.Use(logger.New())

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000,http://localhost:3004,http://localhost:5173"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: corsOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Use(limiter.New(limiter.Config{
		Max:        300,
		Expiration: 1 * time.Minute,
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := app.Group("/api/v1")

	authLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
	})

	authRoutes := api.Group("/auth")
	authRoutes.Get("/setup", h.CheckSetup)
	authRoutes.Post("/register", authLimiter, h.Register)
	authRoutes.Post("/login", authLimiter, h.Login)
	authRoutes.Post("/refresh", h.RefreshToken)

	protected := api.Group("", middleware.RequireAuth())

	protected.Get("/auth/me", h.GetMe)

	admin := protected.Group("/admin", middleware.RequireAdmin())
	admin.Post("/invite-codes", h.CreateInviteCode)
	admin.Get("/invite-codes", h.GetInviteCodes)
	admin.Post("/users/:id/reset-password", h.AdminResetPassword)

	protected.Get("/templates", h.GetTemplates)
	protected.Post("/templates", h.CreateTemplate)
	protected.Put("/templates/:id", h.UpdateTemplate)
	protected.Delete("/templates/:id", h.DeleteTemplate)

	protected.Get("/users", h.GetUsers)

	protected.Get("/reports", h.GetReports)
	protected.Get("/reports/:id", h.GetReport)
	protected.Post("/reports", h.CreateReport)
	protected.Put("/reports/:id", h.UpdateReport)
	protected.Post("/reports/save", h.SaveReport)

	protected.Get("/config", h.GetConfig)
	protected.Put("/config", h.UpdateConfig)

	protected.Post("/sync/github", h.SyncGitHub)
	protected.Post("/sync/gitlab", h.SyncGitLab)
	protected.Post("/sync/jira", h.SyncJira)
	protected.Post("/sync/hiworks", h.SyncHiworks)
	protected.Post("/sync/hiworks/test", h.TestHiworks)

	protected.Get("/gitlab/projects", h.ListGitLabProjects)

	protected.Post("/ai/generate", h.GenerateAIReport)

	protected.Post("/teams", h.CreateTeam)
	protected.Get("/teams", h.GetMyTeams)
	protected.Get("/teams/:id", h.GetTeam)
	protected.Put("/teams/:id", h.UpdateTeam)
	protected.Delete("/teams/:id", h.DeleteTeam)

	protected.Post("/teams/:id/members", h.AddTeamMember)
	protected.Get("/teams/:id/members", h.GetTeamMembers)
	protected.Put("/teams/:id/members/:memberId", h.UpdateTeamMember)
	protected.Delete("/teams/:id/members/:memberId", h.RemoveTeamMember)

	protected.Post("/teams/:id/submit", h.SubmitReport)
	protected.Delete("/teams/:id/submit/:reportId", h.UnsubmitReport)
	protected.Get("/teams/:id/submissions", h.GetTeamSubmissions)
	protected.Get("/teams/:id/my-submission", h.GetMySubmission)
	protected.Get("/teams/:id/my-submissions", h.GetMySubmissions)

	protected.Get("/teams/:id/reports/:reportId", h.GetTeamMemberReport)
	protected.Put("/teams/:id/reports/:reportId", h.UpdateTeamMemberReport)
	protected.Get("/teams/:id/consolidated", h.GetConsolidatedReport)
	protected.Post("/teams/:id/ai/summarize", h.SummarizeConsolidatedReport)

	protected.Get("/teams/:id/projects", h.GetTeamProjects)
	protected.Post("/teams/:id/projects", h.CreateTeamProject)
	protected.Post("/teams/:id/projects/auto", h.AutoCreateTeamProject)
	protected.Put("/teams/:id/projects/reorder", h.ReorderTeamProjects)
	protected.Put("/teams/:id/projects/:pid", h.UpdateTeamProject)
	protected.Delete("/teams/:id/projects/:pid", h.DeleteTeamProject)

	protected.Put("/teams/:id/consolidated-edit", h.SaveConsolidatedEdit)
	protected.Get("/teams/:id/consolidated-edit", h.GetConsolidatedEdit)
	protected.Delete("/teams/:id/consolidated-edit", h.DeleteConsolidatedEdit)

	protected.Get("/teams/:id/history", h.GetTeamHistory)

	app.Use("/api", func(c *fiber.Ctx) error {
		path := c.Path()
		if !strings.HasPrefix(path, "/api/v1") {
			newPath := strings.Replace(path, "/api", "/api/v1", 1)
			return c.Redirect(newPath, fiber.StatusTemporaryRedirect)
		}
		return c.Next()
	})

	app.Static("/", "./dist")

	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./dist/index.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "28080"
	}

	slog.Info("Server starting", "port", port)
	log.Fatal(app.Listen(":" + port))
}
