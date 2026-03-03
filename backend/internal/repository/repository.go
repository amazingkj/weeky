package repository

import "github.com/jiin/weeky/internal/model"

// Repository defines the interface for data persistence
type IRepository interface {
	Close() error

	// User methods
	CreateUser(email, passwordHash, name string, isAdmin bool) (*model.User, error)
	// CreateFirstAdmin atomically checks that no users exist and creates the first admin.
	// Returns the user if created, or (nil, nil) if users already exist.
	CreateFirstAdmin(email, passwordHash, name string) (*model.User, error)
	GetUserByEmail(email string) (*model.User, error)
	GetUserByID(id int64) (*model.User, error)
	CountUsers() (int64, error)
	// ReassignLegacyData moves user_id=0 configs/reports to the given user
	ReassignLegacyData(userID int64) error

	// Invite code methods
	CreateInviteCode(code string, createdBy int64) (*model.InviteCode, error)
	GetInviteCodeByCode(code string) (*model.InviteCode, error)
	UseInviteCode(code string, usedBy int64) error
	GetInviteCodes(createdBy int64) ([]model.InviteCode, error)

	// Template methods
	GetTemplates() ([]model.Template, error)
	CreateTemplate(name, style string) (*model.Template, error)
	UpdateTemplate(id int64, name, style string) error
	DeleteTemplate(id int64) error

	// User list
	GetAllUsers() ([]model.User, error)

	// Report methods
	GetReport(id int64, userID int64) (*model.Report, error)
	CreateReport(req model.CreateReportRequest, userID int64) (*model.Report, error)
	UpdateReport(id int64, req model.CreateReportRequest, userID int64) error
	GetReportsByUser(userID int64) ([]model.Report, error)
	GetReportByDateAndUser(reportDate string, userID int64) (*model.Report, error)

	// Config methods
	GetConfigs(userID int64) ([]model.Config, error)
	GetConfig(key string, userID int64) (*model.Config, error)
	SetConfig(key, value string, userID int64) error
	DeleteConfig(key string, userID int64) error

	// Team methods
	CreateTeam(name, description string, createdBy int64) (*model.Team, error)
	GetTeam(id int64) (*model.Team, error)
	GetTeamsByUser(userID int64) ([]model.Team, error)
	UpdateTeam(id int64, name, description string) error
	DeleteTeam(id int64) error

	// Team member methods
	AddTeamMember(teamID, userID int64, role model.TeamRole, roleCode model.RoleCode) (*model.TeamMember, error)
	GetTeamMembers(teamID int64) ([]model.TeamMember, error)
	GetTeamMember(teamID, userID int64) (*model.TeamMember, error)
	UpdateTeamMember(id int64, role model.TeamRole, roleCode model.RoleCode, name string) error
	RemoveTeamMember(id int64) error

	// Report submission methods
	SubmitReport(reportID, teamID, userID int64) (*model.ReportSubmission, error)
	UnsubmitReport(reportID, teamID int64) error
	GetSubmissions(teamID int64, reportDate string) ([]model.ReportSubmission, error)
	GetSubmissionByUser(teamID, userID int64, reportDate string) (*model.ReportSubmission, error)
	GetSubmissionsByUser(teamID, userID int64) ([]model.ReportSubmission, error)

	// Consolidated report
	GetReportByID(id int64) (*model.Report, error)
	UpdateReportByID(id int64, req model.CreateReportRequest) error

	// Team project methods
	CreateTeamProject(teamID int64, name, client string) (*model.TeamProject, error)
	GetTeamProjects(teamID int64, activeOnly bool) ([]model.TeamProject, error)
	GetTeamProject(id int64) (*model.TeamProject, error)
	UpdateTeamProject(id int64, name, client string, isActive *bool) error
	DeleteTeamProject(id int64) error
	GetOrCreateTeamProject(teamID int64, name string) (*model.TeamProject, error)
	ReorderTeamProjects(teamID int64, ids []int64) error

	// Consolidated edit methods
	SaveConsolidatedEdit(teamID int64, reportDate, data string, updatedBy int64) error
	GetConsolidatedEdit(teamID int64, reportDate string) (*model.ConsolidatedEdit, error)
	DeleteConsolidatedEdit(teamID int64, reportDate string) error
}

// Ensure Repository implements IRepository
var _ IRepository = (*Repository)(nil)
