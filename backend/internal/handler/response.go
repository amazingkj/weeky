package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// Error codes
const (
	ErrCodeValidation = "VALIDATION_ERROR"
	ErrCodeNotFound   = "NOT_FOUND"
	ErrCodeInternal   = "INTERNAL_ERROR"
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeBadRequest = "BAD_REQUEST"
)

// respondError sends a standardized error response
func respondError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(ErrorResponse{Error: message})
}

// respondErrorWithCode sends an error response with an error code
func respondErrorWithCode(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(ErrorResponse{Error: message, Code: code})
}

// Common error responses
func badRequest(c *fiber.Ctx, message string) error {
	return respondError(c, fiber.StatusBadRequest, message)
}

func notFound(c *fiber.Ctx, message string) error {
	return respondError(c, fiber.StatusNotFound, message)
}

func internalError(c *fiber.Ctx, err error) error {
	slog.Error("Internal error", "path", c.Path(), "method", c.Method(), "error", err.Error())
	return respondError(c, fiber.StatusInternalServerError, "Internal server error")
}

func internalErrorWithMessage(c *fiber.Ctx, message string) error {
	return respondError(c, fiber.StatusInternalServerError, message)
}
