package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jiin/weeky/internal/model"
)

type GitHubService struct {
	client *http.Client
}

func NewGitHubService() *GitHubService {
	return &GitHubService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type githubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	HTMLURL string `json:"html_url"`
}

type githubPR struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	HTMLURL   string    `json:"html_url"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	MergedAt  *string   `json:"merged_at"`
}

func (s *GitHubService) Sync(req model.GitHubSyncRequest) (*model.SyncResult, error) {
	result := &model.SyncResult{
		Source:   "github",
		Items:    []model.SyncItem{},
		SyncedAt: time.Now(),
	}

	commits, err := s.fetchCommits(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commits: %w", err)
	}
	result.Items = append(result.Items, commits...)

	prs, err := s.fetchPRs(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PRs: %w", err)
	}
	result.Items = append(result.Items, prs...)

	return result, nil
}

func (s *GitHubService) fetchCommits(req model.GitHubSyncRequest) ([]model.SyncItem, error) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/commits?since=%sT00:00:00Z&until=%sT23:59:59Z",
		req.Owner, req.Repo, req.StartDate, req.EndDate,
	)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+req.Token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var commits []githubCommit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, err
	}

	items := make([]model.SyncItem, 0, len(commits))
	for _, c := range commits {
		message := c.Commit.Message
		if idx := len(message); idx > 80 {
			message = message[:80] + "..."
		}

		items = append(items, model.SyncItem{
			Title: message,
			Date:  c.Commit.Author.Date[:10], // YYYY-MM-DD
			URL:   c.HTMLURL,
			Type:  "commit",
		})
	}

	return items, nil
}

func (s *GitHubService) fetchPRs(req model.GitHubSyncRequest) ([]model.SyncItem, error) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/pulls?state=all&sort=updated&direction=desc",
		req.Owner, req.Repo,
	)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+req.Token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var prs []githubPR
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, err
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)
	endDate = endDate.Add(24 * time.Hour) // Include end date

	items := make([]model.SyncItem, 0)
	for _, pr := range prs {
		if pr.CreatedAt.Before(startDate) || pr.CreatedAt.After(endDate) {
			continue
		}

		items = append(items, model.SyncItem{
			Title: fmt.Sprintf("#%d %s", pr.Number, pr.Title),
			Date:  pr.CreatedAt.Format("2006-01-02"),
			URL:   pr.HTMLURL,
			Type:  "pr",
		})
	}

	return items, nil
}
