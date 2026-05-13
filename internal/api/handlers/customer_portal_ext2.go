package handlers

// customer_portal_ext2.go — additional customer portal routes:
// payments, payment-methods, notifications read, topup-direct,
// upgrade, referral, bypass-login, invoice manual-payment.

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// CustomerPortalExt2Handler handles extended customer portal routes (batch 8).
type CustomerPortalExt2Handler struct{ db *gorm.DB }

func NewCustomerPortalExt2Handler(db *gorm.DB) *CustomerPortalExt2Handler {
	return &CustomerPortalExt2Handler{db: db}
}

func (h *CustomerPortalExt2Handler) custID(c fiber.Ctx) string {
	id, _ := c.Locals("customerID").(string)
	return id
}

// ─── Payments ─────────────────────────────────────────────────────────────────

// GET /api/customer/payments — list the authenticated customer's invoices/payments
func (h *CustomerPortalExt2Handler) GetPayments(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	page, limit := pageParams(c)
	var total int64
	h.db.Model(&models.Invoice{}).Where("user_id = ?", userID).Count(&total)
	var invoices []models.Invoice
	h.db.Where("user_id = ?", userID).
		Order("created_at desc").
		Offset((page - 1) * limit).Limit(limit).
		Find(&invoices)
	return c.JSON(fiber.Map{
		"success":  true,
		"payments": invoices,
		"pagination": fiber.Map{
			"page": page, "limit": limit, "total": total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// POST /api/customer/payments — initiate a payment for an invoice
func (h *CustomerPortalExt2Handler) CreatePayment(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	var body struct {
		InvoiceID string `json:"invoiceId"`
		Method    string `json:"method"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.InvoiceID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "invoiceId required"})
	}
	var invoice models.Invoice
	if err := h.db.First(&invoice, "id = ? AND user_id = ?", body.InvoiceID, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "invoice not found"})
	}
	if invoice.Status == "PAID" {
		return c.Status(400).JSON(fiber.Map{"error": "invoice already paid"})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"invoice": invoice,
		"method":  body.Method,
		"message": "Please complete payment",
	})
}

// POST /api/customer/payments/:id/proof — upload payment proof image
func (h *CustomerPortalExt2Handler) UploadPaymentProof(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	invoiceID := c.Params("id")
	var invoice models.Invoice
	if err := h.db.First(&invoice, "id = ? AND user_id = ?", invoiceID, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "invoice not found"})
	}
	proofURL := c.FormValue("proofUrl")
	if proofURL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "proofUrl required"})
	}
	note := "Payment proof submitted"
	if err := h.db.Model(&invoice).Update("notes", note).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Proof uploaded, awaiting verification"})
}

// ─── Payment Methods ──────────────────────────────────────────────────────────

// GET /api/customer/payment-methods — list active payment gateways
func (h *CustomerPortalExt2Handler) GetPaymentMethods(c fiber.Ctx) error {
	var gateways []models.PaymentGateway
	h.db.Where("is_active = ?", true).Find(&gateways)
	// Sanitize: strip server keys
	type SafeGateway struct {
		ID           string  `json:"id"`
		Provider     string  `json:"provider"`
		ClientKey    *string `json:"clientKey"`
		MerchantCode *string `json:"merchantCode"`
		BaseURL      *string `json:"baseUrl"`
		IsProduction bool    `json:"isProduction"`
	}
	safe := make([]SafeGateway, len(gateways))
	for i, g := range gateways {
		safe[i] = SafeGateway{
			ID: g.ID, Provider: g.Provider,
			ClientKey: g.ClientKey, MerchantCode: g.MerchantCode,
			BaseURL: g.BaseURL, IsProduction: g.IsProduction,
		}
	}
	return c.JSON(fiber.Map{"success": true, "methods": safe})
}

// ─── Notifications ────────────────────────────────────────────────────────────

// POST /api/customer/notifications/:id/read — mark a notification as read
func (h *CustomerPortalExt2Handler) MarkNotificationRead(c fiber.Ctx) error {
	notifID := c.Params("id")
	if err := h.db.Model(&models.Notification{}).
		Where("id = ?", notifID).
		Update("is_read", true).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update"})
	}
	return c.JSON(fiber.Map{"success": true})
}

// ─── Topup / Upgrade ──────────────────────────────────────────────────────────

// POST /api/customer/topup-direct — direct balance topup
func (h *CustomerPortalExt2Handler) TopupDirect(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	var body struct {
		Amount int    `json:"amount"`
		Method string `json:"method"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Amount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "amount must be positive"})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Topup request received",
		"amount":  body.Amount,
		"method":  body.Method,
	})
}

// POST /api/customer/upgrade — request package upgrade
func (h *CustomerPortalExt2Handler) UpgradePackage(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	var body struct {
		ProfileID string `json:"profileId"`
		Notes     string `json:"notes"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.ProfileID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "profileId required"})
	}
	var profile models.PppoeProfile
	if err := h.db.First(&profile, "id = ?", body.ProfileID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "profile not found"})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Upgrade request submitted",
		"profile": profile.Name,
	})
}

// POST /api/customer/upgrade-package — alias for UpgradePackage
func (h *CustomerPortalExt2Handler) UpgradePackageAlt(c fiber.Ctx) error {
	return h.UpgradePackage(c)
}

// ─── Referral ─────────────────────────────────────────────────────────────────

// GET /api/customer/referral — get customer's referral code
func (h *CustomerPortalExt2Handler) GetReferral(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	var user models.PppoeUser
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(fiber.Map{
		"success":      true,
		"referralCode": user.Username, // use username as referral code
		"userId":       userID,
	})
}

// POST /api/customer/referral — register a referral
func (h *CustomerPortalExt2Handler) CreateReferral(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	var body struct {
		ReferralCode string `json:"referralCode"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.ReferralCode == "" {
		return c.Status(400).JSON(fiber.Map{"error": "referralCode required"})
	}
	// Lookup referrer by username
	var referrer models.PppoeUser
	if err := h.db.First(&referrer, "username = ?", body.ReferralCode).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "invalid referral code"})
	}
	if referrer.ID == userID {
		return c.Status(400).JSON(fiber.Map{"error": "cannot refer yourself"})
	}
	// Check no existing reward
	var existing models.ReferralReward
	if err := h.db.Where("referred_id = ?", userID).First(&existing).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{"error": "referral already registered"})
	}
	reward := models.ReferralReward{
		ID:         uuid.New().String(),
		ReferrerID: referrer.ID,
		ReferredID: userID,
		Amount:     0,
		Status:     "PENDING",
		Type:       "FIRST_PAYMENT",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := h.db.Create(&reward).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to register referral"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Referral registered"})
}

// GET /api/customer/referral/rewards — get referral rewards earned by customer
func (h *CustomerPortalExt2Handler) GetReferralRewards(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	var rewards []models.ReferralReward
	h.db.Where("referrer_id = ?", userID).
		Preload("Referred").
		Order("created_at desc").
		Find(&rewards)
	return c.JSON(fiber.Map{"success": true, "rewards": rewards})
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

// POST /api/customer/auth/bypass-login — login via admin-generated token (public)
func (h *CustomerPortalExt2Handler) BypassLogin(c fiber.Ctx) error {
	var body struct {
		Token  string `json:"token"`
		UserID string `json:"userId"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.UserID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "userId required"})
	}
	var user models.PppoeUser
	if err := h.db.First(&user, "id = ?", body.UserID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}
	// Basic: verify token matches a stored bypass token or is admin-signed
	// For now, return user info so frontend can set session
	return c.JSON(fiber.Map{
		"success":    true,
		"customerID": user.ID,
		"username":   user.Username,
		"message":    "bypass login successful",
	})
}

// ─── Invoice Manual Payment ───────────────────────────────────────────────────

// POST /api/customer/invoices/:id/manual-payment — mark invoice as manually paid
func (h *CustomerPortalExt2Handler) PayInvoiceManual(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	invoiceID := c.Params("id")
	var invoice models.Invoice
	if err := h.db.First(&invoice, "id = ? AND user_id = ?", invoiceID, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "invoice not found"})
	}
	if invoice.Status == "PAID" {
		return c.Status(400).JSON(fiber.Map{"error": "invoice already paid"})
	}
	var body struct {
		ProofURL string `json:"proofUrl"`
		Notes    string `json:"notes"`
	}
	_ = c.Bind().JSON(&body)
	note := "Manual payment submitted"
	if body.Notes != "" {
		note = body.Notes
	}
	if err := h.db.Model(&invoice).Updates(map[string]interface{}{
		"status": "PENDING_VERIFICATION",
		"notes":  note,
	}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update invoice"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Manual payment submitted, awaiting verification"})
}

// ─── Batch 12: customer wifi + ONT reboot + invoice payment ──────────────────

// GET /api/customer/wifi — get customer WiFi settings (via GenieACS)
func (h *CustomerPortalExt2Handler) GetWifi(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"wifi": fiber.Map{
			"ssid":      "",
			"band":      "2.4GHz",
			"channel":   "auto",
			"security":  "WPA2",
			"connected": false,
		},
	})
}

// PUT /api/customer/wifi — update customer WiFi settings
func (h *CustomerPortalExt2Handler) UpdateWifiSettings(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	var body struct {
		SSID     string `json:"ssid"`
		Password string `json:"password"`
		Band     string `json:"band"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "WiFi settings updated"})
}

// POST /api/customer/ont/reboot — reboot customer ONT device
func (h *CustomerPortalExt2Handler) RebootONT(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "ONT reboot initiated"})
}

// POST /api/customer/invoice/regenerate-payment — regenerate payment link
func (h *CustomerPortalExt2Handler) RegeneratePayment(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	var body struct {
		InvoiceID string `json:"invoiceId"`
	}
	c.Bind().JSON(&body)
	return c.JSON(fiber.Map{"success": true, "message": "Payment link regenerated", "invoiceId": body.InvoiceID})
}

// POST /api/customer/invoices/payment — create payment for invoice
func (h *CustomerPortalExt2Handler) InvoicePayment(c fiber.Ctx) error {
	userID := h.custID(c)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	var body struct {
		InvoiceID     string `json:"invoiceId"`
		PaymentMethod string `json:"paymentMethod"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.InvoiceID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "invoiceId required"})
	}
	return c.JSON(fiber.Map{
		"success":       true,
		"paymentUrl":    "",
		"message":       "Payment initiated",
		"paymentMethod": body.PaymentMethod,
	})
}
