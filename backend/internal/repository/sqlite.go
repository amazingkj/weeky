package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
	db.Exec("PRAGMA foreign_keys=ON")

	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return &Repository{db: db}, nil
}

// Migration system
type migration struct {
	version int
	sql     string
}

var migrations = []migration{
	{
		version: 1,
		sql: `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			is_admin INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS invite_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			created_by INTEGER NOT NULL REFERENCES users(id),
			used_by INTEGER REFERENCES users(id),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			used_at DATETIME
		);

		CREATE TABLE IF NOT EXISTS templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			style TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		-- New reports table with user_id
		CREATE TABLE IF NOT EXISTS reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL DEFAULT 0,
			team_name TEXT NOT NULL,
			author_name TEXT NOT NULL,
			report_date TEXT NOT NULL,
			this_week TEXT DEFAULT '[]',
			next_week TEXT DEFAULT '[]',
			issues TEXT DEFAULT '',
			template_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		-- New configs table with user_id and composite unique key
		CREATE TABLE IF NOT EXISTS configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL DEFAULT 0,
			key TEXT NOT NULL,
			value TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, key)
		);
		`,
	},
}

func runMigrations(db *sql.DB) error {
	// Create migrations table if not exists
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to create _migrations table: %w", err)
	}

	// Check if this is a legacy database (has tables but no migrations record)
	var hasLegacyConfigs bool
	err = db.QueryRow("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='configs'").Scan(&hasLegacyConfigs)
	if err != nil {
		return fmt.Errorf("failed to check legacy tables: %w", err)
	}

	var migrationCount int
	db.QueryRow("SELECT COUNT(*) FROM _migrations").Scan(&migrationCount)

	if hasLegacyConfigs && migrationCount == 0 {
		// Legacy database: migrate existing data
		if err := migrateLegacyDB(db); err != nil {
			return fmt.Errorf("failed to migrate legacy database: %w", err)
		}
		return nil
	}

	// Run pending migrations
	for _, m := range migrations {
		var exists bool
		db.QueryRow("SELECT COUNT(*) > 0 FROM _migrations WHERE version = ?", m.version).Scan(&exists)
		if exists {
			continue
		}

		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("migration v%d failed: %w", m.version, err)
		}

		if _, err := db.Exec("INSERT INTO _migrations (version) VALUES (?)", m.version); err != nil {
			return fmt.Errorf("failed to record migration v%d: %w", m.version, err)
		}
	}

	return nil
}

// migrateLegacyDB handles migration from the old schema (no user_id) to the new schema
func migrateLegacyDB(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create new tables
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			is_admin INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS invite_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			created_by INTEGER NOT NULL REFERENCES users(id),
			used_by INTEGER REFERENCES users(id),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			used_at DATETIME
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create new tables: %w", err)
	}

	// Migrate configs: rename old table, create new, copy data
	_, err = tx.Exec(`
		ALTER TABLE configs RENAME TO _configs_old;

		CREATE TABLE configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL DEFAULT 0,
			key TEXT NOT NULL,
			value TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, key)
		);

		INSERT INTO configs (id, user_id, key, value, updated_at)
		SELECT id, 0, key, value, updated_at FROM _configs_old;

		DROP TABLE _configs_old;
	`)
	if err != nil {
		return fmt.Errorf("failed to migrate configs table: %w", err)
	}

	// Migrate reports: add user_id column
	// Check if user_id column already exists
	var hasUserID bool
	rows, err := tx.Query("PRAGMA table_info(reports)")
	if err != nil {
		return fmt.Errorf("failed to check reports schema: %w", err)
	}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dfltValue *string
		var pk int
		rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk)
		if name == "user_id" {
			hasUserID = true
		}
	}
	rows.Close()

	if !hasUserID {
		_, err = tx.Exec(`ALTER TABLE reports ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0`)
		if err != nil {
			return fmt.Errorf("failed to add user_id to reports: %w", err)
		}
	}

	// Record migration
	_, err = tx.Exec("INSERT INTO _migrations (version) VALUES (?)", 1)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) Close() error {
	return r.db.Close()
}

// ============ User methods ============

func (r *Repository) CreateUser(email, passwordHash, name string, isAdmin bool) (*model.User, error) {
	adminInt := 0
	if isAdmin {
		adminInt = 1
	}
	result, err := r.db.Exec(
		"INSERT INTO users (email, password_hash, name, is_admin) VALUES (?, ?, ?, ?)",
		email, passwordHash, name, adminInt,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &model.User{
		ID:        id,
		Email:     email,
		Name:      name,
		IsAdmin:   isAdmin,
		CreatedAt: time.Now(),
	}, nil
}

func (r *Repository) GetUserByEmail(email string) (*model.User, error) {
	var u model.User
	var isAdmin int
	err := r.db.QueryRow(
		"SELECT id, email, password_hash, name, is_admin, created_at FROM users WHERE email = ?",
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &isAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.IsAdmin = isAdmin == 1
	return &u, nil
}

func (r *Repository) GetUserByID(id int64) (*model.User, error) {
	var u model.User
	var isAdmin int
	err := r.db.QueryRow(
		"SELECT id, email, password_hash, name, is_admin, created_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &isAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.IsAdmin = isAdmin == 1
	return &u, nil
}

func (r *Repository) CountUsers() (int64, error) {
	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (r *Repository) CreateFirstAdmin(email, passwordHash, name string) (*model.User, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Atomically check no users exist inside the transaction
	var count int64
	if err := tx.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, nil // not the first user
	}

	result, err := tx.Exec(
		"INSERT INTO users (email, password_hash, name, is_admin) VALUES (?, ?, ?, 1)",
		email, passwordHash, name,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &model.User{
		ID:        id,
		Email:     email,
		Name:      name,
		IsAdmin:   true,
		CreatedAt: time.Now(),
	}, nil
}

func (r *Repository) ReassignLegacyData(userID int64) error {
	_, err := r.db.Exec("UPDATE configs SET user_id = ? WHERE user_id = 0", userID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec("UPDATE reports SET user_id = ? WHERE user_id = 0", userID)
	return err
}

// ============ Invite code methods ============

func (r *Repository) CreateInviteCode(code string, createdBy int64) (*model.InviteCode, error) {
	result, err := r.db.Exec(
		"INSERT INTO invite_codes (code, created_by) VALUES (?, ?)",
		code, createdBy,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &model.InviteCode{
		ID:        id,
		Code:      code,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}, nil
}

func (r *Repository) GetInviteCodeByCode(code string) (*model.InviteCode, error) {
	var ic model.InviteCode
	err := r.db.QueryRow(
		"SELECT id, code, created_by, used_by, created_at, used_at FROM invite_codes WHERE code = ?",
		code,
	).Scan(&ic.ID, &ic.Code, &ic.CreatedBy, &ic.UsedBy, &ic.CreatedAt, &ic.UsedAt)
	if err != nil {
		return nil, err
	}
	return &ic, nil
}

func (r *Repository) UseInviteCode(code string, usedBy int64) error {
	_, err := r.db.Exec(
		"UPDATE invite_codes SET used_by = ?, used_at = CURRENT_TIMESTAMP WHERE code = ? AND used_by IS NULL",
		usedBy, code,
	)
	return err
}

func (r *Repository) GetInviteCodes(createdBy int64) ([]model.InviteCode, error) {
	rows, err := r.db.Query(
		"SELECT id, code, created_by, used_by, created_at, used_at FROM invite_codes WHERE created_by = ? ORDER BY created_at DESC",
		createdBy,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []model.InviteCode
	for rows.Next() {
		var ic model.InviteCode
		if err := rows.Scan(&ic.ID, &ic.Code, &ic.CreatedBy, &ic.UsedBy, &ic.CreatedAt, &ic.UsedAt); err != nil {
			return nil, err
		}
		codes = append(codes, ic)
	}
	return codes, rows.Err()
}

// ============ Template methods ============

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

// ============ Report methods ============

func (r *Repository) GetReport(id int64, userID int64) (*model.Report, error) {
	var report model.Report
	var thisWeekJSON, nextWeekJSON string

	err := r.db.QueryRow(
		"SELECT id, team_name, author_name, report_date, this_week, next_week, issues, template_id, created_at FROM reports WHERE id = ? AND user_id = ?",
		id, userID,
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

func (r *Repository) CreateReport(req model.CreateReportRequest, userID int64) (*model.Report, error) {
	thisWeekJSON, err := json.Marshal(req.ThisWeek)
	if err != nil {
		return nil, err
	}
	nextWeekJSON, err := json.Marshal(req.NextWeek)
	if err != nil {
		return nil, err
	}

	result, err := r.db.Exec(
		"INSERT INTO reports (user_id, team_name, author_name, report_date, this_week, next_week, issues, template_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		userID, req.TeamName, req.AuthorName, req.ReportDate, string(thisWeekJSON), string(nextWeekJSON), req.Issues, req.TemplateID,
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

func (r *Repository) GetReportsByUser(userID int64) ([]model.Report, error) {
	rows, err := r.db.Query(
		"SELECT id, team_name, author_name, report_date, this_week, next_week, issues, template_id, created_at FROM reports WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.Report
	for rows.Next() {
		var report model.Report
		var thisWeekJSON, nextWeekJSON string
		if err := rows.Scan(&report.ID, &report.TeamName, &report.AuthorName, &report.ReportDate, &thisWeekJSON, &nextWeekJSON, &report.Issues, &report.TemplateID, &report.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(thisWeekJSON), &report.ThisWeek); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(nextWeekJSON), &report.NextWeek); err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

// ============ Config methods ============

func (r *Repository) GetConfigs(userID int64) ([]model.Config, error) {
	rows, err := r.db.Query("SELECT id, key, value, updated_at FROM configs WHERE user_id = ? ORDER BY key", userID)
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

func (r *Repository) GetConfig(key string, userID int64) (*model.Config, error) {
	var c model.Config
	err := r.db.QueryRow("SELECT id, key, value, updated_at FROM configs WHERE key = ? AND user_id = ?", key, userID).
		Scan(&c.ID, &c.Key, &c.Value, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) SetConfig(key, value string, userID int64) error {
	_, err := r.db.Exec(`
		INSERT INTO configs (user_id, key, value, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, userID, key, value)
	return err
}

func (r *Repository) DeleteConfig(key string, userID int64) error {
	_, err := r.db.Exec("DELETE FROM configs WHERE key = ? AND user_id = ?", key, userID)
	return err
}
