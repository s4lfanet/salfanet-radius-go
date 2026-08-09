package handlers

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/tzutil"
	"gorm.io/gorm"
)

type FreeradiusHandler struct{ db *gorm.DB }

func NewFreeradiusHandler(db *gorm.DB) *FreeradiusHandler {
	return &FreeradiusHandler{db: db}
}

func runCmd(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// GET /api/freeradius/status
func (h *FreeradiusHandler) GetStatus(c fiber.Ctx) error {
	active, _ := runCmd("systemctl", "is-active", "freeradius")
	running := active == "active"

	var pidInt int
	var uptime, version, startTimeISO string
	var cpu, memory, memoryMB float64

	if running {
		pidStr, _ := runCmd("sh", "-c", "pgrep -x freeradius | head -1")
		pidInt, _ = strconv.Atoi(pidStr)

		if pidInt > 0 {
			psOut, _ := runCmd("sh", "-c", fmt.Sprintf("ps -p %d -o %%cpu,%%mem,rss --no-headers 2>/dev/null", pidInt))
			parts := strings.Fields(psOut)
			if len(parts) >= 3 {
				cpu, _ = strconv.ParseFloat(parts[0], 64)
				memory, _ = strconv.ParseFloat(parts[1], 64)
				rss, _ := strconv.ParseInt(parts[2], 10, 64)
				memoryMB = math.Round(float64(rss)/1024*100) / 100
			}
		}

		rawTS, _ := runCmd("systemctl", "show", "freeradius", "--property=ActiveEnterTimestamp", "--value")
		if rawTS != "" && rawTS != "n/a" {
			tzRe := regexp.MustCompile(`\s+[A-Z]{2,5}$`)
			cleanTS := tzRe.ReplaceAllString(rawTS, "")
			loc := tzutil.Location()
			if t, err := time.ParseInLocation("Mon 2006-01-02 15:04:05", cleanTS, loc); err == nil {
				startTimeISO = t.UTC().Format("2006-01-02T15:04:05.000Z")
				dur := time.Since(t)
				days := int(dur.Hours()) / 24
				hrs := int(dur.Hours()) % 24
				mins := int(dur.Minutes()) % 60
				secs := int(dur.Seconds()) % 60
				if days > 0 {
					uptime = fmt.Sprintf("%d-%02d:%02d:%02d", days, hrs, mins, secs)
				} else {
					uptime = fmt.Sprintf("%02d:%02d:%02d", hrs, mins, secs)
				}
			}
		}

		verOut, _ := runCmd("sh", "-c", "freeradius -v 2>&1 | head -1")
		verRe := regexp.MustCompile(`(\d+\.\d+\.\d+)`)
		if m := verRe.FindString(verOut); m != "" {
			version = m
		}
	}

	var activeConnections, totalAuthRequests, totalAcctRequests int64
	h.db.Model(&models.Radacct{}).Where("acctstoptime IS NULL").Count(&activeConnections)
	h.db.Raw("SELECT COUNT(*) FROM radpostauth").Scan(&totalAuthRequests)
	h.db.Model(&models.Radacct{}).Count(&totalAcctRequests)

	return c.JSON(fiber.Map{
		"success": true,
		"status": fiber.Map{
			"running":           running,
			"pid":               pidInt,
			"uptime":            uptime,
			"cpu":               cpu,
			"memory":            memory,
			"memoryMB":          memoryMB,
			"version":           version,
			"startTime":         startTimeISO,
			"activeConnections": activeConnections,
			"totalAuthRequests": totalAuthRequests,
			"totalAcctRequests": totalAcctRequests,
			"lastRestart":       startTimeISO,
		},
	})
}

// POST /api/freeradius/start
func (h *FreeradiusHandler) Start(c fiber.Ctx) error {
	out, err := runCmd("systemctl", "start", "freeradius")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": out})
	}
	time.Sleep(2 * time.Second)
	active, _ := runCmd("systemctl", "is-active", "freeradius")
	if active != "active" {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Service failed to start"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "FreeRADIUS started successfully"})
}

// POST /api/freeradius/stop
func (h *FreeradiusHandler) Stop(c fiber.Ctx) error {
	out, err := runCmd("systemctl", "stop", "freeradius")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": out})
	}
	time.Sleep(1 * time.Second)
	active, _ := runCmd("systemctl", "is-active", "freeradius")
	if active == "active" {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Service failed to stop"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "FreeRADIUS stopped successfully"})
}

// POST /api/freeradius/restart
func (h *FreeradiusHandler) Restart(c fiber.Ctx) error {
	out, err := runCmd("systemctl", "restart", "freeradius")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": out})
	}
	time.Sleep(2 * time.Second)
	active, _ := runCmd("systemctl", "is-active", "freeradius")
	if active != "active" {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Service failed to restart"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "FreeRADIUS restarted successfully"})
}

// GET /api/freeradius/logs
func (h *FreeradiusHandler) GetLogs(c fiber.Ctx) error {
	out, _ := runCmd("sh", "-c", "journalctl -u freeradius -n 200 --no-pager 2>/dev/null || tail -n 200 /var/log/freeradius/radius.log 2>/dev/null")
	logLines := strings.Split(out, "\n")
	return c.JSON(fiber.Map{"success": true, "logs": logLines})
}

// GET /api/freeradius/radcheck
func (h *FreeradiusHandler) GetRadcheck(c fiber.Ctx) error {
	search := c.Query("search", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	type Row struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Attr     string `json:"attribute"`
		Op       string `json:"op"`
		Value    string `json:"value"`
	}
	var result []Row
	var total int64

	if err := h.db.Raw("SELECT id, username, attribute, op, value FROM radcheck WHERE username LIKE ? LIMIT ? OFFSET ?",
		"%"+search+"%", limit, offset).Scan(&result).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	h.db.Raw("SELECT COUNT(*) FROM radcheck WHERE username LIKE ?", "%"+search+"%").Scan(&total)
	if result == nil {
		result = []Row{}
	}
	return c.JSON(fiber.Map{"success": true, "data": result, "total": total, "page": page, "limit": limit})
}

// POST /api/freeradius/radcheck
func (h *FreeradiusHandler) CreateRadcheck(c fiber.Ctx) error {
	var body struct {
		Username  string `json:"username"`
		Attribute string `json:"attribute"`
		Op        string `json:"op"`
		Value     string `json:"value"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	if err := h.db.Exec("INSERT INTO radcheck (username, attribute, op, value) VALUES (?, ?, ?, ?)",
		body.Username, body.Attribute, body.Op, body.Value).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

// DELETE /api/freeradius/radcheck?id=...
func (h *FreeradiusHandler) DeleteRadcheck(c fiber.Ctx) error {
	id := c.Query("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "id required"})
	}
	if err := h.db.Exec("DELETE FROM radcheck WHERE id = ?", id).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

// POST /api/freeradius/radtest
func (h *FreeradiusHandler) RunRadtest(c fiber.Ctx) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		NasIP    string `json:"nasIP"`
		NasPort  int    `json:"nasPort"`
		Secret   string `json:"secret"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	if body.Username == "" || body.Password == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "username and password are required"})
	}
	server := body.NasIP
	if server == "" {
		server = "127.0.0.1"
	}
	if body.NasPort > 0 && body.NasPort != 1812 {
		server = fmt.Sprintf("%s:%d", server, body.NasPort)
	}
	secret := body.Secret
	if secret == "" {
		secret = "testing123"
	}

	start := time.Now()
	rawOutput, _ := runCmd("radtest", body.Username, body.Password, server, "0", secret)
	duration := int(time.Since(start).Milliseconds())

	isAccept := strings.Contains(rawOutput, "Access-Accept")
	isReject := strings.Contains(rawOutput, "Access-Reject")
	responseType := "Unknown"
	responseCode := "0"
	if isAccept {
		responseType = "Access-Accept"
		responseCode = "2"
	} else if isReject {
		responseType = "Access-Reject"
		responseCode = "3"
	}

	type Attr struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	var attributes []Attr
	attrRe := regexp.MustCompile(`^\s*([^=\s]+)\s*=\s*(.+)$`)
	parsing := false
	for _, line := range strings.Split(rawOutput, "\n") {
		if strings.Contains(line, "Received Access-Accept") || strings.Contains(line, "Received Access-Reject") {
			parsing = true
			continue
		}
		if parsing {
			if m := attrRe.FindStringSubmatch(line); len(m) == 3 {
				attributes = append(attributes, Attr{Name: m[1], Value: strings.TrimSpace(m[2])})
			}
		}
	}
	if attributes == nil {
		attributes = []Attr{}
	}

	return c.JSON(fiber.Map{
		"result": fiber.Map{
			"success":      isAccept,
			"responseCode": responseCode,
			"responseType": responseType,
			"duration":     duration,
			"attributes":   attributes,
			"rawOutput":    rawOutput,
		},
	})
}

// GET /api/freeradius/config/list
func (h *FreeradiusHandler) ListConfigs(c fiber.Ctx) error {
	configDir := "/etc/freeradius/3.0"
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		configDir = "/etc/freeradius"
	}

	type FileItem struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}
	type Group struct {
		ID    string     `json:"id"`
		Name  string     `json:"name"`
		Files []FileItem `json:"files"`
	}

	groups := []Group{
		{
			ID:   "main",
			Name: "Main Configuration",
			Files: []FileItem{
				{Name: "radiusd.conf", Path: "radiusd.conf", Type: "file"},
				{Name: "clients.conf", Path: "clients.conf", Type: "file"},
				{Name: "users", Path: "users", Type: "file"},
				{Name: "proxy.conf", Path: "proxy.conf", Type: "file"},
				{Name: "dictionary", Path: "dictionary", Type: "file"},
			},
		},
	}

	dirLabels := map[string]string{
		"sites-enabled":   "Sites Enabled",
		"sites-available": "Sites Available",
		"mods-enabled":    "Mods Enabled",
		"mods-available":  "Mods Available",
		"policy.d":        "Policy",
	}
	allowedDirs := []string{"sites-enabled", "sites-available", "mods-enabled", "mods-available", "policy.d"}

	for _, dirName := range allowedDirs {
		fullPath := filepath.Join(configDir, dirName)
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			continue
		}
		var files []FileItem
		for _, e := range entries {
			if !e.IsDir() {
				ft := "file"
				if e.Type()&os.ModeSymlink != 0 {
					ft = "link"
				}
				files = append(files, FileItem{
					Name: e.Name(),
					Path: dirName + "/" + e.Name(),
					Type: ft,
				})
			}
		}
		if len(files) > 0 {
			label := dirLabels[dirName]
			if label == "" {
				label = dirName
			}
			groups = append(groups, Group{ID: dirName, Name: label, Files: files})
		}
	}

	return c.JSON(fiber.Map{"success": true, "groups": groups})
}

// POST /api/freeradius/config/read
func (h *FreeradiusHandler) ReadConfig(c fiber.Ctx) error {
	var body struct {
		Filename string `json:"filename"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Filename == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "filename is required"})
	}
	configDir := "/etc/freeradius/3.0"
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		configDir = "/etc/freeradius"
	}
	fullPath := filepath.Join(configDir, filepath.Clean(body.Filename))
	if !strings.HasPrefix(fullPath, configDir) {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": "access denied"})
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "file not found"})
	}
	return c.JSON(fiber.Map{"success": true, "content": string(data)})
}

// POST /api/freeradius/config/save
func (h *FreeradiusHandler) SaveConfig(c fiber.Ctx) error {
	var body struct {
		Filename string `json:"filename"`
		Content  string `json:"content"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	if body.Filename == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "filename is required"})
	}
	configDir := "/etc/freeradius/3.0"
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		configDir = "/etc/freeradius"
	}
	fullPath := filepath.Join(configDir, filepath.Clean(body.Filename))
	if !strings.HasPrefix(fullPath, configDir) {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": "access denied"})
	}
	if err := os.WriteFile(fullPath, []byte(body.Content), 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Config saved"})
}
