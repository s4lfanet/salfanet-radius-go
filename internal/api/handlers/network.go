package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// NetworkHandler handles network map endpoints (OLT, ODC, ODP, OTB, Router).
type NetworkHandler struct {
	db *gorm.DB
}

func NewNetworkHandler(db *gorm.DB) *NetworkHandler { return &NetworkHandler{db: db} }

// ─── OLTs for map ────────────────────────────────────────────────────────────

func (h *NetworkHandler) ListOLTsForMap(c fiber.Ctx) error {
	var olts []models.NetworkOLT
	h.db.Find(&olts)
	return c.JSON(olts)
}

// ─── ODC ─────────────────────────────────────────────────────────────────────

func (h *NetworkHandler) ListODCs(c fiber.Ctx) error {
	var odcs []models.NetworkODC
	h.db.Find(&odcs)
	return c.JSON(odcs)
}

func (h *NetworkHandler) CreateODC(c fiber.Ctx) error {
	var body models.NetworkODC
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.ID = uuid.New().String()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}

func (h *NetworkHandler) UpdateODC(c fiber.Ctx) error {
	id := c.Params("id")
	var odc models.NetworkODC
	if err := h.db.First(&odc, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if err := c.Bind().JSON(&odc); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	h.db.Save(&odc)
	return c.JSON(odc)
}

func (h *NetworkHandler) DeleteODC(c fiber.Ctx) error {
	id := c.Params("id")
	h.db.Delete(&models.NetworkODC{}, "id = ?", id)
	return c.SendStatus(fiber.StatusNoContent)
}

// ─── ODP ─────────────────────────────────────────────────────────────────────

func (h *NetworkHandler) ListODPs(c fiber.Ctx) error {
	var odps []models.NetworkODP
	h.db.Find(&odps)
	return c.JSON(odps)
}

func (h *NetworkHandler) CreateODP(c fiber.Ctx) error {
	var body models.NetworkODP
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.ID = uuid.New().String()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}

func (h *NetworkHandler) UpdateODP(c fiber.Ctx) error {
	id := c.Params("id")
	var odp models.NetworkODP
	if err := h.db.First(&odp, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if err := c.Bind().JSON(&odp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	h.db.Save(&odp)
	return c.JSON(odp)
}

func (h *NetworkHandler) DeleteODP(c fiber.Ctx) error {
	id := c.Params("id")
	h.db.Delete(&models.NetworkODP{}, "id = ?", id)
	return c.SendStatus(fiber.StatusNoContent)
}

// ─── OTB ─────────────────────────────────────────────────────────────────────

func (h *NetworkHandler) ListOTBs(c fiber.Ctx) error {
	var otbs []models.NetworkOTB
	h.db.Find(&otbs)
	return c.JSON(otbs)
}

func (h *NetworkHandler) CreateOTB(c fiber.Ctx) error {
	var body models.NetworkOTB
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.ID = uuid.New().String()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}

// ─── Routers ─────────────────────────────────────────────────────────────────

func (h *NetworkHandler) ListRouters(c fiber.Ctx) error {
	var routers []models.Router
	h.db.Order("name").Find(&routers)

	// Fetch VPN clients for the "Tambah Router Baru" dropdown
	type vpnClientRow struct {
		ID             string  `gorm:"column:id"`
		Name           string  `gorm:"column:name"`
		VpnIp          string  `gorm:"column:vpnIp"`
		IsRadiusServer bool    `gorm:"column:isRadiusServer"`
		ApiUsername    *string `gorm:"column:apiUsername"`
		ApiPassword    *string `gorm:"column:apiPassword"`
	}
	var vpnClientRows []vpnClientRow
	h.db.Table("vpn_clients").Order("name").Find(&vpnClientRows)

	// Also include VPS WireGuard/L2TP peers from vps_peers (Go-managed table)
	type vpsPeerRow struct {
		ID          string  `gorm:"column:id"`
		PeerName    string  `gorm:"column:peer_name"`
		PeerIp      string  `gorm:"column:peer_ip"`
		NasSecret   *string `gorm:"column:nas_secret"`
		ApiUsername *string `gorm:"column:api_username"`
		ApiPassword *string `gorm:"column:api_password"`
	}
	var vpsPeerRows []vpsPeerRow
	// Use Exec-safe query: vps_peers is always created by runMigrations at startup
	h.db.Table("vps_peers").Where("is_active = ?", 1).Order("peer_name").Find(&vpsPeerRows)

	// Build NAS secret map
	type nasSecretRow struct {
		VpnClientId string `gorm:"column:vpnClientId"`
		Secret      string `gorm:"column:secret"`
	}
	secretMap := make(map[string]*string)
	if len(vpnClientRows) > 0 {
		ids := make([]string, len(vpnClientRows))
		for i, cl := range vpnClientRows {
			ids[i] = cl.ID
		}
		var nasRows []nasSecretRow
		h.db.Table("nas").Select("vpnClientId, secret").
			Where("vpnClientId IN ?", ids).Find(&nasRows)
		for _, row := range nasRows {
			s := row.Secret
			secretMap[row.VpnClientId] = &s
		}
	}

	type vpnClientResp struct {
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		VpnIp          string  `json:"vpnIp"`
		IsRadiusServer bool    `json:"isRadiusServer"`
		ApiUsername    *string `json:"apiUsername"`
		ApiPassword    *string `json:"apiPassword"`
		NasSecret      *string `json:"nasSecret"`
	}
	vpnClientResult := make([]vpnClientResp, 0, len(vpnClientRows)+len(vpsPeerRows))
	for _, cl := range vpnClientRows {
		vpnClientResult = append(vpnClientResult, vpnClientResp{
			ID:             cl.ID,
			Name:           cl.Name,
			VpnIp:          cl.VpnIp,
			IsRadiusServer: cl.IsRadiusServer,
			ApiUsername:    cl.ApiUsername,
			ApiPassword:    cl.ApiPassword,
			NasSecret:      secretMap[cl.ID],
		})
	}
	// Append VPS peers (WireGuard/L2TP) — these use peer_name and peer_ip
	for _, p := range vpsPeerRows {
		vpnClientResult = append(vpnClientResult, vpnClientResp{
			ID:          p.ID,
			Name:        p.PeerName,
			VpnIp:       p.PeerIp,
			ApiUsername: p.ApiUsername,
			ApiPassword: p.ApiPassword,
			NasSecret:   p.NasSecret,
		})
	}

	return c.JSON(fiber.Map{
		"routers":    routers,
		"vpnClients": vpnClientResult,
	})
}

func (h *NetworkHandler) CreateRouter(c fiber.Ctx) error {
	var body models.Router
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.ID = uuid.New().String()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}
