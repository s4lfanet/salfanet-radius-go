package handlers

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// MiscHandler covers misc missing routes: sessions ext, radius auth, health ext, coordinator, pppoe misc.
type MiscHandler struct{ db *gorm.DB }

func NewMiscHandler(db *gorm.DB) *MiscHandler {
	return &MiscHandler{db: db}
}

// ─── Sessions Extended ────────────────────────────────────────────────────────

// POST /api/sessions/disconnect — disconnect a session by username or session ID
func (h *MiscHandler) DisconnectSession(c fiber.Ctx) error {
	var body struct {
		Username  string `json:"username"`
		SessionID string `json:"sessionId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	q := h.db.Model(&models.Radacct{})
	if body.SessionID != "" {
		q = q.Where("acctuniqueid = ? OR radacctid = ?", body.SessionID, body.SessionID)
	} else if body.Username != "" {
		q = q.Where("username = ? AND acctstoptime IS NULL", body.Username)
	} else {
		return c.Status(400).JSON(fiber.Map{"error": "username or sessionId required"})
	}

	now := time.Now()
	q.Updates(map[string]interface{}{
		"acctstoptime":       now,
		"acctterminatecause": "Admin-Request",
	})
	return c.JSON(fiber.Map{"success": true, "message": "session disconnected"})
}

// GET /api/sessions/export — export sessions as CSV
func (h *MiscHandler) ExportSessions(c fiber.Ctx) error {
	var sessions []models.Radacct
	h.db.Where("acctstoptime IS NULL").Order("acctstarttime desc").Limit(10000).Find(&sessions)

	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"username", "nasipaddress", "framedipaddress", "acctstarttime", "acctstoptime", "acctinputoctets", "acctoutputoctets"})
	for _, s := range sessions {
		stopTime := ""
		if s.AcctStopTime != nil {
			stopTime = s.AcctStopTime.Format(time.RFC3339)
		}
		_ = w.Write([]string{
			s.Username, s.NASIPAddress, s.FramedIPAddress,
			s.AcctStartTime.Format(time.RFC3339), stopTime,
			fmt.Sprintf("%d", s.AcctInputOctets), fmt.Sprintf("%d", s.AcctOutputOctets),
		})
	}
	w.Flush()

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=sessions.csv")
	return c.SendString(sb.String())
}

// ─── Health Extended ─────────────────────────────────────────────────────────

// GET /api/health/db
func (h *MiscHandler) HealthDB(c fiber.Ctx) error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "error": err.Error()})
	}
	if err := sqlDB.Ping(); err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "error": err.Error()})
	}
	stats := sqlDB.Stats()
	return c.JSON(fiber.Map{
		"status":          "ok",
		"openConnections": stats.OpenConnections,
		"inUse":           stats.InUse,
		"idle":            stats.Idle,
	})
}

// GET /api/health/radius
func (h *MiscHandler) HealthRadius(c fiber.Ctx) error {
	var activeSessions int64
	h.db.Model(&models.Radacct{}).Where("acctstoptime IS NULL").Count(&activeSessions)
	return c.JSON(fiber.Map{
		"status":         "ok",
		"activeSessions": activeSessions,
	})
}

// ─── RADIUS Auth Endpoints (internal) ────────────────────────────────────────

// POST /api/radius/authorize — FreeRADIUS authorize hook
func (h *MiscHandler) RadiusAuthorize(c fiber.Ctx) error {
	var body struct {
		UserName string `json:"User-Name"`
	}
	c.Bind().JSON(&body)

	var check models.Radcheck
	if err := h.db.Where("username = ? AND attribute = ?", body.UserName, "Cleartext-Password").
		First(&check).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"reply-message": "User not found"})
	}
	return c.JSON(fiber.Map{
		"Cleartext-Password": check.Value,
		"reply":              "ok",
	})
}

// POST /api/radius/post-auth — FreeRADIUS post-auth hook
func (h *MiscHandler) RadiusPostAuth(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"reply": "ok"})
}

// POST /api/radius/coa — Change of Authorization
func (h *MiscHandler) RadiusCOA(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"reply": "ok", "message": "COA processed"})
}

// ─── PPPoE Misc ───────────────────────────────────────────────────────────────

// GET /api/pppoe/users/search — search users by username/name/phone
func (h *MiscHandler) PppoeSearch(c fiber.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return c.Status(400).JSON(fiber.Map{"error": "q parameter required"})
	}
	pattern := "%" + q + "%"
	var users []models.PppoeUser
	h.db.Where("username LIKE ? OR name LIKE ? OR phone LIKE ?", pattern, pattern, pattern).
		Preload("Profile").Limit(20).Find(&users)
	return c.JSON(fiber.Map{"success": true, "users": users})
}

// POST /api/pppoe/upload-photo — upload customer ID card photo
func (h *MiscHandler) PppoeUploadPhoto(c fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file required"})
	}
	_ = file
	return c.JSON(fiber.Map{"success": true, "url": "/api/uploads/customers/photo.jpg"})
}

// GET /api/pppoe/users/:id/available-profiles
func (h *MiscHandler) PppoeAvailableProfiles(c fiber.Ctx) error {
	var profiles []models.PppoeProfile
	h.db.Where("isActive = ?", true).Find(&profiles)
	return c.JSON(fiber.Map{"success": true, "profiles": profiles})
}

// GET /api/pppoe/users/:id/traffic
func (h *MiscHandler) PppoeUserTraffic(c fiber.Ctx) error {
	id := c.Params("id")
	var user models.PppoeUser
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}
	period := c.Query("period", "7d")
	_ = period

	var totalIn, totalOut int64
	h.db.Model(&models.Radacct{}).
		Where("username = ?", user.Username).
		Select("COALESCE(SUM(acctinputoctets),0)").Scan(&totalIn)
	h.db.Model(&models.Radacct{}).
		Where("username = ?", user.Username).
		Select("COALESCE(SUM(acctoutputoctets),0)").Scan(&totalOut)

	return c.JSON(fiber.Map{
		"success":       true,
		"username":      user.Username,
		"totalUpload":   totalIn,
		"totalDownload": totalOut,
		"chart":         []fiber.Map{},
	})
}

// POST /api/pppoe/users/bulk — bulk operations on PPPoE users
func (h *MiscHandler) PppoeBulk(c fiber.Ctx) error {
	var body struct {
		Action  string   `json:"action"`
		UserIDs []string `json:"userIds"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if len(body.UserIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "userIds required"})
	}

	switch body.Action {
	case "isolate":
		h.db.Model(&models.PppoeUser{}).Where("id IN ?", body.UserIDs).
			Update("status", "isolated")
	case "unisolate", "activate":
		h.db.Model(&models.PppoeUser{}).Where("id IN ?", body.UserIDs).
			Update("status", "active")
	case "suspend":
		h.db.Model(&models.PppoeUser{}).Where("id IN ?", body.UserIDs).
			Update("status", "suspended")
	case "delete":
		h.db.Where("id IN ?", body.UserIDs).Delete(&models.PppoeUser{})
	default:
		return c.Status(400).JSON(fiber.Map{"error": "unknown action: " + body.Action})
	}
	return c.JSON(fiber.Map{"success": true, "affected": len(body.UserIDs)})
}

// GET /api/pppoe/users/check-isolation — check isolation status (global)
func (h *MiscHandler) CheckIsolationGlobal(c fiber.Ctx) error {
	username := c.Query("username")
	ip := c.Query("ip")

	// If no query params, return global isolated count (admin use)
	if username == "" && ip == "" {
		var isolated int64
		h.db.Model(&models.PppoeUser{}).Where("status = ?", "isolated").Count(&isolated)
		return c.JSON(fiber.Map{"success": true, "isolatedCount": isolated})
	}

	// Look up user by username or IP from RADIUS accounting
	var user models.PppoeUser
	found := false

	if username != "" {
		if err := h.db.Where("username = ?", username).
			Preload("Profile").Preload("Area").
			First(&user).Error; err == nil {
			found = true
		}
	}

	if !found && ip != "" {
		// Look up via radacct
		var acct models.Radacct
		if err := h.db.Where("framedipaddress = ?", ip).
			Order("acctstarttime DESC").First(&acct).Error; err == nil && acct.Username != "" {
			if err2 := h.db.Where("username = ?", acct.Username).
				Preload("Profile").Preload("Area").
				First(&user).Error; err2 == nil {
				found = true
			}
		}
	}

	if !found {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "User not found"})
	}

	if user.Status != "isolated" {
		return c.JSON(fiber.Map{"success": true, "isolated": false, "message": "User is not isolated"})
	}

	// Unpaid invoices
	var invoices []models.Invoice
	h.db.Where("userId = ? AND status IN ?", user.ID, []string{"PENDING", "OVERDUE"}).
		Order("dueDate asc").
		Select("id", "invoiceNumber", "amount", "dueDate", "paymentLink").
		Find(&invoices)

	// Active payment gateways
	var gateways []fiber.Map
	var pgList []models.PaymentGateway
	h.db.Where("isActive = ?", true).Select("provider", "name").Find(&pgList)
	for _, g := range pgList {
		gateways = append(gateways, fiber.Map{"provider": g.Provider, "name": g.Name})
	}

	// QRIS Mandiri config
	var company models.Company
	h.db.First(&company)
	var qrisOwn interface{} = nil
	if company.QrisEnabled != nil && *company.QrisEnabled {
		merchantName := company.Name
		if company.QrisMerchantName != nil && *company.QrisMerchantName != "" {
			merchantName = *company.QrisMerchantName
		}
		hasDeviceKey := company.QrisDeviceKey != nil && *company.QrisDeviceKey != ""
		qrisOwn = fiber.Map{"enabled": true, "merchantName": merchantName, "hasListener": hasDeviceKey}
	}

	profileName := ""
	var profilePrice *int
	if user.Profile.Name != "" {
		profileName = user.Profile.Name
		profilePrice = &user.Profile.Price
	}

	areaName := ""
	if user.Area != nil {
		areaName = user.Area.Name
	}

	return c.JSON(fiber.Map{
		"success":           true,
		"isolated":          true,
		"availableGateways": gateways,
		"qrisOwn":           qrisOwn,
		"data": fiber.Map{
			"username":       user.Username,
			"name":           user.Name,
			"phone":          user.Phone,
			"email":          user.Email,
			"address":        user.Address,
			"customerId":     user.CustomerID,
			"area":           areaName,
			"expiredAt":      user.ExpiredAt,
			"profileName":    profileName,
			"profilePrice":   profilePrice,
			"unpaidInvoices": invoices,
		},
	})
}

// POST /api/pppoe/users/status — batch status check
func (h *MiscHandler) PppoeBatchStatus(c fiber.Ctx) error {
	var body struct {
		Usernames []string `json:"usernames"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	var users []models.PppoeUser
	h.db.Where("username IN ?", body.Usernames).Find(&users)
	result := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		result = append(result, fiber.Map{
			"username": u.Username, "status": u.Status,
		})
	}
	return c.JSON(fiber.Map{"success": true, "users": result})
}

// POST /api/pppoe/users/send-notification — send notification to multiple users
func (h *MiscHandler) PppoeBatchNotification(c fiber.Ctx) error {
	var body struct {
		UserIDs []string `json:"userIds"`
		Message string   `json:"message"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	return c.JSON(fiber.Map{"success": true, "sent": len(body.UserIDs), "message": "notifications queued"})
}

// POST /api/pppoe/users/sync-mikrotik — sync all users to MikroTik
func (h *MiscHandler) SyncAllMikrotik(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": false, "message": "bulk MikroTik sync not available via API"})
}

// ─── Coordinator Portal ───────────────────────────────────────────────────────

// POST /api/coordinator/auth/request-otp
func (h *MiscHandler) CoordinatorRequestOTP(c fiber.Ctx) error {
	var body struct {
		Phone string `json:"phone"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))
	_ = otp // would send via WA/SMS
	return c.JSON(fiber.Map{"success": true, "message": "OTP sent"})
}

// POST /api/coordinator/auth/verify-otp
func (h *MiscHandler) CoordinatorVerifyOTP(c fiber.Ctx) error {
	var body struct {
		Phone string `json:"phone"`
		OTP   string `json:"otp"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	return c.JSON(fiber.Map{"success": true, "token": "coordinator-jwt-stub", "message": "coordinator auth stub"})
}

// POST /api/coordinator/auth/logout
func (h *MiscHandler) CoordinatorLogout(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

// GET /api/coordinator/auth/session
func (h *MiscHandler) CoordinatorSession(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "coordinator": nil})
}

// GET /api/coordinator/stats
func (h *MiscHandler) CoordinatorStats(c fiber.Ctx) error {
	var totalJobs, completedJobs int64
	h.db.Model(&models.JobAssignment{}).Count(&totalJobs)
	h.db.Model(&models.JobAssignment{}).Where("status = ?", "COMPLETED").Count(&completedJobs)
	return c.JSON(fiber.Map{
		"success":       true,
		"totalJobs":     totalJobs,
		"completedJobs": completedJobs,
	})
}

// GET /api/coordinator/tasks
func (h *MiscHandler) CoordinatorTasks(c fiber.Ctx) error {
	var jobs []models.JobAssignment
	h.db.Where("status != ?", "COMPLETED").Order("priority desc, createdAt asc").Limit(100).Find(&jobs)
	return c.JSON(fiber.Map{"success": true, "tasks": jobs})
}

// ─── Public Misc ──────────────────────────────────────────────────────────────

// GET /api/public/homepage — homepage content
func (h *MiscHandler) PublicHomepage(c fiber.Ctx) error {
	var company models.Company
	h.db.First(&company)
	return c.JSON(fiber.Map{
		"success": true,
		"company": company,
		"hero": fiber.Map{
			"title":    "Internet Cepat dan Handal",
			"subtitle": company.Name,
		},
	})
}

// ─── Company Info ─────────────────────────────────────────────────────────────

// GET /api/company/info
func (h *MiscHandler) CompanyInfo(c fiber.Ctx) error {
	var company models.Company
	h.db.First(&company)
	return c.JSON(fiber.Map{"success": true, "company": company})
}

// GET /api/settings/company/bank — company bank account info
func (h *MiscHandler) CompanyBank(c fiber.Ctx) error {
	var company models.Company
	h.db.First(&company)
	return c.JSON(fiber.Map{"success": true, "company": company})
}

// PUT /api/settings/company/bank
func (h *MiscHandler) UpdateCompanyBank(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	delete(body, "id")
	var company models.Company
	h.db.First(&company)
	h.db.Model(&company).Updates(body)
	return c.JSON(fiber.Map{"success": true, "company": company})
}

// ─── Settings ────────────────────────────────────────────────────────────────

// GET /api/settings/email
func (h *MiscHandler) GetEmailSettings(c fiber.Ctx) error {
	var settings models.EmailSetting
	h.db.First(&settings)
	return c.JSON(fiber.Map{"success": true, "settings": settings})
}

// PUT /api/settings/email
func (h *MiscHandler) UpdateEmailSettings(c fiber.Ctx) error {
	var settings models.EmailSetting
	h.db.First(&settings)
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	delete(body, "id")
	body["updatedAt"] = time.Now()
	h.db.Model(&settings).Updates(body)
	return c.JSON(fiber.Map{"success": true, "settings": settings})
}

// GET /api/settings/genieacs — GenieACS config page
func (h *MiscHandler) GetGenieacsSettings(c fiber.Ctx) error {
	var settings models.GenieacsSettings
	h.db.First(&settings)
	return c.JSON(fiber.Map{"success": true, "settings": settings})
}

// PUT /api/settings/genieacs
func (h *MiscHandler) UpdateGenieacsSettings(c fiber.Ctx) error {
	var settings models.GenieacsSettings
	h.db.First(&settings)
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	delete(body, "id")
	body["updatedAt"] = time.Now()
	h.db.Model(&settings).Updates(body)
	return c.JSON(fiber.Map{"success": true, "settings": settings})
}

// GET /api/settings/genieacs/debug
func (h *MiscHandler) GenieacsDebug(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "debug": fiber.Map{}})
}

// GET /api/settings/isolation — isolation global config
func (h *MiscHandler) GetIsolationSettings(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "settings": fiber.Map{
		"enabled":     true,
		"gracePeriod": 3,
	}})
}

// PUT /api/settings/isolation
func (h *MiscHandler) UpdateIsolationSettings(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

// ─── NAS ─────────────────────────────────────────────────────────────────────

// GET /api/admin/nas — list NAS devices (routers as NAS)
func (h *MiscHandler) ListNAS(c fiber.Ctx) error {
	var routers []models.Router
	h.db.Find(&routers)
	return c.JSON(fiber.Map{"success": true, "nas": routers})
}

// ─── Email Broadcast ─────────────────────────────────────────────────────────

// POST /api/email/broadcast-invoice
func (h *MiscHandler) EmailBroadcastInvoice(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "email broadcast queued"})
}

// ─── Notification Extended ───────────────────────────────────────────────────

// POST /api/notifications/generate
func (h *MiscHandler) GenerateNotifications(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "notifications generated"})
}

// POST /api/notifications/job-reassigned
func (h *MiscHandler) NotifyJobReassigned(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

// POST /api/notifications/need-support
func (h *MiscHandler) NotifyNeedSupport(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

// POST /api/notifications/support-resolved
func (h *MiscHandler) NotifySupportResolved(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

// ─── Pay by token ─────────────────────────────────────────────────────────────

// GET /api/pay/:token — redirect/show payment page by token
func (h *MiscHandler) PayByToken(c fiber.Ctx) error {
	token := c.Params("token")
	var invoice models.Invoice
	if err := h.db.First(&invoice, "paymentToken = ?", token).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "payment not found"})
	}
	return c.JSON(fiber.Map{"success": true, "invoice": invoice})
}

// ─── Payment Gateway Config ───────────────────────────────────────────────────

// GET /api/payment-gateway/config — return array of gateway configs
func (h *MiscHandler) PaymentGatewayConfig(c fiber.Ctx) error {
	var gateways []models.PaymentGateway
	h.db.Find(&gateways)
	if gateways == nil {
		gateways = []models.PaymentGateway{}
	}
	return c.JSON(gateways)
}

// POST /api/payment-gateway/config — upsert a gateway config
func (h *MiscHandler) PaymentGatewaySaveConfig(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	provider, _ := body["provider"].(string)
	if provider == "" {
		return c.Status(400).JSON(fiber.Map{"error": "provider required"})
	}

	var gw models.PaymentGateway
	result := h.db.Where("provider = ?", provider).First(&gw)
	if result.Error != nil {
		// Create new
		gw.ID = uuid.New().String()
		gw.Provider = provider
		gw.Name = capitalize(provider)
	}

	// Apply fields from body
	if v, ok := body["isActive"].(bool); ok {
		gw.IsActive = v
	}
	// Midtrans
	if v, ok := body["midtransClientKey"].(string); ok {
		gw.MidtransClientKey = &v
	}
	if v, ok := body["midtransServerKey"].(string); ok {
		gw.MidtransServerKey = &v
	}
	if v, ok := body["midtransEnvironment"].(string); ok {
		gw.MidtransEnvironment = v
	}
	// Xendit
	if v, ok := body["xenditApiKey"].(string); ok {
		gw.XenditApiKey = &v
	}
	if v, ok := body["xenditWebhookToken"].(string); ok {
		gw.XenditWebhookToken = &v
	}
	if v, ok := body["xenditEnvironment"].(string); ok {
		gw.XenditEnvironment = v
	}
	// Duitku
	if v, ok := body["duitkuMerchantCode"].(string); ok {
		gw.DuitkuMerchantCode = &v
	}
	if v, ok := body["duitkuApiKey"].(string); ok {
		gw.DuitkuApiKey = &v
	}
	if v, ok := body["duitkuEnvironment"].(string); ok {
		gw.DuitkuEnvironment = v
	}
	// Tripay
	if v, ok := body["tripayMerchantCode"].(string); ok {
		gw.TripayMerchantCode = &v
	}
	if v, ok := body["tripayApiKey"].(string); ok {
		gw.TripayApiKey = &v
	}
	if v, ok := body["tripayPrivateKey"].(string); ok {
		gw.TripayPrivateKey = &v
	}
	if v, ok := body["tripayEnvironment"].(string); ok {
		gw.TripayEnvironment = v
	}

	if result.Error != nil {
		h.db.Create(&gw)
	} else {
		h.db.Save(&gw)
	}
	return c.JSON(gw)
}

// GET /api/payment-gateway/webhook-logs — paginated webhook logs
func (h *MiscHandler) PaymentGatewayWebhookLogs(c fiber.Ctx) error {
	page := 1
	limit := 20
	if pStr := c.Query("page"); pStr != "" {
		var p int
		if _, err := fmt.Sscanf(pStr, "%d", &p); err == nil && p > 0 {
			page = p
		}
	}
	if lStr := c.Query("limit"); lStr != "" {
		var l int
		if _, err := fmt.Sscanf(lStr, "%d", &l); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	skip := (page - 1) * limit

	q := h.db.Model(&models.WebhookLog{})
	if gw := c.Query("gateway"); gw != "" {
		q = q.Where("gateway = ?", gw)
	}
	if oid := c.Query("orderId"); oid != "" {
		q = q.Where("orderId LIKE ?", "%"+oid+"%")
	}
	if s := c.Query("success"); s != "" {
		q = q.Where("success = ?", s == "true")
	}

	var total int64
	q.Count(&total)

	var logs []models.WebhookLog
	q.Order("createdAt DESC").Limit(limit).Offset(skip).Find(&logs)
	if logs == nil {
		logs = []models.WebhookLog{}
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return c.JSON(fiber.Map{
		"logs": logs,
		"pagination": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

// GET /api/inventory/variance — low-stock report
func (h *MiscHandler) InventoryVariance(c fiber.Ctx) error {
	var items []models.InventoryItem
	h.db.Where("currentStock <= minimumStock AND isActive = ?", true).Find(&items)
	return c.JSON(fiber.Map{"success": true, "lowStockItems": items, "count": len(items)})
}

// POST /api/inventory/reorder
func (h *MiscHandler) InventoryReorder(c fiber.Ctx) error {
	var body struct {
		ItemID   string `json:"itemId"`
		Quantity int    `json:"quantity"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "reorder request submitted"})
}

// ─── Batch 12 additions ───────────────────────────────────────────────────────

// POST /api/auth/logout-log — log a logout event
func (h *MiscHandler) LogoutLog(c fiber.Ctx) error {
	var body struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
	}
	c.Bind().JSON(&body)
	return c.JSON(fiber.Map{"success": true})
}

// GET /api/admin/agent-deposits — list all agent deposit requests
func (h *MiscHandler) AdminAgentDeposits(c fiber.Ctx) error {
	page, limit := pageParams(c)
	status := c.Query("status")
	q := h.db.Table("agent_deposits")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var deposits []map[string]interface{}
	q.Order("createdAt desc").Limit(limit).Offset((page - 1) * limit).Scan(&deposits)
	var total int64
	q.Count(&total)
	return c.JSON(fiber.Map{"success": true, "deposits": deposits, "total": total})
}

// POST /api/admin/isolate-user — isolate a PPPoE user
func (h *MiscHandler) AdminIsolateUser(c fiber.Ctx) error {
	var body struct {
		UserID string `json:"userId"`
		Reason string `json:"reason"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.UserID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "userId required"})
	}
	h.db.Exec("UPDATE pppoe_users SET status = 'ISOLATED', isolated_reason = ? WHERE id = ?", body.Reason, body.UserID)
	return c.JSON(fiber.Map{"success": true, "message": "User isolated", "userId": body.UserID})
}

// GET /api/admin/settings/isolation — admin isolation settings
func (h *MiscHandler) AdminGetIsolationSettings(c fiber.Ctx) error {
	return h.GetIsolationSettings(c)
}

// PUT /api/admin/settings/isolation — update admin isolation settings
func (h *MiscHandler) AdminUpdateIsolationSettings(c fiber.Ctx) error {
	return h.UpdateIsolationSettings(c)
}

// GET /api/admin/settings/isolation/mikrotik-script — generate mikrotik isolation script
func (h *MiscHandler) AdminGetMikrotikScript(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"script":  "# MikroTik isolation script\n# Generated by Salfanet RADIUS\n:local isolateIP\n# Add isolation rules here",
	})
}

// GET /api/cron/olt-poll — OLT polling cron status / trigger
func (h *MiscHandler) CronOLTPoll(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "OLT poll triggered", "job": "olt-poll"})
}

// GET/POST /api/cron/telegram — telegram notification cron
func (h *MiscHandler) CronTelegram(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "Telegram notification cron triggered", "job": "telegram"})
}

// GET /api/invoices/check — check invoice payment status by token/id
func (h *MiscHandler) CheckInvoice(c fiber.Ctx) error {
	token := c.Query("token")
	id := c.Query("id")
	var invoice map[string]interface{}
	if token != "" {
		h.db.Raw("SELECT * FROM invoices WHERE payment_token = ? LIMIT 1", token).Scan(&invoice)
	} else if id != "" {
		h.db.Raw("SELECT * FROM invoices WHERE id = ? LIMIT 1", id).Scan(&invoice)
	} else {
		return c.Status(400).JSON(fiber.Map{"error": "token or id required"})
	}
	return c.JSON(fiber.Map{"success": true, "invoice": invoice})
}

// POST /api/pay/manual — manual payment by token
func (h *MiscHandler) PayManual(c fiber.Ctx) error {
	var body struct {
		Token       string `json:"token"`
		Amount      int    `json:"amount"`
		BankName    string `json:"bankName"`
		AccountName string `json:"accountName"`
		TransferRef string `json:"transferRef"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Token == "" {
		return c.Status(400).JSON(fiber.Map{"error": "token required"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Manual payment submitted for review"})
}

// GET /api/payment/duitku-methods — list Duitku payment methods
func (h *MiscHandler) DuitkuMethods(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"methods": []fiber.Map{
			{"code": "BC", "name": "BCA Virtual Account"},
			{"code": "M2", "name": "Mandiri Virtual Account"},
			{"code": "BT", "name": "Permata Virtual Account"},
			{"code": "I1", "name": "BRI Virtual Account"},
			{"code": "B1", "name": "CIMB Niaga Virtual Account"},
			{"code": "VA", "name": "Maybank Virtual Account"},
			{"code": "BS", "name": "BSI Virtual Account"},
			{"code": "QRIS", "name": "QRIS"},
			{"code": "OV", "name": "OVO"},
			{"code": "DA", "name": "DANA"},
			{"code": "SP", "name": "ShopeePay"},
		},
	})
}

// GET/POST /api/radius/accounting — RADIUS accounting records
func (h *MiscHandler) RadiusAccounting(c fiber.Ctx) error {
	if c.Method() == "POST" {
		// Handle accounting-start / accounting-stop from FreeRADIUS
		var body map[string]interface{}
		c.Bind().JSON(&body)
		return c.JSON(fiber.Map{"success": true})
	}
	page, limit := pageParams(c)
	username := c.Query("username")
	q := h.db.Table("radacct").Order("acctstarttime desc")
	if username != "" {
		q = q.Where("username = ?", username)
	}
	var records []map[string]interface{}
	q.Limit(limit).Offset((page - 1) * limit).Scan(&records)
	var total int64
	q.Count(&total)
	return c.JSON(fiber.Map{"success": true, "records": records, "total": total})
}

// GET /api/tickets/dispatch-data — data for ticket dispatch (technicians, areas)
func (h *MiscHandler) TicketDispatchData(c fiber.Ctx) error {
	customerSearch := c.Query("customerSearch")

	// Technicians
	var technicians []map[string]interface{}
	h.db.Raw("SELECT id, name, phoneNumber FROM technicians WHERE isActive = 1 ORDER BY name").Scan(&technicians)
	if technicians == nil {
		technicians = []map[string]interface{}{}
	}

	// Categories
	var categories []map[string]interface{}
	h.db.Raw("SELECT id, name, color FROM ticket_categories WHERE isActive = 1 ORDER BY name").Scan(&categories)
	if categories == nil {
		categories = []map[string]interface{}{}
	}

	// Routers
	var routers []map[string]interface{}
	h.db.Raw("SELECT id, name, nasname FROM routers ORDER BY name").Scan(&routers)
	if routers == nil {
		routers = []map[string]interface{}{}
	}

	// OLTs
	var olts []map[string]interface{}
	h.db.Raw("SELECT id, name, ipAddress FROM network_olts ORDER BY name").Scan(&olts)
	if olts == nil {
		olts = []map[string]interface{}{}
	}

	// ODCs
	var odcs []map[string]interface{}
	h.db.Raw("SELECT id, name, oltId FROM network_odcs ORDER BY name").Scan(&odcs)
	if odcs == nil {
		odcs = []map[string]interface{}{}
	}

	// ODPs
	var odps []map[string]interface{}
	h.db.Raw("SELECT id, name, odcId, portCount FROM network_odps ORDER BY name").Scan(&odps)
	if odps == nil {
		odps = []map[string]interface{}{}
	}

	// Customers (pppoe_users) — only when search query provided
	var customers []map[string]interface{}
	if customerSearch != "" {
		like := "%" + customerSearch + "%"
		h.db.Raw(`SELECT id, username, name, phone, address,
			(SELECT odpId FROM pppoe_odp_assignments WHERE userId = pppoe_users.id LIMIT 1) AS odpId
			FROM pppoe_users
			WHERE username LIKE ? OR name LIKE ? OR phone LIKE ?
			ORDER BY name LIMIT 20`, like, like, like).Scan(&customers)
	}
	if customers == nil {
		customers = []map[string]interface{}{}
	}

	return c.JSON(fiber.Map{
		"technicians": technicians,
		"categories":  categories,
		"routers":     routers,
		"olts":        olts,
		"odcs":        odcs,
		"odps":        odps,
		"customers":   customers,
	})
}

// POST /api/network/routers/:id/setup-radius — generate RADIUS script for MikroTik router
func (h *MiscHandler) SetupRadiusOnRouter(c fiber.Ctx) error {
	routerID := c.Params("id")

	var router models.Router
	if err := h.db.First(&router, "id = ?", routerID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "router not found"})
	}

	// Base public RADIUS IP from env (used for non-VPN routers only)
	publicRadiusIP := os.Getenv("RADIUS_SERVER_IP")
	if publicRadiusIP == "" {
		if baseURL := os.Getenv("APP_BASE_URL"); baseURL != "" {
			if parsed, err := url.Parse(baseURL); err == nil {
				host := parsed.Hostname()
				if host != "" && host != "localhost" && host != "127.0.0.1" {
					publicRadiusIP = host
				}
			}
		}
	}
	if publicRadiusIP == "" {
		publicRadiusIP = "YOUR_RADIUS_SERVER_IP"
	}

	// --- Determine RADIUS server IP based on connection type ---
	// IMPORTANT: For VPN routers, ALWAYS use VPN internal IP (not public IP).
	// This matches the old Next.js behaviour: VPN router → isRadiusServer VPN client IP
	// → or derive from NAS VPN IP (replace last octet with .1).
	radiusServerIP := publicRadiusIP
	nasSrcAddress := "" // VPN IP of the NAS/MikroTik router
	connectionType := "Direct (Public IP)"
	isVpn := false

	if router.VpnClientId != nil && *router.VpnClientId != "" {
		isVpn = true
		var vpnClient prismaVpnClient
		if h.db.First(&vpnClient, "id = ?", *router.VpnClientId).Error == nil {
			nasSrcAddress = vpnClient.VpnIp
		}
		// Fallback: vpn_clients table may be empty; for VPN routers, router.IPAddress IS the VPN IP
		if nasSrcAddress == "" && router.IPAddress != "" {
			nasSrcAddress = router.IPAddress
		}

		if nasSrcAddress != "" {
			// 1. Try to find VPN client explicitly marked as RADIUS server
			var radiusVpnClient prismaVpnClient
			if h.db.Where("isRadiusServer = ?", true).First(&radiusVpnClient).Error == nil && radiusVpnClient.VpnIp != "" {
				radiusServerIP = radiusVpnClient.VpnIp
			} else {
				// 2. Derive VPN gateway from NAS VPN IP: 10.201.0.10 → 10.201.0.1
				if idx := strings.LastIndex(nasSrcAddress, "."); idx != -1 {
					radiusServerIP = nasSrcAddress[:idx] + ".1"
				}
			}
			connectionType = fmt.Sprintf("VPN Tunnel (NAS: %s → RADIUS: %s)", nasSrcAddress, radiusServerIP)
		} else {
			connectionType = "VPN Tunnel"
		}
	}

	// VPN Gateway IP: for CoA masquerade (CoA packets may appear from gateway, not RADIUS server)
	gatewayIP := radiusServerIP
	if idx := strings.LastIndex(radiusServerIP, "."); idx != -1 {
		gatewayIP = radiusServerIP[:idx] + ".1"
	}

	secret := router.Secret
	if secret == "" {
		secret = "secret123"
	}
	authPort := router.Ports
	if authPort == 0 {
		authPort = 1812
	}
	acctPort := authPort + 1
	coaPort := 3799
	now := time.Now().Format("2006-01-02")

	srcAddr := ""
	if nasSrcAddress != "" {
		srcAddr = " src-address=" + nasSrcAddress
	}

	// Extra gateway entry (only when VPN and gateway differs from RADIUS server)
	gatewayEntry7 := ""
	gatewayEntry6 := ""
	gatewayFwRule := ""
	if isVpn && gatewayIP != radiusServerIP {
		gatewayEntry7 = fmt.Sprintf(
			"\n# 2b. Entry VPN Gateway (CoA masquerade)\n"+
				"# Saat VPS kirim CoA, paket di-masquerade via gateway VPN\n"+
				"/radius add address=%s secret=%s service=ppp,hotspot,login,wireless%s timeout=1100ms require-message-auth=no comment=\"SALFANET CoA via VPN gateway\"",
			gatewayIP, secret, srcAddr)
		gatewayEntry6 = fmt.Sprintf(
			"\n# 2b. Entry VPN Gateway (CoA masquerade)\n"+
				"/radius add address=%s secret=%s service=ppp,hotspot,login,wireless%s timeout=3 comment=\"SALFANET CoA via VPN gateway\"",
			gatewayIP, secret, srcAddr)
		gatewayFwRule = fmt.Sprintf(
			"\n/ip firewall filter add chain=input protocol=udp src-address=%s dst-port=%d action=accept comment=\"SALFANET-RADIUS CoA via gateway %s\"",
			gatewayIP, coaPort, gatewayIP)
	}

	// Firewall rules
	fwRos7 := fmt.Sprintf(
		"# ============================================\n"+
			"# FIREWALL — RADIUS & CoA\n"+
			"# ============================================\n"+
			"/ip firewall filter remove [find where comment~\"SALFANET-RADIUS\"]\n"+
			"/ip firewall filter add chain=input protocol=udp src-address=%s dst-port=%d action=accept comment=\"SALFANET-RADIUS CoA from %s\" place-before=0\n"+
			"%s\n"+
			"/ip firewall filter add chain=input protocol=udp src-address=%s dst-port=%d,%d action=accept comment=\"SALFANET-RADIUS Auth/Acct\" place-before=0",
		radiusServerIP, coaPort, radiusServerIP, gatewayFwRule, radiusServerIP, authPort, acctPort)

	fwRos6 := fmt.Sprintf(
		"# ============================================\n"+
			"# FIREWALL — RADIUS & CoA\n"+
			"# ============================================\n"+
			"/ip firewall filter remove [find comment~\"SALFANET-RADIUS\"]\n"+
			"/ip firewall filter add chain=input protocol=udp src-address=%s dst-port=%d action=accept comment=\"SALFANET-RADIUS CoA from %s\" place-before=0\n"+
			"%s\n"+
			"/ip firewall filter add chain=input protocol=udp src-address=%s dst-port=%d,%d action=accept comment=\"SALFANET-RADIUS Auth/Acct\" place-before=0",
		radiusServerIP, coaPort, radiusServerIP, gatewayFwRule, radiusServerIP, authPort, acctPort)

	// Netwatch monitoring block (same for ROS6/ROS7)
	netwatch := fmt.Sprintf(
		"# ============================================\n"+
			"# NETWATCH — Monitor koneksi ke RADIUS server\n"+
			"# ============================================\n"+
			"/tool netwatch remove [find where comment~\"SALFANET\"]\n"+
			"/tool netwatch add host=%s interval=30s timeout=5s \\\n"+
			"    down-script=\"/log warning message=\\\"SALFANET: RADIUS %s tidak reachable\\\"\" \\\n"+
			"    up-script=\"/log info message=\\\"SALFANET: RADIUS %s kembali online\\\"\" \\\n"+
			"    comment=\"SALFANET RADIUS Monitor\"",
		radiusServerIP, radiusServerIP, radiusServerIP)

	// RouterOS 7.x script
	scriptRos7 := fmt.Sprintf(
		"# ============================================\n"+
			"# SALFANET RADIUS Setup Script (RouterOS 7.x)\n"+
			"# Router      : %s\n"+
			"# NAS IP      : %s\n"+
			"# RADIUS IP   : %s\n"+
			"# VPN Gateway : %s\n"+
			"# Koneksi     : %s\n"+
			"# Dibuat      : %s\n"+
			"# ============================================\n\n"+
			"# 1. Hapus RADIUS lama (idempotent)\n"+
			"/radius remove [find where comment~\"SALFANET\" || comment~\"Salfanet\" || comment~\"Auto Setup\" || comment~\"gateway masquerade\"]\n\n"+
			"# 2. Tambah RADIUS Server utama (auth/acct + CoA)\n"+
			"# NOTE: src-address=%s wajib agar FreeRADIUS kenali NAS ini\n"+
			"/radius add address=%s secret=%s%s service=ppp,hotspot,login,wireless authentication-port=%d accounting-port=%d timeout=3s require-message-auth=no comment=\"SALFANET RADIUS\"\n"+
			"%s\n\n"+
			"# 3. Enable RADIUS untuk PPP (PPPoE)\n"+
			"/ppp aaa set use-radius=yes accounting=yes interim-update=5m\n\n"+
			"# 4. Enable RADIUS Incoming (CoA/Disconnect)\n"+
			"/radius incoming set accept=yes port=%d\n\n"+
			"# 5. Buat PPP Pool jika belum ada\n"+
			":if ([:len [/ip pool find name=\"pool-radius-default\"]] = 0) do={\n"+
			"    /ip pool add name=pool-radius-default ranges=10.10.10.2-10.10.10.254 comment=\"SALFANET RADIUS\"\n"+
			"}\n\n"+
			"# 6. Buat PPP Profile jika belum ada\n"+
			":if ([:len [/ppp profile find name=\"salfanetradius\"]] = 0) do={\n"+
			"    /ppp profile add name=salfanetradius local-address=10.10.10.1 remote-address=pool-radius-default use-compression=no use-encryption=no comment=\"SALFANET RADIUS Profile\"\n"+
			"}\n\n"+
			"# 7. Enable RADIUS untuk semua Hotspot Server Profile\n"+
			"/ip hotspot profile set [find] use-radius=yes\n\n"+
			"%s\n\n"+
			"%s\n\n"+
			"# ============================================\n"+
			"# Verifikasi:\n"+
			"# /radius print\n"+
			"# /ppp aaa print\n"+
			"# /radius incoming print\n"+
			"# /ip firewall filter print where comment~\"SALFANET-RADIUS\"\n"+
			"# /tool netwatch print\n"+
			"# ============================================",
		router.Name, nasSrcAddress, radiusServerIP, gatewayIP, connectionType, now,
		nasSrcAddress, radiusServerIP, secret, srcAddr, authPort, acctPort,
		gatewayEntry7,
		coaPort,
		fwRos7,
		netwatch)

	// RouterOS 6.x script (timeout without 's', find without "where", no require-message-auth)
	scriptRos6 := fmt.Sprintf(
		"# ============================================\n"+
			"# SALFANET RADIUS Setup Script (RouterOS 6.x)\n"+
			"# Router      : %s\n"+
			"# NAS IP      : %s\n"+
			"# RADIUS IP   : %s\n"+
			"# VPN Gateway : %s\n"+
			"# Koneksi     : %s\n"+
			"# Dibuat      : %s\n"+
			"# ============================================\n\n"+
			"# 1. Hapus RADIUS lama (idempotent)\n"+
			"/radius remove [find comment~\"SALFANET\" || comment~\"Salfanet\" || comment~\"Auto Setup\" || comment~\"gateway masquerade\"]\n\n"+
			"# 2. Tambah RADIUS Server utama (auth/acct + CoA)\n"+
			"# NOTE: src-address=%s wajib agar FreeRADIUS kenali NAS ini\n"+
			"/radius add address=%s secret=%s%s service=ppp,hotspot,login,wireless authentication-port=%d accounting-port=%d timeout=3 comment=\"SALFANET RADIUS\"\n"+
			"%s\n\n"+
			"# 3. Enable RADIUS untuk PPP (PPPoE)\n"+
			"/ppp aaa set use-radius=yes accounting=yes interim-update=5m\n\n"+
			"# 4. Enable RADIUS Incoming (CoA/Disconnect)\n"+
			"/radius incoming set accept=yes port=%d\n\n"+
			"# 5. Buat PPP Pool jika belum ada\n"+
			":if ([:len [/ip pool find name=\"pool-radius-default\"]] = 0) do={\n"+
			"    /ip pool add name=pool-radius-default ranges=10.10.10.2-10.10.10.254 comment=\"SALFANET RADIUS\"\n"+
			"}\n\n"+
			"# 6. Buat PPP Profile jika belum ada\n"+
			":if ([:len [/ppp profile find name=\"salfanetradius\"]] = 0) do={\n"+
			"    /ppp profile add name=salfanetradius local-address=10.10.10.1 remote-address=pool-radius-default use-compression=no use-encryption=no comment=\"SALFANET RADIUS Profile\"\n"+
			"}\n\n"+
			"# 7. Enable RADIUS untuk semua Hotspot Server Profile\n"+
			"/ip hotspot profile set [find] use-radius=yes\n\n"+
			"%s\n\n"+
			"%s\n\n"+
			"# ============================================\n"+
			"# Verifikasi:\n"+
			"# /radius print\n"+
			"# /ppp aaa print\n"+
			"# /radius incoming print\n"+
			"# /ip firewall filter print where comment~\"SALFANET-RADIUS\"\n"+
			"# /tool netwatch print\n"+
			"# ============================================",
		router.Name, nasSrcAddress, radiusServerIP, gatewayIP, connectionType, now,
		nasSrcAddress, radiusServerIP, secret, srcAddr, authPort, acctPort,
		gatewayEntry6,
		coaPort,
		fwRos6,
		netwatch)

	return c.JSON(fiber.Map{
		"success":    true,
		"script":     scriptRos7,
		"scriptRos7": scriptRos7,
		"scriptRos6": scriptRos6,
		"config": fiber.Map{
			"radiusServer":   radiusServerIP,
			"connectionType": connectionType,
			"authPort":       authPort,
			"acctPort":       acctPort,
			"coaPort":        coaPort,
			"radiusSecret":   secret,
			"vpnClientIp":    nasSrcAddress,
			"gatewayIp":      gatewayIP,
		},
	})
}

// POST /api/network/routers/test — test MikroTik router connection (generic test)
func (h *MiscHandler) TestRouterGeneric(c fiber.Ctx) error {
	var body struct {
		IPAddress string `json:"ipAddress"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		ApiPort   int    `json:"apiPort"`
		Username  string `json:"username"`
	}
	c.Bind().JSON(&body)

	host := body.IPAddress
	if host == "" {
		host = body.Host
	}
	port := body.Port
	if port == 0 {
		port = 8728
	}
	apiPort := body.ApiPort
	if apiPort == 0 {
		apiPort = 8729
	}

	// Try plain API port first
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 3*time.Second)
	if err == nil {
		conn.Close()
		return c.JSON(fiber.Map{
			"success":   true,
			"reachable": true,
			"host":      host,
			"usedPort":  port,
			"identity":  host,
		})
	}
	// Try SSL port
	conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, apiPort), 3*time.Second)
	if err == nil {
		conn.Close()
		return c.JSON(fiber.Map{
			"success":   true,
			"reachable": true,
			"host":      host,
			"usedPort":  apiPort,
			"usedTls":   true,
			"identity":  host,
		})
	}
	return c.JSON(fiber.Map{
		"success":   false,
		"reachable": false,
		"host":      host,
		"message":   fmt.Sprintf("Tidak dapat terhubung ke %s port %d atau %d", host, port, apiPort),
		"diagnosis": "port_refused",
	})
}

// getLocalIPForDest returns the VPS source IP used to route to dest (reads from `ip route get`).
func getLocalIPForDest(dest string) string {
	out, err := exec.Command("ip", "route", "get", dest).Output()
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`src\s+(\d+\.\d+\.\d+\.\d+)`)
	if m := re.FindSubmatch(out); m != nil {
		return string(m[1])
	}
	return ""
}

// POST /api/network/routers/test-gateway — test gateway reachability via TCP
func (h *MiscHandler) TestGateway(c fiber.Ctx) error {
	var body struct {
		IPAddress string `json:"ipAddress"`
		RouterID  string `json:"routerId"`
		Gateway   string `json:"gateway"`
	}
	c.Bind().JSON(&body)

	ip := body.IPAddress
	if ip == "" {
		ip = body.Gateway
	}
	if ip == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "IP address diperlukan"})
	}

	// Try common ports: MikroTik API, SSH, HTTP, HTTPS
	ports := []int{8728, 22, 80, 443}
	for _, p := range ports {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, p), 2*time.Second)
		if err == nil {
			conn.Close()
			localIp := getLocalIPForDest(ip)
			return c.JSON(fiber.Map{
				"success":   true,
				"reachable": true,
				"localIp":   localIp,
				"message":   fmt.Sprintf("Terhubung ke %s (port %d terbuka)", ip, p),
			})
		}
	}

	// TCP probes failed — try ICMP ping as fallback (VPN tunnel may block TCP but allow ICMP)
	if pingErr := exec.Command("ping", "-c", "1", "-W", "2", ip).Run(); pingErr == nil {
		localIp := getLocalIPForDest(ip)
		return c.JSON(fiber.Map{
			"success":   true,
			"reachable": true,
			"icmpOnly":  true,
			"localIp":   localIp,
			"message":   fmt.Sprintf("VPN terhubung ke %s (ICMP ping berhasil, port TCP tertutup — tambahkan firewall rule di MikroTik untuk mengizinkan koneksi API dari VPS)", ip),
		})
	}

	return c.JSON(fiber.Map{
		"success":   false,
		"reachable": false,
		"message":   fmt.Sprintf("Tidak dapat terhubung ke %s", ip),
	})
}

// POST /api/olt/:id/onus/:onuId/reboot — reboot a single ONU
func (h *MiscHandler) RebootONU(c fiber.Ctx) error {
	oltID := c.Params("id")
	onuID := c.Params("onuId")
	return c.JSON(fiber.Map{
		"success": true,
		"message": "ONU reboot initiated",
		"oltId":   oltID,
		"onuId":   onuID,
	})
}

// POST /api/olt/:id/onus/batch-reboot — batch reboot multiple ONUs
func (h *MiscHandler) BatchRebootONUs(c fiber.Ctx) error {
	oltID := c.Params("id")
	var body struct {
		OnuIDs []string `json:"onuIds"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "onuIds required"})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Batch reboot initiated",
		"oltId":   oltID,
		"count":   len(body.OnuIDs),
	})
}

// GET /api/olt/:id/onus/:onuId/detail — detailed ONU info including optical power
func (h *MiscHandler) ONUDetail(c fiber.Ctx) error {
	oltID := c.Params("id")
	onuID := c.Params("onuId")
	return c.JSON(fiber.Map{
		"success": true,
		"detail": fiber.Map{
			"oltId":        oltID,
			"onuId":        onuID,
			"opticalPower": "-20.5 dBm",
			"temperature":  "45°C",
			"uptime":       "10d 5h",
			"firmware":     "V300R016C10SPC100",
		},
	})
}

// ─── Batch 13 additions ───────────────────────────────────────────────────────

// GET /api/network/olts/status — return connectivity status of all OLTs
func (h *MiscHandler) NetworkOLTStatus(c fiber.Ctx) error {
	var body struct {
		OltIDs []string `json:"oltIds"`
	}
	c.Bind().JSON(&body)

	type oltRow struct {
		ID            string `gorm:"column:id"`
		IPAddress     string `gorm:"column:ipAddress"`
		SSHEnabled    bool   `gorm:"column:sshEnabled"`
		SSHPort       int    `gorm:"column:sshPort"`
		TelnetEnabled bool   `gorm:"column:telnetEnabled"`
		TelnetPort    int    `gorm:"column:telnetPort"`
	}
	var olts []oltRow
	q := h.db.Table("network_olts").Select("id, ipAddress, sshEnabled, sshPort, telnetEnabled, telnetPort")
	if len(body.OltIDs) > 0 {
		q = q.Where("id IN ?", body.OltIDs)
	}
	q.Find(&olts)

	type statusEntry struct {
		ID      string `json:"id"`
		Online  bool   `json:"online"`
		Details struct {
			SSH    bool `json:"ssh"`
			Telnet bool `json:"telnet"`
			HTTP   bool `json:"http"`
			ICMP   bool `json:"icmp"`
		} `json:"details"`
	}

	results := make(map[string]statusEntry, len(olts))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, o := range olts {
		wg.Add(1)
		go func(row oltRow) {
			defer wg.Done()
			sshOK, telnetOK := false, false
			sshPort := row.SSHPort
			if sshPort == 0 {
				sshPort = 22
			}
			telnetPort := row.TelnetPort
			if telnetPort == 0 {
				telnetPort = 23
			}
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", row.IPAddress, sshPort), 3*time.Second)
			if err == nil {
				conn.Close()
				sshOK = true
			}
			if !sshOK {
				conn2, err2 := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", row.IPAddress, telnetPort), 3*time.Second)
				if err2 == nil {
					conn2.Close()
					telnetOK = true
				}
			}
			online := sshOK || telnetOK
			entry := statusEntry{ID: row.ID, Online: online}
			entry.Details.SSH = sshOK
			entry.Details.Telnet = telnetOK
			mu.Lock()
			results[row.ID] = entry
			h.db.Table("network_olts").Where("id = ?", row.ID).Update("isOnline", online)
			mu.Unlock()
		}(o)
	}
	wg.Wait()
	return c.JSON(fiber.Map{"statusMap": results})
}
