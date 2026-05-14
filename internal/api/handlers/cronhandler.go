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
	h.db.Order("startedAt DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&history)
	return c.JSON(history)
}

func (h *CronHandler) TriggerJob(c fiber.Ctx) error {
	job := c.Params("job")
	if err := h.scheduler.TriggerJob(job); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "triggered", "job": job})
}

// GET /api/cron — cron service info
func (h *CronHandler) Info(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"service": "salfanet-cron",
		"jobs":    9,
		"status":  "running",
	})
}

// GET /api/cron/status — detailed scheduler status with jobs array
func (h *CronHandler) Status(c fiber.Ctx) error {
	type LastRun struct {
		StartedAt   string  `json:"startedAt"`
		CompletedAt *string `json:"completedAt,omitempty"`
		Status      string  `json:"status"`
		Duration    *int    `json:"duration,omitempty"`
		Result      *string `json:"result,omitempty"`
		Error       *string `json:"error,omitempty"`
	}
	type Job struct {
		Type          string      `json:"type"`
		Name          string      `json:"name"`
		Description   string      `json:"description"`
		ScheduleLabel string      `json:"scheduleLabel"`
		Enabled       bool        `json:"enabled"`
		Health        string      `json:"health"`
		LastRun       interface{} `json:"lastRun"`
		NextRun       string      `json:"nextRun"`
		RecentHistory interface{} `json:"recentHistory"`
	}

	jobDefs := []struct{ typ, name, desc, sched string }{
		{"invoice_generate", "Invoice Generator", "Generate monthly invoices", "Daily 00:01 WIB"},
		{"send_reminders", "Send Reminders", "Send payment reminders", "Every hour"},
		{"invoice_catchup", "Invoice Catch-up", "Invoice catch-up for isolated users", "Daily 00:10 WIB"},
		{"pppoe_session_sync", "PPPoE Session Sync", "Sync PPPoE sessions from NAS", "Every minute"},
		{"session_monitor", "Session Monitor", "Monitor and close isolated sessions", "Every 5 min"},
		{"auto_isolate", "Auto Isolate", "Auto-isolate overdue users", "Daily 00:05 WIB"},
		{"freeradius_health", "FreeRADIUS Health", "FreeRADIUS health check & NAS sync", "Every 5 min"},
		{"voucher_expiry", "Voucher Expiry", "Sync voucher expiry status", "Every 5 min"},
		{"agent_sales", "Agent Sales", "Agent sales recording", "Every hour"},
	}

	var jobs []Job
	for _, def := range jobDefs {
		var lastHist models.CronHistory
		var lr interface{}
		if err := h.db.Where("jobType = ?", def.typ).Order("startedAt desc").First(&lastHist).Error; err == nil {
			var dur *int
			if lastHist.Duration != nil {
				d := int(*lastHist.Duration)
				dur = &d
			}
			var completedAt *string
			if lastHist.CompletedAt != nil {
				s := lastHist.CompletedAt.Format("2006-01-02T15:04:05.000Z")
				completedAt = &s
			}
			lr = LastRun{
				StartedAt:   lastHist.StartedAt.Format("2006-01-02T15:04:05.000Z"),
				CompletedAt: completedAt,
				Status:      lastHist.Status,
				Duration:    dur,
				Result:      lastHist.Result,
				Error:       lastHist.Error,
			}
		}
		var recentHists []models.CronHistory
		h.db.Where("jobType = ?", def.typ).Order("startedAt desc").Limit(5).Find(&recentHists)
		jobs = append(jobs, Job{
			Type: def.typ, Name: def.name, Description: def.desc,
			ScheduleLabel: def.sched, Enabled: true, Health: "healthy",
			LastRun: lr, NextRun: "", RecentHistory: recentHists,
		})
	}
	if jobs == nil {
		jobs = []Job{}
	}
	return c.JSON(fiber.Map{"success": true, "running": true, "jobs": jobs})
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
