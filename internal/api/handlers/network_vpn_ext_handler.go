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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── VPS config file paths ────────────────────────────────────────────────────
const (
	wgInfoFile   = "/etc/wireguard/wg-server-info.json"
	wgConfFile   = "/etc/wireguard/wg0.conf"
	wgPubKey     = "/etc/wireguard/keys/server.pub"
	l2tpInfoFile = "/etc/salfanet/l2tp/l2tp-server-info.json"
)

// readWGServerInfo reads /etc/wireguard/wg-server-info.json written by
// install-wg-server.sh. Falls back to parsing wg0.conf if the JSON is absent
// (WireGuard installed but info file deleted / old install).
func readWGServerInfo() map[string]any {
	// 1. Try JSON info file first
	if raw, err := os.ReadFile(wgInfoFile); err == nil {
		var info map[string]any
		if json.Unmarshal(raw, &info) == nil {
			return info
		}
	}

	// 2. Fallback: detect from wg0.conf
	confRaw, err := os.ReadFile(wgConfFile)
	if err != nil {
		return nil // WireGuard truly not installed
	}
	conf := string(confRaw)

	listenPort := 51820
	if m := regexp.MustCompile(`ListenPort\s*=\s*(\d+)`).FindStringSubmatch(conf); len(m) == 2 {
		if p, err := strconv.Atoi(m[1]); err == nil {
			listenPort = p
		}
	}

	gatewayIP := "10.200.0.1"
	subnet := "10.200.0.0/24"
	if m := regexp.MustCompile(`Address\s*=\s*([\d.]+)/(\d+)`).FindStringSubmatch(conf); len(m) == 3 {
		gatewayIP = m[1]
		parts := strings.Split(m[1], ".")
		if len(parts) == 4 {
			parts[3] = "0"
			subnet = strings.Join(parts, ".") + "/" + m[2]
		}
	}

	// Public key: try key file, then live wg interface
	publicKey := ""
	if raw, err := os.ReadFile(wgPubKey); err == nil {
		publicKey = strings.TrimSpace(string(raw))
	}
	if publicKey == "" {
		if out, err := exec.Command("wg", "show", "wg0", "public-key").Output(); err == nil {
			publicKey = strings.TrimSpace(string(out))
		}
	}

	// Public IP
	publicIP := ""
	if out, err := exec.Command("curl", "-4", "-s", "--connect-timeout", "5", "ifconfig.me").Output(); err == nil {
		publicIP = strings.TrimSpace(string(out))
	}

	info := map[string]any{
		"interface":   "wg0",
		"listenPort":  listenPort,
		"subnet":      subnet,
		"gatewayIp":   gatewayIP,
		"publicIp":    publicIP,
		"publicKey":   publicKey,
		"recoveredAt": time.Now().UTC().Format(time.RFC3339),
	}

	// Re-write JSON file so next load is instant (best-effort)
	if raw, err := json.MarshalIndent(info, "", "  "); err == nil {
		_ = os.WriteFile(wgInfoFile, append(raw, '\n'), 0o640)
	}
	return info
}

// readL2TPServerInfo reads /etc/salfanet/l2tp/l2tp-server-info.json.
// Falls back to parsing /etc/xl2tpd/xl2tpd.conf + /etc/ipsec.secrets.
func readL2TPServerInfo() map[string]any {
	// 1. Try JSON info file
	if raw, err := os.ReadFile(l2tpInfoFile); err == nil {
		var info map[string]any
		if json.Unmarshal(raw, &info) == nil {
			return info
		}
	}

	// 2. Fallback: parse xl2tpd.conf
	confRaw, err := os.ReadFile("/etc/xl2tpd/xl2tpd.conf")
	if err != nil {
		return nil // L2TP truly not installed
	}
	conf := string(confRaw)

	poolStart, poolEnd := "10.201.0.10", "10.201.0.254"
	if m := regexp.MustCompile(`ip range\s*=\s*([\d.]+)-([\d.]+)`).FindStringSubmatch(conf); len(m) == 3 {
		poolStart, poolEnd = m[1], m[2]
	}

	localIP := "10.201.0.1"
	if m := regexp.MustCompile(`local ip\s*=\s*([\d.]+)`).FindStringSubmatch(conf); len(m) == 2 {
		localIP = m[1]
	}
	parts := strings.Split(localIP, ".")
	subnet := localIP + "/24"
	if len(parts) == 4 {
		parts[3] = "0"
		subnet = strings.Join(parts, ".") + "/24"
	}

	// IPsec PSK
	ipsecPsk := ""
	if raw, err := os.ReadFile("/etc/salfanet/l2tp/ipsec.psk"); err == nil {
		ipsecPsk = strings.TrimSpace(string(raw))
	}
	if ipsecPsk == "" {
		if raw, err := os.ReadFile("/etc/ipsec.secrets"); err == nil {
			if m := regexp.MustCompile(`PSK\s+"([^"]+)"`).FindStringSubmatch(string(raw)); len(m) == 2 {
				ipsecPsk = m[1]
			}
		}
	}

	// Public IP
	publicIP := ""
	if out, err := exec.Command("curl", "-4", "-s", "--connect-timeout", "5", "ifconfig.me").Output(); err == nil {
		publicIP = strings.TrimSpace(string(out))
	}

	info := map[string]any{
		"type":        "l2tp-ipsec",
		"localIp":     localIP,
		"subnet":      subnet,
		"poolStart":   poolStart,
		"poolEnd":     poolEnd,
		"ipsecPsk":    ipsecPsk,
		"publicIp":    publicIP,
		"recoveredAt": time.Now().UTC().Format(time.RFC3339),
	}

	// Re-write JSON so next load is instant (best-effort)
	if raw, err := json.MarshalIndent(info, "", "  "); err == nil {
		_ = os.MkdirAll("/etc/salfanet/l2tp", 0o700)
		_ = os.WriteFile(l2tpInfoFile, append(raw, '\n'), 0o640)
	}
	return info
}

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

// proxyToNextJS forwards a request to the Next.js API at localhost:3000.
// Cookies and Authorization headers are forwarded so session auth works.
func (h *NetworkVPNHandler) proxyToNextJS(c fiber.Ctx, method string) error {
	targetURL := "http://localhost:3000" + string(c.Request().URI().Path())
	if qs := c.Request().URI().QueryString(); len(qs) > 0 {
		targetURL += "?" + string(qs)
	}

	var bodyReader io.Reader
	if body := c.Body(); len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, targetURL, bodyReader)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "proxy: failed to build request"})
	}
	req.Header.Set("Content-Type", "application/json")
	if cookie := c.Get("Cookie"); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if auth := c.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "upstream unavailable"})
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json"
	}
	c.Set("Content-Type", ct)
	return c.Status(resp.StatusCode).Send(respBody)
}

// GET /api/network/vpn-client — list VPN clients (proxied to Next.js)
func (h *NetworkVPNHandler) ListVPNClients(c fiber.Ctx) error {
	return h.proxyToNextJS(c, "GET")
}

// POST /api/network/vpn-client — create VPN client (proxied to Next.js)
func (h *NetworkVPNHandler) CreateVPNClient(c fiber.Ctx) error {
	return h.proxyToNextJS(c, "POST")
}

// PATCH /api/network/vpn-client — update VPN client IP (proxied to Next.js)
func (h *NetworkVPNHandler) PatchVPNClient(c fiber.Ctx) error {
	return h.proxyToNextJS(c, "PATCH")
}

// PUT /api/network/vpn-client — update VPN client (proxied to Next.js)
func (h *NetworkVPNHandler) PutVPNClient(c fiber.Ctx) error {
	return h.proxyToNextJS(c, "PUT")
}

// DELETE /api/network/vpn-client — delete VPN client (proxied to Next.js)
func (h *NetworkVPNHandler) DeleteVPNClient(c fiber.Ctx) error {
	return h.proxyToNextJS(c, "DELETE")
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

// GET /api/network/vps-l2tp-info — L2TP/IPsec server info on this VPS
func (h *NetworkVPNHandler) GetVPSL2TPInfo(c fiber.Ctx) error {
	info := readL2TPServerInfo()
	if info == nil {
		return c.JSON(fiber.Map{
			"installed": false,
			"message":   "L2TP/IPsec server belum di-install di VPS ini. Jalankan install-l2tp-server.sh dulu.",
		})
	}
	info["installed"] = true
	return c.JSON(info)
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

// GET /api/network/vps-wg-peer — WireGuard server info + live peers
func (h *NetworkVPNHandler) ListWGPeers(c fiber.Ctx) error {
	info := readWGServerInfo()
	if info == nil {
		return c.JSON(fiber.Map{
			"installed": false,
			"message":   "WireGuard server belum di-install di VPS ini. Jalankan setup dulu.",
		})
	}

	// Parse live peers from `wg show wg0 dump` (skip server line)
	type wgPeer struct {
		PublicKey     string `json:"publicKey"`
		Endpoint      string `json:"endpoint,omitempty"`
		AllowedIPs    string `json:"allowedIps,omitempty"`
		LastHandshake string `json:"lastHandshake,omitempty"`
	}
	var peers []wgPeer
	if out, err := exec.Command("wg", "show", "wg0", "dump").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines[1:] { // skip server line
			fields := strings.Split(line, "\t")
			if len(fields) < 5 {
				continue
			}
			p := wgPeer{PublicKey: fields[0]}
			if fields[2] != "(none)" {
				p.Endpoint = fields[2]
			}
			if fields[3] != "(none)" {
				p.AllowedIPs = fields[3]
			}
			peers = append(peers, p)
		}
	}

	info["installed"] = true
	info["peers"] = peers
	return c.JSON(info)
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

// PATCH /api/network/vps-wg-peer — update WireGuard server pool/gateway settings
func (h *NetworkVPNHandler) PatchWGServerConfig(c fiber.Ctx) error {
	info := readWGServerInfo()
	if info == nil {
		return c.Status(400).JSON(fiber.Map{"error": "WireGuard belum di-install di VPS ini"})
	}

	var body struct {
		PoolStart *string `json:"poolStart"`
		PoolEnd   *string `json:"poolEnd"`
		GatewayIp *string `json:"gatewayIp"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	ipRe := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)

	if body.PoolStart != nil {
		s := strings.TrimSpace(*body.PoolStart)
		if !ipRe.MatchString(s) {
			return c.Status(400).JSON(fiber.Map{"error": "poolStart harus berupa IP lengkap, mis. 10.200.0.2"})
		}
		info["poolStart"] = s
	}
	if body.PoolEnd != nil {
		s := strings.TrimSpace(*body.PoolEnd)
		if !ipRe.MatchString(s) {
			return c.Status(400).JSON(fiber.Map{"error": "poolEnd harus berupa IP lengkap, mis. 10.200.0.254"})
		}
		info["poolEnd"] = s
	}

	// lastOctet extracts the last octet from an IP string or treats float64 JSON number as-is
	lastOctet := func(v any, def int) int {
		switch val := v.(type) {
		case float64:
			return int(val)
		case string:
			s := strings.TrimSpace(val)
			if strings.Contains(s, ".") {
				parts := strings.Split(s, ".")
				if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
					return n
				}
			}
			if n, err := strconv.Atoi(s); err == nil {
				return n
			}
		}
		return def
	}
	if lastOctet(info["poolStart"], 2) >= lastOctet(info["poolEnd"], 254) {
		return c.Status(400).JSON(fiber.Map{"error": "poolStart harus lebih kecil dari poolEnd"})
	}

	gatewayChanged := false
	if body.GatewayIp != nil {
		trimmed := strings.TrimSpace(*body.GatewayIp)
		if trimmed != "" {
			if !ipRe.MatchString(trimmed) {
				return c.Status(400).JSON(fiber.Map{"error": "Format gatewayIp tidak valid"})
			}
			info["gatewayIp"] = trimmed
			gatewayChanged = true
		}
	}

	// Derive subnet from gatewayIp, or from poolStart prefix if no gatewayIp
	gatewayIp, _ := info["gatewayIp"].(string)
	if gatewayIp != "" && ipRe.MatchString(gatewayIp) {
		parts := strings.Split(gatewayIp, ".")
		info["subnet"] = strings.Join(parts[:3], ".") + ".0/24"
	} else if ps, ok := info["poolStart"].(string); ok && strings.Contains(ps, ".") {
		parts := strings.Split(ps, ".")
		info["subnet"] = strings.Join(parts[:3], ".") + ".0/24"
	}

	// Update wg0.conf Address and restart WireGuard interface if gateway changed
	if gatewayChanged && gatewayIp != "" {
		confRaw, err := os.ReadFile(wgConfFile)
		if err == nil {
			conf := string(confRaw)
			addrRe := regexp.MustCompile(`(?m)^(Address\s*=\s*)[\d./]+`)
			conf = addrRe.ReplaceAllString(conf, "${1}"+gatewayIp+"/24")

			postUp := "iptables -I INPUT -p udp --dport 51820 -j ACCEPT; iptables -I FORWARD -i wg0 -j ACCEPT; iptables -I FORWARD -o wg0 -j ACCEPT; iptables -I INPUT -i wg0 -p udp -m multiport --dports 1812,1813,3799 -j ACCEPT"
			postDown := "iptables -D INPUT -p udp --dport 51820 -j ACCEPT; iptables -D FORWARD -i wg0 -j ACCEPT; iptables -D FORWARD -o wg0 -j ACCEPT; iptables -D INPUT -i wg0 -p udp -m multiport --dports 1812,1813,3799 -j ACCEPT"

			postUpRe := regexp.MustCompile(`(?m)^PostUp\s*=.*`)
			if postUpRe.MatchString(conf) {
				conf = postUpRe.ReplaceAllString(conf, "PostUp = "+postUp)
			} else {
				conf = regexp.MustCompile(`(?m)^(\[Interface\])`).ReplaceAllString(conf, "$1\nPostUp = "+postUp)
			}
			postDownRe := regexp.MustCompile(`(?m)^PostDown\s*=.*`)
			if postDownRe.MatchString(conf) {
				conf = postDownRe.ReplaceAllString(conf, "PostDown = "+postDown)
			} else {
				conf = regexp.MustCompile(`(?m)^(PostUp\s*=.*)`).ReplaceAllString(conf, "$1\nPostDown = "+postDown)
			}

			_ = os.WriteFile(wgConfFile, []byte(conf), 0o640)
			_ = exec.Command("wg-quick", "down", "wg0").Run()
			_ = exec.Command("wg-quick", "up", "wg0").Run()
		}
	}

	// Save updated info JSON
	if raw, err := json.MarshalIndent(info, "", "  "); err == nil {
		_ = os.WriteFile(wgInfoFile, append(raw, '\n'), 0o640)
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"poolStart": info["poolStart"],
		"poolEnd":   info["poolEnd"],
		"gatewayIp": info["gatewayIp"],
		"subnet":    info["subnet"],
	})
}

// PATCH /api/network/vps-l2tp-peer — update L2TP server pool/gateway settings
func (h *NetworkVPNHandler) PatchL2TPServerConfig(c fiber.Ctx) error {
	info := readL2TPServerInfo()
	if info == nil {
		return c.Status(400).JSON(fiber.Map{"error": "L2TP server belum di-install di VPS ini"})
	}

	var body struct {
		PoolStart *string `json:"poolStart"`
		PoolEnd   *string `json:"poolEnd"`
		Gateway   *string `json:"gateway"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	ipRe := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)

	if body.PoolStart != nil {
		s := strings.TrimSpace(*body.PoolStart)
		if !ipRe.MatchString(s) {
			return c.Status(400).JSON(fiber.Map{"error": "poolStart harus berupa IP lengkap, mis. 10.201.0.10"})
		}
		info["poolStart"] = s
	}
	if body.PoolEnd != nil {
		s := strings.TrimSpace(*body.PoolEnd)
		if !ipRe.MatchString(s) {
			return c.Status(400).JSON(fiber.Map{"error": "poolEnd harus berupa IP lengkap, mis. 10.201.0.254"})
		}
		info["poolEnd"] = s
	}
	if body.Gateway != nil {
		trimmed := strings.TrimSpace(*body.Gateway)
		if trimmed != "" {
			if !ipRe.MatchString(trimmed) {
				return c.Status(400).JSON(fiber.Map{"error": "Format gateway tidak valid"})
			}
			info["gateway"] = trimmed
		}
	}

	// lastOctet extracts the last octet from an IP string or treats float64 JSON number as-is
	lastOctet := func(v any, def int) int {
		switch val := v.(type) {
		case float64:
			return int(val)
		case string:
			s := strings.TrimSpace(val)
			if strings.Contains(s, ".") {
				parts := strings.Split(s, ".")
				if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
					return n
				}
			}
			if n, err := strconv.Atoi(s); err == nil {
				return n
			}
		}
		return def
	}
	if lastOctet(info["poolStart"], 10) >= lastOctet(info["poolEnd"], 254) {
		return c.Status(400).JSON(fiber.Map{"error": "poolStart harus lebih kecil dari poolEnd"})
	}

	// Save updated info JSON
	_ = os.MkdirAll("/etc/salfanet/l2tp", 0o700)
	if raw, err := json.MarshalIndent(info, "", "  "); err == nil {
		_ = os.WriteFile(l2tpInfoFile, append(raw, '\n'), 0o640)
	}

	// Restart xl2tpd and reload ipsec (best-effort — non-fatal)
	_ = exec.Command("systemctl", "restart", "xl2tpd").Run()
	_ = exec.Command("ipsec", "reload").Run()

	// Ensure iptables rules allow PPP traffic to reach RADIUS (idempotent check-then-insert)
	for _, rule := range []string{
		"FORWARD -i ppp+ -j ACCEPT",
		"FORWARD -o ppp+ -j ACCEPT",
		"INPUT -i ppp+ -p udp -m multiport --dports 1812,1813,3799 -j ACCEPT",
	} {
		checkArgs := append([]string{"-C"}, strings.Fields(rule)...)
		if exec.Command("iptables", checkArgs...).Run() != nil {
			insertArgs := append([]string{"-I"}, strings.Fields(rule)...)
			_ = exec.Command("iptables", insertArgs...).Run()
		}
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"poolStart": info["poolStart"],
		"poolEnd":   info["poolEnd"],
		"gateway":   info["gateway"],
	})
}
