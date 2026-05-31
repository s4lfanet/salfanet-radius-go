package handlers

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/poller"
	snmputil "github.com/s4lfanet/salfanet-radius-go/internal/olt/snmp"
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
	if err := h.db.Preload("Routers.Router").Find(&olts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Gather ONU counts grouped by oltId + status in one query
	type statRow struct {
		OltID  string `gorm:"column:oltId"`
		Status string `gorm:"column:status"`
		Count  int    `gorm:"column:cnt"`
	}
	var rows []statRow
	h.db.Raw(`SELECT oltId, status, COUNT(*) AS cnt FROM olt_onu_status GROUP BY oltId, status`).Scan(&rows)

	type onuStats struct {
		Online    int `json:"online"`
		Offline   int `json:"offline"`
		LOS       int `json:"los"`
		DyingGasp int `json:"dying_gasp"`
		Unconfig  int `json:"unconfig"`
		Total     int `json:"total"`
	}
	statsMap := make(map[string]*onuStats)
	for _, r := range rows {
		if _, ok := statsMap[r.OltID]; !ok {
			statsMap[r.OltID] = &onuStats{}
		}
		s := statsMap[r.OltID]
		s.Total += r.Count
		switch r.Status {
		case "online":
			s.Online += r.Count
		case "offline":
			s.Offline += r.Count
		case "los":
			s.LOS += r.Count
		case "dying_gasp":
			s.DyingGasp += r.Count
		case "auth_failed":
			s.Unconfig += r.Count
		}
	}

	type oltResponse struct {
		models.NetworkOLT
		Count    map[string]int `json:"_count"`
		OnuStats *onuStats      `json:"onu_stats,omitempty"`
	}

	result := make([]oltResponse, 0, len(olts))
	for _, olt := range olts {
		s := statsMap[olt.ID]
		total := 0
		if s != nil {
			total = s.Total
		}
		result = append(result, oltResponse{
			NetworkOLT: olt,
			Count:      map[string]int{"olt_onu_status": total, "odps": 0},
			OnuStats:   s,
		})
	}
	return c.JSON(result)
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
		Preload("Alerts", func(db *gorm.DB) *gorm.DB {
			return db.Where("isResolved = ?", false).Order("createdAt DESC").Limit(50)
		}).
		Preload("Routers.Router").
		Preload("MonitoringLogs", func(db *gorm.DB) *gorm.DB {
			return db.Order("createdAt DESC").Limit(100)
		}).
		Preload("PerformanceMetrics", func(db *gorm.DB) *gorm.DB {
			return db.Order("recordedAt DESC").Limit(100)
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
	return c.JSON(fiber.Map{"background": true, "message": "Sync started — data will refresh automatically"})
}

// ─── ONU endpoints ───────────────────────────────────────────────────────────

// ListONUs godoc
// GET /api/olt/:id/onus — returns ONU statuses with customer + ODP info.
// ?all=true returns all ONUs without pagination (for real-time polling).
func (h *OLTHandler) ListONUs(c fiber.Ctx) error {
	id := c.Params("id")

	type onuRow struct {
		ID               string     `json:"id"`
		Frame            int        `json:"frame"`
		Slot             int        `json:"slot"`
		Port             int        `json:"port"`
		OnuID            int        `json:"onuId"`
		OnuIndex         int        `json:"onuIndex"`
		SerialNumber     *string    `json:"serialNumber"`
		MACAddress       *string    `json:"macAddress"`
		Description      *string    `json:"description"`
		Status           string     `json:"status"`
		RxPower          *float64   `json:"rxPower"`
		TxPower          *float64   `json:"txPower"`
		Distance         *int       `json:"distance"`
		Temperature      *float64   `json:"temperature"`
		Voltage          *float64   `json:"voltage"`
		BiasCurrent      *float64   `json:"biasCurrent"`
		BandwidthUp      int64      `json:"bandwidthUp"`
		BandwidthDown    int64      `json:"bandwidthDown"`
		CustomerID       *string    `json:"customerId"`
		CustomerName     *string    `json:"customerName"`
		CustomerUsername *string    `json:"customerUsername"`
		CustomerPhone    *string    `json:"customerPhone"`
		LastSeenAt       *time.Time `json:"lastSeenAt"`
		LastOfflineAt    *time.Time `json:"lastOfflineAt"`
		OdpID            *string    `json:"odpId"`
		OdpName          *string    `json:"odpName"`
	}

	whereClause := "o.oltId = ?"
	args := []interface{}{id}
	if status := c.Query("status"); status != "" {
		whereClause += " AND o.status = ?"
		args = append(args, status)
	}
	if search := c.Query("search"); search != "" {
		whereClause += " AND (o.serialNumber LIKE ? OR o.description LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	baseSQL := `
		SELECT o.id, o.frame, o.slot, o.port, o.onuId, o.onuIndex,
		       o.serialNumber, o.macAddress, o.description, o.status,
		       CASE WHEN o.rxPower IS NOT NULL AND o.rxPower <= 30 THEN o.rxPower ELSE NULL END AS rxPower,
		       CASE WHEN o.txPower IS NOT NULL AND o.txPower <= 30 THEN o.txPower ELSE NULL END AS txPower,
		       o.distance, o.temperature, o.voltage, o.biasCurrent,
		       o.bandwidthUp, o.bandwidthDown, o.customerId,
		       o.lastSeenAt, o.lastOfflineAt,
		       u.name  AS customerName,
		       u.username AS customerUsername,
		       u.phone AS customerPhone,
		       o.odpId AS odpId,
		       odp.name AS odpName
		FROM olt_onu_status o
		LEFT JOIN pppoe_users u ON u.id = o.customerId
		LEFT JOIN network_odps odp ON odp.id = o.odpId
		WHERE ` + whereClause + `
		ORDER BY o.slot, o.port, o.onuId`

	// ?all=true → skip pagination
	if c.Query("all") == "true" {
		var rows []onuRow
		if err := h.db.Raw(baseSQL, args...).Scan(&rows).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true, "onus": rows, "total": len(rows)})
	}

	// Paginated response
	page := 1
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	pageSize := 200
	if v, err := strconv.Atoi(c.Query("pageSize")); err == nil && v > 0 {
		pageSize = v
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	offset := (page - 1) * pageSize

	var total int64
	h.db.Model(&models.OLTONUStatus{}).Where("oltId = ?", id).Count(&total)

	var rows []onuRow
	paginatedSQL := baseSQL + fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)
	if err := h.db.Raw(paginatedSQL, args...).Scan(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"onus":     rows,
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
// GET /api/olt/:id/onus/:onuId/assign — return searchable customer list for assignment modal
func (h *OLTHandler) GetAssignONUCandidates(c fiber.Ctx) error {
	oltID := c.Params("id")
	onuID := c.Params("onuId")
	q := strings.TrimSpace(c.Query("q"))

	var onu models.OLTONUStatus
	if err := h.db.Preload("Customer").Where("oltId = ? AND id = ?", oltID, onuID).First(&onu).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "ONU not found"})
	}

	var customers []models.PppoeUser
	db := h.db.Select("id, username, name, phone, customer_id").Limit(50)
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("name LIKE ? OR username LIKE ? OR phone LIKE ? OR customer_id LIKE ?", like, like, like, like)
	}
	db.Find(&customers)

	return c.JSON(fiber.Map{
		"success":         true,
		"customers":       customers,
		"currentCustomer": onu.Customer,
	})
}

// POST /api/olt/:id/onus/:onuId/assign — assign or unassign a customer from an ONU
func (h *OLTHandler) AssignONU(c fiber.Ctx) error {
	oltID := c.Params("id")
	onuID := c.Params("onuId")

	var body struct {
		CustomerID *string `json:"customerId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := h.db.Model(&models.OLTONUStatus{}).
		Where("oltId = ? AND id = ?", oltID, onuID).
		Update("customerId", body.CustomerID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "ONU customer assignment updated"})
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

	trafficProfiles, err := zte.GetTrafficProfiles(pool)
	if err != nil {
		log.Warn().Err(err).Str("olt", oltID).Msg("failed to fetch traffic profiles")
	}

	typeNames := make([]string, len(onuTypes))
	for i, t := range onuTypes {
		typeNames[i] = t.Name
	}

	profileNames := make([]string, len(tcontProfiles))
	for i, p := range tcontProfiles {
		profileNames[i] = p.Name
	}

	trafficProfileNames := make([]string, len(trafficProfiles))
	for i, p := range trafficProfiles {
		trafficProfileNames[i] = p.Name
	}

	return c.JSON(fiber.Map{
		"onuTypes":        typeNames,
		"tcontProfiles":   profileNames,
		"trafficProfiles": trafficProfileNames,
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

// GetChassis is implemented in olt_chassis.go

// WebSocketOLT handles WebSocket connections for real-time ONU status.
// This is called after the WebSocket upgrade — see router.go for the upgrade setup.
func (h *OLTHandler) WebSocketOLT(conn interface{}, oltID string) {
	// conn is *websocket.Conn; type assertion happens in the fiber websocket handler
	// This is handled directly in the router via gofiber/contrib/websocket
	_ = fmt.Sprintf("ws handler for olt %s registered", oltID)
}

// ─── OLT Uplink ──────────────────────────────────────────────────────────────

// uplinkPortRe validates ZTE uplink interface names: gei_1/N, gei_1/N/M, xgei_1/N, xgei_1/N/M
var uplinkPortRe = regexp.MustCompile(`(?i)^(?:gei|xgei)_\d+/\d+(?:/\d+)?$`)

// uplinkCliErrRe detects ZTE CLI error messages in command output
var uplinkCliErrRe = regexp.MustCompile(`(?i)%Error|Invalid input detected|Invalid parameter|Incomplete command|Ambiguous command|Failure:`)

// GET /api/olt/:id/uplink?port=<iface>&tab=status|vlan|config|optical
func (h *OLTHandler) GetUplink(c fiber.Ctx) error {
	id := c.Params("id")
	port := c.Query("port")
	tab := c.Query("tab")
	if tab == "" {
		tab = "status"
	}
	if port == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "port parameter required"})
	}
	if !uplinkPortRe.MatchString(port) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid port name"})
	}

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}

	// Build Telnet pool — reuse the poller's persistent pool if available.
	var pool *telnet.Pool
	var ownPool bool // true = we created the pool and must close it
	if (olt.TelnetEnabled || olt.SSHEnabled) && olt.Username != nil && olt.Password != nil {
		pool = h.poller.GetPool(id)
		if pool == nil {
			tport := olt.TelnetPort
			if tport == 0 {
				tport = 23
			}
			tcfg := telnet.DefaultConfig(olt.IPAddress, tport, *olt.Username, *olt.Password)
			tcfg.CommandTimeout = 20 * time.Second
			pool = telnet.NewPool(tcfg)
			ownPool = true
		}
	}
	if ownPool {
		defer pool.Close()
	}

	// Build SNMP config
	var snmpCfg *snmputil.Config
	if olt.SNMPEnabled {
		community := "public"
		if olt.SNMPCommunity != "" {
			community = olt.SNMPCommunity
		}
		snmpPort := 161
		if olt.SNMPPort > 0 {
			snmpPort = olt.SNMPPort
		}
		cfg := snmputil.DefaultConfig(olt.IPAddress, community, snmpPort)
		snmpCfg = &cfg
	}

	type tabResult struct {
		Raw    string            `json:"raw"`
		Parsed map[string]string `json:"parsed"`
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	var data tabResult

	switch tab {
	case "status":
		parsed := map[string]string{}
		raw := ""
		if pool != nil {
			out, err := pool.ExecuteMultiple([]string{
				"show interface port-status " + port,
				"show interface " + port,
			})
			if err == nil {
				parts := splitAtPrompt(out)
				if len(parts) >= 1 && !uplinkCliErrRe.MatchString(parts[0]) {
					raw = parts[0]
					parsed = uplinkParsePortStatus(parts[0], port)
				}
				if parsed["Admin Status"] == "" && len(parts) >= 2 && !uplinkCliErrRe.MatchString(parts[1]) {
					raw = parts[1]
					parsed = uplinkParseInterfaceStatus(parts[1])
				}
			}
		}
		// SNMP fallback for missing fields
		if (parsed["Admin Status"] == "" || parsed["Link Status"] == "") && snmpCfg != nil {
			snmpParsed := uplinkGetStatusFromSNMP(ctx, *snmpCfg, port)
			for k, v := range snmpParsed {
				if _, exists := parsed[k]; !exists {
					parsed[k] = v
				}
			}
		}
		data = tabResult{Raw: raw, Parsed: parsed}

	case "vlan":
		if pool == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Telnet not configured for this OLT"})
		}
		out, err := pool.Execute("show running-config interface " + port)
		raw, parsed := "", map[string]string{}
		if err == nil {
			raw = out
			parsed = uplinkParseRunningConfig(out)
		}
		data = tabResult{Raw: raw, Parsed: parsed}

	case "config":
		if pool == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Telnet not configured for this OLT"})
		}
		out, err := pool.Execute("show running-config interface " + port)
		raw, parsed := "", map[string]string{}
		if err == nil && !uplinkCliErrRe.MatchString(out) {
			raw = out
			parsed = uplinkParseRunningConfig(out)
		}
		data = tabResult{Raw: raw, Parsed: parsed}

	case "optical":
		if pool == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Telnet not configured for this OLT"})
		}
		out, err := pool.ExecuteMultiple([]string{
			"show interface optical-module-info " + port,
			"show ddmi interface " + port,
		})
		raw, parsed := "", map[string]string{}
		if err == nil {
			parts := splitAtPrompt(out)
			if len(parts) >= 1 && !uplinkCliErrRe.MatchString(parts[0]) {
				p := uplinkParseOpticalModuleInfo(parts[0])
				if len(p) > 0 {
					parsed = p
					raw = parts[0]
				}
			}
			if len(parsed) == 0 && len(parts) >= 2 && !uplinkCliErrRe.MatchString(parts[1]) {
				parsed = uplinkParseDdmi(parts[1])
				raw = parts[1]
			}
		}
		data = tabResult{Raw: raw, Parsed: parsed}

	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid tab"})
	}

	return c.JSON(fiber.Map{"success": true, "port": port, "tab": tab, "data": data})
}

// POST /api/olt/:id/uplink — configure uplink port (addVlan, removeVlan, enable, disable, setPvid, removePvid, setDescription)
func (h *OLTHandler) CreateUplink(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Port        string `json:"port"`
		Action      string `json:"action"`
		VlanID      string `json:"vlanId"`
		Mode        string `json:"mode"`
		Description string `json:"description"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if body.Port == "" || body.Action == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "port and action required"})
	}
	if !uplinkPortRe.MatchString(body.Port) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid port name"})
	}

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}
	if (!olt.TelnetEnabled && !olt.SSHEnabled) || olt.Username == nil || olt.Password == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Telnet not configured for this OLT"})
	}

	tport := olt.TelnetPort
	if tport == 0 {
		tport = 23
	}
	tcfg := telnet.DefaultConfig(olt.IPAddress, tport, *olt.Username, *olt.Password)
	tcfg.CommandTimeout = 8 * time.Second
	pool := telnet.NewPool(tcfg)
	defer pool.Close()

	port := body.Port
	var commandSets [][]string

	switch body.Action {
	case "addVlan":
		vid, err := strconv.Atoi(body.VlanID)
		if err != nil || vid < 1 || vid > 4094 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid VLAN ID"})
		}
		vlanCmd := "switchport vlan " + body.VlanID + " tag"
		if body.Mode == "access" {
			vlanCmd = "switchport default vlan " + body.VlanID
		}
		commandSets = [][]string{{"configure terminal", "interface " + port, vlanCmd, "exit", "end"}}

	case "removeVlan":
		vid, err := strconv.Atoi(body.VlanID)
		if err != nil || vid < 1 || vid > 4094 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid VLAN ID"})
		}
		_ = vid
		// ZTE C320: correct syntax is "no switchport vlan X" (without "tag" suffix)
		commandSets = [][]string{
			{"configure terminal", "interface " + port, "no switchport vlan " + body.VlanID, "exit", "end"},
		}

	case "enable":
		commandSets = [][]string{{"configure terminal", "interface " + port, "no shutdown", "exit", "end"}}

	case "disable":
		commandSets = [][]string{{"configure terminal", "interface " + port, "shutdown", "exit", "end"}}

	case "setDescription":
		if body.Description == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "description required"})
		}
		safe := uplinkSanitizeDesc(body.Description)
		commandSets = [][]string{{"configure terminal", "interface " + port, "description " + safe, "exit", "end"}}

	case "setPvid":
		vid, err := strconv.Atoi(body.VlanID)
		if err != nil || vid < 1 || vid > 4094 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid VLAN ID"})
		}
		_ = vid
		commandSets = [][]string{{"configure terminal", "interface " + port, "switchport default vlan " + body.VlanID, "exit", "end"}}

	case "removePvid":
		commandSets = [][]string{{"configure terminal", "interface " + port, "no switchport default vlan", "exit", "end"}}

	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Unknown action"})
	}

	lastDetail := ""
	for _, cmds := range commandSets {
		out, err := pool.ExecuteMultiple(cmds)
		if err != nil {
			lastDetail = err.Error()
			break
		}
		cliErr := false
		for _, part := range splitAtPrompt(out) {
			if uplinkCliErrRe.MatchString(part) {
				lastDetail = strings.TrimSpace(part)
				cliErr = true
				break
			}
		}
		if cliErr {
			continue
		}
		return c.JSON(fiber.Map{"success": true, "port": port, "action": body.Action})
	}

	if lastDetail != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  "OLT command failed",
			"detail": lastDetail,
		})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error":  "Uplink action failed",
		"detail": "",
	})
}

// ─── PON Port Action ─────────────────────────────────────────────────────────

// PONPortAction godoc
// POST /api/olt/:id/pon — enable, disable, or set description on a gpon-olt port.
func (h *OLTHandler) PONPortAction(c fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		Slot        int    `json:"slot"`
		Port        int    `json:"port"`
		Action      string `json:"action"`      // "enable" | "disable" | "setDescription"
		Description string `json:"description"` // required for setDescription
	}
	if err := c.Bind().JSON(&body); err != nil || body.Slot < 1 || body.Port < 1 || body.Action == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "slot, port, and action are required"})
	}

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}
	if (!olt.TelnetEnabled && !olt.SSHEnabled) || olt.Username == nil || olt.Password == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Telnet not configured for this OLT"})
	}

	iface := fmt.Sprintf("gpon-olt_1/%d/%d", body.Slot, body.Port)

	var cmds []string
	switch body.Action {
	case "enable":
		cmds = []string{"configure terminal", "interface " + iface, "no shutdown", "exit", "end"}
	case "disable":
		cmds = []string{"configure terminal", "interface " + iface, "shutdown", "exit", "end"}
	case "setDescription":
		if body.Description == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "description required"})
		}
		safe := uplinkSanitizeDesc(body.Description)
		cmds = []string{"configure terminal", "interface " + iface, "description " + safe, "exit", "end"}
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unknown action: " + body.Action})
	}

	// Prefer the poller's persistent pool; fall back to a one-shot pool.
	pool := h.poller.GetPool(id)
	var ownPool bool
	if pool == nil {
		tport := olt.TelnetPort
		if tport == 0 {
			tport = 23
		}
		tcfg := telnet.DefaultConfig(olt.IPAddress, tport, *olt.Username, *olt.Password)
		tcfg.CommandTimeout = 10 * time.Second
		pool = telnet.NewPool(tcfg)
		ownPool = true
	}
	if ownPool {
		defer pool.Close()
	}

	out, err := pool.ExecuteMultiple(cmds)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	for _, part := range splitAtPrompt(out) {
		if uplinkCliErrRe.MatchString(part) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error":  "OLT rejected command",
				"detail": strings.TrimSpace(part),
			})
		}
	}
	return c.JSON(fiber.Map{"success": true, "interface": iface, "action": body.Action})
}

// ─── Update ONU ──────────────────────────────────────────────────────────────

// UpdateONU godoc
// PATCH /api/olt/:id/onus/:onuId — update ONU name and/or description in DB and on the OLT.
func (h *OLTHandler) UpdateONU(c fiber.Ctx) error {
	oltID := c.Params("id")
	onuID := c.Params("onuId")

	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		OdpID       *string `json:"odpId"`    // set ODP assignment (UUID of network_odps)
		ClearOdp    bool    `json:"clearOdp"` // true to remove ODP link
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Name == nil && body.Description == nil && body.OdpID == nil && !body.ClearOdp {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name, description, or odpId required"})
	}

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}

	var onuStatus models.OLTONUStatus
	if err := h.db.Where("oltId = ? AND id = ?", oltID, onuID).First(&onuStatus).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ONU not found"})
	}

	// Persist description change in the database.
	if body.Description != nil {
		if err := h.db.Model(&onuStatus).Update("description", *body.Description).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	// Persist ODP assignment.
	if body.ClearOdp {
		if err := h.db.Model(&onuStatus).Update("odpId", nil).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	} else if body.OdpID != nil {
		// Verify ODP exists before linking
		var odpCount int64
		h.db.Model(&models.NetworkODP{}).Where("id = ?", *body.OdpID).Count(&odpCount)
		if odpCount == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ODP not found"})
		}
		if err := h.db.Model(&onuStatus).Update("odpId", *body.OdpID).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		// If only setting ODP (no name/description changes), return early
		if body.Name == nil && body.Description == nil {
			return c.JSON(fiber.Map{"success": true, "message": "ODP assigned"})
		}
	}

	// Push change to OLT via Telnet if available.
	if (olt.TelnetEnabled || olt.SSHEnabled) && olt.Username != nil && olt.Password != nil {
		iface := fmt.Sprintf("gpon-onu_%d/%d/%d:%d", onuStatus.Frame, onuStatus.Slot, onuStatus.Port, onuStatus.OnuID)

		pool := h.poller.GetPool(oltID)
		var ownPool bool
		if pool == nil {
			tport := olt.TelnetPort
			if tport == 0 {
				tport = 23
			}
			tcfg := telnet.DefaultConfig(olt.IPAddress, tport, *olt.Username, *olt.Password)
			tcfg.CommandTimeout = 10 * time.Second
			pool = telnet.NewPool(tcfg)
			ownPool = true
		}
		if ownPool {
			defer pool.Close()
		}

		cmds := []string{"configure terminal", "interface " + iface}
		if body.Name != nil {
			cmds = append(cmds, "name "+uplinkSanitizeDesc(*body.Name))
		}
		if body.Description != nil {
			cmds = append(cmds, "description "+uplinkSanitizeDesc(*body.Description))
		}
		cmds = append(cmds, "exit", "end")

		out, err := pool.ExecuteMultiple(cmds)
		if err != nil {
			// Telnet failure is non-fatal; DB was already updated.
			log.Warn().Err(err).Str("iface", iface).Msg("UpdateONU: Telnet command failed")
		} else {
			for _, part := range splitAtPrompt(out) {
				if uplinkCliErrRe.MatchString(part) {
					log.Warn().Str("detail", strings.TrimSpace(part)).Str("iface", iface).Msg("UpdateONU: OLT CLI error (DB updated)")
					break
				}
			}
		}
	}

	return c.JSON(fiber.Map{"success": true, "message": "ONU updated"})
}

// ── Uplink parser helpers ─────────────────────────────────────────────────────

// uplinkParsePortStatus parses "show interface port-status <port>" tabular output.
// ZTE C320 column layout (0-indexed):
//
//	[0]=Port [1]=PhType [2]=Speed [3]=Duplex [4]=ActualSpeed(Mbps)
//	[5]=FEC  [6]=CRC16  [7]=Pause [8]=FlowCtrl [9]=AdminStatus [10]=LinkStatus
func uplinkParsePortStatus(output, portName string) map[string]string {
	result := map[string]string{}
	escaped := regexp.QuoteMeta(portName)
	lineRe := regexp.MustCompile(`(?i)^\s*` + escaped + `\s+`)
	for _, rawLine := range strings.Split(output, "\n") {
		if !lineRe.MatchString(rawLine) {
			continue
		}
		parts := strings.Fields(strings.TrimSpace(rawLine))
		if len(parts) < 8 {
			break
		}
		// ZTE C320 port-status columns (0-based):
		// [0]=Port [1]=HybridStatus [2]=NativeVLAN [3]=Negotiation [4]=Speed(Mbps)
		// [5]=Duplex [6]=FlowCtrl [7]=AdminStatus [8]=LinkStatus
		result["Physical Type"] = parts[1]
		if len(parts) > 5 {
			result["Duplex"] = parts[5]
		}
		if len(parts) > 4 && parts[4] != "N/A" && parts[4] != "0" {
			result["Speed"] = parts[4] + " Mbps"
		}
		if len(parts) > 6 {
			result["Flow Control"] = parts[6]
		}
		adminRaw, linkRaw := "", ""
		if len(parts) > 7 {
			adminRaw = parts[7]
		}
		if len(parts) > 8 {
			linkRaw = parts[8]
		}
		if strings.EqualFold(adminRaw, "activate") {
			result["Admin Status"] = "Up"
		} else if strings.EqualFold(adminRaw, "deactivate") {
			result["Admin Status"] = "Down"
		} else if adminRaw != "" {
			result["Admin Status"] = adminRaw
		}
		if strings.EqualFold(linkRaw, "up") {
			result["Link Status"] = "Up"
		} else if strings.EqualFold(linkRaw, "down") {
			result["Link Status"] = "Down"
		} else if linkRaw != "" {
			result["Link Status"] = linkRaw
		}
		break
	}
	return result
}

// uplinkParseInterfaceStatus parses "show interface <port>" key:value output.
func uplinkParseInterfaceStatus(output string) map[string]string {
	result := map[string]string{}
	// ZTE C320 uplink (xgei) uses "up/down"; GPON ports use "activate/deactivate"
	stateRe := regexp.MustCompile(`(?i)^(\S+)\s+is\s+(activate|deactivate|up|down)\s*,\s*line protocol is\s+(up|down)`)
	descRe := regexp.MustCompile(`(?i)^Description is\s+(.+?)\.?$`)
	kvRe := regexp.MustCompile(`^\s*([^:]+?)\s*:\s*(.+)$`)
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if m := stateRe.FindStringSubmatch(line); m != nil {
			adminRaw := strings.ToLower(m[2])
			if adminRaw == "activate" || adminRaw == "up" {
				result["Admin Status"] = "Up"
			} else {
				result["Admin Status"] = "Down"
			}
			if strings.EqualFold(m[3], "up") {
				result["Link Status"] = "Up"
			} else {
				result["Link Status"] = "Down"
			}
			continue
		}
		if m := descRe.FindStringSubmatch(line); m != nil {
			result["Description"] = strings.TrimSpace(m[1])
			continue
		}
		if m := kvRe.FindStringSubmatch(line); m != nil {
			result[strings.TrimSpace(m[1])] = strings.TrimSpace(m[2])
		}
	}
	return result
}

// uplinkParseRunningConfig parses "show running-config interface <port>" output.
func uplinkParseRunningConfig(output string) map[string]string {
	result := map[string]string{}
	taggedVlans := []string{}
	skipRe := regexp.MustCompile(`(?i)^Building configuration|^interface\s+|^!$|^end$`)
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || skipRe.MatchString(line) {
			continue
		}
		var m []string
		if m = regexp.MustCompile(`(?i)^description\s+(.+)$`).FindStringSubmatch(line); m != nil {
			result["Description"] = m[1]
		} else if m = regexp.MustCompile(`(?i)^switchport mode\s+(\S+)$`).FindStringSubmatch(line); m != nil {
			result["Mode"] = m[1]
		} else if m = regexp.MustCompile(`(?i)^switchport tls\s+(\S+)$`).FindStringSubmatch(line); m != nil {
			result["TLS"] = m[1]
		} else if m = regexp.MustCompile(`(?i)^switchport default vlan\s+(\d+)$`).FindStringSubmatch(line); m != nil {
			result["Pvid"] = m[1]
		} else if m = regexp.MustCompile(`(?i)^switchport vlan\s+(\d+)\s+untag$`).FindStringSubmatch(line); m != nil {
			result["Pvid"] = m[1]
		} else if m = regexp.MustCompile(`(?i)^switchport vlan\s+(.+?)\s+tag$`).FindStringSubmatch(line); m != nil {
			for _, v := range strings.Split(m[1], ",") {
				if v = strings.TrimSpace(v); v != "" {
					taggedVlans = append(taggedVlans, v)
				}
			}
		} else if m = regexp.MustCompile(`(?i)^vlan\s+(.+?)\s+tag$`).FindStringSubmatch(line); m != nil {
			for _, v := range strings.FieldsFunc(m[1], func(r rune) bool { return r == ' ' || r == ',' }) {
				if v = strings.TrimSpace(v); v != "" {
					taggedVlans = append(taggedVlans, v)
				}
			}
		} else if m = regexp.MustCompile(`(?i)^pvid\s+(\d+)$`).FindStringSubmatch(line); m != nil {
			result["Pvid"] = m[1]
		} else if m = regexp.MustCompile(`(?i)^mode\s+(\S+)$`).FindStringSubmatch(line); m != nil {
			if result["Mode"] == "" {
				result["Mode"] = m[1]
			}
		} else if strings.EqualFold(line, "no shutdown") {
			result["Admin Status"] = "Up"
		} else if strings.EqualFold(line, "shutdown") {
			result["Admin Status"] = "Down"
		}
	}
	if len(taggedVlans) > 0 {
		seen := map[string]bool{}
		unique := []string{}
		for _, v := range taggedVlans {
			if !seen[v] {
				seen[v] = true
				unique = append(unique, v)
			}
		}
		result["Tagged Vlan"] = strings.Join(unique, " ")
	}
	return result
}

// uplinkParseDdmi parses "show ddmi interface <port>" output (key: value lines).
func uplinkParseDdmi(output string) map[string]string {
	result := map[string]string{}
	re := regexp.MustCompile(`^\s*([^:()]+(?:\([^)]*\))?[^:]*?)\s*:\s*(.+)$`)
	unitRe := regexp.MustCompile(`\s*\([^)]*\)\s*$`)
	for _, rawLine := range strings.Split(output, "\n") {
		m := re.FindStringSubmatch(rawLine)
		if m == nil {
			continue
		}
		key := strings.TrimSpace(unitRe.ReplaceAllString(strings.TrimSpace(m[1]), ""))
		val := strings.TrimSpace(m[2])
		if key != "" && val != "" {
			result[key] = val
		}
	}
	return result
}

// uplinkParseOpticalModuleInfo parses "show interface optical-module-info <port>" output.
func uplinkParseOpticalModuleInfo(output string) map[string]string {
	result := map[string]string{}
	keyMap := map[string]string{
		"vendor-name":    "Vendor",
		"vendor-pn":      "Part Number",
		"vendor-sn":      "Serial Number",
		"wavelength":     "Wavelength",
		"fiber-type":     "Fiber Type",
		"connector":      "Connector Type",
		"rxpower":        "RX Power",
		"txpower":        "TX Power",
		"txbias-current": "TX Bias Current",
		"temperature":    "Temperature",
		"supply-vol":     "Supply Voltage",
	}
	re := regexp.MustCompile(`([A-Za-z][A-Za-z0-9\- ]+?)\s*:\s*(.+?)(?:\s{2,}|$)`)
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || !strings.Contains(line, ":") {
			continue
		}
		for _, m := range re.FindAllStringSubmatch(line, -1) {
			rawKey := strings.TrimSpace(m[1])
			rawVal := strings.TrimSpace(m[2])
			normalizedKey := strings.ToLower(rawKey)
			mappedKey := rawKey
			if k, ok := keyMap[normalizedKey]; ok {
				mappedKey = k
			}
			if rawVal != "" {
				result[mappedKey] = rawVal
			}
		}
	}
	return result
}

// uplinkGetStatusFromSNMP fetches interface status from SNMP IF-MIB as a fallback.
func uplinkGetStatusFromSNMP(ctx context.Context, cfg snmputil.Config, ifaceName string) map[string]string {
	result := map[string]string{}
	data := fetchIfMib(ctx, cfg)
	if data == nil {
		return result
	}
	normalize := func(s string) string {
		return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, "-", "_"), " ", "_"))
	}
	idx := ""
	for suffix, name := range data.descr {
		if normalize(name) == normalize(ifaceName) {
			idx = suffix
			break
		}
	}
	if idx == "" {
		return result
	}
	adminVal, _ := strconv.Atoi(data.admin[idx])
	operVal, _ := strconv.Atoi(data.oper[idx])
	if adminVal == 1 {
		result["Admin Status"] = "Up"
	} else {
		result["Admin Status"] = "Down"
	}
	if operVal == 1 {
		result["Link Status"] = "Up"
	} else {
		result["Link Status"] = "Down"
	}
	if s := data.speed[idx]; s != "" && s != "0" {
		result["Speed"] = s + "M"
	}
	if a := data.alias[idx]; a != "" {
		result["Description"] = a
	}
	return result
}

// uplinkSanitizeDesc strips non-printable-ASCII and truncates to 64 chars.
func uplinkSanitizeDesc(desc string) string {
	safe := make([]rune, 0, len(desc))
	for _, r := range desc {
		if r >= 0x20 && r <= 0x7E {
			safe = append(safe, r)
		}
	}
	s := string(safe)
	if len(s) > 64 {
		s = s[:64]
	}
	return s
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
