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

// proxyPUT sends a PUT to GenieACS with JSON body
func (h *SettingsGenieacsHandler) proxyPUT(targetURL, authHeader string, payload interface{}) (interface{}, int, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, 500, err
	}
	req, err := http.NewRequest("PUT", targetURL, strings.NewReader(string(b)))
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

// proxyDELETE sends a DELETE to GenieACS
func (h *SettingsGenieacsHandler) proxyDELETE(targetURL, authHeader string) (int, error) {
	req, err := http.NewRequest("DELETE", targetURL, nil)
	if err != nil {
		return 500, err
	}
	req.Header.Set("Authorization", authHeader)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return 502, err
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)
	return resp.StatusCode, nil
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

	// Derive status from lastInform — if within last 60 minutes, consider online
	// Uses a generous threshold to account for GenieACS server restarts, NAT timeouts,
	// and network delays. GenieACS UI uses a similar lenient threshold.
	status := "Offline"
	if lastInform != "" {
		if t, err := time.Parse(time.RFC3339, lastInform); err == nil {
			if time.Since(t) < 60*time.Minute {
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
	// GenieACS _id values may contain URL-encoded chars like %2D, %20.
	// Fiber's c.Params already URL-decodes the path segment, so deviceID
	// is the raw stored value (e.g. "80F7A6-FD512XW%2DR460-...").
	// We must re-encode with url.QueryEscape so the HTTP query string
	// preserves %2D (otherwise GenieACS HTTP server decodes %2D to '-').
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

	// Helper to get _value from a parameter object
	getParamVal := func(obj map[string]interface{}, key string) string {
		if p, ok := obj[key].(map[string]interface{}); ok {
			if v, ok := p["_value"]; ok {
				return fmt.Sprintf("%v", v)
			}
		}
		return ""
	}

	// Extra fields for detail view (defaults)
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

	// Parse InternetGatewayDevice tree for detailed info
	if igd, ok := dev["InternetGatewayDevice"].(map[string]interface{}); ok {
		base["_raw"] = igd

		// DeviceInfo
		if di, ok := igd["DeviceInfo"].(map[string]interface{}); ok {
			if v := getParamVal(di, "SoftwareVersion"); v != "" {
				base["softwareVersion"] = v
			}
			if v := getParamVal(di, "HardwareVersion"); v != "" {
				base["hardwareVersion"] = v
			}
			if v := getParamVal(di, "MACAddress"); v != "" {
				base["macAddress"] = v
			}
		}

		// LANDevice
		if lan, ok := igd["LANDevice"].(map[string]interface{}); ok {
			if lan1, ok := lan["1"].(map[string]interface{}); ok {
				// LANHostConfigConfig
				if hc, ok := lan1["LANHostConfigConfig"].(map[string]interface{}); ok {
					base["lanIP"] = getParamVal(hc, "IPInterfaceIPAddress")
					base["lanSubnet"] = getParamVal(hc, "IPInterfaceSubnetMask")
					base["dhcpEnabled"] = getParamVal(hc, "DHCPServerEnable")
					base["dhcpStart"] = getParamVal(hc, "MinAddress")
					base["dhcpEnd"] = getParamVal(hc, "MaxAddress")
				}

				// WLANConfiguration
				if wlan, ok := lan1["WLANConfiguration"].(map[string]interface{}); ok {
					wlanConfigs := []interface{}{}
					for i := 1; i <= 8; i++ {
						wl, ok := wlan[fmt.Sprintf("%d", i)].(map[string]interface{})
						if !ok {
							continue
						}
						ssid := getParamVal(wl, "SSID")
						if ssid == "" {
							continue
						}
						enable := getParamVal(wl, "Enable")
						standard := getParamVal(wl, "Standard")
						channel := getParamVal(wl, "Channel")
						beaconType := getParamVal(wl, "BeaconType")
						authMode := getParamVal(wl, "IEEE11iAuthenticationMode")
						encMode := getParamVal(wl, "IEEE11iEncryptionModes")
						keyPass := getParamVal(wl, "KeyPassphrase")
						totalAssoc := getParamVal(wl, "TotalAssociations")
						assocInt := 0
						if totalAssoc != "" {
							if n, err := strconv.Atoi(totalAssoc); err == nil {
								assocInt = n
							}
						}
						// Determine band from standard
						band := "2.4GHz"
						if strings.Contains(strings.ToUpper(standard), "5") || strings.Contains(strings.ToUpper(standard), "AC") || strings.Contains(strings.ToUpper(standard), "AX") {
							band = "5GHz"
						}
						// Determine security display
						security := beaconType
						if authMode != "" {
							security = authMode
						}
						wlanConfigs = append(wlanConfigs, fiber.Map{
							"index":             i,
							"ssid":              ssid,
							"enabled":           enable == "true" || enable == "True" || enable == "1",
							"channel":           channel,
							"standard":          standard,
							"security":          security,
							"password":          keyPass,
							"band":              band,
							"beaconType":        beaconType,
							"authMode":          authMode,
							"encryptionMode":    encMode,
							"totalAssociations": assocInt,
						})
					}
					base["wlanConfigs"] = wlanConfigs
					// Check dual band
					bands := map[string]bool{}
					for _, wc := range wlanConfigs {
						if wcm, ok := wc.(fiber.Map); ok {
							if b, ok := wcm["band"].(string); ok {
								bands[b] = true
							}
						}
					}
					base["isDualBand"] = len(bands) > 1
				}

				// Hosts (connected devices)
				if hosts, ok := lan1["Hosts"].(map[string]interface{}); ok {
					if host, ok := hosts["Host"].(map[string]interface{}); ok {
						connectedDevices := []interface{}{}
						totalConnected := 0
						for i := 1; i <= 64; i++ {
							h, ok := host[fmt.Sprintf("%d", i)].(map[string]interface{})
							if !ok {
								continue
							}
							hostName := getParamVal(h, "HostName")
							ipAddr := getParamVal(h, "IPAddress")
							macAddr := getParamVal(h, "MACAddress")
							active := getParamVal(h, "Active")
							l2if := getParamVal(h, "Layer2Interface")
							ifType := getParamVal(h, "InterfaceType")
							if hostName == "" && macAddr == "" {
								continue
							}
							connectedDevices = append(connectedDevices, fiber.Map{
								"hostName":        hostName,
								"ipAddress":       ipAddr,
								"macAddress":      macAddr,
								"active":          active == "true" || active == "True" || active == "1",
								"interfaceType":   ifType,
								"layer2Interface": l2if,
							})
							totalConnected++
						}
						base["connectedDevices"] = connectedDevices
						base["totalConnected"] = totalConnected
					}
				}
			}
		}

		// WANDevice
		if wan, ok := igd["WANDevice"].(map[string]interface{}); ok {
			wanConnections := []interface{}{}
			for wi := 1; wi <= 4; wi++ {
				wd, ok := wan[fmt.Sprintf("%d", wi)].(map[string]interface{})
				if !ok {
					continue
				}
				if wanc, ok := wd["WANConnectionDevice"].(map[string]interface{}); ok {
					for wci := 1; wci <= 4; wci++ {
						wcd, ok := wanc[fmt.Sprintf("%d", wci)].(map[string]interface{})
						if !ok {
							continue
						}

						// WANPPPConnection
						if ppp, ok := wcd["WANPPPConnection"].(map[string]interface{}); ok {
							for ci := 1; ci <= 4; ci++ {
								conn, ok := ppp[fmt.Sprintf("%d", ci)].(map[string]interface{})
								if !ok {
									continue
								}
								username := getParamVal(conn, "Username")
								status := getParamVal(conn, "ConnectionStatus")
								if username == "" && status != "Connected" {
									continue
								}
								enable := getParamVal(conn, "Enable")
								wanConnections = append(wanConnections, fiber.Map{
									"wanDeviceIndex":           wi,
									"wanConnectionDeviceIndex": wci,
									"connectionIndex":          ci,
									"connectionType":           "PPPoE",
									"path":                     fmt.Sprintf("InternetGatewayDevice.WANDevice.%d.WANConnectionDevice.%d.WANPPPConnection.%d", wi, wci, ci),
									"name":                     fmt.Sprintf("WANPPPConnection.%d", ci),
									"enable":                   enable == "true" || enable == "True" || enable == "1",
									"connectionStatus":         status,
									"externalIPAddress":        getParamVal(conn, "ExternalIPAddress"),
									"username":                 username,
									"password":                 "",
									"macAddress":               getParamVal(conn, "MACAddress"),
									"dnsServers":               getParamVal(conn, "DNSServers"),
									"vlanId":                   getParamVal(conn, "X_HW_VLAN"),
									"serviceList":              getParamVal(conn, "X_HW_SERVICELIST"),
								})
							}
						}

						// WANIPConnection
						if ipc, ok := wcd["WANIPConnection"].(map[string]interface{}); ok {
							for ci := 1; ci <= 4; ci++ {
								conn, ok := ipc[fmt.Sprintf("%d", ci)].(map[string]interface{})
								if !ok {
									continue
								}
								enable := getParamVal(conn, "Enable")
								status := getParamVal(conn, "ConnectionStatus")
								ipAddr := getParamVal(conn, "ExternalIPAddress")
								if status != "Connected" && ipAddr == "" {
									continue
								}
								wanConnections = append(wanConnections, fiber.Map{
									"wanDeviceIndex":           wi,
									"wanConnectionDeviceIndex": wci,
									"connectionIndex":          ci,
									"connectionType":           "IP",
									"path":                     fmt.Sprintf("InternetGatewayDevice.WANDevice.%d.WANConnectionDevice.%d.WANIPConnection.%d", wi, wci, ci),
									"name":                     fmt.Sprintf("WANIPConnection.%d", ci),
									"enable":                   enable == "true" || enable == "True" || enable == "1",
									"connectionStatus":         status,
									"externalIPAddress":        ipAddr,
									"username":                 "",
									"password":                 "",
									"macAddress":               getParamVal(conn, "MACAddress"),
									"dnsServers":               getParamVal(conn, "DNSServers"),
									"vlanId":                   getParamVal(conn, "X_HW_VLAN"),
									"serviceList":              getParamVal(conn, "X_HW_SERVICELIST"),
								})
							}
						}
					}
				}

				// Optical info from WANDevice.1.X_GponInterafceConfig or similar
				if wi == 1 {
					if gpon, ok := wd["X_GponInterafceConfig"].(map[string]interface{}); ok {
						if v := getParamVal(gpon, "TXPower"); v != "" {
							base["txPower"] = v
						}
						if v := getParamVal(gpon, "Temperature"); v != "" {
							base["temp"] = v
						}
						if v := getParamVal(gpon, "Voltage"); v != "" {
							base["voltage"] = v
						}
					}
				}
			}
			base["wanConnections"] = wanConnections
		}
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
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}

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
		return c.JSON(fiber.Map{"success": true, "message": "refresh task executed", "data": result, "taskExecuted": true})
	}
	return c.JSON(fiber.Map{"success": true, "message": "refresh task queued, device will process on next inform", "data": result, "taskExecuted": false})
}

// sendDirectConnectionRequest sends a connection request directly to the device
// using digest auth, bypassing GenieACS (which may not have network route to device)
func (h *SettingsGenieacsHandler) sendDirectConnectionRequest(crURL, crUser, crPass, deviceID string) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", crURL, nil)
	if err != nil {
		log.Error().Err(err).Str("device", deviceID).Msg("genieacs: direct CR request creation failed")
		return
	}
	if crUser == "" && crPass == "" {
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
	io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Info().Int("status", 200).Str("device", deviceID).Msg("genieacs: direct CR sent (basic auth)")
		return
	}

	// If 401, try digest auth
	if resp.StatusCode == 401 {
		authHeader := resp.Header.Get("WWW-Authenticate")
		if strings.Contains(authHeader, "Digest") {
			digestURI := crURL
			if u, e := url.Parse(crURL); e == nil {
				digestURI = u.RequestURI()
			}
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
	configType := c.Query("configType", "DEVICE_LIST")
	var configs []models.ParameterDisplayConfig
	if err := h.db.Where("configType = ?", configType).Order("displayOrder ASC").Find(&configs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	result := make([]fiber.Map, 0, len(configs))
	for _, cfg := range configs {
		var paths []string
		_ = json.Unmarshal([]byte(cfg.ParameterPaths), &paths)
		var colorCoding interface{}
		if cfg.ColorCoding != nil {
			_ = json.Unmarshal([]byte(*cfg.ColorCoding), &colorCoding)
		}
		entry := fiber.Map{
			"id":             cfg.ID,
			"configType":     cfg.ConfigType,
			"section":        cfg.Section,
			"parameterName":  cfg.ParameterName,
			"label":          cfg.Label,
			"parameterPaths": paths,
			"enabled":        cfg.Enabled,
			"displayOrder":   cfg.DisplayOrder,
		}
		if cfg.ColumnWidth != nil {
			entry["columnWidth"] = *cfg.ColumnWidth
		}
		if cfg.Format != nil {
			entry["format"] = *cfg.Format
		}
		if colorCoding != nil {
			entry["colorCoding"] = colorCoding
		}
		if cfg.Icon != nil {
			entry["icon"] = *cfg.Icon
		}
		result = append(result, entry)
	}
	return c.JSON(fiber.Map{"success": true, "configs": result})
}

// POST /api/settings/genieacs/parameter-display
func (h *SettingsGenieacsHandler) CreateParameterDisplay(c fiber.Ctx) error {
	var body struct {
		ConfigType     string   `json:"configType"`
		Section        string   `json:"section"`
		ParameterName  string   `json:"parameterName"`
		Label          string   `json:"label"`
		ParameterPaths []string `json:"parameterPaths"`
		Enabled        bool     `json:"enabled"`
		DisplayOrder   int      `json:"displayOrder"`
		ColumnWidth    string   `json:"columnWidth"`
		Format         string   `json:"format"`
		ColorCoding    any      `json:"colorCoding"`
		Icon           string   `json:"icon"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid JSON body"})
	}
	pathsJSON, _ := json.Marshal(body.ParameterPaths)
	cfg := models.ParameterDisplayConfig{
		ConfigType:     body.ConfigType,
		Section:        body.Section,
		ParameterName:  body.ParameterName,
		Label:          body.Label,
		ParameterPaths: string(pathsJSON),
		Enabled:        body.Enabled,
		DisplayOrder:   body.DisplayOrder,
	}
	if body.ColumnWidth != "" {
		cfg.ColumnWidth = &body.ColumnWidth
	}
	if body.Format != "" {
		cfg.Format = &body.Format
	}
	if body.Icon != "" {
		cfg.Icon = &body.Icon
	}
	if body.ColorCoding != nil {
		ccJSON, _ := json.Marshal(body.ColorCoding)
		ccStr := string(ccJSON)
		cfg.ColorCoding = &ccStr
	}
	if err := h.db.Create(&cfg).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "config": cfg})
}

// PUT /api/settings/genieacs/parameter-display/:id
func (h *SettingsGenieacsHandler) UpdateParameterDisplay(c fiber.Ctx) error {
	id := c.Params("id")
	var cfg models.ParameterDisplayConfig
	if err := h.db.First(&cfg, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "config not found"})
	}
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid JSON body"})
	}
	updates := map[string]interface{}{}
	if v, ok := body["label"]; ok {
		updates["label"] = v
	}
	if v, ok := body["parameterName"]; ok {
		updates["parameterName"] = v
	}
	if v, ok := body["section"]; ok {
		updates["section"] = v
	}
	if v, ok := body["enabled"]; ok {
		updates["enabled"] = v
	}
	if v, ok := body["displayOrder"]; ok {
		updates["displayOrder"] = v
	}
	if v, ok := body["columnWidth"]; ok {
		if s, ok2 := v.(string); ok2 && s != "" {
			updates["columnWidth"] = s
		} else {
			updates["columnWidth"] = nil
		}
	}
	if v, ok := body["format"]; ok {
		if s, ok2 := v.(string); ok2 && s != "" {
			updates["format"] = s
		} else {
			updates["format"] = nil
		}
	}
	if v, ok := body["icon"]; ok {
		if s, ok2 := v.(string); ok2 && s != "" {
			updates["icon"] = s
		} else {
			updates["icon"] = nil
		}
	}
	if v, ok := body["colorCoding"]; ok {
		if v != nil {
			ccJSON, _ := json.Marshal(v)
			updates["colorCoding"] = string(ccJSON)
		} else {
			updates["colorCoding"] = nil
		}
	}
	if v, ok := body["parameterPaths"]; ok {
		pathsJSON, _ := json.Marshal(v)
		updates["parameterPaths"] = string(pathsJSON)
	}
	if len(updates) > 0 {
		if err := h.db.Model(&cfg).Updates(updates).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
		}
	}
	return c.JSON(fiber.Map{"success": true, "message": "updated"})
}

// PUT /api/settings/genieacs/parameter-display (bulk update ordering)
func (h *SettingsGenieacsHandler) BulkUpdateParameterDisplay(c fiber.Ctx) error {
	var body struct {
		Configs []struct {
			ID           int `json:"id"`
			DisplayOrder int `json:"displayOrder"`
		} `json:"configs"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid JSON body"})
	}
	for _, item := range body.Configs {
		h.db.Model(&models.ParameterDisplayConfig{}).Where("id = ?", item.ID).Update("displayOrder", item.DisplayOrder)
	}
	return c.JSON(fiber.Map{"success": true, "message": "order saved"})
}

// DELETE /api/settings/genieacs/parameter-display/:id
func (h *SettingsGenieacsHandler) DeleteParameterDisplay(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.db.Delete(&models.ParameterDisplayConfig{}, id).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "deleted"})
}

// POST /api/settings/genieacs/parameter-display/reset
func (h *SettingsGenieacsHandler) ResetParameterDisplay(c fiber.Ctx) error {
	h.db.Where("1=1").Delete(&models.ParameterDisplayConfig{})
	defaults := []models.ParameterDisplayConfig{
		{ConfigType: "DEVICE_LIST", Section: "main", ParameterName: "serialNumber", Label: "Device", ParameterPaths: `["VirtualParameters.getSerialNumber"]`, Enabled: true, DisplayOrder: 1},
		{ConfigType: "DEVICE_LIST", Section: "main", ParameterName: "pppoeUsername", Label: "Network", ParameterPaths: `["VirtualParameters.pppoeUsername"]`, Enabled: true, DisplayOrder: 2},
		{ConfigType: "DEVICE_LIST", Section: "main", ParameterName: "rxPower", Label: "Signal", ParameterPaths: `["VirtualParameters.RXPower"]`, Enabled: true, DisplayOrder: 3},
		{ConfigType: "DEVICE_LIST", Section: "main", ParameterName: "lastInform", Label: "Last Inform", ParameterPaths: `["_lastInform"]`, Enabled: true, DisplayOrder: 4},
		{ConfigType: "DEVICE_LIST", Section: "main", ParameterName: "connectionStatus", Label: "Status", ParameterPaths: `["_registered"]`, Enabled: true, DisplayOrder: 5},
	}
	for _, d := range defaults {
		h.db.Create(&d)
	}
	return c.JSON(fiber.Map{"success": true, "message": "reset to defaults"})
}

// GET /api/settings/genieacs/virtual-parameters — fetch from GenieACS
func (h *SettingsGenieacsHandler) ListVirtualParameters(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	result, status, err := h.proxyGET(host+"/virtual_parameters", auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	items, _ := result.([]interface{})
	if items == nil {
		items = []interface{}{}
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": items})
}

// GET /api/settings/genieacs/virtual-parameters/:id — fetch single VP from GenieACS
func (h *SettingsGenieacsHandler) GetVirtualParameter(c fiber.Ctx) error {
	vpID := c.Params("id")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	q := `[["_id","=","` + vpID + `"]]`
	result, status, err := h.proxyGET(host+"/virtual_parameters?query="+url.QueryEscape(q), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	items, ok := result.([]interface{})
	if !ok || len(items) == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "virtual parameter not found"})
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": items[0]})
}

// POST /api/settings/genieacs/virtual-parameters — create VP in GenieACS
func (h *SettingsGenieacsHandler) CreateVirtualParameter(c fiber.Ctx) error {
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid JSON body"})
	}
	// Map frontend fields to GenieACS format
	genieBody := fiber.Map{}
	if name, ok := body["name"].(string); ok {
		genieBody["_id"] = name
	}
	if expr, ok := body["expression"].(string); ok {
		genieBody["script"] = expr
	}
	// Also pass through any _id or script fields directly
	if id, ok := body["_id"].(string); ok {
		genieBody["_id"] = id
	}
	if script, ok := body["script"].(string); ok {
		genieBody["script"] = script
	}
	result, status, err := h.proxyPOST(host+"/virtual_parameters", auth, genieBody)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": result})
}

// PUT /api/settings/genieacs/virtual-parameters/:id — update VP in GenieACS
func (h *SettingsGenieacsHandler) UpdateVirtualParameter(c fiber.Ctx) error {
	vpID := c.Params("id")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid JSON body"})
	}
	// Map frontend fields to GenieACS format
	genieBody := fiber.Map{}
	if expr, ok := body["expression"].(string); ok {
		genieBody["script"] = expr
	}
	if script, ok := body["script"].(string); ok {
		genieBody["script"] = script
	}
	if name, ok := body["name"].(string); ok && name != vpID {
		genieBody["_id"] = name
	}
	result, status, err := h.proxyPUT(host+"/virtual_parameters/"+url.PathEscape(vpID), auth, genieBody)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if status >= 400 {
		return c.Status(status).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("GenieACS returned HTTP %d", status), "details": result})
	}
	return c.Status(status).JSON(fiber.Map{"success": true, "data": result})
}

// DELETE /api/settings/genieacs/virtual-parameters/:id — delete VP from GenieACS
func (h *SettingsGenieacsHandler) DeleteVirtualParameter(c fiber.Ctx) error {
	vpID := c.Params("id")
	host, auth, err := h.getCredentials()
	if err != nil {
		return h.notConfiguredErr(c)
	}
	status, err := h.proxyDELETE(host+"/virtual_parameters/"+url.PathEscape(vpID), auth)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(status).JSON(fiber.Map{"success": status < 400})
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
