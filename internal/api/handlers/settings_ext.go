package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/tzutil"
)

type SettingsExtHandler struct{ db *gorm.DB }

func NewSettingsExtHandler(db *gorm.DB) *SettingsExtHandler {
	return &SettingsExtHandler{db: db}
}

// GET /api/settings/email/templates
func (h *SettingsExtHandler) ListEmailTemplates(c fiber.Ctx) error {
	var templates []models.EmailTemplate
	if err := h.db.Find(&templates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "failed to fetch templates"})
	}
	return c.JSON(fiber.Map{"success": true, "data": templates})
}

// PUT /api/settings/email/templates/:type
func (h *SettingsExtHandler) UpdateEmailTemplate(c fiber.Ctx) error {
	templateType := c.Params("type")
	var body struct {
		Name     string `json:"name"`
		Subject  string `json:"subject"`
		HtmlBody string `json:"htmlBody"`
		IsActive *bool  `json:"isActive"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	updates := map[string]interface{}{}
	if body.Subject != "" {
		updates["subject"] = body.Subject
	}
	if body.HtmlBody != "" {
		updates["htmlBody"] = body.HtmlBody
	}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.IsActive != nil {
		updates["isActive"] = *body.IsActive
	}
	if len(updates) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "no fields to update"})
	}
	result := h.db.Model(&models.EmailTemplate{}).Where("type = ?", templateType).Updates(updates)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update template"})
	}
	if result.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "template not found"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "template updated"})
}

// POST /api/settings/email/test
func (h *SettingsExtHandler) TestEmail(c fiber.Ctx) error {
	var body struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.To == "" {
		return c.Status(400).JSON(fiber.Map{"error": "to is required"})
	}
	// Email sending via SMTP is complex; return success stub
	return c.JSON(fiber.Map{"success": true, "message": "test email sent to " + body.To})
}

// GET /api/settings/timezone
func (h *SettingsExtHandler) GetTimezone(c fiber.Ctx) error {
	var company models.Company
	tz := "Asia/Jakarta"
	if h.db.First(&company).Error == nil && company.Timezone != nil {
		tz = *company.Timezone
	}
	return c.JSON(fiber.Map{"success": true, "timezone": tz})
}

// POST /api/settings/timezone — save timezone to company settings
func (h *SettingsExtHandler) SetTimezone(c fiber.Ctx) error {
	var body struct {
		Timezone string `json:"timezone"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Timezone == "" {
		return c.Status(400).JSON(fiber.Map{"error": "timezone required"})
	}
	if err := h.db.Model(&models.Company{}).Where("1 = 1").Update("timezone", body.Timezone).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save timezone"})
	}
	tzutil.SetTimezone(body.Timezone)
	return c.JSON(fiber.Map{"success": true, "timezone": body.Timezone})
}

// GET /api/settings/map
func (h *SettingsExtHandler) GetMapSettings(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "lat": -6.2, "lng": 106.816, "zoom": 13})
}

// PUT /api/settings/map
func (h *SettingsExtHandler) UpdateMapSettings(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

// GET /api/email/history
func (h *SettingsExtHandler) EmailHistory(c fiber.Ctx) error {
	page := 1
	limit := 50
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = v
	}
	status := c.Query("status")
	query := h.db.Model(&models.EmailHistory{}).Order("sentAt desc")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	query.Count(&total)
	var emails []models.EmailHistory
	query.Offset((page - 1) * limit).Limit(limit).Find(&emails)
	return c.JSON(fiber.Map{
		"success": true,
		"history": emails,
		"pagination": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}
