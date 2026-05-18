package handlers

// genieacs_ext.go — full GenieACS proxy routes (devices, provisions, presets,
// virtual-parameters, files, faults, config, backup, auto-provision, tasks retry).
// All methods are added to the existing GenieacsHandler struct.

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
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
	return c.Status(status).JSON(result)
}

// GET /api/genieacs/devices/:deviceId
func (h *GenieacsHandler) GetDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/devices/"+url.PathEscape(deviceID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// DELETE /api/genieacs/devices/:deviceId
func (h *GenieacsHandler) DeleteDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	status, err := h.proxyDELETE(host+"/devices/"+url.PathEscape(deviceID), auth)
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
	result, status, err := h.proxyGET(host+"/devices/"+url.PathEscape(deviceID)+"/all-parameters", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
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
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// GET /api/genieacs/devices/:deviceId/parameters
func (h *GenieacsHandler) GetDeviceParameters(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/devices/"+url.PathEscape(deviceID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
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
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// GET /api/genieacs/devices/:deviceId/tasks
func (h *GenieacsHandler) GetDeviceTasks(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	q := `[["deviceId","=","` + deviceID + `"]]`
	result, status, err := h.proxyGET(host+"/tasks?query="+url.QueryEscape(q), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
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
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
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
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// GET /api/genieacs/devices/:deviceId/wifi
func (h *GenieacsHandler) GetDeviceWifi(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/devices/"+url.PathEscape(deviceID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
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
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// POST /api/genieacs/devices/:deviceId/refresh
func (h *GenieacsHandler) RefreshDevice(c fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	task := fiber.Map{"name": "getParameterValues", "parameterNames": []string{"InternetGatewayDevice.", "Device."}}
	result, status, err := h.proxyPOST(host+"/devices/"+url.PathEscape(deviceID)+"/tasks", auth, task)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
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
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// ─── Tasks ────────────────────────────────────────────────────────────────────

// POST /api/genieacs/tasks/:taskId/retry
func (h *GenieacsHandler) RetryTask(c fiber.Ctx) error {
	taskID := c.Params("taskId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyPOST(host+"/tasks/"+taskID+"/retry", auth, nil)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
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
	return c.Status(status).JSON(result)
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
	result, status, err := h.proxyGET(host+"/presets/"+url.PathEscape(presetID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
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
	return c.Status(status).JSON(result)
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
	result, status, err := h.proxyGET(host+"/provisions/"+url.PathEscape(provID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
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

// ─── Virtual Parameters ───────────────────────────────────────────────────────

// GET /api/genieacs/virtual-parameters
func (h *GenieacsHandler) ListVirtualParameters(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/virtual-parameters", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// POST /api/genieacs/virtual-parameters
func (h *GenieacsHandler) CreateVirtualParameter(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	result, status, err := h.proxyPOST(host+"/virtual-parameters", auth, body)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// GET /api/genieacs/virtual-parameters/:vpId
func (h *GenieacsHandler) GetVirtualParameter(c fiber.Ctx) error {
	vpID := c.Params("vpId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/virtual-parameters/"+url.PathEscape(vpID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// PUT /api/genieacs/virtual-parameters/:vpId
func (h *GenieacsHandler) UpdateVirtualParameter(c fiber.Ctx) error {
	vpID := c.Params("vpId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body interface{}
	_ = c.Bind().JSON(&body)
	return h.proxyPUT(c, host+"/virtual-parameters/"+url.PathEscape(vpID), auth, body)
}

// DELETE /api/genieacs/virtual-parameters/:vpId
func (h *GenieacsHandler) DeleteVirtualParameter(c fiber.Ctx) error {
	vpID := c.Params("vpId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	status, err := h.proxyDELETE(host+"/virtual-parameters/"+url.PathEscape(vpID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(fiber.Map{"success": status < 400})
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
	return c.Status(status).JSON(result)
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

// DELETE /api/genieacs/files — delete by filename query param
func (h *GenieacsHandler) DeleteFile(c fiber.Ctx) error {
	filename := c.Query("filename")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	u := host + "/files"
	if filename != "" {
		u += "/" + url.PathEscape(filename)
	}
	status, err := h.proxyDELETE(u, auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
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
	q := c.Query("query", "")
	u := host + "/faults"
	if q != "" {
		u += "?query=" + url.QueryEscape(q)
	}
	result, status, err := h.proxyGET(u, auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
}

// DELETE /api/genieacs/faults/:faultId
func (h *GenieacsHandler) DeleteFault(c fiber.Ctx) error {
	faultID := c.Params("faultId")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	status, err := h.proxyDELETE(host+"/faults/"+url.PathEscape(faultID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(fiber.Map{"success": status < 400})
}

// ─── Sync ─────────────────────────────────────────────────────────────────────

// POST /api/genieacs/sync
func (h *GenieacsHandler) SyncDevices(c fiber.Ctx) error {
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
	return c.Status(status).JSON(result)
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

// DELETE /api/genieacs/config
func (h *GenieacsHandler) DeleteConfig(c fiber.Ctx) error {
	key := c.Query("key")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	u := host + "/config"
	if key != "" {
		u += "/" + url.PathEscape(key)
	}
	status, err := h.proxyDELETE(u, auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
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
	result, status, err := h.proxyGET(host+"/provisions?type=auto", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(status).JSON(result)
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
