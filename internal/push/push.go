package push

// push.go — Port dari src/server/services/push-notification.service.ts
//
// Web Push notification service using VAPID authentication.
// Supports sending to customers, agents, technicians, and admins.
// Handles subscription management, broadcast sending, and expired subscription cleanup.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	wp "github.com/SherClockHolmes/webpush-go"
)

// Payload represents a push notification payload.
type Payload struct {
	Title             string            `json:"title"`
	Body              string            `json:"body"`
	URL               string            `json:"url,omitempty"`
	Tag               string            `json:"tag,omitempty"`
	Icon              string            `json:"icon,omitempty"`
	Badge             string            `json:"badge,omitempty"`
	Image             string            `json:"image,omitempty"`
	RequireInteraction bool             `json:"requireInteraction,omitempty"`
	Data              map[string]any    `json:"data,omitempty"`
}

// SendResult holds the result of a push send operation.
type SendResult struct {
	Sent   int `json:"sent"`
	Failed int `json:"failed"`
	Total  int `json:"total"`
}

// BroadcastInput holds parameters for a broadcast send.
type BroadcastInput struct {
	Title         string         `json:"title"`
	Body          string         `json:"body"`
	Type          string         `json:"type"`
	RecipientRole string         `json:"recipientRole"` // customer, agent, technician, all
	TargetType    string         `json:"targetType"`    // all, active, expired, area, selected
	TargetIDs     []string       `json:"targetIds"`
	SentBy        *string        `json:"sentBy"`
	Data          map[string]any `json:"data"`
}

type storedSub struct {
	ID       string
	Endpoint string
	P256dh   string
	Auth     string
}

var (
	vapidOnce     sync.Once
	vapidPubKey   string
	vapidPrivKey  string
	vapidSubject  string
)

func ensureVapid() error {
	vapidOnce.Do(func() {
		vapidPubKey = os.Getenv("VAPID_PUBLIC_KEY")
		vapidPrivKey = os.Getenv("VAPID_PRIVATE_KEY")
		vapidSubject = fmt.Sprintf("mailto:%s", getEnvOrDefault("VAPID_CONTACT_EMAIL", "admin@example.com"))
	})
	if vapidPubKey == "" || vapidPrivKey == "" {
		return fmt.Errorf("VAPID keys not configured")
	}
	return nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// NormalizePushURL normalizes a URL for the customer portal.
func NormalizePushURL(rawURL string) string {
	if rawURL == "" {
		return "/customer"
	}
	url := strings.TrimSpace(rawURL)
	if url == "" {
		return "/customer"
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	if strings.HasPrefix(url, "/customer") {
		return url
	}
	if strings.HasPrefix(url, "/(tabs)/") {
		return strings.Replace(url, "/(tabs)", "/customer", 1)
	}
	if url == "/(tabs)" {
		return "/customer"
	}
	mappedRoutes := map[string]string{
		"invoices":       "/customer/invoices",
		"profile":        "/customer/profile",
		"tickets":        "/customer/tickets",
		"history":        "/customer/history",
		"referral":       "/customer/referral",
		"upgrade":        "/customer/upgrade",
		"notifications":  "/customer",
		"home":           "/customer",
	}
	normalizedKey := strings.TrimLeft(url, "/")
	if mapped, ok := mappedRoutes[normalizedKey]; ok {
		return mapped
	}
	if strings.HasPrefix(url, "/") {
		return url
	}
	return "/customer/" + normalizedKey
}

func buildPayload(p *Payload) []byte {
	url := NormalizePushURL(p.URL)
	if p.Icon == "" {
		p.Icon = "/pwa/icon-192.svg"
	}
	if p.Badge == "" {
		p.Badge = "/pwa/badge.svg"
	}
	if p.Tag == "" {
		p.Tag = "salfanet-notification"
	}
	data := p.Data
	if data == nil {
		data = map[string]any{}
	}
	data["url"] = url

	payload := struct {
		Title             string         `json:"title"`
		Body              string         `json:"body"`
		Icon              string         `json:"icon"`
		Badge             string         `json:"badge"`
		Image             string         `json:"image"`
		Tag               string         `json:"tag"`
		RequireInteraction bool           `json:"requireInteraction"`
		Data              map[string]any `json:"data"`
	}{
		Title:              p.Title,
		Body:               p.Body,
		Icon:               p.Icon,
		Badge:              p.Badge,
		Image:              p.Image,
		Tag:                p.Tag,
		RequireInteraction: p.RequireInteraction,
		Data:               data,
	}
	b, _ := json.Marshal(payload)
	return b
}

func isExpiredError(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	return resp.StatusCode == 404 || resp.StatusCode == 410
}

// sendToSubs sends a push notification to a list of subscriptions.
// role determines which DB table to update for deactivation/lastUsed.
func sendToSubs(db *gorm.DB, subs []storedSub, p *Payload, role string) SendResult {
	if err := ensureVapid(); err != nil {
		log.Error().Err(err).Msg("[WebPush] VAPID not configured")
		return SendResult{Total: len(subs)}
	}

	if len(subs) == 0 {
		return SendResult{}
	}

	payload := buildPayload(p)
	sent := 0
	failed := 0
	var usedIDs []string
	var expiredIDs []string

	for _, sub := range subs {
		s := &wp.Subscription{
			Endpoint: sub.Endpoint,
			Keys: wp.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}
		resp, err := wp.SendNotification(payload, s, &wp.Options{
			Subscriber:      vapidSubject,
			VAPIDPublicKey:  vapidPubKey,
			VAPIDPrivateKey: vapidPrivKey,
			TTL:             60,
		})
		if err != nil {
			failed++
			log.Error().Err(err).Str("endpoint", sub.Endpoint).Msg("[WebPush] send failed")
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			sent++
			usedIDs = append(usedIDs, sub.ID)
		} else {
			failed++
			if isExpiredError(resp) {
				expiredIDs = append(expiredIDs, sub.ID)
			}
			log.Error().Int("status", resp.StatusCode).Str("endpoint", sub.Endpoint).Msg("[WebPush] send error")
		}
	}

	// Update DB: mark used and deactivate expired
	now := time.Now()
	if len(usedIDs) > 0 {
		markSubsUsed(db, role, usedIDs, now)
	}
	if len(expiredIDs) > 0 {
		deactivateSubs(db, role, expiredIDs)
	}

	return SendResult{Sent: sent, Failed: failed, Total: len(subs)}
}

func markSubsUsed(db *gorm.DB, role string, ids []string, now time.Time) {
	switch role {
	case "agent":
		db.Model(&models.AgentPushSubscription{}).Where("id IN ?", ids).Update("lastUsedAt", now)
	case "technician":
		db.Model(&models.TechnicianPushSubscription{}).Where("id IN ?", ids).Update("lastUsedAt", now)
	case "admin":
		db.Model(&models.AdminPushSubscription{}).Where("id IN ?", ids).Update("lastUsedAt", now)
	default:
		db.Model(&models.PushSubscription{}).Where("id IN ?", ids).Update("lastUsedAt", now)
	}
}

func deactivateSubs(db *gorm.DB, role string, ids []string) {
	switch role {
	case "agent":
		db.Model(&models.AgentPushSubscription{}).Where("id IN ?", ids).Update("isActive", false)
	case "technician":
		db.Model(&models.TechnicianPushSubscription{}).Where("id IN ?", ids).Update("isActive", false)
	case "admin":
		db.Model(&models.AdminPushSubscription{}).Where("id IN ?", ids).Update("isActive", false)
	default:
		db.Model(&models.PushSubscription{}).Where("id IN ?", ids).Update("isActive", false)
	}
}

// ─── Per-role targeted sends ──────────────────────────────────────────────────

// SendToUser sends a push notification to a specific customer.
func SendToUser(db *gorm.DB, userID string, p *Payload) SendResult {
	var subs []models.PushSubscription
	db.Where("userId = ? AND isActive = true", userID).Find(&subs)
	return sendToSubs(db, toStoredSubs(subs), p, "customer")
}

// SendToUsers sends a push notification to multiple customers.
func SendToUsers(db *gorm.DB, userIDs []string, p *Payload) SendResult {
	if len(userIDs) == 0 {
		return SendResult{}
	}
	var subs []models.PushSubscription
	db.Where("userId IN ? AND isActive = true", userIDs).Find(&subs)
	return sendToSubs(db, toStoredSubs(subs), p, "customer")
}

// SendToAgent sends a push notification to a specific agent.
func SendToAgent(db *gorm.DB, agentID string, p *Payload) SendResult {
	var subs []models.AgentPushSubscription
	db.Where("agentId = ? AND isActive = true", agentID).Find(&subs)
	return sendToSubs(db, toAgentSubs(subs), p, "agent")
}

// SendToTechnician sends a push notification to a specific technician.
func SendToTechnician(db *gorm.DB, technicianID string, p *Payload) SendResult {
	var subs []models.TechnicianPushSubscription
	db.Where("technicianId = ? AND isActive = true", technicianID).Find(&subs)
	return sendToSubs(db, toTechSubs(subs), p, "technician")
}

// SendToAllTechnicians sends a push notification to all active technicians.
func SendToAllTechnicians(db *gorm.DB, p *Payload) SendResult {
	var subs []models.TechnicianPushSubscription
	db.Where("isActive = true").Find(&subs)
	return sendToSubs(db, toTechSubs(subs), p, "technician")
}

// ─── Broadcast ────────────────────────────────────────────────────────────────

// SendBroadcast sends a push notification to multiple roles based on input.
func SendBroadcast(db *gorm.DB, input *BroadcastInput) (*models.PushBroadcast, SendResult) {
	if err := ensureVapid(); err != nil {
		log.Error().Err(err).Msg("[WebPush] VAPID not configured")
		return nil, SendResult{}
	}

	recipientRole := input.RecipientRole
	if recipientRole == "" {
		recipientRole = "customer"
	}

	p := &Payload{
		Title: input.Title,
		Body:  input.Body,
		Tag:   input.Type,
		Data:  input.Data,
	}
	if link, ok := input.Data["link"].(string); ok {
		p.URL = link
	}

	var totalSent, totalFailed, totalCount int

	if recipientRole == "agent" || recipientRole == "all" {
		var subs []models.AgentPushSubscription
		db.Where("isActive = true").Find(&subs)
		r := sendToSubs(db, toAgentSubs(subs), p, "agent")
		totalSent += r.Sent
		totalFailed += r.Failed
		totalCount += r.Total
	}

	if recipientRole == "technician" || recipientRole == "all" {
		var subs []models.TechnicianPushSubscription
		db.Where("isActive = true").Find(&subs)
		r := sendToSubs(db, toTechSubs(subs), p, "technician")
		totalSent += r.Sent
		totalFailed += r.Failed
		totalCount += r.Total

		var adminSubs []models.AdminPushSubscription
		db.Where("isActive = true").Find(&adminSubs)
		r = sendToSubs(db, toAdminSubs(adminSubs), p, "admin")
		totalSent += r.Sent
		totalFailed += r.Failed
		totalCount += r.Total
	}

	if recipientRole == "customer" || recipientRole == "all" {
		custSubs := getBroadcastTargets(db, input.TargetType, input.TargetIDs)
		r := sendToSubs(db, custSubs, p, "customer")
		totalSent += r.Sent
		totalFailed += r.Failed
		totalCount += r.Total
	}

	result := SendResult{Sent: totalSent, Failed: totalFailed, Total: totalCount}
	if totalCount == 0 {
		return nil, result
	}

	// Record broadcast
	targetIDsStr := ""
	if len(input.TargetIDs) > 0 {
		b, _ := json.Marshal(input.TargetIDs)
		targetIDsStr = string(b)
	}
	dataStr := ""
	if input.Data != nil {
		b, _ := json.Marshal(input.Data)
		dataStr = string(b)
	}
	broadcast := models.PushBroadcast{
		ID:          uuid.NewString(),
		Title:       input.Title,
		Body:        input.Body,
		Type:        input.Type,
		TargetType:  fmt.Sprintf("%s:%s", recipientRole, input.TargetType),
		TargetIDs:   &targetIDsStr,
		SentCount:   totalSent,
		FailedCount: totalFailed,
		SentBy:      input.SentBy,
		Data:        &dataStr,
		CreatedAt:   time.Now(),
	}
	db.Create(&broadcast)

	return &broadcast, result
}

func getBroadcastTargets(db *gorm.DB, targetType string, targetIDs []string) []storedSub {
	query := db.Model(&models.PushSubscription{}).Where("isActive = true")
	if targetType == "area" && len(targetIDs) > 0 {
		// Join with pppoe_users to filter by area
		query = query.Joins("JOIN pppoe_users ON pppoe_users.id = push_subscriptions.userId").
			Where("pppoe_users.areaId IN ?", targetIDs)
	} else if targetType == "selected" && len(targetIDs) > 0 {
		query = query.Where("userId IN ?", targetIDs)
	} else if targetType == "active" {
		query = query.Joins("JOIN pppoe_users ON pppoe_users.id = push_subscriptions.userId").
			Where("pppoe_users.status = 'active'")
	} else if targetType == "expired" {
		query = query.Joins("JOIN pppoe_users ON pppoe_users.id = push_subscriptions.userId").
			Where("pppoe_users.status = 'expired'")
	}

	var subs []models.PushSubscription
	query.Find(&subs)
	return toStoredSubs(subs)
}

// ─── Dashboard stats ──────────────────────────────────────────────────────────

// DashboardStats holds push notification dashboard statistics.
type DashboardStats struct {
	TotalUsers          int64   `json:"totalUsers"`
	UsersWithTokens     int64   `json:"usersWithTokens"`
	TotalSubscriptions  int64   `json:"totalSubscriptions"`
	Areas               []Area  `json:"areas"`
	TotalBroadcasts     int64   `json:"totalBroadcasts"`
	AgentSubscribers    int64   `json:"agentSubscribers"`
	TechnicianSubscribers int64 `json:"technicianSubscribers"`
	AdminSubscribers    int64   `json:"adminSubscribers"`
}

type Area struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetDashboardStats returns push notification dashboard statistics.
func GetDashboardStats(db *gorm.DB) DashboardStats {
	var stats DashboardStats

	db.Model(&models.PppoeUser{}).Where("status = 'active'").Count(&stats.TotalUsers)

	type areaRow struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var areas []areaRow
	db.Raw("SELECT id, name FROM areas ORDER BY name").Scan(&areas)
	stats.Areas = make([]Area, len(areas))
	for i, a := range areas {
		stats.Areas[i] = Area{ID: a.ID, Name: a.Name}
	}

	db.Model(&models.PushBroadcast{}).Count(&stats.TotalBroadcasts)
	db.Model(&models.PushSubscription{}).Where("isActive = true").Count(&stats.TotalSubscriptions)

	// Distinct users with active subscriptions
	db.Model(&models.PushSubscription{}).Where("isActive = true").Distinct("userId").Count(&stats.UsersWithTokens)

	db.Model(&models.AgentPushSubscription{}).Where("isActive = true").Distinct("agentId").Count(&stats.AgentSubscribers)
	db.Model(&models.TechnicianPushSubscription{}).Where("isActive = true").Distinct("technicianId").Count(&stats.TechnicianSubscribers)
	db.Model(&models.AdminPushSubscription{}).Where("isActive = true").Distinct("adminId").Count(&stats.AdminSubscribers)

	return stats
}

// GetPublicVapidKey returns the VAPID public key.
func GetPublicVapidKey() string {
	return os.Getenv("VAPID_PUBLIC_KEY")
}

// ─── Subscription converters ──────────────────────────────────────────────────

func toStoredSubs(subs []models.PushSubscription) []storedSub {
	result := make([]storedSub, len(subs))
	for i, s := range subs {
		result[i] = storedSub{ID: s.ID, Endpoint: s.Endpoint, P256dh: s.P256dh, Auth: s.Auth}
	}
	return result
}

func toAgentSubs(subs []models.AgentPushSubscription) []storedSub {
	result := make([]storedSub, len(subs))
	for i, s := range subs {
		result[i] = storedSub{ID: s.ID, Endpoint: s.Endpoint, P256dh: s.P256dh, Auth: s.Auth}
	}
	return result
}

func toTechSubs(subs []models.TechnicianPushSubscription) []storedSub {
	result := make([]storedSub, len(subs))
	for i, s := range subs {
		result[i] = storedSub{ID: s.ID, Endpoint: s.Endpoint, P256dh: s.P256dh, Auth: s.Auth}
	}
	return result
}

func toAdminSubs(subs []models.AdminPushSubscription) []storedSub {
	result := make([]storedSub, len(subs))
	for i, s := range subs {
		result[i] = storedSub{ID: s.ID, Endpoint: s.Endpoint, P256dh: s.P256dh, Auth: s.Auth}
	}
	return result
}
