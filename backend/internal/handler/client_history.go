package handler

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/model"
)

// GetClientHistory: 팀장/그룹장용 고객사별 업무 히스토리.
// 팀원 원본 보고서 중 팀에 "제출된" 것만 포함 (미제출·임시저장 제외) + 팀 사이트 보고서.
func (h *Handler) GetClientHistory(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	if err := h.requireTeamLeaderOrGroup(c, teamID, getUserID(c)); err != nil {
		return err
	}

	weeks, err := strconv.Atoi(c.Query("weeks", "12"))
	if err != nil || weeks < 1 || weeks > 24 {
		weeks = 12
	}

	now := time.Now()
	offset := int(now.Weekday()) - 1
	if now.Weekday() == time.Sunday {
		offset = 6
	}
	currentMonday := now.AddDate(0, 0, -offset)
	from := currentMonday.AddDate(0, 0, -7*(weeks-1)).Format("2006-01-02")
	to := currentMonday.AddDate(0, 0, 6).Format("2006-01-02")

	reports, err := h.repo.GetSubmittedReportsByTeamRange(teamID, from, to)
	if err != nil {
		return internalError(c, err)
	}
	siteReports, err := h.repo.GetSiteReportsByTeamRange(teamID, from, to)
	if err != nil {
		return internalError(c, err)
	}
	siteProjects, err := h.repo.GetSiteProjects(teamID, false)
	if err != nil {
		return internalError(c, err)
	}
	siteClients := make(map[int64]string, len(siteProjects))
	for _, p := range siteProjects {
		siteClients[p.ID] = p.ClientName
	}

	return c.JSON(model.ClientHistoryResponse{
		From:    from,
		To:      to,
		Entries: buildClientHistoryEntries(reports, siteReports, siteClients),
	})
}

// buildClientHistoryEntries: 고객사가 지정된 업무만 모아 보고일 내림차순으로 반환.
// 본사 보고서는 task.client, 사이트 보고서는 사이트 프로젝트의 고객사를 기준으로 함.
func buildClientHistoryEntries(reports []model.SubmittedReport, siteReports []model.SiteReport, siteClients map[int64]string) []model.ClientHistoryEntry {
	entries := []model.ClientHistoryEntry{}

	for _, r := range latestReportPerUserWeek(reports) {
		add := func(section string, t model.Task, progress string) {
			client := strings.TrimSpace(t.Client)
			if client == "" {
				return
			}
			entries = append(entries, model.ClientHistoryEntry{
				Client:     client,
				ReportDate: r.ReportDate,
				Kind:       "report",
				Section:    section,
				Authors:    []string{r.UserName},
				Project:    t.Title,
				Work:       t.Details,
				Progress:   progress,
				DueDate:    t.DueDate,
			})
		}
		for _, t := range r.ThisWeek {
			add("this_week", t, strconv.Itoa(t.Progress))
		}
		for _, t := range r.NextWeek {
			add("next_week", t, "")
		}
	}

	for _, sr := range siteReports {
		client := strings.TrimSpace(siteClients[sr.SiteProjectID])
		if client == "" {
			continue
		}
		authors := sr.AuthorNames
		if authors == nil {
			authors = []string{}
		}
		for _, t := range sr.ThisWeek {
			entries = append(entries, model.ClientHistoryEntry{
				Client: client, ReportDate: sr.ReportDate, Kind: "site", Section: "this_week",
				Authors: authors, Project: sr.ProjectName, Work: t.Title, Progress: t.Progress, DueDate: t.DueDate,
			})
		}
		for _, t := range sr.NextWeek {
			entries = append(entries, model.ClientHistoryEntry{
				Client: client, ReportDate: sr.ReportDate, Kind: "site", Section: "next_week",
				Authors: authors, Project: sr.ProjectName, Work: t.Title, DueDate: t.DueDate,
			})
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].ReportDate > entries[j].ReportDate
	})
	return entries
}

// latestReportPerUserWeek: 보고서가 날짜 단위라 한 사람이 같은 주에 여러 건 제출할 수 있음
// (예: 화·수요일 각각 저장 후 제출) → (사용자, 주차)별로 가장 최근 보고서만 남겨 중복 표시 방지
func latestReportPerUserWeek(reports []model.SubmittedReport) []model.SubmittedReport {
	type key struct {
		userID int64
		monday string
	}
	latest := make(map[key]int, len(reports)) // key → reports 인덱스
	var order []key
	for i, r := range reports {
		k := key{r.UserID, weekMonday(r.ReportDate)}
		j, ok := latest[k]
		if !ok {
			order = append(order, k)
			latest[k] = i
		} else if r.ReportDate > reports[j].ReportDate {
			latest[k] = i
		}
	}
	result := make([]model.SubmittedReport, 0, len(order))
	for _, k := range order {
		result = append(result, reports[latest[k]])
	}
	return result
}

// weekMonday: YYYY-MM-DD가 속한 주의 월요일. 파싱 실패 시 원문 그대로 (그 날짜 단독 주로 취급)
func weekMonday(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	offset := int(t.Weekday()) - 1
	if t.Weekday() == time.Sunday {
		offset = 6
	}
	return t.AddDate(0, 0, -offset).Format("2006-01-02")
}
