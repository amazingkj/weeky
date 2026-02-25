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

// ============ User list ============

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

// ============ Report methods ============

func (r *Repository) GetReport(id int64, userID int64) (*model.Report, error) {
	var report model.Report
	var thisWeekJSON, nextWeekJSON string

	err := r.db.QueryRow(
		"SELECT id, team_name, author_name, report_date, this_week, next_week, issues, notes, next_issues, next_notes, template_id, created_at FROM reports WHERE id = ? AND user_id = ?",
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
		"SELECT id, team_name, author_name, report_date, this_week, next_week, issues, notes, next_issues, next_notes, template_id, created_at FROM reports WHERE report_date BETWEEN ? AND ? AND user_id = ? ORDER BY created_at DESC LIMIT 1",
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
		"SELECT id, team_name, author_name, report_date, this_week, next_week, issues, notes, next_issues, next_notes, template_id, created_at FROM reports WHERE user_id = ? ORDER BY created_at DESC",
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

// ============ Team methods ============

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

	tx.Exec("DELETE FROM report_submissions WHERE team_id = ?", id)
	tx.Exec("DELETE FROM team_members WHERE team_id = ?", id)
	if _, err := tx.Exec("DELETE FROM teams WHERE id = ?", id); err != nil {
		return err
	}
	return tx.Commit()
}

// ============ Team member methods ============

func (r *Repository) AddTeamMember(teamID, userID int64, role model.TeamRole, roleCode model.RoleCode) (*model.TeamMember, error) {
	result, err := r.db.Exec(
		"INSERT INTO team_members (team_id, user_id, role, role_code) VALUES (?, ?, ?, ?)",
		teamID, userID, string(role), string(roleCode),
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Fetch user info for response
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

func (r *Repository) UpdateTeamMember(id int64, role model.TeamRole, roleCode model.RoleCode) error {
	_, err := r.db.Exec("UPDATE team_members SET role = ?, role_code = ? WHERE id = ?", string(role), string(roleCode), id)
	return err
}

func (r *Repository) RemoveTeamMember(id int64) error {
	_, err := r.db.Exec("DELETE FROM team_members WHERE id = ?", id)
	return err
}

// ============ Report submission methods ============

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

// ============ Report by ID (for team leader access) ============

func (r *Repository) GetReportByID(id int64) (*model.Report, error) {
	var report model.Report
	var thisWeekJSON, nextWeekJSON string

	err := r.db.QueryRow(
		"SELECT id, team_name, author_name, report_date, this_week, next_week, issues, notes, next_issues, next_notes, template_id, created_at FROM reports WHERE id = ?",
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
