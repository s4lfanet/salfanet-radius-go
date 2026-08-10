package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/notify"
	"github.com/s4lfanet/salfanet-radius-go/internal/radius"
)

// BillingHandler handles invoice, payment, and transaction endpoints.
type BillingHandler struct {
	db     *gorm.DB
	radius *radius.Service
}

func NewBillingHandler(db *gorm.DB, rad *radius.Service) *BillingHandler {
	return &BillingHandler{db: db, radius: rad}
}

// ─── Invoices ─────────────────────────────────────────────────────────────────

func (h *BillingHandler) ListInvoices(c fiber.Ctx) error {
	var invoices []models.Invoice
	query := h.db.Preload("User")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if userID := c.Query("userId"); userID != "" {
		query = query.Where("userId = ?", userID)
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("invoiceNumber LIKE ? OR customerName LIKE ?",
			"%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Model(&models.Invoice{}).Count(&total)
	page, pageSize := pageParams(c)
	query.Order("createdAt DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&invoices)

	return c.JSON(fiber.Map{"data": invoices, "total": total, "page": page, "pageSize": pageSize})
}

func (h *BillingHandler) GetInvoice(c fiber.Ctx) error {
	id := c.Params("id")
	var inv models.Invoice
	if err := h.db.Preload("User.Profile").First(&inv, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(inv)
}

func (h *BillingHandler) CreateInvoice(c fiber.Ctx) error {
	var body models.Invoice
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.ID = uuid.New().String()
	if body.InvoiceNumber == "" {
		body.InvoiceNumber = fmt.Sprintf("INV-%s-%d", time.Now().Format("200601"), time.Now().UnixNano())
	}
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}

func (h *BillingHandler) UpdateInvoice(c fiber.Ctx) error {
	id := c.Params("id")
	var inv models.Invoice
	if err := h.db.First(&inv, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if err := c.Bind().JSON(&inv); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	h.db.Save(&inv)
	return c.JSON(inv)
}

func (h *BillingHandler) DeleteInvoice(c fiber.Ctx) error {
	id := c.Params("id")
	h.db.Delete(&models.Invoice{}, "id = ?", id)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *BillingHandler) PayInvoice(c fiber.Ctx) error {
	id := c.Params("id")
	var inv models.Invoice
	if err := h.db.First(&inv, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if inv.Status == models.InvoicePaid {
		return c.Status(400).JSON(fiber.Map{"error": "invoice already paid"})
	}

	var body struct {
		Method string `json:"method"`
		Amount int    `json:"amount"`
		Notes  string `json:"notes"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	now := time.Now()
	inv.Status = models.InvoicePaid
	inv.PaidAt = &now
	h.db.Save(&inv)

	// Record manual payment
	notes := body.Notes
	pppoeUserID := ""
	if inv.UserID != nil {
		pppoeUserID = *inv.UserID
	}
	mp := models.ManualPayment{
		ID:           uuid.New().String(),
		InvoiceID:    inv.ID,
		PppoeUserID:  pppoeUserID,
		Amount:       float64(inv.Amount),
		BankName:     body.Method,
		AccountName:  "-",
		TransferDate: now,
		Notes:        &notes,
		Status:       "APPROVED",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	h.db.Create(&mp)

	// Extend subscription + restore RADIUS
	h.postPaymentProcess(&inv, now)

	// Send WA confirmation
	if inv.CustomerPhone != nil && inv.CustomerName != nil {
		_ = notify.SendPaymentSuccess(*inv.CustomerPhone, *inv.CustomerName, inv.InvoiceNumber, inv.Amount)
	}

	return c.JSON(fiber.Map{"message": "paid", "invoice": inv})
}

func (h *BillingHandler) SendReminderWA(c fiber.Ctx) error {
	id := c.Params("id")
	var inv models.Invoice
	if err := h.db.First(&inv, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if inv.CustomerPhone == nil || inv.CustomerName == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer phone/name missing"})
	}
	paymentLink := ""
	if inv.PaymentLink != nil {
		paymentLink = *inv.PaymentLink
	}
	if err := notify.SendInvoiceReminder(*inv.CustomerPhone, *inv.CustomerName, inv.InvoiceNumber, inv.Amount, inv.DueDate, paymentLink); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "sent"})
}

func (h *BillingHandler) GenerateMonthlyInvoices(c fiber.Ctx) error {
	// Trigger via cron service — just return stub
	return c.JSON(fiber.Map{"message": "trigger invoice generation manually via POST /api/cron/trigger/invoice_generate"})
}

// ─── Manual Payments ─────────────────────────────────────────────────────────

func (h *BillingHandler) ListManualPayments(c fiber.Ctx) error {
	var payments []models.ManualPayment
	h.db.Order("createdAt DESC").Limit(200).Find(&payments)
	return c.JSON(payments)
}

// ─── Transactions ─────────────────────────────────────────────────────────────

func (h *BillingHandler) ListTransactions(c fiber.Ctx) error {
	var transactions []models.Transaction
	query := h.db.Preload("Category")

	if txType := c.Query("type"); txType != "" {
		query = query.Where("type = ?", txType)
	}

	// Date range filter
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			query = query.Where("date >= ?", t)
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			query = query.Where("date <= ?", t.Add(24*time.Hour))
		}
	}

	var total int64
	query.Model(&models.Transaction{}).Count(&total)
	page, pageSize := pageParams(c)
	query.Order("date DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&transactions)

	return c.JSON(fiber.Map{"data": transactions, "total": total})
}

func (h *BillingHandler) CreateTransaction(c fiber.Ctx) error {
	var body models.Transaction
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.ID = uuid.New().String()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}

func (h *BillingHandler) ListTransactionCategories(c fiber.Ctx) error {
	var cats []models.TransactionCategory
	h.db.Order("name").Find(&cats)
	return c.JSON(cats)
}

func (h *BillingHandler) CreateTransactionCategory(c fiber.Ctx) error {
	var body models.TransactionCategory
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.ID = uuid.New().String()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}

// ─── Payment Gateway Webhooks ─────────────────────────────────────────────────

func (h *BillingHandler) WebhookMidtrans(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	orderID, _ := body["order_id"].(string)
	txStatus, _ := body["transaction_status"].(string)
	fraudStatus, _ := body["fraud_status"].(string)

	isPaid := (txStatus == "capture" && fraudStatus == "accept") ||
		txStatus == "settlement"

	if isPaid && orderID != "" {
		h.markInvoicePaidByToken(orderID)
	}
	return c.SendStatus(fiber.StatusOK)
}

func (h *BillingHandler) WebhookXendit(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	status, _ := body["status"].(string)
	externalID, _ := body["external_id"].(string)

	if status == "PAID" && externalID != "" {
		h.markInvoicePaidByToken(externalID)
	}
	return c.SendStatus(fiber.StatusOK)
}

func (h *BillingHandler) WebhookDuitku(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	resultCode, _ := body["resultCode"].(string)
	merchantOrderID, _ := body["merchantOrderId"].(string)

	if resultCode == "00" && merchantOrderID != "" {
		h.markInvoicePaidByToken(merchantOrderID)
	}
	return c.SendStatus(fiber.StatusOK)
}

func (h *BillingHandler) WebhookTripay(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	status, _ := body["status"].(string)
	merchantRef, _ := body["merchant_ref"].(string)

	if status == "PAID" && merchantRef != "" {
		h.markInvoicePaidByToken(merchantRef)
	}
	return c.SendStatus(fiber.StatusOK)
}

func (h *BillingHandler) markInvoicePaidByToken(token string) {
	var inv models.Invoice
	if err := h.db.Where("paymentToken = ? OR invoiceNumber = ?", token, token).First(&inv).Error; err != nil {
		return
	}
	if inv.Status == models.InvoicePaid {
		return
	}
	now := time.Now()
	inv.Status = models.InvoicePaid
	inv.PaidAt = &now
	h.db.Save(&inv)

	// Extend subscription + restore RADIUS
	h.postPaymentProcess(&inv, now)

	if inv.CustomerPhone != nil && inv.CustomerName != nil {
		_ = notify.SendPaymentSuccess(*inv.CustomerPhone, *inv.CustomerName, inv.InvoiceNumber, inv.Amount)
	}
}

// postPaymentProcess extends the user's subscription and restores RADIUS access.
// Called after any invoice is marked PAID (manual, gateway webhook, QRIS).
func (h *BillingHandler) postPaymentProcess(inv *models.Invoice, paidAt time.Time) {
	if inv.UserID == nil || *inv.UserID == "" {
		return
	}
	var user models.PppoeUser
	if err := h.db.Preload("Profile").First(&user, "id = ?", *inv.UserID).Error; err != nil {
		log.Error().Err(err).Str("invoiceId", inv.ID).Msg("billing: post-payment user lookup failed")
		return
	}

	// Extend expiry based on profile validity
	var newExpiry time.Time
	if user.ExpiredAt != nil {
		newExpiry = *user.ExpiredAt
	} else {
		newExpiry = paidAt
	}
	validity := user.Profile.ValidityValue
	if validity <= 0 {
		validity = 1
	}
	switch user.Profile.ValidityUnit {
	case "MONTHS":
		newExpiry = newExpiry.AddDate(0, validity, 0)
	case "DAYS":
		newExpiry = newExpiry.AddDate(0, 0, validity)
	default:
		newExpiry = newExpiry.AddDate(0, 1, 0)
	}

	updates := map[string]interface{}{
		"expiredAt":       newExpiry,
		"lastPaymentDate": paidAt,
		"status":          "active",
	}
	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		log.Error().Err(err).Str("userId", user.ID).Msg("billing: post-payment update failed")
		return
	}

	// Restore RADIUS if user was isolated/suspended
	if user.Status == "isolated" || user.Status == "suspended" {
		rateLimit := ""
		if user.Profile.RateLimit != nil {
			rateLimit = *user.Profile.RateLimit
		}
		if err := h.radius.RestoreUser(user.Username, user.Profile.GroupName, user.IPAddress); err != nil {
			log.Error().Err(err).Str("username", user.Username).Msg("billing: post-payment RADIUS restore failed")
		} else {
			// Also upsert rate limit in case it was missing
			_ = h.radius.SetRateLimit(user.Username, rateLimit)
			log.Info().Str("username", user.Username).Str("invoiceId", inv.ID).Msg("billing: user restored after payment")
		}
	}
}
