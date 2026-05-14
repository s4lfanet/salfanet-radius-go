package handlers

// troubleshooting_handler.go — troubleshooting checklists and job tracking
// GET /api/troubleshooting/checklists
// GET /api/troubleshooting/jobs, /jobs/:id, /jobs/:id/materials

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TroubleshootingHandler manages troubleshooting checklists and jobs.
type TroubleshootingHandler struct{ db *gorm.DB }

func NewTroubleshootingHandler(db *gorm.DB) *TroubleshootingHandler {
	return &TroubleshootingHandler{db: db}
}

type troubleshootingChecklist struct {
	ID          string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Title       string    `json:"title"`
	Description *string   `gorm:"type:text" json:"description"`
	Category    string    `json:"category"`
	Steps       string    `gorm:"type:text" json:"steps"`
	IsActive    bool      `gorm:"default:true" json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (troubleshootingChecklist) TableName() string { return "troubleshooting_checklists" }

type troubleshootingJob struct {
	ID           string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Title        string    `json:"title"`
	Description  *string   `gorm:"type:text" json:"description"`
	ChecklistID  *string   `json:"checklistId"`
	AssignedToID *string   `json:"assignedToId"`
	Status       string    `gorm:"default:OPEN" json:"status"`
	Priority     string    `gorm:"default:MEDIUM" json:"priority"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (troubleshootingJob) TableName() string { return "troubleshooting_jobs" }

type troubleshootingMaterial struct {
	ID      string `gorm:"primaryKey;type:varchar(191)" json:"id"`
	JobID   string `gorm:"index" json:"jobId"`
	Name    string `json:"name"`
	Qty     int    `json:"qty"`
	Unit    string `json:"unit"`
	Notes   string `json:"notes"`
}

func (troubleshootingMaterial) TableName() string { return "troubleshooting_materials" }

// GET /api/troubleshooting/checklists
func (h *TroubleshootingHandler) ListChecklists(c fiber.Ctx) error {
	var rows []troubleshootingChecklist
	h.db.Where("isActive = ?", true).Order("title").Find(&rows)
	return c.JSON(fiber.Map{"success": true, "checklists": rows})
}

// POST /api/troubleshooting/checklists
func (h *TroubleshootingHandler) CreateChecklist(c fiber.Ctx) error {
	var body troubleshootingChecklist
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.IsActive = true
	body.CreatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "checklist": body})
}

// GET /api/troubleshooting/jobs
func (h *TroubleshootingHandler) ListJobs(c fiber.Ctx) error {
	status := c.Query("status")
	page, limit := pageParams(c)
	q := h.db.Model(&troubleshootingJob{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var jobs []troubleshootingJob
	q.Order("createdAt desc").Offset((page - 1) * limit).Limit(limit).Find(&jobs)
	return c.JSON(fiber.Map{
		"success": true, "jobs": jobs,
		"pagination": fiber.Map{"page": page, "limit": limit, "total": total},
	})
}

// GET /api/troubleshooting/jobs/:id
func (h *TroubleshootingHandler) GetJob(c fiber.Ctx) error {
	var job troubleshootingJob
	if err := h.db.First(&job, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "job not found"})
	}
	return c.JSON(fiber.Map{"success": true, "job": job})
}

// GET /api/troubleshooting/jobs/:id/materials
func (h *TroubleshootingHandler) JobMaterials(c fiber.Ctx) error {
	jobID := c.Params("id")
	var materials []troubleshootingMaterial
	h.db.Where("job_id = ?", jobID).Find(&materials)
	return c.JSON(fiber.Map{"success": true, "materials": materials})
}
