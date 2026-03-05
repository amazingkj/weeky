package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jiin/weeky/internal/model"
)

type GitLabService struct {
	client *http.Client
}

func NewGitLabService() *GitLabService {
	return &GitLabService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type gitlabCommit struct {
	ID             string `json:"id"`
	ShortID        string `json:"short_id"`
	Title          string `json:"title"`
	Message        string `json:"message"`
	CommittedDate  string `json:"committed_date"`
	WebURL         string `json:"web_url"`
}

type gitlabProject struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
	Namespace         struct {
		FullPath string `json:"full_path"`
	} `json:"namespace"`
}

type gitlabMR struct {
	IID       int       `json:"iid"`
	Title     string    `json:"title"`
	WebURL    string    `json:"web_url"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	MergedAt  *string   `json:"merged_at"`
}

func (s *GitLabService) ListProjects(baseURL, token string) ([]model.GitLabProject, error) {
	if err := ValidateExternalURL(baseURL); err != nil {
		return nil, fmt.Errorf("invalid GitLab URL: %w", err)
	}

	var allProjects []model.GitLabProject
	page := 1
	perPage := 100

	for {
		apiURL := fmt.Sprintf("%s/api/v4/projects?membership=true&simple=true&per_page=%d&page=%d&order_by=last_activity_at",
			baseURL, perPage, page)

		httpReq, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("PRIVATE-TOKEN", token)

		resp, err := s.client.Do(httpReq)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
		}

		var projects []gitlabProject
		if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
			return nil, err
		}

		for _, p := range projects {
			namespace := p.Namespace.FullPath
			projectName := p.Name
			parts := splitLast(p.PathWithNamespace, "/")
			if len(parts) == 2 {
				namespace = parts[0]
				projectName = parts[1]
			}

			allProjects = append(allProjects, model.GitLabProject{
				ID:        p.ID,
				Name:      p.Name,
				FullPath:  p.PathWithNamespace,
				Namespace: namespace,
				Project:   projectName,
				WebURL:    p.WebURL,
			})
		}

		if len(projects) < perPage {
			break
		}
		page++
	}

	return allProjects, nil
}

func splitLast(s, sep string) []string {
	idx := -1
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == sep {
			idx = i
			break
		}
	}
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}

func (s *GitLabService) Sync(req model.GitLabSyncRequest) (*model.SyncResult, error) {
	if err := ValidateExternalURL(req.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid GitLab URL: %w", err)
	}

	result := &model.SyncResult{
		Source:   "gitlab",
		Items:    []model.SyncItem{},
		SyncedAt: time.Now(),
	}

	commits, err := s.fetchCommits(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commits: %w", err)
	}
	result.Items = append(result.Items, commits...)

	mrs, err := s.fetchMRs(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MRs: %w", err)
	}
	result.Items = append(result.Items, mrs...)

	return result, nil
}

func (s *GitLabService) fetchCommits(req model.GitLabSyncRequest) ([]model.SyncItem, error) {
	projectPath := url.PathEscape(fmt.Sprintf("%s/%s", req.Namespace, req.Project))

	apiURL := fmt.Sprintf(
		"%s/api/v4/projects/%s/repository/commits?since=%sT00:00:00Z&until=%sT23:59:59Z",
		req.BaseURL, projectPath, req.StartDate, req.EndDate,
	)

	httpReq, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("PRIVATE-TOKEN", req.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
	}

	var commits []gitlabCommit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, err
	}

	projectLabel := fmt.Sprintf("%s/%s", req.Namespace, req.Project)

	items := make([]model.SyncItem, 0, len(commits))
	for _, c := range commits {
		title := c.Title
		if len(title) > 80 {
			title = title[:80] + "..."
		}

		// Parse date
		date := c.CommittedDate
		if len(date) >= 10 {
			date = date[:10]
		}

		items = append(items, model.SyncItem{
			Title:  title,
			Date:   date,
			URL:    c.WebURL,
			Type:   "commit",
			Source: projectLabel,
		})
	}

	return items, nil
}

func (s *GitLabService) fetchMRs(req model.GitLabSyncRequest) ([]model.SyncItem, error) {
	projectPath := url.PathEscape(fmt.Sprintf("%s/%s", req.Namespace, req.Project))

	apiURL := fmt.Sprintf(
		"%s/api/v4/projects/%s/merge_requests?state=all&order_by=updated_at&sort=desc&created_after=%sT00:00:00Z&created_before=%sT23:59:59Z",
		req.BaseURL, projectPath, req.StartDate, req.EndDate,
	)

	httpReq, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("PRIVATE-TOKEN", req.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
	}

	var mrs []gitlabMR
	if err := json.NewDecoder(resp.Body).Decode(&mrs); err != nil {
		return nil, err
	}

	projectLabel := fmt.Sprintf("%s/%s", req.Namespace, req.Project)

	items := make([]model.SyncItem, 0)
	for _, mr := range mrs {
		items = append(items, model.SyncItem{
			Title:  fmt.Sprintf("!%d %s", mr.IID, mr.Title),
			Date:   mr.CreatedAt.Format("2006-01-02"),
			URL:    mr.WebURL,
			Type:   "mr",
			Source: projectLabel,
		})
	}

	return items, nil
}
