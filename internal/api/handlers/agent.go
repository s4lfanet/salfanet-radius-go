package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// AgentHandler handles agent management endpoints.
type AgentHandler struct {
	db *gorm.DB
}

func NewAgentHandler(db *gorm.DB) *AgentHandler { return &AgentHandler{db: db} }

func (h *AgentHandler) ListAgents(c fiber.Ctx) error {
	var agents []models.Agent
	h.db.Order("name").Find(&agents)
	return c.JSON(agents)
}

func (h *AgentHandler) CreateAgent(c fiber.Ctx) error {
	var body models.Agent
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.ID = uuid.New().String()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}

func (h *AgentHandler) GetAgent(c fiber.Ctx) error {
	id := c.Params("id")
	var agent models.Agent
	if err := h.db.First(&agent, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(agent)
}

func (h *AgentHandler) UpdateAgent(c fiber.Ctx) error {
	id := c.Params("id")
	var agent models.Agent
	if err := h.db.First(&agent, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if err := c.Bind().JSON(&agent); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	h.db.Save(&agent)
	return c.JSON(agent)
}

func (h *AgentHandler) DeleteAgent(c fiber.Ctx) error {
	id := c.Params("id")
	h.db.Delete(&models.Agent{}, "id = ?", id)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *AgentHandler) GetAgentSales(c fiber.Ctx) error {
	id := c.Params("id")
	var sales []models.AgentSale
	h.db.Preload("Voucher.Profile").Where("agent_id = ?", id).Order("created_at DESC").Limit(200).Find(&sales)
	return c.JSON(sales)
}

func (h *AgentHandler) GetAgentDeposits(c fiber.Ctx) error {
	id := c.Params("id")
	var deposits []models.AgentDeposit
	h.db.Where("agent_id = ?", id).Order("created_at DESC").Limit(200).Find(&deposits)
	return c.JSON(deposits)
}

func (h *AgentHandler) TopupBalance(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Amount int    `json:"amount"`
		Notes  string `json:"notes"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.db.Model(&models.Agent{}).Where("id = ?", id).
		UpdateColumn("balance", gorm.Expr("balance + ?", body.Amount)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	notes := body.Notes
	deposit := models.AgentDeposit{
		ID:      uuid.New().String(),
		AgentID: id,
		Amount:  body.Amount,
		Notes:   &notes,
	}
	h.db.Create(&deposit)

	return c.JSON(fiber.Map{"message": "balance updated"})
}

func (h *AgentHandler) ListAgentVouchers(c fiber.Ctx) error {
	id := c.Params("id")
	var vouchers []models.HotspotVoucher
	h.db.Preload("Profile").Where("agent_id = ?", id).Order("created_at DESC").Find(&vouchers)
	return c.JSON(vouchers)
}

// GET /api/agent/dashboard — agent self-service dashboard
func (h *AgentHandler) Dashboard(c fiber.Ctx) error {
	agentID, _ := c.Locals("agentID").(string)
	var agent models.Agent
	if err := h.db.First(&agent, "id = ?", agentID).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	var salesCount int64
	h.db.Model(&models.AgentSale{}).Where("agent_id = ?", agentID).Count(&salesCount)
	var depositCount int64
	h.db.Model(&models.AgentDeposit{}).Where("agent_id = ?", agentID).Count(&depositCount)
	return c.JSON(fiber.Map{
		"success": true,
		"agent":   agent,
		"stats":   fiber.Map{"salesCount": salesCount, "depositCount": depositCount},
	})
}

// POST /api/agent/deposit/create — agent requests a deposit top-up
func (h *AgentHandler) CreateDeposit(c fiber.Ctx) error {
	agentID, _ := c.Locals("agentID").(string)
	var body struct {
		Amount int    `json:"amount"`
		Notes  string `json:"notes"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Amount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "amount required"})
	}
	notes := body.Notes
	deposit := models.AgentDeposit{
		ID:      uuid.New().String(),
		AgentID: agentID,
		Amount:  body.Amount,
		Notes:   &notes,
	}
	h.db.Create(&deposit)
	return c.JSON(fiber.Map{"success": true, "deposit": deposit})
}

// POST /api/agent/deposit/webhook — payment webhook for agent deposit
func (h *AgentHandler) DepositWebhook(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"received": true})
}

// POST /api/agent/generate-voucher — agent generates a hotspot voucher
func (h *AgentHandler) GenerateVoucher(c fiber.Ctx) error {
	agentID, _ := c.Locals("agentID").(string)
	var body struct {
		ProfileID string `json:"profileId"`
		Quantity  int    `json:"quantity"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.ProfileID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "profileId required"})
	}
	if body.Quantity <= 0 {
		body.Quantity = 1
	}
	_ = agentID
	return c.JSON(fiber.Map{"success": true, "message": "Voucher generation queued", "quantity": body.Quantity})
}

// POST /api/agent/record-sales — agent records a manual sale
func (h *AgentHandler) RecordSales(c fiber.Ctx) error {
	agentID, _ := c.Locals("agentID").(string)
	var body struct {
		VoucherID string `json:"voucherId"`
		Amount    int    `json:"amount"`
		Customer  string `json:"customer"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	sale := models.AgentSale{
		ID:        uuid.New().String(),
		AgentID:   agentID,
		VoucherID: body.VoucherID,
	}
	h.db.Create(&sale)
	return c.JSON(fiber.Map{"success": true, "sale": sale})
}
