package repository

import "github.com/jiin/weeky/internal/model"

// Repository defines the interface for data persistence
type IRepository interface {
	Close() error

	// Template methods
	GetTemplates() ([]model.Template, error)
	CreateTemplate(name, style string) (*model.Template, error)
	UpdateTemplate(id int64, name, style string) error
	DeleteTemplate(id int64) error

	// Report methods
	GetReport(id int64) (*model.Report, error)
	CreateReport(req model.CreateReportRequest) (*model.Report, error)

	// Config methods
	GetConfigs() ([]model.Config, error)
	GetConfig(key string) (*model.Config, error)
	SetConfig(key, value string) error
	DeleteConfig(key string) error
}

// Ensure Repository implements IRepository
var _ IRepository = (*Repository)(nil)
