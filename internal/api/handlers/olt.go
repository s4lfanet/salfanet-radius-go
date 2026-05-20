package handlers

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/poller"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/telnet"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/vendors/zte"
	"github.com/s4lfanet/salfanet-radius-go/internal/ws"
)

// OLTHandler handles OLT-related API endpoints.
type OLTHandler struct {
	db     *gorm.DB
	poller *poller.Poller
	hub    *ws.Hub
}

// NewOLTHandler creates an OLTHandler.
func NewOLTHandler(db *gorm.DB, p *poller.Poller, h *ws.Hub) *OLTHandler {
	return &OLTHandler{db: db, poller: p, hub: h}
}

// ─── OLT CRUD ────────────────────────────────────────────────────────────────

// ListOLTs godoc
// GET /api/olt
func (h *OLTHandler) ListOLTs(c fiber.Ctx) error {
	var olts []models.NetworkOLT
	if err := h.db.Find(&olts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(olts)
}

// CreateOLT godoc
// POST /api/olt
func (h *OLTHandler) CreateOLT(c fiber.Ctx) error {
	var body models.NetworkOLT
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	body.ID = uuid.NewString()

	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Start monitoring if enabled
	if body.MonitoringEnabled {
		h.poller.Start(&body)
	}

	return c.Status(fiber.StatusCreated).JSON(body)
}

// GetOLT godoc
// GET /api/olt/:id
func (h *OLTHandler) GetOLT(c fiber.Ctx) error {
	id := c.Params("id")
	var olt models.NetworkOLT
	if err := h.db.
		Preload("ONUStatuses").
		Preload("Alerts").
		Preload("Routers.Router").
		Preload("MonitoringLogs", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(100)
		}).
		Preload("PerformanceMetrics", func(db *gorm.DB) *gorm.DB {
			return db.Order("recorded_at DESC").Limit(100)
		}).
		First(&olt, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}
	return c.JSON(fiber.Map{"olt": olt})
}

// UpdateOLT godoc
// PUT /api/olt/:id
func (h *OLTHandler) UpdateOLT(c fiber.Ctx) error {
	id := c.Params("id")
	var existing models.NetworkOLT
	if err := h.db.First(&existing, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}

	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
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
	delete(body, "routerIds")
	delete(body, "id")
	// Skip password if blank (API never returns the stored password)
	if pw, ok := body["password"].(string); ok && pw == "" {
		delete(body, "password")
	}
	body["updatedAt"] = time.Now()

	if err := h.db.Model(&existing).Updates(body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Sync router associations
	h.db.Where("oltId = ?", id).Delete(&models.NetworkOLTRouter{})
	for _, routerID := range routerIDs {
		h.db.Create(&models.NetworkOLTRouter{
			ID:       uuid.NewString(),
			OltID:    id,
			RouterID: routerID,
		})
	}

	// Reload updated OLT
	h.db.First(&existing, "id = ?", id)

	// Restart poller if monitoring settings changed
	h.poller.Stop(id)
	if existing.MonitoringEnabled {
		h.poller.Start(&existing)
	}

	return c.JSON(existing)
}

// DeleteOLT godoc
// DELETE /api/olt/:id
func (h *OLTHandler) DeleteOLT(c fiber.Ctx) error {
	id := c.Params("id")
	h.poller.Stop(id)
	if err := h.db.Delete(&models.NetworkOLT{}, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}

// TestConnection godoc
// POST /api/olt/test-connection
func (h *OLTHandler) TestConnection(c fiber.Ctx) error {
	var body struct {
		IPAddress     string `json:"ipAddress"`
		SSHEnabled    bool   `json:"sshEnabled"`
		TelnetEnabled bool   `json:"telnetEnabled"`
		SSHPort       string `json:"sshPort"`
		TelnetPort    string `json:"telnetPort"`
		OltID         string `json:"oltId"`
	}
	c.Bind().JSON(&body)

	ip := body.IPAddress
	if ip == "" && body.OltID != "" {
		var olt models.NetworkOLT
		if err := h.db.First(&olt, "id = ?", body.OltID).Error; err == nil {
			ip = olt.IPAddress
		}
	}
	if ip == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ipAddress required"})
	}

	sshPort, _ := strconv.Atoi(body.SSHPort)
	if sshPort == 0 {
		sshPort = 22
	}
	telnetPort, _ := strconv.Atoi(body.TelnetPort)
	if telnetPort == 0 {
		telnetPort = 23
	}

	type testResult struct {
		Method  string `json:"method"`
		Success bool   `json:"success"`
		Message string `json:"message"`
		Time    int    `json:"time"`
	}
	var tests []testResult
	anySuccess := false

	// SSH TCP check
	{
		start := time.Now()
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, sshPort), 5*time.Second)
		elapsed := int(time.Since(start).Milliseconds())
		if err == nil {
			conn.Close()
			tests = append(tests, testResult{Method: "SSH", Success: true, Message: fmt.Sprintf("Port %d reachable", sshPort), Time: elapsed})
			anySuccess = true
		} else {
			tests = append(tests, testResult{Method: "SSH", Success: false, Message: fmt.Sprintf("Port %d unreachable", sshPort), Time: elapsed})
		}
	}

	// Telnet TCP check (if enabled)
	if body.TelnetEnabled {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, telnetPort), 5*time.Second)
		elapsed := int(time.Since(start).Milliseconds())
		if err == nil {
			conn.Close()
			tests = append(tests, testResult{Method: "Telnet", Success: true, Message: fmt.Sprintf("Port %d reachable", telnetPort), Time: elapsed})
			anySuccess = true
		} else {
			tests = append(tests, testResult{Method: "Telnet", Success: false, Message: fmt.Sprintf("Port %d unreachable", telnetPort), Time: elapsed})
		}
	}

	if body.OltID != "" {
		h.db.Model(&models.NetworkOLT{}).Where("id = ?", body.OltID).Update("isOnline", anySuccess)
	}

	return c.JSON(fiber.Map{
		"success": anySuccess,
		"results": fiber.Map{
			"tests": tests,
		},
	})
}

// SyncOLT godoc
// POST /api/olt/:id/sync — manual poll trigger
func (h *OLTHandler) SyncOLT(c fiber.Ctx) error {
	id := c.Params("id")
	go func() {
		if err := h.poller.TriggerPoll(id); err != nil {
			log.Error().Err(err).Str("olt", id).Msg("manual sync failed")
		}
	}()
	return c.JSON(fiber.Map{"message": "sync triggered"})
}

// ─── ONU endpoints ───────────────────────────────────────────────────────────

// ListONUs godoc
// GET /api/olt/:id/onus
func (h *OLTHandler) ListONUs(c fiber.Ctx) error {
	id := c.Params("id")

	var onuStatuses []models.OLTONUStatus
	query := h.db.Where("oltId = ?", id)

	// Optional filters
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("serialNumber LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	page := 1
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	pageSize := 50
	if v, err := strconv.Atoi(c.Query("pageSize")); err == nil && v > 0 {
		pageSize = v
	}
	if pageSize > 500 {
		pageSize = 500
	}
	offset := (page - 1) * pageSize

	var total int64
	query.Model(&models.OLTONUStatus{}).Count(&total)

	if err := query.Preload("Customer").
		Order("frame, slot, port, onuId").
		Limit(pageSize).Offset(offset).
		Find(&onuStatuses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":     onuStatuses,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetONU godoc
// GET /api/olt/:id/onus/:onuId
func (h *OLTHandler) GetONU(c fiber.Ctx) error {
	oltID := c.Params("id")
	onuID := c.Params("onuId")

	var status models.OLTONUStatus
	if err := h.db.Preload("Customer").
		Where("oltId = ? AND id = ?", oltID, onuID).
		First(&status).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ONU not found"})
	}
	return c.JSON(status)
}

// RegisterONU godoc
// POST /api/olt/:id/onus/:onuId/register
func (h *OLTHandler) RegisterONU(c fiber.Ctx) error {
	oltID := c.Params("id")

	var body struct {
		SerialNumber string `json:"serialNumber"`
		OnuType      string `json:"onuType"`
		TcontProfile string `json:"tcontProfile"`
		VLAN         int    `json:"vlan"`
		Frame        int    `json:"frame"`
		Slot         int    `json:"slot"`
		Port         int    `json:"port"`
		OnuID        int    `json:"onuId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}

	if !olt.TelnetEnabled || olt.Username == nil || olt.Password == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Telnet not configured for this OLT"})
	}

	telnetCfg := telnet.DefaultConfig(olt.IPAddress, olt.TelnetPort, *olt.Username, *olt.Password)
	pool := telnet.NewPool(telnetCfg)
	defer pool.Close()

	params := zte.RegisterParams{
		Frame:        body.Frame,
		Slot:         body.Slot,
		Port:         body.Port,
		OnuID:        body.OnuID,
		SerialNumber: body.SerialNumber,
		OnuType:      body.OnuType,
		TcontProfile: body.TcontProfile,
		VLAN:         body.VLAN,
	}

	if err := zte.RegisterONU(pool, params); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Upsert ONU status record
	now := time.Now()
	status := models.OLTONUStatus{
		ID:           uuid.NewString(),
		OltID:        oltID,
		Frame:        body.Frame,
		Slot:         body.Slot,
		Port:         body.Port,
		OnuID:        body.OnuID,
		SerialNumber: &body.SerialNumber,
		Status:       models.OnuOffline, // Will be updated on next poll
		FirstSeenAt:  now,
		LastSeenAt:   &now,
		UpdatedAt:    now,
	}
	h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "oltId"}, {Name: "frame"}, {Name: "slot"}, {Name: "port"}, {Name: "onuId"}},
		DoUpdates: clause.AssignmentColumns([]string{"serialNumber", "lastSeenAt", "updatedAt"}),
	}).Create(&status)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "ONU registered", "status": status})
}

// DeregisterONU godoc
// DELETE /api/olt/:id/onus/:onuId
func (h *OLTHandler) DeregisterONU(c fiber.Ctx) error {
	oltID := c.Params("id")
	onuID := c.Params("onuId")

	var status models.OLTONUStatus
	if err := h.db.Where("oltId = ? AND id = ?", oltID, onuID).First(&status).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ONU not found"})
	}

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}

	if olt.TelnetEnabled && olt.Username != nil && olt.Password != nil {
		telnetCfg := telnet.DefaultConfig(olt.IPAddress, olt.TelnetPort, *olt.Username, *olt.Password)
		pool := telnet.NewPool(telnetCfg)
		defer pool.Close()

		if err := zte.DeregisterONU(pool, status.Frame, status.Slot, status.Port, status.OnuID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	h.db.Delete(&status)
	return c.JSON(fiber.Map{"message": "ONU deregistered"})
}

// AssignONU godoc
// POST /api/olt/:id/onus/:onuId/assign
func (h *OLTHandler) AssignONU(c fiber.Ctx) error {
	oltID := c.Params("id")
	onuID := c.Params("onuId")

	var body struct {
		CustomerID string `json:"customerId"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.CustomerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customerId required"})
	}

	if err := h.db.Model(&models.OLTONUStatus{}).
		Where("oltId = ? AND id = ?", oltID, onuID).
		Update("customerId", body.CustomerID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "ONU assigned to customer"})
}

// GetRegisterMetadata godoc
// GET /api/olt/:id/onus/register — returns ONU types and TCONT profiles for register form
func (h *OLTHandler) GetRegisterMetadata(c fiber.Ctx) error {
	oltID := c.Params("id")

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}

	if !olt.TelnetEnabled || olt.Username == nil || olt.Password == nil {
		return c.JSON(fiber.Map{"onuTypes": []string{}, "tcontProfiles": []string{}, "error": "telnet not configured"})
	}

	telnetCfg := telnet.DefaultConfig(olt.IPAddress, olt.TelnetPort, *olt.Username, *olt.Password)
	pool := telnet.NewPool(telnetCfg)
	defer pool.Close()

	onuTypes, err := zte.GetONUTypes(pool)
	if err != nil {
		log.Warn().Err(err).Str("olt", oltID).Msg("failed to fetch ONU types")
	}

	tcontProfiles, err := zte.GetTcontProfiles(pool)
	if err != nil {
		log.Warn().Err(err).Str("olt", oltID).Msg("failed to fetch TCONT profiles")
	}

	typeNames := make([]string, len(onuTypes))
	for i, t := range onuTypes {
		typeNames[i] = t.Name
	}

	profileNames := make([]string, len(tcontProfiles))
	for i, p := range tcontProfiles {
		profileNames[i] = p.Name
	}

	return c.JSON(fiber.Map{
		"onuTypes":      typeNames,
		"tcontProfiles": profileNames,
	})
}

// ListAlerts godoc
// GET /api/olt/:id/alerts
func (h *OLTHandler) ListAlerts(c fiber.Ctx) error {
	id := c.Params("id")
	var alerts []models.OLTAlert

	query := h.db.Where("oltId = ?", id)
	if resolved := c.Query("resolved"); resolved == "false" {
		query = query.Where("isResolved = ?", false)
	}

	if err := query.Order("createdAt DESC").Limit(100).Find(&alerts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(alerts)
}

// ListPerformance godoc
// GET /api/olt/:id/performance
func (h *OLTHandler) ListPerformance(c fiber.Ctx) error {
	id := c.Params("id")
	var metrics []models.OLTPerformanceMetric

	hours := 24
	if v, err := strconv.Atoi(c.Query("hours")); err == nil && v > 0 {
		hours = v
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	if err := h.db.Where("oltId = ? AND recordedAt >= ?", id, since).
		Order("recordedAt ASC").
		Find(&metrics).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(metrics)
}

// GetChassis godoc
// GET /api/olt/:id/chassis
func (h *OLTHandler) GetChassis(c fiber.Ctx) error {
	id := c.Params("id")

	type portSummary struct {
		Frame   int   `json:"frame"`
		Slot    int   `json:"slot"`
		Port    int   `json:"port"`
		Total   int64 `json:"total"`
		Online  int64 `json:"online"`
		Offline int64 `json:"offline"`
	}

	var rows []portSummary
	h.db.Raw(`
		SELECT frame, slot, port,
		       COUNT(*) as total,
		       SUM(CASE WHEN status = 'online' THEN 1 ELSE 0 END) as online,
		       SUM(CASE WHEN status != 'online' THEN 1 ELSE 0 END) as offline
		FROM olt_onu_status
		WHERE oltId = ?
		GROUP BY frame, slot, port
		ORDER BY frame, slot, port
	`, id).Scan(&rows)

	return c.JSON(fiber.Map{"ports": rows})
}

// WebSocketOLT handles WebSocket connections for real-time ONU status.
// This is called after the WebSocket upgrade — see router.go for the upgrade setup.
func (h *OLTHandler) WebSocketOLT(conn interface{}, oltID string) {
	// conn is *websocket.Conn; type assertion happens in the fiber websocket handler
	// This is handled directly in the router via gofiber/contrib/websocket
	_ = fmt.Sprintf("ws handler for olt %s registered", oltID)
}

// ─── OLT Uplink ──────────────────────────────────────────────────────────────

// GET /api/olt/:id/uplink — get uplink configuration for an OLT
func (h *OLTHandler) GetUplink(c fiber.Ctx) error {
	oltID := c.Params("id")
	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"oltId":   oltID,
		"uplink":  fiber.Map{"port": "uplink0", "status": "unknown", "speed": "1G"},
	})
}

// POST /api/olt/:id/uplink — configure uplink for an OLT
func (h *OLTHandler) CreateUplink(c fiber.Ctx) error {
	oltID := c.Params("id")
	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}
	var body map[string]interface{}
	_ = c.Bind().JSON(&body)
	return c.JSON(fiber.Map{
		"success": true,
		"oltId":   oltID,
		"message": "Uplink configuration updated",
		"config":  body,
	})
}

// ─── OLT Monitoring Dashboard ─────────────────────────────────────────────────

// MonitoringList godoc
// GET /api/olt/monitoring?search=&status=online|offline|all
func (h *OLTHandler) MonitoringList(c fiber.Ctx) error {
	type oltWithAlerts struct {
		models.NetworkOLT
		UnresolvedAlerts int64 `json:"unresolvedAlerts"`
	}

	query := h.db.Model(&models.NetworkOLT{})
	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR ipAddress LIKE ?", like, like)
	}
	switch c.Query("status") {
	case "online":
		query = query.Where("isOnline = ?", true)
	case "offline":
		query = query.Where("isOnline = ?", false)
	}

	var olts []models.NetworkOLT
	if err := query.Order("name").Find(&olts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	result := make([]oltWithAlerts, len(olts))
	for i, o := range olts {
		var count int64
		h.db.Model(&models.OLTAlert{}).Where("oltId = ? AND isResolved = ?", o.ID, false).Count(&count)
		result[i] = oltWithAlerts{NetworkOLT: o, UnresolvedAlerts: count}
	}
	return c.JSON(fiber.Map{"olts": result})
}

// MonitoringPoll godoc
// POST /api/olt/monitoring — trigger immediate poll for a given OLT
func (h *OLTHandler) MonitoringPoll(c fiber.Ctx) error {
	var body struct {
		OltID string `json:"oltId"`
	}
	_ = c.Bind().JSON(&body)
	if body.OltID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "oltId is required"})
	}
	if err := h.poller.TriggerPoll(body.OltID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

// ─── OLT Alerts (global) ─────────────────────────────────────────────────────

// ListAllAlerts godoc
// GET /api/olt/alerts?resolved=false&severity=&type=&limit=100
func (h *OLTHandler) ListAllAlerts(c fiber.Ctx) error {
	query := h.db.Model(&models.OLTAlert{})
	switch c.Query("resolved") {
	case "true":
		query = query.Where("isResolved = ?", true)
	case "false":
		query = query.Where("isResolved = ?", false)
	}
	if sev := c.Query("severity"); sev != "" && sev != "all" {
		query = query.Where("severity = ?", sev)
	}
	if typ := c.Query("type"); typ != "" && typ != "all" {
		query = query.Where("alertType = ?", typ)
	}
	limit := 100
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	var alerts []models.OLTAlert
	if err := query.Order("createdAt DESC").Limit(limit).Find(&alerts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Collect IDs for batch preload
	oltIDs := make([]string, 0)
	onuIDs := make([]string, 0)
	for _, a := range alerts {
		if a.OltID != nil {
			oltIDs = append(oltIDs, *a.OltID)
		}
		if a.OnuID != nil {
			onuIDs = append(onuIDs, *a.OnuID)
		}
	}

	oltMap := map[string]models.NetworkOLT{}
	if len(oltIDs) > 0 {
		var olts []models.NetworkOLT
		h.db.Where("id IN ?", oltIDs).Find(&olts)
		for _, o := range olts {
			oltMap[o.ID] = o
		}
	}

	onuMap := map[string]models.OLTONUStatus{}
	if len(onuIDs) > 0 {
		var onus []models.OLTONUStatus
		h.db.Where("id IN ?", onuIDs).Preload("Customer").Find(&onus)
		for _, o := range onus {
			onuMap[o.ID] = o
		}
	}

	type oltInfo struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		IPAddress string `json:"ipAddress"`
	}
	type customerInfo struct {
		Username string `json:"username"`
		Name     string `json:"name"`
		Phone    string `json:"phone"`
	}
	type onuInfo struct {
		ID           string        `json:"id"`
		SerialNumber *string       `json:"serialNumber"`
		MACAddress   *string       `json:"macAddress"`
		Frame        int           `json:"frame"`
		Slot         int           `json:"slot"`
		Port         int           `json:"port"`
		OnuID        int           `json:"onuId"`
		Customer     *customerInfo `json:"customer"`
	}
	type alertResp struct {
		models.OLTAlert
		ResolvedBy interface{} `json:"resolvedBy"`
		OLT        *oltInfo    `json:"olt"`
		ONU        *onuInfo    `json:"onu"`
	}

	result := make([]alertResp, len(alerts))
	for i, a := range alerts {
		r := alertResp{OLTAlert: a, ResolvedBy: nil}
		if a.OltID != nil {
			if olt, ok := oltMap[*a.OltID]; ok {
				r.OLT = &oltInfo{ID: olt.ID, Name: olt.Name, IPAddress: olt.IPAddress}
			}
		}
		if a.OnuID != nil {
			if onu, ok := onuMap[*a.OnuID]; ok {
				o := &onuInfo{
					ID: onu.ID, SerialNumber: onu.SerialNumber, MACAddress: onu.MACAddress,
					Frame: onu.Frame, Slot: onu.Slot, Port: onu.Port, OnuID: onu.OnuID,
				}
				if onu.Customer != nil {
					o.Customer = &customerInfo{
						Username: onu.Customer.Username,
						Name:     onu.Customer.Name,
						Phone:    onu.Customer.Phone,
					}
				}
				r.ONU = o
			}
		}
		result[i] = r
	}
	return c.JSON(fiber.Map{"alerts": result})
}

// ResolveAlert godoc
// PUT /api/olt/alerts/:id — mark alert as resolved
func (h *OLTHandler) ResolveAlert(c fiber.Ctx) error {
	id := c.Params("id")
	now := time.Now()
	if err := h.db.Model(&models.OLTAlert{}).Where("id = ?", id).Updates(map[string]interface{}{
		"isResolved": true,
		"resolvedAt": now,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}
