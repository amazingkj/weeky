package handler

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/auth"
	"github.com/jiin/weeky/internal/middleware"
	"github.com/jiin/weeky/internal/model"
	"github.com/jiin/weeky/internal/repository"
)

// createUserWithToken creates a user with the given email and returns its ID and access token
func createUserWithToken(t *testing.T, repo *repository.MockRepository, email string) (int64, string) {
	t.Helper()
	hash, _ := auth.HashPassword("testpass123")
	user, err := repo.CreateUser(email, hash, email, false)
	if err != nil {
		t.Fatalf("Failed to create user %s: %v", email, err)
	}
	token, err := auth.GenerateToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	return user.ID, token
}

func setupTeamTestApp(t *testing.T) (*fiber.App, *repository.MockRepository) {
	t.Helper()

	repo := repository.NewMock()
	h := New(repo)
	app := fiber.New()

	// Middleware to inject userID from token for tests
	app.Use(func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			tokenStr := authHeader[len("Bearer "):]
			claims, err := auth.ValidateToken(tokenStr)
			if err == nil {
				c.Locals("userID", claims.UserID)
				c.Locals("email", claims.Email)
				c.Locals("isAdmin", claims.IsAdmin)
			}
		}
		return c.Next()
	})

	api := app.Group("/api")
	api.Get("/teams/:id/reports/:reportId", h.GetTeamMemberReport)
	api.Put("/teams/:id/reports/:reportId", h.UpdateTeamMemberReport)
	api.Put("/teams/:id/members/:memberId", h.UpdateTeamMember)
	api.Delete("/teams/:id/members/:memberId", h.RemoveTeamMember)
	api.Delete("/teams/:id/submit/:reportId", h.UnsubmitReport)

	return app, repo
}

// 보고서 IDOR: 다른 팀의 리더가 자기 팀에 제출되지 않은 보고서를 열람/수정할 수 없어야 함
func TestTeamMemberReport_IDOR(t *testing.T) {
	app, repo := setupTeamTestApp(t)

	victimID, victimToken := createUserWithToken(t, repo, "victim@test.com")
	attackerID, attackerToken := createUserWithToken(t, repo, "attacker@test.com")

	report, err := repo.CreateReport(model.CreateReportRequest{
		TeamName: "개발팀", AuthorName: "피해자", ReportDate: "2024-01-15",
	}, victimID)
	if err != nil {
		t.Fatalf("Failed to create report: %v", err)
	}

	teamA, _ := repo.CreateTeam("팀A", "", victimID)
	repo.AddTeamMember(teamA.ID, victimID, model.TeamRoleLeader, model.RoleCodeS)
	if _, err := repo.SubmitReport(report.ID, teamA.ID, victimID); err != nil {
		t.Fatalf("Failed to submit report: %v", err)
	}

	// 공격자가 자기 팀을 만들어 리더가 됨
	teamB, _ := repo.CreateTeam("팀B", "", attackerID)
	repo.AddTeamMember(teamB.ID, attackerID, model.TeamRoleLeader, model.RoleCodeS)

	t.Run("OtherTeamLeaderCannotReadReport", func(t *testing.T) {
		url := fmt.Sprintf("/api/teams/%d/reports/%d", teamB.ID, report.ID)
		req := httptest.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+attackerToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 404 {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("OtherTeamLeaderCannotUpdateReport", func(t *testing.T) {
		url := fmt.Sprintf("/api/teams/%d/reports/%d", teamB.ID, report.ID)
		payload := `{"team_name":"해킹","author_name":"공격자","report_date":"2024-01-15","this_week":[],"next_week":[]}`
		req := httptest.NewRequest("PUT", url, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+attackerToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 404 {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("OwnTeamLeaderCanReadSubmittedReport", func(t *testing.T) {
		url := fmt.Sprintf("/api/teams/%d/reports/%d", teamA.ID, report.ID)
		req := httptest.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+victimToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

// 팀 멤버 IDOR: 다른 팀의 리더가 타 팀 멤버를 수정/삭제할 수 없어야 함
func TestTeamMember_IDOR(t *testing.T) {
	app, repo := setupTeamTestApp(t)

	victimID, victimToken := createUserWithToken(t, repo, "victim@test.com")
	memberID, _ := createUserWithToken(t, repo, "member@test.com")
	attackerID, attackerToken := createUserWithToken(t, repo, "attacker@test.com")

	teamA, _ := repo.CreateTeam("팀A", "", victimID)
	repo.AddTeamMember(teamA.ID, victimID, model.TeamRoleLeader, model.RoleCodeS)
	targetMember, _ := repo.AddTeamMember(teamA.ID, memberID, model.TeamRoleMember, model.RoleCodeS)

	teamB, _ := repo.CreateTeam("팀B", "", attackerID)
	repo.AddTeamMember(teamB.ID, attackerID, model.TeamRoleLeader, model.RoleCodeS)

	t.Run("OtherTeamLeaderCannotUpdateMember", func(t *testing.T) {
		url := fmt.Sprintf("/api/teams/%d/members/%d", teamB.ID, targetMember.ID)
		payload := `{"role":"leader","role_code":"B"}`
		req := httptest.NewRequest("PUT", url, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+attackerToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 403 {
			t.Errorf("Expected status 403, got %d", resp.StatusCode)
		}
	})

	t.Run("OtherTeamLeaderCannotRemoveMember", func(t *testing.T) {
		url := fmt.Sprintf("/api/teams/%d/members/%d", teamB.ID, targetMember.ID)
		req := httptest.NewRequest("DELETE", url, nil)
		req.Header.Set("Authorization", "Bearer "+attackerToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 403 {
			t.Errorf("Expected status 403, got %d", resp.StatusCode)
		}
	})

	t.Run("OwnTeamLeaderCanUpdateMember", func(t *testing.T) {
		url := fmt.Sprintf("/api/teams/%d/members/%d", teamA.ID, targetMember.ID)
		payload := `{"role":"member","role_code":"D"}`
		req := httptest.NewRequest("PUT", url, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+victimToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 204 {
			t.Errorf("Expected status 204, got %d", resp.StatusCode)
		}
	})
}

// 제출 취소 권한: 일반 멤버는 본인 제출만, 팀장은 팀원 제출도 취소 가능
func TestUnsubmitReport_Permissions(t *testing.T) {
	app, repo := setupTeamTestApp(t)

	leaderID, leaderToken := createUserWithToken(t, repo, "leader@test.com")
	ownerID, ownerToken := createUserWithToken(t, repo, "owner@test.com")
	otherID, otherToken := createUserWithToken(t, repo, "other@test.com")

	team, _ := repo.CreateTeam("팀A", "", leaderID)
	repo.AddTeamMember(team.ID, leaderID, model.TeamRoleLeader, model.RoleCodeS)
	repo.AddTeamMember(team.ID, ownerID, model.TeamRoleMember, model.RoleCodeS)
	repo.AddTeamMember(team.ID, otherID, model.TeamRoleMember, model.RoleCodeS)

	report, err := repo.CreateReport(model.CreateReportRequest{
		TeamName: "팀A", AuthorName: "작성자", ReportDate: "2024-01-15",
	}, ownerID)
	if err != nil {
		t.Fatalf("Failed to create report: %v", err)
	}

	submit := func() {
		if _, err := repo.SubmitReport(report.ID, team.ID, ownerID); err != nil {
			t.Fatalf("Failed to submit report: %v", err)
		}
	}
	unsubmit := func(token string) int {
		url := fmt.Sprintf("/api/teams/%d/submit/%d", team.ID, report.ID)
		req := httptest.NewRequest("DELETE", url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		return resp.StatusCode
	}

	submit()

	t.Run("OtherMemberCannotUnsubmit", func(t *testing.T) {
		if code := unsubmit(otherToken); code != 403 {
			t.Errorf("Expected status 403, got %d", code)
		}
		if _, err := repo.GetSubmissionByReport(report.ID, team.ID); err != nil {
			t.Error("Submission should still exist after forbidden unsubmit")
		}
	})

	t.Run("OwnerCanUnsubmit", func(t *testing.T) {
		if code := unsubmit(ownerToken); code != 204 {
			t.Errorf("Expected status 204, got %d", code)
		}
	})

	t.Run("LeaderCanUnsubmitMemberSubmission", func(t *testing.T) {
		submit()
		if code := unsubmit(leaderToken); code != 204 {
			t.Errorf("Expected status 204, got %d", code)
		}
	})
}

// refresh 토큰으로는 RequireAuth로 보호된 API에 접근할 수 없어야 함
func TestRequireAuth_RejectsRefreshToken(t *testing.T) {
	repo := repository.NewMock()
	h := New(repo)
	app := fiber.New()
	app.Get("/api/auth/me", middleware.RequireAuth(), h.GetMe)

	hash, _ := auth.HashPassword("testpass123")
	user, err := repo.CreateUser("refresh@test.com", hash, "Refresh User", false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("RefreshTokenRejected", func(t *testing.T) {
		refreshToken, err := auth.GenerateRefreshToken(user.ID, user.Email, user.IsAdmin)
		if err != nil {
			t.Fatalf("Failed to generate refresh token: %v", err)
		}
		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+refreshToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 401 {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("AccessTokenAccepted", func(t *testing.T) {
		accessToken, err := auth.GenerateToken(user.ID, user.Email, user.IsAdmin)
		if err != nil {
			t.Fatalf("Failed to generate access token: %v", err)
		}
		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}
