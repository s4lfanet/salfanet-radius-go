package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// RoadmapHandler handles remaining roadmap items:
// 1.3 Edit Payment Method, 1.7 External API Keys,
// 2.3 Profile Overrides, 2.4 Waiting List, 2.5 ONT Removal Tasks.
type RoadmapHandler struct {
	db *gorm.DB
}

func NewRoadmapHandler(db *gorm.DB) *RoadmapHandler {
	return &RoadmapHandler{db: db}
}

// ─── 1.3 Edit Payment Method ─────────────────────────────────────────────────

// PUT /api/roadmap/payments/:id/method — edit payment method post-lunas
func (h *RoadmapHandler) EditPaymentMethod(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Method string `json:"method"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	validMethods := map[string]bool{
		"cash": true, "transfer": true, "midtrans": true,
		"xendit": true, "tripay": true, "qris": true, "duitku": true,
	}
	if !validMethods[body.Method] {
		return c.Status(400).JSON(fiber.Map{"error": "invalid method, must be one of: cash, transfer, midtrans, xendit, tripay, qris, duitku"})
	}

	var payment models.Payment
	if err := h.db.First(&payment, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "payment not found"})
	}

	oldMethod := payment.Method
	if err := h.db.Model(&payment).Updates(map[string]interface{}{
		"method":                  body.Method,
		"paymentMethodEditCount":  payment.PaymentMethodEditCount + 1,
	}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"message":  fmt.Sprintf("Payment method changed from %s to %s", oldMethod, body.Method),
		"editCount": payment.PaymentMethodEditCount + 1,
	})
}

// ─── 1.7 External API Keys ───────────────────────────────────────────────────

// GET /api/roadmap/api-keys — list API keys
func (h *RoadmapHandler) ListAPIKeys(c fiber.Ctx) error {
	var keys []models.APIKey
	h.db.Find(&keys)
	return c.JSON(fiber.Map{"data": keys})
}

// POST /api/roadmap/api-keys — generate new API key
func (h *RoadmapHandler) CreateAPIKey(c fiber.Ctx) error {
	var body struct {
		Label string `json:"label"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Label == "" {
		return c.Status(400).JSON(fiber.Map{"error": "label required"})
	}

	// Generate random API key
	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate key"})
	}
	apiKey := "sk_" + hex.EncodeToString(rawKey)
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	key := models.APIKey{
		ID:      uuid.New().String(),
		KeyHash: keyHash,
		Label:   body.Label,
		IsActive: true,
	}

	if err := h.db.Create(&key).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Return the raw key only once
	return c.JSON(fiber.Map{
		"success": true,
		"apiKey":  apiKey,
		"key":     key,
		"message": "Save this API key — it will not be shown again",
	})
}

// DELETE /api/roadmap/api-keys/:id — revoke API key
func (h *RoadmapHandler) RevokeAPIKey(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.db.Model(&models.APIKey{}).Where("id = ?", id).Update("isActive", false).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "API key revoked"})
}

// GET /api/external/users/status — public API endpoint (X-API-Key header)
func (h *RoadmapHandler) ExternalUserStatus(c fiber.Ctx) error {
	apiKey := c.Get("X-API-Key")
	if apiKey == "" {
		return c.Status(401).JSON(fiber.Map{"error": "X-API-Key header required"})
	}

	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	var key models.APIKey
	if err := h.db.Where("keyHash = ? AND isActive = true", keyHash).First(&key).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid or revoked API key"})
	}

	// Update last used
	now := time.Now()
	h.db.Model(&key).Update("lastUsedAt", &now)

	// Return user statuses
	var users []struct {
		Username     string  `json:"username"`
		Name         *string `json:"name"`
		Status       string  `json:"status"`
		ProfileName  *string `json:"profileName"`
		RouterName   *string `json:"routerName"`
		CustomerID   *string `json:"customerId"`
	}
	h.db.Table("pppoe_users").
		Select("pppoe_users.username, pppoe_users.name, pppoe_users.status, pppoe_profiles.name as profileName, nas.name as routerName, pppoe_users.customerId").
		Joins("LEFT JOIN pppoe_profiles ON pppoe_users.profileId = pppoe_profiles.id").
		Joins("LEFT JOIN nas ON pppoe_users.nasId = nas.id").
		Where("pppoe_users.status IN ('active', 'isolated', 'stopped')").
		Find(&users)

	return c.JSON(fiber.Map{
		"total":   len(users),
		"users":   users,
	})
}

// ─── 2.3 Profile Overrides per NAS ───────────────────────────────────────────

// GET /api/roadmap/profiles/:id/overrides — list overrides for a profile
func (h *RoadmapHandler) ListProfileOverrides(c fiber.Ctx) error {
	profileID := c.Params("id")
	var overrides []models.ProfileRouterMap
	h.db.Where("profileId = ?", profileID).Find(&overrides)
	return c.JSON(fiber.Map{"data": overrides})
}

// PUT /api/roadmap/profiles/:id/overrides — set override per router
func (h *RoadmapHandler) SetProfileOverride(c fiber.Ctx) error {
	profileID := c.Params("id")
	var body struct {
		RouterID        string `json:"routerId"`
		MikrotikProfile string `json:"mikrotikProfile"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.RouterID == "" || body.MikrotikProfile == "" {
		return c.Status(400).JSON(fiber.Map{"error": "routerId and mikrotikProfile required"})
	}

	override := models.ProfileRouterMap{
		ID:              uuid.New().String(),
		ProfileID:       profileID,
		RouterID:        body.RouterID,
		MikrotikProfile: body.MikrotikProfile,
	}

	// Upsert: if exists for this profile+router, update
	if err := h.db.Where("profileId = ? AND routerId = ?", profileID, body.RouterID).
		Assign(models.ProfileRouterMap{MikrotikProfile: body.MikrotikProfile}).
		FirstOrCreate(&override).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "override": override})
}

// DELETE /api/roadmap/profiles/:id/overrides/:overrideId — delete override
func (h *RoadmapHandler) DeleteProfileOverride(c fiber.Ctx) error {
	overrideID := c.Params("overrideId")
	if err := h.db.Delete(&models.ProfileRouterMap{}, "id = ?", overrideID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Override deleted"})
}

// ─── 2.4 Waiting List ────────────────────────────────────────────────────────

// GET /api/roadmap/waiting-list — list with filters
func (h *RoadmapHandler) ListWaitingList(c fiber.Ctx) error {
	status := c.Query("status", "")
	territoryID := c.Query("territoryId", "")

	page, pageSize := pageParams(c)
	query := h.db.Model(&models.WaitingList{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if territoryID != "" {
		query = query.Where("territoryId = ?", territoryID)
	}

	var total int64
	query.Count(&total)

	var entries []models.WaitingList
	query.Order("createdAt DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&entries)

	return c.JSON(fiber.Map{"data": entries, "total": total, "page": page, "pageSize": pageSize})
}

// POST /api/roadmap/waiting-list — create entry
func (h *RoadmapHandler) CreateWaitingList(c fiber.Ctx) error {
	var entry models.WaitingList
	if err := c.Bind().JSON(&entry); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	entry.ID = uuid.New().String()
	entry.Status = "waiting"

	if err := h.db.Create(&entry).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "entry": entry})
}

// PUT /api/roadmap/waiting-list/:id — update entry
func (h *RoadmapHandler) UpdateWaitingList(c fiber.Ctx) error {
	id := c.Params("id")
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.db.Model(&models.WaitingList{}).Where("id = ?", id).Updates(body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Waiting list entry updated"})
}

// POST /api/roadmap/waiting-list/:id/assign — assign technician
func (h *RoadmapHandler) AssignWaitingList(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		TechnicianUsername string `json:"technicianUsername"`
		AssignedBy         string `json:"assignedBy"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.TechnicianUsername == "" {
		return c.Status(400).JSON(fiber.Map{"error": "technicianUsername required"})
	}

	assignment := models.WaitingListAssignment{
		ID:                 uuid.New().String(),
		WaitingListID:      id,
		TechnicianUsername: body.TechnicianUsername,
		AssignedBy:         body.AssignedBy,
	}

	if err := h.db.Create(&assignment).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "assignment": assignment})
}

// POST /api/roadmap/waiting-list/:id/convert — convert to PSB (placeholder)
func (h *RoadmapHandler) ConvertWaitingList(c fiber.Ctx) error {
	id := c.Params("id")
	var entry models.WaitingList
	if err := h.db.First(&entry, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "waiting list entry not found"})
	}

	if entry.Status != "waiting" {
		return c.Status(400).JSON(fiber.Map{"error": "only waiting entries can be converted"})
	}

	// Mark as installed
	h.db.Model(&entry).Update("status", "installed")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Entry marked as installed. Create PPPoE user via PSB endpoint with this data.",
		"entry":   entry,
	})
}

// ─── 2.5 ONT Removal Tasks ───────────────────────────────────────────────────

// GET /api/roadmap/ont-removal-tasks — list with filters
func (h *RoadmapHandler) ListOntRemovalTasks(c fiber.Ctx) error {
	status := c.Query("status", "")
	assignedTo := c.Query("assignedTo", "")

	page, pageSize := pageParams(c)
	query := h.db.Model(&models.OntRemovalTask{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if assignedTo != "" {
		query = query.Where("assignedTo = ?", assignedTo)
	}

	var total int64
	query.Count(&total)

	var tasks []models.OntRemovalTask
	query.Order("createdAt DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&tasks)

	return c.JSON(fiber.Map{"data": tasks, "total": total, "page": page, "pageSize": pageSize})
}

// POST /api/roadmap/ont-removal-tasks — create task
func (h *RoadmapHandler) CreateOntRemovalTask(c fiber.Ctx) error {
	var task models.OntRemovalTask
	if err := c.Bind().JSON(&task); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	task.ID = uuid.New().String()
	task.Status = "pending"

	if err := h.db.Create(&task).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "task": task})
}

// POST /api/roadmap/ont-removal-tasks/:id/complete — technician completes task
func (h *RoadmapHandler) CompleteOntRemovalTask(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		ProofPhoto string `json:"proofPhoto"`
		Notes      string `json:"notes"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	updates := map[string]interface{}{"status": "done"}
	if body.ProofPhoto != "" {
		updates["proofPhoto"] = body.ProofPhoto
	}
	if body.Notes != "" {
		updates["notes"] = body.Notes
	}

	if err := h.db.Model(&models.OntRemovalTask{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Task marked as done"})
}

// POST /api/roadmap/ont-removal-tasks/:id/confirm — admin confirms
func (h *RoadmapHandler) ConfirmOntRemovalTask(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		ConfirmedBy string `json:"confirmedBy"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	now := time.Now()
	if err := h.db.Model(&models.OntRemovalTask{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      "confirmed",
		"confirmedBy": body.ConfirmedBy,
		"confirmedAt": &now,
	}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Task confirmed"})
}

// POST /api/roadmap/ont-removal-tasks/:id/cancel — cancel with reason
func (h *RoadmapHandler) CancelOntRemovalTask(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		CancelReason string `json:"cancelReason"`
		CancelledBy  string `json:"cancelledBy"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	now := time.Now()
	if err := h.db.Model(&models.OntRemovalTask{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       "cancelled",
		"cancelReason": body.CancelReason,
		"cancelledBy":  body.CancelledBy,
		"cancelledAt":  &now,
	}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Task cancelled"})
}
