package main

import (
	"crypto/subtle"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jiin/weeky/internal/config"
	"github.com/jiin/weeky/internal/crypto"
	"github.com/jiin/weeky/internal/handler"
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

	// Database path
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./weeky.db"
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
		AppName: "weeky",
	})

	// Middleware
	app.Use(logger.New())

	// CORS - read allowed origins from env, default to localhost dev server
	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000,http://localhost:3004,http://localhost:5173"
	}
	allowHeaders := "Origin, Content-Type, Accept, X-API-Key"
	app.Use(cors.New(cors.Config{
		AllowOrigins: corsOrigins,
		AllowHeaders: allowHeaders,
	}))

	// Rate limiting - 60 requests per minute per IP
	app.Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
	}))

	// API Key authentication middleware (only active when API_KEY env is set)
	apiKey := os.Getenv("API_KEY")
	if apiKey != "" {
		app.Use(apiKeyAuth(apiKey))
	}

	// API routes
	api := app.Group("/api/v1")

	// Template routes
	api.Get("/templates", h.GetTemplates)
	api.Post("/templates", h.CreateTemplate)
	api.Put("/templates/:id", h.UpdateTemplate)
	api.Delete("/templates/:id", h.DeleteTemplate)

	// Report routes
	api.Get("/reports/:id", h.GetReport)
	api.Post("/reports", h.CreateReport)

	// Config routes
	api.Get("/config", h.GetConfig)
	api.Put("/config", h.UpdateConfig)

	// Sync routes
	api.Post("/sync/github", h.SyncGitHub)
	api.Post("/sync/gitlab", h.SyncGitLab)
	api.Post("/sync/jira", h.SyncJira)
	api.Post("/sync/hiworks", h.SyncHiworks)

	// AI routes
	api.Post("/ai/generate", h.GenerateAIReport)

	// Backward-compatible /api routes (redirect to /api/v1)
	app.Use("/api", func(c *fiber.Ctx) error {
		path := c.Path()
		if !strings.HasPrefix(path, "/api/v1") {
			newPath := strings.Replace(path, "/api", "/api/v1", 1)
			return c.Redirect(newPath, fiber.StatusTemporaryRedirect)
		}
		return c.Next()
	})

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Static files (for production)
	app.Static("/", "./dist")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "28080"
	}

	slog.Info("Server starting", "port", port)
	log.Fatal(app.Listen(":" + port))
}

// apiKeyAuth returns a middleware that validates the X-API-Key header
func apiKeyAuth(expectedKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip auth for health check
		if c.Path() == "/health" {
			return c.Next()
		}
		// Skip auth for static files
		if !strings.HasPrefix(c.Path(), "/api") {
			return c.Next()
		}

		key := c.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(key), []byte(expectedKey)) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or missing API key",
			})
		}
		return c.Next()
	}
}
