package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/config"
	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// WhatsappHandler handles WhatsApp provider and template management.
type WhatsappHandler struct {
	db         *gorm.DB
	httpClient *http.Client
}

func NewWhatsappHandler(db *gorm.DB) *WhatsappHandler {
	return &WhatsappHandler{
		db:         db,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// ─── Providers ───────────────────────────────────────────────────────────────

func (h *WhatsappHandler) ListProviders(c fiber.Ctx) error {
	var providers []models.WhatsappProvider
	h.db.Order("priority").Find(&providers)
	return c.JSON(providers)
}

func (h *WhatsappHandler) CreateProvider(c fiber.Ctx) error {
	var body models.WhatsappProvider
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.ID = uuid.New().String()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}

func (h *WhatsappHandler) UpdateProvider(c fiber.Ctx) error {
	id := c.Params("id")
	var p models.WhatsappProvider
	if err := h.db.First(&p, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if err := c.Bind().JSON(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	h.db.Save(&p)
	return c.JSON(p)
}

// ─── Templates ───────────────────────────────────────────────────────────────

func (h *WhatsappHandler) ListTemplates(c fiber.Ctx) error {
	var templates []models.WhatsappTemplate
	h.db.Order("type").Find(&templates)
	return c.JSON(fiber.Map{"success": true, "data": templates})
}

func (h *WhatsappHandler) UpdateTemplate(c fiber.Ctx) error {
	id := c.Params("type") // frontend sends UUID via :id, route param is named :type
	var tmpl models.WhatsappTemplate
	// Try lookup by UUID first, then by type string (legacy)
	if err := h.db.First(&tmpl, "id = ?", id).Error; err != nil {
		if err2 := h.db.Where("type = ?", id).First(&tmpl).Error; err2 != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
		}
	}
	if err := c.Bind().JSON(&tmpl); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	h.db.Save(&tmpl)
	return c.JSON(fiber.Map{"success": true, "template": tmpl})
}

// ─── Send Manual ─────────────────────────────────────────────────────────────

func (h *WhatsappHandler) SendMessage(c fiber.Ctx) error {
	var body struct {
		Phone   string `json:"phone"`
		Message string `json:"message"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	payload, _ := json.Marshal(map[string]string{
		"phone":   body.Phone,
		"message": body.Message,
	})
	resp, err := h.httpClient.Post(config.C.WAServiceURL+"/send", "application/json", bytes.NewReader(payload))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Record in history
	status := "sent"
	if resp.StatusCode >= 400 {
		status = "failed"
	}
	errStr := ""
	hist := models.WhatsappHistory{
		ID:      uuid.New().String(),
		Phone:   body.Phone,
		Message: body.Message,
		Status:  status,
		SentAt:  time.Now(),
	}
	if status == "failed" {
		errStr = string(respBody)
		hist.Error = &errStr
	}
	h.db.Create(&hist)

	if resp.StatusCode >= 400 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": string(respBody)})
	}
	return c.JSON(fiber.Map{"success": true, "message": "sent", "provider": "whatsapp"})
}

// ─── History ─────────────────────────────────────────────────────────────────

func (h *WhatsappHandler) ListHistory(c fiber.Ctx) error {
	var history []models.WhatsappHistory
	h.db.Order("sentAt DESC").Limit(200).Find(&history)
	return c.JSON(history)
}

// ─── Reminder Settings ───────────────────────────────────────────────────────

func (h *WhatsappHandler) GetReminderSettings(c fiber.Ctx) error {
	var gs models.WhatsappGlobalSettings
	if err := h.db.First(&gs).Error; err != nil {
		// No row yet — return defaults
		return c.JSON(fiber.Map{
			"success": true,
			"settings": fiber.Map{
				"enabled":      true,
				"reminderDays": []int{-7, -5, -3, 0},
				"reminderTime": "09:00",
				"otpEnabled":   true,
				"otpExpiry":    5,
				"batchSize":    10,
				"batchDelay":   60,
				"randomize":    true,
			},
		})
	}
	var days []int
	_ = json.Unmarshal([]byte(gs.ReminderDays), &days)
	return c.JSON(fiber.Map{
		"success": true,
		"settings": fiber.Map{
			"id":           gs.ID,
			"enabled":      gs.Enabled,
			"reminderDays": days,
			"reminderTime": gs.ReminderTime,
			"otpEnabled":   gs.OtpEnabled,
			"otpExpiry":    gs.OtpExpiry,
			"batchSize":    gs.BatchSize,
			"batchDelay":   gs.BatchDelay,
			"randomize":    gs.Randomize,
		},
	})
}

func (h *WhatsappHandler) UpdateReminderSettings(c fiber.Ctx) error {
	var body struct {
		Enabled      bool   `json:"enabled"`
		ReminderDays []int  `json:"reminderDays"`
		ReminderTime string `json:"reminderTime"`
		OtpEnabled   bool   `json:"otpEnabled"`
		OtpExpiry    int    `json:"otpExpiry"`
		BatchSize    int    `json:"batchSize"`
		BatchDelay   int    `json:"batchDelay"`
		Randomize    bool   `json:"randomize"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	daysJSON, _ := json.Marshal(body.ReminderDays)
	var gs models.WhatsappGlobalSettings
	if err := h.db.First(&gs).Error; err != nil {
		// Create new record
		gs = models.WhatsappGlobalSettings{
			ID:           uuid.New().String(),
			Enabled:      body.Enabled,
			ReminderDays: string(daysJSON),
			ReminderTime: body.ReminderTime,
			OtpEnabled:   body.OtpEnabled,
			OtpExpiry:    body.OtpExpiry,
			BatchSize:    body.BatchSize,
			BatchDelay:   body.BatchDelay,
			Randomize:    body.Randomize,
		}
		h.db.Create(&gs)
	} else {
		gs.Enabled = body.Enabled
		gs.ReminderDays = string(daysJSON)
		gs.ReminderTime = body.ReminderTime
		gs.OtpEnabled = body.OtpEnabled
		gs.OtpExpiry = body.OtpExpiry
		gs.BatchSize = body.BatchSize
		gs.BatchDelay = body.BatchDelay
		gs.Randomize = body.Randomize
		h.db.Save(&gs)
	}
	return c.JSON(fiber.Map{"success": true, "message": "saved"})
}
