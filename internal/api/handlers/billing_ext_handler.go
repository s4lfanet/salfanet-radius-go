package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// BillingExtHandler handles Phase 2 billing extension endpoints.
type BillingExtHandler struct {
	db *gorm.DB
}

func NewBillingExtHandler(db *gorm.DB) *BillingExtHandler {
	return &BillingExtHandler{db: db}
}

// ─── Invoice Discount ────────────────────────────────────────────────────────

// PUT /api/billing/invoices/:id/discount — apply discount to an invoice
func (h *BillingExtHandler) ApplyDiscount(c fiber.Ctx) error {
	id := c.Params("id")
	var inv models.Invoice
	if err := h.db.First(&inv, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Invoice not found"})
	}

	var body struct {
		Amount int    `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Amount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "discount amount must be positive"})
	}
	if body.Amount > inv.Amount {
		return c.Status(400).JSON(fiber.Map{"error": "discount cannot exceed invoice amount"})
	}
	if inv.Status != models.InvoicePending && inv.Status != models.InvoiceOverdue {
		return c.Status(400).JSON(fiber.Map{"error": "can only discount PENDING or OVERDUE invoices"})
	}

	originalAmount := inv.Amount
	inv.Amount = originalAmount - body.Amount
	inv.OriginalAmount = &originalAmount
	inv.DiscountAmount = &body.Amount
	if body.Reason != "" {
		inv.DiscountReason = &body.Reason
	}

	if err := h.db.Save(&inv).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":        "Discount applied",
		"invoice":        inv,
		"originalAmount": originalAmount,
		"discountAmount": body.Amount,
		"newAmount":      inv.Amount,
	})
}

// DELETE /api/billing/invoices/:id/discount — remove discount
func (h *BillingExtHandler) RemoveDiscount(c fiber.Ctx) error {
	id := c.Params("id")
	var inv models.Invoice
	if err := h.db.First(&inv, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Invoice not found"})
	}

	if inv.OriginalAmount != nil {
		inv.Amount = *inv.OriginalAmount
	}
	inv.OriginalAmount = nil
	inv.DiscountAmount = nil
	inv.DiscountReason = nil

	if err := h.db.Save(&inv).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Discount removed", "invoice": inv})
}

// ─── Cancel Invoice ──────────────────────────────────────────────────────────

// POST /api/billing/invoices/:id/cancel — cancel an invoice
func (h *BillingExtHandler) CancelInvoice(c fiber.Ctx) error {
	id := c.Params("id")
	var inv models.Invoice
	if err := h.db.First(&inv, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Invoice not found"})
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	if inv.Status == models.InvoicePaid {
		return c.Status(400).JSON(fiber.Map{"error": "cannot cancel a PAID invoice"})
	}
	if inv.Status == models.InvoiceCancelled {
		return c.Status(400).JSON(fiber.Map{"error": "invoice already cancelled"})
	}

	userID, _ := c.Locals("userId").(string)
	now := time.Now()
	inv.Status = models.InvoiceCancelled
	inv.CancelledAt = &now
	inv.CancelledBy = &userID
	if body.Reason != "" {
		inv.CancelReason = &body.Reason
	}

	if err := h.db.Save(&inv).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Invoice cancelled", "invoice": inv})
}

// ─── Package Change Log ──────────────────────────────────────────────────────

// GET /api/users/:id/package-logs — list package change history for a user
func (h *BillingExtHandler) ListPackageChangeLogs(c fiber.Ctx) error {
	userID := c.Params("id")
	var logs []models.PackageChangeLog
	if err := h.db.Where("userId = ?", userID).Order("changedAt DESC").Find(&logs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": logs})
}

// GET /api/package-change-logs — list all package change logs (with pagination)
func (h *BillingExtHandler) ListAllPackageChangeLogs(c fiber.Ctx) error {
	page, pageSize := pageParams(c)
	var logs []models.PackageChangeLog
	query := h.db.Model(&models.PackageChangeLog{})

	if userID := c.Query("userId"); userID != "" {
		query = query.Where("userId = ?", userID)
	}

	var total int64
	query.Count(&total)
	query.Order("changedAt DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs)

	return c.JSON(fiber.Map{"data": logs, "total": total, "page": page, "pageSize": pageSize})
}

// CreatePackageChangeLog creates a package change log entry (called internally when profile changes)
func (h *BillingExtHandler) CreatePackageChangeLog(userID, username, oldProfileID, oldProfileName, newProfileID, newProfileName, changedBy, changedByName, reason string) error {
	log := models.PackageChangeLog{
		ID:             uuid.New().String(),
		UserID:         userID,
		Username:       username,
		OldProfileID:   &oldProfileID,
		OldProfileName: &oldProfileName,
		NewProfileID:   &newProfileID,
		NewProfileName: &newProfileName,
		ChangedBy:      changedBy,
		ChangedByName:  &changedByName,
		Reason:         &reason,
	}
	return h.db.Create(&log).Error
}

// ─── Installation Log ────────────────────────────────────────────────────────

// GET /api/installation-logs — list installation logs with pagination
func (h *BillingExtHandler) ListInstallationLogs(c fiber.Ctx) error {
	page, pageSize := pageParams(c)
	var logs []models.InstallationLog
	query := h.db.Model(&models.InstallationLog{})

	if installerID := c.Query("installerId"); installerID != "" {
		query = query.Where("installerId = ?", installerID)
	}
	if fromDate := c.Query("from"); fromDate != "" {
		query = query.Where("installDate >= ?", fromDate)
	}
	if toDate := c.Query("to"); toDate != "" {
		query = query.Where("installDate <= ?", toDate)
	}

	var total int64
	query.Count(&total)
	query.Order("installDate DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs)

	return c.JSON(fiber.Map{"data": logs, "total": total, "page": page, "pageSize": pageSize})
}

// GET /api/installation-logs/:id — get single installation log
func (h *BillingExtHandler) GetInstallationLog(c fiber.Ctx) error {
	id := c.Params("id")
	var log models.InstallationLog
	if err := h.db.First(&log, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Installation log not found"})
	}
	return c.JSON(log)
}

// GET /api/users/:id/installation-log — get installation log for a user
func (h *BillingExtHandler) GetUserInstallationLog(c fiber.Ctx) error {
	userID := c.Params("id")
	var log models.InstallationLog
	if err := h.db.Where("userId = ?", userID).First(&log).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "No installation log found"})
	}
	return c.JSON(log)
}

// CreateInstallationLog creates an installation log entry (called internally during PSB)
func (h *BillingExtHandler) CreateInstallationLog(userID, username string, customerID, fullname, phone, address, identityNumber, profileName, territoryName, installerID, installerName string, latitude, longitude *float64) error {
	log := models.InstallationLog{
		ID:             uuid.New().String(),
		UserID:         userID,
		Username:       username,
		CustomerID:     &customerID,
		Fullname:       &fullname,
		Phone:          &phone,
		Address:        &address,
		IdentityNumber: &identityNumber,
		ProfileName:    &profileName,
		TerritoryName:  &territoryName,
		InstallerID:    installerID,
		InstallerName:  &installerName,
		InstallDate:    time.Now(),
		Latitude:       latitude,
		Longitude:      longitude,
	}
	return h.db.Create(&log).Error
}
