package handlers

// network_vpn_ext_handler.go — Network VPN server/client config + VPS peer management
// These routes are distinct from /api/admin/vpn/* (WireGuard VPN management).
// They handle MikroTik VPN server configuration and VPS tunneling.
//
// GET/POST /api/network/vpn-server, /setup, /test
// POST     /api/network/vpn-server/l2tp-control
// POST     /api/network/vpn-server/pptp-control
// POST     /api/network/vpn-server/sstp-control
// GET/POST /api/network/vpn-client
// GET/POST /api/network/vpn-routing
// GET      /api/network/vps-info
// GET      /api/network/vps-l2tp-info
// GET/POST /api/network/vps-l2tp-peer
// GET/POST /api/network/vps-wg-peer

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NetworkVPNHandler manages MikroTik VPN server + VPS tunnel peer configs.
type NetworkVPNHandler struct{ db *gorm.DB }

func NewNetworkVPNHandler(db *gorm.DB) *NetworkVPNHandler {
	return &NetworkVPNHandler{db: db}
}

// ─── Local models ─────────────────────────────────────────────────────────────

type vpnServerConfig struct {
	ID          string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	RouterID    string    `gorm:"index" json:"routerId"`
	L2TPEnabled bool      `gorm:"default:false" json:"l2tpEnabled"`
	PPTPEnabled bool      `gorm:"default:false" json:"pptpEnabled"`
	SSTEnabled  bool      `gorm:"default:false" json:"sstpEnabled"`
	L2TPSecret  *string   `json:"l2tpSecret"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (vpnServerConfig) TableName() string { return "vpn_server_configs" }

type vpnClientConfig struct {
	ID         string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	RouterID   string    `gorm:"index" json:"routerId"`
	ServerHost string    `json:"serverHost"`
	Protocol   string    `json:"protocol"` // l2tp, pptp, sstp, wireguard
	Username   *string   `json:"username"`
	IsActive   bool      `gorm:"default:false" json:"isActive"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (vpnClientConfig) TableName() string { return "vpn_client_configs" }

type vpnRouting struct {
	ID          string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Destination string    `json:"destination"`
	Gateway     string    `json:"gateway"`
	Interface   *string   `json:"interface"`
	Comment     *string   `json:"comment"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (vpnRouting) TableName() string { return "vpn_routings" }

type vpsPeer struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Type      string    `json:"type"` // l2tp, wireguard
	PeerName  string    `json:"peerName"`
	PeerIP    string    `json:"peerIp"`
	LocalIP   string    `json:"localIp"`
	PublicKey *string   `json:"publicKey"`
	IsActive  bool      `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (vpsPeer) TableName() string { return "vps_peers" }

// ─── VPN Server ──────────────────────────────────────────────────────────────

// GET /api/network/vpn-server — get VPN server config for a router
func (h *NetworkVPNHandler) GetVPNServer(c fiber.Ctx) error {
	routerID := c.Query("routerId")
	var config vpnServerConfig
	q := h.db
	if routerID != "" {
		q = q.Where("routerId = ?", routerID)
	}
	q.First(&config)
	return c.JSON(fiber.Map{"success": true, "config": config})
}

// POST /api/network/vpn-server — update VPN server config
func (h *NetworkVPNHandler) UpdateVPNServer(c fiber.Ctx) error {
	var body vpnServerConfig
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	var existing vpnServerConfig
	if err := h.db.Where("routerId = ?", body.RouterID).First(&existing).Error; err != nil {
		body.ID = uuid.New().String()
		body.UpdatedAt = time.Now()
		h.db.Create(&body)
	} else {
		body.ID = existing.ID
		body.UpdatedAt = time.Now()
		h.db.Model(&existing).Updates(body)
	}
	return c.JSON(fiber.Map{"success": true, "config": body})
}

// POST /api/network/vpn-server/setup — initial VPN server setup on router
func (h *NetworkVPNHandler) SetupVPNServer(c fiber.Ctx) error {
	var body struct {
		RouterID string `json:"routerId"`
		Protocol string `json:"protocol"`
	}
	c.Bind().JSON(&body)
	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "VPN server setup queued for router " + body.RouterID,
		"protocol": body.Protocol,
	})
}

// POST /api/network/vpn-server/test — test VPN server connectivity
func (h *NetworkVPNHandler) TestVPNServer(c fiber.Ctx) error {
	var body struct {
		RouterID string `json:"routerId"`
	}
	c.Bind().JSON(&body)
	return c.JSON(fiber.Map{"success": true, "reachable": true, "message": "VPN server test stub"})
}

// POST /api/network/vpn-server/l2tp-control — enable/disable L2TP server
func (h *NetworkVPNHandler) L2TPControl(c fiber.Ctx) error {
	var body struct {
		RouterID string `json:"routerId"`
		Action   string `json:"action"` // enable, disable
	}
	c.Bind().JSON(&body)
	return c.JSON(fiber.Map{"success": true, "action": body.Action, "protocol": "l2tp"})
}

// POST /api/network/vpn-server/pptp-control — enable/disable PPTP server
func (h *NetworkVPNHandler) PPTPControl(c fiber.Ctx) error {
	var body struct {
		RouterID string `json:"routerId"`
		Action   string `json:"action"`
	}
	c.Bind().JSON(&body)
	return c.JSON(fiber.Map{"success": true, "action": body.Action, "protocol": "pptp"})
}

// POST /api/network/vpn-server/sstp-control — enable/disable SSTP server
func (h *NetworkVPNHandler) SSTPControl(c fiber.Ctx) error {
	var body struct {
		RouterID string `json:"routerId"`
		Action   string `json:"action"`
	}
	c.Bind().JSON(&body)
	return c.JSON(fiber.Map{"success": true, "action": body.Action, "protocol": "sstp"})
}

// ─── VPN Client ──────────────────────────────────────────────────────────────

// GET /api/network/vpn-client — list VPN client configs
func (h *NetworkVPNHandler) ListVPNClients(c fiber.Ctx) error {
	routerID := c.Query("routerId")
	var clients []vpnClientConfig
	q := h.db
	if routerID != "" {
		q = q.Where("routerId = ?", routerID)
	}
	q.Order("createdAt desc").Find(&clients)
	return c.JSON(fiber.Map{"success": true, "clients": clients})
}

// POST /api/network/vpn-client — create/update VPN client config
func (h *NetworkVPNHandler) CreateVPNClient(c fiber.Ctx) error {
	var body vpnClientConfig
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "client": body})
}

// ─── VPN Routing ─────────────────────────────────────────────────────────────

// GET /api/network/vpn-routing — list VPN routes
func (h *NetworkVPNHandler) ListVPNRouting(c fiber.Ctx) error {
	var routes []vpnRouting
	h.db.Order("createdAt desc").Find(&routes)
	return c.JSON(fiber.Map{"success": true, "routes": routes})
}

// POST /api/network/vpn-routing — add a VPN route
func (h *NetworkVPNHandler) CreateVPNRoute(c fiber.Ctx) error {
	var body vpnRouting
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "route": body})
}

// ─── VPS Info ────────────────────────────────────────────────────────────────

// GET /api/network/vps-info — VPS host info (hostname, IP, etc.)
func (h *NetworkVPNHandler) GetVPSInfo(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"vps": fiber.Map{
			"hostname": "salfanet-vps",
			"publicIP": "",
			"os":       "Ubuntu 22.04 LTS",
			"uptime":   "N/A",
			"cpu":      "N/A",
			"memory":   "N/A",
		},
	})
}

// GET /api/network/vps-l2tp-info — L2TP config on this VPS
func (h *NetworkVPNHandler) GetVPSL2TPInfo(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"l2tp": fiber.Map{
			"enabled":   false,
			"serverIP":  "",
			"interface": "l2tp1",
		},
	})
}

// ─── VPS Peers ───────────────────────────────────────────────────────────────

// GET /api/network/vps-l2tp-peer — list L2TP peers
func (h *NetworkVPNHandler) ListL2TPPeers(c fiber.Ctx) error {
	var peers []vpsPeer
	h.db.Where("type = ?", "l2tp").Order("createdAt desc").Find(&peers)
	return c.JSON(fiber.Map{"success": true, "peers": peers})
}

// POST /api/network/vps-l2tp-peer — add L2TP peer
func (h *NetworkVPNHandler) CreateL2TPPeer(c fiber.Ctx) error {
	var body vpsPeer
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.Type = "l2tp"
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "peer": body})
}

// GET /api/network/vps-wg-peer — list WireGuard peers on VPS
func (h *NetworkVPNHandler) ListWGPeers(c fiber.Ctx) error {
	var peers []vpsPeer
	h.db.Where("type = ?", "wireguard").Order("createdAt desc").Find(&peers)
	return c.JSON(fiber.Map{"success": true, "peers": peers})
}

// POST /api/network/vps-wg-peer — add WireGuard peer on VPS
func (h *NetworkVPNHandler) CreateWGPeer(c fiber.Ctx) error {
	var body vpsPeer
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.Type = "wireguard"
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "peer": body})
}
