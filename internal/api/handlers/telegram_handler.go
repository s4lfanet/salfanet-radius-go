package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

type TelegramHandler struct{ db *gorm.DB }

func NewTelegramHandler(db *gorm.DB) *TelegramHandler { return &TelegramHandler{db: db} }

// getTGSettings loads TelegramBackupSettings from DB (returns nil if not found).
func (h *TelegramHandler) getTGSettings() (*models.TelegramBackupSettings, error) {
	var s models.TelegramBackupSettings
	if err := h.db.First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// GET /api/telegram/settings
func (h *TelegramHandler) GetSettings(c fiber.Ctx) error {
	s, err := h.getTGSettings()
	if err != nil {
		return c.JSON(fiber.Map{
			"success": true,
			"settings": fiber.Map{
				"botToken": "", "chatId": "", "enabled": false,
				"schedule": "daily", "scheduleTime": "02:00", "keepLastN": 7,
				"sendDailyReport": false, "sendAlerts": false,
			},
		})
	}
	return c.JSON(fiber.Map{"success": true, "settings": s})
}

// PUT /api/telegram/settings
func (h *TelegramHandler) UpdateSettings(c fiber.Ctx) error {
	var body struct {
		BotToken      string  `json:"botToken"`
		ChatId        string  `json:"chatId"`
		Enabled       bool    `json:"enabled"`
		Schedule      string  `json:"schedule"`
		ScheduleTime  string  `json:"scheduleTime"`
		KeepLastN     int     `json:"keepLastN"`
		BackupTopicId *string `json:"backupTopicId"`
		HealthTopicId *string `json:"healthTopicId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Schedule == "" {
		body.Schedule = "daily"
	}
	if body.ScheduleTime == "" {
		body.ScheduleTime = "02:00"
	}
	if body.KeepLastN <= 0 {
		body.KeepLastN = 7
	}

	var tgs models.TelegramBackupSettings
	if err := h.db.First(&tgs).Error; err != nil {
		tgs = models.TelegramBackupSettings{
			ID:            generateID(),
			BotToken:      body.BotToken,
			ChatId:        body.ChatId,
			Enabled:       body.Enabled,
			Schedule:      body.Schedule,
			ScheduleTime:  body.ScheduleTime,
			KeepLastN:     body.KeepLastN,
			BackupTopicId: body.BackupTopicId,
			HealthTopicId: body.HealthTopicId,
		}
		h.db.Create(&tgs)
	} else {
		h.db.Model(&tgs).Updates(map[string]interface{}{
			"botToken":      body.BotToken,
			"chatId":        body.ChatId,
			"enabled":       body.Enabled,
			"schedule":      body.Schedule,
			"scheduleTime":  body.ScheduleTime,
			"keepLastN":     body.KeepLastN,
			"backupTopicId": body.BackupTopicId,
			"healthTopicId": body.HealthTopicId,
		})
	}
	return c.JSON(fiber.Map{"success": true, "message": "settings saved"})
}

// POST /api/telegram/test — send a test message
func (h *TelegramHandler) Test(c fiber.Ctx) error {
	s, err := h.getTGSettings()
	if err != nil || s.BotToken == "" || s.ChatId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "telegram not configured — set bot token and chat ID first"})
	}
	msg := "✅ <b>Test pesan dari Salfanet RADIUS</b>\n\nKoneksi Telegram berhasil! Notifikasi akan dikirim ke chat ini."
	if sendErr := sendTelegramMessage(s.BotToken, s.ChatId, msg); sendErr != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to send: " + sendErr.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "test message sent"})
}

// POST /api/telegram/send-backup — send latest backup file to Telegram
func (h *TelegramHandler) SendBackup(c fiber.Ctx) error {
	s, err := h.getTGSettings()
	if err != nil || s.BotToken == "" || s.ChatId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "telegram not configured"})
	}

	// Find latest successful backup
	var hist models.BackupHistory
	if err := h.db.Where("status = ?", "success").Order("createdAt desc").First(&hist).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "no backup found — create a backup first"})
	}
	if hist.Filepath == nil {
		return c.Status(404).JSON(fiber.Map{"error": "backup file path missing"})
	}
	if _, statErr := os.Stat(*hist.Filepath); os.IsNotExist(statErr) {
		return c.Status(404).JSON(fiber.Map{"error": "backup file not found on disk"})
	}

	caption := fmt.Sprintf("🗄 <b>Database Backup</b>\nFile: %s\nSize: %d KB\nDate: %s",
		hist.Filename, hist.Filesize/1024, hist.CreatedAt.Format("2006-01-02 15:04:05"))
	if sendErr := sendTelegramDocument(s.BotToken, s.ChatId, *hist.Filepath, caption); sendErr != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to send backup: " + sendErr.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "backup sent to telegram", "file": hist.Filename})
}

// POST /api/telegram/test-backup — create a fresh backup and send to Telegram
func (h *TelegramHandler) TestBackup(c fiber.Ctx) error {
	s, err := h.getTGSettings()
	if err != nil || s.BotToken == "" || s.ChatId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "telegram not configured — set bot token and chat ID first"})
	}

	// Create backup dir if needed
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "cannot create backup directory"})
	}

	now := time.Now()
	filename := fmt.Sprintf("test_backup_%s.sql.gz", now.Format("20060102_150405"))
	fullPath := filepath.Join(backupDir, filename)

	if dumpErr := doMysqlDump(fullPath); dumpErr != nil {
		return c.Status(500).JSON(fiber.Map{"error": "backup failed: " + dumpErr.Error()})
	}

	info, _ := os.Stat(fullPath)
	var filesize int64
	if info != nil {
		filesize = info.Size()
	}

	caption := fmt.Sprintf("🧪 <b>Test Backup Telegram</b>\nFile: %s\nSize: %d KB\nDate: %s",
		filename, filesize/1024, now.Format("2006-01-02 15:04:05"))
	if sendErr := sendTelegramDocument(s.BotToken, s.ChatId, fullPath, caption); sendErr != nil {
		os.Remove(fullPath)
		return c.Status(500).JSON(fiber.Map{"error": "backup created but send failed: " + sendErr.Error()})
	}

	// Save to history
	fp := fullPath
	hist := models.BackupHistory{
		ID:       generateID(),
		Filename: filename,
		Filepath: &fp,
		Filesize: filesize,
		Type:     "test",
		Status:   "success",
		Method:   "telegram",
	}
	h.db.Create(&hist)

	return c.JSON(fiber.Map{"success": true, "message": "test backup created and sent to telegram", "file": filename})
}

// POST /api/telegram/send-health — send a system health report
func (h *TelegramHandler) SendHealth(c fiber.Ctx) error {
	s, err := h.getTGSettings()
	if err != nil || s.BotToken == "" || s.ChatId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "telegram not configured"})
	}

	// Count users and backups for a basic health report
	var userCount, backupCount int64
	h.db.Table("ppp_oe_users").Count(&userCount)
	h.db.Model(&models.BackupHistory{}).Where("status = ?", "success").Count(&backupCount)

	msg := fmt.Sprintf("💚 <b>Salfanet RADIUS — Health Report</b>\n\n"+
		"🕐 Time: %s\n"+
		"👥 Active Users: %d\n"+
		"🗄 Successful Backups: %d\n"+
		"✅ Status: Online",
		time.Now().Format("2006-01-02 15:04:05"),
		userCount,
		backupCount,
	)

	if sendErr := sendTelegramMessage(s.BotToken, s.ChatId, msg); sendErr != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to send: " + sendErr.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "health report sent to telegram"})
}
