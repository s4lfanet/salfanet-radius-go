package handlers

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

type SettingsGenieacsHandler struct {
	db         *gorm.DB
	httpClient *http.Client
}

func NewSettingsGenieacsHandler(db *gorm.DB) *SettingsGenieacsHandler {
	return &SettingsGenieacsHandler{
		db:         db,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// getCredentials returns GenieACS host + basic auth header from DB
func (h *SettingsGenieacsHandler) getCredentials() (host, authHeader string, err error) {
	var s models.GenieacsSettings
	if err = h.db.Where("isActive = ?", true).First(&s).Error; err != nil {
		return "", "", fmt.Errorf("GenieACS belum dikonfigurasi")
	}
	if s.Host == "" {
		return "", "", fmt.Errorf("GenieACS host tidak dikonfigurasi")
	}
	auth := base64.StdEncoding.EncodeToString([]byte(s.Username + ":" + s.Password))
	return s.Host, "Basic " + auth, nil
}

// notConfiguredErr returns HTTP 200 with notConfigured:true
func (h *SettingsGenieacsHandler) notConfiguredErr(c fiber.Ctx) error {
	return c.Status(200).JSON(fiber.Map{
		"success":       false,
		"notConfigured": true,
		"error":         "GenieACS belum dikonfigurasi",
	})
}

// proxyGET sends a GET to GenieACS and returns the parsed JSON body
func (h *SettingsGenieacsHandler) proxyGET(targetURL, authHeader string) (interface{}, int, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, 500, err
	}
	req.Header.Set("Authorization", authHeader)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, 502, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 500, err
	}
	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), resp.StatusCode, nil
	}
	return result, resp.StatusCode, nil
}

// proxyPOST sends a POST to GenieACS with JSON body
func (h *SettingsGenieacsHandler) proxyPOST(targetURL, authHeader string, payload interface{}) (interface{}, int, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, 500, err
	}
	req, err := http.NewRequest("POST", targetURL, strings.NewReader(string(b)))
	if err != nil {
		return nil, 500, err
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, 502, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 500, err
	}
	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), resp.StatusCode, nil
	}
	return result, resp.StatusCode, nil
}

// vpValue extracts _value from a VirtualParameters entry (which may be a dict or a raw value)
func vpValue(vp map[string]interface{}, key string) string {
	if vp == nil {
		return ""
	}
	v, ok := vp[key]
	if !ok {
		return ""
	}
	if m, ok := v.(map[string]interface{}); ok {
		if val, ok := m["_value"]; ok {
			return fmt.Sprintf("%v", val)
		}
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// mapDevice maps a raw GenieACS device object to the flat structure the frontend expects
func mapDevice(dev map[string]interface{}) fiber.Map {
	deviceID, _ := dev["_id"].(string)
	lastInform, _ := dev["_lastInform"].(string)

	vp, _ := dev["VirtualParameters"].(map[string]interface{})
	deviceIDObj, _ := dev["_deviceId"].(map[string]interface{})

	manufacturer := ""
	oui := ""
	productClass := ""
	serialFromDeviceID := ""
	if deviceIDObj != nil {
		if v, ok := deviceIDObj["_Manufacturer"].(string); ok {
			manufacturer = v
		}
		if v, ok := deviceIDObj["_OUI"].(string); ok {
			oui = v
		}
		if v, ok := deviceIDObj["_ProductClass"].(string); ok {
			productClass = v
		}
		if v, ok := deviceIDObj["_SerialNumber"].(string); ok {
			serialFromDeviceID = v
		}
	}

	serialNumber := vpValue(vp, "getSerialNumber")
	if serialNumber == "" {
		serialNumber = serialFromDeviceID
	}

	// Derive status from lastInform — if within last 15 minutes, consider online
	status := "Offline"
	if lastInform != "" {
		if t, err := time.Parse(time.RFC3339, lastInform); err == nil {
			if time.Since(t) < 15*time.Minute {
				status = "Online"
			}
		}
	}

	return fiber.Map{
		"_id":           deviceID,
		"serialNumber":  serialNumber,
		"manufacturer":  manufacturer,
		"model":         productClass,
		"oui":           oui,
		"pppoeUsername": vpValue(vp, "pppoeUsername"),
		"pppoeIP":       vpValue(vp, "pppoeIP"),
		"tr069IP":       vpValue(vp, "IPTR069"),
		"rxPower":       vpValue(vp, "RXPower"),
		"ponMode":       vpValue(vp, "getponmode"),
		"uptime":        vpValue(vp, "getdeviceuptime"),
		"status":        status,
		"lastInform":    lastInform,
	}
}

// GET /api/settings/genieacs/devices — fetch real device list from GenieACS
func (h *SettingsGenieacsHandler) ListDevices(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/devices/?projection=_id,_lastInform,_lastBoot,_registered,_deviceId,VirtualParameters", auth)
	if err != nil {
		log.Error().Err(err).Msg("genieacs settings: failed to fetch devices")
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status != 200 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status)})
	}

	// GenieACS returns a flat array of device objects
	rawDevices, ok := result.([]interface{})
	if !ok {
		return c.JSON(fiber.Map{"success": true, "devices": []fiber.Map{}, "total": 0})
	}

	devices := make([]fiber.Map, 0, len(rawDevices))
	for _, raw := range rawDevices {
		dev, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		devices = append(devices, mapDevice(dev))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"devices": devices,
		"total":   len(devices),
	})
}

// GET /api/settings/genieacs/devices/:deviceId
func (h *SettingsGenieacsHandler) GetDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/devices/?_id="+url.QueryEscape(deviceID)+"&projection=_id,_lastInform,_lastBoot,_registered,_deviceId,VirtualParameters", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status != 200 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status)})
	}
	devs, ok := result.([]interface{})
	if !ok || len(devs) == 0 {
		return c.JSON(fiber.Map{"success": false, "error": "device not found"})
	}
	dev, ok := devs[0].(map[string]interface{})
	if !ok {
		return c.JSON(fiber.Map{"success": false, "error": "unexpected response format"})
	}
	return c.JSON(fiber.Map{"success": true, "device": mapDevice(dev)})
}

// DELETE /api/settings/genieacs/devices/:deviceId
func (h *SettingsGenieacsHandler) DeleteDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	req, err := http.NewRequest("DELETE", host+"/devices/?_id="+url.QueryEscape(deviceID), nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	req.Header.Set("Authorization", auth)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return c.Status(resp.StatusCode).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", resp.StatusCode)})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Device deleted"})
}

// GET /api/settings/genieacs/devices/:deviceId/detail — full device detail with nested params
func (h *SettingsGenieacsHandler) DeviceDetail(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/devices/?_id="+url.QueryEscape(deviceID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status != 200 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status)})
	}
	devs, ok := result.([]interface{})
	if !ok || len(devs) == 0 {
		return c.JSON(fiber.Map{"success": false, "error": "device not found"})
	}
	dev, ok := devs[0].(map[string]interface{})
	if !ok {
		return c.JSON(fiber.Map{"success": false, "error": "unexpected response format"})
	}

	// Build flat device info from mapDevice, then add extra detail fields
	base := mapDevice(dev)
	vp, _ := dev["VirtualParameters"].(map[string]interface{})
	deviceIDObj, _ := dev["_deviceId"].(map[string]interface{})

	// Extra fields for detail view
	base["txPower"] = ""
	base["macAddress"] = vpValue(vp, "PonMac")
	base["softwareVersion"] = ""
	base["hardwareVersion"] = ""
	base["temp"] = vpValue(vp, "gettemp")
	base["voltage"] = ""
	base["biasCurrent"] = ""
	base["lanIP"] = ""
	base["lanSubnet"] = ""
	base["dhcpEnabled"] = ""
	base["dhcpStart"] = ""
	base["dhcpEnd"] = ""
	base["dns1"] = ""
	base["memoryFree"] = ""
	base["memoryTotal"] = ""
	base["cpuUsage"] = ""
	base["wlanConfigs"] = []interface{}{}
	base["wanConnections"] = []interface{}{}
	base["connectedDevices"] = []interface{}{}
	base["totalConnected"] = 0
	base["isDualBand"] = false
	base["tags"] = []string{}

	// Try to extract from InternetGatewayDevice tree for deeper info
	if igd, ok := dev["InternetGatewayDevice"].(map[string]interface{}); ok {
		base["_raw"] = igd // pass raw tree for frontend to parse if needed
	}

	if deviceIDObj != nil {
		base["oui"] = deviceIDObj["_OUI"]
	}

	return c.JSON(fiber.Map{"success": true, "device": base})
}

// GET /api/settings/genieacs/devices/:deviceId/parameters — fetch all parameters
func (h *SettingsGenieacsHandler) DeviceParameters(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/devices/?_id="+url.QueryEscape(deviceID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status != 200 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status)})
	}
	// Response is an array of devices; take the first one
	devs, ok := result.([]interface{})
	if !ok || len(devs) == 0 {
		return c.JSON(fiber.Map{"success": false, "error": "device not found"})
	}
	dev, ok := devs[0].(map[string]interface{})
	if !ok {
		return c.JSON(fiber.Map{"success": false, "error": "unexpected response format"})
	}
	// Flatten the parameter tree into a list of {path, value, type} objects
	parameters := flattenParameters(dev, "")
	return c.JSON(fiber.Map{"success": true, "parameters": parameters})
}

// POST /api/settings/genieacs/devices/:deviceId/reboot
func (h *SettingsGenieacsHandler) RebootDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	task := fiber.Map{"name": "reboot"}
	result, status, err := h.proxyPOST(host+"/devices/"+url.PathEscape(deviceID)+"/tasks", auth, task)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	return c.JSON(fiber.Map{"success": true, "message": "reboot task queued", "data": result})
}

// POST /api/settings/genieacs/devices/:deviceId/refresh
func (h *SettingsGenieacsHandler) RefreshDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	task := fiber.Map{"name": "getParameterValues", "parameterNames": []string{"InternetGatewayDevice.", "Device."}}
	result, status, err := h.proxyPOST(host+"/devices/"+url.PathEscape(deviceID)+"/tasks", auth, task)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	return c.JSON(fiber.Map{"success": true, "message": "refresh task queued", "data": result})
}

// GET /api/settings/genieacs/tasks — fetch real tasks from GenieACS
func (h *SettingsGenieacsHandler) ListTasks(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/tasks", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	tasks, _ := result.([]interface{})
	if tasks == nil {
		tasks = []interface{}{}
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "tasks": tasks})
}

// POST /api/settings/genieacs/test
func (h *SettingsGenieacsHandler) TestConnection(c fiber.Ctx) error {
	var body struct {
		Host     string `json:"host"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	if body.Host == "" {
		return c.JSON(fiber.Map{"success": false, "error": "Host URL is required"})
	}

	// If password is empty, try to use saved credentials from DB
	password := body.Password
	if password == "" {
		var s models.GenieacsSettings
		if err := h.db.Where("isActive = ?", true).First(&s).Error; err == nil {
			password = s.Password
		}
	}

	// Build GenieACS NBI API URL — try /devices endpoint
	host := strings.TrimRight(body.Host, "/")
	testURL := host + "/devices"

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return c.JSON(fiber.Map{"success": false, "error": "Invalid URL: " + err.Error()})
	}

	// Set Basic Auth
	auth := base64.StdEncoding.EncodeToString([]byte(body.Username + ":" + password))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := client.Do(req)
	if err != nil {
		return c.JSON(fiber.Map{"success": false, "error": "Connection failed: " + err.Error()})
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(fiber.Map{"success": false, "error": "Failed to read response: " + err.Error()})
	}

	// GenieACS NBI returns 200 on success
	if resp.StatusCode == 200 {
		// Try to parse as JSON array to count devices
		deviceCount := 0
		var devices []interface{}
		if err := json.Unmarshal(bodyBytes, &devices); err == nil {
			deviceCount = len(devices)
		}
		return c.JSON(fiber.Map{
			"success":     true,
			"message":     "Connection successful",
			"deviceCount": deviceCount,
		})
	}

	// Non-200 response
	return c.JSON(fiber.Map{
		"success": false,
		"error":   fmt.Sprintf("GenieACS returned HTTP %d: %s", resp.StatusCode, string(bodyBytes[:min(len(bodyBytes), 200)])),
	})
}

// GET /api/settings/genieacs/parameter-display
func (h *SettingsGenieacsHandler) ListParameterDisplay(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "configs": []fiber.Map{}})
}

// PUT /api/settings/genieacs/parameter-display/:id
func (h *SettingsGenieacsHandler) UpdateParameterDisplay(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "updated"})
}

// POST /api/settings/genieacs/parameter-display/reset
func (h *SettingsGenieacsHandler) ResetParameterDisplay(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "reset to defaults"})
}

// GET /api/settings/genieacs/virtual-parameters
func (h *SettingsGenieacsHandler) ListVirtualParameters(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "data": []fiber.Map{}})
}

// GET /api/settings/genieacs/virtual-parameters/:id
func (h *SettingsGenieacsHandler) GetVirtualParameter(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "parameter": nil})
}

// ─── Isolation Templates ──────────────────────────────────────────────────────

// GET /api/settings/isolation/templates
func (h *SettingsGenieacsHandler) ListIsolationTemplates(c fiber.Ctx) error {
	var templates []models.IsolationTemplate
	h.db.Where("isActive = ?", true).Find(&templates)
	return c.JSON(fiber.Map{"success": true, "data": templates})
}

// GET /api/settings/isolation/templates/:id
func (h *SettingsGenieacsHandler) GetIsolationTemplate(c fiber.Ctx) error {
	id := c.Params("id")
	var tmpl models.IsolationTemplate
	if err := h.db.First(&tmpl, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "template not found"})
	}
	return c.JSON(fiber.Map{"success": true, "template": tmpl})
}

// PUT /api/settings/isolation/templates/:id
func (h *SettingsGenieacsHandler) UpdateIsolationTemplate(c fiber.Ctx) error {
	id := c.Params("id")
	var tmpl models.IsolationTemplate
	if err := h.db.First(&tmpl, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "template not found"})
	}
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	delete(body, "id")
	h.db.Model(&tmpl).Updates(body)
	return c.JSON(fiber.Map{"success": true, "template": tmpl})
}

// POST /api/settings/restart-services — restart services
func (h *SettingsGenieacsHandler) RestartServices(c fiber.Ctx) error {
	var body struct {
		Service string `json:"service"` // "freeradius", "nginx", "all"
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "restart triggered for " + body.Service,
		"note":    "actual restart requires privileged access",
	})
}

// GET /api/sessions/realtime — real-time session count (polling endpoint)
func (h *SettingsGenieacsHandler) RealtimeSessions(c fiber.Ctx) error {
	var count int64
	h.db.Model(&models.Radacct{}).Where("acctstoptime IS NULL").Count(&count)
	return c.JSON(fiber.Map{
		"success":  true,
		"active":   count,
		"polledAt": time.Now().Format(time.RFC3339),
	})
}

// checkFreeradiusRunning checks systemd service state via `systemctl is-active`.
func checkFreeradiusRunning() (bool, string) {
	out, err := exec.Command("systemctl", "is-active", "freeradius").Output()
	state := strings.TrimSpace(string(out))
	if err == nil && state == "active" {
		// Get uptime from systemd
		uptimeOut, _ := exec.Command("systemctl", "show", "freeradius",
			"--property=ActiveEnterTimestamp", "--value").Output()
		uptime := strings.TrimSpace(string(uptimeOut))
		if uptime == "" || uptime == "n/a" {
			uptime = "Active"
		}
		return true, uptime
	}
	if state == "" {
		state = "stopped"
	}
	return false, state
}

// GET /api/system/radius — radius system info
func (h *SettingsGenieacsHandler) SystemRadius(c fiber.Ctx) error {
	var totalUsers, activeUsers int64
	h.db.Model(&models.PppoeUser{}).Count(&totalUsers)
	h.db.Model(&models.PppoeUser{}).Where("status = ?", "active").Count(&activeUsers)

	var activeSessions int64
	h.db.Model(&models.Radacct{}).Where("acctstoptime IS NULL").Count(&activeSessions)

	// Check actual systemd service state
	isRunning, uptimeStr := checkFreeradiusRunning()
	status := "stopped"
	if isRunning {
		status = "running"
	}

	return c.JSON(fiber.Map{
		"success":        true,
		"status":         status,
		"uptime":         uptimeStr,
		"totalUsers":     totalUsers,
		"activeUsers":    activeUsers,
		"activeSessions": activeSessions,
	})
}

// GET /api/sse/voucher-updates — SSE endpoint for voucher updates
func (h *SettingsGenieacsHandler) SSEVoucherUpdates(c fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // disable nginx buffering for SSE

	type rawFasthttpCtx interface {
		RequestCtx() *fasthttp.RequestCtx
	}
	rc, ok := c.(rawFasthttpCtx)
	if !ok {
		return c.Status(500).SendString("SSE not supported")
	}

	db := h.db
	rc.RequestCtx().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Notify client that SSE is ready
		if _, err := fmt.Fprintf(w, "event: connected\ndata: {}\n\n"); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		// Send initial voucher stats
		type voucherStats struct {
			Total      int64 `json:"total"`
			Waiting    int64 `json:"waiting"`
			Active     int64 `json:"active"`
			Expired    int64 `json:"expired"`
			TotalValue int64 `json:"totalValue"`
		}
		var total, waiting, active, expired int64
		db.Model(&models.HotspotVoucher{}).Count(&total)
		db.Model(&models.HotspotVoucher{}).Where("status = ?", "UNUSED").Count(&waiting)
		db.Model(&models.HotspotVoucher{}).Where("status = ?", "ACTIVE").Count(&active)
		db.Model(&models.HotspotVoucher{}).Where("status IN ?", []string{"EXPIRED", "USED"}).Count(&expired)
		statsData := voucherStats{Total: total, Waiting: waiting, Active: active, Expired: expired, TotalValue: 0}
		statsJSON, _ := json.Marshal(map[string]interface{}{
			"stats":   statsData,
			"changes": map[string]int{"activated": 0, "expired": 0},
		})
		if _, err := fmt.Fprintf(w, "event: voucher-stats\ndata: %s\n\n", statsJSON); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		// Keep connection alive with heartbeat; exit when client disconnects
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
				return
			}
			if err := w.Flush(); err != nil {
				return
			}
		}
	})
	return nil
}

// suppress unused import
var _ = strconv.Itoa

// flattenParameters recursively walks a GenieACS parameter tree and returns a flat list
// of {path, value, type, writable, object, timestamp} objects for the frontend parameter browser.
func flattenParameters(node interface{}, prefix string) []fiber.Map {
	var params []fiber.Map
	m, ok := node.(map[string]interface{})
	if !ok {
		return params
	}
	for k, v := range m {
		// Skip metadata keys
		if k == "_object" || k == "_timestamp" || k == "_writable" || k == "_type" || k == "_value" {
			continue
		}
		path := prefix + k
		if child, ok := v.(map[string]interface{}); ok {
			// Check if this is a leaf parameter (has _value)
			if _, hasValue := child["_value"]; hasValue {
				val := fmt.Sprintf("%v", child["_value"])
				paramType, _ := child["_type"].(string)
				writable := false
				if w, ok := child["_writable"].(bool); ok {
					writable = w
				}
				isObject := false
				if o, ok := child["_object"].(bool); ok {
					isObject = o
				}
				var timestamp interface{}
				if ts, ok := child["_timestamp"]; ok {
					timestamp = ts
				}
				params = append(params, fiber.Map{
					"path":      path,
					"value":     val,
					"type":      paramType,
					"writable":  writable,
					"object":    isObject,
					"timestamp": timestamp,
				})
			}
			// Recurse into children
			params = append(params, flattenParameters(child, path+".")...)
		}
	}
	return params
}
