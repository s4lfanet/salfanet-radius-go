package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

type ActivityLogHandler struct{ db *gorm.DB }

func NewActivityLogHandler(db *gorm.DB) *ActivityLogHandler {
	return &ActivityLogHandler{db: db}
}

// GET /api/admin/activity-logs
func (h *ActivityLogHandler) List(c fiber.Ctx) error {
	limit := 20
	offset := 0
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = v
	}
	if v, err := strconv.Atoi(c.Query("offset")); err == nil && v >= 0 {
		offset = v
	}
	module := c.Query("module")
	status := c.Query("status")
	search := c.Query("search")

	query := h.db.Model(&models.ActivityLog{}).Order("createdAt desc")
	if module != "" && module != "all" {
		query = query.Where("module = ?", module)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		query = query.Where("username LIKE ? OR action LIKE ? OR description LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var logs []models.ActivityLog
	query.Offset(offset).Limit(limit).Find(&logs)

	return c.JSON(fiber.Map{
		"success":    true,
		"activities": logs,
		"total":      total,
		"hasMore":    int64(offset+limit) < total,
	})
}
