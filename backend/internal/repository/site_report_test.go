package repository

import (
	"os"
	"testing"

	"github.com/jiin/weeky/internal/model"
)

func setupSiteTestDB(t *testing.T) (*Repository, func()) {
	t.Helper()
	tmp, err := os.CreateTemp("", "weeky_site_*.db")
	if err != nil {
		t.Fatalf("temp db: %v", err)
	}
	tmp.Close()
	repo, err := New(tmp.Name())
	if err != nil {
		os.Remove(tmp.Name())
		t.Fatalf("new repo: %v", err)
	}
	return repo, func() { repo.Close(); os.Remove(tmp.Name()) }
}

func mkUser(t *testing.T, repo *Repository, email, name string) *model.User {
	t.Helper()
	u, err := repo.CreateUser(email, "x", name, false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return u
}

func mkTeam(t *testing.T, repo *Repository, leaderID int64) *model.Team {
	t.Helper()
	team, err := repo.CreateTeam("APIM팀", "", leaderID)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if _, err := repo.AddTeamMember(team.ID, leaderID, model.TeamRoleLeader, model.RoleCodeS); err != nil {
		t.Fatalf("AddTeamMember: %v", err)
	}
	return team
}

func TestSiteProject_CRUD_WithAuthors(t *testing.T) {
	repo, cleanup := setupSiteTestDB(t)
	defer cleanup()

	leader := mkUser(t, repo, "lead@x.com", "팀장")
	a1 := mkUser(t, repo, "a@x.com", "이민구")
	a2 := mkUser(t, repo, "b@x.com", "문정현")
	team := mkTeam(t, repo, leader.ID)

	created, err := repo.CreateSiteProject(team.ID, model.CreateSiteProjectRequest{
		ProjectName: "건강보험공단",
		ClientName:  "건강보험공단",
		AuthorIDs:   []int64{a1.ID, a2.ID},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(created.Authors) != 2 {
		t.Fatalf("authors=%d, want 2", len(created.Authors))
	}
	if created.Authors[0].UserName != "이민구" || created.Authors[1].UserName != "문정현" {
		t.Errorf("author order: got %q, %q", created.Authors[0].UserName, created.Authors[1].UserName)
	}

	// IsSiteProjectAuthor 동작 확인 (allowlist의 핵심)
	ok, _ := repo.IsSiteProjectAuthor(created.ID, a1.ID)
	if !ok {
		t.Error("a1 should be author")
	}
	ok, _ = repo.IsSiteProjectAuthor(created.ID, leader.ID)
	if ok {
		t.Error("leader is NOT registered as site author — should be false")
	}

	// 작성자만 변경 (이름은 그대로)
	if err := repo.UpdateSiteProject(created.ID, model.UpdateSiteProjectRequest{
		ProjectName: created.ProjectName,
		ClientName:  created.ClientName,
		AuthorIDs:   []int64{a2.ID}, // a1 제거
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := repo.GetSiteProject(created.ID)
	if len(got.Authors) != 1 || got.Authors[0].UserID != a2.ID {
		t.Errorf("after update authors=%v", got.Authors)
	}

	// GetSiteProjectsByAuthor: a2는 보이지만 a1은 안 보임
	ps, _ := repo.GetSiteProjectsByAuthor(team.ID, a2.ID)
	if len(ps) != 1 {
		t.Errorf("a2 projects=%d, want 1", len(ps))
	}
	ps, _ = repo.GetSiteProjectsByAuthor(team.ID, a1.ID)
	if len(ps) != 0 {
		t.Errorf("a1 projects=%d, want 0 (removed)", len(ps))
	}
}

func TestSiteReport_SaveAndRoundTrip(t *testing.T) {
	repo, cleanup := setupSiteTestDB(t)
	defer cleanup()

	leader := mkUser(t, repo, "lead@x.com", "팀장")
	a1 := mkUser(t, repo, "a@x.com", "이민구")
	a2 := mkUser(t, repo, "b@x.com", "문정현")
	team := mkTeam(t, repo, leader.ID)
	proj, _ := repo.CreateSiteProject(team.ID, model.CreateSiteProjectRequest{
		ProjectName: "건강보험공단",
		AuthorIDs:   []int64{a1.ID, a2.ID},
	})

	req := model.SaveSiteReportRequest{
		SiteProjectID:  proj.ID,
		ReportDate:     "2026-04-24", // Friday → weekRange snaps to Mon 2026-04-20
		ReportDateText: "2026-04-23",
		ThisWeek: []model.SiteTask{
			{Title: "■ 한화손해보험\n<OpenAPI>\n1. DB 이관", ElapsedDays: "2M", StartDate: "03/04", DueDate: "04/30", Progress: "80%"},
		},
		NextWeek: []model.SiteNextTask{
			{Title: "■ 한화손해보험\n1. 후속 작업", StartDate: "05/04", DueDate: "05/08"},
		},
		Notes: "- 연휴: 5/01 근로자의날",
	}
	saved, err := repo.SaveSiteReport(team.ID, a1.ID, req)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if saved.AuthorUserID != a1.ID {
		t.Errorf("AuthorUserID=%d, want %d", saved.AuthorUserID, a1.ID)
	}
	// 헤더 출력용 스냅샷: 작성자 두 명 모두 들어가야 함 (실제 저장은 a1이 했어도)
	if len(saved.AuthorNames) != 2 || saved.AuthorNames[0] != "이민구" || saved.AuthorNames[1] != "문정현" {
		t.Errorf("AuthorNames=%v, want [이민구 문정현]", saved.AuthorNames)
	}
	if saved.ProjectName != "건강보험공단" {
		t.Errorf("ProjectName=%q", saved.ProjectName)
	}
	if saved.ReportDateText != "2026-04-23" {
		t.Errorf("ReportDateText=%q (작성자 입력 그대로 보존되어야 함)", saved.ReportDateText)
	}
	if len(saved.ThisWeek) != 1 || saved.ThisWeek[0].ElapsedDays != "2M" {
		t.Errorf("ThisWeek roundtrip: %+v", saved.ThisWeek)
	}
	if len(saved.NextWeek) != 1 || saved.NextWeek[0].StartDate != "05/04" {
		t.Errorf("NextWeek roundtrip: %+v", saved.NextWeek)
	}

	// upsert: 같은 (project, week)에 다시 저장하면 덮어씀
	req.Notes = "수정됨"
	if _, err := repo.SaveSiteReport(team.ID, a2.ID, req); err != nil {
		t.Fatalf("Save again: %v", err)
	}
	all, _ := repo.GetSiteReportsByTeamAndDate(team.ID, "2026-04-22")
	if len(all) != 1 {
		t.Errorf("len(team reports)=%d, want 1 after upsert", len(all))
	}
	if all[0].Notes != "수정됨" || all[0].AuthorUserID != a2.ID {
		t.Errorf("upsert failed: notes=%q author=%d", all[0].Notes, all[0].AuthorUserID)
	}
}

func TestGetSiteSubmittersByTeamAndDate(t *testing.T) {
	repo, cleanup := setupSiteTestDB(t)
	defer cleanup()

	leader := mkUser(t, repo, "lead@x.com", "팀장")
	a1 := mkUser(t, repo, "a@x.com", "이민구")
	a2 := mkUser(t, repo, "b@x.com", "문정현")
	a3 := mkUser(t, repo, "c@x.com", "김지인")
	team := mkTeam(t, repo, leader.ID)

	// 다른 팀 — 격리 검증용
	otherLeader := mkUser(t, repo, "other@x.com", "타팀장")
	otherTeam, _ := repo.CreateTeam("타팀", "", otherLeader.ID)
	repo.AddTeamMember(otherTeam.ID, otherLeader.ID, model.TeamRoleLeader, model.RoleCodeS)
	otherProj, _ := repo.CreateSiteProject(otherTeam.ID, model.CreateSiteProjectRequest{
		ProjectName: "타팀프로젝트",
		AuthorIDs:   []int64{otherLeader.ID},
	})

	// Case A: 다중 author 프로젝트 — 한 명만 작성해도 등록된 모두가 제출자로 카운트
	projMulti, _ := repo.CreateSiteProject(team.ID, model.CreateSiteProjectRequest{
		ProjectName: "건강보험공단",
		AuthorIDs:   []int64{a1.ID, a2.ID},
	})
	// Case B: 단일 author 프로젝트
	projSingle, _ := repo.CreateSiteProject(team.ID, model.CreateSiteProjectRequest{
		ProjectName: "한화손해보험",
		AuthorIDs:   []int64{a3.ID},
	})

	weekDate := "2026-04-24" // 금요일

	// 아직 보고서 없음 — 빈 집합
	ids, err := repo.GetSiteSubmittersByTeamAndDate(team.ID, weekDate)
	if err != nil {
		t.Fatalf("empty case: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty, got %v", ids)
	}

	// projMulti에 a1이 대표로 작성
	if _, err := repo.SaveSiteReport(team.ID, a1.ID, model.SaveSiteReportRequest{
		SiteProjectID: projMulti.ID,
		ReportDate:    weekDate,
		ThisWeek:      []model.SiteTask{{Title: "작업1", Progress: "100%"}},
	}); err != nil {
		t.Fatalf("save multi: %v", err)
	}

	ids, err = repo.GetSiteSubmittersByTeamAndDate(team.ID, weekDate)
	if err != nil {
		t.Fatalf("multi case: %v", err)
	}
	got := map[int64]bool{}
	for _, id := range ids {
		got[id] = true
	}
	// 다중 author 모두 제출자로 카운트
	if !got[a1.ID] || !got[a2.ID] {
		t.Errorf("expected a1+a2 in submitters, got %v", ids)
	}
	if got[a3.ID] {
		t.Errorf("a3 not in this project — should NOT be submitter")
	}
	if len(ids) != 2 {
		t.Errorf("len=%d, want 2", len(ids))
	}

	// projSingle도 작성
	if _, err := repo.SaveSiteReport(team.ID, a3.ID, model.SaveSiteReportRequest{
		SiteProjectID: projSingle.ID,
		ReportDate:    weekDate,
		ThisWeek:      []model.SiteTask{{Title: "작업2", Progress: "50%"}},
	}); err != nil {
		t.Fatalf("save single: %v", err)
	}

	ids, _ = repo.GetSiteSubmittersByTeamAndDate(team.ID, weekDate)
	if len(ids) != 3 {
		t.Errorf("len=%d, want 3 (a1+a2+a3)", len(ids))
	}

	// 다른 팀의 사이트 보고서는 격리되어야 함
	if _, err := repo.SaveSiteReport(otherTeam.ID, otherLeader.ID, model.SaveSiteReportRequest{
		SiteProjectID: otherProj.ID,
		ReportDate:    weekDate,
		ThisWeek:      []model.SiteTask{{Title: "타팀작업"}},
	}); err != nil {
		t.Fatalf("save other team: %v", err)
	}
	ids, _ = repo.GetSiteSubmittersByTeamAndDate(team.ID, weekDate)
	if len(ids) != 3 {
		t.Errorf("len=%d after other team save, want 3 (teamID 격리)", len(ids))
	}

	// 다른 주차는 빈 집합
	ids, _ = repo.GetSiteSubmittersByTeamAndDate(team.ID, "2026-05-01")
	if len(ids) != 0 {
		t.Errorf("different week should be empty, got %v", ids)
	}
}
