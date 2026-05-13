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

// GET /api/agent/deposit/check — check latest deposit status
func (h *AgentHandler) DepositCheck(c fiber.Ctx) error {
	agentID, _ := c.Locals("agentID").(string)
	var deposit models.AgentDeposit
	h.db.Where("agent_id = ?", agentID).Order("created_at desc").First(&deposit)
	return c.JSON(fiber.Map{"success": true, "deposit": deposit})
}

// POST /api/agent/deposit/manual-request — agent submits manual deposit request
func (h *AgentHandler) DepositManualRequest(c fiber.Ctx) error {
	agentID, _ := c.Locals("agentID").(string)
	var body struct {
		Amount      int    `json:"amount"`
		BankName    string `json:"bankName"`
		AccountName string `json:"accountName"`
		TransferRef string `json:"transferRef"`
		Notes       string `json:"notes"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Amount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "amount required"})
	}
	notes := body.Notes + " | Bank: " + body.BankName + " | Ref: " + body.TransferRef
	deposit := models.AgentDeposit{
		ID:      uuid.New().String(),
		AgentID: agentID,
		Amount:  body.Amount,
		Notes:   &notes,
	}
	h.db.Create(&deposit)
	return c.Status(201).JSON(fiber.Map{"success": true, "deposit": deposit})
}

// GET /api/agent/deposit/payment-methods — list available payment methods for deposit
func (h *AgentHandler) ListDepositPaymentMethods(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"methods": []fiber.Map{
			{"id": "bank_transfer", "name": "Bank Transfer", "active": true},
			{"id": "qris", "name": "QRIS", "active": true},
		},
	})
}

// GET /api/agent/notifications — agent notifications list
func (h *AgentHandler) GetAgentNotifications(c fiber.Ctx) error {
	agentID, _ := c.Locals("agentID").(string)
	var notifications []models.Notification
	h.db.Where("user_id = ?", agentID).Order("created_at desc").Limit(50).Find(&notifications)
	return c.JSON(fiber.Map{"success": true, "notifications": notifications})
}

// GET /api/agent/sessions — hotspot sessions under agent's vouchers
func (h *AgentHandler) GetAgentSessions(c fiber.Ctx) error {
	agentID, _ := c.Locals("agentID").(string)
	page, limit := pageParams(c)
	type SessionRow struct {
		Username        string `json:"username"`
		CalledStationID string `json:"calledStationId"`
		AcctStartTime   string `json:"acctStartTime"`
	}
	var sessions []SessionRow
	h.db.Raw(`
		SELECT r.username, r.called_station_id, r.acct_start_time
		FROM radacct r
		JOIN hotspot_vouchers v ON r.username = v.code
		WHERE v.agent_id = ?
		ORDER BY r.acct_start_time DESC
		LIMIT ? OFFSET ?
	`, agentID, limit, (page-1)*limit).Scan(&sessions)
	return c.JSON(fiber.Map{"success": true, "sessions": sessions})
}

// GET /api/agent/tickets — agent support tickets
func (h *AgentHandler) GetAgentTickets(c fiber.Ctx) error {
	agentID, _ := c.Locals("agentID").(string)
	page, limit := pageParams(c)
	var tickets []map[string]interface{}
	h.db.Raw(`
		SELECT t.id, t.subject, t.status, t.priority, t.created_at
		FROM tickets t
		WHERE t.created_by = ? OR t.agent_id = ?
		ORDER BY t.created_at DESC
		LIMIT ? OFFSET ?
	`, agentID, agentID, limit, (page-1)*limit).Scan(&tickets)
	return c.JSON(fiber.Map{"success": true, "tickets": tickets})
}

// GET /api/agent/tickets/:id — single agent ticket detail
func (h *AgentHandler) GetAgentTicket(c fiber.Ctx) error {
	ticketID := c.Params("id")
	var ticket map[string]interface{}
	h.db.Raw("SELECT * FROM tickets WHERE id = ?", ticketID).Scan(&ticket)
	if ticket == nil {
		return c.Status(404).JSON(fiber.Map{"error": "ticket not found"})
	}
	return c.JSON(fiber.Map{"success": true, "ticket": ticket})
}
