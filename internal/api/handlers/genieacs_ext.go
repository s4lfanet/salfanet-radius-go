package handlers

// genieacs_ext.go — full GenieACS proxy routes (devices, provisions, presets,
// virtual-parameters, files, faults, config, backup, auto-provision, tasks retry).
// All methods are added to the existing GenieacsHandler struct.

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// ─── Device routes ────────────────────────────────────────────────────────────

// GET /api/genieacs/devices
func (h *GenieacsHandler) ListDevices(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	query := c.Queries()
	u := host + "/devices"
	if len(query) > 0 {
		params := url.Values{}
		for k, v := range query {
			params.Set(k, v)
		}
		u += "?" + params.Encode()
	}
	result, status, err := h.proxyGET(u, auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	items, _ := result.([]interface{})
	if items == nil {
		items = []interface{}{}
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": items})
}

// GET /api/genieacs/devices/:deviceId
func (h *GenieacsHandler) GetDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/devices/?_id="+url.QueryEscape(deviceID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	devs, ok := result.([]interface{})
	if !ok || len(devs) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "device not found"})
	}
	return c.Status(status).JSON(fiber.Map{"data": devs[0]})
}

// DELETE /api/genieacs/devices/:deviceId
func (h *GenieacsHandler) DeleteDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	status, err := h.proxyDELETE(host+"/devices/?_id="+url.QueryEscape(deviceID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(fiber.Map{"success": status < 400})
}

// GET /api/genieacs/devices/:deviceId/all-parameters
func (h *GenieacsHandler) DeviceAllParameters(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/devices/?_id="+url.QueryEscape(deviceID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	devs, ok := result.([]interface{})
	if !ok || len(devs) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "device not found"})
	}
	dev, ok := devs[0].(map[string]interface{})
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "unexpected response format"})
	}
	parameters := flattenParameters(dev, "")
	return c.Status(status).JSON(fiber.Map{"data": parameters})
}

// POST /api/genieacs/devices/:deviceId/download
func (h *GenieacsHandler) DeviceDownload(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	result, status, err := h.proxyPOST(host+"/devices/"+url.PathEscape(deviceID)+"/tasks", auth, body)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": result})
}

// GET /api/genieacs/devices/:deviceId/parameters
func (h *GenieacsHandler) GetDeviceParameters(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/devices/?_id="+url.QueryEscape(deviceID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	devs, ok := result.([]interface{})
	if !ok || len(devs) == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "device not found"})
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": devs[0]})
}

// POST /api/genieacs/devices/:deviceId/parameters
func (h *GenieacsHandler) SetDeviceParameters(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	task := fiber.Map{"name": "setParameterValues", "parameterValues": body}
	result, status, err := h.proxyPOST(host+"/devices/"+url.PathEscape(deviceID)+"/tasks", auth, task)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": result})
}

// GET /api/genieacs/devices/:deviceId/tasks
func (h *GenieacsHandler) GetDeviceTasks(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/tasks", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	allTasks, _ := result.([]interface{})
	var filtered []interface{}
	for _, t := range allTasks {
		if m, ok := t.(map[string]interface{}); ok {
			if dev, ok := m["device"].(string); ok && dev == deviceID {
				filtered = append(filtered, t)
			}
		}
	}
	if filtered == nil {
		filtered = []interface{}{}
	}
	return c.Status(status).JSON(fiber.Map{"data": filtered})
}

// POST /api/genieacs/devices/:deviceId/tasks
func (h *GenieacsHandler) CreateDeviceTask(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	result, status, err := h.proxyPOST(host+"/devices/"+url.PathEscape(deviceID)+"/tasks", auth, body)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": result})
}

// POST /api/genieacs/devices/:deviceId/wan
func (h *GenieacsHandler) DeviceWAN(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	task := fiber.Map{"name": "setParameterValues", "parameterValues": body}
	result, status, err := h.proxyPOST(host+"/devices/"+url.PathEscape(deviceID)+"/tasks", auth, task)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": result})
}

// GET /api/genieacs/devices/:deviceId/wifi
func (h *GenieacsHandler) GetDeviceWifi(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/devices/?_id="+url.QueryEscape(deviceID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	devs, ok := result.([]interface{})
	if !ok || len(devs) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "device not found"})
	}
	return c.Status(status).JSON(fiber.Map{"data": devs[0]})
}

// POST /api/genieacs/devices/:deviceId/reboot
func (h *GenieacsHandler) RebootDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	task := fiber.Map{"name": "reboot"}
	result, status, err := h.proxyPOST(host+"/devices/"+url.PathEscape(deviceID)+"/tasks", auth, task)
	if err != nil {
		log.Error().Err(err).Str("device", deviceID).Str("task", "reboot").Msg("genieacs: reboot task failed")
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		log.Error().Int("status", status).Str("device", deviceID).Str("task", "reboot").Msg("genieacs: reboot task error")
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	log.Info().Str("device", deviceID).Str("task", "reboot").Msg("genieacs: reboot task queued")
	return c.Status(status).JSON(fiber.Map{"success": true, "data": result})
}

// POST /api/genieacs/devices/:deviceId/refresh
func (h *GenieacsHandler) RefreshDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}

	// 1. Fetch device data to get ConnectionRequestURL and credentials
	devResult, _, _ := h.proxyGET(host+"/devices/?_id="+url.QueryEscape(deviceID), auth)
	crURL, crUser, crPass := "", "", ""
	if arr, ok := devResult.([]interface{}); ok && len(arr) > 0 {
		if dev, ok := arr[0].(map[string]interface{}); ok {
			if igd, ok := dev["InternetGatewayDevice"].(map[string]interface{}); ok {
				if mgmt, ok := igd["ManagementServer"].(map[string]interface{}); ok {
					if cr, ok := mgmt["ConnectionRequestURL"].(map[string]interface{}); ok {
						if v, ok := cr["_value"].(string); ok {
							crURL = v
						}
					}
					if u, ok := mgmt["ConnectionRequestUsername"].(map[string]interface{}); ok {
						if v, ok := u["_value"].(string); ok {
							crUser = v
						}
					}
					if p, ok := mgmt["ConnectionRequestPassword"].(map[string]interface{}); ok {
						if v, ok := p["_value"].(string); ok {
							crPass = v
						}
					}
				}
			}
		}
	}

	// 2. Create task
	task := fiber.Map{"name": "getParameterValues", "parameterNames": []string{"InternetGatewayDevice.DeviceInfo.SerialNumber", "InternetGatewayDevice.DeviceInfo.Manufacturer", "InternetGatewayDevice.DeviceInfo.ModelName", "InternetGatewayDevice.DeviceInfo.SoftwareVersion", "InternetGatewayDevice.ManagementServer.ConnectionRequestURL", "InternetGatewayDevice.ManagementServer.PeriodicInformInterval", "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.ExternalIPAddress", "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Username", "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.ConnectionStatus"}}
	result, status, err := h.proxyPOST(host+"/devices/"+url.PathEscape(deviceID)+"/tasks", auth, task)
	if err != nil {
		log.Error().Err(err).Str("device", deviceID).Str("task", "refresh").Msg("genieacs: refresh task failed")
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		log.Error().Int("status", status).Str("device", deviceID).Str("task", "refresh").Msg("genieacs: refresh task error")
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	log.Info().Str("device", deviceID).Str("task", "refresh").Msg("genieacs: refresh task queued")

	// Extract task ID from result
	taskID := ""
	if m, ok := result.(map[string]interface{}); ok {
		if id, ok := m["_id"].(string); ok {
			taskID = id
		}
	}

	// 3. Send direct connection request to device (bypassing GenieACS which may not reach device)
	if crURL != "" {
		go h.sendDirectConnectionRequest(crURL, crUser, crPass, deviceID)
	} else {
		// Fallback: try GenieACS connection request
		crGenieURL := host + "/devices/" + url.PathEscape(deviceID) + "/tasks?connection_request"
		_, crStatus, crErr := h.proxyPOST(crGenieURL, auth, nil)
		if crErr != nil {
			log.Error().Err(crErr).Str("device", deviceID).Msg("genieacs: connection request failed")
		} else if crStatus >= 400 {
			log.Error().Int("status", crStatus).Str("device", deviceID).Msg("genieacs: connection request error")
		} else {
			log.Info().Str("device", deviceID).Msg("genieacs: connection request sent")
		}
	}

	// 4. Poll task status for up to 20 seconds
	taskExecuted := false
	if taskID != "" {
		for i := 0; i < 10; i++ {
			time.Sleep(2 * time.Second)
			tasksResult, _, _ := h.proxyGET(host+"/tasks", auth)
			if arr, ok := tasksResult.([]interface{}); ok {
				found := false
				for _, t := range arr {
					if tm, ok := t.(map[string]interface{}); ok {
						if id, ok := tm["_id"].(string); ok && id == taskID {
							found = true
							break
						}
					}
				}
				if !found {
					taskExecuted = true
					break
				}
			}
		}
	}

	if taskExecuted {
		log.Info().Str("device", deviceID).Msg("genieacs: refresh task executed successfully")
		return c.Status(status).JSON(fiber.Map{"success": true, "data": result, "taskExecuted": true})
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": result, "taskExecuted": false, "message": "Task queued, device will process on next inform"})
}

// sendDirectConnectionRequest sends a connection request directly to the device
// using digest auth, bypassing GenieACS (which may not have network route to device)
func (h *GenieacsHandler) sendDirectConnectionRequest(crURL, crUser, crPass, deviceID string) {
	client := &http.Client{Timeout: 10 * time.Second}

	// First request without auth to get digest challenge
	req, err := http.NewRequest("GET", crURL, nil)
	if err != nil {
		log.Error().Err(err).Str("device", deviceID).Msg("genieacs: direct CR request creation failed")
		return
	}
	if crUser == "" && crPass == "" {
		// No auth needed, just send
		resp, err := client.Do(req)
		if err != nil {
			log.Error().Err(err).Str("device", deviceID).Msg("genieacs: direct CR request failed")
			return
		}
		defer resp.Body.Close()
		io.ReadAll(resp.Body)
		log.Info().Int("status", resp.StatusCode).Str("device", deviceID).Msg("genieacs: direct CR sent")
		return
	}

	// Try basic auth first
	req.SetBasicAuth(crUser, crPass)
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("device", deviceID).Msg("genieacs: direct CR request failed")
		return
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Info().Int("status", 200).Str("device", deviceID).Msg("genieacs: direct CR sent (basic auth)")
		return
	}

	// If 401, check for digest auth challenge
	if resp.StatusCode == 401 {
		authHeader := resp.Header.Get("WWW-Authenticate")
		if strings.Contains(authHeader, "Digest") {
			// Extract path from full URL for digest URI
			digestURI := crURL
			if u, e := url.Parse(crURL); e == nil {
				digestURI = u.RequestURI()
			}
			// Parse digest challenge and retry with digest auth
			digestAuth := buildDigestAuth(authHeader, digestURI, crUser, crPass, "GET")
			req2, _ := http.NewRequest("GET", crURL, nil)
			req2.Header.Set("Authorization", digestAuth)
			resp2, err := client.Do(req2)
			if err != nil {
				log.Error().Err(err).Str("device", deviceID).Msg("genieacs: direct CR digest request failed")
				return
			}
			defer resp2.Body.Close()
			io.ReadAll(resp2.Body)
			log.Info().Int("status", resp2.StatusCode).Str("device", deviceID).Msg("genieacs: direct CR sent (digest auth)")
			return
		}
	}

	log.Info().Int("status", resp.StatusCode).Str("device", deviceID).Msg("genieacs: direct CR sent")
	_ = bodyBytes
}

// buildDigestAuth builds a Digest auth header from a WWW-Authenticate challenge
func buildDigestAuth(challenge, uri, username, password, method string) string {
	// Parse challenge parameters
	params := parseDigestParams(challenge)
	realm := params["realm"]
	nonce := params["nonce"]
	qop := params["qop"]
	opaque := params["opaque"]

	cnonce := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))))
	nc := "00000001"

	ha1Data := fmt.Sprintf("%s:%s:%s", username, realm, password)
	ha1 := fmt.Sprintf("%x", md5.Sum([]byte(ha1Data)))
	ha2Data := fmt.Sprintf("%s:%s", method, uri)
	ha2 := fmt.Sprintf("%x", md5.Sum([]byte(ha2Data)))

	var response string
	if qop == "auth" {
		respData := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2)
		response = fmt.Sprintf("%x", md5.Sum([]byte(respData)))
	} else {
		respData := fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2)
		response = fmt.Sprintf("%x", md5.Sum([]byte(respData)))
	}

	auth := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`, username, realm, nonce, uri, response)
	if qop != "" {
		auth += fmt.Sprintf(`, qop=%s, nc=%s, cnonce="%s"`, qop, nc, cnonce)
	}
	if opaque != "" {
		auth += fmt.Sprintf(`, opaque="%s"`, opaque)
	}
	return auth
}

// parseDigestParams extracts key=value pairs from a Digest WWW-Authenticate header
func parseDigestParams(challenge string) map[string]string {
	result := make(map[string]string)
	// Remove "Digest " prefix
	s := strings.TrimPrefix(challenge, "Digest ")
	s = strings.TrimSpace(s)

	// Split by comma, but handle quoted values
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		idx := strings.Index(part, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:idx])
		val := strings.TrimSpace(part[idx+1:])
		val = strings.Trim(val, `"`)
		result[key] = val
	}
	return result
}

// POST /api/genieacs/devices/:deviceId/factory-reset
func (h *GenieacsHandler) FactoryResetDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	task := fiber.Map{"name": "factoryReset"}
	result, status, err := h.proxyPOST(host+"/devices/"+url.PathEscape(deviceID)+"/tasks", auth, task)
	if err != nil {
		log.Error().Err(err).Str("device", deviceID).Str("task", "factoryReset").Msg("genieacs: factory-reset task failed")
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		log.Error().Int("status", status).Str("device", deviceID).Str("task", "factoryReset").Msg("genieacs: factory-reset task error")
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	log.Info().Str("device", deviceID).Str("task", "factoryReset").Msg("genieacs: factory-reset task queued")
	return c.Status(status).JSON(fiber.Map{"success": true, "data": result})
}

// ─── Tasks ────────────────────────────────────────────────────────────────────

// POST /api/genieacs/tasks/:taskId/retry
func (h *GenieacsHandler) RetryTask(c fiber.Ctx) error {
	taskID := c.Params("taskId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyPOST(host+"/tasks/"+url.PathEscape(taskID)+"/retry", auth, nil)
	if err != nil {
		log.Error().Err(err).Str("task", taskID).Msg("genieacs: retry task failed")
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		log.Error().Int("status", status).Str("task", taskID).Msg("genieacs: retry task error")
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	log.Info().Str("task", taskID).Msg("genieacs: task retry queued")
	return c.Status(status).JSON(fiber.Map{"success": true, "data": result})
}

// ─── Presets ─────────────────────────────────────────────────────────────────

// GET /api/genieacs/presets
func (h *GenieacsHandler) ListPresets(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/presets", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	items, _ := result.([]interface{})
	if items == nil {
		items = []interface{}{}
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": items})
}

// POST /api/genieacs/presets
func (h *GenieacsHandler) CreatePreset(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	result, status, err := h.proxyPOST(host+"/presets", auth, body)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// GET /api/genieacs/presets/:presetId
func (h *GenieacsHandler) GetPreset(c fiber.Ctx) error {
	presetID := c.Params("presetId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	q := `[["_id","=","` + presetID + `"]]`
	result, status, err := h.proxyGET(host+"/presets?query="+url.QueryEscape(q), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	items, ok := result.([]interface{})
	if !ok || len(items) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "preset not found"})
	}
	return c.Status(status).JSON(fiber.Map{"data": items[0]})
}

// PUT /api/genieacs/presets/:presetId
func (h *GenieacsHandler) UpdatePreset(c fiber.Ctx) error {
	presetID := c.Params("presetId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	return h.proxyPUT(c, host+"/presets/"+url.PathEscape(presetID), auth, body)
}

// DELETE /api/genieacs/presets/:presetId
func (h *GenieacsHandler) DeletePreset(c fiber.Ctx) error {
	presetID := c.Params("presetId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	status, err := h.proxyDELETE(host+"/presets/"+url.PathEscape(presetID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(fiber.Map{"success": status < 400})
}

// ─── Provisions ──────────────────────────────────────────────────────────────

// GET /api/genieacs/provisions
func (h *GenieacsHandler) ListProvisions(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/provisions", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	items, _ := result.([]interface{})
	if items == nil {
		items = []interface{}{}
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": items})
}

// POST /api/genieacs/provisions
func (h *GenieacsHandler) CreateProvision(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	result, status, err := h.proxyPOST(host+"/provisions", auth, body)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// GET /api/genieacs/provisions/:provisionId
func (h *GenieacsHandler) GetProvision(c fiber.Ctx) error {
	provID := c.Params("provisionId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	q := `[["_id","=","` + provID + `"]]`
	result, status, err := h.proxyGET(host+"/provisions?query="+url.QueryEscape(q), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	items, ok := result.([]interface{})
	if !ok || len(items) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "provision not found"})
	}
	return c.Status(status).JSON(fiber.Map{"data": items[0]})
}

// PUT /api/genieacs/provisions/:provisionId
func (h *GenieacsHandler) UpdateProvision(c fiber.Ctx) error {
	provID := c.Params("provisionId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	return h.proxyPUT(c, host+"/provisions/"+url.PathEscape(provID), auth, body)
}

// DELETE /api/genieacs/provisions/:provisionId
func (h *GenieacsHandler) DeleteProvision(c fiber.Ctx) error {
	provID := c.Params("provisionId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	status, err := h.proxyDELETE(host+"/provisions/"+url.PathEscape(provID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(fiber.Map{"success": status < 400})
}

// ─── Virtual Parameters (VP Scripts with DB + GenieACS sync) ───────────────────

// syncVpToGenieACS pushes a VP script to GenieACS NBI and updates sync status
func (h *GenieacsHandler) syncVpToGenieACS(vp *models.GenieacsVpScript) {
	host, auth, err := h.getCredentials()
	if err != nil {
		errMsg := "GenieACS not configured"
		vp.SyncError = &errMsg
		h.db.Save(vp)
		return
	}
	body := map[string]interface{}{
		"_id":    vp.Name,
		"script": vp.Script,
	}
	// GenieACS NBI uses PUT for both create and update of virtual parameters
	_, status, err := h.proxyPUTRaw(host+"/virtual_parameters/"+url.PathEscape(vp.Name), auth, body)
	if err != nil || status >= 400 {
		errMsg := fmt.Sprintf("GenieACS sync failed: HTTP %d", status)
		if err != nil {
			errMsg = err.Error()
		}
		vp.SyncError = &errMsg
		h.db.Save(vp)
		return
	}
	now := time.Now()
	vp.SyncedAt = &now
	vp.SyncError = nil
	h.db.Save(vp)
}

// proxyPUTRaw is a helper that does a PUT and returns raw status
func (h *GenieacsHandler) proxyPUTRaw(targetURL, authHeader string, payload interface{}) (interface{}, int, error) {
	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, 500, err
		}
		bodyReader = strings.NewReader(string(b))
	}
	req, err := http.NewRequest("PUT", targetURL, bodyReader)
	if err != nil {
		return nil, 500, err
	}
	req.Header.Set("Authorization", authHeader)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, 502, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result interface{}
	_ = json.Unmarshal(respBody, &result)
	return result, resp.StatusCode, nil
}

// GET /api/genieacs/virtual-parameters
func (h *GenieacsHandler) ListVirtualParameters(c fiber.Ctx) error {
	var scripts []models.GenieacsVpScript
	if err := h.db.Order("name ASC").Find(&scripts).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	result := make([]fiber.Map, 0, len(scripts))
	for _, s := range scripts {
		entry := fiber.Map{
			"_id":         s.Name,
			"script":      s.Script,
			"description": s.Description,
			"syncedAt":    s.SyncedAt,
			"syncError":   s.SyncError,
		}
		result = append(result, entry)
	}
	return c.JSON(fiber.Map{"success": true, "data": result})
}

// POST /api/genieacs/virtual-parameters
func (h *GenieacsHandler) CreateVirtualParameter(c fiber.Ctx) error {
	var body struct {
		ID          string `json:"_id"`
		Name        string `json:"name"`
		Script      string `json:"script"`
		Description string `json:"description"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid JSON"})
	}
	name := body.Name
	if name == "" {
		name = body.ID
	}
	if name == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "_id or name is required"})
	}
	// Check if already exists
	var existing models.GenieacsVpScript
	if h.db.Where("name = ?", name).First(&existing).Error == nil {
		return c.Status(409).JSON(fiber.Map{"success": false, "error": "VP script with this name already exists"})
	}
	vp := models.GenieacsVpScript{
		ID:     uuid.New().String(),
		Name:   name,
		Script: body.Script,
	}
	if body.Description != "" {
		vp.Description = &body.Description
	}
	if err := h.db.Create(&vp).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	h.syncVpToGenieACS(&vp)
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"_id":         vp.Name,
		"script":      vp.Script,
		"description": vp.Description,
		"syncedAt":    vp.SyncedAt,
		"syncError":   vp.SyncError,
	}})
}

// GET /api/genieacs/virtual-parameters/:vpId
func (h *GenieacsHandler) GetVirtualParameter(c fiber.Ctx) error {
	vpID := c.Params("vpId")
	var vp models.GenieacsVpScript
	if err := h.db.Where("name = ?", vpID).First(&vp).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "virtual parameter not found"})
	}
	return c.JSON(fiber.Map{"data": fiber.Map{
		"_id":         vp.Name,
		"script":      vp.Script,
		"description": vp.Description,
		"syncedAt":    vp.SyncedAt,
		"syncError":   vp.SyncError,
	}})
}

// PUT /api/genieacs/virtual-parameters/:vpId
func (h *GenieacsHandler) UpdateVirtualParameter(c fiber.Ctx) error {
	vpID := c.Params("vpId")
	var vp models.GenieacsVpScript
	if err := h.db.Where("name = ?", vpID).First(&vp).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "virtual parameter not found"})
	}
	var body struct {
		Script      string `json:"script"`
		Description string `json:"description"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid JSON"})
	}
	vp.Script = body.Script
	if body.Description != "" {
		vp.Description = &body.Description
	}
	h.db.Save(&vp)
	h.syncVpToGenieACS(&vp)
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"_id":         vp.Name,
		"script":      vp.Script,
		"description": vp.Description,
		"syncedAt":    vp.SyncedAt,
		"syncError":   vp.SyncError,
	}})
}

// DELETE /api/genieacs/virtual-parameters/:vpId
func (h *GenieacsHandler) DeleteVirtualParameter(c fiber.Ctx) error {
	vpID := c.Params("vpId")
	var vp models.GenieacsVpScript
	if err := h.db.Where("name = ?", vpID).First(&vp).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "virtual parameter not found"})
	}
	// Delete from GenieACS
	host, auth, err := h.getCredentials()
	if err == nil {
		h.proxyDELETE(host+"/virtual_parameters/"+url.PathEscape(vpID), auth)
	}
	// Delete from DB
	h.db.Delete(&vp)
	return c.JSON(fiber.Map{"success": true})
}

// ─── Files ────────────────────────────────────────────────────────────────────

// GET /api/genieacs/files
func (h *GenieacsHandler) ListFiles(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/files", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	items, _ := result.([]interface{})
	if items == nil {
		items = []interface{}{}
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": items})
}

// POST /api/genieacs/files — multipart upload
func (h *GenieacsHandler) UploadFile(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	// Forward raw body to GenieACS
	req, err := http.NewRequest("POST", host+"/files", bytes.NewReader(c.Body()))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", string(c.Request().Header.ContentType()))
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Set("Content-Type", resp.Header.Get("Content-Type"))
	return c.Status(resp.StatusCode).Send(body)
}

// DELETE /api/genieacs/files — delete by filename from query param or body {fileName}
func (h *GenieacsHandler) DeleteFile(c fiber.Ctx) error {
	filename := c.Query("filename")
	if filename == "" {
		var body struct {
			FileName string `json:"fileName"`
		}
		if err := c.Bind().JSON(&body); err == nil {
			filename = body.FileName
		}
	}
	if filename == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "filename required"})
	}
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	u := host + "/files/" + url.PathEscape(filename)
	status, err := h.proxyDELETE(u, auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(status).JSON(fiber.Map{"success": status < 400})
}

// ─── Faults ───────────────────────────────────────────────────────────────────

// GET /api/genieacs/faults
func (h *GenieacsHandler) ListFaults(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	deviceFilter := c.Query("device", "")
	u := host + "/faults"
	if deviceFilter != "" {
		q := `[["device","=","` + deviceFilter + `"]]`
		u += "?query=" + url.QueryEscape(q)
	}
	result, status, err := h.proxyGET(u, auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	items, _ := result.([]interface{})
	if items == nil {
		items = []interface{}{}
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": items})
}

// DELETE /api/genieacs/faults/:faultId  OR  DELETE /api/genieacs/faults (body {id})
func (h *GenieacsHandler) DeleteFault(c fiber.Ctx) error {
	faultID := c.Params("faultId")
	if faultID == "" {
		var body struct {
			ID string `json:"id"`
		}
		if err := c.Bind().JSON(&body); err == nil {
			faultID = body.ID
		}
	}
	if faultID == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "fault id required"})
	}
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	status, err := h.proxyDELETE(host+"/faults/"+url.PathEscape(faultID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(status).JSON(fiber.Map{"success": status < 400})
}

// ─── Sync ─────────────────────────────────────────────────────────────────────

// POST /api/genieacs/sync
func (h *GenieacsHandler) SyncDevices(c fiber.Ctx) error {
	var body struct {
		Types []string `json:"types"`
	}
	_ = c.Bind().JSON(&body)

	// If syncing virtual parameters
	for _, t := range body.Types {
		if t == "virtualParameters" {
			var scripts []models.GenieacsVpScript
			h.db.Find(&scripts)
			success, failed := 0, 0
			for i := range scripts {
				h.syncVpToGenieACS(&scripts[i])
				if scripts[i].SyncError == nil {
					success++
				} else {
					failed++
				}
			}
			return c.JSON(fiber.Map{
				"success": true,
				"data": fiber.Map{
					"virtualParameters": fiber.Map{
						"success": success,
						"failed":  failed,
					},
				},
			})
		}
	}

	// Default: trigger device sync
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	// Trigger a full refresh on all devices
	result, status, err := h.proxyPOST(host+"/devices/force-sync", auth, nil)
	if err != nil {
		// Fallback: just check connectivity
		_, _, pingErr := h.proxyGET(host+"/devices?limit=1", auth)
		if pingErr == nil {
			return c.JSON(fiber.Map{"success": true, "message": "GenieACS connected"})
		}
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// ─── Config ───────────────────────────────────────────────────────────────────

// GET /api/genieacs/config
func (h *GenieacsHandler) ListConfig(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/config", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	items, _ := result.([]interface{})
	if items == nil {
		items = []interface{}{}
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": items})
}

// PUT /api/genieacs/config
func (h *GenieacsHandler) UpdateConfig(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	return h.proxyPUT(c, host+"/config", auth, body)
}

// DELETE /api/genieacs/config — delete by key from query param or body {id}
func (h *GenieacsHandler) DeleteConfig(c fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		var body struct {
			ID string `json:"id"`
		}
		if err := c.Bind().JSON(&body); err == nil {
			key = body.ID
		}
	}
	if key == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "config key required"})
	}
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	u := host + "/config/" + url.PathEscape(key)
	status, err := h.proxyDELETE(u, auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(status).JSON(fiber.Map{"success": status < 400})
}

// ─── Backup ───────────────────────────────────────────────────────────────────

// GET /api/genieacs/backup
func (h *GenieacsHandler) GetBackup(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/backup", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// POST /api/genieacs/backup
func (h *GenieacsHandler) CreateBackup(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyPOST(host+"/backup", auth, nil)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// ─── Auto-Provision ───────────────────────────────────────────────────────────

// GET /api/genieacs/auto-provision
func (h *GenieacsHandler) ListAutoProvision(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/provisions", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	items, _ := result.([]interface{})
	if items == nil {
		items = []interface{}{}
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": fiber.Map{"provisions": items}})
}

// POST /api/genieacs/auto-provision
func (h *GenieacsHandler) CreateAutoProvision(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	result, status, err := h.proxyPOST(host+"/provisions", auth, body)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// DELETE /api/genieacs/auto-provision
func (h *GenieacsHandler) DeleteAutoProvision(c fiber.Ctx) error {
	name := c.Query("name")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	u := host + "/provisions"
	if name != "" {
		u += "/" + url.PathEscape(name)
	}
	status, err := h.proxyDELETE(u, auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(fiber.Map{"success": status < 400})
}

// ─── Helper: proxyPUT ─────────────────────────────────────────────────────────

func (h *GenieacsHandler) proxyPUT(c fiber.Ctx, targetURL, authHeader string, payload interface{}) error {
	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		bodyReader = strings.NewReader(string(b))
	}
	req, err := http.NewRequest("PUT", targetURL, bodyReader)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	req.Header.Set("Authorization", authHeader)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Set("Content-Type", "application/json")
	return c.Status(resp.StatusCode).Send(body)
}
