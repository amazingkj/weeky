package model

import "time"

// User represents an authenticated user
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
}

// InviteCode represents an invitation code for new user registration
type InviteCode struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	CreatedBy int64      `json:"created_by"`
	UsedBy    *int64     `json:"used_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

// Auth request/response types
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	InviteCode string `json:"invite_code"`
}

type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type Task struct {
	Title       string `json:"title"`
	Details     string `json:"details,omitempty"`
	Description string `json:"description,omitempty"` // 진행사항 상세내용
	DueDate     string `json:"due_date"`
	Progress    int    `json:"progress"` // 0-100
}

type Template struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Style     string    `json:"style"` // JSON: font, colors, etc
	CreatedAt time.Time `json:"created_at"`
}

type Report struct {
	ID         int64     `json:"id"`
	TeamName   string    `json:"team_name"`
	AuthorName string    `json:"author_name"`
	ReportDate string    `json:"report_date"`
	ThisWeek   []Task    `json:"this_week"`
	NextWeek   []Task    `json:"next_week"`
	Issues     string    `json:"issues"`
	Notes      string    `json:"notes"`
	NextIssues string    `json:"next_issues"`
	NextNotes  string    `json:"next_notes"`
	TemplateID int64     `json:"template_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateTemplateRequest struct {
	Name  string `json:"name"`
	Style string `json:"style"`
}

type CreateReportRequest struct {
	TeamName   string `json:"team_name"`
	AuthorName string `json:"author_name"`
	ReportDate string `json:"report_date"`
	ThisWeek   []Task `json:"this_week"`
	NextWeek   []Task `json:"next_week"`
	Issues     string `json:"issues"`
	Notes      string `json:"notes"`
	NextIssues string `json:"next_issues"`
	NextNotes  string `json:"next_notes"`
	TemplateID int64  `json:"template_id"`
}

// Config stores encrypted API tokens and settings
type Config struct {
	ID        int64     `json:"id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"` // encrypted
	UpdatedAt time.Time `json:"updated_at"`
}

// SyncItem represents a single item from external services
type SyncItem struct {
	Title   string `json:"title"`
	Content string `json:"content,omitempty"` // 메일 본문 등 상세 내용
	Date    string `json:"date"`
	URL     string `json:"url"`
	Type    string `json:"type"`              // commit, pr, issue, email
	Source  string `json:"source,omitempty"`   // 출처 프로젝트명 (e.g., "group/project")
}

// SyncResult contains results from external service sync
type SyncResult struct {
	Source   string     `json:"source"` // github, jira, gmail
	Items    []SyncItem `json:"items"`
	SyncedAt time.Time  `json:"synced_at"`
}

// GitHub sync request
type GitHubSyncRequest struct {
	Token     string `json:"token"`
	Owner     string `json:"owner"`
	Repo      string `json:"repo"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// GitLab sync request
type GitLabSyncRequest struct {
	Token     string `json:"token"`
	BaseURL   string `json:"base_url"`   // e.g., https://gitlab.com or self-hosted
	Namespace string `json:"namespace"`  // group or username
	Project   string `json:"project"`    // project name
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// Jira sync request
type JiraSyncRequest struct {
	BaseURL   string `json:"base_url"`
	Email     string `json:"email"`
	Token     string `json:"token"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// Hiworks sync request (web scraping)
type HiworksSyncRequest struct {
	OfficeID  string `json:"office_id"`  // 회사 ID (xxx.hiworks.com의 xxx)
	UserID    string `json:"user_id"`    // 사용자 ID
	Password  string `json:"password"`   // 비밀번호
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// GitLabProject represents a GitLab project for multi-project selection
type GitLabProject struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	FullPath  string `json:"full_path"`
	Namespace string `json:"namespace"`
	Project   string `json:"project"`
	WebURL    string `json:"web_url"`
}

// ConfigUpdateRequest for updating config
type ConfigUpdateRequest struct {
	Configs map[string]string `json:"configs"`
}
