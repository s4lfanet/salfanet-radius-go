package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// TerritoryHandler handles territory and collector management endpoints.
type TerritoryHandler struct {
	db *gorm.DB
}

func NewTerritoryHandler(db *gorm.DB) *TerritoryHandler {
	return &TerritoryHandler{db: db}
}

// ─── Territory CRUD ──────────────────────────────────────────────────────────

// GET /api/territories — list all territories with collector info
func (h *TerritoryHandler) ListTerritories(c fiber.Ctx) error {
	var territories []models.Territory
	query := h.db.Preload("Collector").Preload("Areas")

	if isActive := c.Query("isActive"); isActive != "" {
		query = query.Where("isActive = ?", isActive == "true")
	}

	if err := query.Order("name ASC").Find(&territories).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Enrich with user count per territory
	type result struct {
		models.Territory
		UserCount int64 `json:"userCount"`
	}
	var results []result
	for _, t := range territories {
		var count int64
		h.db.Model(&models.PppoeUser{}).Where("territoryId = ?", t.ID).Count(&count)
		results = append(results, result{Territory: t, UserCount: count})
	}

	return c.JSON(fiber.Map{"data": results})
}

// GET /api/territories/:id — get single territory
func (h *TerritoryHandler) GetTerritory(c fiber.Ctx) error {
	id := c.Params("id")
	var territory models.Territory
	if err := h.db.Preload("Collector").Preload("Areas").First(&territory, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Territory not found"})
	}
	return c.JSON(territory)
}

// POST /api/territories — create territory
func (h *TerritoryHandler) CreateTerritory(c fiber.Ctx) error {
	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
		CollectorID *string `json:"collectorId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}

	territory := models.Territory{
		ID:          uuid.New().String(),
		Name:        body.Name,
		Description: body.Description,
		CollectorID: body.CollectorID,
		IsActive:    true,
	}

	if err := h.db.Create(&territory).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(territory)
}

// PUT /api/territories/:id — update territory
func (h *TerritoryHandler) UpdateTerritory(c fiber.Ctx) error {
	id := c.Params("id")
	var territory models.Territory
	if err := h.db.First(&territory, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Territory not found"})
	}

	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		CollectorID *string `json:"collectorId"`
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
	if body.CollectorID != nil {
		updates["collectorId"] = *body.CollectorID
	}
	if body.IsActive != nil {
		updates["isActive"] = *body.IsActive
	}

	if err := h.db.Model(&territory).Updates(updates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(territory)
}

// DELETE /api/territories/:id — soft delete (set isActive=false)
func (h *TerritoryHandler) DeleteTerritory(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.db.Model(&models.Territory{}).Where("id = ?", id).Update("isActive", false).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Territory deactivated"})
}

// ─── Territory Areas ─────────────────────────────────────────────────────────

// GET /api/territories/:id/areas — list areas for a territory
func (h *TerritoryHandler) ListAreas(c fiber.Ctx) error {
	territoryID := c.Params("id")
	var areas []models.TerritoryArea
	if err := h.db.Where("territoryId = ?", territoryID).Order("kelurahanNama ASC").Find(&areas).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": areas})
}

// POST /api/territories/:id/areas — add area to territory
func (h *TerritoryHandler) AddArea(c fiber.Ctx) error {
	territoryID := c.Params("id")
	var body struct {
		KelurahanKode *string `json:"kelurahanKode"`
		KelurahanNama *string `json:"kelurahanNama"`
		KecamatanNama *string `json:"kecamatanNama"`
		KabupatenNama *string `json:"kabupatenNama"`
		ProvinsiNama  *string `json:"provinsiNama"`
		DusunNama     *string `json:"dusunNama"`
		CollectorID   *string `json:"collectorId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	area := models.TerritoryArea{
		ID:            uuid.New().String(),
		TerritoryID:   territoryID,
		KelurahanKode: body.KelurahanKode,
		KelurahanNama: body.KelurahanNama,
		KecamatanNama: body.KecamatanNama,
		KabupatenNama: body.KabupatenNama,
		ProvinsiNama:  body.ProvinsiNama,
		DusunNama:     body.DusunNama,
		CollectorID:   body.CollectorID,
	}

	if err := h.db.Create(&area).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(area)
}

// DELETE /api/territories/:id/areas/:areaId — remove area from territory
func (h *TerritoryHandler) RemoveArea(c fiber.Ctx) error {
	areaID := c.Params("areaId")
	if err := h.db.Delete(&models.TerritoryArea{}, "id = ?", areaID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Area removed"})
}

// ─── Collector List ──────────────────────────────────────────────────────────

// GET /api/territories/collectors — list all users with COLLECTOR role
func (h *TerritoryHandler) ListCollectors(c fiber.Ctx) error {
	var collectors []models.User
	if err := h.db.Where("role = ?", models.RoleCollector).Order("name ASC").Find(&collectors).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": collectors})
}

// ─── Settlement Report ───────────────────────────────────────────────────────

// GET /api/settlements?collectorId=&date= — daily settlement report
func (h *TerritoryHandler) GetSettlement(c fiber.Ctx) error {
	collectorID := c.Query("collectorId")
	dateStr := c.Query("date")

	if collectorID == "" || dateStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "collectorId and date are required"})
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid date format, use YYYY-MM-DD"})
	}

	// Get all PAID invoices for this collector on this date
	type InvoiceRow struct {
		ID            string     `json:"id"`
		InvoiceNumber string     `json:"invoiceNumber"`
		Amount        int        `json:"amount"`
		CustomerName  *string    `json:"customerName"`
		PaidAt        *time.Time `json:"paidAt"`
		Method        string     `json:"method"`
	}

	var invoices []InvoiceRow
	query := h.db.Table("invoices AS i").
		Select("i.id, i.invoiceNumber, i.amount, i.customerName, i.paidAt, p.method").
		Joins("LEFT JOIN payments p ON p.invoiceId = i.id").
		Where("i.status = ?", string(models.InvoicePaid)).
		Where("DATE(i.paidAt) = ?", date.Format("2006-01-02"))

	// Filter by collector's territory users
	query = query.Joins("LEFT JOIN pppoe_users u ON u.id = i.userId").
		Where("u.territoryId IN (SELECT id FROM territories WHERE collectorId = ?)", collectorID)

	if err := query.Order("i.paidAt DESC").Scan(&invoices).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	totalAmount := 0
	for _, inv := range invoices {
		totalAmount += inv.Amount
	}

	// Check if settlement already exists
	var settlement models.Settlement
	hasSettlement := h.db.Where("collectorId = ? AND periodDate = ?", collectorID, date).First(&settlement).Error == nil

	return c.JSON(fiber.Map{
		"date":           dateStr,
		"collectorId":    collectorID,
		"totalAmount":    totalAmount,
		"invoiceCount":   len(invoices),
		"invoices":       invoices,
		"settlement":     hasSettlement,
		"settlementData": settlement,
	})
}

// GET /api/settlements/range?from=&to=&collectorId= — range settlement report
func (h *TerritoryHandler) GetSettlementRange(c fiber.Ctx) error {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	collectorID := c.Query("collectorId")

	if fromStr == "" || toStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "from and to are required"})
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid from date"})
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid to date"})
	}

	// Build query for paid invoices in date range
	query := h.db.Table("invoices AS i").
		Select("DATE(i.paidAt) as date, COUNT(*) as count, SUM(i.amount) as total").
		Where("i.status = ?", string(models.InvoicePaid)).
		Where("DATE(i.paidAt) BETWEEN ? AND ?", from.Format("2006-01-02"), to.Format("2006-01-02"))

	if collectorID != "" {
		query = query.Joins("LEFT JOIN pppoe_users u ON u.id = i.userId").
			Where("u.territoryId IN (SELECT id FROM territories WHERE collectorId = ?)", collectorID)
	}

	type DaySummary struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
		Total int64  `json:"total"`
	}
	var summaries []DaySummary
	if err := query.Group("DATE(i.paidAt)").Order("date ASC").Scan(&summaries).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Get collector summary if collectorID is provided
	var collectorSummaries []fiber.Map
	if collectorID == "" {
		// Get per-collector summary
		collectorQuery := h.db.Table("invoices AS i").
			Select("t.collectorId, u_collector.name as collectorName, COUNT(*) as count, SUM(i.amount) as total").
			Joins("LEFT JOIN pppoe_users u ON u.id = i.userId").
			Joins("LEFT JOIN territories t ON t.id = u.territoryId").
			Joins("LEFT JOIN users u_collector ON u_collector.id = t.collectorId").
			Where("i.status = ?", string(models.InvoicePaid)).
			Where("DATE(i.paidAt) BETWEEN ? AND ?", from.Format("2006-01-02"), to.Format("2006-01-02")).
			Where("t.collectorId IS NOT NULL").
			Group("t.collectorId, u_collector.name")

		type CollectorSummary struct {
			CollectorID   string  `json:"collectorId"`
			CollectorName *string `json:"collectorName"`
			Count         int64   `json:"count"`
			Total         int64   `json:"total"`
		}
		var cs []CollectorSummary
		if err := collectorQuery.Scan(&cs).Error; err == nil {
			for _, s := range cs {
				collectorSummaries = append(collectorSummaries, fiber.Map{
					"collectorId":   s.CollectorID,
					"collectorName": s.CollectorName,
					"count":         s.Count,
					"total":         s.Total,
				})
			}
		}
	}

	return c.JSON(fiber.Map{
		"from":       fromStr,
		"to":         toStr,
		"daily":      summaries,
		"collectors": collectorSummaries,
	})
}

// POST /api/settlements/confirm — confirm a settlement
func (h *TerritoryHandler) ConfirmSettlement(c fiber.Ctx) error {
	var body struct {
		CollectorID string `json:"collectorId"`
		Date        string `json:"date"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.CollectorID == "" || body.Date == "" {
		return c.Status(400).JSON(fiber.Map{"error": "collectorId and date are required"})
	}

	date, err := time.Parse("2006-01-02", body.Date)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid date format"})
	}

	// Calculate totals
	var totalAmount int64
	var invoiceCount int64
	h.db.Table("invoices AS i").
		Joins("LEFT JOIN pppoe_users u ON u.id = i.userId").
		Where("i.status = ?", string(models.InvoicePaid)).
		Where("DATE(i.paidAt) = ?", date.Format("2006-01-02")).
		Where("u.territoryId IN (SELECT id FROM territories WHERE collectorId = ?)", body.CollectorID).
		Count(&invoiceCount).
		Select("COALESCE(SUM(i.amount), 0)").
		Scan(&totalAmount)

	// Get user ID from context for confirmedBy
	userID := c.Locals("userId")
	confirmedBy, _ := userID.(string)

	// Upsert settlement
	settlement := models.Settlement{
		ID:           uuid.New().String(),
		CollectorID:  body.CollectorID,
		PeriodDate:   date,
		TotalAmount:  int(totalAmount),
		InvoiceCount: int(invoiceCount),
		Status:       "confirmed",
		ConfirmedBy:  &confirmedBy,
		ConfirmedAt:  &[]time.Time{time.Now()}[0],
	}

	// Try to find existing
	var existing models.Settlement
	if h.db.Where("collectorId = ? AND periodDate = ?", body.CollectorID, date).First(&existing).Error == nil {
		// Update existing
		settlement.ID = existing.ID
		h.db.Model(&existing).Updates(map[string]interface{}{
			"status":       "confirmed",
			"totalAmount":  int(totalAmount),
			"invoiceCount": int(invoiceCount),
			"confirmedBy":  confirmedBy,
			"confirmedAt":  time.Now(),
		})
	} else {
		// Create new
		if err := h.db.Create(&settlement).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.JSON(fiber.Map{
		"message":    "Settlement confirmed",
		"settlement": settlement,
	})
}

// GET /api/settlements/list — list all settlements with pagination
func (h *TerritoryHandler) ListSettlements(c fiber.Ctx) error {
	page, pageSize := pageParams(c)
	var settlements []models.Settlement
	query := h.db.Preload("Collector")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if collectorID := c.Query("collectorId"); collectorID != "" {
		query = query.Where("collectorId = ?", collectorID)
	}

	var total int64
	query.Model(&models.Settlement{}).Count(&total)
	query.Order("periodDate DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&settlements)

	return c.JSON(fiber.Map{
		"data":     settlements,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
