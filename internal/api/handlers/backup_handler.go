package handlers

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

type BackupHandler struct{ db *gorm.DB }

func NewBackupHandler(db *gorm.DB) *BackupHandler { return &BackupHandler{db: db} }

const backupDir = "/var/www/salfanet-radius/backups"

// parseDBCredentials parses DATABASE_URL env var to extract MySQL credentials.
// Falls back to individual DB_* env vars, then to hardcoded defaults.
func parseDBCredentials() (host, user, pass, dbname string) {
	host, user, pass, dbname = "127.0.0.1", "root", "", "salfanet_radius"
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		u, err := url.Parse(databaseURL)
		if err == nil && u.User != nil {
			user = u.User.Username()
			pass, _ = u.User.Password()
			dbname = strings.TrimPrefix(u.Path, "/")
			if h := u.Hostname(); h != "" {
				host = h
			}
		}
	} else {
		if v := os.Getenv("DB_HOST"); v != "" {
			host = v
		}
		if v := os.Getenv("DB_USER"); v != "" {
			user = v
		}
		if v := os.Getenv("DB_PASSWORD"); v != "" {
			pass = v
		}
		if v := os.Getenv("DB_NAME"); v != "" {
			dbname = v
		}
	}
	return
}

// doMysqlDump runs mysqldump and gzips the output to targetPath.
func doMysqlDump(targetPath string) error {
	host, user, pass, dbname := parseDBCredentials()
	shellCmd := fmt.Sprintf("mysqldump -h%s -u%s --no-tablespaces %s | gzip > %s", host, user, dbname, targetPath)
	cmd := exec.Command("sh", "-c", shellCmd)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mysqldump: %s — %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// doMysqlRestore restores a gzipped SQL dump file to the database.
func doMysqlRestore(filePath string) error {
	host, user, pass, dbname := parseDBCredentials()
	shellCmd := fmt.Sprintf("zcat %s | mysql -h%s -u%s %s", filePath, host, user, dbname)
	cmd := exec.Command("sh", "-c", shellCmd)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restore: %s — %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// sendTelegramMessage sends a plain text message to a Telegram chat.
func sendTelegramMessage(botToken, chatId, text string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := fmt.Sprintf(`{"chat_id":%q,"text":%q,"parse_mode":"HTML"}`, chatId, text)
	resp, err := http.Post(apiURL, "application/json", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("telegram sendMessage: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram sendMessage status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// sendTelegramDocument uploads a file to a Telegram chat as a document.
func sendTelegramDocument(botToken, chatId, filePath, caption string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open backup file: %w", err)
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
		return fmt.Errorf("telegram sendDocument: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram sendDocument status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GET /api/backup/history
func (h *BackupHandler) History(c fiber.Ctx) error {
	limit := 50
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = v
	}
	var histories []models.BackupHistory
	h.db.Order("createdAt desc").Limit(limit).Find(&histories)
	if histories == nil {
		histories = []models.BackupHistory{}
	}
	return c.JSON(fiber.Map{"success": true, "history": histories})
}

// POST /api/backup/create
func (h *BackupHandler) Create(c fiber.Ctx) error {
	var body struct {
		Method string `json:"method"` // local / telegram / both
	}
	c.Bind().JSON(&body)
	method := body.Method
	if method == "" {
		method = "local"
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "cannot create backup directory: " + err.Error()})
	}

	now := time.Now()
	filename := fmt.Sprintf("backup_%s.sql.gz", now.Format("20060102_150405"))
	fullPath := filepath.Join(backupDir, filename)

	dumpErr := doMysqlDump(fullPath)

	var filesize int64
	if info, statErr := os.Stat(fullPath); statErr == nil {
		filesize = info.Size()
	}

	status := "success"
	var errMsg *string
	if dumpErr != nil {
		status = "failed"
		e := dumpErr.Error()
		errMsg = &e
	}

	hist := models.BackupHistory{
		ID:       generateID(),
		Filename: filename,
		Filepath: &fullPath,
		Filesize: filesize,
		Type:     "manual",
		Status:   status,
		Method:   method,
		Error:    errMsg,
	}
	h.db.Create(&hist)

	if dumpErr != nil {
		return c.Status(500).JSON(fiber.Map{"error": "backup failed", "details": dumpErr.Error()})
	}

	// If method includes telegram, send to Telegram
	if method == "telegram" || method == "both" {
		var tgs models.TelegramBackupSettings
		if err := h.db.First(&tgs).Error; err == nil && tgs.Enabled && tgs.BotToken != "" {
			caption := fmt.Sprintf("🗄 Manual backup — %s\nSize: %d KB", filename, filesize/1024)
			_ = sendTelegramDocument(tgs.BotToken, tgs.ChatId, fullPath, caption)
		}
	}

	return c.JSON(fiber.Map{"success": true, "backup": hist})
}

// DELETE /api/backup/:id
func (h *BackupHandler) Delete(c fiber.Ctx) error {
	id := c.Params("id")
	var hist models.BackupHistory
	if err := h.db.Where("id = ?", id).First(&hist).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "backup not found"})
	}
	if hist.Filepath != nil {
		os.Remove(*hist.Filepath)
	}
	h.db.Delete(&models.BackupHistory{}, "id = ?", id)
	return c.JSON(fiber.Map{"success": true})
}

// GET /api/backup/download/:id
func (h *BackupHandler) Download(c fiber.Ctx) error {
	id := c.Params("id")
	var hist models.BackupHistory
	if err := h.db.Where("id = ?", id).First(&hist).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "backup not found"})
	}
	if hist.Filepath == nil {
		return c.Status(404).JSON(fiber.Map{"error": "backup file not found"})
	}
	return c.Download(*hist.Filepath, hist.Filename)
}

// POST /api/backup/restore
func (h *BackupHandler) Restore(c fiber.Ctx) error {
	var body struct {
		BackupID string `json:"backupId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.BackupID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "backupId required"})
	}
	var hist models.BackupHistory
	if err := h.db.Where("id = ?", body.BackupID).First(&hist).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "backup not found"})
	}
	if hist.Filepath == nil {
		return c.Status(400).JSON(fiber.Map{"error": "no file to restore"})
	}
	if _, err := os.Stat(*hist.Filepath); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": "backup file missing from disk"})
	}
	if err := doMysqlRestore(*hist.Filepath); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "restore failed", "details": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Database restored successfully"})
}

// GET /api/backup/telegram/settings
func (h *BackupHandler) GetTelegramSettings(c fiber.Ctx) error {
	var tgs models.TelegramBackupSettings
	if err := h.db.First(&tgs).Error; err != nil {
		// No settings yet — return safe defaults
		return c.JSON(fiber.Map{
			"success": true,
			"settings": fiber.Map{
				"botToken": "", "chatId": "", "enabled": false,
				"schedule": "daily", "scheduleTime": "02:00", "keepLastN": 7,
			},
		})
	}
	return c.JSON(fiber.Map{"success": true, "settings": tgs})
}

// PUT /api/backup/telegram/settings
func (h *BackupHandler) UpdateTelegramSettings(c fiber.Ctx) error {
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
		// Create new
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
	return c.JSON(fiber.Map{"success": true, "message": "telegram backup settings saved"})
}

// GET /api/backup — alias for backup history list
func (h *BackupHandler) ListBackups(c fiber.Ctx) error {
	return h.History(c)
}

// GET /api/backup/health
func (h *BackupHandler) Health(c fiber.Ctx) error {
	var count int64
	h.db.Model(&models.BackupHistory{}).Count(&count)
	dbInfo, _ := h.db.DB()
	dbOk := dbInfo != nil && dbInfo.Ping() == nil

	var lastBackup *string
	var lastHist models.BackupHistory
	if err := h.db.Where("status = ?", "success").Order("createdAt desc").First(&lastHist).Error; err == nil {
		s := lastHist.CreatedAt.Format(time.RFC3339)
		lastBackup = &s
	}

	status := "healthy"
	if !dbOk {
		status = "error"
	}

	var tableCount int64
	h.db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()").Scan(&tableCount)

	var dbSize string
	h.db.Raw("SELECT CONCAT(ROUND(SUM(data_length + index_length) / 1024 / 1024, 2), ' MB') FROM information_schema.tables WHERE table_schema = DATABASE()").Scan(&dbSize)
	if dbSize == "" {
		dbSize = "0 MB"
	}

	return c.JSON(fiber.Map{
		"success": true,
		"health": fiber.Map{
			"status":      status,
			"size":        dbSize,
			"tables":      tableCount,
			"connections": "1/100",
			"lastBackup":  lastBackup,
			"uptime":      "active",
		},
	})
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
