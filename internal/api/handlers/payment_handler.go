package handlers

// payment_handler.go — payment gateway integration:
// POST /api/payment/create, GET /api/payment/check-order, POST /api/payment/webhook (public).

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// PaymentHandler handles payment gateway routes.
type PaymentHandler struct{ db *gorm.DB }

func NewPaymentHandler(db *gorm.DB) *PaymentHandler {
	return &PaymentHandler{db: db}
}

// POST /api/payment/create — initiate a payment order via active gateway
func (h *PaymentHandler) CreatePayment(c fiber.Ctx) error {
	var body struct {
		InvoiceID string `json:"invoiceId"`
		Method    string `json:"method"` // e.g. "midtrans", "xendit", "tripay"
		Amount    int    `json:"amount"`
		Phone     string `json:"phone"`
		Name      string `json:"name"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.InvoiceID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "invoiceId required"})
	}

	var invoice models.Invoice
	if err := h.db.First(&invoice, "id = ?", body.InvoiceID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "invoice not found"})
	}
	if invoice.Status == "PAID" {
		return c.Status(400).JSON(fiber.Map{"error": "invoice already paid"})
	}

	// Look up active gateway
	var gateway models.PaymentGateway
	q := h.db.Where("isActive = ?", true)
	if body.Method != "" {
		q = q.Where("provider = ?", body.Method)
	}
	if err := q.First(&gateway).Error; err != nil {
		return c.Status(503).JSON(fiber.Map{"error": "no active payment gateway configured"})
	}

	// Generate order token
	orderID := uuid.New().String()
	paymentLink := "https://pay.example.com/order/" + orderID // placeholder

	// Update invoice with payment token
	if err := h.db.Model(&invoice).Updates(map[string]interface{}{
		"payment_token": orderID,
		"payment_link":  paymentLink,
	}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update invoice"})
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"orderId":     orderID,
		"paymentLink": paymentLink,
		"gateway":     gateway.Provider,
		"amount":      invoice.Amount,
	})
}

// GET /api/payment/check-order — check order status by orderId or invoiceId
func (h *PaymentHandler) CheckOrder(c fiber.Ctx) error {
	orderID := c.Query("orderId")
	invoiceID := c.Query("invoiceId")
	if orderID == "" && invoiceID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "orderId or invoiceId required"})
	}

	var invoice models.Invoice
	var err error
	if orderID != "" {
		err = h.db.First(&invoice, "payment_token = ?", orderID).Error
	} else {
		err = h.db.First(&invoice, "id = ?", invoiceID).Error
	}
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "order not found"})
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"orderId":   invoice.PaymentToken,
		"invoiceId": invoice.ID,
		"status":    invoice.Status,
		"amount":    invoice.Amount,
		"paidAt":    invoice.PaidAt,
	})
}

// POST /api/payment/webhook — receives payment status updates from gateway (public, no auth)
func (h *PaymentHandler) Webhook(c fiber.Ctx) error {
	var payload map[string]interface{}
	if err := c.Bind().JSON(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	// Extract order ID (field name varies by gateway)
	var orderID string
	for _, key := range []string{"order_id", "orderId", "external_id", "referenceNumber"} {
		if v, ok := payload[key].(string); ok && v != "" {
			orderID = v
			break
		}
	}
	if orderID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "order_id not found in payload"})
	}

	// Determine payment status
	var status string
	for _, key := range []string{"transaction_status", "status", "payment_status"} {
		if v, ok := payload[key].(string); ok {
			status = v
			break
		}
	}

	var invoice models.Invoice
	if err := h.db.First(&invoice, "payment_token = ?", orderID).Error; err != nil {
		// Not found — return 200 so gateway doesn't retry endlessly
		return c.JSON(fiber.Map{"received": true})
	}

	switch status {
	case "settlement", "capture", "paid", "PAID":
		now := time.Now()
		h.db.Model(&invoice).Updates(map[string]interface{}{
			"status":  "PAID",
			"paidAt": now,
		})
	case "expire", "cancel", "EXPIRED":
		h.db.Model(&invoice).Update("status", "EXPIRED")
	}

	return c.JSON(fiber.Map{"received": true})
}

// GET /api/payment/gateways — list all payment gateways (public, for checkout UI)
func (h *PaymentHandler) ListGateways(c fiber.Ctx) error {
	var gateways []models.PaymentGateway
	h.db.Where("isActive = ?", true).Order("createdAt").Find(&gateways)
	return c.JSON(fiber.Map{"success": true, "gateways": gateways})
}
