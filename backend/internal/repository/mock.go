package repository

import (
	"errors"
	"sync"
	"time"

	"github.com/jiin/weeky/internal/model"
)

// MockRepository is an in-memory implementation for testing
type MockRepository struct {
	mu        sync.RWMutex
	templates map[int64]model.Template
	reports   map[int64]model.Report
	configs   map[string]model.Config
	nextID    int64
}

// NewMock creates a new mock repository
func NewMock() *MockRepository {
	return &MockRepository{
		templates: make(map[int64]model.Template),
		reports:   make(map[int64]model.Report),
		configs:   make(map[string]model.Config),
		nextID:    1,
	}
}

func (m *MockRepository) Close() error {
	return nil
}

// Template methods
func (m *MockRepository) GetTemplates() ([]model.Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	templates := make([]model.Template, 0, len(m.templates))
	for _, t := range m.templates {
		templates = append(templates, t)
	}
	return templates, nil
}

func (m *MockRepository) CreateTemplate(name, style string) (*model.Template, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID
	m.nextID++

	template := model.Template{
		ID:        id,
		Name:      name,
		Style:     style,
		CreatedAt: time.Now(),
	}
	m.templates[id] = template
	return &template, nil
}

func (m *MockRepository) UpdateTemplate(id int64, name, style string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.templates[id]; ok {
		t.Name = name
		t.Style = style
		m.templates[id] = t
		return nil
	}
	return errors.New("template not found")
}

func (m *MockRepository) DeleteTemplate(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.templates, id)
	return nil
}

// Report methods
func (m *MockRepository) GetReport(id int64) (*model.Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report, ok := m.reports[id]
	if !ok {
		return nil, errors.New("report not found")
	}
	return &report, nil
}

func (m *MockRepository) CreateReport(req model.CreateReportRequest) (*model.Report, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID
	m.nextID++

	report := model.Report{
		ID:         id,
		TeamName:   req.TeamName,
		AuthorName: req.AuthorName,
		ReportDate: req.ReportDate,
		ThisWeek:   req.ThisWeek,
		NextWeek:   req.NextWeek,
		Issues:     req.Issues,
		TemplateID: req.TemplateID,
		CreatedAt:  time.Now(),
	}
	m.reports[id] = report
	return &report, nil
}

// Config methods
func (m *MockRepository) GetConfigs() ([]model.Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	configs := make([]model.Config, 0, len(m.configs))
	for _, c := range m.configs {
		configs = append(configs, c)
	}
	return configs, nil
}

func (m *MockRepository) GetConfig(key string) (*model.Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, ok := m.configs[key]
	if !ok {
		return nil, errors.New("config not found")
	}
	return &config, nil
}

func (m *MockRepository) SetConfig(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.configs[key] = model.Config{
		ID:        int64(len(m.configs) + 1),
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
	return nil
}

func (m *MockRepository) DeleteConfig(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.configs, key)
	return nil
}

// Ensure MockRepository implements IRepository
var _ IRepository = (*MockRepository)(nil)
