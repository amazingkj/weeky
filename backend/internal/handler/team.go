package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/model"
	"github.com/jiin/weeky/internal/service"
)

func isAdmin(c *fiber.Ctx) bool {
	v, _ := c.Locals("isAdmin").(bool)
	return v
}

// ============ Team CRUD ============

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

	// Auto-add creator as leader
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

// ============ Team Members ============

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

	if err := h.repo.UpdateTeamMember(memberID, req.Role, req.RoleCode); err != nil {
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

// ============ Report Submissions ============

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

	// Verify the report belongs to the user
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

// ============ AI Summarization for Consolidated Report ============

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

	// Get Claude API key
	apiKey, err := h.GetConfigValue("claude_api_key", userID)
	if err != nil || apiKey == "" {
		return badRequest(c, "Claude API 키가 설정되지 않았습니다. 설정에서 API 키를 입력해주세요.")
	}

	// Get consolidated data
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

	// Build a text summary of all members' reports
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

	// Convert consolidated report text into SyncItems for the AI generator
	items := []model.SyncItem{
		{
			Title:   "팀 취합 보고서",
			Content: reportText.String(),
			Date:    req.ReportDate,
			Type:    "email",
		},
	}

	generator := h.services.NewAIGenerator(apiKey)
	result, err := generator.GenerateReport(service.GenerateReportRequest{
		Items:     items,
		StartDate: req.ReportDate,
		EndDate:   req.ReportDate,
		Style:     "detailed",
	})
	if err != nil {
		return internalError(c, err)
	}

	return c.JSON(result)
}
