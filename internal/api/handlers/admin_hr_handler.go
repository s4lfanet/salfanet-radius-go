package handlers

// admin_hr_handler.go — HR management: attendance, cash advances, commissions
// GET/POST /api/admin/attendance, POST /api/admin/attendance/bulk-delete
// GET/POST /api/admin/attendance-locations
// GET/POST /api/admin/cash-advances, GET/PUT/DELETE /api/admin/cash-advances/:id
// POST /api/admin/cash-advances/pay/:id
// GET/POST /api/admin/commissions, GET/PUT/DELETE /api/admin/commissions/:id
// POST /api/admin/commissions/:id/approve, /reject

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminHRHandler handles attendance, cash advances, and commission management.
type AdminHRHandler struct{ db *gorm.DB }

func NewAdminHRHandler(db *gorm.DB) *AdminHRHandler {
	return &AdminHRHandler{db: db}
}

// ─── Attendance ───────────────────────────────────────────────────────────────

type attendanceRecord struct {
	ID          string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	EmployeeID  string     `gorm:"index" json:"employeeId"`
	Date        string     `json:"date"`
	CheckIn     *time.Time `json:"checkIn"`
	CheckOut    *time.Time `json:"checkOut"`
	Status      string     `gorm:"default:PRESENT" json:"status"`
	Notes       *string    `gorm:"type:text" json:"notes"`
	LocationLat *float64   `json:"locationLat"`
	LocationLng *float64   `json:"locationLng"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (attendanceRecord) TableName() string { return "attendance_records" }

type attendanceLocation struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name      string    `json:"name"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Radius    int       `gorm:"default:100" json:"radius"` // meters
	IsActive  bool      `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
}

func (attendanceLocation) TableName() string { return "attendance_locations" }

// GET /api/admin/attendance
func (h *AdminHRHandler) ListAttendance(c fiber.Ctx) error {
	date := c.Query("date")
	employeeID := c.Query("employeeId")
	page, limit := pageParams(c)
	q := h.db.Model(&attendanceRecord{})
	if date != "" {
		q = q.Where("date = ?", date)
	}
	if employeeID != "" {
		q = q.Where("employeeId = ?", employeeID)
	}
	var total int64
	q.Count(&total)
	var records []attendanceRecord
	q.Order("date desc, checkIn desc").Offset((page - 1) * limit).Limit(limit).Find(&records)
	return c.JSON(fiber.Map{
		"success": true, "attendance": records,
		"pagination": fiber.Map{"page": page, "limit": limit, "total": total},
	})
}

// POST /api/admin/attendance — admin creates an attendance record
func (h *AdminHRHandler) CreateAttendance(c fiber.Ctx) error {
	var body attendanceRecord
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "attendance": body})
}

// POST /api/admin/attendance/bulk-delete
func (h *AdminHRHandler) BulkDeleteAttendance(c fiber.Ctx) error {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := c.Bind().JSON(&body); err != nil || len(body.IDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "ids required"})
	}
	h.db.Where("id IN ?", body.IDs).Delete(&attendanceRecord{})
	return c.JSON(fiber.Map{"success": true, "message": "Deleted"})
}

// GET /api/admin/attendance-locations
func (h *AdminHRHandler) ListLocations(c fiber.Ctx) error {
	var locations []attendanceLocation
	h.db.Where("isActive = ?", true).Order("name").Find(&locations)
	return c.JSON(fiber.Map{"success": true, "locations": locations})
}

// POST /api/admin/attendance-locations
func (h *AdminHRHandler) CreateLocation(c fiber.Ctx) error {
	var body attendanceLocation
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.IsActive = true
	body.CreatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "location": body})
}

// ─── Cash Advances ────────────────────────────────────────────────────────────

type cashAdvance struct {
	ID           string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	EmployeeID   string     `gorm:"index" json:"employeeId"`
	Amount       int        `json:"amount"`
	Reason       string     `gorm:"type:text" json:"reason"`
	Status       string     `gorm:"default:PENDING" json:"status"` // PENDING, APPROVED, REJECTED, PAID
	ApprovedBy   *string    `json:"approvedBy"`
	ApprovedAt   *time.Time `json:"approvedAt"`
	PaidAt       *time.Time `json:"paidAt"`
	Installments int        `gorm:"default:1" json:"installments"`
	Notes        *string    `gorm:"type:text" json:"notes"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (cashAdvance) TableName() string { return "cash_advances" }

// GET /api/admin/cash-advances
func (h *AdminHRHandler) ListCashAdvances(c fiber.Ctx) error {
	status := c.Query("status")
	page, limit := pageParams(c)
	q := h.db.Model(&cashAdvance{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var advances []cashAdvance
	q.Order("createdAt desc").Offset((page - 1) * limit).Limit(limit).Find(&advances)
	return c.JSON(fiber.Map{
		"success": true, "advances": advances,
		"pagination": fiber.Map{"page": page, "limit": limit, "total": total},
	})
}

// POST /api/admin/cash-advances
func (h *AdminHRHandler) CreateCashAdvance(c fiber.Ctx) error {
	var body cashAdvance
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.Status = "PENDING"
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "advance": body})
}

// GET /api/admin/cash-advances/:id
func (h *AdminHRHandler) GetCashAdvance(c fiber.Ctx) error {
	var adv cashAdvance
	if err := h.db.First(&adv, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cash advance not found"})
	}
	return c.JSON(fiber.Map{"success": true, "advance": adv})
}

// PUT /api/admin/cash-advances/:id
func (h *AdminHRHandler) UpdateCashAdvance(c fiber.Ctx) error {
	var adv cashAdvance
	if err := h.db.First(&adv, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cash advance not found"})
	}
	var body map[string]interface{}
	c.Bind().JSON(&body)
	body["updatedAt"] = time.Now()
	h.db.Model(&adv).Updates(body)
	return c.JSON(fiber.Map{"success": true, "advance": adv})
}

// DELETE /api/admin/cash-advances/:id
func (h *AdminHRHandler) DeleteCashAdvance(c fiber.Ctx) error {
	h.db.Where("id = ?", c.Params("id")).Delete(&cashAdvance{})
	return c.JSON(fiber.Map{"success": true, "message": "Cash advance deleted"})
}

// POST /api/admin/cash-advances/pay/:id — mark advance as paid
func (h *AdminHRHandler) PayCashAdvance(c fiber.Ctx) error {
	now := time.Now()
	h.db.Model(&cashAdvance{}).Where("id = ?", c.Params("id")).
		Updates(map[string]interface{}{"status": "PAID", "paidAt": now, "updatedAt": now})
	return c.JSON(fiber.Map{"success": true, "message": "Cash advance marked as paid"})
}

// ─── Commissions ──────────────────────────────────────────────────────────────

type commission struct {
	ID          string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	EmployeeID  string     `gorm:"index" json:"employeeId"`
	Type        string     `json:"type"` // INSTALLATION, SALES, REFERRAL
	Amount      int        `json:"amount"`
	Description string     `gorm:"type:text" json:"description"`
	Status      string     `gorm:"default:PENDING" json:"status"`
	ApprovedBy  *string    `json:"approvedBy"`
	ApprovedAt  *time.Time `json:"approvedAt"`
	PaidAt      *time.Time `json:"paidAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (commission) TableName() string { return "commissions" }

// GET /api/admin/commissions
func (h *AdminHRHandler) ListCommissions(c fiber.Ctx) error {
	status := c.Query("status")
	page, limit := pageParams(c)
	q := h.db.Model(&commission{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var commissions []commission
	q.Order("createdAt desc").Offset((page - 1) * limit).Limit(limit).Find(&commissions)
	return c.JSON(fiber.Map{
		"success": true, "commissions": commissions,
		"pagination": fiber.Map{"page": page, "limit": limit, "total": total},
	})
}

// POST /api/admin/commissions
func (h *AdminHRHandler) CreateCommission(c fiber.Ctx) error {
	var body commission
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.Status = "PENDING"
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "commission": body})
}

// GET /api/admin/commissions/:id
func (h *AdminHRHandler) GetCommission(c fiber.Ctx) error {
	var comm commission
	if err := h.db.First(&comm, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "commission not found"})
	}
	return c.JSON(fiber.Map{"success": true, "commission": comm})
}

// PUT /api/admin/commissions/:id
func (h *AdminHRHandler) UpdateCommission(c fiber.Ctx) error {
	var comm commission
	if err := h.db.First(&comm, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "commission not found"})
	}
	var body map[string]interface{}
	c.Bind().JSON(&body)
	body["updatedAt"] = time.Now()
	h.db.Model(&comm).Updates(body)
	return c.JSON(fiber.Map{"success": true, "commission": comm})
}

// DELETE /api/admin/commissions/:id
func (h *AdminHRHandler) DeleteCommission(c fiber.Ctx) error {
	h.db.Where("id = ?", c.Params("id")).Delete(&commission{})
	return c.JSON(fiber.Map{"success": true, "message": "Commission deleted"})
}

// POST /api/admin/commissions/:id/approve
func (h *AdminHRHandler) ApproveCommission(c fiber.Ctx) error {
	now := time.Now()
	h.db.Model(&commission{}).Where("id = ?", c.Params("id")).
		Updates(map[string]interface{}{"status": "APPROVED", "approvedAt": now, "updatedAt": now})
	return c.JSON(fiber.Map{"success": true, "message": "Commission approved"})
}

// POST /api/admin/commissions/:id/reject
func (h *AdminHRHandler) RejectCommission(c fiber.Ctx) error {
	h.db.Model(&commission{}).Where("id = ?", c.Params("id")).
		Updates(map[string]interface{}{"status": "REJECTED", "updatedAt": time.Now()})
	return c.JSON(fiber.Map{"success": true, "message": "Commission rejected"})
}
