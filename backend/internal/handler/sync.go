package handler

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/model"
	"github.com/jiin/weeky/internal/service"
)

func (h *Handler) SyncGitHub(c *fiber.Ctx) error {
	var req model.GitHubSyncRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Invalid request body")
	}

	if req.Owner == "" || req.Repo == "" {
		return badRequest(c, "Owner와 Repo를 입력해주세요.")
	}

	if req.StartDate == "" || req.EndDate == "" {
		return badRequest(c, "조회 기간을 설정해주세요.")
	}

	userID := getUserID(c)

	if req.Token == "" {
		token, err := h.GetConfigValue("github_token", userID)
		if err != nil || token == "" {
			return badRequest(c, "GitHub 토큰이 설정되지 않았습니다. 연동 설정에서 토큰을 입력해주세요.")
		}
		req.Token = token
	}

	result, err := h.services.GitHub.Sync(req)
	if err != nil {
		return internalError(c, err)
	}

	return c.JSON(result)
}

func (h *Handler) SyncJira(c *fiber.Ctx) error {
	var req model.JiraSyncRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Invalid request body")
	}

	if req.BaseURL == "" {
		return badRequest(c, "Jira Base URL을 입력해주세요.")
	}

	if req.StartDate == "" || req.EndDate == "" {
		return badRequest(c, "조회 기간을 설정해주세요.")
	}

	userID := getUserID(c)

	if req.Email == "" {
		email, err := h.GetConfigValue("jira_email", userID)
		if err != nil || email == "" {
			return badRequest(c, "Jira 이메일이 설정되지 않았습니다. 연동 설정에서 이메일을 입력해주세요.")
		}
		req.Email = email
	}

	if req.Token == "" {
		token, err := h.GetConfigValue("jira_token", userID)
		if err != nil || token == "" {
			return badRequest(c, "Jira API 토큰이 설정되지 않았습니다. 연동 설정에서 토큰을 입력해주세요.")
		}
		req.Token = token
	}

	result, err := h.services.Jira.Sync(req)
	if err != nil {
		return internalErrorWithMessage(c, err.Error())
	}

	return c.JSON(result)
}

func (h *Handler) SyncGitLab(c *fiber.Ctx) error {
	var req model.GitLabSyncRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Invalid request body")
	}

	if req.Namespace == "" || req.Project == "" {
		return badRequest(c, "Namespace와 Project를 입력해주세요.")
	}

	if req.StartDate == "" || req.EndDate == "" {
		return badRequest(c, "조회 기간을 설정해주세요.")
	}

	if req.BaseURL == "" {
		req.BaseURL = "https://gitlab.direa.synology.me"
	}

	userID := getUserID(c)

	if req.Token == "" {
		token, err := h.GetConfigValue("gitlab_token", userID)
		if err != nil || token == "" {
			return badRequest(c, "GitLab 토큰이 설정되지 않았습니다. 연동 설정에서 토큰을 입력해주세요.")
		}
		req.Token = token
	}

	result, err := h.services.GitLab.Sync(req)
	if err != nil {
		return internalError(c, err)
	}

	return c.JSON(result)
}

func (h *Handler) SyncHiworks(c *fiber.Ctx) error {
	var req model.HiworksSyncRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Invalid request body")
	}

	if req.StartDate == "" || req.EndDate == "" {
		return badRequest(c, "조회 기간을 설정해주세요.")
	}

	userID := getUserID(c)

	if req.OfficeID == "" {
		officeID, err := h.GetConfigValue("hiworks_office_id", userID)
		if err != nil {
			slog.Error("Failed to decrypt hiworks_office_id", "error", err)
		}
		req.OfficeID = officeID
	}
	if req.UserID == "" {
		hwUserID, err := h.GetConfigValue("hiworks_user_id", userID)
		if err != nil {
			slog.Error("Failed to decrypt hiworks_user_id", "error", err)
		}
		req.UserID = hwUserID
	}
	if req.Password == "" {
		password, err := h.GetConfigValue("hiworks_password", userID)
		if err != nil {
			slog.Error("Failed to decrypt hiworks_password", "error", err)
		}
		req.Password = password
	}

	if req.OfficeID == "" || req.UserID == "" || req.Password == "" {
		return badRequest(c, "Hiworks 로그인 정보가 필요합니다 (회사ID, 사용자ID, 비밀번호)")
	}

	result, err := h.services.Hiworks.Sync(req)
	if err != nil {
		return internalErrorWithMessage(c, err.Error())
	}

	return c.JSON(result)
}

func (h *Handler) TestHiworks(c *fiber.Ctx) error {
	userID := getUserID(c)

	officeID, _ := h.GetConfigValue("hiworks_office_id", userID)
	hwUserID, _ := h.GetConfigValue("hiworks_user_id", userID)
	password, _ := h.GetConfigValue("hiworks_password", userID)

	if officeID == "" || hwUserID == "" || password == "" {
		return badRequest(c, "Hiworks 로그인 정보가 저장되지 않았습니다. 먼저 설정을 저장하세요.")
	}

	if err := h.services.Hiworks.TestLogin(officeID, hwUserID, password); err != nil {
		return internalErrorWithMessage(c, fmt.Sprintf("Hiworks 로그인 실패: %v", err))
	}

	return c.JSON(fiber.Map{"status": "ok", "message": "Hiworks 연결 성공"})
}

func (h *Handler) ListGitLabProjects(c *fiber.Ctx) error {
	userID := getUserID(c)

	token, err := h.GetConfigValue("gitlab_token", userID)
	if err != nil || token == "" {
		return badRequest(c, "GitLab 토큰이 설정되지 않았습니다. 먼저 Personal Access Token을 저장해주세요.")
	}

	baseURL := "https://gitlab.direa.synology.me"
	if customURL, err := h.GetConfigValue("gitlab_base_url", userID); err == nil && customURL != "" {
		baseURL = customURL
	}

	projects, err := h.services.GitLab.ListProjects(baseURL, token)
	if err != nil {
		return internalError(c, err)
	}

	return c.JSON(projects)
}

func (h *Handler) GenerateAIReport(c *fiber.Ctx) error {
	var req service.GenerateReportRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Invalid request body")
	}

	if len(req.Items) == 0 {
		return badRequest(c, "연동된 항목이 없습니다")
	}

	if req.StartDate == "" || req.EndDate == "" {
		return badRequest(c, "날짜 범위가 필요합니다")
	}

	userID := getUserID(c)

	apiKey, err := h.GetConfigValue("claude_api_key", userID)
	if err != nil || apiKey == "" {
		return badRequest(c, "Claude API 키가 설정되지 않았습니다. 연동 설정에서 API 키를 입력해주세요.")
	}

	generator := h.services.NewAIGenerator(apiKey)
	result, err := generator.GenerateReport(req)
	if err != nil {
		return internalError(c, err)
	}

	return c.JSON(result)
}
