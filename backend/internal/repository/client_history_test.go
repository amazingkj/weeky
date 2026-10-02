package repository

import (
	"path/filepath"
	"testing"

	"github.com/jiin/weeky/internal/model"
)

func TestGetSubmittedReportsByTeamRange_OnlySubmittedInRange(t *testing.T) {
	repo, err := New(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer repo.Close()

	user, _ := repo.CreateUser("u@test.com", "hash", "홍길동", false)
	team, _ := repo.CreateTeam("팀A", "", user.ID)

	mk := func(date string) *model.Report {
		r, err := repo.CreateReport(model.CreateReportRequest{
			ReportDate: date,
			ThisWeek:   []model.Task{{Title: "CruzAPIM", Client: "삼성카드", Progress: 40}},
		}, user.ID)
		if err != nil {
			t.Fatalf("CreateReport: %v", err)
		}
		return r
	}
	inRange := mk("2026-09-25")
	outOfRange := mk("2026-06-05")
	mk("2026-09-18") // 미제출

	for _, r := range []*model.Report{inRange, outOfRange} {
		if _, err := repo.SubmitReport(r.ID, team.ID, user.ID); err != nil {
			t.Fatalf("SubmitReport: %v", err)
		}
	}

	got, err := repo.GetSubmittedReportsByTeamRange(team.ID, "2026-07-06", "2026-10-04")
	if err != nil {
		t.Fatalf("GetSubmittedReportsByTeamRange: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("기간 내 제출 보고서 1건이어야 함, got %d: %+v", len(got), got)
	}
	if got[0].UserName != "홍길동" || got[0].ReportDate != "2026-09-25" || got[0].ThisWeek[0].Client != "삼성카드" {
		t.Errorf("unexpected: %+v", got[0])
	}

	sites, err := repo.GetSiteReportsByTeamRange(team.ID, "2026-07-06", "2026-10-04")
	if err != nil || len(sites) != 0 {
		t.Fatalf("site reports: %v, %+v", err, sites)
	}
}
