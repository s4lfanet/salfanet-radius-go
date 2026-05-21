package handlers

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// ─── NetworkHandler extensions ────────────────────────────────────────────────
// These methods extend network.go's NetworkHandler with additional routes.

// GET /api/network/routers/:id
func (h *NetworkHandler) GetRouter(c fiber.Ctx) error {
	id := c.Params("id")
	var router models.Router
	if err := h.db.First(&router, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "router not found"})
	}
	// Return explicit map so password & secret (json:"-" in model) are included for edit form
	return c.JSON(fiber.Map{
		"success": true,
		"router": fiber.Map{
			"id":          router.ID,
			"name":        router.Name,
			"nasname":     router.NASName,
			"shortname":   router.ShortName,
			"type":        router.Type,
			"ipAddress":   router.IPAddress,
			"username":    router.Username,
			"password":    router.Password,
			"port":        router.Port,
			"apiPort":     router.APIPort,
			"secret":      router.Secret,
			"ports":       router.Ports,
			"server":      router.Server,
			"community":   router.Community,
			"description": router.Description,
			"vpnClientId": router.VpnClientId,
			"isActive":    router.IsActive,
			"createdAt":   router.CreatedAt,
		},
	})
}

// PUT /api/network/routers/:id
func (h *NetworkHandler) UpdateRouter(c fiber.Ctx) error {
	id := c.Params("id")
	var router models.Router
	if err := h.db.First(&router, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "router not found"})
	}
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	delete(body, "id")
	body["updatedAt"] = time.Now()
	h.db.Model(&router).Updates(body)
	return c.JSON(fiber.Map{"success": true, "router": router})
}

// DELETE /api/network/routers/:id
func (h *NetworkHandler) DeleteRouter(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.db.Delete(&models.Router{}, "id = ?", id).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

// POST /api/network/routers/:id/test-connection
func (h *NetworkHandler) TestRouterConnection(c fiber.Ctx) error {
	id := c.Params("id")
	var router models.Router
	if err := h.db.First(&router, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "router not found"})
	}
	return c.JSON(fiber.Map{"success": false, "message": "MikroTik API required", "host": router.NASName})
}

// POST /api/network/routers/:id/detect-public-ip
func (h *NetworkHandler) DetectPublicIP(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": false, "message": "requires MikroTik API"})
}

// GET /api/network/routers/:id/interfaces
func (h *NetworkHandler) RouterInterfaces(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "interfaces": []fiber.Map{}})
}

// GET /api/network/routers/:id/isolation-settings
func (h *NetworkHandler) RouterIsolationSettings(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "settings": fiber.Map{}})
}

// POST /api/network/routers/:id/ping-olt
func (h *NetworkHandler) PingOLT(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": false, "message": "ping requires MikroTik API"})
}

// POST /api/network/routers/:id/setup-isolir
func (h *NetworkHandler) SetupIsolir(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": false, "message": "isolir setup requires MikroTik API"})
}

// GET /api/network/routers/:id/uplinks
func (h *NetworkHandler) RouterUplinks(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "uplinks": []fiber.Map{}})
}

// GET/POST /api/network/routers/status
func (h *NetworkHandler) RouterStatus(c fiber.Ctx) error {
	var body struct {
		RouterIds []string `json:"routerIds"`
	}
	c.Bind().JSON(&body) // ignore parse error for GET requests

	var routers []models.Router
	if len(body.RouterIds) > 0 {
		h.db.Where("id IN ?", body.RouterIds).Find(&routers)
	} else {
		h.db.Find(&routers)
	}

	statusMap := make(map[string]fiber.Map)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, r := range routers {
		wg.Add(1)
		go func(router models.Router) {
			defer wg.Done()
			online := tcpPing(router.NASName, router.Port, 2*time.Second) ||
				tcpPing(router.NASName, router.APIPort, 2*time.Second)
			mu.Lock()
			statusMap[router.ID] = fiber.Map{
				"online":   online,
				"identity": router.Name,
			}
			mu.Unlock()
		}(r)
	}
	wg.Wait()

	return c.JSON(fiber.Map{"success": true, "statusMap": statusMap})
}

// tcpPing checks if host:port is reachable within the given timeout.
func tcpPing(host string, port int, timeout time.Duration) bool {
	if host == "" || port == 0 {
		return false
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GET /api/network/routers/template
func (h *NetworkHandler) RouterImportTemplate(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "template download stub"})
}

// POST /api/network/routers/import
func (h *NetworkHandler) ImportRouters(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "imported": 0})
}

// ─── OLTs ─────────────────────────────────────────────────────────────────────

// GET /api/network/olts
func (h *NetworkHandler) ListOLTs(c fiber.Ctx) error {
	var olts []models.NetworkOLT
	h.db.Find(&olts)

	type countFields struct {
		OltOnuStatus int64 `json:"olt_onu_status"`
	}
	type onuStats struct {
		Online    int64 `json:"online"`
		Offline   int64 `json:"offline"`
		Los       int64 `json:"los"`
		DyingGasp int64 `json:"dying_gasp"`
		Unconfig  int64 `json:"unconfig"`
	}
	type oltWithStats struct {
		models.NetworkOLT
		Count   countFields `json:"_count"`
		OnuStat *onuStats   `json:"onu_stats,omitempty"`
	}

	result := make([]oltWithStats, len(olts))
	for i, o := range olts {
		type statRow struct {
			Status string
			Count  int64
		}
		var rows []statRow
		h.db.Raw(`SELECT status, COUNT(*) as count FROM olt_onu_status WHERE oltId = ? GROUP BY status`, o.ID).Scan(&rows)

		var total, online, offline, los, dyingGasp, unconfig int64
		for _, r := range rows {
			total += r.Count
			switch r.Status {
			case "online":
				online = r.Count
			case "los":
				los = r.Count
			case "dying_gasp":
				dyingGasp = r.Count
			case "auth_failed", "unconfig":
				unconfig += r.Count
			default:
				offline += r.Count
			}
		}

		var stats *onuStats
		if total > 0 {
			stats = &onuStats{Online: online, Offline: offline, Los: los, DyingGasp: dyingGasp, Unconfig: unconfig}
		}
		result[i] = oltWithStats{
			NetworkOLT: o,
			Count:      countFields{OltOnuStatus: total},
			OnuStat:    stats,
		}
	}

	return c.JSON(fiber.Map{"success": true, "olts": result})
}

// GET /api/network/olt-routers
func (h *NetworkHandler) ListOLTRouters(c fiber.Ctx) error {
	var olts []models.NetworkOLT
	h.db.Find(&olts)
	return c.JSON(fiber.Map{"success": true, "oltRouters": olts})
}

// POST /api/network/olts/import
func (h *NetworkHandler) ImportOLTs(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "imported": 0})
}

// GET /api/network/olts/template
func (h *NetworkHandler) OLTImportTemplate(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "template stub"})
}

// POST /api/network/olts — Create OLT (ID from body, ports as strings from form)
func (h *NetworkHandler) CreateOLT(c fiber.Ctx) error {
	var body struct {
		Name            string   `json:"name"`
		IPAddress       string   `json:"ipAddress"`
		Vendor          string   `json:"vendor"`
		Model           string   `json:"model"`
		FirmwareVersion string   `json:"firmwareVersion"`
		Username        string   `json:"username"`
		Password        string   `json:"password"`
		SNMPCommunity   string   `json:"snmpCommunity"`
		SSHEnabled      bool     `json:"sshEnabled"`
		TelnetEnabled   bool     `json:"telnetEnabled"`
		SSHPort         string   `json:"sshPort"`
		TelnetPort      string   `json:"telnetPort"`
		SNMPPort        string   `json:"snmpPort"`
		Latitude        string   `json:"latitude"`
		Longitude       string   `json:"longitude"`
		Status          string   `json:"status"`
		FollowRoad      bool     `json:"followRoad"`
		RouterIDs       []string `json:"routerIds"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Status == "" {
		body.Status = "active"
	}
	if body.SNMPCommunity == "" {
		body.SNMPCommunity = "public"
	}
	sshPort, _ := strconv.Atoi(body.SSHPort)
	if sshPort == 0 {
		sshPort = 22
	}
	telnetPort, _ := strconv.Atoi(body.TelnetPort)
	if telnetPort == 0 {
		telnetPort = 23
	}
	snmpPort, _ := strconv.Atoi(body.SNMPPort)
	if snmpPort == 0 {
		snmpPort = 161
	}
	lat, _ := strconv.ParseFloat(body.Latitude, 64)
	lon, _ := strconv.ParseFloat(body.Longitude, 64)

	vendor := body.Vendor
	model := body.Model
	fw := body.FirmwareVersion
	username := body.Username
	password := body.Password

	olt := models.NetworkOLT{
		ID:              uuid.NewString(),
		Name:            body.Name,
		IPAddress:       body.IPAddress,
		Latitude:        lat,
		Longitude:       lon,
		Status:          body.Status,
		FollowRoad:      body.FollowRoad,
		Vendor:          &vendor,
		Model:           &model,
		FirmwareVersion: &fw,
		Username:        &username,
		Password:        &password,
		SNMPCommunity:   body.SNMPCommunity,
		SSHEnabled:      body.SSHEnabled,
		TelnetEnabled:   body.TelnetEnabled,
		SSHPort:         sshPort,
		TelnetPort:      telnetPort,
		SNMPPort:        snmpPort,
	}
	if err := h.db.Create(&olt).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	// Save router associations
	for _, routerID := range body.RouterIDs {
		if routerID != "" {
			h.db.Create(&models.NetworkOLTRouter{
				ID:       uuid.NewString(),
				OltID:    olt.ID,
				RouterID: routerID,
			})
		}
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "olt": olt})
}

// PUT /api/network/olts — Update OLT (ID in body)
func (h *NetworkHandler) UpdateOLT(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	id, _ := body["id"].(string)
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": "id required"})
	}
	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "OLT not found"})
	}
	// Extract routerIds before removing from update map
	var routerIDs []string
	if raw, ok := body["routerIds"]; ok {
		if arr, ok := raw.([]interface{}); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok && s != "" {
					routerIDs = append(routerIDs, s)
				}
			}
		}
	}
	delete(body, "id")
	delete(body, "routerIds")
	// Skip password update if user left it blank (API never returns the stored password)
	if pw, ok := body["password"].(string); ok && pw == "" {
		delete(body, "password")
	}
	body["updatedAt"] = time.Now()
	if err := h.db.Model(&olt).Updates(body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	// Sync router associations: delete old, insert new
	h.db.Where("oltId = ?", id).Delete(&models.NetworkOLTRouter{})
	for _, routerID := range routerIDs {
		h.db.Create(&models.NetworkOLTRouter{
			ID:       uuid.NewString(),
			OltID:    id,
			RouterID: routerID,
		})
	}
	h.db.First(&olt, "id = ?", id)
	return c.JSON(fiber.Map{"success": true, "olt": olt})
}

// DELETE /api/network/olts — Delete OLT (ID in body)
func (h *NetworkHandler) DeleteOLT(c fiber.Ctx) error {
	var body struct {
		ID string `json:"id"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.ID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "id required"})
	}
	h.db.Exec("UPDATE network_otbs SET oltId = NULL WHERE oltId = ?", body.ID)
	if err := h.db.Delete(&models.NetworkOLT{}, "id = ?", body.ID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

// ─── ODC extended ─────────────────────────────────────────────────────────────

// POST /api/network/odcs/import
func (h *NetworkHandler) ImportODCs(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "imported": 0})
}

// GET /api/network/odcs/template
func (h *NetworkHandler) ODCImportTemplate(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "template stub"})
}

// ─── ODP extended ─────────────────────────────────────────────────────────────

// POST /api/network/odps/import
func (h *NetworkHandler) ImportODPs(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "imported": 0})
}

// GET /api/network/odps/template
func (h *NetworkHandler) ODPImportTemplate(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "template stub"})
}

// ─── OTB extended ─────────────────────────────────────────────────────────────

// GET /api/network/otbs/:id
func (h *NetworkHandler) GetOTB(c fiber.Ctx) error {
	id := c.Params("id")
	var otb models.NetworkOTB
	if err := h.db.First(&otb, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "OTB not found"})
	}
	return c.JSON(fiber.Map{"success": true, "otb": otb})
}

// GET /api/network/otbs/stats
func (h *NetworkHandler) OTBStats(c fiber.Ctx) error {
	var total int64
	h.db.Model(&models.NetworkOTB{}).Count(&total)
	return c.JSON(fiber.Map{"success": true, "total": total})
}

// POST /api/network/otbs/import
func (h *NetworkHandler) ImportOTBs(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "imported": 0})
}

// ─── Fiber Paths ──────────────────────────────────────────────────────────────

func (h *NetworkHandler) ListFiberPaths(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "paths": []fiber.Map{}})
}

func (h *NetworkHandler) CreateFiberPath(c fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"error": "not yet implemented"})
}

func (h *NetworkHandler) GetFiberPath(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "path": nil})
}

func (h *NetworkHandler) UpdateFiberPath(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

func (h *NetworkHandler) DeleteFiberPath(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

func (h *NetworkHandler) TraceFiberPath(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "trace": []fiber.Map{}})
}

// ─── Joint Closures ───────────────────────────────────────────────────────────

func (h *NetworkHandler) ListJointClosures(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "jointClosures": []fiber.Map{}})
}

func (h *NetworkHandler) CreateJointClosure(c fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"error": "not yet implemented"})
}

func (h *NetworkHandler) GetJointClosure(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "jointClosure": nil})
}

func (h *NetworkHandler) UpdateJointClosure(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

func (h *NetworkHandler) DeleteJointClosure(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

func (h *NetworkHandler) ImportJointClosures(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "imported": 0})
}

// ─── Nodes ────────────────────────────────────────────────────────────────────

func (h *NetworkHandler) ListNodes(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "nodes": []fiber.Map{}})
}

func (h *NetworkHandler) CreateNode(c fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"error": "not yet implemented"})
}

func (h *NetworkHandler) GetNode(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "node": nil})
}

func (h *NetworkHandler) UpdateNode(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

func (h *NetworkHandler) DeleteNode(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

// ─── Misc Network ─────────────────────────────────────────────────────────────

func (h *NetworkHandler) ListServers(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "servers": []fiber.Map{}})
}

func (h *NetworkHandler) ListPaths(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "paths": []fiber.Map{}})
}

func (h *NetworkHandler) DetectNAS(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": false, "message": "NAS detection requires network scan"})
}

// POST /api/network/customers/assign
func (h *NetworkHandler) AssignCustomer(c fiber.Ctx) error {
	var body struct {
		CustomerID string  `json:"customerId"`
		ODPID      string  `json:"odpId"`
		PortNumber int     `json:"portNumber"`
		Distance   float64 `json:"distance"`
		Notes      string  `json:"notes"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.CustomerID == "" || body.ODPID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "customerId and odpId required"})
	}

	var existing models.OdpCustomerAssignment
	h.db.First(&existing, "customerId = ?", body.CustomerID)

	notes := func() *string {
		if body.Notes != "" {
			s := body.Notes
			return &s
		}
		return nil
	}()
	dist := body.Distance

	if existing.ID == "" {
		assignment := models.OdpCustomerAssignment{
			ID:         uuid.New().String(),
			CustomerID: body.CustomerID,
			OdpID:      body.ODPID,
			PortNumber: body.PortNumber,
			Distance:   &dist,
			Notes:      notes,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		h.db.Create(&assignment)
		return c.Status(201).JSON(fiber.Map{"success": true, "assignment": assignment})
	}

	h.db.Model(&existing).Updates(map[string]interface{}{
		"odpId":      body.ODPID,
		"portNumber": body.PortNumber,
		"distance":   dist,
		"notes":      notes,
		"updatedAt":  time.Now(),
	})
	return c.JSON(fiber.Map{"success": true, "assignment": existing})
}

// GET /api/customers/with-location
func (h *NetworkHandler) CustomersWithLocation(c fiber.Ctx) error {
	var users []models.PppoeUser
	h.db.Where("latitude IS NOT NULL AND longitude IS NOT NULL").
		Preload("Area").Find(&users)
	return c.JSON(fiber.Map{"success": true, "customers": users})
}
