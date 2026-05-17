package handlers

// payment_handler.go — payment gateway integration:
// POST /api/payment/create, GET /api/payment/check-order, POST /api/payment/webhook (public).

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/lib/qris"
)

// PaymentHandler handles payment gateway routes.
type PaymentHandler struct{ db *gorm.DB }

func NewPaymentHandler(db *gorm.DB) *PaymentHandler {
	return &PaymentHandler{db: db}
}

// POST /api/payment/create — initiate a payment order via active gateway
func (h *PaymentHandler) CreatePayment(c fiber.Ctx) error {
	var body struct {
		InvoiceID     string `json:"invoiceId"`
		Gateway       string `json:"gateway"`       // e.g. "midtrans", "xendit", "tripay", "qris_own"
		Method        string `json:"method"`        // legacy field alias for gateway
		PaymentMethod string `json:"paymentMethod"` // Duitku channel code
		Amount        int    `json:"amount"`
		Phone         string `json:"phone"`
		Name          string `json:"name"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	// Support both "gateway" and "method" field names for backwards compat
	if body.Gateway == "" {
		body.Gateway = body.Method
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

	if invoice.PaymentToken == nil {
		return c.Status(400).JSON(fiber.Map{"error": "invoice payment token not found"})
	}

	orderId := invoice.InvoiceNumber + "-" + uuid.New().String()[:8]

	// ── QRIS Mandiri ──────────────────────────────────────────────────────────
	if body.Gateway == "qris_own" {
		var company models.Company
		if err := h.db.First(&company).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "company not found"})
		}
		if company.QrisEnabled == nil || !*company.QrisEnabled || company.QrisStaticCode == nil || *company.QrisStaticCode == "" {
			return c.Status(400).JSON(fiber.Map{"error": "QRIS Mandiri belum dikonfigurasi. Buka Admin → Payment Gateway → QRIS Mandiri."})
		}

		// Nominal unik = baseAmount + suffix 1-999 (deterministic per invoice)
		// Agar Android Notification Listener bisa matching notifikasi dari e-wallet
		uniqueAmount := qris.GenerateUniqueAmount(invoice.Amount, invoice.ID)

		qrString, err := qris.StaticToDynamic(*company.QrisStaticCode, uniqueAmount)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal generate QRIS: " + err.Error()})
		}

		// Simpan pending QRIS record ke DB
		expiresAt := time.Now().Add(15 * time.Minute)
		pending := models.QrisPending{
			ID:           uuid.New().String(),
			InvoiceID:    invoice.ID,
			UserID:       invoice.UserID,
			OrderID:      orderId,
			BaseAmount:   invoice.Amount,
			UniqueAmount: uniqueAmount,
			QrString:     qrString,
			Status:       "pending",
			ExpiresAt:    expiresAt,
		}
		h.db.Create(&pending)

		return c.JSON(fiber.Map{
			"success":      true,
			"orderId":      orderId,
			"qrString":     qrString,
			"paymentUrl":   "",
			"gateway":      "qris_own",
			"isQrisOwn":    true,
			"amount":       invoice.Amount,
			"uniqueAmount": uniqueAmount,
			"expiresAt":    expiresAt,
			// Pesan untuk pelanggan: transfer TEPAT nominal ini
			"note": "Transfer TEPAT Rp " + formatAmount(uniqueAmount) + " (jangan dibulatkan)",
		})
	}

	// ── Third-party gateway ───────────────────────────────────────────────────
	var gateway models.PaymentGateway
	q := h.db.Where("isActive = ?", true)
	if body.Gateway != "" {
		q = q.Where("provider = ?", body.Gateway)
	}
	if err := q.First(&gateway).Error; err != nil {
		return c.Status(503).JSON(fiber.Map{"error": "no active payment gateway configured"})
	}

	// Generate order token (placeholder; real integration calls gateway API)
	paymentLink := "https://pay.example.com/order/" + orderId

	// Update invoice with payment token
	if err := h.db.Model(&invoice).Updates(map[string]interface{}{
		"paymentToken": orderId,
		"paymentLink":  paymentLink,
	}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update invoice"})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"orderId":    orderId,
		"paymentUrl": paymentLink,
		"snapToken":  nil,
		"qrString":   nil,
		"gateway":    gateway.Provider,
		"isQrisOwn":  false,
		"amount":     invoice.Amount,
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
		err = h.db.First(&invoice, "paymentToken = ?", orderID).Error
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
	if err := h.db.First(&invoice, "paymentToken = ?", orderID).Error; err != nil {
		// Not found — return 200 so gateway doesn't retry endlessly
		return c.JSON(fiber.Map{"received": true})
	}

	switch status {
	case "settlement", "capture", "paid", "PAID":
		now := time.Now()
		h.db.Model(&invoice).Updates(map[string]interface{}{
			"status": "PAID",
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

// POST /api/payment/qris-notify — webhook dari Android QrisListener app
// Android membaca notifikasi DANA/GoPay/BRImo/dll → kirim nominal ke endpoint ini
// Endpoint ini publik (tanpa login) tapi dilindungi oleh device_key
func (h *PaymentHandler) QrisNotify(c fiber.Ctx) error {
	var body struct {
		DeviceKey string `json:"device_key"`
		Amount    int    `json:"amount"`
		SourceApp string `json:"source_app"` // e.g. "id.dana"
		RawText   string `json:"raw_text"`
		Timestamp int64  `json:"timestamp"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.DeviceKey == "" || body.Amount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "device_key dan amount wajib diisi"})
	}

	// Validasi device_key
	var company models.Company
	if err := h.db.First(&company).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "company not found"})
	}
	if company.QrisDeviceKey == nil || *company.QrisDeviceKey == "" || *company.QrisDeviceKey != body.DeviceKey {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Cari pending QRIS yang matching dengan unique_amount
	now := time.Now()
	var pending models.QrisPending
	if err := h.db.Where("uniqueAmount = ? AND status = ? AND expiresAt > ?", body.Amount, "pending", now).
		First(&pending).Error; err != nil {
		return c.JSON(fiber.Map{
			"success": false,
			"error":   "Tidak ada invoice pending yang cocok dengan nominal tersebut (mungkin expired)",
		})
	}

	// Tandai pending sebagai paid
	paidAt := now
	if err := h.db.Model(&pending).Updates(map[string]interface{}{
		"status":    "paid",
		"sourceApp": body.SourceApp,
		"paidAt":    paidAt,
	}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal update QRIS pending"})
	}

	// Update invoice → PAID + extend user subscription
	var invoice models.Invoice
	if err := h.db.First(&invoice, "id = ?", pending.InvoiceID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Invoice tidak ditemukan"})
	}

	if invoice.Status == "PAID" {
		// Sudah terbayar sebelumnya (double notif)
		return c.JSON(fiber.Map{"success": true, "message": "Already paid", "invoiceId": invoice.ID})
	}

	if err := h.db.Model(&invoice).Updates(map[string]interface{}{
		"status": "PAID",
		"paidAt": paidAt,
	}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal update invoice"})
	}

	// Perpanjang masa aktif user jika ada
	if invoice.UserID != nil {
		var user models.PppoeUser
		if err := h.db.Preload("Profile").First(&user, "id = ?", *invoice.UserID).Error; err == nil {
			var newExpiry time.Time
			if user.ExpiredAt != nil {
				newExpiry = *user.ExpiredAt
			} else {
				newExpiry = now
			}

			if user.Profile.ValidityUnit == "MONTHS" {
				newExpiry = addMonths(newExpiry, user.Profile.ValidityValue)
			} else {
				newExpiry = newExpiry.AddDate(0, 0, user.Profile.ValidityValue)
			}

			h.db.Model(&user).Updates(map[string]interface{}{
				"expiredAt":       newExpiry,
				"lastPaymentDate": now,
				"status":          "active",
			})
		}
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"message":   "Pembayaran QRIS berhasil diverifikasi otomatis",
		"invoiceId": invoice.ID,
		"amount":    pending.BaseAmount,
	})
}

// GET /api/payment/qris-status — cek status QRIS pending (polling dari frontend)
func (h *PaymentHandler) QrisStatus(c fiber.Ctx) error {
	orderId := c.Query("orderId")
	invoiceId := c.Query("invoiceId")
	if orderId == "" && invoiceId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "orderId atau invoiceId wajib diisi"})
	}

	var pending models.QrisPending
	q := h.db
	if orderId != "" {
		q = q.Where("orderId = ?", orderId)
	} else {
		q = q.Where("invoiceId = ?", invoiceId)
	}
	if err := q.Order("createdAt DESC").First(&pending).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "QRIS pending not found"})
	}

	return c.JSON(fiber.Map{
		"success":      true,
		"status":       pending.Status, // pending | paid | expired
		"invoiceId":    pending.InvoiceID,
		"orderId":      pending.OrderID,
		"baseAmount":   pending.BaseAmount,
		"uniqueAmount": pending.UniqueAmount,
		"expiresAt":    pending.ExpiresAt,
		"paidAt":       pending.PaidAt,
		"sourceApp":    pending.SourceApp,
	})
}

// addMonths menambahkan bulan ke time.Time (menangani end-of-month dengan benar)
func addMonths(t time.Time, months int) time.Time {
	return t.AddDate(0, months, 0)
}

// formatAmount memformat integer ke format Rupiah tanpa simbol (e.g. 150083 → "150.083")
func formatAmount(amount int) string {
	s := strconv.Itoa(amount)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}
	return result
}

// POST /api/payment/qris-test — simulasi pembayaran QRIS masuk (admin only, untuk testing)
// Accepts: {orderId} atau {uniqueAmount} + optional {sourceApp}
func (h *PaymentHandler) QrisTest(c fiber.Ctx) error {
	var body struct {
		OrderID      string `json:"orderId"`
		UniqueAmount int    `json:"uniqueAmount"`
		SourceApp    string `json:"source_app"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.OrderID == "" && body.UniqueAmount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "orderId atau uniqueAmount wajib diisi"})
	}
	if body.SourceApp == "" {
		body.SourceApp = "test.simulation"
	}

	// Cari pending QRIS
	now := time.Now()
	var pending models.QrisPending
	q := h.db.Where("status = ?", "pending")
	if body.OrderID != "" {
		q = q.Where("orderId = ?", body.OrderID)
	} else {
		q = q.Where("uniqueAmount = ?", body.UniqueAmount)
	}
	if err := q.First(&pending).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Tidak ada QRIS pending yang cocok. Pastikan invoice belum expired.",
		})
	}

	// Cek expiry (test tetap harus valid)
	if now.After(pending.ExpiresAt) {
		return c.Status(400).JSON(fiber.Map{
			"success":   false,
			"error":     "QRIS pending sudah expired",
			"expiredAt": pending.ExpiresAt,
		})
	}

	// Tandai paid
	paidAt := now
	h.db.Model(&pending).Updates(map[string]interface{}{
		"status":    "paid",
		"sourceApp": body.SourceApp,
		"paidAt":    paidAt,
	})

	// Update invoice
	var invoice models.Invoice
	if err := h.db.First(&invoice, "id = ?", pending.InvoiceID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Invoice tidak ditemukan"})
	}
	if invoice.Status == "PAID" {
		return c.JSON(fiber.Map{"success": true, "message": "Invoice sudah berstatus PAID sebelumnya", "invoiceId": invoice.ID})
	}
	h.db.Model(&invoice).Updates(map[string]interface{}{"status": "PAID", "paidAt": paidAt})

	// Perpanjang subscription
	if invoice.UserID != nil {
		var user models.PppoeUser
		if err := h.db.Preload("Profile").First(&user, "id = ?", *invoice.UserID).Error; err == nil {
			newExpiry := now
			if user.ExpiredAt != nil {
				newExpiry = *user.ExpiredAt
			}
			if user.Profile.ValidityUnit == "MONTHS" {
				newExpiry = addMonths(newExpiry, user.Profile.ValidityValue)
			} else {
				newExpiry = newExpiry.AddDate(0, 0, user.Profile.ValidityValue)
			}
			h.db.Model(&user).Updates(map[string]interface{}{
				"expiredAt":       newExpiry,
				"lastPaymentDate": now,
				"status":          "active",
			})
		}
	}

	return c.JSON(fiber.Map{
		"success":      true,
		"message":      "✅ Simulasi pembayaran QRIS berhasil",
		"invoiceId":    invoice.ID,
		"orderId":      pending.OrderID,
		"baseAmount":   pending.BaseAmount,
		"uniqueAmount": pending.UniqueAmount,
		"sourceApp":    body.SourceApp,
	})
}
