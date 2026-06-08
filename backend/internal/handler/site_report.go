package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/model"
)

// requireTeamLeaderOrGroup: 팀장/그룹장/admin만 통과
func (h *Handler) requireTeamLeaderOrGroup(c *fiber.Ctx, teamID, userID int64) error {
	if isAdmin(c) {
		return nil
	}
	member, err := h.repo.GetTeamMember(teamID, userID)
	if err != nil || (member.Role != model.TeamRoleLeader && member.Role != model.TeamRoleGroupLeader) {
		return respondError(c, fiber.StatusForbidden, "팀장 또는 그룹장 권한이 필요합니다")
	}
	return nil
}

// requireTeamMember: 팀 멤버 또는 admin만 통과
func (h *Handler) requireTeamMember(c *fiber.Ctx, teamID, userID int64) error {
	if isAdmin(c) {
		return nil
	}
	if _, err := h.repo.GetTeamMember(teamID, userID); err != nil {
		return respondError(c, fiber.StatusForbidden, "팀 멤버가 아닙니다")
	}
	return nil
}

// --- SiteProject endpoints ---

func (h *Handler) GetSiteProjects(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	userID := getUserID(c)
	if err := h.requireTeamMember(c, teamID, userID); err != nil {
		return err
	}

	activeOnly := c.Query("active_only") == "true"
	projects, err := h.repo.GetSiteProjects(teamID, activeOnly)
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(projects)
}

func (h *Handler) GetMySiteProjects(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	userID := getUserID(c)
	if err := h.requireTeamMember(c, teamID, userID); err != nil {
		return err
	}

	projects, err := h.repo.GetSiteProjectsByAuthor(teamID, userID)
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(projects)
}

func (h *Handler) CreateSiteProject(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	userID := getUserID(c)
	if err := h.requireTeamLeaderOrGroup(c, teamID, userID); err != nil {
		return err
	}

	var req model.CreateSiteProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}
	if req.ProjectName == "" {
		return badRequest(c, "프로젝트명은 필수입니다")
	}

	project, err := h.repo.CreateSiteProject(teamID, req)
	if err != nil {
		return badRequest(c, "이미 존재하는 프로젝트명이거나 오류가 발생했습니다")
	}
	return c.Status(201).JSON(project)
}

func (h *Handler) UpdateSiteProject(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	pid, err := strconv.ParseInt(c.Params("pid"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 프로젝트 ID입니다")
	}
	userID := getUserID(c)
	if err := h.requireTeamLeaderOrGroup(c, teamID, userID); err != nil {
		return err
	}

	project, err := h.repo.GetSiteProject(pid)
	if err != nil {
		return notFound(c, "프로젝트를 찾을 수 없습니다")
	}
	if project.TeamID != teamID {
		return respondError(c, fiber.StatusForbidden, "해당 팀의 프로젝트가 아닙니다")
	}

	var req model.UpdateSiteProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}
	if req.ProjectName == "" {
		return badRequest(c, "프로젝트명은 필수입니다")
	}

	if err := h.repo.UpdateSiteProject(pid, req); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Handler) DeleteSiteProject(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	pid, err := strconv.ParseInt(c.Params("pid"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 프로젝트 ID입니다")
	}
	userID := getUserID(c)
	if err := h.requireTeamLeaderOrGroup(c, teamID, userID); err != nil {
		return err
	}

	project, err := h.repo.GetSiteProject(pid)
	if err != nil {
		return notFound(c, "프로젝트를 찾을 수 없습니다")
	}
	if project.TeamID != teamID {
		return respondError(c, fiber.StatusForbidden, "해당 팀의 프로젝트가 아닙니다")
	}

	if err := h.repo.DeleteSiteProject(pid); err != nil {
		return internalError(c, err)
	}
	return c.SendStatus(204)
}

// --- SiteReport endpoints ---

func (h *Handler) SaveSiteReport(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	userID := getUserID(c)
	if err := h.requireTeamMember(c, teamID, userID); err != nil {
		return err
	}

	var req model.SaveSiteReportRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}
	if req.SiteProjectID == 0 || req.ReportDate == "" {
		return badRequest(c, "site_project_id와 report_date는 필수입니다")
	}

	project, err := h.repo.GetSiteProject(req.SiteProjectID)
	if err != nil || project.TeamID != teamID {
		return notFound(c, "사이트 프로젝트를 찾을 수 없습니다")
	}

	// admin이 아니면 작성자 등록되어 있어야 함
	if !isAdmin(c) {
		ok, err := h.repo.IsSiteProjectAuthor(req.SiteProjectID, userID)
		if err != nil {
			return internalError(c, err)
		}
		if !ok {
			return respondError(c, fiber.StatusForbidden, "이 사이트 보고서의 작성자가 아닙니다")
		}
	}

	report, err := h.repo.SaveSiteReport(teamID, userID, req)
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(report)
}

func (h *Handler) GetSiteReport(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	userID := getUserID(c)
	if err := h.requireTeamMember(c, teamID, userID); err != nil {
		return err
	}

	siteProjectID, err := strconv.ParseInt(c.Query("site_project_id"), 10, 64)
	if err != nil || siteProjectID == 0 {
		return badRequest(c, "site_project_id는 필수입니다")
	}
	reportDate := c.Query("report_date")
	if reportDate == "" {
		return badRequest(c, "report_date는 필수입니다")
	}

	project, err := h.repo.GetSiteProject(siteProjectID)
	if err != nil || project.TeamID != teamID {
		return notFound(c, "사이트 프로젝트를 찾을 수 없습니다")
	}

	report, err := h.repo.GetSiteReportByProjectAndDate(siteProjectID, reportDate)
	if err != nil {
		return c.JSON(fiber.Map{"exists": false})
	}
	return c.JSON(fiber.Map{"exists": true, "report": report})
}

// GetMySiteReports: 본인이 작성자로 배정된 사이트 보고서 전체 이력 (내 히스토리용)
func (h *Handler) GetMySiteReports(c *fiber.Ctx) error {
	teamID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "잘못된 팀 ID입니다")
	}
	userID := getUserID(c)
	if err := h.requireTeamMember(c, teamID, userID); err != nil {
		return err
	}

	reports, err := h.repo.GetSiteReportsByUser(teamID, userID)
	if err != nil {
		return internalError(c, err)
	}
	if reports == nil {
		reports = []model.SiteReport{}
	}
	return c.JSON(reports)
}

// GetTeamSiteReports: 팀 단위 사이트 보고서 조회 (취합 미리보기 + PPT 출력용)
// 팀장/그룹장만 접근. 사이트 작성자도 본인 보고서만 보고 싶다면 GetSiteReport(단일) 사용.
func (h *Handler) GetTeamSiteReports(c *fiber.Ctx) error {
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

	reports, err := h.repo.GetSiteReportsByTeamAndDate(teamID, reportDate)
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(reports)
}