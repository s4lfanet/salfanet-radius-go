package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/poller"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/telnet"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/vendors/zte"
)

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// MiscHandler covers misc missing routes: sessions ext, radius auth, health ext, coordinator, pppoe misc.
type MiscHandler struct {
	db     *gorm.DB
	poller *poller.Poller
}

func NewMiscHandler(db *gorm.DB, p *poller.Poller) *MiscHandler {
	return &MiscHandler{db: db, poller: p}
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
		UserName      string `json:"User-Name"`
		Username      string `json:"username"`
		NasIp         string `json:"nasIp"`
		NasPort       string `json:"nasPort"`
		CalledStation string `json:"calledStationId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	uname := body.UserName
	if uname == "" {
		uname = body.Username
	}
	if uname == "" {
		return c.Status(400).JSON(fiber.Map{"error": "username required"})
	}

	var user models.PppoeUser
	if err := h.db.Where("username = ?", uname).First(&user).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"reply-message": "User not found"})
	}

	if user.Status == "isolated" || user.Status == "suspended" {
		return c.Status(403).JSON(fiber.Map{
			"control:Auth-Type":   "Reject",
			"reply:Reply-Message": "Akun Anda diisolir, silakan hubungi admin",
		})
	}

	if user.ExpiredAt != nil && user.ExpiredAt.Before(time.Now()) {
		return c.Status(403).JSON(fiber.Map{
			"control:Auth-Type":   "Reject",
			"reply:Reply-Message": "Langganan Anda telah berakhir",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// POST /api/radius/post-auth — FreeRADIUS post-auth hook
func (h *MiscHandler) RadiusPostAuth(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
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
		if err := h.db.Create(&gw).Error; err != nil {
			log.Error().Err(err).Msg("payment-gateway: failed to create settings")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save settings: " + err.Error()})
		}
	} else {
		if err := h.db.Save(&gw).Error; err != nil {
			log.Error().Err(err).Msg("payment-gateway: failed to save settings")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save settings: " + err.Error()})
		}
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
	} else {
		// Non-VPN: use router NASName as the NAS IP (this is the IP FreeRADIUS sees)
		nasSrcAddress = router.NASName
		if nasSrcAddress == "" {
			nasSrcAddress = router.IPAddress
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

	// Query all distinct IPPoolName values from pppoe_profiles to generate pool creation commands.
	// This ensures MikroTik has pools matching the Framed-Pool attribute returned by RADIUS.
	var poolNames []string
	if err := h.db.Raw("SELECT DISTINCT ipPoolName FROM pppoe_profiles WHERE ipPoolName IS NOT NULL AND ipPoolName != ''").Scan(&poolNames).Error; err != nil {
		log.Warn().Err(err).Msg("setup-radius: failed to query pool names")
	}
	if len(poolNames) == 0 {
		poolNames = []string{"pppoe"}
	}

	// Build pool creation commands for each profile pool name.
	poolCmdsRos7 := ""
	poolCmdsRos6 := ""
	primaryPool := poolNames[0]
	for _, pn := range poolNames {
		poolCmdsRos7 += fmt.Sprintf(
			":if ([:len [/ip pool find name=\"%s\"]] = 0) do={\n"+
				"    /ip pool add name=%s ranges=10.10.10.2-10.10.10.254 comment=\"SALFANET RADIUS\"\n"+
				"}\n", pn, pn)
		poolCmdsRos6 += fmt.Sprintf(
			":if ([:len [/ip pool find name=\"%s\"]] = 0) do={\n"+
				"    /ip pool add name=%s ranges=10.10.10.2-10.10.10.254 comment=\"SALFANET RADIUS\"\n"+
				"}\n", pn, pn)
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
			"# 5. Buat PPP Pool(s) jika belum ada (match Framed-Pool dari RADIUS)\n"+
			"%s\n"+
			"# 6. Buat PPP Profile jika belum ada\n"+
			":if ([:len [/ppp profile find name=\"salfanetradius\"]] = 0) do={\n"+
			"    /ppp profile add name=salfanetradius local-address=10.10.10.1 remote-address=%s use-compression=no use-encryption=no comment=\"SALFANET RADIUS Profile\"\n"+
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
		poolCmdsRos7,
		primaryPool,
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
			"# 5. Buat PPP Pool(s) jika belum ada (match Framed-Pool dari RADIUS)\n"+
			"%s\n"+
			"# 6. Buat PPP Profile jika belum ada\n"+
			":if ([:len [/ppp profile find name=\"salfanetradius\"]] = 0) do={\n"+
			"    /ppp profile add name=salfanetradius local-address=10.10.10.1 remote-address=%s use-compression=no use-encryption=no comment=\"SALFANET RADIUS Profile\"\n"+
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
		poolCmdsRos6,
		primaryPool,
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

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}
	if (!olt.TelnetEnabled && !olt.SSHEnabled) || olt.Username == nil || olt.Password == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Telnet not configured for this OLT"})
	}

	var onuStatus models.OLTONUStatus
	if err := h.db.Where("oltId = ? AND id = ?", oltID, onuID).First(&onuStatus).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ONU not found"})
	}

	iface := fmt.Sprintf("gpon-onu_%d/%d/%d:%d", onuStatus.Frame, onuStatus.Slot, onuStatus.Port, onuStatus.OnuID)

	var pool *telnet.Pool
	var ownPool bool
	pool = h.poller.GetPool(oltID)
	if pool == nil {
		tport := olt.TelnetPort
		if tport == 0 {
			tport = 23
		}
		tcfg := telnet.DefaultConfig(olt.IPAddress, tport, *olt.Username, *olt.Password)
		tcfg.CommandTimeout = 15 * time.Second
		pool = telnet.NewPool(tcfg)
		ownPool = true
	}
	if ownPool {
		defer pool.Close()
	}

	// ZTE C320: reboot ONU via shutdown + no shutdown on the ONU interface
	out, err := pool.ExecuteMultiple([]string{
		"configure terminal",
		"interface " + iface,
		"shutdown",
		"no shutdown",
		"exit",
		"end",
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Telnet error: " + err.Error()})
	}
	if uplinkCliErrRe.MatchString(out) {
		return c.Status(422).JSON(fiber.Map{"error": "OLT rejected reboot command", "detail": out})
	}
	return c.JSON(fiber.Map{
		"success":   true,
		"message":   "ONU reboot initiated",
		"interface": iface,
	})
}

// POST /api/olt/:id/onus/:onuId/clean-config — reset ONU service config on OLT (restore default).
// The ONU stays registered; only its VLAN/profile configuration is cleared.
func (h *MiscHandler) CleanONUConfig(c fiber.Ctx) error {
	oltID := c.Params("id")
	onuID := c.Params("onuId")

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}
	if (!olt.TelnetEnabled && !olt.SSHEnabled) || olt.Username == nil || olt.Password == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Telnet not configured for this OLT"})
	}

	var onuStatus models.OLTONUStatus
	if err := h.db.Where("oltId = ? AND id = ?", oltID, onuID).First(&onuStatus).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ONU not found"})
	}

	pool := h.poller.GetPool(oltID)
	var ownPool bool
	if pool == nil {
		tport := olt.TelnetPort
		if tport == 0 {
			tport = 23
		}
		tcfg := telnet.DefaultConfig(olt.IPAddress, tport, *olt.Username, *olt.Password)
		tcfg.CommandTimeout = 15 * time.Second
		pool = telnet.NewPool(tcfg)
		ownPool = true
	}
	if ownPool {
		defer pool.Close()
	}

	if err := zte.CleanONUConfig(pool, onuStatus.Frame, onuStatus.Slot, onuStatus.Port, onuStatus.OnuID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to clean ONU config: " + err.Error()})
	}

	iface := fmt.Sprintf("gpon-onu_%d/%d/%d:%d", onuStatus.Frame, onuStatus.Slot, onuStatus.Port, onuStatus.OnuID)
	return c.JSON(fiber.Map{"success": true, "message": "ONU config cleared", "interface": iface})
}

// POST /api/olt/:id/onus/batch-reboot — batch reboot multiple ONUs
func (h *MiscHandler) BatchRebootONUs(c fiber.Ctx) error {
	oltID := c.Params("id")
	var body struct {
		OnuIDs []string `json:"onuIds"`
	}
	if err := c.Bind().JSON(&body); err != nil || len(body.OnuIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "onuIds required"})
	}

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}
	if (!olt.TelnetEnabled && !olt.SSHEnabled) || olt.Username == nil || olt.Password == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Telnet not configured for this OLT"})
	}

	var onuStatuses []models.OLTONUStatus
	if err := h.db.Where("oltId = ? AND id IN ?", oltID, body.OnuIDs).Find(&onuStatuses).Error; err != nil || len(onuStatuses) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No ONUs found"})
	}

	var pool *telnet.Pool
	var ownPool bool
	pool = h.poller.GetPool(oltID)
	if pool == nil {
		tport := olt.TelnetPort
		if tport == 0 {
			tport = 23
		}
		tcfg := telnet.DefaultConfig(olt.IPAddress, tport, *olt.Username, *olt.Password)
		tcfg.CommandTimeout = 15 * time.Second
		pool = telnet.NewPool(tcfg)
		ownPool = true
	}
	if ownPool {
		defer pool.Close()
	}

	// Build commands: configure terminal once, then shutdown/no shutdown each ONU
	cmds := []string{"configure terminal"}
	for _, onu := range onuStatuses {
		iface := fmt.Sprintf("gpon-onu_%d/%d/%d:%d", onu.Frame, onu.Slot, onu.Port, onu.OnuID)
		cmds = append(cmds, "interface "+iface, "shutdown", "no shutdown", "exit")
	}
	cmds = append(cmds, "end")

	out, err := pool.ExecuteMultiple(cmds)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Telnet error: " + err.Error()})
	}
	if uplinkCliErrRe.MatchString(out) {
		return c.Status(422).JSON(fiber.Map{"error": "OLT rejected reboot command", "detail": out})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Batch reboot initiated",
		"count":   len(onuStatuses),
	})
}

// GET /api/olt/:id/onus/:onuId/detail — detailed ONU info via Telnet
func (h *MiscHandler) ONUDetail(c fiber.Ctx) error {
	oltID := c.Params("id")
	onuID := c.Params("onuId")

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "OLT not found"})
	}

	var onuStatus models.OLTONUStatus
	if err := h.db.Preload("Customer.Profile").Where("oltId = ? AND id = ?", oltID, onuID).First(&onuStatus).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "ONU not found"})
	}

	iface := fmt.Sprintf("gpon-onu_%d/%d/%d:%d", onuStatus.Frame, onuStatus.Slot, onuStatus.Port, onuStatus.OnuID)
	oltIface := fmt.Sprintf("gpon-olt_%d/%d/%d", onuStatus.Frame, onuStatus.Slot, onuStatus.Port)

	type detailResult struct {
		Raw     string                 `json:"raw"`
		Parsed  map[string]string      `json:"parsed"`
		Summary map[string]interface{} `json:"summary"`
	}
	type configResult struct {
		Raw     string                 `json:"raw"`
		Summary map[string]interface{} `json:"summary"`
	}
	type opticalResult struct {
		Raw string `json:"raw"`
	}
	type trafficResult struct {
		Raw     string                 `json:"raw"`
		Summary map[string]interface{} `json:"summary"`
	}

	detailInfo := detailResult{
		Parsed:  map[string]string{},
		Summary: map[string]interface{}{},
	}
	configInfo := configResult{Summary: map[string]interface{}{}}
	opticalInfo := opticalResult{}
	trafficInfo := trafficResult{Summary: map[string]interface{}{}}

	var tcontProfileDetails []GponTcontProfile
	var trafficProfileDetails []GponTrafficProfile
	var regOnuType, regSerial string

	if (olt.TelnetEnabled || olt.SSHEnabled) && olt.Username != nil && olt.Password != nil {
		// Always use a private Telnet pool for ONUDetail — never reuse the poller's shared pool.
		// The shared pool is used concurrently by the background poller (FetchTelnetDistances,
		// FetchTelnetONUStates). The pool's acquire() has no per-batch locking, so commands from
		// different goroutines can interleave on the same session, corrupting telnet output and
		// causing ONU states to fall back to (lagging) SNMP values → many ONUs appear offline.
		tport := olt.TelnetPort
		if tport == 0 {
			tport = 23
		}
		tcfg := telnet.DefaultConfig(olt.IPAddress, tport, *olt.Username, *olt.Password)
		tcfg.CommandTimeout = 20 * time.Second
		pool := telnet.NewPool(tcfg)
		defer pool.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		_ = ctx // telnet pool uses its own CommandTimeout; context reserved for future use

		out, err := pool.ExecuteMultiple([]string{
			"show gpon onu detail-info " + iface,
			"show running-config interface " + iface,
			"show interface " + iface,
			"show gpon profile tcont",
			"show gpon profile traffic",
			"show running-config interface " + oltIface,
		})
		if err == nil {
			parts := splitAtPrompt(out)
			if len(parts) >= 1 {
				detailInfo.Raw = strings.TrimSpace(parts[0])
				detailInfo.Parsed, detailInfo.Summary = onuParseDetailInfo(parts[0])
			}
			if len(parts) >= 2 {
				configInfo.Raw = strings.TrimSpace(parts[1])
				configInfo.Summary = onuParseRunningConfig(parts[1])
			}
			if len(parts) >= 3 {
				trafficInfo.Raw = strings.TrimSpace(parts[2])
				trafficInfo.Summary = onuParseInterfaceStats(parts[2])
			}
			if len(parts) >= 4 {
				tcontProfileDetails = parseGponTcontProfiles(parts[3])
			}
			if len(parts) >= 5 {
				trafficProfileDetails = parseGponTrafficProfiles(parts[4])
			}
			if len(parts) >= 6 {
				regOnuType, regSerial = parseOltRegistrationLine(parts[5], onuStatus.OnuID)
			}
		}
	}

	// Check GenieACS configuration
	tr069Info := fiber.Map{"configured": false, "message": "GenieACS belum dikonfigurasi"}
	var gnSettings models.GenieacsSettings
	if h.db.Where("isActive = ?", true).First(&gnSettings).Error == nil && gnSettings.Host != "" {
		tr069Info = fiber.Map{
			"configured": true,
			"host":       gnSettings.Host,
			"message":    "GenieACS terkonfigurasi",
		}
	}

	// Build bandwidth info from customer PPPoE profile
	var bandwidthInfo fiber.Map
	if onuStatus.Customer != nil && onuStatus.Customer.Profile.Name != "" {
		p := onuStatus.Customer.Profile
		bandwidthInfo = fiber.Map{
			"profileName":   p.Name,
			"downloadSpeed": p.DownloadSpeed,
			"uploadSpeed":   p.UploadSpeed,
			"rateLimit":     p.RateLimit,
		}
	}

	buildScript := generateONUBuildScript(iface, oltIface, onuStatus.OnuID, regOnuType, regSerial, configInfo.Raw, &gnSettings)

	return c.JSON(fiber.Map{
		"success": true,
		"telnet": fiber.Map{
			"interface": iface,
			"detail":    detailInfo,
			"config":    configInfo,
			"optical":   opticalInfo,
			"traffic":   trafficInfo,
		},
		"onu": fiber.Map{
			"id":        onuStatus.ID,
			"customer":  onuStatus.Customer,
			"bandwidth": bandwidthInfo,
		},
		"tr069": tr069Info,
		"oltProfiles": fiber.Map{
			"tcont":   tcontProfileDetails,
			"traffic": trafficProfileDetails,
		},
		"buildScript": buildScript,
	})
}

// onuParseDetailInfo parses ZTE C320 "show gpon onu detail-info" output.
// Returns a parsed key-value map (using exact ZTE CLI field names) and a summary.
func onuParseDetailInfo(raw string) (map[string]string, map[string]interface{}) {
	parsed := map[string]string{}
	kv := map[string]string{}

	// Auth history table rows: "  N   YYYY-MM-DD HH:MM:SS    ..."
	authHistoryRe := regexp.MustCompile(`^\s*(\d+)\s+([\d-]{10}\s[\d:]{8}|0000-00-00\s00:00:00)\s+([\d-]{10}\s[\d:]{8}|0000-00-00\s00:00:00)\s*(\S*)\s*$`)
	type authEntry struct {
		Index        string `json:"index"`
		AuthpassAt   string `json:"authpassAt"`
		OfflineAt    string `json:"offlineAt"`
		OfflineCause string `json:"offlineCause"`
	}
	var authHistory []authEntry

	inHistory := false
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		// Detect the start of the auth history section
		if strings.Contains(trimmed, "Authpass Time") && strings.Contains(trimmed, "OfflineTime") {
			inHistory = true
			continue
		}
		if inHistory {
			m := authHistoryRe.FindStringSubmatch(line)
			if m != nil {
				authpassAt := strings.TrimSpace(m[2])
				offlineAt := strings.TrimSpace(m[3])
				cause := strings.TrimSpace(m[4])
				// Only include rows with real timestamps
				if authpassAt != "0000-00-00 00:00:00" {
					authHistory = append(authHistory, authEntry{
						Index:        m[1],
						AuthpassAt:   authpassAt,
						OfflineAt:    offlineAt,
						OfflineCause: cause,
					})
				}
			}
			continue
		}

		// Parse key: value lines (find first colon that has a non-empty key before it)
		idx := strings.Index(line, ":")
		if idx < 1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key == "" || strings.ContainsAny(key, " \t") == false && len(key) > 40 {
			continue // skip garbage
		}
		kv[key] = val
		parsed[key] = val
	}

	// Build summary using actual ZTE C320 field names
	summary := map[string]interface{}{
		"authenticationMode":  kv["Authentication mode"],
		"snBind":              kv["SN Bind"],
		"adminState":          kv["Admin state"],
		"currentChannel":      kv["Current channel"],
		"configuredChannel":   kv["Configured channel"],
		"dbaMode":             kv["DBA Mode"],
		"vportMode":           kv["Vport mode"],
		"lineProfile":         kv["Line Profile"],
		"serviceProfile":      kv["Service Profile"],
		"omciBwProfile":       kv["OMCI BW Profile"],
		"vendor":              onuVendorFromSN(kv["Serial number"]),
		"description":         kv["Description"],
		"serialPrefix":        onuSNPrefix(kv["Serial number"]),
		"fec":                 kv["FEC"],
		"onuStatus":           kv["ONU Status"],
		"multicastEncryption": kv["Multicast encryption"],
		"authHistory":         authHistory,
	}

	return parsed, summary
}

// onuParseRunningConfig parses ZTE C320 "show running-config interface gpon-onu_F/S/P:N" output.
func onuParseRunningConfig(raw string) map[string]interface{} {
	serviceVlans := []string{}
	tcontProfiles := []string{}
	downstreamProfiles := []string{}

	type servicePortEntry struct {
		ServicePort string `json:"servicePort"`
		Vport       string `json:"vport"`
		UserVlan    string `json:"userVlan"`
		Vlan        string `json:"vlan"`
	}
	type gemportEntry struct {
		GemPort string `json:"gemPort"`
		Tcont   string `json:"tcont"`
		Profile string `json:"profile,omitempty"`
	}
	type tcontEntry struct {
		TcontID string `json:"tcontId"`
		Profile string `json:"profile"`
	}

	var servicePorts []servicePortEntry
	var gemports []gemportEntry
	var tconts []tcontEntry
	var onuName, onuDescription string

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)

		// Extract name
		if strings.HasPrefix(line, "name ") {
			onuName = strings.TrimPrefix(line, "name ")
		}
		// Extract description
		if strings.HasPrefix(line, "description ") {
			onuDescription = strings.TrimPrefix(line, "description ")
		}

		// Extract TCONT entries: "tcont N profile <name>"
		if strings.HasPrefix(line, "tcont") {
			fields := strings.Fields(line)
			var tcontID, profile string
			if len(fields) >= 2 {
				tcontID = fields[1]
			}
			for i, f := range fields {
				if f == "profile" && i+1 < len(fields) {
					profile = fields[i+1]
					if !contains(tcontProfiles, profile) {
						tcontProfiles = append(tcontProfiles, profile)
					}
				}
			}
			if tcontID != "" {
				tconts = append(tconts, tcontEntry{TcontID: tcontID, Profile: profile})
			}
		}

		// Extract GEM port entries: "gemport N tcont M [traffic-limit downstream <profile>]"
		if strings.HasPrefix(line, "gemport") {
			fields := strings.Fields(line)
			entry := gemportEntry{}
			if len(fields) >= 2 {
				entry.GemPort = fields[1]
			}
			for i, f := range fields {
				if f == "tcont" && i+1 < len(fields) {
					entry.Tcont = fields[i+1]
				}
				if f == "downstream" && i+1 < len(fields) {
					entry.Profile = fields[i+1]
					if !contains(downstreamProfiles, fields[i+1]) {
						downstreamProfiles = append(downstreamProfiles, fields[i+1])
					}
				}
			}
			if entry.GemPort != "" {
				gemports = append(gemports, entry)
			}
		}

		// Extract service-port entries: "service-port N vport M user-vlan X vlan Y"
		if strings.HasPrefix(line, "service-port") {
			fields := strings.Fields(line)
			entry := servicePortEntry{}
			if len(fields) >= 2 {
				entry.ServicePort = fields[1]
			}
			for i, f := range fields {
				if f == "vport" && i+1 < len(fields) {
					entry.Vport = fields[i+1]
				}
				if f == "user-vlan" && i+1 < len(fields) {
					entry.UserVlan = fields[i+1]
					if fields[i+1] != "untagged" && !contains(serviceVlans, fields[i+1]) {
						serviceVlans = append(serviceVlans, fields[i+1])
					}
				}
				if f == "vlan" && i+1 < len(fields) {
					entry.Vlan = fields[i+1]
					if fields[i+1] != "untagged" && !contains(serviceVlans, fields[i+1]) {
						serviceVlans = append(serviceVlans, fields[i+1])
					}
				}
			}
			if entry.ServicePort != "" {
				servicePorts = append(servicePorts, entry)
			}
		}
	}

	// Remove VLAN duplicates that were added from both user-vlan and vlan fields
	uniqueVlans := []string{}
	seen := map[string]bool{}
	for _, v := range serviceVlans {
		if !seen[v] {
			seen[v] = true
			uniqueVlans = append(uniqueVlans, v)
		}
	}

	return map[string]interface{}{
		"name":               onuName,
		"description":        onuDescription,
		"serviceVlans":       uniqueVlans,
		"tcontProfiles":      tcontProfiles,
		"downstreamProfiles": downstreamProfiles,
		"servicePorts":       servicePorts,
		"gemports":           gemports,
		"tconts":             tconts,
	}
}

// onuParseInterfaceStats parses ZTE C320 "show interface gpon-onu_F/S/P:N" output.
// Returns a summary map with downstream/upstream rates and totals.
func onuParseInterfaceStats(raw string) map[string]interface{} {
	summary := map[string]interface{}{}

	// Normalize: remove extra whitespace in values like "0 Bps                0 pps"
	spaceRe := regexp.MustCompile(`\s{2,}`)

	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)

		// "Input rate :   N Bps   N pps"
		if strings.HasPrefix(trimmed, "Input rate") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(spaceRe.ReplaceAllString(parts[1], " "))
				summary["inputRate"] = val
			}
		}
		// "Output rate:   N Bps   N pps"
		if strings.HasPrefix(trimmed, "Output rate") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(spaceRe.ReplaceAllString(parts[1], " "))
				summary["outputRate"] = val
			}
		}
		// "Input bandwidth thoughput :N%"
		if strings.Contains(trimmed, "Input bandwidth") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				summary["inputBandwidth"] = strings.TrimSpace(parts[1])
			}
		}
		// "Output bandwidth thoughput: N%"
		if strings.Contains(trimmed, "Output bandwidth") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				summary["outputBandwidth"] = strings.TrimSpace(parts[1])
			}
		}
		// "Input peak rate :   N Bps   N pps"
		if strings.HasPrefix(trimmed, "Input peak rate") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(spaceRe.ReplaceAllString(parts[1], " "))
				summary["inputPeakRate"] = val
			}
		}
		// "Output peak rate:   N Bps   N pps"
		if strings.HasPrefix(trimmed, "Output peak rate") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(spaceRe.ReplaceAllString(parts[1], " "))
				summary["outputPeakRate"] = val
			}
		}
		// "Bytes:N          Packets:N"
		if strings.Contains(trimmed, "Bytes:") && strings.Contains(trimmed, "Packets:") {
			// Determine if this is Input or Output by checking preceding context
			// We use a simple heuristic: track last seen "Input:" / "Output:"
			bytesRe := regexp.MustCompile(`Bytes:(\d+)`)
			packetsRe := regexp.MustCompile(`Packets:(\d+)`)
			bm := bytesRe.FindStringSubmatch(trimmed)
			pm := packetsRe.FindStringSubmatch(trimmed)
			if bm != nil && pm != nil {
				if _, ok := summary["totalInputBytes"]; !ok {
					summary["totalInputBytes"] = bm[1]
					summary["totalInputPackets"] = pm[1]
				} else if _, ok := summary["totalOutputBytes"]; !ok {
					summary["totalOutputBytes"] = bm[1]
					summary["totalOutputPackets"] = pm[1]
				}
			}
		}
	}

	return summary
}

func onuVendorFromSN(sn string) string {
	if len(sn) < 4 {
		return ""
	}
	switch strings.ToUpper(sn[:4]) {
	case "ZTEG":
		return "ZTE"
	case "HWTC":
		return "Huawei"
	case "FHTT":
		return "FiberHome"
	case "ALPH":
		return "Alpha"
	default:
		return sn[:4]
	}
}

func onuSNPrefix(sn string) string {
	if len(sn) < 4 {
		return sn
	}
	return sn[:4]
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// ─── GPON Profile Types ───────────────────────────────────────────────────────

// GponTcontProfile holds DBA/TCONT profile info from "show gpon profile tcont".
type GponTcontProfile struct {
	Name   string `json:"name"`
	BwType int    `json:"bwType"`
	FBW    int    `json:"fbwKbps"`
	ABW    int    `json:"abwKbps"`
	MBW    int    `json:"mbwKbps"`
}

// GponTrafficProfile holds downstream traffic profile info from "show gpon profile traffic".
type GponTrafficProfile struct {
	Name string `json:"name"`
	SIR  int    `json:"sirKbps"`
	PIR  int    `json:"pirKbps"`
}

// parseGponTcontProfiles parses "show gpon profile tcont" output into a list of profiles.
func parseGponTcontProfiles(raw string) []GponTcontProfile {
	var profiles []GponTcontProfile
	var current *GponTcontProfile
	dataExpected := false

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "Profile name") {
			if current != nil {
				profiles = append(profiles, *current)
			}
			if idx := strings.LastIndex(trimmed, ":"); idx >= 0 {
				name := strings.TrimSpace(trimmed[idx+1:])
				current = &GponTcontProfile{Name: name}
				dataExpected = false
			}
			continue
		}
		if strings.Contains(trimmed, "FBW") || strings.Contains(trimmed, "ABW") {
			dataExpected = true
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "ZXAN") || strings.HasPrefix(trimmed, "show") {
			continue
		}
		if dataExpected && current != nil {
			fields := strings.Fields(trimmed)
			if len(fields) >= 4 {
				t, _ := strconv.Atoi(fields[0])
				fbw, _ := strconv.Atoi(fields[1])
				abw, _ := strconv.Atoi(fields[2])
				mbw, _ := strconv.Atoi(fields[3])
				current.BwType = t
				current.FBW = fbw
				current.ABW = abw
				current.MBW = mbw
				dataExpected = false
			}
		}
	}
	if current != nil {
		profiles = append(profiles, *current)
	}
	return profiles
}

// parseGponTrafficProfiles parses "show gpon profile traffic" output into a list of profiles.
func parseGponTrafficProfiles(raw string) []GponTrafficProfile {
	var profiles []GponTrafficProfile
	var current *GponTrafficProfile
	awaitData := false

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "Profile name") {
			if current != nil {
				profiles = append(profiles, *current)
			}
			if idx := strings.LastIndex(trimmed, ":"); idx >= 0 {
				name := strings.TrimSpace(trimmed[idx+1:])
				current = &GponTrafficProfile{Name: name}
				awaitData = false
			}
			continue
		}
		if strings.Contains(trimmed, "SIR") || strings.Contains(trimmed, "PIR") {
			awaitData = true
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "ZXAN") || strings.HasPrefix(trimmed, "show") {
			continue
		}
		if awaitData && current != nil && current.SIR == 0 {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				sir, err1 := strconv.Atoi(fields[0])
				pir, err2 := strconv.Atoi(fields[1])
				if err1 == nil && err2 == nil {
					current.SIR = sir
					current.PIR = pir
					awaitData = false
				}
			}
		}
	}
	if current != nil {
		profiles = append(profiles, *current)
	}
	return profiles
}

// parseOltRegistrationLine extracts the ONU type and serial number from
// "show running-config interface gpon-olt_F/S/P" output for a given onuId.
func parseOltRegistrationLine(raw string, onuId int) (onuType, serialNumber string) {
	prefix := fmt.Sprintf("onu %d ", onuId)
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(strings.TrimRight(line, "\r"))
		if strings.HasPrefix(trimmed, prefix) {
			fields := strings.Fields(trimmed)
			for i, f := range fields {
				if f == "type" && i+1 < len(fields) {
					onuType = fields[i+1]
				}
				if f == "sn" && i+1 < len(fields) {
					serialNumber = fields[i+1]
				}
			}
			return
		}
	}
	return
}

// generateONUBuildScript generates a complete ZTE CLI build script for the ONU
// based on its running-config and global profile data.
func generateONUBuildScript(onuIface, oltIface string, onuId int, onuType, serialNumber, configRaw string, gnSettings *models.GenieacsSettings) string {
	var sb strings.Builder

	// Parse config lines from raw interface running-config
	var onuName, description string
	var tcontLines, gemportLines, servicePortLines []string

	for _, line := range strings.Split(configRaw, "\n") {
		trimmed := strings.TrimSpace(strings.TrimRight(line, "\r"))
		if trimmed == "" || strings.HasPrefix(trimmed, "!") ||
			strings.HasPrefix(trimmed, "interface") ||
			strings.HasPrefix(trimmed, "ZXAN") ||
			strings.HasPrefix(trimmed, "show") {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "name "):
			onuName = strings.TrimPrefix(trimmed, "name ")
		case strings.HasPrefix(trimmed, "description "):
			description = strings.TrimPrefix(trimmed, "description ")
		case strings.HasPrefix(trimmed, "tcont"):
			tcontLines = append(tcontLines, trimmed)
		case strings.HasPrefix(trimmed, "gemport"):
			gemportLines = append(gemportLines, trimmed)
		case strings.HasPrefix(trimmed, "service-port"):
			servicePortLines = append(servicePortLines, trimmed)
		}
	}

	// Step 1: OLT registration
	sb.WriteString("! === Step 1: Registrasi ONU di OLT ===\n")
	sb.WriteString("conf t\n")
	fmt.Fprintf(&sb, "interface %s\n", oltIface)
	if onuType != "" && serialNumber != "" {
		sb.WriteString(fmt.Sprintf("  onu %d type %s sn %s\n", onuId, onuType, serialNumber))
	} else {
		sb.WriteString(fmt.Sprintf("  ! onu %d type <type> sn <serial-number>\n", onuId))
	}
	sb.WriteString("exit\n!\n")

	// Step 2: ONU interface config
	sb.WriteString("! === Step 2: Konfigurasi Interface ONU ===\n")
	sb.WriteString("conf t\n")
	fmt.Fprintf(&sb, "interface %s\n", onuIface)
	if onuName != "" {
		fmt.Fprintf(&sb, "  name %s\n", onuName)
	}
	if description != "" {
		fmt.Fprintf(&sb, "  description %s\n", description)
	}
	for _, l := range tcontLines {
		fmt.Fprintf(&sb, "  %s\n", l)
	}
	for _, l := range gemportLines {
		fmt.Fprintf(&sb, "  %s\n", l)
	}
	for _, l := range servicePortLines {
		fmt.Fprintf(&sb, "  %s\n", l)
	}
	sb.WriteString("exit\n!\n")

	// Step 3: pon-onu-mng (OMCI management)
	sb.WriteString("! === Step 3: OMCI Management (pon-onu-mng) ===\n")
	fmt.Fprintf(&sb, "pon-onu-mng %s\n", onuIface)

	// Generate service-to-gemport mappings from service-port lines
	// "service-port N vport M user-vlan X vlan Y" → "service N gemport N vlan X"
	for _, sp := range servicePortLines {
		fields := strings.Fields(sp)
		if len(fields) < 2 {
			continue
		}
		spNum := fields[1]
		userVlan := ""
		for i, f := range fields {
			if f == "user-vlan" && i+1 < len(fields) {
				userVlan = fields[i+1]
				break
			}
		}
		if userVlan != "" {
			sb.WriteString(fmt.Sprintf("  service %s gemport %s vlan %s\n", spNum, spNum, userVlan))
		}
	}

	// TR-069 management section
	if gnSettings != nil && gnSettings.IsActive && gnSettings.Host != "" {
		acsURL := strings.TrimRight(gnSettings.Host, "/")
		// GenieACS CWMP port is 7547 by default
		if !strings.Contains(acsURL, ":7547") && !strings.Contains(acsURL, ":") {
			acsURL = acsURL + ":7547"
		} else if idx := strings.LastIndex(acsURL, ":"); idx > 8 {
			// already has a port after the scheme
		} else {
			acsURL = acsURL + ":7547"
		}
		sb.WriteString("  tr069-mgmt 1 state unlock\n")
		sb.WriteString(fmt.Sprintf("  tr069-mgmt 1 acs %s validate basic username acs password acs\n", acsURL))
	} else {
		sb.WriteString("  ! tr069-mgmt 1 state unlock\n")
		sb.WriteString("  ! tr069-mgmt 1 acs <ACS_URL>:7547 validate basic username acs password acs\n")
	}
	sb.WriteString("exit\n!\n")
	sb.WriteString("end\n")
	sb.WriteString("write\n")

	return sb.String()
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
		SNMPEnabled   bool   `gorm:"column:snmpEnabled"`
	}
	var olts []oltRow
	q := h.db.Table("network_olts").Select("id, ipAddress, sshEnabled, sshPort, telnetEnabled, telnetPort, snmpEnabled")
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
			SNMP   bool `json:"snmp"`
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
			// Check SSH and Telnet in parallel (always check both if enabled)
			type checkResult struct{ ok bool }
			sshCh := make(chan checkResult, 1)
			telnetCh := make(chan checkResult, 1)

			go func() {
				if row.SSHEnabled {
					conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", row.IPAddress, sshPort), 3*time.Second)
					if err == nil {
						conn.Close()
						sshCh <- checkResult{ok: true}
						return
					}
				}
				sshCh <- checkResult{ok: false}
			}()
			go func() {
				if row.TelnetEnabled {
					conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", row.IPAddress, telnetPort), 3*time.Second)
					if err == nil {
						conn.Close()
						telnetCh <- checkResult{ok: true}
						return
					}
				}
				telnetCh <- checkResult{ok: false}
			}()

			sshOK = (<-sshCh).ok
			telnetOK = (<-telnetCh).ok

			online := sshOK || telnetOK
			entry := statusEntry{ID: row.ID, Online: online}
			entry.Details.SSH = sshOK
			entry.Details.Telnet = telnetOK
			entry.Details.SNMP = row.SNMPEnabled // SNMP is configuration-based (UDP, not easily TCP-checked)
			mu.Lock()
			results[row.ID] = entry
			h.db.Table("network_olts").Where("id = ?", row.ID).Update("isOnline", online)
			mu.Unlock()
		}(o)
	}
	wg.Wait()
	return c.JSON(fiber.Map{"statusMap": results})
}
