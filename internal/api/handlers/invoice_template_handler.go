package handlers

// invoice_template_handler.go — CRUD for invoice templates
// GET/POST /api/invoice-templates, GET/PUT/DELETE /api/invoice-templates/:id, POST /api/invoice-templates/:id/default

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InvoiceTemplateHandler manages invoice HTML templates stored in email_templates.
type InvoiceTemplateHandler struct{ db *gorm.DB }

func NewInvoiceTemplateHandler(db *gorm.DB) *InvoiceTemplateHandler {
	return &InvoiceTemplateHandler{db: db}
}

// invoiceTemplate is a lightweight struct for the invoice_templates table (if it exists)
// or we use the email_templates table with type filter.
type invoiceTemplateRow struct {
	ID           string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name         string    `json:"name"`
	Subject      string    `json:"subject"`
	HtmlBody     string    `gorm:"type:text" json:"htmlBody"`
	IsDefault    bool      `gorm:"default:false" json:"isDefault"`
	TemplateType string    `gorm:"default:INVOICE" json:"type"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (invoiceTemplateRow) TableName() string { return "invoice_templates" }

// GET /api/invoice-templates
func (h *InvoiceTemplateHandler) List(c fiber.Ctx) error {
	var rows []invoiceTemplateRow
	h.db.Order("created_at desc").Find(&rows)
	return c.JSON(fiber.Map{"success": true, "templates": rows})
}

// POST /api/invoice-templates
func (h *InvoiceTemplateHandler) Create(c fiber.Ctx) error {
	var body invoiceTemplateRow
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	if body.TemplateType == "" {
		body.TemplateType = "INVOICE"
	}
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "template": body})
}

// GET /api/invoice-templates/:id
func (h *InvoiceTemplateHandler) Get(c fiber.Ctx) error {
	var row invoiceTemplateRow
	if err := h.db.First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "template not found"})
	}
	return c.JSON(fiber.Map{"success": true, "template": row})
}

// PUT /api/invoice-templates/:id
func (h *InvoiceTemplateHandler) Update(c fiber.Ctx) error {
	var row invoiceTemplateRow
	if err := h.db.First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "template not found"})
	}
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body["updated_at"] = time.Now()
	h.db.Model(&row).Updates(body)
	return c.JSON(fiber.Map{"success": true, "template": row})
}

// DELETE /api/invoice-templates/:id
func (h *InvoiceTemplateHandler) Delete(c fiber.Ctx) error {
	var row invoiceTemplateRow
	if err := h.db.First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "template not found"})
	}
	h.db.Delete(&row)
	return c.JSON(fiber.Map{"success": true, "message": "Template deleted"})
}

// POST /api/invoice-templates/:id/default — set as default template
func (h *InvoiceTemplateHandler) SetDefault(c fiber.Ctx) error {
	id := c.Params("id")
	// Unset all defaults
	h.db.Model(&invoiceTemplateRow{}).Where("is_default = ?", true).Update("is_default", false)
	// Set this one as default
	h.db.Model(&invoiceTemplateRow{}).Where("id = ?", id).Update("is_default", true)
	return c.JSON(fiber.Map{"success": true, "message": "Default template updated"})
}
