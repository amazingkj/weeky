package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func respondError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(ErrorResponse{Error: message})
}

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
	slog.Error("Internal error", "path", c.Path(), "method", c.Method(), "message", message)
	return respondError(c, fiber.StatusInternalServerError, message)
}
