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

// weekRange returns the Monday and Sunday of the week containing the given date.
func weekRange(dateStr string) (monday, sunday string) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr, dateStr
	}
	wd := t.Weekday()
	offset := int(wd) - 1 // Monday=0 offset
	if wd == time.Sunday {
		offset = 6
	}
	mon := t.AddDate(0, 0, -offset)
	sun := mon.AddDate(0, 0, 6)
	return mon.Format("2006-01-02"), sun.Format("2006-01-02")
}

func New(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA busy_timeout=5000")
	db.Exec("PRAGMA foreign_keys=ON")

	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return &Repository{db: db}, nil
}

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
	{
		version: 2,
		sql:     `ALTER TABLE reports ADD COLUMN notes TEXT DEFAULT '';`,
	},
	{
		version: 3,
		sql: `ALTER TABLE reports ADD COLUMN next_issues TEXT DEFAULT '';
ALTER TABLE reports ADD COLUMN next_notes TEXT DEFAULT '';`,
	},
	{
		version: 4,
		sql: `CREATE TABLE IF NOT EXISTS teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			created_by INTEGER NOT NULL REFERENCES users(id),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS team_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id),
			role TEXT NOT NULL DEFAULT 'member',
			role_code TEXT NOT NULL DEFAULT 'S',
			joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(team_id, user_id)
		);

		CREATE TABLE IF NOT EXISTS report_submissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			report_id INTEGER NOT NULL REFERENCES reports(id),
			team_id INTEGER NOT NULL REFERENCES teams(id),
			user_id INTEGER NOT NULL REFERENCES users(id),
			status TEXT NOT NULL DEFAULT 'draft',
			submitted_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(report_id, team_id)
		);`,
	},
	{
		version: 5,
		sql: `CREATE TABLE IF NOT EXISTS team_projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			client TEXT DEFAULT '',
			is_active INTEGER DEFAULT 1,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(team_id, name)
		);`,
	},
	{
		version: 6,
		sql: `CREATE TABLE IF NOT EXISTS team_projects_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			client TEXT DEFAULT '',
			is_active INTEGER DEFAULT 1,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(team_id, name, client)
		);
		INSERT INTO team_projects_new (id, team_id, name, client, is_active, sort_order, created_at)
			SELECT id, team_id, name, client, is_active, sort_order, created_at FROM team_projects;
		DROP TABLE team_projects;
		ALTER TABLE team_projects_new RENAME TO team_projects;`,
	},
	{
		version: 7,
		sql: `CREATE TABLE IF NOT EXISTS consolidated_edits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
			report_date TEXT NOT NULL,
			data TEXT NOT NULL,
			updated_by INTEGER NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(team_id, report_date)
		);`,
	},
	{
		version: 8,
		sql: `CREATE TABLE IF NOT EXISTS consolidation_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
			rule_type TEXT NOT NULL,
			pattern TEXT NOT NULL DEFAULT '',
			replacement TEXT NOT NULL DEFAULT '',
			scope_title TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	},
	{
		version: 9,
		sql: `CREATE TABLE IF NOT EXISTS site_projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
			project_name TEXT NOT NULL,
			client_name TEXT DEFAULT '',
			is_active INTEGER NOT NULL DEFAULT 1,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(team_id, project_name)
		);

		CREATE TABLE IF NOT EXISTS site_project_authors (
			site_project_id INTEGER NOT NULL REFERENCES site_projects(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id),
			sort_order INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (site_project_id, user_id)
		);

		CREATE TABLE IF NOT EXISTS site_reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
			site_project_id INTEGER NOT NULL REFERENCES site_projects(id) ON DELETE CASCADE,
			author_user_id INTEGER NOT NULL REFERENCES users(id),
			author_names TEXT NOT NULL DEFAULT '[]',
			project_name TEXT NOT NULL DEFAULT '',
			report_date TEXT NOT NULL,
			report_date_text TEXT NOT NULL DEFAULT '',
			this_week TEXT NOT NULL DEFAULT '[]',
			next_week TEXT NOT NULL DEFAULT '[]',
			notes TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(site_project_id, report_date)
		);`,
	},
}

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to create _migrations table: %w", err)
	}

	var hasLegacyConfigs bool
	err = db.QueryRow("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='configs'").Scan(&hasLegacyConfigs)
	if err != nil {
		return fmt.Errorf("failed to check legacy tables: %w", err)
	}

	var migrationCount int
	db.QueryRow("SELECT COUNT(*) FROM _migrations").Scan(&migrationCount)

	if hasLegacyConfigs && migrationCount == 0 {
		if err := migrateLegacyDB(db); err != nil {
			return fmt.Errorf("failed to migrate legacy database: %w", err)
		}
		return nil
	}

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

func migrateLegacyDB(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

	_, err = tx.Exec("INSERT INTO _migrations (version) VALUES (?)", 1)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) Close() error {
	return r.db.Close()
}

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

func (r *Repository) UpdateUserPassword(userID int64, passwordHash string) error {
	_, err := r.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", passwordHash, userID)
	return err
}

func (r *Repository) UpdateUserAdmin(userID int64, isAdmin bool) error {
	val := 0
	if isAdmin {
		val = 1
	}
	_, err := r.db.Exec("UPDATE users SET is_admin = ? WHERE id = ?", val, userID)
	return err
}

func (r *Repository) ReassignLegacyData(userID int64) error {
	_, err := r.db.Exec("UPDATE configs SET user_id = ? WHERE user_id = 0", userID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec("UPDATE reports SET user_id = ? WHERE user_id = 0", userID)
	return err
}

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
	return templates, rows.Err()
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

func (r *Repository) GetAllUsers() ([]model.User, error) {
	rows, err := r.db.Query("SELECT id, email, name, is_admin, created_at FROM users ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		var isAdmin int
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &isAdmin, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.IsAdmin = isAdmin == 1
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *Repository) GetReport(id int64, userID int64) (*model.Report, error) {
	var report model.Report
	var thisWeekJSON, nextWeekJSON string

	err := r.db.QueryRow(
		"SELECT id, team_name, author_name, report_date, COALESCE(this_week, '[]'), COALESCE(next_week, '[]'), COALESCE(issues, ''), COALESCE(notes, ''), COALESCE(next_issues, ''), COALESCE(next_notes, ''), COALESCE(template_id, 0), created_at FROM reports WHERE id = ? AND user_id = ?",
		id, userID,
	).Scan(&report.ID, &report.TeamName, &report.AuthorName, &report.ReportDate, &thisWeekJSON, &nextWeekJSON, &report.Issues, &report.Notes, &report.NextIssues, &report.NextNotes, &report.TemplateID, &report.CreatedAt)

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
		"INSERT INTO reports (user_id, team_name, author_name, report_date, this_week, next_week, issues, notes, next_issues, next_notes, template_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userID, req.TeamName, req.AuthorName, req.ReportDate, string(thisWeekJSON), string(nextWeekJSON), req.Issues, req.Notes, req.NextIssues, req.NextNotes, req.TemplateID,
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
		Notes:      req.Notes,
		NextIssues: req.NextIssues,
		NextNotes:  req.NextNotes,
		TemplateID: req.TemplateID,
		CreatedAt:  time.Now(),
	}, nil
}

func (r *Repository) UpdateReport(id int64, req model.CreateReportRequest, userID int64) error {
	thisWeekJSON, err := json.Marshal(req.ThisWeek)
	if err != nil {
		return err
	}
	nextWeekJSON, err := json.Marshal(req.NextWeek)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(
		`UPDATE reports SET team_name=?, author_name=?, report_date=?, this_week=?, next_week=?,
		 issues=?, notes=?, next_issues=?, next_notes=?, template_id=? WHERE id=? AND user_id=?`,
		req.TeamName, req.AuthorName, req.ReportDate, string(thisWeekJSON), string(nextWeekJSON),
		req.Issues, req.Notes, req.NextIssues, req.NextNotes, req.TemplateID, id, userID,
	)
	return err
}

func (r *Repository) GetReportByDateAndUser(reportDate string, userID int64) (*model.Report, error) {
	var report model.Report
	var thisWeekJSON, nextWeekJSON string

	mon, sun := weekRange(reportDate)
	err := r.db.QueryRow(
		"SELECT id, team_name, author_name, report_date, COALESCE(this_week, '[]'), COALESCE(next_week, '[]'), COALESCE(issues, ''), COALESCE(notes, ''), COALESCE(next_issues, ''), COALESCE(next_notes, ''), COALESCE(template_id, 0), created_at FROM reports WHERE report_date BETWEEN ? AND ? AND user_id = ? ORDER BY created_at DESC LIMIT 1",
		mon, sun, userID,
	).Scan(&report.ID, &report.TeamName, &report.AuthorName, &report.ReportDate, &thisWeekJSON, &nextWeekJSON, &report.Issues, &report.Notes, &report.NextIssues, &report.NextNotes, &report.TemplateID, &report.CreatedAt)

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

func (r *Repository) GetReportsByUser(userID int64) ([]model.Report, error) {
	rows, err := r.db.Query(
		"SELECT id, team_name, author_name, report_date, COALESCE(this_week, '[]'), COALESCE(next_week, '[]'), COALESCE(issues, ''), COALESCE(notes, ''), COALESCE(next_issues, ''), COALESCE(next_notes, ''), COALESCE(template_id, 0), created_at FROM reports WHERE user_id = ? ORDER BY created_at DESC",
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
		if err := rows.Scan(&report.ID, &report.TeamName, &report.AuthorName, &report.ReportDate, &thisWeekJSON, &nextWeekJSON, &report.Issues, &report.Notes, &report.NextIssues, &report.NextNotes, &report.TemplateID, &report.CreatedAt); err != nil {
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
	return configs, rows.Err()
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

func (r *Repository) CreateTeam(name, description string, createdBy int64) (*model.Team, error) {
	result, err := r.db.Exec(
		"INSERT INTO teams (name, description, created_by) VALUES (?, ?, ?)",
		name, description, createdBy,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &model.Team{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}, nil
}

func (r *Repository) GetTeam(id int64) (*model.Team, error) {
	var t model.Team
	err := r.db.QueryRow(
		"SELECT id, name, description, created_by, created_at FROM teams WHERE id = ?", id,
	).Scan(&t.ID, &t.Name, &t.Description, &t.CreatedBy, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) GetTeamsByUser(userID int64) ([]model.Team, error) {
	rows, err := r.db.Query(
		`SELECT t.id, t.name, t.description, t.created_by, t.created_at
		 FROM teams t
		 JOIN team_members tm ON t.id = tm.team_id
		 WHERE tm.user_id = ?
		 ORDER BY t.created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []model.Team
	for rows.Next() {
		var t model.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.CreatedBy, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (r *Repository) UpdateTeam(id int64, name, description string) error {
	_, err := r.db.Exec("UPDATE teams SET name = ?, description = ? WHERE id = ?", name, description, id)
	return err
}

func (r *Repository) DeleteTeam(id int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM report_submissions WHERE team_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM team_members WHERE team_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM teams WHERE id = ?", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) AddTeamMember(teamID, userID int64, role model.TeamRole, roleCode model.RoleCode) (*model.TeamMember, error) {
	result, err := r.db.Exec(
		"INSERT INTO team_members (team_id, user_id, role, role_code) VALUES (?, ?, ?, ?)",
		teamID, userID, role, roleCode,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	user, _ := r.GetUserByID(userID)
	tm := &model.TeamMember{
		ID:       id,
		TeamID:   teamID,
		UserID:   userID,
		Role:     role,
		RoleCode: roleCode,
		JoinedAt: time.Now(),
	}
	if user != nil {
		tm.UserName = user.Name
		tm.UserEmail = user.Email
	}
	return tm, nil
}

func (r *Repository) GetTeamMembers(teamID int64) ([]model.TeamMember, error) {
	rows, err := r.db.Query(
		`SELECT tm.id, tm.team_id, tm.user_id, tm.role, tm.role_code, tm.joined_at, u.name, u.email
		 FROM team_members tm
		 JOIN users u ON tm.user_id = u.id
		 WHERE tm.team_id = ?
		 ORDER BY tm.joined_at`, teamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.TeamMember
	for rows.Next() {
		var m model.TeamMember
		if err := rows.Scan(&m.ID, &m.TeamID, &m.UserID, &m.Role, &m.RoleCode, &m.JoinedAt, &m.UserName, &m.UserEmail); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *Repository) GetTeamMember(teamID, userID int64) (*model.TeamMember, error) {
	var m model.TeamMember
	err := r.db.QueryRow(
		`SELECT tm.id, tm.team_id, tm.user_id, tm.role, tm.role_code, tm.joined_at, u.name, u.email
		 FROM team_members tm
		 JOIN users u ON tm.user_id = u.id
		 WHERE tm.team_id = ? AND tm.user_id = ?`, teamID, userID,
	).Scan(&m.ID, &m.TeamID, &m.UserID, &m.Role, &m.RoleCode, &m.JoinedAt, &m.UserName, &m.UserEmail)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repository) UpdateTeamMember(id int64, role model.TeamRole, roleCode model.RoleCode, name string) error {
	_, err := r.db.Exec("UPDATE team_members SET role = ?, role_code = ? WHERE id = ?", role, roleCode, id)
	if err != nil {
		return err
	}
	if name != "" {
		_, err = r.db.Exec("UPDATE users SET name = ? WHERE id = (SELECT user_id FROM team_members WHERE id = ?)", name, id)
	}
	return err
}

func (r *Repository) RemoveTeamMember(id int64) error {
	_, err := r.db.Exec("DELETE FROM team_members WHERE id = ?", id)
	return err
}

func (r *Repository) SubmitReport(reportID, teamID, userID int64) (*model.ReportSubmission, error) {
	result, err := r.db.Exec(
		`INSERT INTO report_submissions (report_id, team_id, user_id, status, submitted_at)
		 VALUES (?, ?, ?, 'submitted', CURRENT_TIMESTAMP)
		 ON CONFLICT(report_id, team_id) DO UPDATE SET status = 'submitted', submitted_at = CURRENT_TIMESTAMP`,
		reportID, teamID, userID,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &model.ReportSubmission{
		ID:          id,
		ReportID:    reportID,
		TeamID:      teamID,
		UserID:      userID,
		Status:      "submitted",
		SubmittedAt: &now,
		CreatedAt:   now,
	}, nil
}

func (r *Repository) UnsubmitReport(reportID, teamID int64) error {
	_, err := r.db.Exec("DELETE FROM report_submissions WHERE report_id = ? AND team_id = ?", reportID, teamID)
	return err
}

func (r *Repository) GetSubmissions(teamID int64, reportDate string) ([]model.ReportSubmission, error) {
	mon, sun := weekRange(reportDate)
	rows, err := r.db.Query(
		`SELECT rs.id, rs.report_id, rs.team_id, rs.user_id, rs.status, rs.submitted_at, rs.created_at, u.name, u.email
		 FROM report_submissions rs
		 JOIN users u ON rs.user_id = u.id
		 JOIN reports r ON rs.report_id = r.id
		 WHERE rs.team_id = ? AND r.report_date BETWEEN ? AND ?
		 ORDER BY u.name`, teamID, mon, sun,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []model.ReportSubmission
	for rows.Next() {
		var s model.ReportSubmission
		if err := rows.Scan(&s.ID, &s.ReportID, &s.TeamID, &s.UserID, &s.Status, &s.SubmittedAt, &s.CreatedAt, &s.UserName, &s.UserEmail); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (r *Repository) GetSubmissionByUser(teamID, userID int64, reportDate string) (*model.ReportSubmission, error) {
	var s model.ReportSubmission
	mon, sun := weekRange(reportDate)
	err := r.db.QueryRow(
		`SELECT rs.id, rs.report_id, rs.team_id, rs.user_id, rs.status, rs.submitted_at, rs.created_at
		 FROM report_submissions rs
		 JOIN reports r ON rs.report_id = r.id
		 WHERE rs.team_id = ? AND rs.user_id = ? AND r.report_date BETWEEN ? AND ?`, teamID, userID, mon, sun,
	).Scan(&s.ID, &s.ReportID, &s.TeamID, &s.UserID, &s.Status, &s.SubmittedAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) GetSubmissionsByUser(teamID, userID int64) ([]model.ReportSubmission, error) {
	rows, err := r.db.Query(
		`SELECT rs.id, rs.report_id, rs.team_id, rs.user_id, rs.status, rs.submitted_at, rs.created_at, r.report_date
		 FROM report_submissions rs
		 JOIN reports r ON rs.report_id = r.id
		 WHERE rs.team_id = ? AND rs.user_id = ?
		 ORDER BY r.report_date DESC`, teamID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ReportSubmission
	for rows.Next() {
		var s model.ReportSubmission
		if err := rows.Scan(&s.ID, &s.ReportID, &s.TeamID, &s.UserID, &s.Status, &s.SubmittedAt, &s.CreatedAt, &s.ReportDate); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *Repository) GetReportByID(id int64) (*model.Report, error) {
	var report model.Report
	var thisWeekJSON, nextWeekJSON string

	err := r.db.QueryRow(
		"SELECT id, team_name, author_name, report_date, COALESCE(this_week, '[]'), COALESCE(next_week, '[]'), COALESCE(issues, ''), COALESCE(notes, ''), COALESCE(next_issues, ''), COALESCE(next_notes, ''), COALESCE(template_id, 0), created_at FROM reports WHERE id = ?",
		id,
	).Scan(&report.ID, &report.TeamName, &report.AuthorName, &report.ReportDate, &thisWeekJSON, &nextWeekJSON, &report.Issues, &report.Notes, &report.NextIssues, &report.NextNotes, &report.TemplateID, &report.CreatedAt)

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

func (r *Repository) UpdateReportByID(id int64, req model.CreateReportRequest) error {
	thisWeekJSON, err := json.Marshal(req.ThisWeek)
	if err != nil {
		return err
	}
	nextWeekJSON, err := json.Marshal(req.NextWeek)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(
		`UPDATE reports SET team_name=?, author_name=?, report_date=?, this_week=?, next_week=?,
		 issues=?, notes=?, next_issues=?, next_notes=?, template_id=? WHERE id=?`,
		req.TeamName, req.AuthorName, req.ReportDate, string(thisWeekJSON), string(nextWeekJSON),
		req.Issues, req.Notes, req.NextIssues, req.NextNotes, req.TemplateID, id,
	)
	return err
}

func (r *Repository) CreateTeamProject(teamID int64, name, client string) (*model.TeamProject, error) {
	result, err := r.db.Exec(
		"INSERT INTO team_projects (team_id, name, client) VALUES (?, ?, ?)",
		teamID, name, client,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &model.TeamProject{
		ID:        id,
		TeamID:    teamID,
		Name:      name,
		Client:    client,
		IsActive:  true,
		CreatedAt: time.Now(),
	}, nil
}

func (r *Repository) GetTeamProjects(teamID int64, activeOnly bool) ([]model.TeamProject, error) {
	query := "SELECT id, team_id, name, client, is_active, sort_order, created_at FROM team_projects WHERE team_id = ?"
	if activeOnly {
		query += " AND is_active = 1"
	}
	query += " ORDER BY sort_order, name"

	rows, err := r.db.Query(query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.TeamProject
	for rows.Next() {
		var p model.TeamProject
		var isActive int
		var client sql.NullString
		if err := rows.Scan(&p.ID, &p.TeamID, &p.Name, &client, &isActive, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Client = client.String
		p.IsActive = isActive == 1
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *Repository) GetTeamProject(id int64) (*model.TeamProject, error) {
	var p model.TeamProject
	var isActive int
	var client sql.NullString
	err := r.db.QueryRow(
		"SELECT id, team_id, name, client, is_active, sort_order, created_at FROM team_projects WHERE id = ?", id,
	).Scan(&p.ID, &p.TeamID, &p.Name, &client, &isActive, &p.SortOrder, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.Client = client.String
	p.IsActive = isActive == 1
	return &p, nil
}

func (r *Repository) UpdateTeamProject(id int64, name, client string, isActive *bool) error {
	if isActive != nil {
		active := 0
		if *isActive {
			active = 1
		}
		_, err := r.db.Exec("UPDATE team_projects SET name = ?, client = ?, is_active = ? WHERE id = ?", name, client, active, id)
		return err
	}
	_, err := r.db.Exec("UPDATE team_projects SET name = ?, client = ? WHERE id = ?", name, client, id)
	return err
}

func (r *Repository) DeleteTeamProject(id int64) error {
	_, err := r.db.Exec("DELETE FROM team_projects WHERE id = ?", id)
	return err
}

func (r *Repository) GetOrCreateTeamProject(teamID int64, name string) (*model.TeamProject, error) {
	var p model.TeamProject
	var isActive int
	var client sql.NullString
	err := r.db.QueryRow(
		"SELECT id, team_id, name, client, is_active, sort_order, created_at FROM team_projects WHERE team_id = ? AND name = ?",
		teamID, name,
	).Scan(&p.ID, &p.TeamID, &p.Name, &client, &isActive, &p.SortOrder, &p.CreatedAt)
	if err == nil {
		p.Client = client.String
		p.IsActive = isActive == 1
		return &p, nil
	}
	return r.CreateTeamProject(teamID, name, "")
}

func (r *Repository) ReorderTeamProjects(teamID int64, ids []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE team_projects SET sort_order = ? WHERE id = ? AND team_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, id := range ids {
		if _, err := stmt.Exec(i, id, teamID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) SaveConsolidatedEdit(teamID int64, reportDate, data string, updatedBy int64) error {
	_, err := r.db.Exec(
		`INSERT INTO consolidated_edits (team_id, report_date, data, updated_by, updated_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(team_id, report_date) DO UPDATE SET data = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP`,
		teamID, reportDate, data, updatedBy, data, updatedBy,
	)
	return err
}

func (r *Repository) GetConsolidatedEdit(teamID int64, reportDate string) (*model.ConsolidatedEdit, error) {
	var e model.ConsolidatedEdit
	err := r.db.QueryRow(
		"SELECT id, team_id, report_date, data, updated_by, updated_at FROM consolidated_edits WHERE team_id = ? AND report_date = ?",
		teamID, reportDate,
	).Scan(&e.ID, &e.TeamID, &e.ReportDate, &e.Data, &e.UpdatedBy, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *Repository) DeleteConsolidatedEdit(teamID int64, reportDate string) error {
	_, err := r.db.Exec("DELETE FROM consolidated_edits WHERE team_id = ? AND report_date = ?", teamID, reportDate)
	return err
}

// --- ConsolidationRule methods ---

func (r *Repository) CreateConsolidationRule(teamID int64, req model.CreateConsolidationRuleRequest) (*model.ConsolidationRule, error) {
	var nextOrder int
	r.db.QueryRow("SELECT COALESCE(MAX(sort_order)+1, 0) FROM consolidation_rules WHERE team_id = ?", teamID).Scan(&nextOrder)

	result, err := r.db.Exec(
		`INSERT INTO consolidation_rules (team_id, rule_type, pattern, replacement, scope_title, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		teamID, string(req.RuleType), req.Pattern, req.Replacement, req.ScopeTitle, nextOrder,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &model.ConsolidationRule{
		ID:          id,
		TeamID:      teamID,
		RuleType:    req.RuleType,
		Pattern:     req.Pattern,
		Replacement: req.Replacement,
		ScopeTitle:  req.ScopeTitle,
		SortOrder:   nextOrder,
		CreatedAt:   time.Now(),
	}, nil
}

func (r *Repository) GetConsolidationRules(teamID int64) ([]model.ConsolidationRule, error) {
	rows, err := r.db.Query(
		`SELECT id, team_id, rule_type, pattern, replacement, scope_title, sort_order, created_at
		 FROM consolidation_rules WHERE team_id = ? ORDER BY sort_order, id`,
		teamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := []model.ConsolidationRule{}
	for rows.Next() {
		var c model.ConsolidationRule
		var ruleType string
		if err := rows.Scan(&c.ID, &c.TeamID, &ruleType, &c.Pattern, &c.Replacement, &c.ScopeTitle, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.RuleType = model.ConsolidationRuleType(ruleType)
		rules = append(rules, c)
	}
	return rules, rows.Err()
}

func (r *Repository) GetConsolidationRule(id int64) (*model.ConsolidationRule, error) {
	var c model.ConsolidationRule
	var ruleType string
	err := r.db.QueryRow(
		`SELECT id, team_id, rule_type, pattern, replacement, scope_title, sort_order, created_at
		 FROM consolidation_rules WHERE id = ?`, id,
	).Scan(&c.ID, &c.TeamID, &ruleType, &c.Pattern, &c.Replacement, &c.ScopeTitle, &c.SortOrder, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.RuleType = model.ConsolidationRuleType(ruleType)
	return &c, nil
}

func (r *Repository) UpdateConsolidationRule(id int64, req model.UpdateConsolidationRuleRequest) error {
	_, err := r.db.Exec(
		`UPDATE consolidation_rules SET rule_type = ?, pattern = ?, replacement = ?, scope_title = ? WHERE id = ?`,
		string(req.RuleType), req.Pattern, req.Replacement, req.ScopeTitle, id,
	)
	return err
}

func (r *Repository) DeleteConsolidationRule(id int64) error {
	_, err := r.db.Exec("DELETE FROM consolidation_rules WHERE id = ?", id)
	return err
}

func (r *Repository) ReorderConsolidationRules(teamID int64, ids []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE consolidation_rules SET sort_order = ? WHERE id = ? AND team_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, id := range ids {
		if _, err := stmt.Exec(i, id, teamID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// --- SiteProject / SiteProjectAuthor ---

func (r *Repository) loadSiteProjectAuthors(siteProjectID int64) ([]model.SiteProjectAuthor, error) {
	rows, err := r.db.Query(
		`SELECT spa.site_project_id, spa.user_id, u.name, u.email, spa.sort_order
		 FROM site_project_authors spa
		 JOIN users u ON spa.user_id = u.id
		 WHERE spa.site_project_id = ?
		 ORDER BY spa.sort_order, u.name`, siteProjectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	authors := []model.SiteProjectAuthor{}
	for rows.Next() {
		var a model.SiteProjectAuthor
		if err := rows.Scan(&a.SiteProjectID, &a.UserID, &a.UserName, &a.UserEmail, &a.SortOrder); err != nil {
			return nil, err
		}
		authors = append(authors, a)
	}
	return authors, rows.Err()
}

func (r *Repository) replaceSiteProjectAuthors(tx *sql.Tx, siteProjectID int64, userIDs []int64) error {
	if _, err := tx.Exec("DELETE FROM site_project_authors WHERE site_project_id = ?", siteProjectID); err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return nil
	}
	stmt, err := tx.Prepare("INSERT INTO site_project_authors (site_project_id, user_id, sort_order) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, uid := range userIDs {
		if _, err := stmt.Exec(siteProjectID, uid, i); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) CreateSiteProject(teamID int64, req model.CreateSiteProjectRequest) (*model.SiteProject, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var nextOrder int
	tx.QueryRow("SELECT COALESCE(MAX(sort_order)+1, 0) FROM site_projects WHERE team_id = ?", teamID).Scan(&nextOrder)

	result, err := tx.Exec(
		`INSERT INTO site_projects (team_id, project_name, client_name, is_active, sort_order)
		 VALUES (?, ?, ?, 1, ?)`,
		teamID, req.ProjectName, req.ClientName, nextOrder,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	if err := r.replaceSiteProjectAuthors(tx, id, req.AuthorIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	authors, _ := r.loadSiteProjectAuthors(id)
	return &model.SiteProject{
		ID:          id,
		TeamID:      teamID,
		ProjectName: req.ProjectName,
		ClientName:  req.ClientName,
		IsActive:    true,
		SortOrder:   nextOrder,
		CreatedAt:   time.Now(),
		Authors:     authors,
	}, nil
}

func (r *Repository) GetSiteProjects(teamID int64, activeOnly bool) ([]model.SiteProject, error) {
	query := "SELECT id, team_id, project_name, client_name, is_active, sort_order, created_at FROM site_projects WHERE team_id = ?"
	if activeOnly {
		query += " AND is_active = 1"
	}
	query += " ORDER BY sort_order, project_name"

	rows, err := r.db.Query(query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := []model.SiteProject{}
	for rows.Next() {
		var p model.SiteProject
		var isActive int
		if err := rows.Scan(&p.ID, &p.TeamID, &p.ProjectName, &p.ClientName, &isActive, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.IsActive = isActive == 1
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range projects {
		authors, err := r.loadSiteProjectAuthors(projects[i].ID)
		if err != nil {
			return nil, err
		}
		projects[i].Authors = authors
	}
	return projects, nil
}

func (r *Repository) GetSiteProject(id int64) (*model.SiteProject, error) {
	var p model.SiteProject
	var isActive int
	err := r.db.QueryRow(
		"SELECT id, team_id, project_name, client_name, is_active, sort_order, created_at FROM site_projects WHERE id = ?", id,
	).Scan(&p.ID, &p.TeamID, &p.ProjectName, &p.ClientName, &isActive, &p.SortOrder, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.IsActive = isActive == 1
	authors, err := r.loadSiteProjectAuthors(p.ID)
	if err != nil {
		return nil, err
	}
	p.Authors = authors
	return &p, nil
}

func (r *Repository) UpdateSiteProject(id int64, req model.UpdateSiteProjectRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if req.IsActive != nil {
		active := 0
		if *req.IsActive {
			active = 1
		}
		if _, err := tx.Exec(
			"UPDATE site_projects SET project_name = ?, client_name = ?, is_active = ? WHERE id = ?",
			req.ProjectName, req.ClientName, active, id,
		); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(
			"UPDATE site_projects SET project_name = ?, client_name = ? WHERE id = ?",
			req.ProjectName, req.ClientName, id,
		); err != nil {
			return err
		}
	}

	if req.AuthorIDs != nil {
		if err := r.replaceSiteProjectAuthors(tx, id, req.AuthorIDs); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) DeleteSiteProject(id int64) error {
	_, err := r.db.Exec("DELETE FROM site_projects WHERE id = ?", id)
	return err
}

func (r *Repository) GetSiteProjectsByAuthor(teamID, userID int64) ([]model.SiteProject, error) {
	rows, err := r.db.Query(
		`SELECT sp.id, sp.team_id, sp.project_name, sp.client_name, sp.is_active, sp.sort_order, sp.created_at
		 FROM site_projects sp
		 JOIN site_project_authors spa ON spa.site_project_id = sp.id
		 WHERE sp.team_id = ? AND spa.user_id = ? AND sp.is_active = 1
		 ORDER BY sp.sort_order, sp.project_name`, teamID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := []model.SiteProject{}
	for rows.Next() {
		var p model.SiteProject
		var isActive int
		if err := rows.Scan(&p.ID, &p.TeamID, &p.ProjectName, &p.ClientName, &isActive, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.IsActive = isActive == 1
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range projects {
		authors, err := r.loadSiteProjectAuthors(projects[i].ID)
		if err != nil {
			return nil, err
		}
		projects[i].Authors = authors
	}
	return projects, nil
}

func (r *Repository) IsSiteProjectAuthor(siteProjectID, userID int64) (bool, error) {
	var n int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM site_project_authors WHERE site_project_id = ? AND user_id = ?",
		siteProjectID, userID,
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// --- SiteReport ---

func (r *Repository) SaveSiteReport(teamID, userID int64, req model.SaveSiteReportRequest) (*model.SiteReport, error) {
	thisWeekJSON, err := json.Marshal(req.ThisWeek)
	if err != nil {
		return nil, err
	}
	nextWeekJSON, err := json.Marshal(req.NextWeek)
	if err != nil {
		return nil, err
	}

	project, err := r.GetSiteProject(req.SiteProjectID)
	if err != nil {
		return nil, err
	}
	authorNames := make([]string, 0, len(project.Authors))
	for _, a := range project.Authors {
		authorNames = append(authorNames, a.UserName)
	}
	authorNamesJSON, err := json.Marshal(authorNames)
	if err != nil {
		return nil, err
	}

	mon, _ := weekRange(req.ReportDate)

	_, err = r.db.Exec(
		`INSERT INTO site_reports (team_id, site_project_id, author_user_id, author_names, project_name,
			report_date, report_date_text, this_week, next_week, notes, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(site_project_id, report_date) DO UPDATE SET
			author_user_id = excluded.author_user_id,
			author_names = excluded.author_names,
			project_name = excluded.project_name,
			report_date_text = excluded.report_date_text,
			this_week = excluded.this_week,
			next_week = excluded.next_week,
			notes = excluded.notes,
			updated_at = CURRENT_TIMESTAMP`,
		teamID, req.SiteProjectID, userID, string(authorNamesJSON), project.ProjectName,
		mon, req.ReportDateText, string(thisWeekJSON), string(nextWeekJSON), req.Notes,
	)
	if err != nil {
		return nil, err
	}

	return r.GetSiteReportByProjectAndDate(req.SiteProjectID, mon)
}

func (r *Repository) scanSiteReport(scan func(...any) error) (*model.SiteReport, error) {
	var sr model.SiteReport
	var authorNamesJSON, thisWeekJSON, nextWeekJSON string
	if err := scan(&sr.ID, &sr.TeamID, &sr.SiteProjectID, &sr.AuthorUserID, &authorNamesJSON,
		&sr.ProjectName, &sr.ReportDate, &sr.ReportDateText, &thisWeekJSON, &nextWeekJSON,
		&sr.Notes, &sr.CreatedAt, &sr.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(authorNamesJSON), &sr.AuthorNames); err != nil {
		sr.AuthorNames = []string{}
	}
	if err := json.Unmarshal([]byte(thisWeekJSON), &sr.ThisWeek); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(nextWeekJSON), &sr.NextWeek); err != nil {
		return nil, err
	}
	if sr.ThisWeek == nil {
		sr.ThisWeek = []model.SiteTask{}
	}
	if sr.NextWeek == nil {
		sr.NextWeek = []model.SiteNextTask{}
	}
	return &sr, nil
}

const siteReportColumns = `id, team_id, site_project_id, author_user_id, author_names,
	project_name, report_date, report_date_text, this_week, next_week, notes, created_at, updated_at`

// JOIN 쿼리에서 site_project_id 컬럼 모호성을 피하기 위한 sr. 프리픽스 버전.
const siteReportColumnsSR = `sr.id, sr.team_id, sr.site_project_id, sr.author_user_id, sr.author_names,
	sr.project_name, sr.report_date, sr.report_date_text, sr.this_week, sr.next_week, sr.notes, sr.created_at, sr.updated_at`

func (r *Repository) GetSiteReport(id int64) (*model.SiteReport, error) {
	row := r.db.QueryRow(`SELECT `+siteReportColumns+` FROM site_reports WHERE id = ?`, id)
	return r.scanSiteReport(row.Scan)
}

func (r *Repository) GetSiteReportByProjectAndDate(siteProjectID int64, reportDate string) (*model.SiteReport, error) {
	mon, sun := weekRange(reportDate)
	row := r.db.QueryRow(
		`SELECT `+siteReportColumns+` FROM site_reports
		 WHERE site_project_id = ? AND report_date BETWEEN ? AND ?`,
		siteProjectID, mon, sun,
	)
	return r.scanSiteReport(row.Scan)
}

// 해당 주차에 사이트 보고서가 있는 SiteProject의 모든 author user_id를 DISTINCT로 반환.
// 한 SiteProject에 author가 여러 명 등록되면 그 중 한 명만 대표 작성해도 등록된 모두를 제출자로 카운트.
func (r *Repository) GetSiteSubmittersByTeamAndDate(teamID int64, reportDate string) ([]int64, error) {
	mon, sun := weekRange(reportDate)
	rows, err := r.db.Query(
		`SELECT DISTINCT spa.user_id
		 FROM site_reports sr
		 JOIN site_project_authors spa ON spa.site_project_id = sr.site_project_id
		 WHERE sr.team_id = ? AND sr.report_date BETWEEN ? AND ?`,
		teamID, mon, sun,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []int64{}
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		results = append(results, uid)
	}
	return results, rows.Err()
}

func (r *Repository) GetSiteReportsByUser(teamID, userID int64) ([]model.SiteReport, error) {
	rows, err := r.db.Query(
		`SELECT `+siteReportColumnsSR+` FROM site_reports sr
		 JOIN site_project_authors spa ON spa.site_project_id = sr.site_project_id
		 WHERE sr.team_id = ? AND spa.user_id = ?
		 ORDER BY sr.report_date DESC, sr.site_project_id`,
		teamID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []model.SiteReport{}
	for rows.Next() {
		sr, err := r.scanSiteReport(rows.Scan)
		if err != nil {
			return nil, err
		}
		results = append(results, *sr)
	}
	return results, rows.Err()
}

func (r *Repository) GetSiteReportsByTeamAndDate(teamID int64, reportDate string) ([]model.SiteReport, error) {
	mon, sun := weekRange(reportDate)
	rows, err := r.db.Query(
		`SELECT `+siteReportColumns+` FROM site_reports
		 WHERE team_id = ? AND report_date BETWEEN ? AND ?
		 ORDER BY site_project_id`,
		teamID, mon, sun,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []model.SiteReport{}
	for rows.Next() {
		sr, err := r.scanSiteReport(rows.Scan)
		if err != nil {
			return nil, err
		}
		results = append(results, *sr)
	}
	return results, rows.Err()
}
