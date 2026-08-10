package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

type AddonHandler struct{ db *gorm.DB }

func NewAddonHandler(db *gorm.DB) *AddonHandler {
	return &AddonHandler{db: db}
}

// ─── Addon Types CRUD ─────────────────────────────────────────────────────────

// GET /api/addon-types
func (h *AddonHandler) ListAddonTypes(c fiber.Ctx) error {
	var addons []models.AddonType
	h.db.Order("name ASC").Find(&addons)
	return c.JSON(addons)
}

// POST /api/addon-types
func (h *AddonHandler) CreateAddonType(c fiber.Ctx) error {
	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
		Price       int     `json:"price"`
		IsRecurring bool    `json:"isRecurring"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Nama addon wajib diisi"})
	}
	addon := models.AddonType{
		Name:        body.Name,
		Description: body.Description,
		Price:       body.Price,
		IsRecurring: body.IsRecurring,
		IsActive:    true,
	}
	if err := h.db.Create(&addon).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"id": addon.ID, "message": "Addon berhasil dibuat"})
}

// PUT /api/addon-types/:id
func (h *AddonHandler) UpdateAddonType(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Price       *int    `json:"price"`
		IsRecurring *bool   `json:"isRecurring"`
		IsActive    *bool   `json:"isActive"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	updates := map[string]interface{}{}
	if body.Name != nil {
		updates["name"] = *body.Name
	}
	if body.Description != nil {
		updates["description"] = *body.Description
	}
	if body.Price != nil {
		updates["price"] = *body.Price
	}
	if body.IsRecurring != nil {
		updates["isRecurring"] = *body.IsRecurring
	}
	if body.IsActive != nil {
		updates["isActive"] = *body.IsActive
	}
	if err := h.db.Model(&models.AddonType{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Addon diperbarui"})
}

// DELETE /api/addon-types/:id
func (h *AddonHandler) DeleteAddonType(c fiber.Ctx) error {
	id := c.Params("id")
	var count int64
	h.db.Model(&models.CustomerAddon{}).Where("addonTypeId = ? AND endDate IS NULL", id).Count(&count)
	if count > 0 {
		h.db.Model(&models.AddonType{}).Where("id = ?", id).Update("isActive", false)
		return c.JSON(fiber.Map{"message": "Addon dinonaktifkan (masih digunakan pelanggan aktif)"})
	}
	if err := h.db.Delete(&models.AddonType{}, id).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Addon dihapus"})
}

// ─── Customer Addons ──────────────────────────────────────────────────────────

// GET /api/customers/:userId/addons
func (h *AddonHandler) ListCustomerAddons(c fiber.Ctx) error {
	userId := c.Params("userId")
	var addons []models.CustomerAddon
	h.db.Preload("AddonType").Where("userId = ?", userId).Order("startDate DESC").Find(&addons)
	return c.JSON(addons)
}

// POST /api/customers/:userId/addons
func (h *AddonHandler) AssignCustomerAddon(c fiber.Ctx) error {
	userId := c.Params("userId")
	var body struct {
		AddonTypeID   uint    `json:"addonTypeId"`
		PriceOverride *int    `json:"priceOverride"`
		StartDate     string  `json:"startDate"`
		Notes         *string `json:"notes"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.AddonTypeID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "addonTypeId wajib"})
	}

	startDate, err := time.Parse("2006-01-02", body.StartDate)
	if err != nil || body.StartDate == "" {
		startDate = time.Now()
	}

	ca := models.CustomerAddon{
		UserID:        userId,
		AddonTypeID:   body.AddonTypeID,
		PriceOverride: body.PriceOverride,
		StartDate:     startDate,
		Notes:         body.Notes,
	}
	if err := h.db.Create(&ca).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Update unpaid invoices if recurring
	var addonType models.AddonType
	h.db.First(&addonType, body.AddonTypeID)
	if addonType.IsRecurring {
		effectivePrice := addonType.Price
		if body.PriceOverride != nil {
			effectivePrice = *body.PriceOverride
		}
		startPeriod := startDate.Format("2006-01")
		h.db.Exec(`UPDATE invoices SET addonAmount = addonAmount + ?, amount = amount + ? WHERE userId = ? AND status = 'PENDING' AND invoiceType = 'MONTHLY' AND DATE_FORMAT(dueDate, '%Y-%m') >= ?`,
			effectivePrice, effectivePrice, userId, startPeriod)
		// Add invoice_addons line items
		var invoices []models.Invoice
		h.db.Where("userId = ? AND status = 'PENDING' AND invoiceType = 'MONTHLY' AND DATE_FORMAT(dueDate, '%Y-%m') >= ?", userId, startPeriod).Find(&invoices)
		for _, inv := range invoices {
			var existing int64
			h.db.Model(&models.InvoiceAddonItem{}).Where("invoiceId = ? AND addonTypeId = ?", inv.ID, body.AddonTypeID).Count(&existing)
			if existing > 0 {
				continue
			}
			ia := models.InvoiceAddonItem{
				InvoiceID:   inv.ID,
				AddonTypeID: &body.AddonTypeID,
				AddonName:   addonType.Name,
				Amount:      effectivePrice,
			}
			h.db.Create(&ia)
		}
	}

	return c.Status(201).JSON(fiber.Map{"id": ca.ID, "message": "Addon berhasil diassign"})
}

// DELETE /api/customer-addons/:id
func (h *AddonHandler) RemoveCustomerAddon(c fiber.Ctx) error {
	id := c.Params("id")
	var ca models.CustomerAddon
	if err := h.db.First(&ca, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Addon tidak ditemukan"})
	}

	now := time.Now()
	h.db.Model(&ca).Update("endDate", now.Format("2006-01-02"))

	// Remove from unpaid invoices from current period onward
	currentPeriod := time.Now().Format("2006-01")
	var addonType models.AddonType
	h.db.First(&addonType, ca.AddonTypeID)
	if addonType.IsRecurring {
		effectivePrice := addonType.Price
		if ca.PriceOverride != nil {
			effectivePrice = *ca.PriceOverride
		}
		var invoiceAddons []models.InvoiceAddonItem
		h.db.Where("addonTypeId = ? AND invoiceId IN (SELECT id FROM invoices WHERE userId = ? AND status = 'PENDING' AND DATE_FORMAT(dueDate, '%Y-%m') >= ?)",
			ca.AddonTypeID, ca.UserID, currentPeriod).Find(&invoiceAddons)
		for _, ia := range invoiceAddons {
			h.db.Delete(&ia)
			h.db.Exec("UPDATE invoices SET addonAmount = GREATEST(0, addonAmount - ?), amount = GREATEST(0, amount - ?) WHERE id = ?",
				effectivePrice, effectivePrice, ia.InvoiceID)
		}
	}

	return c.JSON(fiber.Map{"message": "Addon dihentikan"})
}

// POST /api/billing/sync-addons
func (h *AddonHandler) SyncAddons(c fiber.Ctx) error {
	var invoices []models.Invoice
	h.db.Where("status = 'PENDING' AND invoiceType = 'MONTHLY'").Find(&invoices)

	fixed := 0
	for _, inv := range invoices {
		if inv.UserID == nil {
			continue
		}
		userId := *inv.UserID
		var addons []models.CustomerAddon
		h.db.Preload("AddonType").Where("userId = ? AND endDate IS NULL", userId).Find(&addons)

		addonTotal := 0
		for _, ca := range addons {
			if !ca.AddonType.IsRecurring {
				continue
			}
			effectivePrice := ca.AddonType.Price
			if ca.PriceOverride != nil {
				effectivePrice = *ca.PriceOverride
			}
			// Check if already in invoice_addons
			var existing int64
			h.db.Model(&models.InvoiceAddonItem{}).Where("invoiceId = ? AND addonTypeId = ?", inv.ID, ca.AddonTypeID).Count(&existing)
			if existing == 0 {
				ia := models.InvoiceAddonItem{
					InvoiceID:   inv.ID,
					AddonTypeID: &ca.AddonTypeID,
					AddonName:   ca.AddonType.Name,
					Amount:      effectivePrice,
				}
				h.db.Create(&ia)
				addonTotal += effectivePrice
			}
		}

		if addonTotal > 0 {
			h.db.Exec("UPDATE invoices SET addonAmount = addonAmount + ?, amount = amount + ? WHERE id = ?", addonTotal, addonTotal, inv.ID)
			fixed++
		}
	}

	return c.JSON(fiber.Map{"message": fmt.Sprintf("Synced %d invoices", fixed), "fixed": fixed})
}

// ─── Helper: get active recurring addons for a user (used by cron) ─────────────

func GetActiveAddonTotal(db *gorm.DB, userId string) (int, []models.InvoiceAddonItem, error) {
	var addons []models.CustomerAddon
	err := db.Preload("AddonType").Where("userId = ? AND endDate IS NULL", userId).Find(&addons).Error
	if err != nil {
		return 0, nil, err
	}
	total := 0
	var items []models.InvoiceAddonItem
	for _, ca := range addons {
		if !ca.AddonType.IsRecurring {
			continue
		}
		effectivePrice := ca.AddonType.Price
		if ca.PriceOverride != nil {
			effectivePrice = *ca.PriceOverride
		}
		total += effectivePrice
		items = append(items, models.InvoiceAddonItem{
			AddonTypeID: &ca.AddonTypeID,
			AddonName:   ca.AddonType.Name,
			Amount:      effectivePrice,
		})
	}
	return total, items, nil
}
