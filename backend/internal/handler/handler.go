package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jiin/weeky/internal/model"
	"github.com/jiin/weeky/internal/repository"
	"github.com/jiin/weeky/internal/service"
)

type Handler struct {
	repo     repository.IRepository
	services *service.Services
}

func New(repo repository.IRepository) *Handler {
	return &Handler{repo: repo, services: service.DefaultServices()}
}

func NewWithServices(repo repository.IRepository, svc *service.Services) *Handler {
	return &Handler{repo: repo, services: svc}
}

func (h *Handler) GetTemplates(c *fiber.Ctx) error {
	templates, err := h.repo.GetTemplates()
	if err != nil {
		return internalError(c, err)
	}
	if templates == nil {
		templates = []model.Template{}
	}
	return c.JSON(templates)
}

func (h *Handler) CreateTemplate(c *fiber.Ctx) error {
	var req model.CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Invalid request body")
	}

	if req.Name == "" {
		return badRequest(c, "Name is required")
	}

	template, err := h.repo.CreateTemplate(req.Name, req.Style)
	if err != nil {
		return internalError(c, err)
	}

	return c.Status(201).JSON(template)
}

func (h *Handler) UpdateTemplate(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "Invalid ID")
	}

	var req model.CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Invalid request body")
	}

	if req.Name == "" {
		return badRequest(c, "Name is required")
	}

	if err := h.repo.UpdateTemplate(id, req.Name, req.Style); err != nil {
		return internalError(c, err)
	}

	return c.SendStatus(204)
}

func (h *Handler) DeleteTemplate(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "Invalid ID")
	}

	if err := h.repo.DeleteTemplate(id); err != nil {
		return internalError(c, err)
	}

	return c.SendStatus(204)
}

func (h *Handler) GetUsers(c *fiber.Ctx) error {
	users, err := h.repo.GetAllUsers()
	if err != nil {
		return internalError(c, err)
	}
	if users == nil {
		users = []model.User{}
	}
	return c.JSON(users)
}

func (h *Handler) GetReport(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "Invalid ID")
	}

	userID := getUserID(c)
	report, err := h.repo.GetReport(id, userID)
	if err != nil {
		return notFound(c, "Report not found")
	}

	return c.JSON(report)
}

func (h *Handler) CreateReport(c *fiber.Ctx) error {
	var req model.CreateReportRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Invalid request body")
	}

	userID := getUserID(c)
	report, err := h.repo.CreateReport(req, userID)
	if err != nil {
		return internalError(c, err)
	}

	return c.Status(201).JSON(report)
}

func (h *Handler) UpdateReport(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return badRequest(c, "Invalid ID")
	}

	var req model.CreateReportRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Invalid request body")
	}

	userID := getUserID(c)
	if err := h.repo.UpdateReport(id, req, userID); err != nil {
		return internalError(c, err)
	}

	report, err := h.repo.GetReport(id, userID)
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(report)
}

func (h *Handler) SaveReport(c *fiber.Ctx) error {
	var req model.CreateReportRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Invalid request body")
	}

	userID := getUserID(c)

	existing, _ := h.repo.GetReportByDateAndUser(req.ReportDate, userID)
	if existing != nil {
		if err := h.repo.UpdateReport(existing.ID, req, userID); err != nil {
			return internalError(c, err)
		}
		report, err := h.repo.GetReport(existing.ID, userID)
		if err != nil {
			return internalError(c, err)
		}
		return c.JSON(report)
	}

	report, err := h.repo.CreateReport(req, userID)
	if err != nil {
		return internalError(c, err)
	}
	return c.Status(201).JSON(report)
}

func (h *Handler) GetReports(c *fiber.Ctx) error {
	userID := getUserID(c)
	reports, err := h.repo.GetReportsByUser(userID)
	if err != nil {
		return internalError(c, err)
	}
	if reports == nil {
		reports = []model.Report{}
	}
	return c.JSON(reports)
}
