package handler

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jiin/weeky/internal/model"
)

func TestGetClientHistory_LeaderOnlyAndSubmittedOnly(t *testing.T) {
	app, repo := setupTeamTestApp(t)
	h := New(repo)
	app.Get("/api/teams/:id/client-history", h.GetClientHistory)

	leaderID, leaderToken := createUserWithToken(t, repo, "leader@test.com")
	memberID, memberToken := createUserWithToken(t, repo, "member@test.com")
	team, _ := repo.CreateTeam("팀A", "", leaderID)
	repo.AddTeamMember(team.ID, leaderID, model.TeamRoleLeader, model.RoleCodeS)
	repo.AddTeamMember(team.ID, memberID, model.TeamRoleMember, model.RoleCodeS)

	today := time.Now().Format("2006-01-02")
	submitted, _ := repo.CreateReport(model.CreateReportRequest{
		ReportDate: today,
		ThisWeek: []model.Task{
			{Title: "CruzAPIM", Client: "삼성카드", Details: "인증 개선", Progress: 50},
			{Title: "내부", Details: "고객사 없음"},
		},
		NextWeek: []model.Task{{Title: "CruzAPIM", Client: "삼성카드", Details: "배포"}},
	}, memberID)
	repo.SubmitReport(submitted.ID, team.ID, memberID)
	// 저장만 하고 제출하지 않은 보고서 — 제외되어야 함
	repo.CreateReport(model.CreateReportRequest{
		ReportDate: today,
		ThisWeek:   []model.Task{{Title: "Mesh", Client: "흥국화재", Details: "미제출"}},
	}, leaderID)

	url := fmt.Sprintf("/api/teams/%d/client-history", team.ID)

	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+memberToken)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Fatalf("일반 멤버는 403이어야 함, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+leaderToken)
	resp, _ = app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("팀장은 200이어야 함, got %d", resp.StatusCode)
	}
	var body model.ClientHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Entries) != 2 {
		t.Fatalf("삼성카드 금주/차주 2건이어야 함, got %+v", body.Entries)
	}
	for _, e := range body.Entries {
		if e.Client != "삼성카드" || len(e.Authors) != 1 || e.Authors[0] != "member@test.com" {
			t.Errorf("unexpected entry: %+v", e)
		}
	}
}

func TestBuildClientHistoryEntries_SiteUsesProjectClient(t *testing.T) {
	sites := []model.SiteReport{
		{
			SiteProjectID: 7, ProjectName: "도공 유지보수", ReportDate: "2026-09-25", AuthorNames: []string{"김", "이"},
			ThisWeek: []model.SiteTask{{Title: "장애 대응", Progress: "100", DueDate: "2026-09-25"}},
			NextWeek: []model.SiteNextTask{{Title: "정기 점검"}},
		},
		{SiteProjectID: 8, ReportDate: "2026-09-25", ThisWeek: []model.SiteTask{{Title: "고객사 미지정"}}},
	}
	reports := []model.SubmittedReport{
		{UserName: "박", ReportDate: "2026-09-18", ThisWeek: []model.Task{{Title: "Mesh", Client: " 흥국화재 ", Progress: 30}}},
	}

	entries := buildClientHistoryEntries(reports, sites, map[int64]string{7: "도로교통공단"})

	if len(entries) != 3 {
		t.Fatalf("got %d entries: %+v", len(entries), entries)
	}
	if entries[0].ReportDate != "2026-09-25" || entries[2].ReportDate != "2026-09-18" {
		t.Errorf("보고일 내림차순이어야 함: %+v", entries)
	}
	site := entries[0]
	if site.Client != "도로교통공단" || site.Project != "도공 유지보수" || site.Work != "장애 대응" || len(site.Authors) != 2 {
		t.Errorf("unexpected site entry: %+v", site)
	}
	if entries[2].Client != "흥국화재" || entries[2].Progress != "30" {
		t.Errorf("unexpected report entry: %+v", entries[2])
	}
}

func TestBuildClientHistoryEntries_DedupesSameUserWeek(t *testing.T) {
	task := func(work string) []model.Task {
		return []model.Task{{Title: "CruzAPIM", Client: "삼성카드", Details: work}}
	}
	reports := []model.SubmittedReport{
		{UserID: 1, UserName: "홍", ReportDate: "2026-09-22", ThisWeek: task("화요일본")}, // 같은 주, 이전
		{UserID: 1, UserName: "홍", ReportDate: "2026-09-23", ThisWeek: task("수요일본")}, // 같은 주, 최신
		{UserID: 2, UserName: "이", ReportDate: "2026-09-22", ThisWeek: task("다른 사람")},
		{UserID: 1, UserName: "홍", ReportDate: "2026-09-15", ThisWeek: task("지난주")},
	}

	entries := buildClientHistoryEntries(reports, nil, nil)

	works := map[string]bool{}
	for _, e := range entries {
		works[e.Work] = true
	}
	if len(entries) != 3 || works["화요일본"] || !works["수요일본"] || !works["다른 사람"] || !works["지난주"] {
		t.Fatalf("같은 사용자·주차는 최신 보고서만 남아야 함: %+v", entries)
	}
}
