package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/auth"
	"github.com/jiin/weeky/internal/crypto"
	"github.com/jiin/weeky/internal/model"
	"github.com/jiin/weeky/internal/repository"
)

func TestMain(m *testing.M) {
	os.Setenv("ENCRYPTION_KEY", "test-encryption-key-for-testing!!")
	crypto.InitDefault()
	auth.SetSecret("test-jwt-secret-for-testing!!")
	os.Exit(m.Run())
}

// createTestUser creates a user and returns a valid JWT token
func createTestUser(t *testing.T, repo *repository.MockRepository) (int64, string) {
	t.Helper()
	hash, _ := auth.HashPassword("testpass123")
	user, err := repo.CreateUser("test@test.com", hash, "Test User", false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	token, err := auth.GenerateToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	return user.ID, token
}

func setupTestHandler(t *testing.T) (*Handler, *fiber.App, *repository.MockRepository, string, func()) {
	t.Helper()

	repo := repository.NewMock()
	h := New(repo)
	app := fiber.New()

	_, token := createTestUser(t, repo)

	// Middleware to inject userID from token for tests
	app.Use(func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			tokenStr := authHeader[len("Bearer "):]
			claims, err := auth.ValidateToken(tokenStr)
			if err == nil {
				c.Locals("userID", claims.UserID)
				c.Locals("email", claims.Email)
				c.Locals("isAdmin", claims.IsAdmin)
			}
		}
		return c.Next()
	})

	// Setup routes
	api := app.Group("/api")
	api.Get("/templates", h.GetTemplates)
	api.Post("/templates", h.CreateTemplate)
	api.Delete("/templates/:id", h.DeleteTemplate)
	api.Get("/reports/:id", h.GetReport)
	api.Post("/reports", h.CreateReport)
	api.Get("/config", h.GetConfig)
	api.Put("/config", h.UpdateConfig)

	cleanup := func() {
		repo.Close()
	}

	return h, app, repo, token, cleanup
}

func TestTemplateHandlers(t *testing.T) {
	_, app, _, token, cleanup := setupTestHandler(t)
	defer cleanup()

	// Test GetTemplates (empty)
	t.Run("GetTemplates_Empty", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/templates", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var templates []model.Template
		json.Unmarshal(body, &templates)
		if len(templates) != 0 {
			t.Errorf("Expected 0 templates, got %d", len(templates))
		}
	})

	// Test CreateTemplate
	t.Run("CreateTemplate", func(t *testing.T) {
		payload := `{"name": "Test Template", "style": "{\"color\": \"red\"}"}`
		req := httptest.NewRequest("POST", "/api/templates", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 201 {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var template model.Template
		json.Unmarshal(body, &template)
		if template.Name != "Test Template" {
			t.Errorf("Expected name 'Test Template', got '%s'", template.Name)
		}
	})

	// Test CreateTemplate without name
	t.Run("CreateTemplate_NoName", func(t *testing.T) {
		payload := `{"style": "{}"}`
		req := httptest.NewRequest("POST", "/api/templates", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	// Test GetTemplates (with data)
	t.Run("GetTemplates_WithData", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/templates", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		body, _ := io.ReadAll(resp.Body)
		var templates []model.Template
		json.Unmarshal(body, &templates)
		if len(templates) == 0 {
			t.Error("Expected at least 1 template")
		}
	})
}

func TestReportHandlers(t *testing.T) {
	_, app, _, token, cleanup := setupTestHandler(t)
	defer cleanup()

	// Test CreateReport
	t.Run("CreateReport", func(t *testing.T) {
		payload := `{
			"team_name": "개발팀",
			"author_name": "홍길동",
			"report_date": "2024-01-15",
			"this_week": [{"title": "API 개발", "due_date": "2024-01-15", "progress": 100}],
			"next_week": [],
			"issues": "",
			"template_id": 0
		}`
		req := httptest.NewRequest("POST", "/api/reports", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 201 {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 201, got %d: %s", resp.StatusCode, body)
		}

		body, _ := io.ReadAll(resp.Body)
		var report model.Report
		json.Unmarshal(body, &report)
		if report.ID == 0 {
			t.Error("Expected non-zero report ID")
		}
	})

	// Test GetReport
	t.Run("GetReport", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/reports/1", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 && resp.StatusCode != 404 {
			t.Errorf("Expected status 200 or 404, got %d", resp.StatusCode)
		}
	})

	// Test GetReport not found
	t.Run("GetReport_NotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/reports/99999", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 404 {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})
}

func TestConfigHandlers(t *testing.T) {
	_, app, _, token, cleanup := setupTestHandler(t)
	defer cleanup()

	// Test GetConfig (empty)
	t.Run("GetConfig_Empty", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/config", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	// Test UpdateConfig
	t.Run("UpdateConfig", func(t *testing.T) {
		payload := `{"configs": {"github_token": "test_token", "jira_email": "test@test.com"}}`
		req := httptest.NewRequest("PUT", "/api/config", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, body)
		}
	})

	// Test GetConfig (with data - should be masked)
	t.Run("GetConfig_Masked", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/config", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		body, _ := io.ReadAll(resp.Body)
		var config map[string]interface{}
		json.Unmarshal(body, &config)

		if val, ok := config["github_token"]; ok {
			if val != "***configured***" {
				t.Errorf("Expected masked value, got '%v'", val)
			}
		}
	})
}
