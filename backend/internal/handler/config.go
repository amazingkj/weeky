package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/crypto"
	"github.com/jiin/weeky/internal/model"
)

// sensitiveKeys are encrypted and masked in API responses
var sensitiveKeys = map[string]bool{
	"gitlab_token":      true,
	"jira_email":        true,
	"jira_token":        true,
	"hiworks_user_id":   true,
	"hiworks_password":  true,
	"claude_api_key":    true,
}

func isSensitiveKey(key string) bool {
	if sensitiveKeys[key] {
		return true
	}
	return strings.HasSuffix(key, "_token") || strings.HasSuffix(key, "_password") ||
		strings.HasSuffix(key, "_secret") || strings.HasSuffix(key, "_api_key")
}

// GetConfig returns all config key-value pairs
// Sensitive values are masked, non-sensitive values are returned as-is
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	configs, err := h.repo.GetConfigs()
	if err != nil {
		return internalError(c, err)
	}

	if configs == nil {
		configs = []model.Config{}
	}

	result := make(map[string]interface{})
	for _, cfg := range configs {
		if cfg.Value == "" {
			result[cfg.Key] = ""
			continue
		}

		if isSensitiveKey(cfg.Key) {
			result[cfg.Key] = "***configured***"
		} else {
			// Non-sensitive: return actual value (stored as plaintext)
			result[cfg.Key] = cfg.Value
		}
	}

	return c.JSON(result)
}

// UpdateConfig updates config values
// Sensitive values are encrypted, non-sensitive values are stored as plaintext
func (h *Handler) UpdateConfig(c *fiber.Ctx) error {
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

		if err := h.repo.SetConfig(key, storeValue); err != nil {
			return internalError(c, err)
		}
	}

	return c.JSON(fiber.Map{"message": "Config updated"})
}

// GetConfigValue retrieves a config value (decrypts if sensitive)
func (h *Handler) GetConfigValue(key string) (string, error) {
	cfg, err := h.repo.GetConfig(key)
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
