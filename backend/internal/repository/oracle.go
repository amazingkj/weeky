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
	{
		version: 8,
		sqls: []string{
			`CREATE TABLE consolidation_rules (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				team_id NUMBER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
				rule_type VARCHAR2(50) NOT NULL,
				pattern VARCHAR2(500) DEFAULT '' NOT NULL,
				replacement VARCHAR2(500) DEFAULT '' NOT NULL,
				scope_title VARCHAR2(500) DEFAULT '' NOT NULL,
				sort_order NUMBER DEFAULT 0 NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
	},
	{
		version: 9,
		sqls: []string{
			`CREATE TABLE site_projects (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				team_id NUMBER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
				project_name VARCHAR2(500) NOT NULL,
				client_name VARCHAR2(500) DEFAULT '',
				is_active NUMBER(1) DEFAULT 1 NOT NULL,
				sort_order NUMBER DEFAULT 0 NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				CONSTRAINT uq_site_projects_team_name UNIQUE (team_id, project_name)
			)`,
			`CREATE TABLE site_project_authors (
				site_project_id NUMBER NOT NULL REFERENCES site_projects(id) ON DELETE CASCADE,
				user_id NUMBER NOT NULL REFERENCES users(id),
				sort_order NUMBER DEFAULT 0 NOT NULL,
				CONSTRAINT pk_site_project_authors PRIMARY KEY (site_project_id, user_id)
			)`,
			`CREATE TABLE site_reports (
				id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				team_id NUMBER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
				site_project_id NUMBER NOT NULL REFERENCES site_projects(id) ON DELETE CASCADE,
				author_user_id NUMBER NOT NULL REFERENCES users(id),
				author_names CLOB DEFAULT '[]',
				project_name VARCHAR2(500) DEFAULT '',
				report_date VARCHAR2(10) NOT NULL,
				report_date_text VARCHAR2(100) DEFAULT '',
				this_week CLOB DEFAULT '[]',
				next_week CLOB DEFAULT '[]',
				notes CLOB DEFAULT '',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				CONSTRAINT uq_site_reports_proj_date UNIQUE (site_project_id, report_date)
			)`,
		},
	},
	{
		// Oracle은 '' = NULL이라 NOT NULL DEFAULT '' 컬럼에 빈 문자열 INSERT 시
		// ORA-01400 발생. v8에서 NOT NULL로 잡았던 텍스트 컬럼을 nullable로 풀어
		// 빈 문자열(=NULL)이 정상적으로 저장되도록 한다.
		version: 10,
		sqls: []string{
			`ALTER TABLE consolidation_rules MODIFY pattern NULL`,
			`ALTER TABLE consolidation_rules MODIFY replacement NULL`,
			`ALTER TABLE consolidation_rules MODIFY scope_title NULL`,
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

func (r *OracleRepository) UpdateUserAdmin(userID int64, isAdmin bool) error {
	val := 0
	if isAdmin {
		val = 1
	}
	_, err := r.db.Exec("UPDATE users SET is_admin = :1 WHERE id = :2", val, userID)
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

func scanReport(scanner interface{ Scan(...any) error }) (*model.Report, error) {
	var report model.Report
	var thisWeekJSON, nextWeekJSON sql.NullString
	var issues, notes, nextIssues, nextNotes sql.NullString
	var templateID sql.NullInt64

	err := scanner.Scan(&report.ID, &report.TeamName, &report.AuthorName, &report.ReportDate,
		&thisWeekJSON, &nextWeekJSON, &issues, &notes,
		&nextIssues, &nextNotes, &templateID, &report.CreatedAt)
	if err != nil {
		return nil, err
	}

	report.Issues = issues.String
	report.Notes = notes.String
	report.NextIssues = nextIssues.String
	report.NextNotes = nextNotes.String
	report.TemplateID = templateID.Int64

	twJSON := thisWeekJSON.String
	if twJSON == "" {
		twJSON = "[]"
	}
	nwJSON := nextWeekJSON.String
	if nwJSON == "" {
		nwJSON = "[]"
	}
	if err := json.Unmarshal([]byte(twJSON), &report.ThisWeek); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(nwJSON), &report.NextWeek); err != nil {
		return nil, err
	}
	return &report, nil
}

const reportColumns = `id, team_name, author_name, report_date, this_week, next_week,
		        issues, notes, next_issues, next_notes, template_id, created_at`

func (r *OracleRepository) GetReport(id int64, userID int64) (*model.Report, error) {
	return scanReport(r.db.QueryRow(
		`SELECT `+reportColumns+` FROM reports WHERE id = :1 AND user_id = :2`,
		id, userID,
	))
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
	mon, sun := weekRange(reportDate)
	return scanReport(r.db.QueryRow(
		`SELECT `+reportColumns+` FROM reports
		 WHERE report_date BETWEEN :1 AND :2 AND user_id = :3
		 ORDER BY created_at DESC
		 FETCH FIRST 1 ROWS ONLY`,
		mon, sun, userID,
	))
}

func (r *OracleRepository) GetReportsByUser(userID int64) ([]model.Report, error) {
	rows, err := r.db.Query(
		`SELECT `+reportColumns+` FROM reports WHERE user_id = :1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.Report
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, *report)
	}
	return reports, rows.Err()
}

func (r *OracleRepository) GetReportByID(id int64) (*model.Report, error) {
	return scanReport(r.db.QueryRow(
		`SELECT `+reportColumns+` FROM reports WHERE id = :1`, id,
	))
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
		 WHEN NOT MATCHED THEN INSERT (user_id, key, value, updated_at) VALUES (:4, :5, :6, CURRENT_TIMESTAMP)`,
		userID, key, value, userID, key, value,
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
		teamID, userID, string(role), string(roleCode), go_ora.Out{Dest: &id, Size: 8},
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
	_, err := r.db.Exec("UPDATE team_members SET role = :1, role_code = :2 WHERE id = :3", string(role), string(roleCode), id)
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
		      VALUES (:3, :4, :5, 'submitted', CURRENT_TIMESTAMP)`,
		reportID, teamID, reportID, teamID, userID,
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
		      VALUES (:5, :6, :7, :8, CURRENT_TIMESTAMP)`,
		teamID, reportDate, data, updatedBy, teamID, reportDate, data, updatedBy,
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

// --- ConsolidationRule methods ---

func (r *OracleRepository) CreateConsolidationRule(teamID int64, req model.CreateConsolidationRuleRequest) (*model.ConsolidationRule, error) {
	var nextOrder int
	r.db.QueryRow("SELECT COALESCE(MAX(sort_order)+1, 0) FROM consolidation_rules WHERE team_id = :1", teamID).Scan(&nextOrder)

	var id int64
	_, err := r.db.Exec(
		`INSERT INTO consolidation_rules (team_id, rule_type, pattern, replacement, scope_title, sort_order)
		 VALUES (:1, :2, :3, :4, :5, :6)
		 RETURNING id INTO :7`,
		teamID, string(req.RuleType), req.Pattern, req.Replacement, req.ScopeTitle, nextOrder,
		go_ora.Out{Dest: &id, Size: 8},
	)
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

func (r *OracleRepository) GetConsolidationRules(teamID int64) ([]model.ConsolidationRule, error) {
	rows, err := r.db.Query(
		`SELECT id, team_id, rule_type, pattern, replacement, scope_title, sort_order, created_at
		 FROM consolidation_rules WHERE team_id = :1 ORDER BY sort_order, id`,
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
		var pattern, replacement, scopeTitle sql.NullString
		if err := rows.Scan(&c.ID, &c.TeamID, &ruleType, &pattern, &replacement, &scopeTitle, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.RuleType = model.ConsolidationRuleType(ruleType)
		c.Pattern = pattern.String
		c.Replacement = replacement.String
		c.ScopeTitle = scopeTitle.String
		rules = append(rules, c)
	}
	return rules, rows.Err()
}

func (r *OracleRepository) GetConsolidationRule(id int64) (*model.ConsolidationRule, error) {
	var c model.ConsolidationRule
	var ruleType string
	var pattern, replacement, scopeTitle sql.NullString
	err := r.db.QueryRow(
		`SELECT id, team_id, rule_type, pattern, replacement, scope_title, sort_order, created_at
		 FROM consolidation_rules WHERE id = :1`, id,
	).Scan(&c.ID, &c.TeamID, &ruleType, &pattern, &replacement, &scopeTitle, &c.SortOrder, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.RuleType = model.ConsolidationRuleType(ruleType)
	c.Pattern = pattern.String
	c.Replacement = replacement.String
	c.ScopeTitle = scopeTitle.String
	return &c, nil
}

func (r *OracleRepository) UpdateConsolidationRule(id int64, req model.UpdateConsolidationRuleRequest) error {
	_, err := r.db.Exec(
		`UPDATE consolidation_rules SET rule_type = :1, pattern = :2, replacement = :3, scope_title = :4 WHERE id = :5`,
		string(req.RuleType), req.Pattern, req.Replacement, req.ScopeTitle, id,
	)
	return err
}

func (r *OracleRepository) DeleteConsolidationRule(id int64) error {
	_, err := r.db.Exec("DELETE FROM consolidation_rules WHERE id = :1", id)
	return err
}

func (r *OracleRepository) ReorderConsolidationRules(teamID int64, ids []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE consolidation_rules SET sort_order = :1 WHERE id = :2 AND team_id = :3")
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

func (r *OracleRepository) loadSiteProjectAuthors(siteProjectID int64) ([]model.SiteProjectAuthor, error) {
	rows, err := r.db.Query(
		`SELECT spa.site_project_id, spa.user_id, u.name, u.email, spa.sort_order
		 FROM site_project_authors spa
		 JOIN users u ON spa.user_id = u.id
		 WHERE spa.site_project_id = :1
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

func (r *OracleRepository) replaceSiteProjectAuthors(tx *sql.Tx, siteProjectID int64, userIDs []int64) error {
	if _, err := tx.Exec("DELETE FROM site_project_authors WHERE site_project_id = :1", siteProjectID); err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return nil
	}
	stmt, err := tx.Prepare("INSERT INTO site_project_authors (site_project_id, user_id, sort_order) VALUES (:1, :2, :3)")
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

func (r *OracleRepository) CreateSiteProject(teamID int64, req model.CreateSiteProjectRequest) (*model.SiteProject, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var nextOrder int
	tx.QueryRow("SELECT COALESCE(MAX(sort_order)+1, 0) FROM site_projects WHERE team_id = :1", teamID).Scan(&nextOrder)

	var id int64
	_, err = tx.Exec(
		`INSERT INTO site_projects (team_id, project_name, client_name, is_active, sort_order)
		 VALUES (:1, :2, :3, 1, :4)
		 RETURNING id INTO :5`,
		teamID, req.ProjectName, req.ClientName, nextOrder, go_ora.Out{Dest: &id, Size: 8},
	)
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
		ID: id, TeamID: teamID, ProjectName: req.ProjectName, ClientName: req.ClientName,
		IsActive: true, SortOrder: nextOrder, CreatedAt: time.Now(), Authors: authors,
	}, nil
}

func (r *OracleRepository) GetSiteProjects(teamID int64, activeOnly bool) ([]model.SiteProject, error) {
	query := "SELECT id, team_id, project_name, client_name, is_active, sort_order, created_at FROM site_projects WHERE team_id = :1"
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
		var clientName sql.NullString
		if err := rows.Scan(&p.ID, &p.TeamID, &p.ProjectName, &clientName, &isActive, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.ClientName = clientName.String
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

func (r *OracleRepository) GetSiteProject(id int64) (*model.SiteProject, error) {
	var p model.SiteProject
	var isActive int
	var clientName sql.NullString
	err := r.db.QueryRow(
		"SELECT id, team_id, project_name, client_name, is_active, sort_order, created_at FROM site_projects WHERE id = :1", id,
	).Scan(&p.ID, &p.TeamID, &p.ProjectName, &clientName, &isActive, &p.SortOrder, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.ClientName = clientName.String
	p.IsActive = isActive == 1
	authors, err := r.loadSiteProjectAuthors(p.ID)
	if err != nil {
		return nil, err
	}
	p.Authors = authors
	return &p, nil
}

func (r *OracleRepository) UpdateSiteProject(id int64, req model.UpdateSiteProjectRequest) error {
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
			"UPDATE site_projects SET project_name = :1, client_name = :2, is_active = :3 WHERE id = :4",
			req.ProjectName, req.ClientName, active, id,
		); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(
			"UPDATE site_projects SET project_name = :1, client_name = :2 WHERE id = :3",
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

func (r *OracleRepository) DeleteSiteProject(id int64) error {
	_, err := r.db.Exec("DELETE FROM site_projects WHERE id = :1", id)
	return err
}

func (r *OracleRepository) GetSiteProjectsByAuthor(teamID, userID int64) ([]model.SiteProject, error) {
	rows, err := r.db.Query(
		`SELECT sp.id, sp.team_id, sp.project_name, sp.client_name, sp.is_active, sp.sort_order, sp.created_at
		 FROM site_projects sp
		 JOIN site_project_authors spa ON spa.site_project_id = sp.id
		 WHERE sp.team_id = :1 AND spa.user_id = :2 AND sp.is_active = 1
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
		var clientName sql.NullString
		if err := rows.Scan(&p.ID, &p.TeamID, &p.ProjectName, &clientName, &isActive, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.ClientName = clientName.String
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

func (r *OracleRepository) IsSiteProjectAuthor(siteProjectID, userID int64) (bool, error) {
	var n int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM site_project_authors WHERE site_project_id = :1 AND user_id = :2",
		siteProjectID, userID,
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// --- SiteReport ---

func (r *OracleRepository) SaveSiteReport(teamID, userID int64, req model.SaveSiteReportRequest) (*model.SiteReport, error) {
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

	// go-ora는 string을 VARCHAR2로 바인딩 → Oracle CLOB 컬럼에 INSERT/UPDATE 시
	// ORA-00932 (expected CLOB got CHAR). go_ora.Clob 타입으로 명시 바인딩.
	authorNamesClob := go_ora.Clob{Valid: true, String: string(authorNamesJSON)}
	thisWeekClob := go_ora.Clob{Valid: true, String: string(thisWeekJSON)}
	nextWeekClob := go_ora.Clob{Valid: true, String: string(nextWeekJSON)}
	notesClob := go_ora.Clob{Valid: true, String: req.Notes}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var existingID int64
	err = tx.QueryRow(
		"SELECT id FROM site_reports WHERE site_project_id = :1 AND report_date = :2",
		req.SiteProjectID, mon,
	).Scan(&existingID)

	switch {
	case err == sql.ErrNoRows:
		_, err = tx.Exec(
			`INSERT INTO site_reports (team_id, site_project_id, author_user_id, author_names,
				project_name, report_date, report_date_text, this_week, next_week, notes, updated_at)
			 VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, CURRENT_TIMESTAMP)`,
			teamID, req.SiteProjectID, userID, authorNamesClob,
			project.ProjectName, mon, req.ReportDateText,
			thisWeekClob, nextWeekClob, notesClob,
		)
		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		_, err = tx.Exec(
			`UPDATE site_reports SET
				author_user_id = :1,
				author_names = :2,
				project_name = :3,
				report_date_text = :4,
				this_week = :5,
				next_week = :6,
				notes = :7,
				updated_at = CURRENT_TIMESTAMP
			 WHERE id = :8`,
			userID, authorNamesClob, project.ProjectName, req.ReportDateText,
			thisWeekClob, nextWeekClob, notesClob, existingID,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetSiteReportByProjectAndDate(req.SiteProjectID, mon)
}

func (r *OracleRepository) scanSiteReport(scan func(...any) error) (*model.SiteReport, error) {
	var sr model.SiteReport
	// Oracle: nullable 텍스트/CLOB 컬럼은 NULL 가능 → sql.NullString으로 받아서 String 추출
	var authorNamesNS, projectNameNS, reportDateTextNS, thisWeekNS, nextWeekNS, notesNS sql.NullString
	if err := scan(&sr.ID, &sr.TeamID, &sr.SiteProjectID, &sr.AuthorUserID, &authorNamesNS,
		&projectNameNS, &sr.ReportDate, &reportDateTextNS, &thisWeekNS, &nextWeekNS,
		&notesNS, &sr.CreatedAt, &sr.UpdatedAt); err != nil {
		return nil, err
	}
	sr.ProjectName = projectNameNS.String
	sr.ReportDateText = reportDateTextNS.String
	sr.Notes = notesNS.String

	authorNamesJSON := authorNamesNS.String
	if authorNamesJSON == "" {
		authorNamesJSON = "[]"
	}
	thisWeekJSON := thisWeekNS.String
	if thisWeekJSON == "" {
		thisWeekJSON = "[]"
	}
	nextWeekJSON := nextWeekNS.String
	if nextWeekJSON == "" {
		nextWeekJSON = "[]"
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

// Oracle은 ''=NULL이라 빈 문자열이 NULL로 저장됨.
// COALESCE(clob_col, char_literal)은 Oracle에서 ORA-00932 (CLOB/CHAR 혼용)이라 SQL에서 안 쓰고
// scanSiteReport에서 sql.NullString으로 NULL을 빈 문자열로 변환.
const oracleSiteReportColumns = `id, team_id, site_project_id, author_user_id, author_names,
	project_name, report_date, report_date_text, this_week, next_week, notes, created_at, updated_at`

// JOIN 쿼리에서 site_project_id 컬럼 모호성을 피하기 위한 sr. 프리픽스 버전.
const oracleSiteReportColumnsSR = `sr.id, sr.team_id, sr.site_project_id, sr.author_user_id, sr.author_names,
	sr.project_name, sr.report_date, sr.report_date_text, sr.this_week, sr.next_week, sr.notes, sr.created_at, sr.updated_at`

func (r *OracleRepository) GetSiteReport(id int64) (*model.SiteReport, error) {
	row := r.db.QueryRow(`SELECT `+oracleSiteReportColumns+` FROM site_reports WHERE id = :1`, id)
	return r.scanSiteReport(row.Scan)
}

func (r *OracleRepository) GetSiteReportByProjectAndDate(siteProjectID int64, reportDate string) (*model.SiteReport, error) {
	mon, sun := weekRange(reportDate)
	row := r.db.QueryRow(
		`SELECT `+oracleSiteReportColumns+` FROM site_reports
		 WHERE site_project_id = :1 AND report_date BETWEEN :2 AND :3`,
		siteProjectID, mon, sun,
	)
	return r.scanSiteReport(row.Scan)
}

// 해당 주차에 사이트 보고서가 있는 SiteProject의 모든 author user_id를 DISTINCT로 반환.
func (r *OracleRepository) GetSiteSubmittersByTeamAndDate(teamID int64, reportDate string) ([]int64, error) {
	mon, sun := weekRange(reportDate)
	rows, err := r.db.Query(
		`SELECT DISTINCT spa.user_id
		 FROM site_reports sr
		 JOIN site_project_authors spa ON spa.site_project_id = sr.site_project_id
		 WHERE sr.team_id = :1 AND sr.report_date BETWEEN :2 AND :3`,
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

func (r *OracleRepository) GetSiteReportsByUser(teamID, userID int64) ([]model.SiteReport, error) {
	rows, err := r.db.Query(
		`SELECT `+oracleSiteReportColumnsSR+` FROM site_reports sr
		 JOIN site_project_authors spa ON spa.site_project_id = sr.site_project_id
		 WHERE sr.team_id = :1 AND spa.user_id = :2
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

func (r *OracleRepository) GetSiteReportsByTeamAndDate(teamID int64, reportDate string) ([]model.SiteReport, error) {
	mon, sun := weekRange(reportDate)
	rows, err := r.db.Query(
		`SELECT `+oracleSiteReportColumns+` FROM site_reports
		 WHERE team_id = :1 AND report_date BETWEEN :2 AND :3
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
