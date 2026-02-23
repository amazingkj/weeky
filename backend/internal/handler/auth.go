package handler

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/auth"
	"github.com/jiin/weeky/internal/model"
)

// CheckSetup returns whether the system has been initialized (has any users)
func (h *Handler) CheckSetup(c *fiber.Ctx) error {
	count, err := h.repo.CountUsers()
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(fiber.Map{
		"initialized": count > 0,
	})
}

// Register creates a new user account
func (h *Handler) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		return badRequest(c, "이메일, 비밀번호, 이름은 필수입니다")
	}

	if len(req.Password) < 6 {
		return badRequest(c, "비밀번호는 6자 이상이어야 합니다")
	}

	// Check if email is already taken
	if existing, _ := h.repo.GetUserByEmail(req.Email); existing != nil {
		return badRequest(c, "이미 사용 중인 이메일입니다")
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return internalError(c, err)
	}

	// Try atomic first-admin creation (handles race condition)
	user, err := h.repo.CreateFirstAdmin(req.Email, passwordHash, req.Name)
	if err != nil {
		return internalError(c, err)
	}

	if user != nil {
		// Successfully created the first admin - reassign legacy data
		h.repo.ReassignLegacyData(user.ID)

		token, err := auth.GenerateToken(user.ID, user.Email, user.IsAdmin)
		if err != nil {
			return internalError(c, err)
		}
		refreshToken, err := auth.GenerateRefreshToken(user.ID, user.Email, user.IsAdmin)
		if err != nil {
			return internalError(c, err)
		}
		return c.Status(fiber.StatusCreated).JSON(model.AuthResponse{
			Token:        token,
			RefreshToken: refreshToken,
			User:         *user,
		})
	}

	// Not the first user - require invite code
	if req.InviteCode == "" {
		return badRequest(c, "초대 코드가 필요합니다")
	}

	ic, err := h.repo.GetInviteCodeByCode(req.InviteCode)
	if err != nil {
		return badRequest(c, "유효하지 않은 초대 코드입니다")
	}
	if ic.UsedBy != nil {
		return badRequest(c, "이미 사용된 초대 코드입니다")
	}

	// Create regular user
	user, err = h.repo.CreateUser(req.Email, passwordHash, req.Name, false)
	if err != nil {
		return internalError(c, err)
	}

	// Mark invite code as used - fail registration if this fails
	if err := h.repo.UseInviteCode(req.InviteCode, user.ID); err != nil {
		return internalError(c, err)
	}

	token, err := auth.GenerateToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		return internalError(c, err)
	}
	refreshToken, err := auth.GenerateRefreshToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		return internalError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(model.AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         *user,
	})
}

// Login authenticates a user and returns a JWT
func (h *Handler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "잘못된 요청입니다")
	}

	if req.Email == "" || req.Password == "" {
		return badRequest(c, "이메일과 비밀번호를 입력해주세요")
	}

	user, err := h.repo.GetUserByEmail(req.Email)
	if err != nil {
		return respondError(c, fiber.StatusUnauthorized, "이메일 또는 비밀번호가 올바르지 않습니다")
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		return respondError(c, fiber.StatusUnauthorized, "이메일 또는 비밀번호가 올바르지 않습니다")
	}

	token, err := auth.GenerateToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		return internalError(c, err)
	}
	refreshToken, err := auth.GenerateRefreshToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		return internalError(c, err)
	}

	return c.JSON(model.AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         *user,
	})
}

// GetMe returns the current authenticated user's info
func (h *Handler) GetMe(c *fiber.Ctx) error {
	userID := getUserID(c)
	user, err := h.repo.GetUserByID(userID)
	if err != nil {
		return notFound(c, "사용자를 찾을 수 없습니다")
	}
	return c.JSON(user)
}

// CreateInviteCode generates a new invite code (admin only, verified from DB)
func (h *Handler) CreateInviteCode(c *fiber.Ctx) error {
	userID := getUserID(c)

	// Verify admin status from DB (not just JWT claim)
	user, err := h.repo.GetUserByID(userID)
	if err != nil || !user.IsAdmin {
		return respondError(c, fiber.StatusForbidden, "관리자 권한이 필요합니다")
	}

	code, err := generateInviteCode()
	if err != nil {
		return internalError(c, err)
	}

	ic, err := h.repo.CreateInviteCode(code, userID)
	if err != nil {
		return internalError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(ic)
}

// GetInviteCodes returns all invite codes created by the admin
func (h *Handler) GetInviteCodes(c *fiber.Ctx) error {
	userID := getUserID(c)

	codes, err := h.repo.GetInviteCodes(userID)
	if err != nil {
		return internalError(c, err)
	}

	if codes == nil {
		codes = []model.InviteCode{}
	}

	return c.JSON(codes)
}

// RefreshToken issues a new access token using a valid refresh token
func (h *Handler) RefreshToken(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&req); err != nil || req.RefreshToken == "" {
		return badRequest(c, "refresh_token이 필요합니다")
	}

	claims, err := auth.ValidateToken(req.RefreshToken)
	if err != nil {
		return respondError(c, fiber.StatusUnauthorized, "유효하지 않은 리프레시 토큰입니다")
	}

	if claims.TokenType != auth.RefreshToken {
		return respondError(c, fiber.StatusUnauthorized, "잘못된 토큰 유형입니다")
	}

	// Verify user still exists
	user, err := h.repo.GetUserByID(claims.UserID)
	if err != nil {
		return respondError(c, fiber.StatusUnauthorized, "사용자를 찾을 수 없습니다")
	}

	// Issue new access token with fresh user data (admin status may have changed)
	newToken, err := auth.GenerateToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		return internalError(c, err)
	}

	return c.JSON(fiber.Map{
		"token": newToken,
	})
}

// getUserID extracts the user ID from the Fiber context (set by auth middleware)
func getUserID(c *fiber.Ctx) int64 {
	id, _ := c.Locals("userID").(int64)
	return id
}

// generateInviteCode creates a random 16-character hex code (8 bytes)
func generateInviteCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
