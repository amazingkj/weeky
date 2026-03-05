package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/crypto"
	"github.com/jiin/weeky/internal/model"
)

var sensitiveKeys = map[string]bool{
	"gitlab_token":     true,
	"jira_email":       true,
	"jira_token":       true,
	"hiworks_user_id":  true,
	"hiworks_password": true,
	"claude_api_key":   true,
}

func isSensitiveKey(key string) bool {
	return sensitiveKeys[key] ||
		strings.HasSuffix(key, "_token") || strings.HasSuffix(key, "_password") ||
		strings.HasSuffix(key, "_secret") || strings.HasSuffix(key, "_api_key")
}

func (h *Handler) GetConfig(c *fiber.Ctx) error {
	userID := getUserID(c)
	configs, err := h.repo.GetConfigs(userID)
	if err != nil {
		return internalError(c, err)
	}

	if configs == nil {
		configs = []model.Config{}
	}

	result := make(map[string]interface{})
	for _, cfg := range configs {
		switch {
		case cfg.Value == "":
			result[cfg.Key] = ""
		case isSensitiveKey(cfg.Key):
			result[cfg.Key] = "***configured***"
		default:
			result[cfg.Key] = cfg.Value
		}
	}

	return c.JSON(result)
}

func (h *Handler) UpdateConfig(c *fiber.Ctx) error {
	userID := getUserID(c)
	var req model.ConfigUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Invalid request body")
	}

	for key, value := range req.Configs {
		if value == "" {
			continue
		}

		storeValue := value
		if isSensitiveKey(key) {
			encrypted, err := crypto.Encrypt(value)
			if err != nil {
				return internalErrorWithMessage(c, "Encryption failed")
			}
			storeValue = encrypted
		}

		if err := h.repo.SetConfig(key, storeValue, userID); err != nil {
			return internalError(c, err)
		}
	}

	return c.JSON(fiber.Map{"message": "Config updated"})
}

func (h *Handler) GetConfigValue(key string, userID int64) (string, error) {
	cfg, err := h.repo.GetConfig(key, userID)
	if err != nil {
		return "", err
	}

	if cfg.Value == "" {
		return "", nil
	}

	if isSensitiveKey(key) {
		return crypto.Decrypt(cfg.Value)
	}
	return cfg.Value, nil
}
