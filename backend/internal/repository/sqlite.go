package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jiin/weeky/internal/model"
	_ "modernc.org/sqlite" // Pure Go SQLite driver (no CGO required)
)

type Repository struct {
	db *sql.DB
}

func New(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// SQLite optimizations for concurrent access
	db.SetMaxOpenConns(1) // SQLite supports single writer
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA busy_timeout=5000")

	if err := createTables(db); err != nil {
		return nil, err
	}

	return &Repository{db: db}, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		style TEXT DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		team_name TEXT NOT NULL,
		author_name TEXT NOT NULL,
		report_date TEXT NOT NULL,
		this_week TEXT DEFAULT '[]',
		next_week TEXT DEFAULT '[]',
		issues TEXT DEFAULT '',
		template_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		value TEXT DEFAULT '',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := db.Exec(schema)
	return err
}

func (r *Repository) Close() error {
	return r.db.Close()
}

// Template methods
func (r *Repository) GetTemplates() ([]model.Template, error) {
	rows, err := r.db.Query("SELECT id, name, style, created_at FROM templates ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []model.Template
	for rows.Next() {
		var t model.Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Style, &t.CreatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *Repository) CreateTemplate(name, style string) (*model.Template, error) {
	result, err := r.db.Exec("INSERT INTO templates (name, style) VALUES (?, ?)", name, style)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &model.Template{
		ID:        id,
		Name:      name,
		Style:     style,
		CreatedAt: time.Now(),
	}, nil
}

func (r *Repository) UpdateTemplate(id int64, name, style string) error {
	_, err := r.db.Exec("UPDATE templates SET name = ?, style = ? WHERE id = ?", name, style, id)
	return err
}

func (r *Repository) DeleteTemplate(id int64) error {
	_, err := r.db.Exec("DELETE FROM templates WHERE id = ?", id)
	return err
}

// Report methods
func (r *Repository) GetReport(id int64) (*model.Report, error) {
	var report model.Report
	var thisWeekJSON, nextWeekJSON string

	err := r.db.QueryRow(
		"SELECT id, team_name, author_name, report_date, this_week, next_week, issues, template_id, created_at FROM reports WHERE id = ?",
		id,
	).Scan(&report.ID, &report.TeamName, &report.AuthorName, &report.ReportDate, &thisWeekJSON, &nextWeekJSON, &report.Issues, &report.TemplateID, &report.CreatedAt)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(thisWeekJSON), &report.ThisWeek); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(nextWeekJSON), &report.NextWeek); err != nil {
		return nil, err
	}

	return &report, nil
}

func (r *Repository) CreateReport(req model.CreateReportRequest) (*model.Report, error) {
	thisWeekJSON, err := json.Marshal(req.ThisWeek)
	if err != nil {
		return nil, err
	}
	nextWeekJSON, err := json.Marshal(req.NextWeek)
	if err != nil {
		return nil, err
	}

	result, err := r.db.Exec(
		"INSERT INTO reports (team_name, author_name, report_date, this_week, next_week, issues, template_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		req.TeamName, req.AuthorName, req.ReportDate, string(thisWeekJSON), string(nextWeekJSON), req.Issues, req.TemplateID,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &model.Report{
		ID:         id,
		TeamName:   req.TeamName,
		AuthorName: req.AuthorName,
		ReportDate: req.ReportDate,
		ThisWeek:   req.ThisWeek,
		NextWeek:   req.NextWeek,
		Issues:     req.Issues,
		TemplateID: req.TemplateID,
		CreatedAt:  time.Now(),
	}, nil
}

// Config methods
func (r *Repository) GetConfigs() ([]model.Config, error) {
	rows, err := r.db.Query("SELECT id, key, value, updated_at FROM configs ORDER BY key")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []model.Config
	for rows.Next() {
		var c model.Config
		if err := rows.Scan(&c.ID, &c.Key, &c.Value, &c.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *Repository) GetConfig(key string) (*model.Config, error) {
	var c model.Config
	err := r.db.QueryRow("SELECT id, key, value, updated_at FROM configs WHERE key = ?", key).
		Scan(&c.ID, &c.Key, &c.Value, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) SetConfig(key, value string) error {
	_, err := r.db.Exec(`
		INSERT INTO configs (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, key, value)
	return err
}

func (r *Repository) DeleteConfig(key string) error {
	_, err := r.db.Exec("DELETE FROM configs WHERE key = ?", key)
	return err
}
