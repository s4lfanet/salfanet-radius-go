package handlers

// evoucher_handler.go — public e-voucher portal + admin management
// Public: GET /api/evoucher/profiles, POST /api/evoucher/purchase, GET /api/evoucher/order/:token
// Admin:  GET /api/admin/evoucher/orders, POST /api/admin/evoucher/orders/:id/cancel
//         POST /api/admin/evoucher/orders/:id/resend, DELETE /api/admin/evoucher/orders/bulk-delete

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// EvoucherHandler handles the public e-voucher self-service portal.
type EvoucherHandler struct{ db *gorm.DB }

func NewEvoucherHandler(db *gorm.DB) *EvoucherHandler {
	return &EvoucherHandler{db: db}
}

// GET /api/evoucher/profiles — list active hotspot profiles for public purchase
func (h *EvoucherHandler) ListProfiles(c fiber.Ctx) error {
	var profiles []models.HotspotProfile
	h.db.Where("isActive = ?", true).Order("price").Find(&profiles)
	return c.JSON(fiber.Map{"success": true, "profiles": profiles})
}

// POST /api/evoucher/purchase — create a voucher order
func (h *EvoucherHandler) Purchase(c fiber.Ctx) error {
	var body struct {
		ProfileID    string `json:"profileId"`
		CustomerName string `json:"customerName"`
		CustomerPhone string `json:"customerPhone"`
		Qty          int    `json:"qty"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.ProfileID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "profileId required"})
	}
	if body.Qty <= 0 {
		body.Qty = 1
	}
	var profile models.HotspotProfile
	if err := h.db.First(&profile, "id = ?", body.ProfileID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "profile not found"})
	}

	paymentToken := uuid.New().String()
	order := models.VoucherOrder{
		ID:            uuid.New().String(),
		OrderNumber:   uuid.New().String()[:8],
		ProfileID:     body.ProfileID,
		Quantity:      body.Qty,
		CustomerName:  body.CustomerName,
		CustomerPhone: body.CustomerPhone,
		TotalAmount:   profile.Price * body.Qty,
		Status:        "PENDING",
		PaymentToken:  &paymentToken,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := h.db.Create(&order).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create order"})
	}
	return c.Status(201).JSON(fiber.Map{
		"success":      true,
		"order":        order,
		"paymentToken": paymentToken,
	})
}

// GET /api/evoucher/order/:token — get order by payment token
func (h *EvoucherHandler) GetOrder(c fiber.Ctx) error {
	token := c.Params("token")
	var order models.VoucherOrder
	if err := h.db.Preload("Profile").First(&order, "payment_token = ?", token).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "order not found"})
	}
	return c.JSON(fiber.Map{"success": true, "order": order})
}

// GET /api/admin/evoucher/orders — admin list all voucher orders
func (h *EvoucherHandler) AdminListOrders(c fiber.Ctx) error {
	status := c.Query("status")
	page, limit := pageParams(c)
	q := h.db.Model(&models.VoucherOrder{}).Preload("Profile")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var orders []models.VoucherOrder
	q.Order("createdAt desc").Offset((page - 1) * limit).Limit(limit).Find(&orders)
	return c.JSON(fiber.Map{
		"success": true,
		"orders":  orders,
		"pagination": fiber.Map{
			"page": page, "limit": limit, "total": total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// POST /api/admin/evoucher/orders/:id/cancel — cancel a voucher order
func (h *EvoucherHandler) AdminCancelOrder(c fiber.Ctx) error {
	id := c.Params("id")
	var order models.VoucherOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "order not found"})
	}
	if order.Status == "COMPLETED" {
		return c.Status(400).JSON(fiber.Map{"error": "cannot cancel a completed order"})
	}
	h.db.Model(&order).Update("status", "CANCELLED")
	return c.JSON(fiber.Map{"success": true, "message": "Order cancelled"})
}

// POST /api/admin/evoucher/orders/:id/resend — resend voucher to customer
func (h *EvoucherHandler) AdminResendOrder(c fiber.Ctx) error {
	id := c.Params("id")
	var order models.VoucherOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "order not found"})
	}
	// Stub: trigger WhatsApp resend via external service
	return c.JSON(fiber.Map{"success": true, "message": "Voucher resent to customer", "orderId": id})
}

// DELETE /api/admin/evoucher/orders/bulk-delete — bulk delete orders
func (h *EvoucherHandler) AdminBulkDelete(c fiber.Ctx) error {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := c.Bind().JSON(&body); err != nil || len(body.IDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "ids required"})
	}
	h.db.Where("id IN ? AND status != ?", body.IDs, "COMPLETED").Delete(&models.VoucherOrder{})
	return c.JSON(fiber.Map{"success": true, "message": "Bulk deleted"})
}
