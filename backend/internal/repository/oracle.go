//go:build oracle

package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jiin/weeky/internal/model"
	go_ora "github.com/sijms/go-ora/v2"
)

type OracleRepository struct {
	db *sql.DB
}

var _ IRepository = (*OracleRepository)(nil)

func NewOracle(dsn string) (*OracleRepository, error) {
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open oracle connection: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping oracle: %w", err)
	}

	if err := runOracleMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("oracle migrations failed: %w", err)
	}

	return &OracleRepository{db: db}, nil
}

type oracleMigration struct {
	version int
	sqls    []string
}

var oracleMigrations = []oracleMigration{
	{
		version: 1,
		sqls: []string{
			`CREATE TABLE users (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				email VARCHAR2(255) NOT NULL UNIQUE,
				password_hash VARCHAR2(255) NOT NULL,
				name VARCHAR2(255) NOT NULL,
				is_admin NUMBER(1) DEFAULT 0 NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE invite_codes (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				code VARCHAR2(255) NOT NULL UNIQUE,
				created_by NUMBER NOT NULL REFERENCES users(id),
				used_by NUMBER REFERENCES users(id),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				used_at TIMESTAMP
			)`,
			`CREATE TABLE templates (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				name VARCHAR2(255) NOT NULL,
				style CLOB DEFAULT '{}',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE reports (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				user_id NUMBER DEFAULT 0 NOT NULL,
				team_name VARCHAR2(255) NOT NULL,
				author_name VARCHAR2(255) NOT NULL,
				report_date VARCHAR2(10) NOT NULL,
				this_week CLOB DEFAULT '[]',
				next_week CLOB DEFAULT '[]',
				issues VARCHAR2(4000) DEFAULT '',
				template_id NUMBER,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE configs (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				user_id NUMBER DEFAULT 0 NOT NULL,
				key VARCHAR2(255) NOT NULL,
				value VARCHAR2(4000) DEFAULT '',
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(user_id, key)
			)`,
		},
	},
	{
		version: 2,
		sqls: []string{
			`ALTER TABLE reports ADD notes VARCHAR2(4000) DEFAULT ''`,
		},
	},
	{
		version: 3,
		sqls: []string{
			`ALTER TABLE reports ADD next_issues VARCHAR2(4000) DEFAULT ''`,
			`ALTER TABLE reports ADD next_notes VARCHAR2(4000) DEFAULT ''`,
		},
	},
	{
		version: 4,
		sqls: []string{
			`CREATE TABLE teams (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				name VARCHAR2(255) NOT NULL,
				description VARCHAR2(4000) DEFAULT '',
				created_by NUMBER NOT NULL REFERENCES users(id),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE team_members (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				team_id NUMBER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
				user_id NUMBER NOT NULL REFERENCES users(id),
				role VARCHAR2(50) DEFAULT 'member' NOT NULL,
				role_code VARCHAR2(10) DEFAULT 'S' NOT NULL,
				joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(team_id, user_id)
			)`,
			`CREATE TABLE report_submissions (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				report_id NUMBER NOT NULL REFERENCES reports(id),
				team_id NUMBER NOT NULL REFERENCES teams(id),
				user_id NUMBER NOT NULL REFERENCES users(id),
				status VARCHAR2(50) DEFAULT 'draft' NOT NULL,
				submitted_at TIMESTAMP,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(report_id, team_id)
			)`,
		},
	},
	{
		version: 5,
		sqls: []string{
			`CREATE TABLE team_projects (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				team_id NUMBER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
				name VARCHAR2(255) NOT NULL,
				client VARCHAR2(255) DEFAULT '',
				is_active NUMBER(1) DEFAULT 1,
				sort_order NUMBER DEFAULT 0,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				CONSTRAINT uq_team_projects_name UNIQUE (team_id, name)
			)`,
		},
	},
	{
		version: 6,
		sqls: []string{
			`ALTER TABLE team_projects DROP CONSTRAINT uq_team_projects_name`,
			`ALTER TABLE team_projects ADD CONSTRAINT uq_team_projects_name_client UNIQUE (team_id, name, client)`,
		},
	},
	{
		version: 7,
		sqls: []string{
			`CREATE TABLE consolidated_edits (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				team_id NUMBER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
				report_date VARCHAR2(10) NOT NULL,
				data CLOB NOT NULL,
				updated_by NUMBER NOT NULL,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(team_id, report_date)
			)`,
		},
	},
}

func runOracleMigrations(db *sql.DB) error {
	// Check if migrations_ table exists
	var cnt int
	err := db.QueryRow(`SELECT COUNT(*) FROM user_tables WHERE table_name = 'MIGRATIONS_'`).Scan(&cnt)
	if err != nil {
		return fmt.Errorf("failed to check migrations_ table: %w", err)
	}

	if cnt == 0 {
		_, err := db.Exec(`CREATE TABLE migrations_ (
			version NUMBER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
		if err != nil {
			return fmt.Errorf("failed to create migrations_ table: %w", err)
		}
	}

	for _, m := range oracleMigrations {
		var exists int
		if err := db.QueryRow("SELECT COUNT(*) FROM migrations_ WHERE version = :1", m.version).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check migration v%d: %w", m.version, err)
		}
		if exists > 0 {
			continue
		}

		for _, s := range m.sqls {
			if _, err := db.Exec(s); err != nil {
				// Skip "column already exists" or "table already exists" errors for idempotency
				errStr := err.Error()
				if strings.Contains(errStr, "ORA-01430") || // column already exists
					strings.Contains(errStr, "ORA-00955") || // name already used
					strings.Contains(errStr, "ORA-02261") { // unique/primary key already exists
					continue
				}
				return fmt.Errorf("oracle migration v%d failed: %w\nSQL: %s", m.version, err, s)
			}
		}

		if _, err := db.Exec("INSERT INTO migrations_ (version) VALUES (:1)", m.version); err != nil {
			return fmt.Errorf("failed to record oracle migration v%d: %w", m.version, err)
		}
	}

	return nil
}

func (r *OracleRepository) Close() error {
	return r.db.Close()
}

// --- User methods ---

func (r *OracleRepository) CreateUser(email, passwordHash, name string, isAdmin bool) (*model.User, error) {
	adminInt := 0
	if isAdmin {
		adminInt = 1
	}
	var id int64
	_, err := r.db.Exec(
		`INSERT INTO users (email, password_hash, name, is_admin)
		 VALUES (:1, :2, :3, :4)
		 RETURNING id INTO :5`,
		email, passwordHash, name, adminInt, go_ora.Out{Dest: &id, Size: 8},
	)
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

func (r *OracleRepository) CreateFirstAdmin(email, passwordHash, name string) (*model.User, error) {
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
		return nil, nil
	}

	var id int64
	_, err = tx.Exec(
		`INSERT INTO users (email, password_hash, name, is_admin)
		 VALUES (:1, :2, :3, 1)
		 RETURNING id INTO :4`,
		email, passwordHash, name, go_ora.Out{Dest: &id, Size: 8},
	)
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

func (r *OracleRepository) GetUserByEmail(email string) (*model.User, error) {
	var u model.User
	var isAdmin int
	err := r.db.QueryRow(
		"SELECT id, email, password_hash, name, is_admin, created_at FROM users WHERE email = :1",
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &isAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.IsAdmin = isAdmin == 1
	return &u, nil
}

func (r *OracleRepository) GetUserByID(id int64) (*model.User, error) {
	var u model.User
	var isAdmin int
	err := r.db.QueryRow(
		"SELECT id, email, password_hash, name, is_admin, created_at FROM users WHERE id = :1",
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &isAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.IsAdmin = isAdmin == 1
	return &u, nil
}

func (r *OracleRepository) CountUsers() (int64, error) {
	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (r *OracleRepository) UpdateUserPassword(userID int64, passwordHash string) error {
	_, err := r.db.Exec("UPDATE users SET password_hash = :1 WHERE id = :2", passwordHash, userID)
	return err
}

func (r *OracleRepository) ReassignLegacyData(userID int64) error {
	_, err := r.db.Exec("UPDATE configs SET user_id = :1 WHERE user_id = 0", userID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec("UPDATE reports SET user_id = :1 WHERE user_id = 0", userID)
	return err
}

func (r *OracleRepository) GetAllUsers() ([]model.User, error) {
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

// --- InviteCode methods ---

func (r *OracleRepository) CreateInviteCode(code string, createdBy int64) (*model.InviteCode, error) {
	var id int64
	_, err := r.db.Exec(
		`INSERT INTO invite_codes (code, created_by)
		 VALUES (:1, :2)
		 RETURNING id INTO :3`,
		code, createdBy, go_ora.Out{Dest: &id, Size: 8},
	)
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

func (r *OracleRepository) GetInviteCodeByCode(code string) (*model.InviteCode, error) {
	var ic model.InviteCode
	err := r.db.QueryRow(
		"SELECT id, code, created_by, used_by, created_at, used_at FROM invite_codes WHERE code = :1",
		code,
	).Scan(&ic.ID, &ic.Code, &ic.CreatedBy, &ic.UsedBy, &ic.CreatedAt, &ic.UsedAt)
	if err != nil {
		return nil, err
	}
	return &ic, nil
}

func (r *OracleRepository) UseInviteCode(code string, usedBy int64) error {
	_, err := r.db.Exec(
		"UPDATE invite_codes SET used_by = :1, used_at = CURRENT_TIMESTAMP WHERE code = :2 AND used_by IS NULL",
		usedBy, code,
	)
	return err
}

func (r *OracleRepository) GetInviteCodes(createdBy int64) ([]model.InviteCode, error) {
	rows, err := r.db.Query(
		"SELECT id, code, created_by, used_by, created_at, used_at FROM invite_codes WHERE created_by = :1 ORDER BY created_at DESC",
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

// --- Template methods ---

func (r *OracleRepository) GetTemplates() ([]model.Template, error) {
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

func (r *OracleRepository) CreateTemplate(name, style string) (*model.Template, error) {
	var id int64
	_, err := r.db.Exec(
		`INSERT INTO templates (name, style) VALUES (:1, :2) RETURNING id INTO :3`,
		name, style, go_ora.Out{Dest: &id, Size: 8},
	)
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

func (r *OracleRepository) UpdateTemplate(id int64, name, style string) error {
	_, err := r.db.Exec("UPDATE templates SET name = :1, style = :2 WHERE id = :3", name, style, id)
	return err
}

func (r *OracleRepository) DeleteTemplate(id int64) error {
	_, err := r.db.Exec("DELETE FROM templates WHERE id = :1", id)
	return err
}

// --- Report methods ---

func (r *OracleRepository) GetReport(id int64, userID int64) (*model.Report, error) {
	var report model.Report
	var thisWeekJSON, nextWeekJSON string

	err := r.db.QueryRow(
		`SELECT id, team_name, author_name, report_date, this_week, next_week,
		        issues, notes, next_issues, next_notes, template_id, created_at
		 FROM reports WHERE id = :1 AND user_id = :2`,
		id, userID,
	).Scan(&report.ID, &report.TeamName, &report.AuthorName, &report.ReportDate,
		&thisWeekJSON, &nextWeekJSON, &report.Issues, &report.Notes,
		&report.NextIssues, &report.NextNotes, &report.TemplateID, &report.CreatedAt)
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

func (r *OracleRepository) CreateReport(req model.CreateReportRequest, userID int64) (*model.Report, error) {
	thisWeekJSON, err := json.Marshal(req.ThisWeek)
	if err != nil {
		return nil, err
	}
	nextWeekJSON, err := json.Marshal(req.NextWeek)
	if err != nil {
		return nil, err
	}

	var id int64
	_, err = r.db.Exec(
		`INSERT INTO reports (user_id, team_name, author_name, report_date, this_week, next_week,
		                      issues, notes, next_issues, next_notes, template_id)
		 VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11)
		 RETURNING id INTO :12`,
		userID, req.TeamName, req.AuthorName, req.ReportDate,
		string(thisWeekJSON), string(nextWeekJSON),
		req.Issues, req.Notes, req.NextIssues, req.NextNotes, req.TemplateID,
		go_ora.Out{Dest: &id, Size: 8},
	)
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

func (r *OracleRepository) UpdateReport(id int64, req model.CreateReportRequest, userID int64) error {
	thisWeekJSON, err := json.Marshal(req.ThisWeek)
	if err != nil {
		return err
	}
	nextWeekJSON, err := json.Marshal(req.NextWeek)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(
		`UPDATE reports SET team_name=:1, author_name=:2, report_date=:3, this_week=:4, next_week=:5,
		 issues=:6, notes=:7, next_issues=:8, next_notes=:9, template_id=:10 WHERE id=:11 AND user_id=:12`,
		req.TeamName, req.AuthorName, req.ReportDate, string(thisWeekJSON), string(nextWeekJSON),
		req.Issues, req.Notes, req.NextIssues, req.NextNotes, req.TemplateID, id, userID,
	)
	return err
}

func (r *OracleRepository) GetReportByDateAndUser(reportDate string, userID int64) (*model.Report, error) {
	var report model.Report
	var thisWeekJSON, nextWeekJSON string

	mon, sun := weekRange(reportDate)
	err := r.db.QueryRow(
		`SELECT id, team_name, author_name, report_date, this_week, next_week,
		        issues, notes, next_issues, next_notes, template_id, created_at
		 FROM reports
		 WHERE report_date BETWEEN :1 AND :2 AND user_id = :3
		 ORDER BY created_at DESC
		 FETCH FIRST 1 ROWS ONLY`,
		mon, sun, userID,
	).Scan(&report.ID, &report.TeamName, &report.AuthorName, &report.ReportDate,
		&thisWeekJSON, &nextWeekJSON, &report.Issues, &report.Notes,
		&report.NextIssues, &report.NextNotes, &report.TemplateID, &report.CreatedAt)
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

func (r *OracleRepository) GetReportsByUser(userID int64) ([]model.Report, error) {
	rows, err := r.db.Query(
		`SELECT id, team_name, author_name, report_date, this_week, next_week,
		        issues, notes, next_issues, next_notes, template_id, created_at
		 FROM reports WHERE user_id = :1 ORDER BY created_at DESC`,
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
		if err := rows.Scan(&report.ID, &report.TeamName, &report.AuthorName, &report.ReportDate,
			&thisWeekJSON, &nextWeekJSON, &report.Issues, &report.Notes,
			&report.NextIssues, &report.NextNotes, &report.TemplateID, &report.CreatedAt); err != nil {
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

func (r *OracleRepository) GetReportByID(id int64) (*model.Report, error) {
	var report model.Report
	var thisWeekJSON, nextWeekJSON string

	err := r.db.QueryRow(
		`SELECT id, team_name, author_name, report_date, this_week, next_week,
		        issues, notes, next_issues, next_notes, template_id, created_at
		 FROM reports WHERE id = :1`,
		id,
	).Scan(&report.ID, &report.TeamName, &report.AuthorName, &report.ReportDate,
		&thisWeekJSON, &nextWeekJSON, &report.Issues, &report.Notes,
		&report.NextIssues, &report.NextNotes, &report.TemplateID, &report.CreatedAt)
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

func (r *OracleRepository) UpdateReportByID(id int64, req model.CreateReportRequest) error {
	thisWeekJSON, err := json.Marshal(req.ThisWeek)
	if err != nil {
		return err
	}
	nextWeekJSON, err := json.Marshal(req.NextWeek)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(
		`UPDATE reports SET team_name=:1, author_name=:2, report_date=:3, this_week=:4, next_week=:5,
		 issues=:6, notes=:7, next_issues=:8, next_notes=:9, template_id=:10 WHERE id=:11`,
		req.TeamName, req.AuthorName, req.ReportDate, string(thisWeekJSON), string(nextWeekJSON),
		req.Issues, req.Notes, req.NextIssues, req.NextNotes, req.TemplateID, id,
	)
	return err
}

// --- Config methods ---

func (r *OracleRepository) GetConfigs(userID int64) ([]model.Config, error) {
	rows, err := r.db.Query(
		"SELECT id, key, value, updated_at FROM configs WHERE user_id = :1 ORDER BY key", userID,
	)
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

func (r *OracleRepository) GetConfig(key string, userID int64) (*model.Config, error) {
	var c model.Config
	err := r.db.QueryRow(
		"SELECT id, key, value, updated_at FROM configs WHERE key = :1 AND user_id = :2", key, userID,
	).Scan(&c.ID, &c.Key, &c.Value, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *OracleRepository) SetConfig(key, value string, userID int64) error {
	_, err := r.db.Exec(
		`MERGE INTO configs c
		 USING (SELECT :1 AS user_id, :2 AS key FROM DUAL) src
		 ON (c.user_id = src.user_id AND c.key = src.key)
		 WHEN MATCHED THEN UPDATE SET c.value = :3, c.updated_at = CURRENT_TIMESTAMP
		 WHEN NOT MATCHED THEN INSERT (user_id, key, value, updated_at) VALUES (:1, :2, :3, CURRENT_TIMESTAMP)`,
		userID, key, value,
	)
	return err
}

func (r *OracleRepository) DeleteConfig(key string, userID int64) error {
	_, err := r.db.Exec("DELETE FROM configs WHERE key = :1 AND user_id = :2", key, userID)
	return err
}

// --- Team methods ---

func (r *OracleRepository) CreateTeam(name, description string, createdBy int64) (*model.Team, error) {
	var id int64
	_, err := r.db.Exec(
		`INSERT INTO teams (name, description, created_by)
		 VALUES (:1, :2, :3)
		 RETURNING id INTO :4`,
		name, description, createdBy, go_ora.Out{Dest: &id, Size: 8},
	)
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

func (r *OracleRepository) GetTeam(id int64) (*model.Team, error) {
	var t model.Team
	err := r.db.QueryRow(
		"SELECT id, name, description, created_by, created_at FROM teams WHERE id = :1", id,
	).Scan(&t.ID, &t.Name, &t.Description, &t.CreatedBy, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *OracleRepository) GetTeamsByUser(userID int64) ([]model.Team, error) {
	rows, err := r.db.Query(
		`SELECT t.id, t.name, t.description, t.created_by, t.created_at
		 FROM teams t
		 JOIN team_members tm ON t.id = tm.team_id
		 WHERE tm.user_id = :1
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

func (r *OracleRepository) UpdateTeam(id int64, name, description string) error {
	_, err := r.db.Exec("UPDATE teams SET name = :1, description = :2 WHERE id = :3", name, description, id)
	return err
}

func (r *OracleRepository) DeleteTeam(id int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM report_submissions WHERE team_id = :1", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM team_members WHERE team_id = :1", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM teams WHERE id = :1", id); err != nil {
		return err
	}
	return tx.Commit()
}

// --- TeamMember methods ---

func (r *OracleRepository) AddTeamMember(teamID, userID int64, role model.TeamRole, roleCode model.RoleCode) (*model.TeamMember, error) {
	var id int64
	_, err := r.db.Exec(
		`INSERT INTO team_members (team_id, user_id, role, role_code)
		 VALUES (:1, :2, :3, :4)
		 RETURNING id INTO :5`,
		teamID, userID, role, roleCode, go_ora.Out{Dest: &id, Size: 8},
	)
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

func (r *OracleRepository) GetTeamMembers(teamID int64) ([]model.TeamMember, error) {
	rows, err := r.db.Query(
		`SELECT tm.id, tm.team_id, tm.user_id, tm.role, tm.role_code, tm.joined_at, u.name, u.email
		 FROM team_members tm
		 JOIN users u ON tm.user_id = u.id
		 WHERE tm.team_id = :1
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

func (r *OracleRepository) GetTeamMember(teamID, userID int64) (*model.TeamMember, error) {
	var m model.TeamMember
	err := r.db.QueryRow(
		`SELECT tm.id, tm.team_id, tm.user_id, tm.role, tm.role_code, tm.joined_at, u.name, u.email
		 FROM team_members tm
		 JOIN users u ON tm.user_id = u.id
		 WHERE tm.team_id = :1 AND tm.user_id = :2`, teamID, userID,
	).Scan(&m.ID, &m.TeamID, &m.UserID, &m.Role, &m.RoleCode, &m.JoinedAt, &m.UserName, &m.UserEmail)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *OracleRepository) UpdateTeamMember(id int64, role model.TeamRole, roleCode model.RoleCode, name string) error {
	_, err := r.db.Exec("UPDATE team_members SET role = :1, role_code = :2 WHERE id = :3", role, roleCode, id)
	if err != nil {
		return err
	}
	if name != "" {
		_, err = r.db.Exec(
			"UPDATE users SET name = :1 WHERE id = (SELECT user_id FROM team_members WHERE id = :2)", name, id,
		)
	}
	return err
}

func (r *OracleRepository) RemoveTeamMember(id int64) error {
	_, err := r.db.Exec("DELETE FROM team_members WHERE id = :1", id)
	return err
}

// --- Submission methods ---

func (r *OracleRepository) SubmitReport(reportID, teamID, userID int64) (*model.ReportSubmission, error) {
	// Use MERGE for upsert
	_, err := r.db.Exec(
		`MERGE INTO report_submissions rs
		 USING (SELECT :1 AS report_id, :2 AS team_id FROM DUAL) src
		 ON (rs.report_id = src.report_id AND rs.team_id = src.team_id)
		 WHEN MATCHED THEN UPDATE SET rs.status = 'submitted', rs.submitted_at = CURRENT_TIMESTAMP
		 WHEN NOT MATCHED THEN INSERT (report_id, team_id, user_id, status, submitted_at)
		      VALUES (:1, :2, :3, 'submitted', CURRENT_TIMESTAMP)`,
		reportID, teamID, userID,
	)
	if err != nil {
		return nil, err
	}

	// Fetch the resulting row
	var s model.ReportSubmission
	err = r.db.QueryRow(
		`SELECT id, report_id, team_id, user_id, status, submitted_at, created_at
		 FROM report_submissions WHERE report_id = :1 AND team_id = :2`,
		reportID, teamID,
	).Scan(&s.ID, &s.ReportID, &s.TeamID, &s.UserID, &s.Status, &s.SubmittedAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *OracleRepository) UnsubmitReport(reportID, teamID int64) error {
	_, err := r.db.Exec("DELETE FROM report_submissions WHERE report_id = :1 AND team_id = :2", reportID, teamID)
	return err
}

func (r *OracleRepository) GetSubmissions(teamID int64, reportDate string) ([]model.ReportSubmission, error) {
	mon, sun := weekRange(reportDate)
	rows, err := r.db.Query(
		`SELECT rs.id, rs.report_id, rs.team_id, rs.user_id, rs.status, rs.submitted_at, rs.created_at, u.name, u.email
		 FROM report_submissions rs
		 JOIN users u ON rs.user_id = u.id
		 JOIN reports r ON rs.report_id = r.id
		 WHERE rs.team_id = :1 AND r.report_date BETWEEN :2 AND :3
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

func (r *OracleRepository) GetSubmissionByUser(teamID, userID int64, reportDate string) (*model.ReportSubmission, error) {
	var s model.ReportSubmission
	mon, sun := weekRange(reportDate)
	err := r.db.QueryRow(
		`SELECT rs.id, rs.report_id, rs.team_id, rs.user_id, rs.status, rs.submitted_at, rs.created_at
		 FROM report_submissions rs
		 JOIN reports r ON rs.report_id = r.id
		 WHERE rs.team_id = :1 AND rs.user_id = :2 AND r.report_date BETWEEN :3 AND :4`,
		teamID, userID, mon, sun,
	).Scan(&s.ID, &s.ReportID, &s.TeamID, &s.UserID, &s.Status, &s.SubmittedAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *OracleRepository) GetSubmissionsByUser(teamID, userID int64) ([]model.ReportSubmission, error) {
	rows, err := r.db.Query(
		`SELECT rs.id, rs.report_id, rs.team_id, rs.user_id, rs.status, rs.submitted_at, rs.created_at, r.report_date
		 FROM report_submissions rs
		 JOIN reports r ON rs.report_id = r.id
		 WHERE rs.team_id = :1 AND rs.user_id = :2
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

// --- TeamProject methods ---

func (r *OracleRepository) CreateTeamProject(teamID int64, name, client string) (*model.TeamProject, error) {
	var id int64
	_, err := r.db.Exec(
		`INSERT INTO team_projects (team_id, name, client)
		 VALUES (:1, :2, :3)
		 RETURNING id INTO :4`,
		teamID, name, client, go_ora.Out{Dest: &id, Size: 8},
	)
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

func (r *OracleRepository) GetTeamProjects(teamID int64, activeOnly bool) ([]model.TeamProject, error) {
	query := "SELECT id, team_id, name, client, is_active, sort_order, created_at FROM team_projects WHERE team_id = :1"
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

func (r *OracleRepository) GetTeamProject(id int64) (*model.TeamProject, error) {
	var p model.TeamProject
	var isActive int
	var client sql.NullString
	err := r.db.QueryRow(
		"SELECT id, team_id, name, client, is_active, sort_order, created_at FROM team_projects WHERE id = :1", id,
	).Scan(&p.ID, &p.TeamID, &p.Name, &client, &isActive, &p.SortOrder, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.Client = client.String
	p.IsActive = isActive == 1
	return &p, nil
}

func (r *OracleRepository) UpdateTeamProject(id int64, name, client string, isActive *bool) error {
	if isActive != nil {
		active := 0
		if *isActive {
			active = 1
		}
		_, err := r.db.Exec("UPDATE team_projects SET name = :1, client = :2, is_active = :3 WHERE id = :4", name, client, active, id)
		return err
	}
	_, err := r.db.Exec("UPDATE team_projects SET name = :1, client = :2 WHERE id = :3", name, client, id)
	return err
}

func (r *OracleRepository) DeleteTeamProject(id int64) error {
	_, err := r.db.Exec("DELETE FROM team_projects WHERE id = :1", id)
	return err
}

func (r *OracleRepository) GetOrCreateTeamProject(teamID int64, name string) (*model.TeamProject, error) {
	var p model.TeamProject
	var isActive int
	var client sql.NullString
	err := r.db.QueryRow(
		"SELECT id, team_id, name, client, is_active, sort_order, created_at FROM team_projects WHERE team_id = :1 AND name = :2",
		teamID, name,
	).Scan(&p.ID, &p.TeamID, &p.Name, &client, &isActive, &p.SortOrder, &p.CreatedAt)
	if err == nil {
		p.Client = client.String
		p.IsActive = isActive == 1
		return &p, nil
	}
	return r.CreateTeamProject(teamID, name, "")
}

func (r *OracleRepository) ReorderTeamProjects(teamID int64, ids []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE team_projects SET sort_order = :1 WHERE id = :2 AND team_id = :3")
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

// --- ConsolidatedEdit methods ---

func (r *OracleRepository) SaveConsolidatedEdit(teamID int64, reportDate, data string, updatedBy int64) error {
	_, err := r.db.Exec(
		`MERGE INTO consolidated_edits ce
		 USING (SELECT :1 AS team_id, :2 AS report_date FROM DUAL) src
		 ON (ce.team_id = src.team_id AND ce.report_date = src.report_date)
		 WHEN MATCHED THEN UPDATE SET ce.data = :3, ce.updated_by = :4, ce.updated_at = CURRENT_TIMESTAMP
		 WHEN NOT MATCHED THEN INSERT (team_id, report_date, data, updated_by, updated_at)
		      VALUES (:1, :2, :3, :4, CURRENT_TIMESTAMP)`,
		teamID, reportDate, data, updatedBy,
	)
	return err
}

func (r *OracleRepository) GetConsolidatedEdit(teamID int64, reportDate string) (*model.ConsolidatedEdit, error) {
	var e model.ConsolidatedEdit
	err := r.db.QueryRow(
		"SELECT id, team_id, report_date, data, updated_by, updated_at FROM consolidated_edits WHERE team_id = :1 AND report_date = :2",
		teamID, reportDate,
	).Scan(&e.ID, &e.TeamID, &e.ReportDate, &e.Data, &e.UpdatedBy, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *OracleRepository) DeleteConsolidatedEdit(teamID int64, reportDate string) error {
	_, err := r.db.Exec("DELETE FROM consolidated_edits WHERE team_id = :1 AND report_date = :2", teamID, reportDate)
	return err
}
