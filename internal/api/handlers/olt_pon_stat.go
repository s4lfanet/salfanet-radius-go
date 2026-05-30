package handlers

// olt_pon_stat.go — per-PON port live stats via Telnet
// GET /api/olt/:id/pon-stat?slot=X&port=Y
// Runs:
//   show interface gpon-olt_1/{slot}/{port}         → traffic rates, bandwidth %, description
//   show interface optical-module-info gpon-olt_1/{slot}/{port} → Temperature, TxPower, Voltage, BiasCurrent

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/telnet"
)

// ponPortStat is the JSON response body for GET /api/olt/:id/pon-stat.
type ponPortStat struct {
	Slot        int    `json:"slot"`
	Port        int    `json:"port"`
	Iface       string `json:"iface"`
	AdminStatus string `json:"adminStatus"` // "activate" | "deactivate"
	LineProto   string `json:"lineProto"`   // "up" | "down"
	Description string `json:"description,omitempty"`
	// Traffic
	InputBps  *int64   `json:"inputBps,omitempty"`  // upstream bytes/s
	OutputBps *int64   `json:"outputBps,omitempty"` // downstream bytes/s
	InputPct  *float64 `json:"inputPct,omitempty"`  // upstream bandwidth %
	OutputPct *float64 `json:"outputPct,omitempty"` // downstream bandwidth %
	// Optical module
	Temperature *float64 `json:"temperature,omitempty"` // Celsius
	TxPower     *float64 `json:"txPower,omitempty"`     // dBm
	RxPower     *float64 `json:"rxPower,omitempty"`     // dBm (N/A on GPON OLT)
	Voltage     *float64 `json:"voltage,omitempty"`     // V
	BiasCurrent *float64 `json:"biasCurrent,omitempty"` // mA
}

// GetPONStat godoc
// GET /api/olt/:id/pon-stat?slot=1&port=1
// Returns live per-PON port stats via Telnet for the given slot/port.
func (h *OLTHandler) GetPONStat(c fiber.Ctx) error {
	id := c.Params("id")
	slot := 1
	port := 1
	if v, err := strconv.Atoi(c.Query("slot")); err == nil && v > 0 {
		slot = v
	}
	if v, err := strconv.Atoi(c.Query("port")); err == nil && v > 0 {
		port = v
	}

	if slot <= 0 || port <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "slot and port must be >= 1"})
	}

	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}
	if olt.Username == nil || olt.Password == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Telnet credentials not configured"})
	}

	tport := olt.TelnetPort
	if tport == 0 {
		tport = 23
	}
	tcfg := telnet.DefaultConfig(olt.IPAddress, tport, *olt.Username, *olt.Password)
	tcfg.CommandTimeout = 15 * time.Second
	pool := telnet.NewPool(tcfg)
	defer pool.Close()

	iface := fmt.Sprintf("gpon-olt_1/%d/%d", slot, port)
	out, err := pool.ExecuteMultiple([]string{
		"show interface " + iface,
		"show interface optical-module-info " + iface,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	parts := splitAtPrompt(out)
	ifaceOut, optOut := "", ""
	if len(parts) >= 1 {
		ifaceOut = parts[0]
	}
	if len(parts) >= 2 {
		optOut = parts[1]
	}

	stat := parsePONInterfaceStat(iface, slot, port, ifaceOut, optOut)
	return c.JSON(fiber.Map{"success": true, "stat": stat})
}

// parsePONInterfaceStat combines traffic stats and optical module info.
func parsePONInterfaceStat(iface string, slot, port int, ifaceOut, optOut string) ponPortStat {
	s := ponPortStat{
		Slot:        slot,
		Port:        port,
		Iface:       iface,
		AdminStatus: "unknown",
		LineProto:   "unknown",
	}

	// ── Parse "show interface gpon-olt_1/slot/port" ───────────────────────────
	for _, rawLine := range strings.Split(ifaceOut, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)

		// "gpon-olt_1/1/1 is activate, line protocol is up."
		// "gpon-olt_1/1/1 is deactivate, line protocol is down."
		if strings.Contains(lower, "line protocol") {
			// Check deactivate FIRST — "deactivate" contains the word "activate"
			if strings.Contains(lower, "deactivate") {
				s.AdminStatus = "deactivate"
			} else if strings.Contains(lower, "activate") {
				s.AdminStatus = "activate"
			}
			if strings.Contains(lower, "line protocol is up") {
				s.LineProto = "up"
			} else {
				s.LineProto = "down"
			}
			continue
		}

		// "Description is Jalur ODC RW 03."
		if strings.HasPrefix(lower, "description is ") {
			s.Description = strings.TrimSuffix(strings.TrimPrefix(line, "Description is "), ".")
			continue
		}

		// "Input rate :             159191 Bps              929 pps"
		if strings.HasPrefix(lower, "input rate") {
			s.InputBps = parsePONBps(line)
			continue
		}
		// "Output rate:            3383584 Bps             2796 pps"
		if strings.HasPrefix(lower, "output rate") {
			s.OutputBps = parsePONBps(line)
			continue
		}
		// "Input Instantaneous bandwidth throughput : 0.1%"
		if strings.HasPrefix(lower, "input instantaneous bandwidth") {
			s.InputPct = parsePONPct(line)
			continue
		}
		// "Output Instantaneous bandwidth throughput: 1.1%"
		if strings.HasPrefix(lower, "output instantaneous bandwidth") {
			s.OutputPct = parsePONPct(line)
			continue
		}
	}

	// ── Parse "show interface optical-module-info gpon-olt_1/slot/port" ───────
	// Reuse the existing uplinkParseOpticalModuleInfo function.
	optInfo := uplinkParseOpticalModuleInfo(optOut)
	if v := parsePONFloatFromUnit(optInfo["Temperature"]); v != nil {
		s.Temperature = v
	}
	if v := parsePONFloatFromUnit(optInfo["TX Power"]); v != nil {
		s.TxPower = v
	}
	if v := parsePONFloatFromUnit(optInfo["RX Power"]); v != nil {
		s.RxPower = v
	}
	if v := parsePONFloatFromUnit(optInfo["Supply Voltage"]); v != nil {
		s.Voltage = v
	}
	if v := parsePONFloatFromUnit(optInfo["TX Bias Current"]); v != nil {
		s.BiasCurrent = v
	}

	return s
}

var rePONBps = regexp.MustCompile(`(\d+)\s+Bps`)

// parsePONBps extracts the first integer Bps value from a rate line.
func parsePONBps(line string) *int64 {
	m := rePONBps.FindStringSubmatch(line)
	if len(m) < 2 {
		return nil
	}
	v, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

var rePONPct = regexp.MustCompile(`(\d+\.?\d*)%`)

// parsePONPct extracts the first percentage value (e.g., "0.1%") from a line.
func parsePONPct(line string) *float64 {
	m := rePONPct.FindStringSubmatch(line)
	if len(m) < 2 {
		return nil
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return nil
	}
	return &v
}

var rePONFloat = regexp.MustCompile(`(-?\d+\.?\d*)`)

// parsePONFloatFromUnit extracts a float from strings like "38.098    (c)", "9.645 (dbm)", "N/A".
func parsePONFloatFromUnit(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "n/a") {
		return nil
	}
	m := rePONFloat.FindStringSubmatch(s)
	if len(m) < 2 {
		return nil
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return nil
	}
	return &v
}
