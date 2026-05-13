package handlers

// payments_approval_handler.go — admin payment approval workflow
// GET /api/payments, POST /api/payments/:id/approve, POST /api/payments/:id/reject, GET /api/payments/manual

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// PaymentsApprovalHandler handles the payment approval workflow for admin.
type PaymentsApprovalHandler struct{ db *gorm.DB }

func NewPaymentsApprovalHandler(db *gorm.DB) *PaymentsApprovalHandler {
	return &PaymentsApprovalHandler{db: db}
}

// GET /api/payments — list all gateway payments
func (h *PaymentsApprovalHandler) List(c fiber.Ctx) error {
	status := c.Query("status")
	page, limit := pageParams(c)

	q := h.db.Model(&models.Payment{}).Preload("Invoice")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var payments []models.Payment
	q.Order("created_at desc").Offset((page - 1) * limit).Limit(limit).Find(&payments)
	return c.JSON(fiber.Map{
		"success":  true,
		"payments": payments,
		"pagination": fiber.Map{
			"page": page, "limit": limit, "total": total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// POST /api/payments/:id/approve — approve a payment and mark invoice as paid
func (h *PaymentsApprovalHandler) Approve(c fiber.Ctx) error {
	paymentID := c.Params("id")
	var payment models.Payment
	if err := h.db.First(&payment, "id = ?", paymentID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "payment not found"})
	}
	now := time.Now()
	h.db.Model(&payment).Updates(map[string]interface{}{
		"status":  "APPROVED",
		"paid_at": now,
	})
	// Also mark invoice as paid
	if payment.InvoiceID != "" {
		h.db.Model(&models.Invoice{}).Where("id = ?", payment.InvoiceID).
			Updates(map[string]interface{}{"status": "PAID", "paid_at": now})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Payment approved"})
}

// POST /api/payments/:id/reject — reject a payment
func (h *PaymentsApprovalHandler) Reject(c fiber.Ctx) error {
	paymentID := c.Params("id")
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind().JSON(&body)
	var payment models.Payment
	if err := h.db.First(&payment, "id = ?", paymentID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "payment not found"})
	}
	h.db.Model(&payment).Update("status", "REJECTED")
	return c.JSON(fiber.Map{"success": true, "message": "Payment rejected"})
}

// GET /api/payments/manual — list manual (non-gateway) payments
func (h *PaymentsApprovalHandler) ListManual(c fiber.Ctx) error {
	page, limit := pageParams(c)
	var total int64
	h.db.Model(&models.ManualPayment{}).Count(&total)
	var payments []models.ManualPayment
	h.db.Preload("Invoice").Preload("PppoeUser").
		Order("created_at desc").
		Offset((page-1)*limit).Limit(limit).
		Find(&payments)
	return c.JSON(fiber.Map{
		"success":  true,
		"payments": payments,
		"pagination": fiber.Map{
			"page": page, "limit": limit, "total": total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// POST /api/payments/manual — create a manual payment record
func (h *PaymentsApprovalHandler) CreateManual(c fiber.Ctx) error {
	var body models.ManualPayment
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.Status = "PENDING"
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "payment": body})
}
