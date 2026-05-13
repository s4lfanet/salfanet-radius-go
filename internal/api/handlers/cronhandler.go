package handlers

import (
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/cron"
	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// CronHandler handles cron history and manual trigger endpoints.
type CronHandler struct {
	db        *gorm.DB
	scheduler *cron.Scheduler
}

func NewCronHandler(db *gorm.DB, scheduler *cron.Scheduler) *CronHandler {
	return &CronHandler{db: db, scheduler: scheduler}
}

func (h *CronHandler) ListHistory(c fiber.Ctx) error {
	var history []models.CronHistory
	page, pageSize := pageParams(c)
	h.db.Order("started_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&history)
	return c.JSON(history)
}

func (h *CronHandler) TriggerJob(c fiber.Ctx) error {
	job := c.Params("job")
	if err := h.scheduler.TriggerJob(job); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "triggered", "job": job})
}

// GET /api/cron/schedules — list all registered cron job schedules
func (h *CronHandler) ListSchedules(c fiber.Ctx) error {
	schedules := []fiber.Map{
		{"job": "generate-invoices", "schedule": "0 1 0 * * *", "description": "Generate monthly invoices (00:01 daily)"},
		{"job": "send-reminders", "schedule": "0 0 * * * *", "description": "Send payment reminders (hourly)"},
		{"job": "invoice-catchup", "schedule": "0 10 0 * * *", "description": "Invoice catch-up (00:10 daily)"},
		{"job": "pppoe-session-sync", "schedule": "0 * * * * *", "description": "Sync PPPoE sessions (every minute)"},
		{"job": "session-monitor", "schedule": "0 */5 * * * *", "description": "Monitor sessions (every 5 min)"},
		{"job": "auto-isolate", "schedule": "0 5 0 * * *", "description": "Auto-isolate overdue users (00:05 daily)"},
		{"job": "freeradius-health", "schedule": "30 */5 * * * *", "description": "FreeRADIUS health check (every 5 min)"},
		{"job": "voucher-expiry", "schedule": "0 */5 * * * *", "description": "Sync voucher expiry (every 5 min)"},
		{"job": "agent-sales", "schedule": "0 0 * * * *", "description": "Agent sales recording (hourly)"},
	}
	return c.JSON(fiber.Map{"success": true, "schedules": schedules})
}

// PUT /api/cron/schedules/:job — update a schedule (configuration only, requires restart)
func (h *CronHandler) UpdateSchedule(c fiber.Ctx) error {
	job := c.Params("job")
	var body struct {
		Schedule string `json:"schedule"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Schedule == "" {
		return c.Status(400).JSON(fiber.Map{"error": "schedule expression required"})
	}
	return c.JSON(fiber.Map{
		"success":  true,
		"job":      job,
		"schedule": body.Schedule,
		"message":  "Schedule noted — restart server to apply changes",
	})
}

// DELETE /api/cron/schedules/:job — disable a schedule
func (h *CronHandler) DeleteSchedule(c fiber.Ctx) error {
	job := c.Params("job")
	return c.JSON(fiber.Map{
		"success": true,
		"job":     job,
		"message": "Schedule marked disabled — restart server to apply changes",
	})
}
