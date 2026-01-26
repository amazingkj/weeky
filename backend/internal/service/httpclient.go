package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultTimeout is the default HTTP client timeout
const DefaultTimeout = 30 * time.Second

// HTTPClient wraps http.Client with common functionality
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient creates a new HTTP client with default timeout
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{Timeout: DefaultTimeout},
	}
}

// NewHTTPClientWithTimeout creates a new HTTP client with custom timeout
func NewHTTPClientWithTimeout(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{Timeout: timeout},
	}
}

// DoJSON makes an HTTP request and decodes JSON response
func (c *HTTPClient) DoJSON(req *http.Request, result interface{}) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("JSON decode failed: %w", err)
	}

	return nil
}

// SetBearerAuth sets Bearer token authorization header
func SetBearerAuth(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
}

// SetBasicAuth sets Basic authorization header
func SetBasicAuth(req *http.Request, user, password string) {
	req.SetBasicAuth(user, password)
}

// SetJSONHeaders sets common JSON request headers
func SetJSONHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// ValidateExternalURL validates that a URL is safe to make requests to.
// It rejects internal/private network addresses to prevent SSRF attacks.
func ValidateExternalURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL is empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Must be http or https
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got %q", parsed.Scheme)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL has no hostname")
	}

	// Block localhost and loopback
	lower := strings.ToLower(hostname)
	if lower == "localhost" || lower == "127.0.0.1" || lower == "::1" || lower == "0.0.0.0" {
		return fmt.Errorf("requests to localhost are not allowed")
	}

	// Block private/internal IP ranges
	ip := net.ParseIP(hostname)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("requests to private/internal IP addresses are not allowed")
		}
	}

	// Block common internal hostnames
	if strings.HasSuffix(lower, ".local") || strings.HasSuffix(lower, ".internal") {
		return fmt.Errorf("requests to internal hostnames are not allowed")
	}

	return nil
}
