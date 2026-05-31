package handlers

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// HotspotHandler handles hotspot profile and voucher endpoints.
type HotspotHandler struct {
	db *gorm.DB
}

func NewHotspotHandler(db *gorm.DB) *HotspotHandler { return &HotspotHandler{db: db} }

// ─── Profiles ────────────────────────────────────────────────────────────────

func (h *HotspotHandler) ListProfiles(c fiber.Ctx) error {
	var profiles []models.HotspotProfile
	h.db.Order("name").Find(&profiles)
	return c.JSON(fiber.Map{"profiles": profiles})
}

func (h *HotspotHandler) CreateProfile(c fiber.Ctx) error {
	var body models.HotspotProfile
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.ID = uuid.New().String()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}

func (h *HotspotHandler) UpdateProfile(c fiber.Ctx) error {
	id := c.Params("id")
	var p models.HotspotProfile
	if err := h.db.First(&p, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if err := c.Bind().JSON(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	h.db.Save(&p)
	return c.JSON(p)
}

func (h *HotspotHandler) DeleteProfile(c fiber.Ctx) error {
	id := c.Params("id")
	h.db.Delete(&models.HotspotProfile{}, "id = ?", id)
	return c.SendStatus(fiber.StatusNoContent)
}

// ─── Vouchers ─────────────────────────────────────────────────────────────────

func (h *HotspotHandler) ListVouchers(c fiber.Ctx) error {
	query := h.db.Preload("Profile").Preload("Agent")

	profileID := c.Query("profileId")
	batchCode := c.Query("batchCode")
	status := c.Query("status")
	agentID := c.Query("agentId")
	routerID := c.Query("routerId")

	if profileID != "" {
		query = query.Where("profileId = ?", profileID)
	}
	if batchCode != "" {
		query = query.Where("batchCode = ?", batchCode)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if agentID != "" {
		query = query.Where("agentId = ?", agentID)
	}
	if routerID != "" {
		query = query.Where("routerId = ?", routerID)
	}

	var total int64
	query.Model(&models.HotspotVoucher{}).Count(&total)

	page, pageSize := pageParams(c)
	var vouchers []models.HotspotVoucher
	query.Order("createdAt DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&vouchers)

	totalPages := int64(1)
	if total > 0 && pageSize > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}

	// Distinct batch codes for filter dropdown
	var batches []string
	bq := h.db.Model(&models.HotspotVoucher{}).Where("batchCode IS NOT NULL")
	if profileID != "" {
		bq = bq.Where("profileId = ?", profileID)
	}
	if agentID != "" {
		bq = bq.Where("agentId = ?", agentID)
	}
	bq.Distinct("batchCode").Pluck("batchCode", &batches)

	// Stats (apply same filters minus status so we get all counts)
	sq := h.db.Model(&models.HotspotVoucher{})
	if profileID != "" {
		sq = sq.Where("profileId = ?", profileID)
	}
	if batchCode != "" {
		sq = sq.Where("batchCode = ?", batchCode)
	}
	if agentID != "" {
		sq = sq.Where("agentId = ?", agentID)
	}
	if routerID != "" {
		sq = sq.Where("routerId = ?", routerID)
	}
	var statTotal, statWaiting, statActive, statExpired int64
	sq.Count(&statTotal)
	sq.Where("status = ?", "WAITING").Count(&statWaiting)
	sq.Where("status = ?", "ACTIVE").Count(&statActive)
	sq.Where("status = ?", "EXPIRED").Count(&statExpired)

	return c.JSON(fiber.Map{
		"vouchers":   vouchers,
		"batches":    batches,
		"total":      total,
		"totalPages": totalPages,
		"stats": fiber.Map{
			"total":      statTotal,
			"waiting":    statWaiting,
			"active":     statActive,
			"expired":    statExpired,
			"totalValue": 0,
		},
	})
}

func (h *HotspotHandler) GenerateVouchers(c fiber.Ctx) error {
	var body struct {
		ProfileID   string `json:"profileId"`
		Quantity    int    `json:"quantity"`
		BatchCode   string `json:"batchCode"`
		AgentID     string `json:"agentId"`
		RouterID    string `json:"routerId"`
		Prefix      string `json:"prefix"`
		CodeLength  int    `json:"codeLength"`
		VoucherType string `json:"voucherType"`
		CodeType    string `json:"codeType"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if body.Quantity <= 0 || body.Quantity > 5000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "quantity must be 1-5000"})
	}
	if body.ProfileID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "profileId required"})
	}
	if body.CodeLength <= 0 {
		body.CodeLength = 8
	}
	batchCode := body.BatchCode
	if batchCode == "" {
		batchCode = fmt.Sprintf("BATCH-%d", time.Now().UnixMilli())
	}
	if body.VoucherType == "" {
		body.VoucherType = "same"
	}
	if body.CodeType == "" {
		body.CodeType = "alphanumeric"
	}

	// Fetch existing codes to avoid duplicates
	var existingCodes []string
	h.db.Model(&models.HotspotVoucher{}).Pluck("code", &existingCodes)
	existingSet := make(map[string]struct{}, len(existingCodes))
	for _, c := range existingCodes {
		existingSet[c] = struct{}{}
	}

	var created []models.HotspotVoucher
	for i := 0; i < body.Quantity; i++ {
		var code string
		for attempt := 0; attempt < 20; attempt++ {
			candidate := body.Prefix + generateVoucherCode(body.CodeLength)
			if _, dup := existingSet[candidate]; !dup {
				code = candidate
				existingSet[code] = struct{}{}
				break
			}
		}
		if code == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal generate kode unik setelah 20 percobaan, coba panjang kode lebih besar"})
		}
		v := models.HotspotVoucher{
			ID:          uuid.New().String(),
			Code:        code,
			ProfileID:   body.ProfileID,
			BatchCode:   &batchCode,
			VoucherType: body.VoucherType,
			CodeType:    body.CodeType,
			Status:      "WAITING",
		}
		if body.AgentID != "" {
			v.AgentID = &body.AgentID
		}
		if body.RouterID != "" {
			v.RouterID = &body.RouterID
		}
		created = append(created, v)
	}

	if err := h.db.Create(&created).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"batchCode": batchCode,
		"count":     len(created),
	})
}

func (h *HotspotHandler) DeleteVoucher(c fiber.Ctx) error {
	id := c.Params("id")
	h.db.Delete(&models.HotspotVoucher{}, "id = ?", id)
	return c.SendStatus(fiber.StatusNoContent)
}

func generateVoucherCode(length int) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			// fallback to time-based if crypto/rand fails
			code[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		} else {
			code[i] = chars[n.Int64()]
		}
	}
	return string(code)
}
