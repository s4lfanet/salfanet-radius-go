package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/eventbus"
)

// AutomationHandler handles Phase 4 automation endpoints.
type AutomationHandler struct {
	db   *gorm.DB
	bus  *eventbus.EventBus
}

func NewAutomationHandler(db *gorm.DB, bus *eventbus.EventBus) *AutomationHandler {
	return &AutomationHandler{db: db, bus: bus}
}

// ─── Notification Templates ──────────────────────────────────────────────────

// GET /api/automation/notification-templates
func (h *AutomationHandler) ListNotificationTemplates(c fiber.Ctx) error {
	var templates []models.NotificationTemplate
	h.db.Find(&templates)
	return c.JSON(fiber.Map{"data": templates})
}

// PUT /api/automation/notification-templates/:id
func (h *AutomationHandler) UpdateNotificationTemplate(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Template  *string `json:"template"`
		IsEnabled *bool   `json:"isEnabled"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	updates := map[string]interface{}{}
	if body.Template != nil {
		updates["template"] = *body.Template
	}
	if body.IsEnabled != nil {
		updates["isEnabled"] = *body.IsEnabled
	}

	if err := h.db.Model(&models.NotificationTemplate{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Template updated"})
}

// POST /api/automation/notification-templates/seed — seed default templates
func (h *AutomationHandler) SeedNotificationTemplates(c fiber.Ctx) error {
	defaults := []models.NotificationTemplate{
		{ID: uuid.New().String(), EventType: "user.isolated", Channel: "wa", Template: "Yth. {customer_name}, layanan internet Anda telah diisolir karena tagihan belum terbayar. Tagihan: Rp {amount}. Bayar sekarang di {payment_url}", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "user.isolated", Channel: "push", Template: "Layanan Anda telah diisolir. Tagihan Rp {amount} belum terbayar.", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "user.isolated", Channel: "portal", Template: "Layanan diisolir — tagihan Rp {amount} jatuh tempo {due_date}", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "user.activated", Channel: "wa", Template: "Yth. {customer_name}, layanan internet Anda telah diaktifkan kembali. Selamat menikmati!", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "user.activated", Channel: "push", Template: "Layanan Anda telah diaktifkan kembali.", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "invoice.paid", Channel: "wa", Template: "Pembayaran Rp {amount} telah diterima. Invoice {invoice_number} LUNAS. Terima kasih!", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "invoice.paid", Channel: "push", Template: "Invoice {invoice_number} telah lunas.", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "invoice.overdue", Channel: "wa", Template: "Pengingat: Invoice {invoice_number} Rp {amount} sudah jatuh tempo. Segera bayar untuk menghindari isolir.", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "invoice.overdue", Channel: "email", Template: "Tagihan Rp {amount} sudah jatuh tempo. Bayar sebelum {due_date} untuk menghindari isolir.", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "onu.offline", Channel: "telegram", Template: "⚠️ ONU {serial_number} offline > 30 menit. Pelanggan: {customer_name}", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "rx.drop", Channel: "telegram", Template: "📉 RX Power drop: {rx_power} dBm pada ONU {serial_number}. Pelanggan: {customer_name}", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "psb.deadline", Channel: "wa", Template: "Yth. {customer_name}, konfirmasi pembayaran awal dalam 24 jam untuk menghindari isolir. Bayar: {payment_url}", IsEnabled: true},
		{ID: uuid.New().String(), EventType: "user.provisioned", Channel: "wa", Template: "Selamat datang {customer_name}! Layanan {package_name} telah aktif. Username: {username}", IsEnabled: true},
	}

	for _, t := range defaults {
		h.db.Where("eventType = ? AND channel = ?", t.EventType, t.Channel).
			FirstOrCreate(&t, &t)
	}

	return c.JSON(fiber.Map{"success": true, "message": fmt.Sprintf("Seeded %d default templates", len(defaults))})
}

// ─── Alert Rules ─────────────────────────────────────────────────────────────

// GET /api/automation/alert-rules
func (h *AutomationHandler) ListAlertRules(c fiber.Ctx) error {
	var rules []models.AlertRule
	h.db.Order("priority DESC, createdAt ASC").Find(&rules)
	return c.JSON(fiber.Map{"data": rules})
}

// POST /api/automation/alert-rules
func (h *AutomationHandler) CreateAlertRule(c fiber.Ctx) error {
	var body struct {
		Name         string `json:"name"`
		TriggerEvent string `json:"triggerEvent"`
		Conditions   string `json:"conditions"`
		Actions      string `json:"actions"`
		Priority     int    `json:"priority"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	rule := models.AlertRule{
		ID:           uuid.New().String(),
		Name:         body.Name,
		TriggerEvent: body.TriggerEvent,
		Conditions:   body.Conditions,
		Actions:      body.Actions,
		IsEnabled:    true,
		Priority:     body.Priority,
	}

	if err := h.db.Create(&rule).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "rule": rule})
}

// PUT /api/automation/alert-rules/:id
func (h *AutomationHandler) UpdateAlertRule(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Name         *string `json:"name"`
		Conditions   *string `json:"conditions"`
		Actions      *string `json:"actions"`
		IsEnabled    *bool   `json:"isEnabled"`
		Priority     *int    `json:"priority"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	updates := map[string]interface{}{}
	if body.Name != nil {
		updates["name"] = *body.Name
	}
	if body.Conditions != nil {
		updates["conditions"] = *body.Conditions
	}
	if body.Actions != nil {
		updates["actions"] = *body.Actions
	}
	if body.IsEnabled != nil {
		updates["isEnabled"] = *body.IsEnabled
	}
	if body.Priority != nil {
		updates["priority"] = *body.Priority
	}

	if err := h.db.Model(&models.AlertRule{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Alert rule updated"})
}

// DELETE /api/automation/alert-rules/:id
func (h *AutomationHandler) DeleteAlertRule(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.db.Delete(&models.AlertRule{}, "id = ?", id).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Alert rule deleted"})
}

// POST /api/automation/alert-rules/seed — seed default alert rules
func (h *AutomationHandler) SeedAlertRules(c fiber.Ctx) error {
	defaults := []models.AlertRule{
		{
			ID: uuid.New().String(), Name: "RX Power Drop",
			TriggerEvent: "rx.drop",
			Conditions:   `{"rxPower": {"op": "<", "value": -28}}`,
			Actions:      `["create_ticket", "notify_admin", "notify_customer"]`,
			IsEnabled: true, Priority: 10,
		},
		{
			ID: uuid.New().String(), Name: "ONU Offline >30min",
			TriggerEvent: "onu.offline",
			Conditions:   `{"offlineMinutes": {"op": ">", "value": 30}}`,
			Actions:      `["create_ticket", "notify_admin"]`,
			IsEnabled: true, Priority: 8,
		},
		{
			ID: uuid.New().String(), Name: "Invoice Overdue 7 Days",
			TriggerEvent: "invoice.overdue",
			Conditions:   `{"overdueDays": {"op": ">", "value": 7}}`,
			Actions:      `["notify_wa", "notify_email"]`,
			IsEnabled: true, Priority: 5,
		},
		{
			ID: uuid.New().String(), Name: "PSB Deadline 2h",
			TriggerEvent: "psb.deadline",
			Conditions:   `{"hoursRemaining": {"op": "<", "value": 2}}`,
			Actions:      `["notify_admin", "notify_customer"]`,
			IsEnabled: true, Priority: 9,
		},
	}

	for _, r := range defaults {
		h.db.Where("name = ?", r.Name).FirstOrCreate(&r, &r)
	}

	return c.JSON(fiber.Map{"success": true, "message": fmt.Sprintf("Seeded %d default alert rules", len(defaults))})
}

// ─── Payment Promises ────────────────────────────────────────────────────────

// GET /api/automation/payment-promises — list all active promises
func (h *AutomationHandler) ListPaymentPromises(c fiber.Ctx) error {
	status := c.Query("status", "active")
	var promises []models.PaymentPromise
	h.db.Where("status = ?", status).Order("promiseDate ASC").Find(&promises)
	return c.JSON(fiber.Map{"data": promises})
}

// POST /api/automation/payment-promises — create a payment promise
func (h *AutomationHandler) CreatePaymentPromise(c fiber.Ctx) error {
	var body struct {
		UserID      string `json:"userId"`
		Username    string `json:"username"`
		PromiseDate string `json:"promiseDate"`
		Notes       string `json:"notes"`
		CreatedBy   string `json:"createdBy"`
		CreatedByName string `json:"createdByName"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	promiseDate, err := time.Parse("2006-01-02", body.PromiseDate)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid date format, use YYYY-MM-DD"})
	}

	promise := models.PaymentPromise{
		ID:            uuid.New().String(),
		UserID:        body.UserID,
		Username:      body.Username,
		PromiseDate:   promiseDate,
		Status:        "active",
		CreatedBy:     body.CreatedBy,
		CreatedByName: &body.CreatedByName,
	}
	if body.Notes != "" {
		promise.Notes = &body.Notes
	}

	if err := h.db.Create(&promise).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Emit event
	if h.bus != nil {
		h.bus.Publish("payment_promise.created", map[string]interface{}{
			"userId":      body.UserID,
			"username":    body.Username,
			"promiseDate": body.PromiseDate,
		})
	}

	return c.JSON(fiber.Map{"success": true, "promise": promise})
}

// POST /api/automation/payment-promises/:id/fulfill — mark as fulfilled
func (h *AutomationHandler) FulfillPaymentPromise(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.db.Model(&models.PaymentPromise{}).Where("id = ?", id).
		Update("status", "fulfilled").Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Promise fulfilled"})
}

// ─── Auto-Activation ─────────────────────────────────────────────────────────

// POST /api/automation/auto-activate/:userId — auto-activate user after payment
func (h *AutomationHandler) AutoActivate(c fiber.Ctx) error {
	userID := c.Params("userId")

	var user models.PppoeUser
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	if user.Status != "isolated" {
		return c.JSON(fiber.Map{"success": true, "message": "User not isolated, skipping", "activated": false})
	}

	// 1. Update user status to active
	if err := h.db.Model(&user).Update("status", "active").Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update user status"})
	}

	// 2. Remove Auth-Type Reject from radcheck
	h.db.Table("radcheck").Where("username = ? AND attribute = 'Auth-Type' AND value = 'Reject'", user.Username).Delete(nil)

	// 3. Emit activation event
	if h.bus != nil {
		h.bus.PublishDedup("user.activated", "user_activated_"+userID, 1*time.Hour, map[string]interface{}{
			"userId":   userID,
			"username": user.Username,
			"name":     user.Name,
		})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "User auto-activated — RADIUS reject removed, status set to active",
		"activated": true,
	})
}

// ─── Provisioning Pipeline ───────────────────────────────────────────────────

// GET /api/automation/provisioning-status/:userId — get provisioning status for a user
func (h *AutomationHandler) GetProvisioningStatus(c fiber.Ctx) error {
	userID := c.Params("userId")
	var statuses []models.ProvisioningStatus
	h.db.Where("userId = ?", userID).Order("createdAt ASC").Find(&statuses)
	return c.JSON(fiber.Map{"data": statuses})
}

// POST /api/automation/provisioning/:userId/retry — retry a failed provisioning step
func (h *AutomationHandler) RetryProvisioningStep(c fiber.Ctx) error {
	userID := c.Params("userId")
	var body struct {
		Step string `json:"step"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	// Reset the failed step to pending
	if err := h.db.Model(&models.ProvisioningStatus{}).
		Where("userId = ? AND step = ? AND status = 'failed'", userID, body.Step).
		Update("status", "pending").Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Emit event to trigger retry
	if h.bus != nil {
		h.bus.Publish("provisioning.retry", map[string]interface{}{
			"userId": userID,
			"step":   body.Step,
		})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Provisioning step queued for retry"})
}

// ─── Event Publishing (for testing/manual trigger) ───────────────────────────

// POST /api/automation/events/publish — manually publish an event (for testing)
func (h *AutomationHandler) PublishEvent(c fiber.Ctx) error {
	var body struct {
		EventType string                 `json:"eventType"`
		Payload   map[string]interface{} `json:"payload"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	if h.bus != nil {
		h.bus.Publish(body.EventType, body.Payload)
	}

	return c.JSON(fiber.Map{"success": true, "message": "Event published", "eventType": body.EventType})
}

// GET /api/automation/events/types — list all supported event types
func (h *AutomationHandler) ListEventTypes(c fiber.Ctx) error {
	eventTypes := []fiber.Map{
		{"type": "user.isolated", "description": "User diisolir (auto/manual)"},
		{"type": "user.activated", "description": "User diaktifkan kembali"},
		{"type": "invoice.paid", "description": "Invoice telah lunas"},
		{"type": "invoice.overdue", "description": "Invoice jatuh tempo"},
		{"type": "onu.offline", "description": "ONU offline >30 menit"},
		{"type": "rx.drop", "description": "RX Power drop below threshold"},
		{"type": "psb.deadline", "description": "PSB 24-jam deadline approaching"},
		{"type": "user.provisioned", "description": "User baru selesai provisioning"},
		{"type": "payment_promise.created", "description": "Janji bayar dibuat"},
		{"type": "provisioning.retry", "description": "Retry provisioning step"},
	}
	return c.JSON(fiber.Map{"data": eventTypes})
}

// ─── Template Rendering Helper ───────────────────────────────────────────────

// RenderTemplate substitutes variables in a template string.
// Variables use {variable_name} syntax.
func RenderTemplate(template string, vars map[string]string) string {
	result := template
	for key, val := range vars {
		result = strings.ReplaceAll(result, "{"+key+"}", val)
	}
	return result
}
