package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Error("NewHTTPClient returned nil")
	}
	if client.client == nil {
		t.Error("HTTPClient.client is nil")
	}
	if client.client.Timeout != DefaultTimeout {
		t.Errorf("Expected timeout %v, got %v", DefaultTimeout, client.client.Timeout)
	}
}

func TestNewHTTPClientWithTimeout(t *testing.T) {
	timeout := 10 * time.Second
	client := NewHTTPClientWithTimeout(timeout)
	if client == nil {
		t.Error("NewHTTPClientWithTimeout returned nil")
	}
	if client.client.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.client.Timeout)
	}
}

func TestHTTPClient_DoJSON(t *testing.T) {
	t.Run("SuccessfulRequest", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"message": "success"})
		}))
		defer server.Close()

		client := NewHTTPClient()
		req, _ := http.NewRequest("GET", server.URL, nil)

		var result map[string]string
		err := client.DoJSON(req, &result)
		if err != nil {
			t.Fatalf("DoJSON failed: %v", err)
		}
		if result["message"] != "success" {
			t.Errorf("Expected message 'success', got '%s'", result["message"])
		}
	})

	t.Run("ServerError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		client := NewHTTPClient()
		req, _ := http.NewRequest("GET", server.URL, nil)

		var result map[string]string
		err := client.DoJSON(req, &result)
		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}))
		defer server.Close()

		client := NewHTTPClient()
		req, _ := http.NewRequest("GET", server.URL, nil)

		var result map[string]string
		err := client.DoJSON(req, &result)
		if err == nil {
			t.Error("Expected error for 404 response")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("not valid json"))
		}))
		defer server.Close()

		client := NewHTTPClient()
		req, _ := http.NewRequest("GET", server.URL, nil)

		var result map[string]string
		err := client.DoJSON(req, &result)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("ConnectionError", func(t *testing.T) {
		client := NewHTTPClientWithTimeout(1 * time.Millisecond)
		req, _ := http.NewRequest("GET", "http://localhost:99999", nil)

		var result map[string]string
		err := client.DoJSON(req, &result)
		if err == nil {
			t.Error("Expected error for connection failure")
		}
	})
}

func TestSetBearerAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	SetBearerAuth(req, "test-token")

	auth := req.Header.Get("Authorization")
	expected := "Bearer test-token"
	if auth != expected {
		t.Errorf("Expected Authorization header '%s', got '%s'", expected, auth)
	}
}

func TestSetBasicAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	SetBasicAuth(req, "user", "pass")

	// Basic auth header should be set
	_, _, ok := req.BasicAuth()
	if !ok {
		t.Error("Basic auth not set")
	}
}

func TestSetJSONHeaders(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	SetJSONHeaders(req)

	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	accept := req.Header.Get("Accept")
	if accept != "application/json" {
		t.Errorf("Expected Accept 'application/json', got '%s'", accept)
	}
}

func TestValidateExternalURL(t *testing.T) {
	t.Run("ValidURLs", func(t *testing.T) {
		validURLs := []string{
			"https://gitlab.com",
			"https://gitlab.example.com",
			"https://jira.atlassian.net",
			"https://my-gitlab.company.com",
			"http://external-server.com:8080",
		}
		for _, u := range validURLs {
			if err := ValidateExternalURL(u); err != nil {
				t.Errorf("Expected %q to be valid, got error: %v", u, err)
			}
		}
	})

	t.Run("BlockedURLs", func(t *testing.T) {
		blockedURLs := []string{
			"",
			"ftp://files.example.com",
			"http://localhost",
			"http://localhost:8080",
			"http://127.0.0.1",
			"http://127.0.0.1:3000",
			"http://0.0.0.0",
			"http://[::1]",
			"http://192.168.1.1",
			"http://10.0.0.1",
			"http://172.16.0.1",
			"http://server.local",
			"http://server.internal",
		}
		for _, u := range blockedURLs {
			if err := ValidateExternalURL(u); err == nil {
				t.Errorf("Expected %q to be blocked, but it was allowed", u)
			}
		}
	})
}
