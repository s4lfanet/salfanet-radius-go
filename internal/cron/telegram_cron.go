package cron

// telegram_cron.go — Port dari src/server/jobs/telegram-cron.ts
//
// Two jobs:
// 1. telegram_backup — creates DB backup and sends to Telegram (dynamic schedule from settings)
// 2. telegram_health — sends comprehensive health report to Telegram (every hour)

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

const telegramBackupDir = "/var/www/salfanet-radius/backups"

// jobTelegramBackup creates a DB backup and sends it to Telegram.
func (s *Scheduler) jobTelegramBackup() {
	h := s.startHistory("telegram_backup")
	defer func() { s.completeHistory(h, recover()) }()

	// Deduplication guard: skip if another run started within 5 minutes
	fiveMinAgo := time.Now().Add(-5 * time.Minute)
	var recent models.CronHistory
	if s.db.Where("jobType = ? AND status IN ? AND startedAt > ?",
		"telegram_backup", []string{"running", "success"}, fiveMinAgo).
		Order("startedAt desc").First(&recent).Error == nil {
		log.Info().Msg("cron: telegram_backup skipped (duplicate)")
		s.finishHistory(h, "Skipped — duplicate run")
		return
	}

	// Get Telegram settings
	var settings models.TelegramBackupSettings
	if err := s.db.Where("enabled = true").Order("createdAt desc").First(&settings).Error; err != nil {
		s.finishHistory(h, "Telegram backup disabled, skipped")
		return
	}

	// Create backup directory
	if err := os.MkdirAll(telegramBackupDir, 0755); err != nil {
		s.failHistory(h, fmt.Errorf("mkdir: %w", err))
		return
	}

	// Create DB dump
	now := time.Now()
	filename := fmt.Sprintf("backup_%s.sql.gz", now.Format("20060102_150405"))
	fullPath := filepath.Join(telegramBackupDir, filename)

	if err := telegramMysqlDump(fullPath); err != nil {
		s.failHistory(h, fmt.Errorf("mysqldump: %w", err))
		return
	}

	var filesize int64
	if info, err := os.Stat(fullPath); err == nil {
		filesize = info.Size()
	}

	// Send to Telegram
	caption := fmt.Sprintf("🗄 Auto backup — %s\nSize: %d KB", filename, filesize/1024)
	if err := telegramSendDocument(settings.BotToken, settings.ChatId, fullPath, caption); err != nil {
		s.failHistory(h, fmt.Errorf("send to telegram: %w", err))
		return
	}

	// Record backup history
	hist := models.BackupHistory{
		ID:       fmt.Sprintf("%d", now.UnixNano()),
		Filename: filename,
		Filepath: &fullPath,
		Filesize: filesize,
		Type:     "auto",
		Status:   "success",
		Method:   "telegram",
	}
	s.db.Create(&hist)

	// Cleanup old backups
	if settings.KeepLastN > 0 {
		var oldBackups []models.BackupHistory
		s.db.Where("type = ?", "auto").Order("createdAt desc").Find(&oldBackups)
		if len(oldBackups) > settings.KeepLastN {
			for _, old := range oldBackups[settings.KeepLastN:] {
				if old.Filepath != nil {
					os.Remove(*old.Filepath)
				}
				s.db.Delete(&old)
			}
		}
	}

	s.finishHistory(h, fmt.Sprintf("Backup sent to Telegram: %s", filename))
	log.Info().Str("file", filename).Msg("cron: telegram_backup done")
}

// jobTelegramHealth sends a comprehensive health report to Telegram.
func (s *Scheduler) jobTelegramHealth() {
	h := s.startHistory("telegram_health")
	defer func() { s.completeHistory(h, recover()) }()

	// Get Telegram settings
	var settings models.TelegramBackupSettings
	if err := s.db.Where("enabled = true").Order("createdAt desc").First(&settings).Error; err != nil {
		s.finishHistory(h, "Telegram health check disabled, skipped")
		return
	}

	// Gather health data
	health := s.gatherHealth()

	// Build report
	report := formatHealthReport(health)

	// Send to Telegram
	chatId := settings.ChatId
	if settings.HealthTopicId != nil && *settings.HealthTopicId != "" {
		chatId = chatId + ":" + *settings.HealthTopicId
	}
	if err := telegramSendMessage(settings.BotToken, chatId, report); err != nil {
		s.failHistory(h, fmt.Errorf("send health to telegram: %w", err))
		return
	}

	s.finishHistory(h, fmt.Sprintf("Health report sent (status: %s)", health["status"]))
	log.Info().Str("status", health["status"].(string)).Msg("cron: telegram_health done")
}

type healthData map[string]interface{}

func (s *Scheduler) gatherHealth() healthData {
	// DB size
	var dbSize float64
	s.db.Raw("SELECT ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) FROM information_schema.tables WHERE table_schema = DATABASE()").Scan(&dbSize)

	var tableCount int64
	s.db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()").Scan(&tableCount)

	// Active sessions
	var activeSessions int64
	s.db.Table("radacct").Where("acctstoptime IS NULL").Count(&activeSessions)

	// Pending overdue invoices
	var pendingInvoices int64
	s.db.Table("invoices").Where("status = ? AND dueDate < ?", "PENDING", time.Now()).Count(&pendingInvoices)

	// PPPoE users
	var totalUsers int64
	s.db.Model(&models.PppoeUser{}).Count(&totalUsers)
	var activeUsers int64
	s.db.Model(&models.PppoeUser{}).Where("status = ?", "active").Count(&activeUsers)

	// Determine status
	status := "healthy"
	var issues []string
	if dbSize > 5000 {
		status = "critical"
		issues = append(issues, "Database size > 5GB")
	} else if dbSize > 1000 {
		status = "warning"
		issues = append(issues, "Database size > 1GB")
	}
	if pendingInvoices > 50 {
		if status == "healthy" {
			status = "warning"
		}
		issues = append(issues, fmt.Sprintf("%d overdue invoices", pendingInvoices))
	}

	return healthData{
		"status":          status,
		"size":            fmt.Sprintf("%.2f MB", dbSize),
		"tables":          tableCount,
		"activeSessions":  activeSessions,
		"totalUsers":      totalUsers,
		"activeUsers":     activeUsers,
		"pendingInvoices": pendingInvoices,
		"issues":          strings.Join(issues, ", "),
	}
}

func formatHealthReport(h healthData) string {
	statusEmoji := "✅"
	switch h["status"] {
	case "warning":
		statusEmoji = "⚠️"
	case "critical":
		statusEmoji = "🔴"
	}

	report := fmt.Sprintf(`<b>%s System Health Report</b>
━━━━━━━━━━━━━━━
<b>Status:</b> %s %s
<b>Database Size:</b> %s
<b>Tables:</b> %d
<b>Active Sessions:</b> %d
<b>Total Users:</b> %d
<b>Active Users:</b> %d
<b>Pending Invoices:</b> %d
<b>Time:</b> %s`,
		statusEmoji,
		h["status"], statusEmoji,
		h["size"],
		h["tables"],
		h["activeSessions"],
		h["totalUsers"],
		h["activeUsers"],
		h["pendingInvoices"],
		time.Now().Format("2006-01-02 15:04:05 WIB"),
	)

	if issues, ok := h["issues"].(string); ok && issues != "" {
		report += fmt.Sprintf("\n<b>⚠️ Issues:</b> %s", issues)
	}

	return report
}

// registerTelegramBackupCron reads settings from DB and registers the backup cron.
// Called once at scheduler startup. If settings change, the server must be restarted
// or TriggerJob("telegram_backup") can be used for manual runs.
func (s *Scheduler) registerTelegramBackupCron() {
	var settings models.TelegramBackupSettings
	if err := s.db.Where("enabled = true").Order("createdAt desc").First(&settings).Error; err != nil {
		log.Info().Msg("cron: telegram_backup disabled (no enabled settings)")
		return
	}

	cronExpr := telegramScheduleToCron(settings.Schedule, settings.ScheduleTime)
	s.cron.AddFunc(cronExpr, s.jobTelegramBackup)
	log.Info().Str("schedule", settings.Schedule).Str("time", settings.ScheduleTime).Str("cron", cronExpr).
		Msg("cron: telegram_backup registered")
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func telegramMysqlDump(targetPath string) error {
	host, user, pass, dbname := "127.0.0.1", "root", "", "salfanet_radius"
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		// Parse DATABASE_URL
		if u, err := parseDatabaseURL(databaseURL); err == nil {
			host = u.host
			user = u.user
			pass = u.pass
			dbname = u.dbname
		}
	}
	shellCmd := fmt.Sprintf("mysqldump -h%s -u%s --no-tablespaces %s | gzip > %s", host, user, dbname, targetPath)
	cmd := exec.Command("sh", "-c", shellCmd)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s — %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

type dbURL struct {
	host, user, pass, dbname string
}

func parseDatabaseURL(url string) (*dbURL, error) {
	// Simple parser for mysql://user:pass@host:port/dbname
	u := &dbURL{}
	if !strings.HasPrefix(url, "mysql://") && !strings.HasPrefix(url, "mysql+") {
		// Try generic URL parsing
	}
	// Remove scheme
	rest := url
	if idx := strings.Index(rest, "://"); idx >= 0 {
		rest = rest[idx+3:]
	}
	// Split user:pass@host:port/db
	atIdx := strings.LastIndex(rest, "@")
	if atIdx >= 0 {
		userPass := rest[:atIdx]
		rest = rest[atIdx+1:]
		colonIdx := strings.Index(userPass, ":")
		if colonIdx >= 0 {
			u.user = userPass[:colonIdx]
			u.pass = userPass[colonIdx+1:]
		} else {
			u.user = userPass
		}
	}
	// Split host:port/db
	slashIdx := strings.Index(rest, "/")
	if slashIdx >= 0 {
		u.host = rest[:slashIdx]
		u.dbname = rest[slashIdx+1:]
	} else {
		u.host = rest
	}
	// Remove query string from dbname
	if qIdx := strings.Index(u.dbname, "?"); qIdx >= 0 {
		u.dbname = u.dbname[:qIdx]
	}
	// Remove port from host
	if cIdx := strings.Index(u.host, ":"); cIdx >= 0 {
		u.host = u.host[:cIdx]
	}
	return u, nil
}

func telegramSendDocument(botToken, chatId, filePath, caption string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chat_id", chatId)
	if caption != "" {
		_ = w.WriteField("caption", caption)
	}
	fw, err := w.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return err
	}
	w.Close()

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", botToken)
	resp, err := http.Post(apiURL, w.FormDataContentType(), &buf)
	if err != nil {
		return fmt.Errorf("sendDocument: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sendDocument status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func telegramSendMessage(botToken, chatId, text string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := fmt.Sprintf(`{"chat_id":%q,"text":%q,"parse_mode":"HTML"}`, chatId, text)
	resp, err := http.Post(apiURL, "application/json", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("sendMessage: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sendMessage status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// telegramScheduleToCron converts TelegramBackupSettings schedule to cron expression.
func telegramScheduleToCron(schedule, scheduleTime string) string {
	parts := strings.Split(scheduleTime, ":")
	hour, minute := 2, 0
	if len(parts) == 2 {
		hour, _ = strconv.Atoi(parts[0])
		minute, _ = strconv.Atoi(parts[1])
	}

	switch schedule {
	case "daily":
		return fmt.Sprintf("0 %d %d * * *", minute, hour)
	case "12h":
		return fmt.Sprintf("0 %d %d,%d * * *", minute, hour, (hour+12)%24)
	case "6h":
		return fmt.Sprintf("0 %d %d,%d,%d,%d * * *", minute, hour, (hour+6)%24, (hour+12)%24, (hour+18)%24)
	case "weekly":
		return fmt.Sprintf("0 %d %d * * 0", minute, hour)
	default:
		return fmt.Sprintf("0 %d %d * * *", minute, hour)
	}
}
