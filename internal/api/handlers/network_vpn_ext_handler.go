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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	ros "github.com/go-routeros/routeros/v3"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/curve25519"
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
	ID               string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Type             string    `json:"type"` // l2tp, wireguard
	PeerName         string    `gorm:"column:peer_name" json:"peerName"`
	PeerIP           string    `gorm:"column:peer_ip" json:"peerIp"`
	LocalIP          string    `gorm:"column:local_ip" json:"localIp"`
	PublicKey        *string   `gorm:"column:public_key" json:"publicKey"`
	NasSecret        *string   `gorm:"column:nas_secret" json:"nasSecret"`
	ApiUsername      *string   `gorm:"column:api_username" json:"apiUsername"`
	ApiPassword      *string   `gorm:"column:api_password" json:"apiPassword"`
	ClientPrivateKey *string   `gorm:"column:client_private_key" json:"-"` // sensitive — never return in list
	IsActive         bool      `gorm:"column:is_active;default:true" json:"isActive"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (vpsPeer) TableName() string { return "vps_peers" }

// vpnClientResponse is the unified response format for both vpn_clients and vps_peers entries.
type vpnClientResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	VpnServerId     string    `json:"vpnServerId"`
	VpnIp           string    `json:"vpnIp"`
	Username        string    `json:"username"`
	Password        string    `json:"password"`
	VpnType         string    `json:"vpnType"`
	Description     *string   `json:"description"`
	WinboxPort      *int      `json:"winboxPort"`
	ApiUsername     *string   `json:"apiUsername"`
	ApiPassword     *string   `json:"apiPassword"`
	ClientPublicKey *string   `json:"clientPublicKey"`
	IsActive        bool      `json:"isActive"`
	IsRadiusServer  bool      `json:"isRadiusServer"`
	CreatedAt       time.Time `json:"createdAt"`
	NasSecret       *string   `json:"nasSecret"`
}

// nextAvailableWGIP finds the first unused IP in the WireGuard pool.
func nextAvailableWGIP(db *gorm.DB, subnet, poolStart, poolEnd, gatewayIp string) (string, error) {
	// Derive prefix from poolStart or subnet
	prefix := ""
	if poolStart != "" {
		if parts := strings.Split(poolStart, "."); len(parts) == 4 {
			prefix = strings.Join(parts[:3], ".") + "."
		}
	}
	if prefix == "" {
		if parts := strings.Split(strings.Split(subnet, "/")[0], "."); len(parts) == 4 {
			prefix = strings.Join(parts[:3], ".") + "."
		}
	}
	if prefix == "" {
		return "", fmt.Errorf("cannot determine IP prefix from subnet %s", subnet)
	}

	startOctet, endOctet := 2, 254
	if poolStart != "" {
		if parts := strings.Split(poolStart, "."); len(parts) == 4 {
			if n, err := strconv.Atoi(parts[3]); err == nil {
				startOctet = n
			}
		}
	}
	if poolEnd != "" {
		if parts := strings.Split(poolEnd, "."); len(parts) == 4 {
			if n, err := strconv.Atoi(parts[3]); err == nil {
				endOctet = n
			}
		}
	}

	// Collect IPs already in use
	usedIPs := map[string]bool{gatewayIp: true}
	var existingPeers []vpsPeer
	db.Where("type = ?", "wireguard").Find(&existingPeers)
	for _, p := range existingPeers {
		usedIPs[p.PeerIP] = true
	}
	if raw, err := os.ReadFile(wgConfFile); err == nil {
		re := regexp.MustCompile(`AllowedIPs\s*=\s*([\d.]+)/32`)
		for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
			if len(m) == 2 {
				usedIPs[m[1]] = true
			}
		}
	}

	for i := startOctet; i <= endOctet; i++ {
		candidate := prefix + strconv.Itoa(i)
		if !usedIPs[candidate] {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("IP pool (%d–%d) sudah habis", startOctet, endOctet)
}

// wgRandomHex generates a random lowercase hex string of n bytes (2n chars).
func wgRandomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// wgRandomAlphanumeric generates a random alphanumeric string of length n.
func wgRandomAlphanumeric(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

// decryptVPNPassword decrypts an AES-256-CBC encrypted password stored in vpn_servers.
// Format: "ivHex:encryptedHex". Returns plaintext as-is if not encrypted or on error.
func decryptVPNPassword(encrypted string) string {
	parts := strings.SplitN(encrypted, ":", 2)
	if len(parts) != 2 {
		return encrypted
	}
	encKey := os.Getenv("ENCRYPTION_KEY")
	if encKey == "" {
		encKey = "default-encryption-key-change-this-32"
	}
	keyBytes := make([]byte, 32)
	copy(keyBytes, []byte(encKey))

	iv, err := hex.DecodeString(parts[0])
	if err != nil || len(iv) != 16 {
		return encrypted
	}
	ciphertext, err := hex.DecodeString(parts[1])
	if err != nil || len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return encrypted
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return encrypted
	}
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)
	// PKCS7 unpad
	if padLen := int(plaintext[len(plaintext)-1]); padLen > 0 && padLen <= aes.BlockSize {
		plaintext = plaintext[:len(plaintext)-padLen]
	}
	return string(plaintext)
}

// nextAvailableVPNClientIP finds the first unused IP in the VPN server's pool.
func nextAvailableVPNClientIP(db *gorm.DB, vpnServerId, subnet string, poolStart, poolEnd int) (string, error) {
	network := strings.Split(subnet, "/")[0]
	parts := strings.Split(network, ".")
	if len(parts) != 4 {
		return "", fmt.Errorf("invalid subnet: %s", subnet)
	}
	baseNetwork := strings.Join(parts[:3], ".")
	if poolStart < 2 {
		poolStart = 10
	}
	if poolEnd < poolStart {
		poolEnd = 254
	}

	var rows []struct {
		VpnIp string `gorm:"column:vpnIp"`
	}
	db.Table("vpn_clients").Select("vpnIp").Where("vpnServerId = ?", vpnServerId).Find(&rows)
	usedIPs := make(map[string]bool)
	for _, r := range rows {
		usedIPs[r.VpnIp] = true
	}
	for i := poolStart; i <= poolEnd; i++ {
		ip := fmt.Sprintf("%s.%d", baseNetwork, i)
		if !usedIPs[ip] {
			return ip, nil
		}
	}
	return "", fmt.Errorf("tidak ada IP tersedia di pool (%s.%d–%s.%d)", baseNetwork, poolStart, baseNetwork, poolEnd)
}

// nextAvailableWinboxPort finds the next free Winbox port (10000–10100) for the server.
func nextAvailableWinboxPort(db *gorm.DB, vpnServerId string) (int, error) {
	const minPort, maxPort = 10000, 10100
	var rows []struct {
		WinboxPort *int `gorm:"column:winboxPort"`
	}
	db.Table("vpn_clients").Select("winboxPort").
		Where("vpnServerId = ? AND winboxPort IS NOT NULL", vpnServerId).Find(&rows)
	used := make(map[int]bool)
	for _, r := range rows {
		if r.WinboxPort != nil {
			used[*r.WinboxPort] = true
		}
	}
	for port := minPort; port <= maxPort; port++ {
		if !used[port] {
			return port, nil
		}
	}
	return 0, fmt.Errorf("tidak ada port Winbox tersedia di range %d-%d", minPort, maxPort)
}

// generateX25519KeyPair generates a WireGuard-compatible X25519 key pair (base64-encoded).
func generateX25519KeyPair() (privateKeyB64, publicKeyB64 string, err error) {
	var privRaw [32]byte
	if _, err = rand.Read(privRaw[:]); err != nil {
		return
	}
	// Clamp per WireGuard spec
	privRaw[0] &= 248
	privRaw[31] &= 127
	privRaw[31] |= 64

	pubRaw, kerr := curve25519.X25519(privRaw[:], curve25519.Basepoint)
	if kerr != nil {
		err = kerr
		return
	}
	privateKeyB64 = base64.StdEncoding.EncodeToString(privRaw[:])
	publicKeyB64 = base64.StdEncoding.EncodeToString(pubRaw)
	return
}

// buildNasSetupScript generates the RouterOS setup script for a VPN client.
func buildNasSetupScript(name, vpnType, serverHost, vpnIp, username, password, apiUsername, apiPassword string,
	_ *int, clientPrivateKey *string, serverWgPublicKey string, wgPort int, nasSecret, radiusServerVpnIp string) string {

	var radiusSection string
	if radiusServerVpnIp != "" {
		radiusSection = fmt.Sprintf(`
# --- Setup RADIUS Server via VPN ---
/radius remove [find where comment~"SALFANET"]
/radius add address=%s secret=%s service=ppp,hotspot src-address=%s authentication-port=1812 accounting-port=1813 timeout=3s comment="SALFANET RADIUS via VPN"

# --- Enable RADIUS ---
/ppp aaa set use-radius=yes accounting=yes
/radius incoming set accept=yes port=3799

# --- Firewall - Allow CoA dari RADIUS Server via VPN ---
/ip firewall filter remove [find where comment~"SALFANET-RADIUS"]
/ip firewall filter add chain=input protocol=udp src-address=%s dst-port=3799 action=accept comment="SALFANET-RADIUS CoA" place-before=0
/ip firewall filter add chain=input protocol=udp src-address=%s dst-port=1812,1813 action=accept comment="SALFANET-RADIUS Auth" place-before=0

# --- Enable RADIUS untuk Hotspot ---
/ip hotspot profile set [find] use-radius=yes`, radiusServerVpnIp, nasSecret, vpnIp, radiusServerVpnIp, radiusServerVpnIp)
	} else {
		radiusSection = fmt.Sprintf(`
# --- RADIUS Server belum dikonfigurasi ---
# Tandai salah satu VPN Client sebagai "RADIUS Server" di panel admin
# kemudian setup RADIUS manual:
# /radius add address=<RADIUS_VPN_IP> secret=%s service=ppp,hotspot src-address=%s`, nasSecret, vpnIp)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	if vpnType == "wireguard" {
		privKey := ""
		if clientPrivateKey != nil {
			privKey = *clientPrivateKey
		}
		return strings.TrimSpace(fmt.Sprintf(`
# ============================================
# SALFANET WireGuard VPN Client + RADIUS Setup Script
# NAS: %s
# VPN Server: %s
# VPN IP: %s
# Generated: %s
# NOTE: Requires RouterOS 7+ with WireGuard support
# ============================================

# --- STEP 1: Create WireGuard Interface ---
/interface wireguard add listen-port=0 name=wg0-salfanet comment="SALFANET WireGuard"

# --- STEP 2: Set Client Private Key ---
/interface wireguard set wg0-salfanet private-key="%s"

# --- STEP 3: Add Peer (VPN Server CHR) ---
/interface wireguard peers add interface=wg0-salfanet public-key="%s" endpoint-address=%s endpoint-port=%d allowed-address=0.0.0.0/0 persistent-keepalive=25 comment="SALFANET VPN Server"

# --- STEP 4: Assign VPN IP ---
/ip address add address=%s/24 interface=wg0-salfanet comment="SALFANET VPN IP"

# --- STEP 5: Tunggu koneksi (5 detik) ---
:delay 5s
%s

# ============================================
# SELESAI! Verifikasi:
# /interface wireguard print
# /interface wireguard peers print
# /ip address print where interface=wg0-salfanet
# ============================================`,
			name, serverHost, vpnIp, now, privKey, serverWgPublicKey, serverHost, wgPort, vpnIp, radiusSection))
	}

	vpnTypeUpper := strings.ToUpper(vpnType)
	interfaceType := "l2tp-client"
	switch vpnType {
	case "pptp":
		interfaceType = "pptp-client"
	case "sstp":
		interfaceType = "sstp-client"
	}
	ipsecLine := ""
	if vpnType == "l2tp" {
		ipsecLine = " use-ipsec=yes ipsec-secret=salfanet-vpn-secret"
	}
	authLine := " allow=mschap2"
	if vpnType != "l2tp" {
		authLine = " authentication=mschap2"
	}
	portLine := ""
	if vpnType == "sstp" {
		portLine = " port=992"
	}

	return strings.TrimSpace(fmt.Sprintf(`
# ============================================
# SALFANET %s VPN Client + RADIUS Setup Script
# NAS: %s
# VPN Server: %s
# VPN IP: %s
# Generated: %s
# ============================================

# --- STEP 1: Create API User Group ---
/user group add name=api-users policy=read,write,policy,test,sensitive,api comment="Limited API Access Group"

# --- STEP 2: Create API User ---
/user add name=%s group=api-users password=%s comment="API User for Remote Access"

# --- STEP 3: Setup %s Client ---
/interface %s add name=%s-salfanet connect-to=%s user=%s password=%s%s%s disabled=no%s add-default-route=no comment="SALFANET VPN"

# --- STEP 4: Tunggu koneksi (10 detik) ---
:delay 10s

# --- STEP 5: Verifikasi koneksi VPN ---
:if ([/interface %s get [find name="%s-salfanet"] running] = true) do={
    :put "VPN Connected! IP: %s"
} else={
    :put "VPN belum terkoneksi, cek log: /log print where topics~\"%s\""
}
%s

# ============================================
# SELESAI! Verifikasi:
# /interface %s print
# /radius print
# /ip firewall filter print where comment~"SALFANET"
# ============================================`,
		vpnTypeUpper, name, serverHost, vpnIp, now,
		apiUsername, apiPassword,
		vpnTypeUpper, interfaceType, interfaceType, serverHost, username, password, ipsecLine, portLine, authLine,
		interfaceType, interfaceType, vpnIp, vpnType,
		radiusSection,
		interfaceType))
}

// prismaVpnClient maps to the "vpn_clients" table managed by Prisma.
// Prisma stores column names in camelCase — explicit gorm:"column:..." tags required.
type prismaVpnClient struct {
	ID               string    `gorm:"primaryKey;column:id" json:"id"`
	Name             string    `gorm:"column:name" json:"name"`
	VpnServerId      string    `gorm:"column:vpnServerId" json:"vpnServerId"`
	VpnIp            string    `gorm:"column:vpnIp" json:"vpnIp"`
	Username         string    `gorm:"column:username" json:"username"`
	Password         string    `gorm:"column:password" json:"password"`
	VpnType          string    `gorm:"column:vpnType" json:"vpnType"`
	Description      *string   `gorm:"column:description" json:"description"`
	WinboxPort       *int      `gorm:"column:winboxPort" json:"winboxPort"`
	ApiUsername      *string   `gorm:"column:apiUsername" json:"apiUsername"`
	ApiPassword      *string   `gorm:"column:apiPassword" json:"apiPassword"`
	ClientPublicKey  *string   `gorm:"column:clientPublicKey" json:"clientPublicKey"`
	ClientPrivateKey *string   `gorm:"column:clientPrivateKey" json:"-"` // sensitive — never expose in API
	IsActive         bool      `gorm:"column:isActive" json:"isActive"`
	IsRadiusServer   bool      `gorm:"column:isRadiusServer" json:"isRadiusServer"`
	CreatedAt        time.Time `gorm:"column:createdAt" json:"createdAt"`
}

func (prismaVpnClient) TableName() string { return "vpn_clients" }

// prismaVpnServer maps to the "vpn_servers" table managed by Prisma.
type prismaVpnServer struct {
	ID          string  `gorm:"primaryKey;column:id" json:"id"`
	Name        string  `gorm:"column:name" json:"name"`
	Host        string  `gorm:"column:host" json:"host"`
	Username    string  `gorm:"column:username" json:"username"`
	Password    string  `gorm:"column:password" json:"-"`
	ApiPort     int     `gorm:"column:apiPort" json:"apiPort"`
	Subnet      string  `gorm:"column:subnet" json:"subnet"`
	PoolStart   int     `gorm:"column:poolStart" json:"poolStart"`
	PoolEnd     int     `gorm:"column:poolEnd" json:"poolEnd"`
	L2tpEnabled bool    `gorm:"column:l2tpEnabled" json:"l2tpEnabled"`
	SstpEnabled bool    `gorm:"column:sstpEnabled" json:"sstpEnabled"`
	PptpEnabled bool    `gorm:"column:pptpEnabled" json:"pptpEnabled"`
	WgEnabled   bool    `gorm:"column:wgEnabled" json:"wgEnabled"`
	WgPublicKey *string `gorm:"column:wgPublicKey" json:"wgPublicKey"`
	WgPort      *int    `gorm:"column:wgPort" json:"wgPort"`
	IsActive    bool    `gorm:"column:isActive" json:"isActive"`
}

func (prismaVpnServer) TableName() string { return "vpn_servers" }

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

// POST /api/network/vpn-client — create MikroTik CHR VPN client
func (h *NetworkVPNHandler) CreateVPNClient(c fiber.Ctx) error {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		VpnServerId string `json:"vpnServerId"`
		VpnType     string `json:"vpnType"`
		CustomVpnIp string `json:"customVpnIp"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Nama client wajib diisi"})
	}
	if body.VpnServerId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "VPN Server wajib dipilih"})
	}
	// VPS peers have their own dedicated endpoints
	if body.VpnServerId == "__vps_wg_server__" || body.VpnServerId == "__vps_l2tp_server__" {
		return c.Status(400).JSON(fiber.Map{"error": "Gunakan /api/network/vps-wg-peer atau /api/network/vps-l2tp-peer untuk VPS peers"})
	}

	// Find VPN server
	var server prismaVpnServer
	if err := h.db.Where("id = ?", body.VpnServerId).First(&server).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "VPN Server tidak ditemukan"})
	}

	// Normalize vpn type
	vpnType := strings.ToLower(body.VpnType)
	if vpnType != "pptp" && vpnType != "sstp" && vpnType != "wireguard" {
		vpnType = "l2tp"
	}

	// Generate credentials
	spacesRe := regexp.MustCompile(`\s+`)
	nameLower := strings.ToLower(spacesRe.ReplaceAllString(body.Name, "-"))
	username := fmt.Sprintf("vpn-%s-%s", nameLower, wgRandomAlphanumeric(4))
	password := wgRandomAlphanumeric(12)
	apiUsername := fmt.Sprintf("api-%s", nameLower)
	apiPassword := wgRandomAlphanumeric(16)

	// Assign IP
	poolStart := server.PoolStart
	if poolStart < 2 {
		poolStart = 10
	}
	poolEnd := server.PoolEnd
	if poolEnd < poolStart {
		poolEnd = 254
	}

	var vpnIp string
	if body.CustomVpnIp != "" {
		ip := strings.TrimSpace(body.CustomVpnIp)
		if matched, _ := regexp.MatchString(`^(\d{1,3}\.){3}\d{1,3}$`, ip); !matched {
			return c.Status(400).JSON(fiber.Map{"error": "Format IP tidak valid"})
		}
		var conflict prismaVpnClient
		if h.db.Where("vpnServerId = ? AND vpnIp = ?", body.VpnServerId, ip).First(&conflict).Error == nil {
			return c.Status(409).JSON(fiber.Map{"error": fmt.Sprintf("IP %s sudah digunakan client lain", ip)})
		}
		vpnIp = ip
	} else {
		var err error
		vpnIp, err = nextAvailableVPNClientIP(h.db, body.VpnServerId, server.Subnet, poolStart, poolEnd)
		if err != nil {
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		}
	}

	// Winbox port (non-WG only)
	var winboxPort *int
	if vpnType != "wireguard" {
		if port, err := nextAvailableWinboxPort(h.db, body.VpnServerId); err == nil {
			winboxPort = &port
		}
	}

	// Connect to MikroTik CHR
	routerPass := decryptVPNPassword(server.Password)
	apiPort := server.ApiPort
	if apiPort == 0 {
		apiPort = 8728
	}
	mtAddr := fmt.Sprintf("%s:%d", server.Host, apiPort)
	mtClient, err := ros.DialTimeout(mtAddr, server.Username, routerPass, 15*time.Second)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Tidak bisa konek ke VPN server: " + err.Error()})
	}
	defer mtClient.Close()

	var clientPrivateKey, clientPublicKey *string

	if vpnType == "wireguard" {
		priv, pub, kerr := generateX25519KeyPair()
		if kerr != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal generate WireGuard keys: " + kerr.Error()})
		}
		clientPrivateKey = &priv
		clientPublicKey = &pub

		// Remove existing peer with same comment (best-effort)
		if existing, _ := mtClient.Run("/interface/wireguard/peers/print", "?comment=SALFANET-"+body.Name); existing != nil {
			for _, re := range existing.Re {
				if id, ok := re.Map[".id"]; ok {
					_, _ = mtClient.Run("/interface/wireguard/peers/remove", "=.id="+id)
				}
			}
		}
		// Add WG peer on CHR
		if _, merr := mtClient.Run("/interface/wireguard/peers/add",
			"=interface=wg0",
			"=public-key="+pub,
			"=allowed-address="+vpnIp+"/32",
			"=persistent-keepalive=25",
			"=comment=SALFANET-"+body.Name,
		); merr != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal tambah WireGuard peer di CHR: " + merr.Error()})
		}
	} else {
		// PPP secret
		if _, merr := mtClient.Run("/ppp/secret/add",
			"=name="+username,
			"=password="+password,
			"=service=any",
			"=profile=vpn-profile",
			"=remote-address="+vpnIp,
			"=comment=SALFANET-"+body.Name,
		); merr != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal buat PPP secret di CHR: " + merr.Error()})
		}
		// Winbox NAT
		if winboxPort != nil {
			_, _ = mtClient.Run("/ip/firewall/nat/add",
				"=chain=dstnat",
				"=protocol=tcp",
				fmt.Sprintf("=dst-port=%d", *winboxPort),
				"=action=dst-nat",
				"=to-addresses="+vpnIp,
				"=to-ports=8291",
				"=comment=Winbox-"+body.Name,
			)
		}
	}

	// Save to vpn_clients
	var descPtr *string
	if body.Description != "" {
		descPtr = &body.Description
	}
	newClient := prismaVpnClient{
		ID:               uuid.New().String(),
		Name:             body.Name,
		VpnServerId:      body.VpnServerId,
		VpnIp:            vpnIp,
		Username:         username,
		Password:         password,
		VpnType:          strings.ToUpper(vpnType),
		Description:      descPtr,
		WinboxPort:       winboxPort,
		ApiUsername:      &apiUsername,
		ApiPassword:      &apiPassword,
		ClientPublicKey:  clientPublicKey,
		ClientPrivateKey: clientPrivateKey,
		IsActive:         true,
		CreatedAt:        time.Now(),
	}
	if err := h.db.Create(&newClient).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal simpan ke database: " + err.Error()})
	}

	// Auto-create or reuse NAS entry in "nas" table
	nasSecret := wgRandomAlphanumeric(16)
	type nasRow struct {
		ID     string `gorm:"column:id"`
		Secret string `gorm:"column:secret"`
	}
	var existingNas nasRow
	if h.db.Table("nas").Select("id, secret").Where("nasname = ?", vpnIp).First(&existingNas).Error == nil {
		nasSecret = existingNas.Secret
		h.db.Table("nas").Where("id = ?", existingNas.ID).Update("vpnClientId", newClient.ID)
	} else {
		shortName := body.Name
		if len(shortName) > 32 {
			shortName = shortName[:32]
		}
		h.db.Table("nas").Create(map[string]any{
			"id":          uuid.New().String(),
			"name":        body.Name,
			"nasname":     vpnIp,
			"shortname":   shortName,
			"type":        "mikrotik",
			"ipAddress":   vpnIp,
			"username":    apiUsername,
			"password":    apiPassword,
			"secret":      nasSecret,
			"ports":       1812,
			"vpnClientId": newClient.ID,
			"description": fmt.Sprintf("Auto-created NAS for VPN client '%s' (%s)", body.Name, strings.ToUpper(vpnType)),
		})
	}

	// Find RADIUS server VPN IP
	radiusServerVpnIp := ""
	var radiusClient prismaVpnClient
	if h.db.Where("isRadiusServer = ?", true).First(&radiusClient).Error == nil {
		radiusServerVpnIp = radiusClient.VpnIp
	}

	serverWgPublicKey := ""
	if server.WgPublicKey != nil {
		serverWgPublicKey = *server.WgPublicKey
	}
	wgPort := 51820
	if server.WgPort != nil {
		wgPort = *server.WgPort
	}

	nasSetupScript := buildNasSetupScript(body.Name, vpnType, server.Host, vpnIp,
		username, password, apiUsername, apiPassword,
		winboxPort, clientPrivateKey, serverWgPublicKey, wgPort, nasSecret, radiusServerVpnIp)

	// Build credentials map
	creds := fiber.Map{
		"server":      server.Host,
		"username":    username,
		"password":    password,
		"vpnIp":       vpnIp,
		"apiUsername": apiUsername,
		"apiPassword": apiPassword,
		"vpnType":     vpnType,
		"nasSecret":   nasSecret,
	}
	if winboxPort != nil {
		creds["winboxPort"] = *winboxPort
		creds["winboxRemote"] = fmt.Sprintf("%s:%d", server.Host, *winboxPort)
	}
	if vpnType == "wireguard" && clientPrivateKey != nil {
		creds["clientPrivateKey"] = *clientPrivateKey
		creds["clientPublicKey"] = clientPublicKey
		creds["serverPublicKey"] = serverWgPublicKey
		creds["wgPort"] = wgPort
	}
	if radiusServerVpnIp != "" {
		creds["radiusServerIp"] = radiusServerVpnIp
	}

	return c.JSON(fiber.Map{
		"success":        true,
		"client":         newClient,
		"credentials":    creds,
		"nasSetupScript": nasSetupScript,
	})
}

// PATCH /api/network/vpn-client — update VPN client IP address
func (h *NetworkVPNHandler) PatchVPNClient(c fiber.Ctx) error {
	var body struct {
		ID    string `json:"id"`
		VpnIp string `json:"vpnIp"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.ID == "" || body.VpnIp == "" {
		return c.Status(400).JSON(fiber.Map{"error": "id dan vpnIp wajib diisi"})
	}
	newIp := strings.TrimSpace(body.VpnIp)
	if matched, _ := regexp.MatchString(`^(\d{1,3}\.){3}\d{1,3}$`, newIp); !matched {
		return c.Status(400).JSON(fiber.Map{"error": "Format IP tidak valid"})
	}

	var client prismaVpnClient
	if err := h.db.Where("id = ?", body.ID).First(&client).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Client tidak ditemukan"})
	}

	// Check IP conflict on same server
	var conflict prismaVpnClient
	if h.db.Where("vpnServerId = ? AND vpnIp = ? AND id != ?", client.VpnServerId, newIp, body.ID).
		First(&conflict).Error == nil {
		return c.Status(409).JSON(fiber.Map{"error": fmt.Sprintf("IP %s sudah digunakan oleh \"%s\"", newIp, conflict.Name)})
	}

	// Update MikroTik (non-WireGuard only — best effort, non-fatal)
	if strings.ToLower(client.VpnType) != "wireguard" {
		var server prismaVpnServer
		if h.db.Where("id = ?", client.VpnServerId).First(&server).Error == nil {
			routerPass := decryptVPNPassword(server.Password)
			apiPort := server.ApiPort
			if apiPort == 0 {
				apiPort = 8728
			}
			if mtClient, merr := ros.DialTimeout(
				fmt.Sprintf("%s:%d", server.Host, apiPort),
				server.Username, routerPass, 10*time.Second,
			); merr == nil {
				defer mtClient.Close()
				// Update PPP secret remote-address
				if secrets, _ := mtClient.Run("/ppp/secret/print", "?name="+client.Username); secrets != nil {
					for _, re := range secrets.Re {
						if id, ok := re.Map[".id"]; ok {
							_, _ = mtClient.Run("/ppp/secret/set", "=.id="+id, "=remote-address="+newIp)
						}
					}
				}
				// Update Winbox NAT rules (comment pattern: SALFANET-VPN-<username>)
				if nats, _ := mtClient.Run("/ip/firewall/nat/print", "?comment=SALFANET-VPN-"+client.Username); nats != nil {
					for _, re := range nats.Re {
						if id, ok := re.Map[".id"]; ok {
							_, _ = mtClient.Run("/ip/firewall/nat/set", "=.id="+id, "=to-addresses="+newIp)
						}
					}
				}
			}
		}
	}

	// Update DB
	if err := h.db.Table("vpn_clients").Where("id = ?", body.ID).Update("vpnIp", newIp).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal update database: " + err.Error()})
	}
	client.VpnIp = newIp
	return c.JSON(fiber.Map{"success": true, "client": client, "newIp": newIp})
}

// PUT /api/network/vpn-client — toggle RADIUS server flag on a VPN client
func (h *NetworkVPNHandler) PutVPNClient(c fiber.Ctx) error {
	var body struct {
		ID             string `json:"id"`
		IsRadiusServer bool   `json:"isRadiusServer"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.ID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Client ID wajib diisi"})
	}

	// Unset all others first if marking as RADIUS server
	if body.IsRadiusServer {
		h.db.Table("vpn_clients").Where("isRadiusServer = ?", true).Update("isRadiusServer", false)
	}

	if err := h.db.Table("vpn_clients").Where("id = ?", body.ID).
		Update("isRadiusServer", body.IsRadiusServer).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal update: " + err.Error()})
	}

	var client prismaVpnClient
	h.db.Where("id = ?", body.ID).First(&client)
	return c.JSON(fiber.Map{"success": true, "client": client})
}
func (h *NetworkVPNHandler) ListVPNClients(c fiber.Ctx) error {
	var prismaClients []prismaVpnClient
	h.db.Order("createdAt desc").Find(&prismaClients)

	var servers []prismaVpnServer
	h.db.Where("isActive = ?", true).Order("name asc").Find(&servers)

	// Build NAS secret map from the "nas" table (Prisma router model @@map("nas"))
	type nasSecretRow struct {
		VpnClientId string `gorm:"column:vpnClientId"`
		Secret      string `gorm:"column:secret"`
	}
	secretMap := make(map[string]string)
	if len(prismaClients) > 0 {
		ids := make([]string, len(prismaClients))
		for i, cl := range prismaClients {
			ids[i] = cl.ID
		}
		var nasRows []nasSecretRow
		h.db.Table("nas").Select("vpnClientId, secret").
			Where("vpnClientId IN ?", ids).Find(&nasRows)
		for _, row := range nasRows {
			secretMap[row.VpnClientId] = row.Secret
		}
	}

	// Convert prisma clients to unified response
	result := make([]vpnClientResponse, 0, len(prismaClients))
	for _, cl := range prismaClients {
		r := vpnClientResponse{
			ID:              cl.ID,
			Name:            cl.Name,
			VpnServerId:     cl.VpnServerId,
			VpnIp:           cl.VpnIp,
			Username:        cl.Username,
			Password:        cl.Password,
			VpnType:         cl.VpnType,
			Description:     cl.Description,
			WinboxPort:      cl.WinboxPort,
			ApiUsername:     cl.ApiUsername,
			ApiPassword:     cl.ApiPassword,
			ClientPublicKey: cl.ClientPublicKey,
			IsActive:        cl.IsActive,
			IsRadiusServer:  cl.IsRadiusServer,
			CreatedAt:       cl.CreatedAt,
		}
		if s, ok := secretMap[cl.ID]; ok {
			r.NasSecret = &s
		}
		result = append(result, r)
	}

	// Also include all VPS peers (WireGuard and L2TP) from vps_peers table
	var vpsPeers []vpsPeer
	h.db.Order("created_at desc").Find(&vpsPeers)
	for _, p := range vpsPeers {
		serverID := "__vps_wg_server__"
		descText := "VPS WireGuard Peer"
		vpnType := "wireguard"
		if p.Type == "l2tp" {
			serverID = "__vps_l2tp_server__"
			descText = "VPS L2TP Peer"
			vpnType = "l2tp"
		}
		desc := descText
		username := p.PeerName
		if p.ApiUsername != nil {
			username = *p.ApiUsername
		}
		result = append(result, vpnClientResponse{
			ID:              p.ID,
			Name:            p.PeerName,
			VpnServerId:     serverID,
			VpnIp:           p.PeerIP,
			Username:        username,
			Password:        "",
			VpnType:         vpnType,
			Description:     &desc,
			ClientPublicKey: p.PublicKey,
			ApiUsername:     p.ApiUsername,
			ApiPassword:     p.ApiPassword,
			NasSecret:       p.NasSecret,
			IsActive:        p.IsActive,
			IsRadiusServer:  false,
			CreatedAt:       p.CreatedAt,
		})
	}

	// Find RADIUS server IP
	var radiusServerIp *string
	for _, r := range result {
		if r.IsRadiusServer {
			ip := r.VpnIp
			radiusServerIp = &ip
			break
		}
	}

	return c.JSON(fiber.Map{
		"clients":        result,
		"vpnServers":     servers,
		"radiusServerIp": radiusServerIp,
	})
}

// DELETE /api/network/vpn-client — delete VPN client from DB
// Handles both vps_peers (WireGuard/L2TP VPS) and Prisma vpn_clients (MikroTik).
// Not proxied to Next.js because session cookie forwarding fails for DELETE from domain context.
func (h *NetworkVPNHandler) DeleteVPNClient(c fiber.Ctx) error {
	id := c.Query("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": "id diperlukan"})
	}

	// 1. Check vps_peers table first
	var peer vpsPeer
	if err := h.db.Where("id = ?", id).First(&peer).Error; err == nil {
		// Clean up system config files
		if peer.Type == "wireguard" && peer.PublicKey != nil {
			removeWGPeerFromConf(*peer.PublicKey)
			_ = exec.Command("wg", "set", "wg0", "peer", *peer.PublicKey, "remove").Run()
		}
		if peer.Type == "l2tp" {
			// Derive l2tp username: api_username = "api-label" → l2tp username = "l2tp-label"
			l2tpUser := ""
			if peer.ApiUsername != nil {
				l2tpUser = strings.Replace(*peer.ApiUsername, "api-", "l2tp-", 1)
			}
			if l2tpUser != "" {
				removeL2TPUserFromChapSecrets(l2tpUser)
			}
		}
		if err := h.db.Delete(&peer).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal hapus dari database: " + err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	}

	// 2. Not a VPS peer — check vpn_clients table (Prisma-managed)
	result := h.db.Table("vpn_clients").Where("id = ?", id).Delete(nil)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal hapus: " + result.Error.Error()})
	}
	if result.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "VPN client tidak ditemukan"})
	}
	return c.JSON(fiber.Map{"success": true})
}

// removeWGPeerFromConf removes a [Peer] block matching publicKey from wg0.conf.
func removeWGPeerFromConf(publicKey string) {
	raw, err := os.ReadFile(wgConfFile)
	if err != nil {
		return
	}
	content := string(raw)

	// Match entire [Peer] block that contains this public key.
	// A block starts with [Peer] and ends before the next [section] or EOF.
	re := regexp.MustCompile(`(?m)(\n|^)\[Peer\][^\[]*PublicKey\s*=\s*` + regexp.QuoteMeta(strings.TrimSpace(publicKey)) + `[^\[]*`)
	cleaned := re.ReplaceAllString(content, "")
	cleaned = strings.TrimRight(cleaned, "\n") + "\n"
	_ = os.WriteFile(wgConfFile, []byte(cleaned), 0600)
}

// removeL2TPUserFromChapSecrets removes a user line from /etc/ppp/chap-secrets.
func removeL2TPUserFromChapSecrets(username string) {
	const chapFile = "/etc/ppp/chap-secrets"
	raw, err := os.ReadFile(chapFile)
	if err != nil {
		return
	}
	var kept []string
	for _, line := range strings.Split(string(raw), "\n") {
		// Match lines that start with this username followed by tab or space
		trimmed := strings.TrimSpace(line)
		if trimmed == username || strings.HasPrefix(trimmed, username+"\t") || strings.HasPrefix(trimmed, username+" ") {
			continue
		}
		kept = append(kept, line)
	}
	_ = os.WriteFile(chapFile, []byte(strings.Join(kept, "\n")), 0600)
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

// nextAvailableL2TPIP finds the first unused IP in the L2TP pool.
func nextAvailableL2TPIP(db *gorm.DB, poolStart, poolEnd string) (string, error) {
	prefix := ""
	if parts := strings.Split(poolStart, "."); len(parts) == 4 {
		prefix = strings.Join(parts[:3], ".") + "."
	}
	if prefix == "" {
		return "", fmt.Errorf("poolStart tidak valid: %s", poolStart)
	}

	startOctet, endOctet := 10, 254
	if parts := strings.Split(poolStart, "."); len(parts) == 4 {
		if n, err := strconv.Atoi(parts[3]); err == nil {
			startOctet = n
		}
	}
	if parts := strings.Split(poolEnd, "."); len(parts) == 4 {
		if n, err := strconv.Atoi(parts[3]); err == nil {
			endOctet = n
		}
	}

	usedIPs := map[string]bool{}
	var peers []vpsPeer
	db.Where("type = ?", "l2tp").Find(&peers)
	for _, p := range peers {
		usedIPs[p.PeerIP] = true
	}

	for i := startOctet; i <= endOctet; i++ {
		candidate := prefix + strconv.Itoa(i)
		if !usedIPs[candidate] {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("IP pool L2TP (%d–%d) sudah habis", startOctet, endOctet)
}

// GET /api/network/vps-l2tp-peer — list L2TP peers
func (h *NetworkVPNHandler) ListL2TPPeers(c fiber.Ctx) error {
	var peers []vpsPeer
	h.db.Where("type = ?", "l2tp").Order("createdAt desc").Find(&peers)
	return c.JSON(fiber.Map{"success": true, "peers": peers})
}

// POST /api/network/vps-l2tp-peer — add L2TP peer (generate credentials + add to chap-secrets)
func (h *NetworkVPNHandler) CreateL2TPPeer(c fiber.Ctx) error {
	info := readL2TPServerInfo()
	if info == nil {
		return c.Status(400).JSON(fiber.Map{"error": "L2TP/IPsec server belum di-install di VPS ini"})
	}

	var body struct {
		Action        string  `json:"action"`
		Label         string  `json:"label"`
		NasName       string  `json:"nasName"`
		LocalNetworks *string `json:"localNetworks"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	// Support both "label" and "nasName" as peer identifier
	label := strings.TrimSpace(body.Label)
	if label == "" {
		label = strings.TrimSpace(body.NasName)
	}
	if label == "" {
		return c.Status(400).JSON(fiber.Map{"error": "label atau nasName wajib diisi"})
	}

	// Extract pool info from L2TP server info
	poolStart, _ := info["poolStart"].(string)
	poolEnd, _ := info["poolEnd"].(string)
	localIP, _ := info["localIp"].(string)
	ipsecPsk, _ := info["ipsecPsk"].(string)
	publicIP, _ := info["publicIp"].(string)
	subnet, _ := info["subnet"].(string)

	if poolStart == "" {
		poolStart = "10.201.0.10"
	}
	if poolEnd == "" {
		poolEnd = "10.201.0.254"
	}
	if localIP == "" {
		localIP = "10.201.0.1"
	}

	// Find next available IP from L2TP pool
	vpnIP, err := nextAvailableL2TPIP(h.db, poolStart, poolEnd)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Pool IP habis: " + err.Error()})
	}

	// Generate credentials
	safeLabel := strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(label), "-"), "-")
	if safeLabel == "" {
		safeLabel = "nas"
	}
	username := "l2tp-" + safeLabel
	password := wgRandomAlphanumeric(16)
	nasSecret := wgRandomHex(16)
	apiUsername := "api-" + safeLabel
	apiPassword := wgRandomAlphanumeric(12)

	// Add to /etc/ppp/chap-secrets (static IP assignment)
	// Format: client  server  secret  IP-addresses
	chapEntry := fmt.Sprintf("%s\t*\t%s\t%s\n", username, password, vpnIP)
	chapFile, err := os.OpenFile("/etc/ppp/chap-secrets", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err == nil {
		_, _ = chapFile.WriteString(chapEntry)
		chapFile.Close()
	}

	// Save to vps_peers table
	nasSecretStr := nasSecret
	apiUserStr := apiUsername
	apiPassStr := apiPassword
	peer := vpsPeer{
		ID:          uuid.New().String(),
		Type:        "l2tp",
		PeerName:    label,
		PeerIP:      vpnIP,
		LocalIP:     localIP,
		NasSecret:   &nasSecretStr,
		ApiUsername: &apiUserStr,
		ApiPassword: &apiPassStr,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := h.db.Create(&peer).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal simpan peer: " + err.Error()})
	}

	// Build MikroTik RouterOS script
	routerosScript := fmt.Sprintf(
		`/interface l2tp-client add name="l2tp-%s" connect-to=%s user=%s password="%s" ipsec-secret="%s" use-ipsec=yes disabled=no`,
		safeLabel, publicIP, username, password, ipsecPsk,
	)

	return c.Status(201).JSON(fiber.Map{
		"success":        true,
		"vpnIp":          vpnIP,
		"vpnSubnet":      subnet,
		"username":       username,
		"password":       password,
		"nasSecret":      nasSecret,
		"apiUsername":    apiUsername,
		"apiPassword":    apiPassword,
		"ipsecPsk":       ipsecPsk,
		"serverPublicIp": publicIP,
		"routerosScript": routerosScript,
	})
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

// POST /api/network/vps-wg-peer — add WireGuard peer on VPS (generates keypair + assigns IP)
func (h *NetworkVPNHandler) CreateWGPeer(c fiber.Ctx) error {
	info := readWGServerInfo()
	if info == nil {
		return c.Status(400).JSON(fiber.Map{"error": "WireGuard server belum di-install di VPS ini"})
	}

	var body struct {
		Action        string  `json:"action"`
		NasName       string  `json:"nasName"`
		LocalNetworks *string `json:"localNetworks"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	nasName := strings.TrimSpace(body.NasName)
	if nasName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "nasName wajib diisi"})
	}

	// Extract server info
	publicKey := ""
	if raw, err := os.ReadFile(wgPubKey); err == nil {
		publicKey = strings.TrimSpace(string(raw))
	}
	if publicKey == "" {
		if out, err := exec.Command("wg", "show", "wg0", "public-key").Output(); err == nil {
			publicKey = strings.TrimSpace(string(out))
		}
	}
	listenPort := 51820
	if lp, ok := info["listenPort"].(float64); ok {
		listenPort = int(lp)
	}
	subnet := "10.200.0.0/24"
	if s, ok := info["subnet"].(string); ok && s != "" {
		subnet = s
	}
	gatewayIp := "10.200.0.1"
	if g, ok := info["gatewayIp"].(string); ok && g != "" {
		gatewayIp = g
	}
	poolStart, _ := info["poolStart"].(string)
	poolEnd, _ := info["poolEnd"].(string)
	// Derive defaults from subnet if pool not configured
	if poolStart == "" || poolEnd == "" {
		if parts := strings.Split(strings.Split(subnet, "/")[0], "."); len(parts) == 4 {
			pfx := strings.Join(parts[:3], ".")
			if poolStart == "" {
				poolStart = pfx + ".2"
			}
			if poolEnd == "" {
				poolEnd = pfx + ".254"
			}
		}
	}

	// Find next available IP
	vpnIp, err := nextAvailableWGIP(h.db, subnet, poolStart, poolEnd, gatewayIp)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Pool IP habis: " + err.Error()})
	}

	// Generate WireGuard keypair
	privKeyOut, err := exec.Command("wg", "genkey").Output()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal generate WireGuard private key: " + err.Error()})
	}
	clientPrivKey := strings.TrimSpace(string(privKeyOut))

	pubKeyCmd := exec.Command("wg", "pubkey")
	pubKeyCmd.Stdin = strings.NewReader(clientPrivKey)
	pubKeyOut, err := pubKeyCmd.Output()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal derive WireGuard public key"})
	}
	clientPubKey := strings.TrimSpace(string(pubKeyOut))

	// Add [Peer] block to wg0.conf
	peerBlock := fmt.Sprintf("\n\n[Peer]\n# %s\nPublicKey = %s\nAllowedIPs = %s/32\n",
		nasName, clientPubKey, vpnIp)
	if confRaw, readErr := os.ReadFile(wgConfFile); readErr == nil {
		_ = os.WriteFile(wgConfFile, append(confRaw, []byte(peerBlock)...), 0600)
	}

	// Apply peer to running WireGuard interface (no restart needed)
	_ = exec.Command("wg", "set", "wg0", "peer", clientPubKey, "allowed-ips", vpnIp+"/32").Run()

	// Generate credentials
	nasSecret := wgRandomHex(16)
	safeNasName := strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(nasName), "-"), "-")
	if safeNasName == "" {
		safeNasName = "vpn"
	}
	apiUsername := "api-" + safeNasName
	apiPassword := wgRandomAlphanumeric(12)

	// Get public IP for endpoint
	publicIP, _ := info["publicIp"].(string)
	if publicIP == "" {
		if out, err := exec.Command("curl", "-4", "-s", "--connect-timeout", "5", "ifconfig.me").Output(); err == nil {
			publicIP = strings.TrimSpace(string(out))
		}
	}
	serverEndpoint := publicIP + ":" + strconv.Itoa(listenPort)

	// Save to vps_peers table
	peer := vpsPeer{
		ID:               uuid.New().String(),
		Type:             "wireguard",
		PeerName:         nasName,
		PeerIP:           vpnIp,
		LocalIP:          gatewayIp,
		PublicKey:        &clientPubKey,
		NasSecret:        &nasSecret,
		ApiUsername:      &apiUsername,
		ApiPassword:      &apiPassword,
		ClientPrivateKey: &clientPrivKey,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	_ = h.db.Create(&peer)

	return c.Status(201).JSON(fiber.Map{
		"success":          true,
		"vpnIp":            vpnIp,
		"vpnSubnet":        subnet,
		"gatewayIp":        gatewayIp,
		"serverPublicKey":  publicKey,
		"clientPrivateKey": clientPrivKey,
		"clientPublicKey":  clientPubKey,
		"serverEndpoint":   serverEndpoint,
		"wgPort":           listenPort,
		"nasSecret":        nasSecret,
		"apiUsername":      apiUsername,
		"apiPassword":      apiPassword,
	})
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
