package cron

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/s4lfanet/salfanet-radius-go/internal/cache"
	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// jobGenieacsSync fetches all devices from GenieACS NBI and updates the in-memory cache.
// This runs every 5 minutes so that device list/status/rxPower/pppoeStatus are always
// fresh without hitting GenieACS on every page load.
func (s *Scheduler) jobGenieacsSync() {
	h := s.startHistory("genieacs_sync")
	defer func() { s.completeHistory(h, recover()) }()

	// Get GenieACS credentials from DB
	var settings models.GenieacsSettings
	if err := s.db.Where("isActive = ?", true).First(&settings).Error; err != nil {
		s.finishHistory(h, "GenieACS not configured, skipped")
		return
	}
	if settings.Host == "" {
		s.finishHistory(h, "GenieACS host empty, skipped")
		return
	}

	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(settings.Username+":"+settings.Password))
	deviceURL := settings.Host + "/devices/?projection=_id,_lastInform,_lastBoot,_registered,_deviceId,VirtualParameters"

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", deviceURL, nil)
	if err != nil {
		s.failHistory(h, fmt.Errorf("create request: %w", err))
		return
	}
	req.Header.Set("Authorization", auth)

	resp, err := client.Do(req)
	if err != nil {
		s.failHistory(h, fmt.Errorf("fetch devices: %w", err))
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.failHistory(h, fmt.Errorf("read body: %w", err))
		return
	}
	if resp.StatusCode != 200 {
		s.failHistory(h, fmt.Errorf("GenieACS returned HTTP %d", resp.StatusCode))
		return
	}

	var rawDevices []map[string]interface{}
	if err := json.Unmarshal(body, &rawDevices); err != nil {
		s.failHistory(h, fmt.Errorf("unmarshal: %w", err))
		return
	}

	// Map to flat structure (same as settings_genieacs.go mapDevice)
	devices := make([]map[string]interface{}, 0, len(rawDevices))
	for _, dev := range rawDevices {
		mapped := mapGenieacsDevice(dev)
		devices = append(devices, mapped)
	}

	// Update cache
	cache.GetGenieacsCache().SetDevices(devices)

	s.finishHistory(h, fmt.Sprintf("Synced %d devices from GenieACS", len(devices)))
	log.Info().Int("devices", len(devices)).Msg("cron: genieacs_sync done")
}

// jobGenieacsSyncSingle fetches a single device from GenieACS and updates the cache.
// Called after a manual refresh to get immediate updated data.
func (s *Scheduler) jobGenieacsSyncSingle(deviceID string) {
	var settings models.GenieacsSettings
	if err := s.db.Where("isActive = ?", true).First(&settings).Error; err != nil {
		return
	}
	if settings.Host == "" {
		return
	}

	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(settings.Username+":"+settings.Password))
	q, _ := json.Marshal(map[string]string{"_id": deviceID})
	deviceURL := settings.Host + "/devices/?query=" + url.QueryEscape(string(q)) + "&projection=_id,_lastInform,_lastBoot,_registered,_deviceId,VirtualParameters"

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", deviceURL, nil)
	if err != nil {
		log.Error().Err(err).Str("device", deviceID).Msg("genieacs_sync_single: create request failed")
		return
	}
	req.Header.Set("Authorization", auth)

	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("device", deviceID).Msg("genieacs_sync_single: fetch failed")
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("device", deviceID).Msg("genieacs_sync_single: read body failed")
		return
	}
	if resp.StatusCode != 200 {
		log.Error().Int("status", resp.StatusCode).Str("device", deviceID).Msg("genieacs_sync_single: bad status")
		return
	}

	var rawDevices []map[string]interface{}
	if err := json.Unmarshal(body, &rawDevices); err != nil || len(rawDevices) == 0 {
		return
	}

	mapped := mapGenieacsDevice(rawDevices[0])
	cache.GetGenieacsCache().SetDevice(deviceID, mapped)
	log.Info().Str("device", deviceID).Msg("genieacs_sync_single: cache updated")
}

// mapGenieacsDevice maps a raw GenieACS device to the flat structure the frontend expects.
// This is a standalone version of mapDevice from settings_genieacs.go (no fiber dependency).
func mapGenieacsDevice(dev map[string]interface{}) map[string]interface{} {
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

	serialNumber := vpValueLocal(vp, "getSerialNumber")
	if serialNumber == "" {
		serialNumber = serialFromDeviceID
	}

	status := "Offline"
	if lastInform != "" {
		if t, err := time.Parse(time.RFC3339, lastInform); err == nil {
			if time.Since(t) < 60*time.Minute {
				status = "Online"
			}
		}
	}

	return map[string]interface{}{
		"_id":           deviceID,
		"serialNumber":  serialNumber,
		"manufacturer":  manufacturer,
		"model":         productClass,
		"oui":           oui,
		"pppoeUsername": vpValueLocal(vp, "pppoeUsername"),
		"pppoeIP":       vpValueLocal(vp, "pppoeIP"),
		"tr069IP":       vpValueLocal(vp, "IPTR069"),
		"rxPower":       vpValueLocal(vp, "RXPower"),
		"ponMode":       vpValueLocal(vp, "getponmode"),
		"uptime":        vpValueLocal(vp, "getdeviceuptime"),
		"status":        status,
		"lastInform":    lastInform,
	}
}

// vpValueLocal is a standalone version of vpValue (no fiber dependency).
func vpValueLocal(vp map[string]interface{}, key string) string {
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
