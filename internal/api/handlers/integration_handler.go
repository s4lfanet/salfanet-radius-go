package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// IntegrationHandler handles Phase 3 integration endpoints.
type IntegrationHandler struct {
	db *gorm.DB
}

func NewIntegrationHandler(db *gorm.DB) *IntegrationHandler {
	return &IntegrationHandler{db: db}
}

// ─── ONU ↔ Customer Auto-Link ────────────────────────────────────────────────

// GET /api/integration/onu/unlinked — list ONUs not yet linked to a customer
func (h *IntegrationHandler) ListUnlinkedONUs(c fiber.Ctx) error {
	page, pageSize := pageParams(c)
	var onus []models.OLTONUStatus
	query := h.db.Model(&models.OLTONUStatus{}).Where("customerId IS NULL").Preload("OLT")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if serial := c.Query("serial"); serial != "" {
		query = query.Where("serialNumber LIKE ?", "%"+serial+"%")
	}
	if oltID := c.Query("oltId"); oltID != "" {
		query = query.Where("oltId = ?", oltID)
	}

	var total int64
	query.Count(&total)
	query.Order("lastSeenAt DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&onus)

	return c.JSON(fiber.Map{"data": onus, "total": total, "page": page, "pageSize": pageSize})
}

// POST /api/integration/onu/:onuId/link-customer — link ONU to customer by serial or manual
func (h *IntegrationHandler) LinkONUToCustomer(c fiber.Ctx) error {
	onuID := c.Params("onuId")
	var body struct {
		CustomerID string `json:"customerId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	var onu models.OLTONUStatus
	if err := h.db.First(&onu, "id = ?", onuID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "ONU not found"})
	}

	var user models.PppoeUser
	if err := h.db.First(&user, "id = ?", body.CustomerID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Customer not found"})
	}

	if err := h.db.Model(&onu).Update("customerId", body.CustomerID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("ONU %s linked to customer %s", *onu.SerialNumber, user.Username),
	})
}

// POST /api/integration/onu/auto-link — auto-link ONUs by matching serial number to pppoe_users
func (h *IntegrationHandler) AutoLinkONUs(c fiber.Ctx) error {
	var onus []models.OLTONUStatus
	h.db.Where("customerId IS NULL AND serialNumber IS NOT NULL").Find(&onus)

	linked := 0
	for _, onu := range onus {
		if onu.SerialNumber == nil {
			continue
		}
		// Try to find a pppoe_user with matching serial number in description or notes
		var user models.PppoeUser
		if err := h.db.Where("description LIKE ? OR notes LIKE ?",
			"%"+*onu.SerialNumber+"%", "%"+*onu.SerialNumber+"%").
			First(&user).Error; err != nil {
			continue
		}
		h.db.Model(&onu).Update("customerId", user.ID)
		linked++
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Auto-linked %d ONUs to customers", linked),
		"linked":  linked,
		"total":   len(onus),
	})
}

// ─── RX Power Monitoring ─────────────────────────────────────────────────────

// GET /api/integration/rx-power/summary — ONU RX power summary across all OLTs
func (h *IntegrationHandler) RxPowerSummary(c fiber.Ctx) error {
	type rxSummary struct {
		TotalONU   int64   `json:"totalOnu"`
		OnlineONU  int64   `json:"onlineOnu"`
		GoodSignal int64   `json:"goodSignal"`
		WeakSignal int64   `json:"weakSignal"`
		BadSignal  int64   `json:"badSignal"`
		NoSignal   int64   `json:"noSignal"`
		AvgRxPower float64 `json:"avgRxPower"`
	}

	var summary rxSummary
	h.db.Model(&models.OLTONUStatus{}).Where("status = ?", "online").Count(&summary.OnlineONU)
	h.db.Model(&models.OLTONUStatus{}).Count(&summary.TotalONU)

	// Good: >= -20 dBm, Weak: -20 to -27 dBm, Bad: < -27 dBm, No: NULL
	h.db.Model(&models.OLTONUStatus{}).
		Where("status = 'online' AND rxPower IS NOT NULL AND rxPower >= -20").
		Count(&summary.GoodSignal)
	h.db.Model(&models.OLTONUStatus{}).
		Where("status = 'online' AND rxPower IS NOT NULL AND rxPower < -20 AND rxPower >= -27").
		Count(&summary.WeakSignal)
	h.db.Model(&models.OLTONUStatus{}).
		Where("status = 'online' AND rxPower IS NOT NULL AND rxPower < -27").
		Count(&summary.BadSignal)
	h.db.Model(&models.OLTONUStatus{}).
		Where("status = 'online' AND (rxPower IS NULL OR rxPower = 0)").
		Count(&summary.NoSignal)

	var avgResult struct{ Avg float64 }
	h.db.Model(&models.OLTONUStatus{}).
		Where("status = 'online' AND rxPower IS NOT NULL AND rxPower != 0").
		Select("COALESCE(AVG(rxPower), 0) as avg").
		Scan(&avgResult)
	summary.AvgRxPower = avgResult.Avg

	// Per-OLT breakdown
	type oltBreakdown struct {
		OltID       string  `json:"oltId"`
		OltName     string  `json:"oltName"`
		TotalONU    int64   `json:"totalOnu"`
		OnlineONU   int64   `json:"onlineOnu"`
		WeakSignal  int64   `json:"weakSignal"`
		BadSignal   int64   `json:"badSignal"`
		AvgRxPower  float64 `json:"avgRxPower"`
	}
	var breakdown []oltBreakdown
	h.db.Raw(`
		SELECT o.id as oltId, o.name as oltName,
			COUNT(s.id) as totalOnu,
			SUM(CASE WHEN s.status = 'online' THEN 1 ELSE 0 END) as onlineOnu,
			SUM(CASE WHEN s.status = 'online' AND s.rxPower IS NOT NULL AND s.rxPower < -20 AND s.rxPower >= -27 THEN 1 ELSE 0 END) as weakSignal,
			SUM(CASE WHEN s.status = 'online' AND s.rxPower IS NOT NULL AND s.rxPower < -27 THEN 1 ELSE 0 END) as badSignal,
			COALESCE(AVG(CASE WHEN s.status = 'online' AND s.rxPower IS NOT NULL AND s.rxPower != 0 THEN s.rxPower END), 0) as avgRxPower
		FROM network_olts o
		LEFT JOIN olt_onu_status s ON s.oltId = o.id
		GROUP BY o.id, o.name
		ORDER BY o.name
	`).Scan(&breakdown)

	return c.JSON(fiber.Map{
		"summary":   summary,
		"breakdown": breakdown,
	})
}

// GET /api/integration/rx-power/onu/:onuId — RX power history for a specific ONU
func (h *IntegrationHandler) RxPowerHistory(c fiber.Ctx) error {
	onuID := c.Params("onuId")
	var onu models.OLTONUStatus
	if err := h.db.First(&onu, "id = ?", onuID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "ONU not found"})
	}

	// Get monitoring logs that contain rxPower data
	type rxLog struct {
		ID        string    `json:"id"`
		RxPower   *float64  `json:"rxPower"`
		TxPower   *float64  `json:"txPower"`
		Status    string    `json:"status"`
		RecordedAt time.Time `json:"recordedAt"`
	}

	// Current values
	current := fiber.Map{
		"rxPower":     onu.RxPower,
		"txPower":     onu.TxPower,
		"status":      onu.Status,
		"serialNumber": onu.SerialNumber,
		"lastSeenAt":  onu.LastSeenAt,
	}

	// Get recent alerts for this ONU
	var alerts []models.OLTAlert
	h.db.Where("onuId = ?", onuID).Order("createdAt DESC").Limit(50).Find(&alerts)

	return c.JSON(fiber.Map{
		"current": current,
		"alerts":  alerts,
		"thresholds": fiber.Map{
			"good": -20.0,
			"weak": -27.0,
			"bad":  -30.0,
		},
	})
}

// GET /api/integration/rx-power/degraded — list ONUs with degraded signal
func (h *IntegrationHandler) ListDegradedONUs(c fiber.Ctx) error {
	page, pageSize := pageParams(c)
	var onus []models.OLTONUStatus
	query := h.db.Model(&models.OLTONUStatus{}).
		Where("status = 'online' AND rxPower IS NOT NULL AND rxPower < -20").
		Preload("OLT").Preload("Customer")

	var total int64
	query.Count(&total)
	query.Order("rxPower ASC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&onus)

	return c.JSON(fiber.Map{"data": onus, "total": total, "page": page, "pageSize": pageSize})
}

// ─── Real-Time Dashboard ─────────────────────────────────────────────────────

// GET /api/integration/dashboard — aggregated stats for real-time dashboard
func (h *IntegrationHandler) Dashboard(c fiber.Ctx) error {
	// Customer stats
	var totalCustomers, activeCustomers, isolatedCustomers, stoppedCustomers int64
	h.db.Model(&models.PppoeUser{}).Count(&totalCustomers)
	h.db.Model(&models.PppoeUser{}).Where("status = 'active'").Count(&activeCustomers)
	h.db.Model(&models.PppoeUser{}).Where("status = 'isolated'").Count(&isolatedCustomers)
	h.db.Model(&models.PppoeUser{}).Where("status = 'stopped'").Count(&stoppedCustomers)

	// Invoice stats (current month)
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var totalInvoices, pendingInvoices, paidInvoices, overdueInvoices int64
	var totalRevenue int64

	h.db.Model(&models.Invoice{}).Where("createdAt >= ?", monthStart).Count(&totalInvoices)
	h.db.Model(&models.Invoice{}).Where("createdAt >= ? AND status = 'PENDING'", monthStart).Count(&pendingInvoices)
	h.db.Model(&models.Invoice{}).Where("createdAt >= ? AND status = 'PAID'", monthStart).Count(&paidInvoices)
	h.db.Model(&models.Invoice{}).Where("createdAt >= ? AND status = 'OVERDUE'", monthStart).Count(&overdueInvoices)
	h.db.Model(&models.Invoice{}).Where("createdAt >= ? AND status = 'PAID'", monthStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)

	// OLT/ONU stats
	var totalOLTs, onlineOLTs, totalONUs, onlineONUs int64
	h.db.Model(&models.NetworkOLT{}).Count(&totalOLTs)
	h.db.Model(&models.NetworkOLT{}).Where("isOnline = true").Count(&onlineOLTs)
	h.db.Model(&models.OLTONUStatus{}).Count(&totalONUs)
	h.db.Model(&models.OLTONUStatus{}).Where("status = 'online'").Count(&onlineONUs)

	// Unresolved alerts
	var unresolvedAlerts int64
	h.db.Model(&models.OLTAlert{}).Where("isResolved = false").Count(&unresolvedAlerts)

	// Active sessions
	var activeSessions int64
	h.db.Table("radacct").Where("acctstoptime IS NULL").Count(&activeSessions)

	// Pending registrations
	var pendingRegs int64
	h.db.Table("pppoe_registrations").Where("status = 'PENDING'").Count(&pendingRegs)

	// Pending tickets
	var openTickets int64
	h.db.Model(&models.Ticket{}).Where("status NOT IN ('RESOLVED', 'CLOSED')").Count(&openTickets)

	return c.JSON(fiber.Map{
		"customers": fiber.Map{
			"total":     totalCustomers,
			"active":    activeCustomers,
			"isolated":  isolatedCustomers,
			"stopped":   stoppedCustomers,
		},
		"invoices": fiber.Map{
			"total":     totalInvoices,
			"pending":   pendingInvoices,
			"paid":      paidInvoices,
			"overdue":   overdueInvoices,
			"revenue":   totalRevenue,
			"month":     now.Format("2006-01"),
		},
		"network": fiber.Map{
			"totalOLTs":     totalOLTs,
			"onlineOLTs":    onlineOLTs,
			"totalONUs":     totalONUs,
			"onlineONUs":    onlineONUs,
			"activeSessions": activeSessions,
		},
		"alerts": fiber.Map{
			"unresolved":    unresolvedAlerts,
			"pendingRegs":   pendingRegs,
			"openTickets":   openTickets,
		},
		"timestamp": now,
	})
}

// GET /api/integration/dashboard/trends — 7-day trend data for charts
func (h *IntegrationHandler) DashboardTrends(c fiber.Ctx) error {
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)

	type dayStat struct {
		Date         string `json:"date"`
		NewCustomers int64  `json:"newCustomers"`
		PaidInvoices int64  `json:"paidInvoices"`
		Revenue      int64  `json:"revenue"`
		Isolations   int64  `json:"isolations"`
	}

	var stats []dayStat
	h.db.Raw(`
		SELECT d.date,
			(SELECT COUNT(*) FROM pppoe_users WHERE DATE(createdAt) = d.date) as newCustomers,
			(SELECT COUNT(*) FROM invoices WHERE DATE(paidAt) = d.date AND status = 'PAID') as paidInvoices,
			(SELECT COALESCE(SUM(amount), 0) FROM invoices WHERE DATE(paidAt) = d.date AND status = 'PAID') as revenue,
			(SELECT COUNT(*) FROM pppoe_users WHERE DATE(updatedAt) = d.date AND status = 'isolated') as isolations
		FROM (
			SELECT DATE(?) + INTERVAL n DAY as date
			FROM (
				SELECT 0 as n UNION SELECT 1 UNION SELECT 2 UNION SELECT 3
				UNION SELECT 4 UNION SELECT 5 UNION SELECT 6
			) numbers
			WHERE DATE(?) + INTERVAL n DAY <= DATE(?)
		) d
		ORDER BY d.date
	`, weekAgo, weekAgo, now).Scan(&stats)

	return c.JSON(fiber.Map{"data": stats})
}

// ─── ONU Auto-Provisioning ───────────────────────────────────────────────────

// POST /api/integration/onu/provision — provision a new ONU (register + link customer + set VLAN)
func (h *IntegrationHandler) ProvisionONU(c fiber.Ctx) error {
	var body struct {
		OltID       string  `json:"oltId"`
		SerialNumber string `json:"serialNumber"`
		CustomerID  *string `json:"customerId"`
		Description *string `json:"description"`
		VlanID      *int    `json:"vlanId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	// Check if ONU already exists
	var existing models.OLTONUStatus
	if err := h.db.Where("oltId = ? AND serialNumber = ?", body.OltID, body.SerialNumber).
		First(&existing).Error; err == nil {
		// ONU exists — just link customer if provided
		if body.CustomerID != nil {
			h.db.Model(&existing).Update("customerId", *body.CustomerID)
		}
		return c.JSON(fiber.Map{
			"success": true,
			"message": "ONU already registered, customer linked",
			"onu":     existing,
		})
	}

	// Create new ONU record (will be updated by poller with real data)
	onu := models.OLTONUStatus{
		ID:           uuid.New().String(),
		OltID:        body.OltID,
		SerialNumber: &body.SerialNumber,
		Description:  body.Description,
		Status:       models.OnuUnregistered,
		CustomerID:   body.CustomerID,
	}
	if body.VlanID != nil {
		onu.VlanID = body.VlanID
	}

	if err := h.db.Create(&onu).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "ONU provisioned — will be registered on next poll cycle",
		"onu":     onu,
	})
}

// GET /api/integration/onu/customer/:customerId — find ONU(s) linked to a customer
func (h *IntegrationHandler) GetCustomerONU(c fiber.Ctx) error {
	customerID := c.Params("customerId")
	var onus []models.OLTONUStatus
	h.db.Preload("OLT").Where("customerId = ?", customerID).Find(&onus)

	if len(onus) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "No ONU found for this customer"})
	}

	return c.JSON(fiber.Map{"data": onus})
}
