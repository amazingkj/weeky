package model

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
}

type InviteCode struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	CreatedBy int64      `json:"created_by"`
	UsedBy    *int64     `json:"used_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

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
	Client      string `json:"client,omitempty"`      // 고객사명
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

type Config struct {
	ID        int64     `json:"id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"` // encrypted
	UpdatedAt time.Time `json:"updated_at"`
}

type SyncItem struct {
	Title    string `json:"title"`
	Content  string `json:"content,omitempty"`  // 메일 본문 등 상세 내용
	Date     string `json:"date"`
	URL      string `json:"url"`
	Type     string `json:"type"`               // commit, pr, issue, email
	Source   string `json:"source,omitempty"`   // 출처 프로젝트명 (e.g., "group/project")
	DueDate  string `json:"due_date,omitempty"` // Jira 기한
	Solution string `json:"solution,omitempty"` // Jira 솔루션명 (e.g., "CruzAPIM 1.5")
	Site     string `json:"site,omitempty"`     // Jira 요청사이트 (고객사 매핑)
}

type SyncResult struct {
	Source   string     `json:"source"` // github, jira, gmail
	Items    []SyncItem `json:"items"`
	SyncedAt time.Time  `json:"synced_at"`
}

type GitHubSyncRequest struct {
	Token     string `json:"token"`
	Owner     string `json:"owner"`
	Repo      string `json:"repo"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type GitLabSyncRequest struct {
	Token     string `json:"token"`
	BaseURL   string `json:"base_url"`   // e.g., https://gitlab.com or self-hosted
	Namespace string `json:"namespace"`  // group or username
	Project   string `json:"project"`    // project name
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type JiraSyncRequest struct {
	BaseURL   string `json:"base_url"`
	Email     string `json:"email"`
	Token     string `json:"token"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type HiworksSyncRequest struct {
	OfficeID  string `json:"office_id"`  // 회사 ID (xxx.hiworks.com의 xxx)
	UserID    string `json:"user_id"`    // 사용자 ID
	Password  string `json:"password"`   // 비밀번호
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type GitLabProject struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	FullPath  string `json:"full_path"`
	Namespace string `json:"namespace"`
	Project   string `json:"project"`
	WebURL    string `json:"web_url"`
}

type ConfigUpdateRequest struct {
	Configs map[string]string `json:"configs"`
}

type TeamRole string

const (
	TeamRoleLeader      TeamRole = "leader"
	TeamRoleGroupLeader TeamRole = "group_leader"
	TeamRoleMember      TeamRole = "member"
)

type RoleCode string

const (
	RoleCodeS RoleCode = "S" // 선임
	RoleCodeD RoleCode = "D" // 대리
	RoleCodeG RoleCode = "G" // 과장
	RoleCodeC RoleCode = "C" // 차장
	RoleCodeB RoleCode = "B" // 부장
)

type Team struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type TeamMember struct {
	ID       int64    `json:"id"`
	TeamID   int64    `json:"team_id"`
	UserID   int64    `json:"user_id"`
	Role     TeamRole `json:"role"`
	RoleCode RoleCode `json:"role_code"`
	JoinedAt time.Time `json:"joined_at"`
	UserName  string `json:"user_name,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
}

type ReportSubmission struct {
	ID          int64      `json:"id"`
	ReportID    int64      `json:"report_id"`
	TeamID      int64      `json:"team_id"`
	UserID      int64      `json:"user_id"`
	Status      string     `json:"status"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UserName   string `json:"user_name,omitempty"`
	UserEmail  string `json:"user_email,omitempty"`
	ReportDate string `json:"report_date,omitempty"`
}

type CreateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AddTeamMemberRequest struct {
	Email    string   `json:"email"`
	Role     TeamRole `json:"role"`
	RoleCode RoleCode `json:"role_code"`
}

type UpdateTeamMemberRequest struct {
	Role     TeamRole `json:"role"`
	RoleCode RoleCode `json:"role_code"`
	Name     string   `json:"name"`
}

type SubmitReportRequest struct {
	ReportID int64 `json:"report_id"`
}

type MemberReportData struct {
	UserID   int64    `json:"user_id"`
	UserName string   `json:"user_name"`
	RoleCode RoleCode `json:"role_code"`
	Report   *Report  `json:"report"`
}

type ConsolidatedReport struct {
	Team       Team               `json:"team"`
	ReportDate string             `json:"report_date"`
	Members    []MemberReportData `json:"members"`
}

type TeamProject struct {
	ID        int64     `json:"id"`
	TeamID    int64     `json:"team_id"`
	Name      string    `json:"name"`
	Client    string    `json:"client"`
	IsActive  bool      `json:"is_active"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTeamProjectRequest struct {
	Name   string `json:"name"`
	Client string `json:"client"`
}

type UpdateTeamProjectRequest struct {
	Name     string `json:"name"`
	Client   string `json:"client"`
	IsActive *bool  `json:"is_active,omitempty"`
}

type ReorderProjectsRequest struct {
	IDs []int64 `json:"ids"`
}

type ConsolidatedEdit struct {
	ID         int64     `json:"id"`
	TeamID     int64     `json:"team_id"`
	ReportDate string    `json:"report_date"`
	Data       string    `json:"data"` // JSON blob
	UpdatedBy  int64     `json:"updated_by"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SaveConsolidatedEditRequest struct {
	ReportDate string          `json:"report_date"`
	ThisWeek   json.RawMessage `json:"this_week"`
	NextWeek   json.RawMessage `json:"next_week"`
	Issues     string          `json:"issues"`
	Notes      string          `json:"notes"`
	NextIssues string          `json:"next_issues"`
	NextNotes  string          `json:"next_notes"`
}

type WeekSummary struct {
	WeekDate       string   `json:"week_date"`
	FridayDate     string   `json:"friday_date"`
	SubmittedCount int      `json:"submitted_count"`
	TotalMembers   int      `json:"total_members"`
	SubmittedNames []string `json:"submitted_names"`
}

type TeamHistoryResponse struct {
	TeamID   int64         `json:"team_id"`
	TeamName string        `json:"team_name"`
	Weeks    []WeekSummary `json:"weeks"`
}
