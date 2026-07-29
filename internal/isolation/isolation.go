package isolation

// isolation.go — Port dari src/server/services/isolation.service.ts
//
// Provides isolation settings fetching (with caching) and IP pool utilities.

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"gorm.io/gorm"
)

// Settings holds isolation configuration loaded from the Company record.
type Settings struct {
	IsolationEnabled     bool
	IsolationIpPool      string
	IsolationServerIp    string
	IsolationRateLimit   string
	IsolationRedirectUrl string
	IsolationAllowDns    bool
	IsolationAllowPayment bool
	GracePeriodDays      int
}

var (
	cache     *Settings
	cacheMu   sync.Mutex
	cacheTime time.Time
	cacheTTL  = 5 * time.Minute
)

// GetSettings returns isolation settings from DB with a 5-minute cache.
func GetSettings(db *gorm.DB) *Settings {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if cache != nil && time.Since(cacheTime) < cacheTTL {
		return cache
	}

	var company models.Company
	if err := db.First(&company).Error; err != nil {
		return defaults()
	}

	s := &Settings{
		IsolationEnabled:     true,
		IsolationIpPool:      "192.168.200.0/24",
		IsolationRateLimit:   "128k/128k",
		IsolationRedirectUrl: "",
		IsolationAllowDns:    true,
		IsolationAllowPayment: true,
		GracePeriodDays:      0,
	}

	if company.IsolationEnabled != nil {
		s.IsolationEnabled = *company.IsolationEnabled
	}
	if company.IsolationIpPool != nil && *company.IsolationIpPool != "" {
		s.IsolationIpPool = *company.IsolationIpPool
	}
	if company.IsolationServerIp != nil {
		s.IsolationServerIp = *company.IsolationServerIp
	}
	if company.IsolationRateLimit != nil && *company.IsolationRateLimit != "" {
		s.IsolationRateLimit = *company.IsolationRateLimit
	}
	if company.IsolationRedirectUrl != nil {
		s.IsolationRedirectUrl = *company.IsolationRedirectUrl
	}
	if company.IsolationAllowDns != nil {
		s.IsolationAllowDns = *company.IsolationAllowDns
	}
	if company.IsolationAllowPayment != nil {
		s.IsolationAllowPayment = *company.IsolationAllowPayment
	}
	if company.GracePeriodDays != nil {
		s.GracePeriodDays = *company.GracePeriodDays
	}

	cache = s
	cacheTime = time.Now()
	return s
}

func defaults() *Settings {
	return &Settings{
		IsolationEnabled:     true,
		IsolationIpPool:      "192.168.200.0/24",
		IsolationRateLimit:   "128k/128k",
		IsolationRedirectUrl: "",
		IsolationAllowDns:    true,
		IsolationAllowPayment: true,
		GracePeriodDays:      0,
	}
}

// ClearCache clears the cached isolation settings.
func ClearCache() {
	cacheMu.Lock()
	cache = nil
	cacheMu.Unlock()
}

// IsIPInIsolationPool checks if an IP address is within the given CIDR pool.
func IsIPInIsolationPool(ipAddress, cidr string) bool {
	if ipAddress == "" || cidr == "" {
		return false
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return false
	}
	return ipNet.Contains(ip)
}

// CIDRRange holds the start, end, and gateway IPs for a pool.
type CIDRRange struct {
	StartIP string
	EndIP   string
	Gateway string
}

// GetCIDRRange calculates the IP range for a MikroTik pool from CIDR notation.
// Gateway is network+1, pool starts at network+100, max 100 IPs.
func GetCIDRRange(cidr string) CIDRRange {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return CIDRRange{
			StartIP: "192.168.200.100",
			EndIP:   "192.168.200.200",
			Gateway: "192.168.200.1",
		}
	}

	networkInt := ipToInt(ipNet.IP)
	maskOnes, _ := ipNet.Mask.Size()
	hostBits := 32 - maskOnes

	gatewayInt := networkInt + 1
	startInt := networkInt + 100
	broadcastInt := networkInt + (uint32(1) << hostBits) - 1
	endInt := startInt + 99
	if endInt > broadcastInt-1 {
		endInt = broadcastInt - 1
	}

	return CIDRRange{
		StartIP: intToIP(startInt),
		EndIP:   intToIP(endInt),
		Gateway: intToIP(gatewayInt),
	}
}

func ipToInt(ip net.IP) uint32 {
	return binary.BigEndian.Uint32(ip.To4())
}

func intToIP(n uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		(n>>24)&0xFF, (n>>16)&0xFF, (n>>8)&0xFF, n&0xFF)
}
