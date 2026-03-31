package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jiin/weeky/internal/model"
)

type JiraService struct {
	client *http.Client
}

func NewJiraService() *JiraService {
	return &JiraService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type jiraSearchRequest struct {
	JQL        string   `json:"jql"`
	Fields     []string `json:"fields"`
	MaxResults int      `json:"maxResults"`
}

type jiraSearchResponse struct {
	Issues []jiraIssue `json:"issues"`
	Total  int         `json:"total"`
}

type jiraIssue struct {
	Key    string `json:"key"`
	Self   string `json:"self"`
	Fields struct {
		Summary   string `json:"summary"`
		Status    struct {
			Name string `json:"name"`
		} `json:"status"`
		Updated     string `json:"updated"`
		ResolutionDate *string `json:"resolutiondate"`
	} `json:"fields"`
}

func (s *JiraService) Sync(req model.JiraSyncRequest) (*model.SyncResult, error) {
	if err := ValidateExternalURL(req.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid Jira URL: %w", err)
	}

	result := &model.SyncResult{
		Source:   "jira",
		Items:    []model.SyncItem{},
		SyncedAt: time.Now(),
	}

	auth := base64.StdEncoding.EncodeToString([]byte(req.Email + ":" + req.Token))
	searchURL := fmt.Sprintf("%s/rest/api/3/search/jql", req.BaseURL)

	jql := fmt.Sprintf(
		`assignee = currentUser() AND updated >= "%s" AND updated <= "%s" ORDER BY updated DESC`,
		req.StartDate, req.EndDate,
	)
	issues, err := s.fetchIssues(searchURL, auth, jql)
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		date := issue.Fields.Updated[:10]
		if issue.Fields.ResolutionDate != nil && len(*issue.Fields.ResolutionDate) >= 10 {
			date = (*issue.Fields.ResolutionDate)[:10]
		}
		result.Items = append(result.Items, model.SyncItem{
			Title:   issue.Fields.Summary,
			Date:    date,
			URL:     fmt.Sprintf("%s/browse/%s", req.BaseURL, issue.Key),
			Type:    "issue",
			Content: issue.Fields.Status.Name,
		})
	}

	return result, nil
}

func (s *JiraService) fetchIssues(searchURL, auth, jql string) ([]jiraIssue, error) {
	body := jiraSearchRequest{
		JQL:        jql,
		Fields:     []string{"summary", "status", "updated", "resolutiondate"},
		MaxResults: 50,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", searchURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jira API returned status %d", resp.StatusCode)
	}

	var searchResp jiraSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return searchResp.Issues, nil
}
