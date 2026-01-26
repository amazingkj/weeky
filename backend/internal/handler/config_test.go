package handler

// Config handler tests are now primarily tested via the crypto package
// This file tests the handler integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/repository"
)

func TestConfigHandler_Integration(t *testing.T) {
	repo := repository.NewMock()
	h := New(repo)
	app := fiber.New()
	app.Get("/api/config", h.GetConfig)
	app.Put("/api/config", h.UpdateConfig)

	// Test empty config
	t.Run("GetConfig_Empty", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/config", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	// Test update config
	t.Run("UpdateConfig", func(t *testing.T) {
		payload := `{"configs": {"test_key": "test_value"}}`
		req := httptest.NewRequest("PUT", "/api/config", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, body)
		}
	})

	// Test get config shows masked value
	t.Run("GetConfig_Masked", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/config", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		body, _ := io.ReadAll(resp.Body)
		var config map[string]interface{}
		json.Unmarshal(body, &config)

		if val, ok := config["test_key"]; ok {
			if val != "***configured***" {
				t.Errorf("Expected masked value, got '%v'", val)
			}
		}
	})

	// Test GetConfigValue internal method
	t.Run("GetConfigValue", func(t *testing.T) {
		val, err := h.GetConfigValue("test_key")
		if err != nil {
			t.Errorf("GetConfigValue failed: %v", err)
		}
		if val != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", val)
		}
	})

	// Test GetConfigValue for non-existent key
	t.Run("GetConfigValue_NotFound", func(t *testing.T) {
		_, err := h.GetConfigValue("non_existent_key")
		if err == nil {
			t.Error("Expected error for non-existent key")
		}
	})
}
