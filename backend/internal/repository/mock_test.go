package repository

import (
	"testing"

	"github.com/jiin/weeky/internal/model"
)

const testUserID int64 = 1

func TestMockConfigCRUD(t *testing.T) {
	repo := NewMock()
	defer repo.Close()

	// Test SetConfig
	t.Run("SetConfig", func(t *testing.T) {
		err := repo.SetConfig("test_key", "test_value", testUserID)
		if err != nil {
			t.Errorf("SetConfig failed: %v", err)
		}
	})

	// Test GetConfig
	t.Run("GetConfig", func(t *testing.T) {
		cfg, err := repo.GetConfig("test_key", testUserID)
		if err != nil {
			t.Errorf("GetConfig failed: %v", err)
		}
		if cfg.Value != "test_value" {
			t.Errorf("Expected value 'test_value', got '%s'", cfg.Value)
		}
	})

	// Test GetConfig for non-existent key
	t.Run("GetConfig_NotFound", func(t *testing.T) {
		_, err := repo.GetConfig("non_existent_key", testUserID)
		if err == nil {
			t.Error("Expected error for non-existent key")
		}
	})

	// Test SetConfig update
	t.Run("SetConfig_Update", func(t *testing.T) {
		err := repo.SetConfig("test_key", "updated_value", testUserID)
		if err != nil {
			t.Errorf("SetConfig update failed: %v", err)
		}

		cfg, err := repo.GetConfig("test_key", testUserID)
		if err != nil {
			t.Errorf("GetConfig after update failed: %v", err)
		}
		if cfg.Value != "updated_value" {
			t.Errorf("Expected value 'updated_value', got '%s'", cfg.Value)
		}
	})

	// Test GetConfigs
	t.Run("GetConfigs", func(t *testing.T) {
		repo.SetConfig("key1", "value1", testUserID)
		repo.SetConfig("key2", "value2", testUserID)

		configs, err := repo.GetConfigs(testUserID)
		if err != nil {
			t.Errorf("GetConfigs failed: %v", err)
		}
		if len(configs) < 2 {
			t.Errorf("Expected at least 2 configs, got %d", len(configs))
		}
	})

	// Test DeleteConfig
	t.Run("DeleteConfig", func(t *testing.T) {
		err := repo.DeleteConfig("test_key", testUserID)
		if err != nil {
			t.Errorf("DeleteConfig failed: %v", err)
		}

		_, err = repo.GetConfig("test_key", testUserID)
		if err == nil {
			t.Error("Expected error after delete")
		}
	})
}

func TestMockTemplateCRUD(t *testing.T) {
	repo := NewMock()
	defer repo.Close()

	// Test CreateTemplate
	t.Run("CreateTemplate", func(t *testing.T) {
		template, err := repo.CreateTemplate("Test Template", `{"color": "blue"}`)
		if err != nil {
			t.Errorf("CreateTemplate failed: %v", err)
		}
		if template.Name != "Test Template" {
			t.Errorf("Expected name 'Test Template', got '%s'", template.Name)
		}
		if template.ID == 0 {
			t.Error("Expected non-zero ID")
		}
	})

	// Test GetTemplates
	t.Run("GetTemplates", func(t *testing.T) {
		templates, err := repo.GetTemplates()
		if err != nil {
			t.Errorf("GetTemplates failed: %v", err)
		}
		if len(templates) == 0 {
			t.Error("Expected at least 1 template")
		}
	})

	// Test DeleteTemplate
	t.Run("DeleteTemplate", func(t *testing.T) {
		template, _ := repo.CreateTemplate("To Delete", "{}")
		err := repo.DeleteTemplate(template.ID)
		if err != nil {
			t.Errorf("DeleteTemplate failed: %v", err)
		}
	})
}

func TestMockReportCRUD(t *testing.T) {
	repo := NewMock()
	defer repo.Close()

	// Test CreateReport
	t.Run("CreateReport", func(t *testing.T) {
		req := model.CreateReportRequest{
			TeamName:   "개발팀",
			AuthorName: "홍길동",
			ReportDate: "2024-01-15",
			ThisWeek: []model.Task{
				{Title: "기능 개발", DueDate: "2024-01-15", Progress: 100},
			},
			NextWeek: []model.Task{
				{Title: "테스트", DueDate: "2024-01-22", Progress: 0},
			},
			Issues:     "특이사항 없음",
			TemplateID: 0,
		}

		report, err := repo.CreateReport(req, testUserID)
		if err != nil {
			t.Errorf("CreateReport failed: %v", err)
		}
		if report.TeamName != "개발팀" {
			t.Errorf("Expected team '개발팀', got '%s'", report.TeamName)
		}
		if len(report.ThisWeek) != 1 {
			t.Errorf("Expected 1 this_week task, got %d", len(report.ThisWeek))
		}
	})

	// Test GetReport
	t.Run("GetReport", func(t *testing.T) {
		req := model.CreateReportRequest{
			TeamName:   "QA팀",
			AuthorName: "김철수",
			ReportDate: "2024-01-16",
			ThisWeek:   []model.Task{},
			NextWeek:   []model.Task{},
			Issues:     "",
			TemplateID: 0,
		}
		created, _ := repo.CreateReport(req, testUserID)

		report, err := repo.GetReport(created.ID, testUserID)
		if err != nil {
			t.Errorf("GetReport failed: %v", err)
		}
		if report.AuthorName != "김철수" {
			t.Errorf("Expected author '김철수', got '%s'", report.AuthorName)
		}
	})

	// Test GetReport not found
	t.Run("GetReport_NotFound", func(t *testing.T) {
		_, err := repo.GetReport(99999, testUserID)
		if err == nil {
			t.Error("Expected error for non-existent report")
		}
	})
}
