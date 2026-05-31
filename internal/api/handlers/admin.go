package handlers

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// AdminHandler handles dashboard/stats endpoints.
type AdminHandler struct {
	db *gorm.DB
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

var monthNamesID = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func formatIDR(amount int64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	s := strconv.FormatInt(amount, 10)
	// insert thousands separators
	result := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += "."
		}
		result += string(ch)
	}
	if negative {
		return "-Rp\u00a0" + result
	}
	return "Rp\u00a0" + result
}

// Stats godoc
// GET /api/dashboard/stats
func (h *AdminHandler) Stats(c fiber.Ctx) error {
	now := time.Now()

	// Parse optional ?month=YYYY-MM
	monthParam := c.Query("month")
	var selectedYear, selectedMonth int
	re := regexp.MustCompile(`^\d{4}-\d{2}$`)
	if re.MatchString(monthParam) {
		fmt.Sscanf(monthParam, "%d-%d", &selectedYear, &selectedMonth)
	} else {
		selectedYear = now.Year()
		selectedMonth = int(now.Month())
	}
	monthKey := fmt.Sprintf("%04d-%02d", selectedYear, selectedMonth)
	periodLabel := fmt.Sprintf("%s %d", monthNamesID[selectedMonth], selectedYear)
	isCurrentMonth := selectedYear == now.Year() && selectedMonth == int(now.Month())

	startOfMonth := time.Date(selectedYear, time.Month(selectedMonth), 1, 0, 0, 0, 0, time.UTC)
	startOfNextMonth := time.Date(selectedYear, time.Month(selectedMonth)+1, 1, 0, 0, 0, 0, time.UTC)
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	sevenDaysFromNow := now.Add(7 * 24 * time.Hour)

	// ── PPPoE user counts ──────────────────────────────────────────────────────
	var totalPppoeUsers, activePppoeUsers, isolatedCount, suspendedCount int64
	h.db.Model(&models.PppoeUser{}).Count(&totalPppoeUsers)
	h.db.Model(&models.PppoeUser{}).Where("status IN ?", []string{"active", "ACTIVE"}).Count(&activePppoeUsers)
	h.db.Model(&models.PppoeUser{}).Where("status IN ?", []string{"isolated", "ISOLATED", "blocked", "BLOCKED"}).Count(&isolatedCount)
	h.db.Model(&models.PppoeUser{}).Where("status IN ?", []string{"suspended", "SUSPENDED"}).Count(&suspendedCount)

	// ── New registrations ───────────────────────────────────────────────────────
	var newRegistrations int64
	h.db.Model(&models.RegistrationRequest{}).Where("status IN ?", []string{"PENDING", "REVIEWING"}).Count(&newRegistrations)

	// ── Active sessions (PPPoE vs Hotspot from radacct) ────────────────────────
	var activeSessionsPPPoE, activeSessionsHotspot int64
	h.db.Model(&models.Radacct{}).Where("acctstoptime IS NULL").Count(&activeSessionsPPPoE)
	// simple: all active = PPPoE sessions for now (exact split would need join)

	// ── Unused vouchers ────────────────────────────────────────────────────────
	var unusedVouchers int64
	h.db.Model(&models.HotspotVoucher{}).Where("status = ?", "WAITING").Count(&unusedVouchers)

	// ── Upcoming invoices (H-7) ────────────────────────────────────────────────
	type upcomingRow struct {
		InvoiceNumber    string    `json:"invoiceNumber"`
		CustomerName     *string   `json:"customerName"`
		CustomerUsername *string   `json:"customerUsername"`
		Amount           int       `json:"amount"`
		DueDate          time.Time `json:"dueDate"`
		Status           string    `json:"status"`
	}
	var rawUpcoming []upcomingRow
	h.db.Model(&models.Invoice{}).
		Where("status IN ? AND dueDate <= ?", []string{"PENDING", "OVERDUE"}, sevenDaysFromNow).
		Order("dueDate asc").Limit(20).
		Find(&rawUpcoming)

	type upcomingOut struct {
		InvoiceNumber    string `json:"invoiceNumber"`
		CustomerName     string `json:"customerName"`
		CustomerUsername string `json:"customerUsername"`
		Amount           int    `json:"amount"`
		DueDate          string `json:"dueDate"`
		Status           string `json:"status"`
		DaysUntilDue     int    `json:"daysUntilDue"`
	}
	upcomingInvoices := make([]upcomingOut, 0, len(rawUpcoming))
	for _, inv := range rawUpcoming {
		name := "-"
		if inv.CustomerName != nil {
			name = *inv.CustomerName
		}
		uname := "-"
		if inv.CustomerUsername != nil {
			uname = *inv.CustomerUsername
		}
		days := int(math.Ceil(inv.DueDate.Sub(now).Hours() / 24))
		upcomingInvoices = append(upcomingInvoices, upcomingOut{
			InvoiceNumber:    inv.InvoiceNumber,
			CustomerName:     name,
			CustomerUsername: uname,
			Amount:           inv.Amount,
			DueDate:          inv.DueDate.Format(time.RFC3339),
			Status:           inv.Status,
			DaysUntilDue:     days,
		})
	}

	// ── Invoice revenue ────────────────────────────────────────────────────────
	var invoiceRevenueToday, invoiceRevenue, totalAllTimeRevenue int64
	var invoiceCountToday, invoiceCountMonth, unpaidInvoicesCount int64
	h.db.Model(&models.Invoice{}).Where("status = ? AND paidAt >= ?", "PAID", startOfToday).
		Select("COALESCE(SUM(amount),0)").Scan(&invoiceRevenueToday)
	h.db.Model(&models.Invoice{}).Where("status = ? AND paidAt >= ?", "PAID", startOfToday).Count(&invoiceCountToday)
	h.db.Model(&models.Invoice{}).Where("status = ? AND paidAt >= ? AND paidAt < ?", "PAID", startOfMonth, startOfNextMonth).
		Select("COALESCE(SUM(amount),0)").Scan(&invoiceRevenue)
	h.db.Model(&models.Invoice{}).Where("status = ? AND paidAt >= ? AND paidAt < ?", "PAID", startOfMonth, startOfNextMonth).Count(&invoiceCountMonth)
	h.db.Model(&models.Invoice{}).Where("status IN ?", []string{"PENDING", "OVERDUE"}).Count(&unpaidInvoicesCount)
	h.db.Model(&models.Invoice{}).Where("status = ?", "PAID").Select("COALESCE(SUM(amount),0)").Scan(&totalAllTimeRevenue)

	// ── Voucher revenue (estimate from sold vouchers) ──────────────────────────
	var voucherRevenue, voucherRevenueToday int64
	h.db.Raw(`
		SELECT COALESCE(SUM(hp.price),0)
		FROM hotspot_vouchers hv
		JOIN hotspot_profiles hp ON hp.id = hv.profileId
		WHERE hv.status IN ('ACTIVE','EXPIRED','SOLD')
		  AND hv.firstLoginAt >= ? AND hv.firstLoginAt < ?
	`, startOfMonth, startOfNextMonth).Scan(&voucherRevenue)
	h.db.Raw(`
		SELECT COALESCE(SUM(hp.price),0)
		FROM hotspot_vouchers hv
		JOIN hotspot_profiles hp ON hp.id = hv.profileId
		WHERE hv.status IN ('ACTIVE','EXPIRED','SOLD')
		  AND hv.firstLoginAt >= ?
	`, startOfToday).Scan(&voucherRevenueToday)

	// ── Agent sales ────────────────────────────────────────────────────────────
	type agentSaleRow struct {
		AgentID   string `json:"agentId"`
		AgentName string `json:"agentName"`
		Sold      int64  `json:"sold"`
		Revenue   int64  `json:"revenue"`
	}
	var agentSales []agentSaleRow
	h.db.Raw(`
		SELECT hv.agentId, ag.name AS agentName,
		       COUNT(*) AS sold, COALESCE(SUM(hp.price),0) AS revenue
		FROM hotspot_vouchers hv
		JOIN agents ag ON ag.id = hv.agentId
		JOIN hotspot_profiles hp ON hp.id = hv.profileId
		WHERE hv.agentId IS NOT NULL
		  AND hv.firstLoginAt >= ? AND hv.firstLoginAt < ?
		GROUP BY hv.agentId, ag.name
		ORDER BY sold DESC
		LIMIT 5
	`, startOfMonth, startOfNextMonth).Scan(&agentSales)
	if agentSales == nil {
		agentSales = []agentSaleRow{}
	}
	var agentSalesTotalCount, agentSalesTotalRevenue int64
	for _, s := range agentSales {
		agentSalesTotalCount += s.Sold
		agentSalesTotalRevenue += s.Revenue
	}

	// ── RADIUS auth log ────────────────────────────────────────────────────────
	type authRow struct {
		Username string    `json:"username"`
		Reply    string    `json:"reply"`
		Authdate time.Time `json:"authdate"`
	}
	var radiusAuthLog []authRow
	h.db.Raw(`SELECT username, reply, authdate FROM radpostauth ORDER BY authdate DESC LIMIT 15`).Scan(&radiusAuthLog)
	if radiusAuthLog == nil {
		radiusAuthLog = []authRow{}
	}
	var acceptToday, rejectToday int64
	h.db.Raw(`SELECT COUNT(*) FROM radpostauth WHERE reply='Access-Accept' AND authdate >= ?`, startOfToday).Scan(&acceptToday)
	h.db.Raw(`SELECT COUNT(*) FROM radpostauth WHERE reply='Access-Reject' AND authdate >= ?`, startOfToday).Scan(&rejectToday)

	// ── System status ──────────────────────────────────────────────────────────
	radiusOnline, _ := checkFreeradiusRunning()

	// ── Recent activities ──────────────────────────────────────────────────────
	var activities []models.ActivityLog
	h.db.Order("createdAt DESC").Limit(10).Find(&activities)
	if activities == nil {
		activities = []models.ActivityLog{}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"stats": fiber.Map{
			"totalPppoeUsers":              totalPppoeUsers,
			"activePppoeUsers":             activePppoeUsers,
			"activeSessionsPPPoE":          activeSessionsPPPoE,
			"activeSessionsHotspot":        activeSessionsHotspot,
			"unusedVouchers":               unusedVouchers,
			"isolatedCount":                isolatedCount,
			"suspendedCount":               suspendedCount,
			"newRegistrations":             newRegistrations,
			"upcomingInvoices":             upcomingInvoices,
			"voucherRevenue":               voucherRevenue,
			"voucherRevenueFormatted":      formatIDR(voucherRevenue),
			"voucherRevenueToday":          voucherRevenueToday,
			"voucherRevenueTodayFormatted": formatIDR(voucherRevenueToday),
			"invoiceRevenue":               invoiceRevenue,
			"invoiceRevenueFormatted":      formatIDR(invoiceRevenue),
			"invoiceRevenueToday":          invoiceRevenueToday,
			"invoiceRevenueTodayFormatted": formatIDR(invoiceRevenueToday),
			"invoiceCountToday":            invoiceCountToday,
			"invoiceCountMonth":            invoiceCountMonth,
			"unpaidInvoicesCount":          unpaidInvoicesCount,
			"totalAllTimeRevenue":          totalAllTimeRevenue,
			"totalAllTimeRevenueFormatted": formatIDR(totalAllTimeRevenue),
		},
		"activities": activities,
		"systemStatus": fiber.Map{
			"radius":   radiusOnline,
			"database": true,
			"api":      true,
		},
		"agentSales": agentSales,
		"agentSalesTotal": fiber.Map{
			"count":   agentSalesTotalCount,
			"revenue": agentSalesTotalRevenue,
		},
		"radiusAuthLog": radiusAuthLog,
		"radiusAuthStats": fiber.Map{
			"acceptToday": acceptToday,
			"rejectToday": rejectToday,
		},
		"periodLabel":    periodLabel,
		"monthKey":       monthKey,
		"isCurrentMonth": isCurrentMonth,
	})
}

// RevenueChart godoc
// GET /api/admin/revenue-chart
func (h *AdminHandler) RevenueChart(c fiber.Ctx) error {
	months := 12
	if v, err := strconv.Atoi(c.Query("months")); err == nil && v > 0 {
		months = v
	}

	type monthRevenue struct {
		Month   string `json:"month"`
		Revenue int64  `json:"revenue"`
		Count   int64  `json:"count"`
	}

	var rows []monthRevenue
	h.db.Raw(`
		SELECT DATE_FORMAT(paidAt, '%Y-%m') as month,
		       SUM(amount) as revenue,
		       COUNT(*) as count
		FROM invoices
		WHERE status = 'PAID'
		  AND paidAt >= DATE_SUB(NOW(), INTERVAL ? MONTH)
		GROUP BY DATE_FORMAT(paidAt, '%Y-%m')
		ORDER BY month ASC
	`, months).Scan(&rows)

	return c.JSON(rows)
}

// Activity godoc
// GET /api/admin/activity
func (h *AdminHandler) Activity(c fiber.Ctx) error {
	var cronHistory []models.CronHistory
	h.db.Order("startedAt DESC").Limit(20).Find(&cronHistory)

	return c.JSON(fiber.Map{
		"cronJobs": cronHistory,
	})
}

// IsolatedUsers godoc
// GET /api/admin/isolated-users
func (h *AdminHandler) IsolatedUsers(c fiber.Ctx) error {
	type isolatedRow struct {
		ID        string     `json:"id"`
		Username  string     `json:"username"`
		Name      string     `json:"name"`
		Phone     *string    `json:"phone"`
		Status    string     `json:"status"`
		ExpiredAt *time.Time `json:"expiredAt"`
		AreaName  *string    `json:"areaName"`
		Profile   *string    `json:"profileName"`
		Price     *int64     `json:"profilePrice"`
		UnpaidAmt *int64     `json:"unpaidAmount"`
		UnpaidCnt int        `json:"unpaidCount"`
	}

	var rows []isolatedRow
	h.db.Raw(`
		SELECT
			u.id, u.username, u.name, u.phone, u.status, u.expiredAt,
			a.name  AS areaName,
			p.name  AS profile,
			p.price AS price,
			COALESCE(SUM(CASE WHEN i.status IN ('PENDING','OVERDUE') THEN i.amount ELSE 0 END), 0) AS unpaidAmt,
			COUNT(CASE WHEN i.status IN ('PENDING','OVERDUE') THEN 1 END)                           AS unpaidCnt
		FROM pppoe_users u
		LEFT JOIN pppoe_areas    a ON u.areaId    = a.id
		LEFT JOIN pppoe_profiles p ON u.profileId = p.id
		LEFT JOIN invoices       i ON i.userId    = u.id
		WHERE u.status IN ('isolated','suspended')
		GROUP BY u.id
		ORDER BY u.expiredAt DESC
	`).Scan(&rows)

	return c.JSON(fiber.Map{"data": rows, "total": len(rows)})
}

// TopupRequests godoc
// GET /api/admin/topup-requests — list PPPoE customer top-up requests
func (h *AdminHandler) TopupRequests(c fiber.Ctx) error {
	status := c.Query("status", "PENDING")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 20
	}

	var requests []models.TopupRequest
	var total int64

	q := h.db.Model(&models.TopupRequest{}).Preload("User")
	if status != "all" && status != "ALL" {
		q = q.Where("status = ?", strings.ToUpper(status))
	}
	q.Count(&total)
	q.Order("createdAt DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&requests)

	return c.JSON(fiber.Map{"requests": requests, "total": total, "page": page})
}

// ApproveTopup godoc
// POST /api/admin/topup-requests/:id/approve
func (h *AdminHandler) ApproveTopup(c fiber.Ctx) error {
	id := c.Params("id")
	now := time.Now()

	var req models.TopupRequest
	if err := h.db.First(&req, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "request not found"})
	}
	if req.Status != "PENDING" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "request already processed"})
	}

	if err := h.db.Model(&req).Updates(map[string]interface{}{"status": "SUCCESS", "processedAt": now}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "approved"})
}

// RejectTopup godoc
// POST /api/admin/topup-requests/:id/reject
func (h *AdminHandler) RejectTopup(c fiber.Ctx) error {
	id := c.Params("id")
	now := time.Now()

	res := h.db.Model(&models.TopupRequest{}).Where("id = ? AND status = 'PENDING'", id).
		Updates(map[string]interface{}{"status": "FAILED", "processedAt": now})
	if res.RowsAffected == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "request not found or already processed"})
	}
	return c.JSON(fiber.Map{"message": "rejected"})
}

// SuspendRequests godoc
// GET /api/admin/suspend-requests
func (h *AdminHandler) SuspendRequests(c fiber.Ctx) error {
	status := c.Query("status", "PENDING")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 20
	}

	var rows []models.SuspendRequest
	var total int64

	q := h.db.Model(&models.SuspendRequest{}).Preload("User")
	if status != "all" && status != "ALL" {
		q = q.Where("status = ?", strings.ToUpper(status))
	}
	q.Count(&total)
	q.Order("requestedAt DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)

	return c.JSON(fiber.Map{"rows": rows, "total": total, "page": page})
}

// ApproveSuspend godoc
// POST /api/admin/suspend-requests/:id/approve
func (h *AdminHandler) ApproveSuspend(c fiber.Ctx) error {
	id := c.Params("id")
	now := time.Now()

	var req struct {
		UserID string
		Status string
	}
	if err := h.db.Raw("SELECT userId, status FROM suspend_requests WHERE id = ?", id).Scan(&req).Error; err != nil || req.UserID == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "request not found"})
	}
	if req.Status != "pending" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "already processed"})
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("UPDATE suspend_requests SET status='approved', processedAt=? WHERE id=?", now, id).Error; err != nil {
			return err
		}
		return tx.Exec("UPDATE pppoe_users SET status='suspended' WHERE id=?", req.UserID).Error
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "approved"})
}

// RejectSuspend godoc
// POST /api/admin/suspend-requests/:id/reject
func (h *AdminHandler) RejectSuspend(c fiber.Ctx) error {
	id := c.Params("id")
	now := time.Now()
	res := h.db.Exec("UPDATE suspend_requests SET status='rejected', processedAt=? WHERE id=? AND status='pending'", now, id)
	if res.RowsAffected == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "request not found or already processed"})
	}
	return c.JSON(fiber.Map{"message": "rejected"})
}

// SuspendRequestAction godoc
// PUT /api/admin/suspend-requests/:id
// Body: { action: "APPROVE" | "REJECT", adminNotes?: string }
func (h *AdminHandler) SuspendRequestAction(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Action     string  `json:"action"`
		AdminNotes *string `json:"adminNotes"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	now := time.Now()
	switch body.Action {
	case "APPROVE":
		var req struct {
			UserID string
			Status string
		}
		if err := h.db.Raw("SELECT userId, status FROM suspend_requests WHERE id = ?", id).Scan(&req).Error; err != nil || req.UserID == "" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "request not found"})
		}
		if req.Status != "pending" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "already processed"})
		}
		h.db.Exec("UPDATE suspend_requests SET status='approved', processedAt=? WHERE id=?", now, id)
		h.db.Exec("UPDATE pppoe_users SET status='suspended' WHERE id=?", req.UserID)
	case "REJECT":
		res := h.db.Exec("UPDATE suspend_requests SET status='rejected', processedAt=? WHERE id=? AND status='pending'", now, id)
		if res.RowsAffected == 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "not found or already processed"})
		}
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "action must be APPROVE or REJECT"})
	}
	return c.JSON(fiber.Map{"success": true})
}
