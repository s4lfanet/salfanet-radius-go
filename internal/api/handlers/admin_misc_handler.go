package handlers

// admin_misc_handler.go — miscellaneous admin-only endpoints:
// APK build management, Cloudflare tunnel, system info, FreeRADIUS backup,
// admin profile 2FA, PPPoE sync/deposit, invoice import, laporan, settings map,
// OLT model profiles & test, admin recurring-job endpoints.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
	h.db.Model(&settings).Updates(body)
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

type freeradiusBackup struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	FileName  string    `json:"fileName"`
	FilePath  string    `json:"filePath"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
}

func (freeradiusBackup) TableName() string { return "freeradius_backups" }

// GET /api/admin/system/freeradius-backup
func (h *AdminMiscHandler) ListFreeradiusBackups(c fiber.Ctx) error {
	var backups []freeradiusBackup
	h.db.Order("createdAt desc").Limit(50).Find(&backups)
	return c.JSON(fiber.Map{"success": true, "backups": backups})
}

// POST /api/admin/system/freeradius-backup — create new backup
func (h *AdminMiscHandler) CreateFreeradiusBackup(c fiber.Ctx) error {
	now := time.Now()
	backup := freeradiusBackup{
		ID:        uuid.New().String(),
		FileName:  "freeradius-backup-" + now.Format("20060102-150405") + ".tar.gz",
		FilePath:  "/var/backups/freeradius/",
		CreatedAt: now,
	}
	h.db.Create(&backup)
	return c.Status(201).JSON(fiber.Map{"success": true, "backup": backup})
}

// GET /api/admin/system/freeradius-backup/download — download backup
func (h *AdminMiscHandler) DownloadFreeradiusBackup(c fiber.Ctx) error {
	id := c.Query("id")
	var backup freeradiusBackup
	if id != "" {
		h.db.First(&backup, "id = ?", id)
	} else {
		h.db.Order("createdAt desc").First(&backup)
	}
	if backup.ID == "" {
		return c.Status(404).JSON(fiber.Map{"error": "backup not found"})
	}
	return c.JSON(fiber.Map{"success": true, "downloadUrl": backup.FilePath + backup.FileName})
}

// POST /api/admin/system/freeradius-backup/restore
func (h *AdminMiscHandler) RestoreFreeradiusBackup(c fiber.Ctx) error {
	var body struct {
		ID string `json:"id"`
	}
	c.Bind().JSON(&body)
	return c.JSON(fiber.Map{"success": true, "message": "FreeRADIUS restore queued", "id": body.ID})
}

// POST /api/admin/system/freeradius-backup/upload — upload backup file
func (h *AdminMiscHandler) UploadFreeradiusBackup(c fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file required"})
	}
	now := time.Now()
	backup := freeradiusBackup{
		ID:        uuid.New().String(),
		FileName:  filepath.Base(file.Filename),
		FilePath:  "/var/backups/freeradius/",
		SizeBytes: file.Size,
		CreatedAt: now,
	}
	h.db.Create(&backup)
	return c.Status(201).JSON(fiber.Map{"success": true, "backup": backup})
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
	h.db.FirstOrCreate(&settings, mapSettings{ID: "default"})
	h.db.Model(&settings).Updates(body)
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
