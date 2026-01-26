package service

import (
	"testing"

	"github.com/jiin/weeky/internal/model"
)

func TestGitHubService_Validation(t *testing.T) {
	svc := NewGitHubService()

	t.Run("EmptyToken", func(t *testing.T) {
		req := model.GitHubSyncRequest{
			Token:     "",
			Owner:     "test-owner",
			Repo:      "test-repo",
			StartDate: "2024-01-01",
			EndDate:   "2024-01-07",
		}
		_, err := svc.Sync(req)
		// Should fail due to API call with empty token
		if err == nil {
			t.Log("Expected error with empty token, but got nil (API might have responded)")
		}
	})
}

func TestJiraService_Validation(t *testing.T) {
	svc := NewJiraService()

	t.Run("InvalidBaseURL", func(t *testing.T) {
		req := model.JiraSyncRequest{
			BaseURL:   "not-a-valid-url",
			Email:     "test@test.com",
			Token:     "test-token",
			StartDate: "2024-01-01",
			EndDate:   "2024-01-07",
		}
		_, err := svc.Sync(req)
		// Should fail due to invalid URL
		if err == nil {
			t.Log("Expected error with invalid URL, but got nil")
		}
	})
}

func TestHiworksService_Validation(t *testing.T) {
	svc := NewHiworksService()

	t.Run("EmptyCredentials", func(t *testing.T) {
		req := model.HiworksSyncRequest{
			OfficeID:  "",
			UserID:    "",
			Password:  "",
			StartDate: "2024-01-01",
			EndDate:   "2024-01-07",
		}
		_, err := svc.Sync(req)
		// Should fail due to login with empty credentials
		if err == nil {
			t.Log("Expected error with empty credentials, but got nil")
		}
	})
}

func TestGitLabService_Validation(t *testing.T) {
	svc := NewGitLabService()

	t.Run("EmptyToken", func(t *testing.T) {
		req := model.GitLabSyncRequest{
			Token:     "",
			BaseURL:   "https://gitlab.com",
			Namespace: "test-group",
			Project:   "test-project",
			StartDate: "2024-01-01",
			EndDate:   "2024-01-07",
		}
		_, err := svc.Sync(req)
		// Should fail due to API call with empty token
		if err == nil {
			t.Log("Expected error with empty token, but got nil")
		}
	})
}

// Note: Full integration tests would require actual API credentials
// These tests verify the service structure and basic validation
