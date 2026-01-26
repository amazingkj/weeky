package service

import "github.com/jiin/weeky/internal/model"

// GitHubSyncer syncs data from GitHub
type GitHubSyncer interface {
	Sync(req model.GitHubSyncRequest) (*model.SyncResult, error)
}

// GitLabSyncer syncs data from GitLab
type GitLabSyncer interface {
	Sync(req model.GitLabSyncRequest) (*model.SyncResult, error)
}

// JiraSyncer syncs data from Jira
type JiraSyncer interface {
	Sync(req model.JiraSyncRequest) (*model.SyncResult, error)
}

// HiworksSyncer syncs data from Hiworks
type HiworksSyncer interface {
	Sync(req model.HiworksSyncRequest) (*model.SyncResult, error)
}

// AIReportGenerator generates reports using AI
type AIReportGenerator interface {
	GenerateReport(req GenerateReportRequest) (*GenerateReportResponse, error)
}

// Services holds all external service dependencies
type Services struct {
	GitHub         GitHubSyncer
	GitLab         GitLabSyncer
	Jira           JiraSyncer
	Hiworks        HiworksSyncer
	NewAIGenerator func(apiKey string) AIReportGenerator
}

// DefaultServices creates Services with real implementations
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
