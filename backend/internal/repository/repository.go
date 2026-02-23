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

	// Report methods
	GetReport(id int64, userID int64) (*model.Report, error)
	CreateReport(req model.CreateReportRequest, userID int64) (*model.Report, error)
	GetReportsByUser(userID int64) ([]model.Report, error)

	// Config methods
	GetConfigs(userID int64) ([]model.Config, error)
	GetConfig(key string, userID int64) (*model.Config, error)
	SetConfig(key, value string, userID int64) error
	DeleteConfig(key string, userID int64) error
}

// Ensure Repository implements IRepository
var _ IRepository = (*Repository)(nil)
