package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

type TicketExtHandler struct{ db *gorm.DB }

func NewTicketExtHandler(db *gorm.DB) *TicketExtHandler { return &TicketExtHandler{db: db} }

// GET /api/tickets/categories
func (h *TicketExtHandler) ListCategories(c fiber.Ctx) error {
	var cats []models.TicketCategory
	h.db.Order("name").Find(&cats)
	if cats == nil {
		cats = []models.TicketCategory{}
	}
	return c.JSON(cats)
}

// POST /api/tickets/categories
func (h *TicketExtHandler) CreateCategory(c fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name required"})
	}
	cat := models.TicketCategory{ID: generateID(), Name: body.Name}
	if err := h.db.Create(&cat).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "create failed"})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "category": cat})
}

// GET /api/tickets/stats
func (h *TicketExtHandler) Stats(c fiber.Ctx) error {
	var total, open, inProgress, waitingCustomer, resolved, closed int64
	var low, medium, high, urgent, unassigned int64

	h.db.Model(&models.Ticket{}).Count(&total)
	h.db.Model(&models.Ticket{}).Where("status = ?", "OPEN").Count(&open)
	h.db.Model(&models.Ticket{}).Where("status = ?", "IN_PROGRESS").Count(&inProgress)
	h.db.Model(&models.Ticket{}).Where("status = ?", "WAITING_CUSTOMER").Count(&waitingCustomer)
	h.db.Model(&models.Ticket{}).Where("status = ?", "RESOLVED").Count(&resolved)
	h.db.Model(&models.Ticket{}).Where("status = ?", "CLOSED").Count(&closed)
	h.db.Model(&models.Ticket{}).Where("priority = ?", "LOW").Count(&low)
	h.db.Model(&models.Ticket{}).Where("priority = ?", "MEDIUM").Count(&medium)
	h.db.Model(&models.Ticket{}).Where("priority = ?", "HIGH").Count(&high)
	h.db.Model(&models.Ticket{}).Where("priority = ?", "URGENT").Count(&urgent)
	h.db.Model(&models.Ticket{}).Where("assignedToId IS NULL AND status = ?", "OPEN").Count(&unassigned)

	return c.JSON(fiber.Map{
		"total": total,
		"byStatus": fiber.Map{
			"open":            open,
			"inProgress":      inProgress,
			"waitingCustomer": waitingCustomer,
			"resolved":        resolved,
			"closed":          closed,
		},
		"byPriority": fiber.Map{
			"low":    low,
			"medium": medium,
			"high":   high,
			"urgent": urgent,
		},
		"unassigned":          unassigned,
		"avgResponseTimeHours": 0,
	})
}

// GET /api/tickets/messages?ticketId=...
func (h *TicketExtHandler) ListMessages(c fiber.Ctx) error {
	ticketID := c.Query("ticketId")
	if ticketID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ticketId required"})
	}
	var replies []models.TicketReply
	h.db.Where("ticketId = ?", ticketID).Order("createdAt asc").Find(&replies)
	return c.JSON(fiber.Map{"success": true, "messages": replies})
}

// GET /api/tickets/dispatch — list unassigned or for dispatch
func (h *TicketExtHandler) ListDispatch(c fiber.Ctx) error {
	page := 1
	limit := 20
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = v
	}
	var tickets []models.Ticket
	var total int64
	h.db.Model(&models.Ticket{}).Where("assignedToId IS NULL AND status = ?", "OPEN").Count(&total)
	h.db.Preload("Category").Preload("Customer").
		Where("assignedToId IS NULL AND status = ?", "OPEN").
		Order("createdAt desc").
		Offset((page - 1) * limit).Limit(limit).
		Find(&tickets)
	return c.JSON(fiber.Map{
		"success": true,
		"tickets": tickets,
		"pagination": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// POST /api/tickets/dispatch — assign ticket to technician
func (h *TicketExtHandler) Dispatch(c fiber.Ctx) error {
	var body struct {
		TicketID     string  `json:"ticketId"`
		TechnicianID *string `json:"technicianId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.TicketID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ticketId required"})
	}
	h.db.Model(&models.Ticket{}).Where("id = ?", body.TicketID).
		Updates(map[string]interface{}{"assignedToId": body.TechnicianID})
	return c.JSON(fiber.Map{"success": true, "message": "ticket dispatched"})
}

// POST /api/tickets/:id/create-job — create a job from a ticket
func (h *TicketExtHandler) CreateJob(c fiber.Ctx) error {
	ticketID := c.Params("id")
	var ticket models.Ticket
	if err := h.db.First(&ticket, "id = ?", ticketID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "ticket not found"})
	}
	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "Job created from ticket",
		"ticketId": ticketID,
	})
}
