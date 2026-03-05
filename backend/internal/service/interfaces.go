package service

import "github.com/jiin/weeky/internal/model"

type GitHubSyncer interface {
	Sync(req model.GitHubSyncRequest) (*model.SyncResult, error)
}

type GitLabSyncer interface {
	Sync(req model.GitLabSyncRequest) (*model.SyncResult, error)
	ListProjects(baseURL, token string) ([]model.GitLabProject, error)
}

type JiraSyncer interface {
	Sync(req model.JiraSyncRequest) (*model.SyncResult, error)
}

type HiworksSyncer interface {
	Sync(req model.HiworksSyncRequest) (*model.SyncResult, error)
	TestLogin(officeID, userID, password string) error
}

type AIReportGenerator interface {
	GenerateReport(req GenerateReportRequest) (*GenerateReportResponse, error)
}

type Services struct {
	GitHub         GitHubSyncer
	GitLab         GitLabSyncer
	Jira           JiraSyncer
	Hiworks        HiworksSyncer
	NewAIGenerator func(apiKey string) AIReportGenerator
}

func DefaultServices() *Services {
	return &Services{
		GitHub:  NewGitHubService(),
		GitLab:  NewGitLabService(),
		Jira:    NewJiraService(),
		Hiworks: NewHiworksService(),
		NewAIGenerator: func(apiKey string) AIReportGenerator {
			return NewClaudeService(apiKey)
		},
	}
}
