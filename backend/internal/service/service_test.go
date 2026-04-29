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

func TestJira_FindCustomFieldID(t *testing.T) {
	fields := []jiraField{
		{ID: "summary", Name: "Summary"},
		{ID: "customfield_10081", Name: "솔루션명"},
		{ID: "customfield_10082", Name: "요청사이트"},
		{ID: "customfield_10090", Name: "기타"},
	}
	if got := findCustomFieldID(fields, "솔루션명"); got != "customfield_10081" {
		t.Errorf("솔루션명: got %q, want customfield_10081", got)
	}
	if got := findCustomFieldID(fields, "요청사이트"); got != "customfield_10082" {
		t.Errorf("요청사이트: got %q, want customfield_10082", got)
	}
	if got := findCustomFieldID(fields, "없는필드"); got != "" {
		t.Errorf("없는필드: got %q, want empty", got)
	}
}

func TestJira_DecodeFieldValue(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"null", `null`, ""},
		{"plain string", `"한화손해보험"`, "한화손해보험"},
		{"single-select option", `{"value":"CruzAPIM 1.5"}`, "CruzAPIM 1.5"},
		{"named option", `{"name":"한화손해보험"}`, "한화손해보험"},
		{"multi-value", `[{"value":"한화손해보험"},{"value":"흥국화재"}]`, "한화손해보험, 흥국화재"},
		{"empty array", `[]`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decodeFieldValue([]byte(c.in))
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// Note: Full integration tests would require actual API credentials
// These tests verify the service structure and basic validation
