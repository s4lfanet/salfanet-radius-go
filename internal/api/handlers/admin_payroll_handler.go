package handlers

// admin_payroll_handler.go — payroll management
// GET /api/admin/payroll, PUT/DELETE /api/admin/payroll/:id
// POST /api/admin/payroll/generate, GET/POST /api/admin/payroll/overtime
// POST /api/admin/payroll/pay/:id

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminPayrollHandler manages employee payroll records.
type AdminPayrollHandler struct{ db *gorm.DB }

func NewAdminPayrollHandler(db *gorm.DB) *AdminPayrollHandler {
	return &AdminPayrollHandler{db: db}
}

type payrollRecord struct {
	ID         string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	EmployeeID string     `gorm:"index" json:"employeeId"`
	Month      string     `json:"month"` // YYYY-MM
	BaseWage   int        `json:"baseWage"`
	Allowance  int        `json:"allowance"`
	Deduction  int        `json:"deduction"`
	Bonus      int        `json:"bonus"`
	Overtime   int        `json:"overtime"`
	NetAmount  int        `json:"netAmount"`
	Status     string     `gorm:"default:DRAFT" json:"status"` // DRAFT, PAID, CANCELLED
	PaidAt     *time.Time `json:"paidAt"`
	Notes      *string    `gorm:"type:text" json:"notes"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

func (payrollRecord) TableName() string { return "payroll_records" }

type overtimeRecord struct {
	ID         string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	EmployeeID string    `gorm:"index" json:"employeeId"`
	Date       string    `json:"date"` // YYYY-MM-DD
	Hours      float64   `json:"hours"`
	Rate       int       `json:"rate"`
	Amount     int       `json:"amount"`
	Status     string    `gorm:"default:PENDING" json:"status"`
	Notes      *string   `gorm:"type:text" json:"notes"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (overtimeRecord) TableName() string { return "payroll_overtime" }

// GET /api/admin/payroll — list payroll records
func (h *AdminPayrollHandler) List(c fiber.Ctx) error {
	month := c.Query("month")
	status := c.Query("status")
	page, limit := pageParams(c)

	q := h.db.Model(&payrollRecord{})
	if month != "" {
		q = q.Where("month = ?", month)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var records []payrollRecord
	q.Order("createdAt desc").Offset((page - 1) * limit).Limit(limit).Find(&records)
	return c.JSON(fiber.Map{
		"success": true, "payroll": records,
		"pagination": fiber.Map{"page": page, "limit": limit, "total": total},
	})
}

// GET /api/admin/payroll/:id
func (h *AdminPayrollHandler) Get(c fiber.Ctx) error {
	var rec payrollRecord
	if err := h.db.First(&rec, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "payroll record not found"})
	}
	return c.JSON(fiber.Map{"success": true, "payroll": rec})
}

// PUT /api/admin/payroll/:id
func (h *AdminPayrollHandler) Update(c fiber.Ctx) error {
	var rec payrollRecord
	if err := h.db.First(&rec, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "payroll record not found"})
	}
	if rec.Status == "PAID" {
		return c.Status(400).JSON(fiber.Map{"error": "cannot edit a paid payroll record"})
	}
	var body map[string]interface{}
	c.Bind().JSON(&body)
	body["updatedAt"] = time.Now()
	h.db.Model(&rec).Updates(body)
	return c.JSON(fiber.Map{"success": true, "payroll": rec})
}

// DELETE /api/admin/payroll/:id
func (h *AdminPayrollHandler) Delete(c fiber.Ctx) error {
	var rec payrollRecord
	if err := h.db.First(&rec, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "payroll record not found"})
	}
	if rec.Status == "PAID" {
		return c.Status(400).JSON(fiber.Map{"error": "cannot delete a paid payroll record"})
	}
	h.db.Delete(&rec)
	return c.JSON(fiber.Map{"success": true, "message": "Payroll record deleted"})
}

// POST /api/admin/payroll/generate — generate payroll for a month
func (h *AdminPayrollHandler) Generate(c fiber.Ctx) error {
	var body struct {
		Month string `json:"month"` // YYYY-MM
	}
	if err := c.Bind().JSON(&body); err != nil || body.Month == "" {
		return c.Status(400).JSON(fiber.Map{"error": "month required (YYYY-MM)"})
	}
	// Stub: in production, this would iterate employees and create records
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Payroll generated for " + body.Month,
		"count":   0,
	})
}

// GET /api/admin/payroll/overtime — list overtime records
func (h *AdminPayrollHandler) ListOvertime(c fiber.Ctx) error {
	status := c.Query("status")
	page, limit := pageParams(c)
	q := h.db.Model(&overtimeRecord{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var records []overtimeRecord
	q.Order("date desc").Offset((page - 1) * limit).Limit(limit).Find(&records)
	return c.JSON(fiber.Map{
		"success": true, "overtime": records,
		"pagination": fiber.Map{"page": page, "limit": limit, "total": total},
	})
}

// POST /api/admin/payroll/overtime — record overtime
func (h *AdminPayrollHandler) CreateOvertime(c fiber.Ctx) error {
	var body overtimeRecord
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	if body.Hours > 0 && body.Rate > 0 {
		body.Amount = int(body.Hours * float64(body.Rate))
	}
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "overtime": body})
}

// POST /api/admin/payroll/pay/:id — mark payroll as paid
func (h *AdminPayrollHandler) Pay(c fiber.Ctx) error {
	var rec payrollRecord
	if err := h.db.First(&rec, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "payroll record not found"})
	}
	now := time.Now()
	h.db.Model(&rec).Updates(map[string]interface{}{
		"status":     "PAID",
		"paidAt":    now,
		"updatedAt": now,
	})
	return c.JSON(fiber.Map{"success": true, "message": "Payroll marked as paid"})
}
