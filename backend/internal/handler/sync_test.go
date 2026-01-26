package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/repository"
)

func setupSyncTestHandler(t *testing.T) (*Handler, *fiber.App, func()) {
	t.Helper()

	repo := repository.NewMock()
	h := New(repo)
	app := fiber.New()

	api := app.Group("/api")
	api.Post("/sync/github", h.SyncGitHub)
	api.Post("/sync/gitlab", h.SyncGitLab)
	api.Post("/sync/jira", h.SyncJira)
	api.Post("/sync/hiworks", h.SyncHiworks)

	cleanup := func() {
		repo.Close()
	}

	return h, app, cleanup
}

func TestSyncGitHub(t *testing.T) {
	_, app, cleanup := setupSyncTestHandler(t)
	defer cleanup()

	t.Run("MissingOwnerAndRepo", func(t *testing.T) {
		payload := `{"start_date": "2024-01-01", "end_date": "2024-01-07"}`
		req := httptest.NewRequest("POST", "/api/sync/github", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]string
		json.Unmarshal(body, &result)
		if result["error"] != "Owner와 Repo를 입력해주세요." {
			t.Errorf("Unexpected error message: %s", result["error"])
		}
	})

	t.Run("MissingDates", func(t *testing.T) {
		payload := `{"owner": "test", "repo": "test-repo"}`
		req := httptest.NewRequest("POST", "/api/sync/github", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("MissingToken", func(t *testing.T) {
		payload := `{"owner": "test", "repo": "test-repo", "start_date": "2024-01-01", "end_date": "2024-01-07"}`
		req := httptest.NewRequest("POST", "/api/sync/github", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		// Should fail because no token configured
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		payload := `{invalid json}`
		req := httptest.NewRequest("POST", "/api/sync/github", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestSyncGitLab(t *testing.T) {
	_, app, cleanup := setupSyncTestHandler(t)
	defer cleanup()

	t.Run("MissingNamespaceAndProject", func(t *testing.T) {
		payload := `{"start_date": "2024-01-01", "end_date": "2024-01-07"}`
		req := httptest.NewRequest("POST", "/api/sync/gitlab", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]string
		json.Unmarshal(body, &result)
		if result["error"] != "Namespace와 Project를 입력해주세요." {
			t.Errorf("Unexpected error message: %s", result["error"])
		}
	})

	t.Run("MissingDates", func(t *testing.T) {
		payload := `{"namespace": "test-group", "project": "test-project"}`
		req := httptest.NewRequest("POST", "/api/sync/gitlab", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("MissingToken", func(t *testing.T) {
		payload := `{"namespace": "test-group", "project": "test-project", "start_date": "2024-01-01", "end_date": "2024-01-07"}`
		req := httptest.NewRequest("POST", "/api/sync/gitlab", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		payload := `{broken`
		req := httptest.NewRequest("POST", "/api/sync/gitlab", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestSyncJira(t *testing.T) {
	_, app, cleanup := setupSyncTestHandler(t)
	defer cleanup()

	t.Run("MissingBaseURL", func(t *testing.T) {
		payload := `{"start_date": "2024-01-01", "end_date": "2024-01-07"}`
		req := httptest.NewRequest("POST", "/api/sync/jira", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]string
		json.Unmarshal(body, &result)
		if result["error"] != "Jira Base URL을 입력해주세요." {
			t.Errorf("Unexpected error message: %s", result["error"])
		}
	})

	t.Run("MissingDates", func(t *testing.T) {
		payload := `{"base_url": "https://test.atlassian.net"}`
		req := httptest.NewRequest("POST", "/api/sync/jira", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("MissingCredentials", func(t *testing.T) {
		payload := `{"base_url": "https://test.atlassian.net", "start_date": "2024-01-01", "end_date": "2024-01-07"}`
		req := httptest.NewRequest("POST", "/api/sync/jira", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		// Should fail because no credentials configured
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		payload := `not json`
		req := httptest.NewRequest("POST", "/api/sync/jira", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestSyncHiworks(t *testing.T) {
	_, app, cleanup := setupSyncTestHandler(t)
	defer cleanup()

	t.Run("MissingDates", func(t *testing.T) {
		payload := `{"office_id": "test", "user_id": "user", "password": "pass"}`
		req := httptest.NewRequest("POST", "/api/sync/hiworks", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("MissingCredentials", func(t *testing.T) {
		payload := `{"start_date": "2024-01-01", "end_date": "2024-01-07"}`
		req := httptest.NewRequest("POST", "/api/sync/hiworks", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]string
		json.Unmarshal(body, &result)
		if result["error"] != "Hiworks 로그인 정보가 필요합니다 (회사ID, 사용자ID, 비밀번호)" {
			t.Errorf("Unexpected error message: %s", result["error"])
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		payload := `{broken`
		req := httptest.NewRequest("POST", "/api/sync/hiworks", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}
