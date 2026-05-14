package handlers

// payroll_template_handler.go — CRUD for payroll templates
// GET/POST /api/payroll-templates, GET/PUT/DELETE /api/payroll-templates/:id, POST /api/payroll-templates/:id/default

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PayrollTemplateHandler manages payroll calculation templates.
type PayrollTemplateHandler struct{ db *gorm.DB }

func NewPayrollTemplateHandler(db *gorm.DB) *PayrollTemplateHandler {
	return &PayrollTemplateHandler{db: db}
}

type payrollTemplateRow struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name      string    `json:"name"`
	BaseWage  int       `json:"baseWage"`
	Allowance int       `json:"allowance"`
	Deduction int       `json:"deduction"`
	Notes     *string   `gorm:"type:text" json:"notes"`
	IsDefault bool      `gorm:"default:false" json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (payrollTemplateRow) TableName() string { return "payroll_templates" }

// GET /api/payroll-templates
func (h *PayrollTemplateHandler) List(c fiber.Ctx) error {
	var rows []payrollTemplateRow
	h.db.Order("createdAt desc").Find(&rows)
	return c.JSON(fiber.Map{"success": true, "templates": rows})
}

// POST /api/payroll-templates
func (h *PayrollTemplateHandler) Create(c fiber.Ctx) error {
	var body payrollTemplateRow
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "template": body})
}

// GET /api/payroll-templates/:id
func (h *PayrollTemplateHandler) Get(c fiber.Ctx) error {
	var row payrollTemplateRow
	if err := h.db.First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "template not found"})
	}
	return c.JSON(fiber.Map{"success": true, "template": row})
}

// PUT /api/payroll-templates/:id
func (h *PayrollTemplateHandler) Update(c fiber.Ctx) error {
	var row payrollTemplateRow
	if err := h.db.First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "template not found"})
	}
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body["updatedAt"] = time.Now()
	h.db.Model(&row).Updates(body)
	return c.JSON(fiber.Map{"success": true, "template": row})
}

// DELETE /api/payroll-templates/:id
func (h *PayrollTemplateHandler) Delete(c fiber.Ctx) error {
	var row payrollTemplateRow
	if err := h.db.First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "template not found"})
	}
	h.db.Delete(&row)
	return c.JSON(fiber.Map{"success": true, "message": "Template deleted"})
}

// POST /api/payroll-templates/:id/default
func (h *PayrollTemplateHandler) SetDefault(c fiber.Ctx) error {
	id := c.Params("id")
	h.db.Model(&payrollTemplateRow{}).Where("isDefault = ?", true).Update("isDefault", false)
	h.db.Model(&payrollTemplateRow{}).Where("id = ?", id).Update("isDefault", true)
	return c.JSON(fiber.Map{"success": true, "message": "Default payroll template updated"})
}
