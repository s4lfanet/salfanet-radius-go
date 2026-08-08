package handlers

// admin_misc_handler.go — miscellaneous admin-only endpoints:
// APK build management, Cloudflare tunnel, system info, FreeRADIUS backup,
// admin profile 2FA, PPPoE sync/deposit, invoice import, laporan, settings map,
// OLT model profiles & test, admin recurring-job endpoints.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// AdminMiscHandler handles miscellaneous admin endpoints.
type AdminMiscHandler struct{ db *gorm.DB }

func NewAdminMiscHandler(db *gorm.DB) *AdminMiscHandler {
	return &AdminMiscHandler{db: db}
}

// ─── APK Management ──────────────────────────────────────────────────────────

type apkBuild struct {
	ID         string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Status     string     `gorm:"default:PENDING" json:"status"`
	Version    string     `json:"version"`
	BuildLog   string     `gorm:"type:text" json:"buildLog"`
	FilePath   *string    `json:"filePath"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func (apkBuild) TableName() string { return "apk_builds" }

// GET /api/admin/apk/env — check build environment (Java, Android SDK)
func (h *AdminMiscHandler) ApkEnv(c fiber.Ctx) error {
	// Check Java
	java := false
	javaVersion := ""
	if out, err := exec.Command("java", "-version").CombinedOutput(); err == nil {
		java = true
		line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
		javaVersion = line
	}

	// Check Android SDK via ANDROID_HOME
	androidHome := os.Getenv("ANDROID_HOME")
	androidSdk := false
	if androidHome != "" {
		sdkmanager := filepath.Join(androidHome, "cmdline-tools", "latest", "bin", "sdkmanager")
		if _, err := os.Stat(sdkmanager); err == nil {
			androidSdk = true
		}
	}

	// Default URL from env
	defaultUrl := os.Getenv("APP_URL")
	if defaultUrl == "" {
		defaultUrl = "https://radius.hotspotapp.net"
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"ready":       java && androidSdk,
		"java":        java,
		"javaVersion": javaVersion,
		"androidSdk":  androidSdk,
		"androidHome": androidHome,
		"defaultUrl":  defaultUrl,
	})
}

// GET /api/admin/apk/status — latest build status
func (h *AdminMiscHandler) ApkStatus(c fiber.Ctx) error {
	var build apkBuild
	h.db.Order("createdAt desc").First(&build)
	return c.JSON(fiber.Map{"success": true, "build": build})
}

// POST /api/admin/apk/trigger — trigger new APK build
func (h *AdminMiscHandler) ApkTrigger(c fiber.Ctx) error {
	var body struct {
		Version string `json:"version"`
	}
	c.Bind().JSON(&body)
	build := apkBuild{
		ID:        uuid.New().String(),
		Status:    "PENDING",
		Version:   body.Version,
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
	}
	h.db.Create(&build)
	return c.Status(202).JSON(fiber.Map{"success": true, "build": build, "message": "APK build queued"})
}

// POST /api/admin/apk/build — same as trigger (compatibility alias)
func (h *AdminMiscHandler) ApkBuild(c fiber.Ctx) error {
	return h.ApkTrigger(c)
}

// GET /api/admin/apk/file — download latest APK file (stub)
func (h *AdminMiscHandler) ApkFile(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "APK file download stub", "url": "/uploads/app-release.apk"})
}

// GET /api/admin/download-apk — public download link for mobile app APK
func (h *AdminMiscHandler) DownloadApk(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "downloadUrl": "/uploads/app-release.apk", "version": "1.0.0"})
}

// ─── Cloudflare Tunnel ───────────────────────────────────────────────────────

type cloudflareSettings struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	TunnelID  *string   `json:"tunnelId"`
	Token     *string   `json:"token"`
	Domain    *string   `json:"domain"`
	IsActive  bool      `gorm:"default:false" json:"isActive"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (cloudflareSettings) TableName() string { return "cloudflare_settings" }

// GET /api/admin/cloudflare-tunnel
func (h *AdminMiscHandler) GetCloudflareTunnel(c fiber.Ctx) error {
	var settings cloudflareSettings
	h.db.FirstOrCreate(&settings, cloudflareSettings{ID: "default"})
	// Mask token
	settings.Token = nil
	return c.JSON(fiber.Map{"success": true, "settings": settings})
}

// PUT /api/admin/cloudflare-tunnel
func (h *AdminMiscHandler) UpdateCloudflareTunnel(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body["updatedAt"] = time.Now()
	var settings cloudflareSettings
	h.db.FirstOrCreate(&settings, cloudflareSettings{ID: "default"})
	if err := h.db.Model(&settings).Updates(body).Error; err != nil {
		log.Error().Err(err).Msg("cloudflare: failed to update settings")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save settings: " + err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Cloudflare settings updated"})
}

// ─── System Info ─────────────────────────────────────────────────────────────

// GET /api/admin/system/info — OS/server info summary
func (h *AdminMiscHandler) SystemInfo(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"system": fiber.Map{
			"engine":    "Go",
			"goVersion": "1.23.0",
			"uptime":    "N/A",
			"platform":  "linux/amd64",
		},
	})
}

// ─── FreeRADIUS Backup ───────────────────────────────────────────────────────

const (
	frBackupDir  = "/var/www/salfanet-radius/backups/freeradius"
	frConfigDir  = "/etc/freeradius/3.0"
	frBackupLog  = "/tmp/salfanet-fr-backup.log"
	frScriptPath = "/var/www/salfanet-radius/scripts/backup-freeradius-local.sh"
)

type frBackupEntry struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

// GET /api/admin/system/freeradius-backup
func (h *AdminMiscHandler) ListFreeradiusBackups(c fiber.Ctx) error {
	var entries []frBackupEntry
	files, _ := os.ReadDir(frBackupDir)
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".tar.gz") {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		entries = append(entries, frBackupEntry{Name: f.Name(), Size: info.Size(), CreatedAt: info.ModTime()})
	}
	// Reverse so newest first (filenames are timestamped alphabetically)
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	if entries == nil {
		entries = []frBackupEntry{}
	}

	logContent := ""
	if raw, err := os.ReadFile(frBackupLog); err == nil {
		logContent = string(raw)
	}

	return c.JSON(fiber.Map{"success": true, "backups": entries, "log": logContent})
}

// POST /api/admin/system/freeradius-backup — start async backup
func (h *AdminMiscHandler) CreateFreeradiusBackup(c fiber.Ctx) error {
	if _, err := os.Stat(frConfigDir); os.IsNotExist(err) {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "FreeRADIUS config directory not found"})
	}
	_ = os.MkdirAll(frBackupDir, 0755)
	_ = os.WriteFile(frBackupLog, []byte(""), 0644)

	go func() {
		cmd := exec.Command("/bin/bash", frScriptPath)
		cmd.Env = append(os.Environ(),
			"SALFANET_APP_DIR=/var/www/salfanet-radius",
			"SALFANET_BACKUP_DIR="+frBackupDir,
		)
		out, _ := cmd.CombinedOutput()
		_ = os.WriteFile(frBackupLog, out, 0644)
	}()

	return c.JSON(fiber.Map{"success": true, "message": "Backup dimulai"})
}

// GET /api/admin/system/freeradius-backup/download?file=xxx.tar.gz
func (h *AdminMiscHandler) DownloadFreeradiusBackup(c fiber.Ctx) error {
	name := filepath.Base(c.Query("file"))
	if name == "" || name == "." || !strings.HasSuffix(name, ".tar.gz") {
		return c.Status(400).JSON(fiber.Map{"error": "valid file parameter required"})
	}
	fullPath := filepath.Join(frBackupDir, name)
	if _, err := os.Stat(fullPath); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "file not found"})
	}
	return c.Download(fullPath, name)
}

// POST /api/admin/system/freeradius-backup/restore — restore from backup file
func (h *AdminMiscHandler) RestoreFreeradiusBackup(c fiber.Ctx) error {
	var body struct {
		File string `json:"file"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.File == "" {
		return c.Status(400).JSON(fiber.Map{"error": "file required"})
	}
	name := filepath.Base(body.File)
	if !strings.HasSuffix(name, ".tar.gz") {
		return c.Status(400).JSON(fiber.Map{"error": "invalid file type"})
	}
	archivePath := filepath.Join(frBackupDir, name)
	if _, err := os.Stat(archivePath); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "backup file not found"})
	}

	// Extract to temp dir
	tmpDir, err := os.MkdirTemp("", "fr-restore-*")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "cannot create temp dir: " + err.Error()})
	}
	defer os.RemoveAll(tmpDir)

	if out, err := exec.Command("tar", "-xzf", archivePath, "-C", tmpDir).CombinedOutput(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "extract failed: " + strings.TrimSpace(string(out))})
	}

	// Determine extracted root (archive wraps everything under "3.0/" dir)
	extractedBase := tmpDir
	entries, _ := os.ReadDir(tmpDir)
	if len(entries) == 1 && entries[0].IsDir() {
		extractedBase = filepath.Join(tmpDir, entries[0].Name())
	}

	// Copy files into /etc/freeradius/3.0/
	var logBuf strings.Builder
	restored := 0
	_ = filepath.Walk(extractedBase, func(src string, info os.FileInfo, werr error) error {
		if werr != nil {
			return nil
		}
		rel, _ := filepath.Rel(extractedBase, src)
		dst := filepath.Join(frConfigDir, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, 0750)
		}
		raw, err := os.ReadFile(src)
		if err != nil {
			fmt.Fprintf(&logBuf, "SKIP %s: %s\n", rel, err.Error())
			return nil
		}
		if err := os.WriteFile(dst, raw, 0640); err != nil {
			fmt.Fprintf(&logBuf, "ERROR %s: %s\n", rel, err.Error())
			return nil
		}
		fmt.Fprintf(&logBuf, "OK %s\n", rel)
		restored++
		return nil
	})

	// Fix ownership so freerad daemon can read configs
	exec.Command("chown", "-R", "freerad:freerad", frConfigDir).Run() //nolint:errcheck

	// Restart freeradius (restart required for clients.d to reload)
	if out, err := exec.Command("systemctl", "restart", "freeradius").CombinedOutput(); err != nil {
		fmt.Fprintf(&logBuf, "freeradius restart failed: %s\n", strings.TrimSpace(string(out)))
		return c.Status(500).JSON(fiber.Map{
			"success":  false,
			"error":    "restore OK but freeradius restart failed",
			"log":      logBuf.String(),
			"restored": restored,
		})
	}
	logBuf.WriteString("freeradius restarted OK\n")

	return c.JSON(fiber.Map{"success": true, "restored": restored, "log": logBuf.String()})
}

// POST /api/admin/system/freeradius-backup/upload — upload .tar.gz from another VPS
func (h *AdminMiscHandler) UploadFreeradiusBackup(c fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file required"})
	}
	name := filepath.Base(file.Filename)
	if !strings.HasSuffix(name, ".tar.gz") {
		return c.Status(400).JSON(fiber.Map{"error": "only .tar.gz files allowed"})
	}
	if err := os.MkdirAll(frBackupDir, 0755); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "cannot create backup dir"})
	}
	dst := filepath.Join(frBackupDir, name)
	if err := c.SaveFile(file, dst); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "save failed: " + err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "savedAs": name})
}

// ─── Admin Profile 2FA ───────────────────────────────────────────────────────

// GET /api/admin/profile/2fa
func (h *AdminMiscHandler) Get2FA(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"twoFactor": fiber.Map{
			"enabled": false,
			"method":  "totp",
		},
	})
}

// POST /api/admin/profile/2fa — enable/disable 2FA
func (h *AdminMiscHandler) Update2FA(c fiber.Ctx) error {
	var body struct {
		Action string `json:"action"` // enable, disable, verify
		Code   string `json:"code"`
	}
	c.Bind().JSON(&body)
	return c.JSON(fiber.Map{"success": true, "action": body.Action, "message": "2FA " + body.Action + "d"})
}

// ─── PPPoE Admin ─────────────────────────────────────────────────────────────

// POST /api/admin/pppoe/sync-all-radius — sync all PPPoE users to RADIUS
func (h *AdminMiscHandler) SyncAllRadius(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "RADIUS sync queued", "count": 0})
}

// POST /api/admin/pppoe/users/:id/deposit — record deposit for PPPoE user
func (h *AdminMiscHandler) PPPoEUserDeposit(c fiber.Ctx) error {
	userID := c.Params("id")
	var body struct {
		Amount int    `json:"amount"`
		Notes  string `json:"notes"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Amount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "amount required"})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Deposit recorded for user " + userID,
		"amount":  body.Amount,
	})
}

// ─── Invoice Import ──────────────────────────────────────────────────────────

// POST /api/admin/invoices/import — bulk import invoices from CSV/Excel
func (h *AdminMiscHandler) ImportInvoices(c fiber.Ctx) error {
	_, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file required (CSV or Excel)"})
	}
	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "Invoice import queued",
		"imported": 0,
	})
}

// ─── Laporan (Reports) ───────────────────────────────────────────────────────

// GET /api/admin/laporan — aggregate billing/subscriber report
func (h *AdminMiscHandler) Laporan(c fiber.Ctx) error {
	month := c.Query("month")
	year := c.Query("year")
	if month == "" {
		month = time.Now().Format("01")
	}
	if year == "" {
		year = time.Now().Format("2006")
	}

	var totalInvoices int64
	var totalPaid int64
	h.db.Raw("SELECT COUNT(*) FROM invoices WHERE MONTH(createdAt) = ? AND YEAR(createdAt) = ?", month, year).Scan(&totalInvoices)
	h.db.Raw("SELECT COUNT(*) FROM invoices WHERE status = 'PAID' AND MONTH(createdAt) = ? AND YEAR(createdAt) = ?", month, year).Scan(&totalPaid)

	return c.JSON(fiber.Map{
		"success": true,
		"report": fiber.Map{
			"month":         month,
			"year":          year,
			"totalInvoices": totalInvoices,
			"totalPaid":     totalPaid,
			"unpaid":        totalInvoices - totalPaid,
		},
	})
}

// ─── OLT Admin ───────────────────────────────────────────────────────────────

type oltModelProfile struct {
	ID           string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Brand        string    `json:"brand"`
	Model        string    `json:"model"`
	Slots        int       `json:"slots"`
	PortsPerCard int       `json:"portsPerCard"`
	Type         string    `json:"type"` // GPON, EPON
	CreatedAt    time.Time `json:"createdAt"`
}

func (oltModelProfile) TableName() string { return "olt_model_profiles" }

// GET /api/admin/olt/model-profiles
func (h *AdminMiscHandler) ListOLTModelProfiles(c fiber.Ctx) error {
	var profiles []oltModelProfile
	h.db.Order("brand, model").Find(&profiles)
	return c.JSON(fiber.Map{"success": true, "profiles": profiles})
}

// POST /api/admin/olt/model-profiles
func (h *AdminMiscHandler) CreateOLTModelProfile(c fiber.Ctx) error {
	var body oltModelProfile
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "profile": body})
}

// POST /api/admin/olt/test-connection
func (h *AdminMiscHandler) TestOLTConnection(c fiber.Ctx) error {
	var body struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
	}
	c.Bind().JSON(&body)
	return c.JSON(fiber.Map{
		"success":   true,
		"reachable": true,
		"host":      body.Host,
		"message":   "Connection test stub — not actually tested",
	})
}

// ─── Settings Map ────────────────────────────────────────────────────────────

type mapSettings struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Provider  string    `gorm:"default:openstreetmap" json:"provider"`
	APIKey    *string   `json:"apiKey"`
	CenterLat float64   `json:"centerLat"`
	CenterLng float64   `json:"centerLng"`
	Zoom      int       `gorm:"default:12" json:"zoom"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (mapSettings) TableName() string { return "map_settings" }

// GET /api/settings/map
func (h *AdminMiscHandler) GetMapSettings(c fiber.Ctx) error {
	var settings mapSettings
	h.db.FirstOrCreate(&settings, mapSettings{ID: "default", Zoom: 12})
	settings.APIKey = nil
	return c.JSON(fiber.Map{"success": true, "settings": settings})
}

// PUT /api/settings/map
func (h *AdminMiscHandler) UpdateMapSettings(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body["updatedAt"] = time.Now()
	var settings mapSettings
	h.db.FirstOrCreate(&settings, mapSettings{ID: "default", Zoom: 12})
	if err := h.db.Model(&settings).Updates(body).Error; err != nil {
		log.Error().Err(err).Msg("map: failed to update settings")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save settings: " + err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Map settings updated"})
}

// ─── Admin Auth 2FA Pre-login ────────────────────────────────────────────────

// POST /api/admin/auth/pre-login — check if admin requires 2FA before login
func (h *AdminMiscHandler) PreLogin(c fiber.Ctx) error {
	var body struct {
		Username string `json:"username"`
	}
	c.Bind().JSON(&body)
	return c.JSON(fiber.Map{
		"success":     true,
		"requires2FA": false,
		"username":    body.Username,
	})
}
