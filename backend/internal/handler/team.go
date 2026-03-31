package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/model"
	"github.com/jiin/weeky/internal/service"
)

func isAdmin(c *fiber.Ctx) bool {
	v, _ := c.Locals("isAdmin").(bool)
	return v
}

func (h *Handler) CreateTeam(c *fiber.Ctx) error {
	var req model.CreateTeamRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}
	if req.Name == "" {
		return badRequest(c, "팀 이름은 필수입니다")
	}

	userID := getUserID(c)
	team, err := h.repo.CreateTeam(req.Name, req.Description, userID)
	if err != nil {
		return internalError(c, err)
	}

	_, err = h.repo.AddTeamMember(team.ID, userID, model.TeamRoleLeader, model.RoleCodeS)
	if err != nil {
		return internalError(c, err)
	}

	return c.Status(201).JSON(team)
}

func (h *Handler) GetMyTeams(c *fiber.Ctx) error {
	userID := getUserID(c)
	teams, err := h.repo.GetTeamsByUser(userID)
	if err != nil {
		return internalError(c, err)
	}
	if teams == nil {
		teams = []model.Team{}
	}
	return c.JSON(teams)
}

func (h *Handler) GetTeam(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	if _, err := h.repo.GetTeamMember(teamID, userID); err != nil {
		return respondError(c, fiber.StatusForbidden, "팀 멤버가 아닙니다")
	}

	team, err := h.repo.GetTeam(teamID)
	if err != nil {
		return notFound(c, "팀을 찾을 수 없습니다")
	}
	return c.JSON(team)
}

func (h *Handler) UpdateTeam(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || member.Role != model.TeamRoleLeader) {
		return respondError(c, fiber.StatusForbidden, "팀장 권한이 필요합니다")
	}

	var req model.CreateTeamRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}

	if err := h.repo.UpdateTeam(teamID, req.Name, req.Description); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) DeleteTeam(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || member.Role != model.TeamRoleLeader) {
		return respondError(c, fiber.StatusForbidden, "팀장 권한이 필요합니다")
	}

	if err := h.repo.DeleteTeam(teamID); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) AddTeamMember(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || member.Role != model.TeamRoleLeader) {
		return respondError(c, fiber.StatusForbidden, "팀장 권한이 필요합니다")
	}

	var req model.AddTeamMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}
	if req.Email == "" {
		return badRequest(c, "이메일은 필수입니다")
	}

	targetUser, err := h.repo.GetUserByEmail(req.Email)
	if err != nil {
		return notFound(c, "해당 이메일의 사용자를 찾을 수 없습니다")
	}

	if req.Role == "" {
		req.Role = model.TeamRoleMember
	}
	if req.RoleCode == "" {
		req.RoleCode = model.RoleCodeS
	}

	newMember, err := h.repo.AddTeamMember(teamID, targetUser.ID, req.Role, req.RoleCode)
	if err != nil {
		slog.Error("AddTeamMember failed", "teamID", teamID, "userID", targetUser.ID, "error", err)
		return badRequest(c, "이미 팀에 추가된 멤버이거나 오류가 발생했습니다")
	}

	return c.Status(201).JSON(newMember)
}

func (h *Handler) GetTeamMembers(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	if _, err := h.repo.GetTeamMember(teamID, userID); err != nil {
		return respondError(c, fiber.StatusForbidden, "팀 멤버가 아닙니다")
	}

	members, err := h.repo.GetTeamMembers(teamID)
	if err != nil {
		return internalError(c, err)
	}
	if members == nil {
		members = []model.TeamMember{}
	}
	return c.JSON(members)
}

func (h *Handler) UpdateTeamMember(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	memberID, err := strconv.ParseInt(c.Params("memberId"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 멤버 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || member.Role != model.TeamRoleLeader) {
		return respondError(c, fiber.StatusForbidden, "팀장 권한이 필요합니다")
	}

	var req model.UpdateTeamMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}

	if err := h.repo.UpdateTeamMember(memberID, req.Role, req.RoleCode, req.Name); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) RemoveTeamMember(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	memberID, err := strconv.ParseInt(c.Params("memberId"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 멤버 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || member.Role != model.TeamRoleLeader) {
		return respondError(c, fiber.StatusForbidden, "팀장 권한이 필요합니다")
	}

	if err := h.repo.RemoveTeamMember(memberID); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) GetMySubmission(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	if _, err := h.repo.GetTeamMember(teamID, userID); err != nil {
		return respondError(c, fiber.StatusForbidden, "팀 멤버가 아닙니다")
	}

	reportDate := c.Query("report_date")
	if reportDate == "" {
		return badRequest(c, "report_date는 필수입니다")
	}

	sub, err := h.repo.GetSubmissionByUser(teamID, userID, reportDate)
	if err != nil {
		return c.JSON(fiber.Map{"submitted": false})
	}
	return c.JSON(fiber.Map{"submitted": true, "submission": sub})
}

func (h *Handler) GetMySubmissions(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	if _, err := h.repo.GetTeamMember(teamID, userID); err != nil {
		return respondError(c, fiber.StatusForbidden, "팀 멤버가 아닙니다")
	}

	subs, err := h.repo.GetSubmissionsByUser(teamID, userID)
	if err != nil {
		return internalError(c, err)
	}
	if subs == nil {
		subs = []model.ReportSubmission{}
	}
	return c.JSON(subs)
}

func (h *Handler) SubmitReport(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	if _, err := h.repo.GetTeamMember(teamID, userID); err != nil {
		return respondError(c, fiber.StatusForbidden, "팀 멤버가 아닙니다")
	}

	var req model.SubmitReportRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}
	if req.ReportID == 0 {
		return badRequest(c, "보고서 ID는 필수입니다")
	}

	report, err := h.repo.GetReport(req.ReportID, userID)
	if err != nil || report == nil {
		return notFound(c, "보고서를 찾을 수 없습니다")
	}

	sub, err := h.repo.SubmitReport(req.ReportID, teamID, userID)
	if err != nil {
		return internalError(c, err)
	}
	return c.Status(201).JSON(sub)
}

func (h *Handler) UnsubmitReport(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	reportID, err := strconv.ParseInt(c.Params("reportId"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 보고서 ID입니다")
	}

	userID := getUserID(c)
	if _, err := h.repo.GetTeamMember(teamID, userID); err != nil {
		return respondError(c, fiber.StatusForbidden, "팀 멤버가 아닙니다")
	}

	if err := h.repo.UnsubmitReport(reportID, teamID); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) GetTeamSubmissions(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || (member.Role != model.TeamRoleLeader && member.Role != model.TeamRoleGroupLeader)) {
		return respondError(c, fiber.StatusForbidden, "팀장 또는 그룹장 권한이 필요합니다")
	}

	reportDate := c.Query("report_date")
	if reportDate == "" {
		return badRequest(c, "report_date는 필수입니다")
	}

	members, err := h.repo.GetTeamMembers(teamID)
	if err != nil {
		return internalError(c, err)
	}

	submissions, err := h.repo.GetSubmissions(teamID, reportDate)
	if err != nil {
		return internalError(c, err)
	}

	subMap := make(map[int64]model.ReportSubmission)
	for _, s := range submissions {
		subMap[s.UserID] = s
	}

	type MemberWithSubmission struct {
		model.TeamMember
		Submission *model.ReportSubmission `json:"submission"`
	}

	result := make([]MemberWithSubmission, 0, len(members))
	for _, m := range members {
		ms := MemberWithSubmission{TeamMember: m}
		if sub, ok := subMap[m.UserID]; ok {
			ms.Submission = &sub
		}
		result = append(result, ms)
	}

	return c.JSON(result)
}

func (h *Handler) GetTeamMemberReport(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	reportID, err := strconv.ParseInt(c.Params("reportId"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 보고서 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || (member.Role != model.TeamRoleLeader && member.Role != model.TeamRoleGroupLeader)) {
		return respondError(c, fiber.StatusForbidden, "팀장 또는 그룹장 권한이 필요합니다")
	}

	report, err := h.repo.GetReportByID(reportID)
	if err != nil {
		slog.Error("GetTeamMemberReport failed", "reportID", reportID, "error", err)
		return notFound(c, "보고서를 찾을 수 없습니다")
	}
	return c.JSON(report)
}

func (h *Handler) UpdateTeamMemberReport(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	reportID, err := strconv.ParseInt(c.Params("reportId"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 보고서 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || (member.Role != model.TeamRoleLeader && member.Role != model.TeamRoleGroupLeader)) {
		return respondError(c, fiber.StatusForbidden, "팀장 또는 그룹장 권한이 필요합니다")
	}

	var req model.CreateReportRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}

	if err := h.repo.UpdateReportByID(reportID, req); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) GetConsolidatedReport(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || (member.Role != model.TeamRoleLeader && member.Role != model.TeamRoleGroupLeader)) {
		return respondError(c, fiber.StatusForbidden, "팀장 또는 그룹장 권한이 필요합니다")
	}

	reportDate := c.Query("report_date")
	if reportDate == "" {
		return badRequest(c, "report_date는 필수입니다")
	}

	team, err := h.repo.GetTeam(teamID)
	if err != nil {
		return notFound(c, "팀을 찾을 수 없습니다")
	}

	members, err := h.repo.GetTeamMembers(teamID)
	if err != nil {
		return internalError(c, err)
	}

	submissions, err := h.repo.GetSubmissions(teamID, reportDate)
	if err != nil {
		return internalError(c, err)
	}

	subMap := make(map[int64]model.ReportSubmission)
	for _, s := range submissions {
		subMap[s.UserID] = s
	}

	var memberReports []model.MemberReportData
	for _, m := range members {
		mrd := model.MemberReportData{
			UserID:   m.UserID,
			UserName: m.UserName,
			RoleCode: m.RoleCode,
		}
		if sub, ok := subMap[m.UserID]; ok {
			report, err := h.repo.GetReportByID(sub.ReportID)
			if err == nil {
				mrd.Report = report
			}
		}
		memberReports = append(memberReports, mrd)
	}

	return c.JSON(model.ConsolidatedReport{
		Team:       *team,
		ReportDate: reportDate,
		Members:    memberReports,
	})
}

func (h *Handler) SummarizeConsolidatedReport(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || (member.Role != model.TeamRoleLeader && member.Role != model.TeamRoleGroupLeader)) {
		return respondError(c, fiber.StatusForbidden, "팀장 또는 그룹장 권한이 필요합니다")
	}

	var req struct {
		ReportDate string `json:"report_date"`
	}
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}
	if req.ReportDate == "" {
		return badRequest(c, "report_date는 필수입니다")
	}

	apiKey, err := h.GetConfigValue("claude_api_key", userID)
	if err != nil || apiKey == "" {
		return badRequest(c, "Claude API 키가 설정되지 않았습니다. 설정에서 API 키를 입력해주세요.")
	}

	team, err := h.repo.GetTeam(teamID)
	if err != nil {
		return notFound(c, "팀을 찾을 수 없습니다")
	}

	members, err := h.repo.GetTeamMembers(teamID)
	if err != nil {
		return internalError(c, err)
	}

	submissions, err := h.repo.GetSubmissions(teamID, req.ReportDate)
	if err != nil {
		return internalError(c, err)
	}

	subMap := make(map[int64]model.ReportSubmission)
	for _, s := range submissions {
		subMap[s.UserID] = s
	}

	var reportText strings.Builder
	fmt.Fprintf(&reportText, "팀: %s\n보고일: %s\n\n", team.Name, req.ReportDate)

	for _, m := range members {
		sub, ok := subMap[m.UserID]
		if !ok {
			continue
		}
		report, err := h.repo.GetReportByID(sub.ReportID)
		if err != nil {
			continue
		}

		fmt.Fprintf(&reportText, "## %s (%s)\n", m.UserName, string(m.RoleCode))
		if len(report.ThisWeek) > 0 {
			reportText.WriteString("### 금주실적:\n")
			for _, t := range report.ThisWeek {
				fmt.Fprintf(&reportText, "- %s: %s (진행률: %d%%)\n", t.Title, t.Details, t.Progress)
			}
		}
		if len(report.NextWeek) > 0 {
			reportText.WriteString("### 차주계획:\n")
			for _, t := range report.NextWeek {
				fmt.Fprintf(&reportText, "- %s: %s\n", t.Title, t.Details)
			}
		}
		if report.Issues != "" {
			fmt.Fprintf(&reportText, "### 이슈: %s\n", report.Issues)
		}
		if report.Notes != "" {
			fmt.Fprintf(&reportText, "### 특이사항: %s\n", report.Notes)
		}
		reportText.WriteString("\n")
	}

	items := []model.SyncItem{
		{
			Title:   "팀 취합 보고서",
			Content: reportText.String(),
			Date:    req.ReportDate,
			Type:    "email",
		},
	}

	projects, projErr := h.repo.GetTeamProjects(teamID, true)
	if projErr != nil {
		slog.Warn("Failed to get team projects for AI context", "teamID", teamID, "error", projErr)
	}
	var projectNames []string
	for _, p := range projects {
		if p.Client != "" {
			projectNames = append(projectNames, fmt.Sprintf("%s (고객사: %s)", p.Name, p.Client))
		} else {
			projectNames = append(projectNames, p.Name)
		}
	}

	generator := h.services.NewAIGenerator(apiKey)
	result, err := generator.GenerateReport(service.GenerateReportRequest{
		Items:        items,
		StartDate:    req.ReportDate,
		EndDate:      req.ReportDate,
		Style:        "detailed",
		ProjectNames: projectNames,
	})
	if err != nil {
		return internalError(c, err)
	}

	return c.JSON(result)
}

func (h *Handler) GetTeamProjects(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	if _, err := h.repo.GetTeamMember(teamID, userID); err != nil {
		return respondError(c, fiber.StatusForbidden, "팀 멤버가 아닙니다")
	}

	activeOnly := c.Query("active_only") == "true"
	projects, err := h.repo.GetTeamProjects(teamID, activeOnly)
	if err != nil {
		return internalError(c, err)
	}
	if projects == nil {
		projects = []model.TeamProject{}
	}
	return c.JSON(projects)
}

func (h *Handler) CreateTeamProject(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || member.Role != model.TeamRoleLeader) {
		return respondError(c, fiber.StatusForbidden, "팀장 권한이 필요합니다")
	}

	var req model.CreateTeamProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}
	if req.Name == "" {
		return badRequest(c, "프로젝트 이름은 필수입니다")
	}

	project, err := h.repo.CreateTeamProject(teamID, req.Name, req.Client)
	if err != nil {
		return badRequest(c, "이미 존재하는 프로젝트 이름이거나 오류가 발생했습니다")
	}
	return c.Status(201).JSON(project)
}

func (h *Handler) AutoCreateTeamProject(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	if _, err := h.repo.GetTeamMember(teamID, userID); err != nil {
		return respondError(c, fiber.StatusForbidden, "팀 멤버가 아닙니다")
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}
	if req.Name == "" {
		return badRequest(c, "프로젝트 이름은 필수입니다")
	}

	project, err := h.repo.GetOrCreateTeamProject(teamID, req.Name)
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(project)
}

func (h *Handler) UpdateTeamProject(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	pid, err := strconv.ParseInt(c.Params("pid"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 프로젝트 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || member.Role != model.TeamRoleLeader) {
		return respondError(c, fiber.StatusForbidden, "팀장 권한이 필요합니다")
	}

	project, err := h.repo.GetTeamProject(pid)
	if err != nil {
		return notFound(c, "프로젝트를 찾을 수 없습니다")
	}
	if project.TeamID != teamID {
		return respondError(c, fiber.StatusForbidden, "해당 팀의 프로젝트가 아닙니다")
	}

	var req model.UpdateTeamProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}

	if err := h.repo.UpdateTeamProject(pid, req.Name, req.Client, req.IsActive); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) DeleteTeamProject(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	pid, err := strconv.ParseInt(c.Params("pid"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 프로젝트 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || member.Role != model.TeamRoleLeader) {
		return respondError(c, fiber.StatusForbidden, "팀장 권한이 필요합니다")
	}

	project, err := h.repo.GetTeamProject(pid)
	if err != nil {
		return notFound(c, "프로젝트를 찾을 수 없습니다")
	}
	if project.TeamID != teamID {
		return respondError(c, fiber.StatusForbidden, "해당 팀의 프로젝트가 아닙니다")
	}

	if err := h.repo.DeleteTeamProject(pid); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) ReorderTeamProjects(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || member.Role != model.TeamRoleLeader) {
		return respondError(c, fiber.StatusForbidden, "팀장 권한이 필요합니다")
	}

	var req model.ReorderProjectsRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}
	if len(req.IDs) == 0 {
		return badRequest(c, "프로젝트 ID 목록이 필요합니다")
	}

	if err := h.repo.ReorderTeamProjects(teamID, req.IDs); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) SaveConsolidatedEdit(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || (member.Role != model.TeamRoleLeader && member.Role != model.TeamRoleGroupLeader)) {
		return respondError(c, fiber.StatusForbidden, "팀장/그룹장 권한이 필요합니다")
	}

	var req model.SaveConsolidatedEditRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}
	if req.ReportDate == "" {
		return badRequest(c, "보고일자가 필요합니다")
	}

	dataBytes, err := json.Marshal(map[string]interface{}{
		"this_week":   req.ThisWeek,
		"next_week":   req.NextWeek,
		"issues":      req.Issues,
		"notes":       req.Notes,
		"next_issues": req.NextIssues,
		"next_notes":  req.NextNotes,
	})
	if err != nil {
		return internalError(c, err)
	}

	if err := h.repo.SaveConsolidatedEdit(teamID, req.ReportDate, string(dataBytes), userID); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) GetConsolidatedEdit(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || (member.Role != model.TeamRoleLeader && member.Role != model.TeamRoleGroupLeader)) {
		return respondError(c, fiber.StatusForbidden, "팀장/그룹장 권한이 필요합니다")
	}

	reportDate := c.Query("report_date")
	if reportDate == "" {
		return badRequest(c, "report_date 파라미터가 필요합니다")
	}

	edit, err := h.repo.GetConsolidatedEdit(teamID, reportDate)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return c.JSON(fiber.Map{"exists": false})
		}
		return internalError(c, err)
	}

	return c.JSON(fiber.Map{
		"exists":     true,
		"data":       json.RawMessage(edit.Data),
		"updated_at": edit.UpdatedAt,
	})
}

func (h *Handler) DeleteConsolidatedEdit(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	member, err := h.repo.GetTeamMember(teamID, userID)
	if !isAdmin(c) && (err != nil || (member.Role != model.TeamRoleLeader && member.Role != model.TeamRoleGroupLeader)) {
		return respondError(c, fiber.StatusForbidden, "팀장/그룹장 권한이 필요합니다")
	}

	reportDate := c.Query("report_date")
	if reportDate == "" {
		return badRequest(c, "report_date 파라미터가 필요합니다")
	}

	if err := h.repo.DeleteConsolidatedEdit(teamID, reportDate); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) GetTeamHistory(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}

	userID := getUserID(c)
	if _, err := h.repo.GetTeamMember(teamID, userID); err != nil {
		return respondError(c, fiber.StatusForbidden, "팀 멤버가 아닙니다")
	}

	weeksStr := c.Query("weeks", "8")
	weeks, err := strconv.Atoi(weeksStr)
	if err != nil || weeks < 1 || weeks > 24 {
		weeks = 8
	}

	team, err := h.repo.GetTeam(teamID)
	if err != nil {
		return notFound(c, "팀을 찾을 수 없습니다")
	}

	members, err := h.repo.GetTeamMembers(teamID)
	if err != nil {
		return internalError(c, err)
	}

	now := time.Now()
	daysSinceMonday := int(now.Weekday()) - 1
	if now.Weekday() == time.Sunday {
		daysSinceMonday = 6
	}
	currentMonday := now.AddDate(0, 0, -daysSinceMonday)

	var weekSummaries []model.WeekSummary
	for i := 0; i < weeks; i++ {
		monday := currentMonday.AddDate(0, 0, -7*i)
		friday := monday.AddDate(0, 0, 4)
		weekDate := monday.Format("2006-01-02")
		fridayDate := friday.Format("2006-01-02")

		submissions, err := h.repo.GetSubmissions(teamID, weekDate)
		if err != nil {
			continue
		}

		var submittedNames []string
		for _, s := range submissions {
			if s.UserName != "" {
				submittedNames = append(submittedNames, s.UserName)
			}
		}

		weekSummaries = append(weekSummaries, model.WeekSummary{
			WeekDate:       weekDate,
			FridayDate:     fridayDate,
			SubmittedCount: len(submissions),
			TotalMembers:   len(members),
			SubmittedNames: submittedNames,
		})
	}

	resp := model.TeamHistoryResponse{
		TeamID:   team.ID,
		TeamName: team.Name,
		Weeks:    weekSummaries,
	}

	return c.JSON(resp)
}

