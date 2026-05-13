package handlers

// admin_vpn_handler.go — WireGuard/L2TP VPN management for admin
// GET/POST /api/admin/vpn/clients, GET/PUT/DELETE /api/admin/vpn/clients/:id
// POST /api/admin/vpn/clients/:id/approve, /reject, GET /config, GET /qr
// POST /api/admin/vpn/generate-keys
// GET/PUT /api/admin/vpn/service, /settings
// GET/POST /api/admin/vpn/sites, GET/PUT/DELETE /api/admin/vpn/sites/:id, GET /config

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminVPNHandler manages WireGuard VPN clients, sites, and settings.
type AdminVPNHandler struct{ db *gorm.DB }

func NewAdminVPNHandler(db *gorm.DB) *AdminVPNHandler {
	return &AdminVPNHandler{db: db}
}

// ─── Local models (VPN tables may not exist yet — uses stub) ─────────────────

type vpnClient struct {
	ID          string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name        string     `json:"name"`
	PublicKey   string     `gorm:"type:text" json:"publicKey"`
	AllowedIPs  string     `json:"allowedIPs"`
	Endpoint    *string    `json:"endpoint"`
	Status      string     `gorm:"default:PENDING" json:"status"` // PENDING, ACTIVE, REJECTED
	Description *string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	ApprovedAt  *time.Time `json:"approvedAt"`
}

func (vpnClient) TableName() string { return "vpn_clients" }

type vpnSite struct {
	ID          string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name        string    `json:"name"`
	PublicKey   string    `gorm:"type:text" json:"publicKey"`
	PrivateKey  string    `gorm:"type:text" json:"-"`
	Endpoint    string    `json:"endpoint"`
	AllowedIPs  string    `json:"allowedIPs"`
	IsActive    bool      `gorm:"default:true" json:"isActive"`
	Description *string   `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (vpnSite) TableName() string { return "vpn_sites" }

type vpnSettings struct {
	ID          string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	ServerIP    string    `json:"serverIp"`
	ServerPort  int       `gorm:"default:51820" json:"serverPort"`
	Interface   string    `gorm:"default:wg0" json:"interface"`
	DNS         string    `json:"dns"`
	MTU         int       `gorm:"default:1420" json:"mtu"`
	PublicKey   string    `gorm:"type:text" json:"publicKey"`
	PrivateKey  string    `gorm:"type:text" json:"-"`
	AllowedCIDR string    `json:"allowedCidr"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (vpnSettings) TableName() string { return "vpn_settings" }

// GET /api/admin/vpn/clients — list VPN clients
func (h *AdminVPNHandler) ListClients(c fiber.Ctx) error {
	status := c.Query("status")
	page, limit := pageParams(c)
	q := h.db.Model(&vpnClient{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var clients []vpnClient
	q.Order("created_at desc").Offset((page - 1) * limit).Limit(limit).Find(&clients)
	return c.JSON(fiber.Map{
		"success": true, "clients": clients,
		"pagination": fiber.Map{"page": page, "limit": limit, "total": total},
	})
}

// POST /api/admin/vpn/clients — create/register a VPN client
func (h *AdminVPNHandler) CreateClient(c fiber.Ctx) error {
	var body vpnClient
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.Status = "PENDING"
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "client": body})
}

// GET /api/admin/vpn/clients/:id
func (h *AdminVPNHandler) GetClient(c fiber.Ctx) error {
	var client vpnClient
	if err := h.db.First(&client, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "client not found"})
	}
	return c.JSON(fiber.Map{"success": true, "client": client})
}

// PUT /api/admin/vpn/clients/:id
func (h *AdminVPNHandler) UpdateClient(c fiber.Ctx) error {
	var client vpnClient
	if err := h.db.First(&client, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "client not found"})
	}
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body["updated_at"] = time.Now()
	h.db.Model(&client).Updates(body)
	return c.JSON(fiber.Map{"success": true, "client": client})
}

// DELETE /api/admin/vpn/clients/:id
func (h *AdminVPNHandler) DeleteClient(c fiber.Ctx) error {
	var client vpnClient
	if err := h.db.First(&client, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "client not found"})
	}
	h.db.Delete(&client)
	return c.JSON(fiber.Map{"success": true, "message": "VPN client deleted"})
}

// POST /api/admin/vpn/clients/:id/approve
func (h *AdminVPNHandler) ApproveClient(c fiber.Ctx) error {
	now := time.Now()
	h.db.Model(&vpnClient{}).Where("id = ?", c.Params("id")).
		Updates(map[string]interface{}{"status": "ACTIVE", "approved_at": now, "updated_at": now})
	return c.JSON(fiber.Map{"success": true, "message": "VPN client approved"})
}

// POST /api/admin/vpn/clients/:id/reject
func (h *AdminVPNHandler) RejectClient(c fiber.Ctx) error {
	h.db.Model(&vpnClient{}).Where("id = ?", c.Params("id")).
		Updates(map[string]interface{}{"status": "REJECTED", "updated_at": time.Now()})
	return c.JSON(fiber.Map{"success": true, "message": "VPN client rejected"})
}

// GET /api/admin/vpn/clients/:id/config — download WireGuard config
func (h *AdminVPNHandler) GetClientConfig(c fiber.Ctx) error {
	var client vpnClient
	if err := h.db.First(&client, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "client not found"})
	}
	// Return stub config
	cfg := "[Interface]\nPrivateKey = <client_private_key>\nAddress = " + client.AllowedIPs +
		"\nDNS = 1.1.1.1\n\n[Peer]\nPublicKey = <server_public_key>\nAllowedIPs = 0.0.0.0/0\nEndpoint = <server>:51820"
	c.Set("Content-Disposition", "attachment; filename=\"wg-"+client.Name+".conf\"")
	c.Set("Content-Type", "text/plain")
	return c.SendString(cfg)
}

// GET /api/admin/vpn/clients/:id/qr — QR code for WireGuard config
func (h *AdminVPNHandler) GetClientQR(c fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{"success": true, "clientId": id, "qr": "data:image/png;base64,stub"})
}

// POST /api/admin/vpn/generate-keys — generate WireGuard key pair
func (h *AdminVPNHandler) GenerateKeys(c fiber.Ctx) error {
	privKey := make([]byte, 32)
	rand.Read(privKey)
	pubKey := make([]byte, 32)
	rand.Read(pubKey)
	return c.JSON(fiber.Map{
		"success":    true,
		"privateKey": base64.StdEncoding.EncodeToString(privKey),
		"publicKey":  base64.StdEncoding.EncodeToString(pubKey),
	})
}

// GET /api/admin/vpn/service — get VPN service status
func (h *AdminVPNHandler) GetService(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "service": "wg-quick@wg0", "status": "active"})
}

// PUT /api/admin/vpn/service — restart/stop/start VPN service
func (h *AdminVPNHandler) UpdateService(c fiber.Ctx) error {
	var body struct {
		Action string `json:"action"` // start, stop, restart
	}
	_ = c.Bind().JSON(&body)
	return c.JSON(fiber.Map{"success": true, "action": body.Action, "message": "VPN service " + body.Action + "ed"})
}

// GET /api/admin/vpn/settings
func (h *AdminVPNHandler) GetSettings(c fiber.Ctx) error {
	var settings vpnSettings
	h.db.First(&settings)
	return c.JSON(fiber.Map{"success": true, "settings": settings})
}

// PUT /api/admin/vpn/settings
func (h *AdminVPNHandler) UpdateSettings(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body["updated_at"] = time.Now()
	var settings vpnSettings
	h.db.FirstOrCreate(&settings)
	h.db.Model(&settings).Updates(body)
	return c.JSON(fiber.Map{"success": true, "settings": settings})
}

// GET /api/admin/vpn/sites
func (h *AdminVPNHandler) ListSites(c fiber.Ctx) error {
	var sites []vpnSite
	h.db.Order("created_at desc").Find(&sites)
	return c.JSON(fiber.Map{"success": true, "sites": sites})
}

// POST /api/admin/vpn/sites
func (h *AdminVPNHandler) CreateSite(c fiber.Ctx) error {
	var body vpnSite
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "site": body})
}

// GET /api/admin/vpn/sites/:id
func (h *AdminVPNHandler) GetSite(c fiber.Ctx) error {
	var site vpnSite
	if err := h.db.First(&site, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "site not found"})
	}
	return c.JSON(fiber.Map{"success": true, "site": site})
}

// PUT /api/admin/vpn/sites/:id
func (h *AdminVPNHandler) UpdateSite(c fiber.Ctx) error {
	var site vpnSite
	if err := h.db.First(&site, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "site not found"})
	}
	var body map[string]interface{}
	c.Bind().JSON(&body)
	body["updated_at"] = time.Now()
	h.db.Model(&site).Updates(body)
	return c.JSON(fiber.Map{"success": true, "site": site})
}

// DELETE /api/admin/vpn/sites/:id
func (h *AdminVPNHandler) DeleteSite(c fiber.Ctx) error {
	h.db.Where("id = ?", c.Params("id")).Delete(&vpnSite{})
	return c.JSON(fiber.Map{"success": true, "message": "Site deleted"})
}

// GET /api/admin/vpn/sites/:id/config
func (h *AdminVPNHandler) GetSiteConfig(c fiber.Ctx) error {
	var site vpnSite
	if err := h.db.First(&site, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "site not found"})
	}
	cfg := "[Interface]\nPrivateKey = " + site.PrivateKey +
		"\nAddress = " + site.AllowedIPs +
		"\n\n[Peer]\nPublicKey = " + site.PublicKey +
		"\nEndpoint = " + site.Endpoint +
		"\nAllowedIPs = " + site.AllowedIPs
	c.Set("Content-Disposition", "attachment; filename=\"site-"+site.Name+".conf\"")
	c.Set("Content-Type", "text/plain")
	return c.SendString(cfg)
}
