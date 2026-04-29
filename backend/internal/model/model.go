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

type ConsolidationRuleType string

const (
	// RuleTypeRenameTitle: 모든 task의 title=pattern → replacement
	RuleTypeRenameTitle ConsolidationRuleType = "rename_title"
	// RuleTypeVirtualClient: title=scope_title이고 client가 비어 있는 task의 client → replacement
	// (예: scope_title="CruzAPIM", replacement="본사" → CruzAPIM 안의 고객사 없는 작업이 "본사"로 묶임)
	RuleTypeVirtualClient ConsolidationRuleType = "virtual_client"
)

type ConsolidationRule struct {
	ID          int64                 `json:"id"`
	TeamID      int64                 `json:"team_id"`
	RuleType    ConsolidationRuleType `json:"rule_type"`
	Pattern     string                `json:"pattern"`               // rename_title: 원본 title / virtual_client: 사용 안 함
	Replacement string                `json:"replacement"`           // 변경 대상 값
	ScopeTitle  string                `json:"scope_title,omitempty"` // virtual_client에서 어느 title 안에서 적용할지
	SortOrder   int                   `json:"sort_order"`
	CreatedAt   time.Time             `json:"created_at"`
}

type CreateConsolidationRuleRequest struct {
	RuleType    ConsolidationRuleType `json:"rule_type"`
	Pattern     string                `json:"pattern"`
	Replacement string                `json:"replacement"`
	ScopeTitle  string                `json:"scope_title,omitempty"`
}

type UpdateConsolidationRuleRequest struct {
	RuleType    ConsolidationRuleType `json:"rule_type"`
	Pattern     string                `json:"pattern"`
	Replacement string                `json:"replacement"`
	ScopeTitle  string                `json:"scope_title,omitempty"`
}

type ReorderConsolidationRulesRequest struct {
	IDs []int64 `json:"ids"`
}

// --- 사이트 파견 보고서 (Site Dispatch Report) ---
//
// 본사 APIM 양식과는 별도 양식으로, 고객사 현장 파견 인원이 작성한다.
// 취합 시 ConsolidationRule/ConsolidatedEdit 적용 없이 그대로 PPT 뒤에 append된다.

// SiteTask: 금주실적 5컬럼 (계획업무 | 소요일 | 시작일 | 완료일 | 실적)
type SiteTask struct {
	Title       string `json:"title"`        // 계획업무 (■/<>/번호 포함 멀티라인 텍스트)
	ElapsedDays string `json:"elapsed_days"` // 소요일
	StartDate   string `json:"start_date"`   // 시작일
	DueDate     string `json:"due_date"`     // 완료일
	Progress    string `json:"progress"`     // 실적 (% 또는 - 등 자유 텍스트)
}

// SiteNextTask: 차주계획 3컬럼 (계획업무 | 시작예정일 | 완료예정일)
type SiteNextTask struct {
	Title     string `json:"title"`
	StartDate string `json:"start_date"`
	DueDate   string `json:"due_date"`
}

type SiteProject struct {
	ID          int64               `json:"id"`
	TeamID      int64               `json:"team_id"`
	ProjectName string              `json:"project_name"` // 헤더 출력용 (예: "한화손해보험 마이데이터, 유지보수")
	ClientName  string              `json:"client_name"`  // 고객사명 (예: "한화손해보험")
	IsActive    bool                `json:"is_active"`
	SortOrder   int                 `json:"sort_order"`
	CreatedAt   time.Time           `json:"created_at"`
	Authors     []SiteProjectAuthor `json:"authors,omitempty"`
}

type SiteProjectAuthor struct {
	SiteProjectID int64  `json:"site_project_id"`
	UserID        int64  `json:"user_id"`
	UserName      string `json:"user_name,omitempty"`
	UserEmail     string `json:"user_email,omitempty"`
	SortOrder     int    `json:"sort_order"`
}

type SiteReport struct {
	ID             int64          `json:"id"`
	TeamID         int64          `json:"team_id"`
	SiteProjectID  int64          `json:"site_project_id"`
	AuthorUserID   int64          `json:"author_user_id"`             // 마지막으로 저장한 사용자
	AuthorNames    []string       `json:"author_names"`               // 헤더 출력용 스냅샷 (작성자 다수 시 줄바꿈으로 표기)
	ProjectName    string         `json:"project_name"`               // 스냅샷
	ReportDate     string         `json:"report_date"`                // weekRange 매핑용 (YYYY-MM-DD)
	ReportDateText string         `json:"report_date_text,omitempty"` // 헤더에 그대로 표시 (작성자 입력)
	ThisWeek       []SiteTask     `json:"this_week"`
	NextWeek       []SiteNextTask `json:"next_week"`
	Notes          string         `json:"notes"` // 특이사항 (단일 영역)
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type CreateSiteProjectRequest struct {
	ProjectName string  `json:"project_name"`
	ClientName  string  `json:"client_name"`
	AuthorIDs   []int64 `json:"author_ids"`
}

type UpdateSiteProjectRequest struct {
	ProjectName string  `json:"project_name"`
	ClientName  string  `json:"client_name"`
	IsActive    *bool   `json:"is_active,omitempty"`
	AuthorIDs   []int64 `json:"author_ids"` // nil이면 변경 안 함, []이면 모두 해제
}

type SaveSiteReportRequest struct {
	SiteProjectID  int64          `json:"site_project_id"`
	ReportDate     string         `json:"report_date"`
	ReportDateText string         `json:"report_date_text,omitempty"`
	ThisWeek       []SiteTask     `json:"this_week"`
	NextWeek       []SiteNextTask `json:"next_week"`
	Notes          string         `json:"notes"`
}
