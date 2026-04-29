package repository

import "github.com/jiin/weeky/internal/model"

type IRepository interface {
	Close() error

	CreateUser(email, passwordHash, name string, isAdmin bool) (*model.User, error)
	CreateFirstAdmin(email, passwordHash, name string) (*model.User, error)
	GetUserByEmail(email string) (*model.User, error)
	GetUserByID(id int64) (*model.User, error)
	CountUsers() (int64, error)
	ReassignLegacyData(userID int64) error
	UpdateUserPassword(userID int64, passwordHash string) error
	UpdateUserAdmin(userID int64, isAdmin bool) error

	CreateInviteCode(code string, createdBy int64) (*model.InviteCode, error)
	GetInviteCodeByCode(code string) (*model.InviteCode, error)
	UseInviteCode(code string, usedBy int64) error
	GetInviteCodes(createdBy int64) ([]model.InviteCode, error)

	GetTemplates() ([]model.Template, error)
	CreateTemplate(name, style string) (*model.Template, error)
	UpdateTemplate(id int64, name, style string) error
	DeleteTemplate(id int64) error

	GetAllUsers() ([]model.User, error)

	GetReport(id int64, userID int64) (*model.Report, error)
	CreateReport(req model.CreateReportRequest, userID int64) (*model.Report, error)
	UpdateReport(id int64, req model.CreateReportRequest, userID int64) error
	GetReportsByUser(userID int64) ([]model.Report, error)
	GetReportByDateAndUser(reportDate string, userID int64) (*model.Report, error)

	GetConfigs(userID int64) ([]model.Config, error)
	GetConfig(key string, userID int64) (*model.Config, error)
	SetConfig(key, value string, userID int64) error
	DeleteConfig(key string, userID int64) error

	CreateTeam(name, description string, createdBy int64) (*model.Team, error)
	GetTeam(id int64) (*model.Team, error)
	GetTeamsByUser(userID int64) ([]model.Team, error)
	UpdateTeam(id int64, name, description string) error
	DeleteTeam(id int64) error

	AddTeamMember(teamID, userID int64, role model.TeamRole, roleCode model.RoleCode) (*model.TeamMember, error)
	GetTeamMembers(teamID int64) ([]model.TeamMember, error)
	GetTeamMember(teamID, userID int64) (*model.TeamMember, error)
	UpdateTeamMember(id int64, role model.TeamRole, roleCode model.RoleCode, name string) error
	RemoveTeamMember(id int64) error

	SubmitReport(reportID, teamID, userID int64) (*model.ReportSubmission, error)
	UnsubmitReport(reportID, teamID int64) error
	GetSubmissions(teamID int64, reportDate string) ([]model.ReportSubmission, error)
	GetSubmissionByUser(teamID, userID int64, reportDate string) (*model.ReportSubmission, error)
	GetSubmissionsByUser(teamID, userID int64) ([]model.ReportSubmission, error)

	GetReportByID(id int64) (*model.Report, error)
	UpdateReportByID(id int64, req model.CreateReportRequest) error

	CreateTeamProject(teamID int64, name, client string) (*model.TeamProject, error)
	GetTeamProjects(teamID int64, activeOnly bool) ([]model.TeamProject, error)
	GetTeamProject(id int64) (*model.TeamProject, error)
	UpdateTeamProject(id int64, name, client string, isActive *bool) error
	DeleteTeamProject(id int64) error
	GetOrCreateTeamProject(teamID int64, name string) (*model.TeamProject, error)
	ReorderTeamProjects(teamID int64, ids []int64) error

	SaveConsolidatedEdit(teamID int64, reportDate, data string, updatedBy int64) error
	GetConsolidatedEdit(teamID int64, reportDate string) (*model.ConsolidatedEdit, error)
	DeleteConsolidatedEdit(teamID int64, reportDate string) error

	CreateConsolidationRule(teamID int64, req model.CreateConsolidationRuleRequest) (*model.ConsolidationRule, error)
	GetConsolidationRules(teamID int64) ([]model.ConsolidationRule, error)
	GetConsolidationRule(id int64) (*model.ConsolidationRule, error)
	UpdateConsolidationRule(id int64, req model.UpdateConsolidationRuleRequest) error
	DeleteConsolidationRule(id int64) error
	ReorderConsolidationRules(teamID int64, ids []int64) error

	CreateSiteProject(teamID int64, req model.CreateSiteProjectRequest) (*model.SiteProject, error)
	GetSiteProjects(teamID int64, activeOnly bool) ([]model.SiteProject, error)
	GetSiteProject(id int64) (*model.SiteProject, error)
	UpdateSiteProject(id int64, req model.UpdateSiteProjectRequest) error
	DeleteSiteProject(id int64) error
	GetSiteProjectsByAuthor(teamID, userID int64) ([]model.SiteProject, error)
	IsSiteProjectAuthor(siteProjectID, userID int64) (bool, error)

	SaveSiteReport(teamID, userID int64, req model.SaveSiteReportRequest) (*model.SiteReport, error)
	GetSiteReport(id int64) (*model.SiteReport, error)
	GetSiteReportByProjectAndDate(siteProjectID int64, reportDate string) (*model.SiteReport, error)
	GetSiteReportsByTeamAndDate(teamID int64, reportDate string) ([]model.SiteReport, error)
}

var _ IRepository = (*Repository)(nil)
