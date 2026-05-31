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
	ID    string `gorm:"primaryKey;type:varchar(191)" json:"id"`
	JobID string `gorm:"index" json:"jobId"`
	Name  string `json:"name"`
	Qty   int    `json:"qty"`
	Unit  string `json:"unit"`
	Notes string `json:"notes"`
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
	h.db.Where("jobId = ?", jobID).Find(&materials)
	return c.JSON(fiber.Map{"success": true, "materials": materials})
}

// PUT /api/troubleshooting/checklists/:id
func (h *TroubleshootingHandler) UpdateChecklist(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
		Category    string  `json:"category"`
		Steps       string  `json:"steps"`
		IsActive    *bool   `json:"isActive"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	upd := map[string]any{}
	if body.Title != "" {
		upd["title"] = body.Title
	}
	if body.Description != nil {
		upd["description"] = body.Description
	}
	if body.Category != "" {
		upd["category"] = body.Category
	}
	if body.Steps != "" {
		upd["steps"] = body.Steps
	}
	if body.IsActive != nil {
		upd["isActive"] = body.IsActive
	}
	if err := h.db.Model(&troubleshootingChecklist{}).Where("id = ?", id).Updates(upd).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update checklist"})
	}
	var row troubleshootingChecklist
	h.db.Where("id = ?", id).First(&row)
	return c.JSON(fiber.Map{"success": true, "checklist": row})
}

// DELETE /api/troubleshooting/checklists/:id
func (h *TroubleshootingHandler) DeleteChecklist(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		id = c.Query("id")
	}
	if err := h.db.Delete(&troubleshootingChecklist{}, "id = ?", id).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete checklist"})
	}
	return c.JSON(fiber.Map{"success": true})
}

// POST /api/troubleshooting/jobs
func (h *TroubleshootingHandler) CreateJob(c fiber.Ctx) error {
	var body struct {
		Title        string  `json:"title"`
		Description  *string `json:"description"`
		ChecklistID  *string `json:"checklistId"`
		AssignedToID *string `json:"assignedToId"`
		Status       string  `json:"status"`
		Priority     string  `json:"priority"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Title == "" {
		return c.Status(400).JSON(fiber.Map{"error": "title is required"})
	}
	if body.Status == "" {
		body.Status = "OPEN"
	}
	if body.Priority == "" {
		body.Priority = "MEDIUM"
	}
	job := troubleshootingJob{
		ID:           uuid.New().String(),
		Title:        body.Title,
		Description:  body.Description,
		ChecklistID:  body.ChecklistID,
		AssignedToID: body.AssignedToID,
		Status:       body.Status,
		Priority:     body.Priority,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := h.db.Create(&job).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create job"})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "job": job})
}
