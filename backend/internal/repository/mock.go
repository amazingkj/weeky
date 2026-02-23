package repository

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jiin/weeky/internal/model"
)

// MockRepository is an in-memory implementation for testing
type MockRepository struct {
	mu          sync.RWMutex
	users       map[int64]model.User
	inviteCodes map[string]model.InviteCode
	templates   map[int64]model.Template
	reports     map[int64]model.Report
	reportOwner map[int64]int64          // reportID -> userID
	configs     map[string]model.Config  // key format: "userID:key"
	nextID      int64
}

// NewMock creates a new mock repository
func NewMock() *MockRepository {
	return &MockRepository{
		users:       make(map[int64]model.User),
		inviteCodes: make(map[string]model.InviteCode),
		templates:   make(map[int64]model.Template),
		reports:     make(map[int64]model.Report),
		reportOwner: make(map[int64]int64),
		configs:     make(map[string]model.Config),
		nextID:      1,
	}
}

func (m *MockRepository) Close() error {
	return nil
}

// ============ User methods ============

func (m *MockRepository) CreateUser(email, passwordHash, name string, isAdmin bool) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check unique email
	for _, u := range m.users {
		if u.Email == email {
			return nil, errors.New("email already exists")
		}
	}

	id := m.nextID
	m.nextID++
	user := model.User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		IsAdmin:      isAdmin,
		CreatedAt:    time.Now(),
	}
	m.users[id] = user
	return &user, nil
}

func (m *MockRepository) CreateFirstAdmin(email, passwordHash, name string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.users) > 0 {
		return nil, nil
	}

	id := m.nextID
	m.nextID++
	user := model.User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		IsAdmin:      true,
		CreatedAt:    time.Now(),
	}
	m.users[id] = user
	return &user, nil
}

func (m *MockRepository) GetUserByEmail(email string) (*model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, u := range m.users {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *MockRepository) GetUserByID(id int64) (*model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return &u, nil
}

func (m *MockRepository) CountUsers() (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.users)), nil
}

func (m *MockRepository) ReassignLegacyData(userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Mock: reassign configs with userID=0
	for k, c := range m.configs {
		expected := configKey(0, c.Key)
		if k == expected {
			delete(m.configs, k)
			newKey := configKey(userID, c.Key)
			m.configs[newKey] = c
		}
	}
	// Mock: reassign reports with userID=0
	for id, ownerID := range m.reportOwner {
		if ownerID == 0 {
			m.reportOwner[id] = userID
		}
	}
	return nil
}

// ============ Invite code methods ============

func (m *MockRepository) CreateInviteCode(code string, createdBy int64) (*model.InviteCode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID
	m.nextID++
	ic := model.InviteCode{
		ID:        id,
		Code:      code,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}
	m.inviteCodes[code] = ic
	return &ic, nil
}

func (m *MockRepository) GetInviteCodeByCode(code string) (*model.InviteCode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ic, ok := m.inviteCodes[code]
	if !ok {
		return nil, errors.New("invite code not found")
	}
	return &ic, nil
}

func (m *MockRepository) UseInviteCode(code string, usedBy int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ic, ok := m.inviteCodes[code]
	if !ok {
		return errors.New("invite code not found")
	}
	if ic.UsedBy != nil {
		return errors.New("invite code already used")
	}
	now := time.Now()
	ic.UsedBy = &usedBy
	ic.UsedAt = &now
	m.inviteCodes[code] = ic
	return nil
}

func (m *MockRepository) GetInviteCodes(createdBy int64) ([]model.InviteCode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var codes []model.InviteCode
	for _, ic := range m.inviteCodes {
		if ic.CreatedBy == createdBy {
			codes = append(codes, ic)
		}
	}
	return codes, nil
}

// ============ Template methods ============

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

// ============ Report methods ============

func (m *MockRepository) GetReport(id int64, userID int64) (*model.Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report, ok := m.reports[id]
	if !ok {
		return nil, errors.New("report not found")
	}
	if ownerID, exists := m.reportOwner[id]; exists && ownerID != userID {
		return nil, errors.New("report not found")
	}
	return &report, nil
}

func (m *MockRepository) CreateReport(req model.CreateReportRequest, userID int64) (*model.Report, error) {
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
		Notes:      req.Notes,
		NextIssues: req.NextIssues,
		NextNotes:  req.NextNotes,
		TemplateID: req.TemplateID,
		CreatedAt:  time.Now(),
	}
	m.reports[id] = report
	m.reportOwner[id] = userID
	return &report, nil
}

func (m *MockRepository) GetReportsByUser(userID int64) ([]model.Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var reports []model.Report
	for id, r := range m.reports {
		if ownerID, exists := m.reportOwner[id]; exists && ownerID == userID {
			reports = append(reports, r)
		}
	}
	return reports, nil
}

// ============ Config methods ============

func configKey(userID int64, key string) string {
	return fmt.Sprintf("%d:%s", userID, key)
}

func (m *MockRepository) GetConfigs(userID int64) ([]model.Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	configs := make([]model.Config, 0)
	for k, c := range m.configs {
		expected := configKey(userID, c.Key)
		if k == expected {
			configs = append(configs, c)
		}
	}
	return configs, nil
}

func (m *MockRepository) GetConfig(key string, userID int64) (*model.Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ck := configKey(userID, key)
	config, ok := m.configs[ck]
	if !ok {
		return nil, errors.New("config not found")
	}
	return &config, nil
}

func (m *MockRepository) SetConfig(key, value string, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ck := configKey(userID, key)
	m.configs[ck] = model.Config{
		ID:        int64(len(m.configs) + 1),
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
	return nil
}

func (m *MockRepository) DeleteConfig(key string, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ck := configKey(userID, key)
	delete(m.configs, ck)
	return nil
}

// Ensure MockRepository implements IRepository
var _ IRepository = (*MockRepository)(nil)
