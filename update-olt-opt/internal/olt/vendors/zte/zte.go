// Package zte implements OLT monitoring and management for ZTE C320 V2.1.
//
// OID reference (verified from live device — ZTE_OID_TABLE.md):
//
//	Base:           1.3.6.1.4.1.3902.1012
//	Description:    .3.28.1.1.2   (zxAnGponOnuCfgTable)
//	Serial:         .3.28.1.1.5   (Hex-STRING: 4 ASCII vendor + 4 hex SN)
//	RegStatus:      .3.50.12.1.1.1 (1 = registered/active)
//	OperState:      .3.50.12.1.1.6 (5 or 4 = online, 0 = unknown, else = offline)
//	RxPower:        .3.50.12.1.1.10 (raw integer; dBm = -(raw/1000))
//	Distance:       .3.50.12.1.1.21 (meters)
//	SeenTable:      1.3.6.1.4.1.3902.1012.3.27.4.1.1
//	PON table:      1.3.6.1.4.1.3902.1012.3.11.3.1.1
package zte

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	snmputil "github.com/s4lfanet/salfanet-radius-go/internal/olt/snmp"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/telnet"
)

// ─── OID constants ────────────────────────────────────────────────────────────

const (
	oidBase = "1.3.6.1.4.1.3902.1012"

	// Reverted to GPON OIDs (1012) because 1015 is EPON and returns 0 results for GPON ONUs.
	oidRegStatus = "1.3.6.1.4.1.3902.1012.3.28.2.1.4"       // zxAnGponOnuPhaseState (1=offline, 2=logging, 3=working)
	oidOperState = "1.3.6.1.4.1.3902.1012.3.50.12.1.1.6"    // zxAnGponOnuOperState
	oidSerial    = "1.3.6.1.4.1.3902.1012.3.28.1.1.5"       // zxAnGponOnuCfgTable / zxAnGponOnuCfgSn
	oidRxPower   = "1.3.6.1.4.1.3902.1012.3.50.12.1.1.10"   // zxAnPonAniOptRxPower
	oidTxPower   = "1.3.6.1.4.1.3902.1012.3.50.12.1.1.11"   // zxAnPonAniOptTxPower (guessed 11 based on typical ZTE layout)
	oidDistance  = "1.3.6.1.4.1.3902.1012.3.50.12.1.1.19"   // zxAnPonAniOptRtt
	oidDesc      = "1.3.6.1.4.1.3902.1012.3.28.1.1.3"       // zxAnGponOnuCfgDesc

	// ponIndex = boardBase + pon * ponIncrement
	board1Base   int64 = 268500992
	board2Base   int64 = 268509184
	ponIncrement int64 = 256
)

// PonIndex returns the SNMP ponIndex for a given board (1 or 2) and PON port (1-8).
func PonIndex(board, pon int) int64 {
	var base int64
	switch board {
	case 1:
		base = board1Base
	case 2:
		base = board2Base
	default:
		base = board1Base
	}
	return base + int64(pon)*ponIncrement
}

// ─── ONU discovery ───────────────────────────────────────────────────────────

// ONUInfo holds the collected data for one ONU.
type ONUInfo struct {
	Frame        int
	Slot         int
	Port         int
	OnuID        int
	SerialNumber string
	Description  string
	Status       models.OltOnuStatus
	RxPower      *float64 // dBm
	TxPower      *float64 // dBm
	Distance     *int     // meters
	Registered   bool
}

// IndexKey is a compact representation of the ONU index used in lookup maps.
type IndexKey struct {
	PonIndex int64
	OnuID    int
}

// DiscoverONUsSNMP walks the entire OLT OID trees and returns all ONU data.
// This uses a global walk approach — we walk each OID base once across the
// entire OLT, and parse ponIndex + onuID from the returned OID suffixes.
// This is far more reliable than per-port walks because it doesn't depend
// on hardcoded ponIndex calculations matching the actual OLT configuration.
//
// ponPorts parameter is ignored (kept for API compatibility).
func DiscoverONUsSNMP(ctx context.Context, snmpCfg snmputil.Config, ponPorts [][2]int) ([]*ONUInfo, error) {
	type oidWalk struct {
		oid string
		key string
	}

	// Walk these 6 OID trees across the entire OLT
	oids := []oidWalk{
		{key: "regStatus", oid: oidRegStatus},
		{key: "operState", oid: oidOperState},
		{key: "serial", oid: oidSerial},
		{key: "rxPower", oid: oidRxPower},
		{key: "txPower", oid: oidTxPower},
		{key: "distance", oid: oidDistance},
		{key: "desc", oid: oidDesc},
	}

	// Collect raw walk results per OID type
	type walkOut struct {
		key     string
		results []snmputil.WalkResult
		err     error
	}

	// Walk each OID sequentially to avoid overwhelming the OLT SNMP agent
	walkResults := make([]walkOut, 0, len(oids))
	for _, o := range oids {
		log.Debug().Str("oid", o.key).Msg("zte: walking OID tree")
		res, err := snmputil.Walk(ctx, snmpCfg, o.oid)
		walkResults = append(walkResults, walkOut{key: o.key, results: res, err: err})
		if err != nil {
			log.Warn().Err(err).Str("oid", o.key).Msg("zte: walk error (continuing)")
		} else {
			log.Debug().Str("oid", o.key).Int("results", len(res)).Msg("zte: walk complete")
		}
	}

	// Parse results — OID suffixes have the form: .ponIndex.onuId
	regStatus := make(map[IndexKey]int64)
	regStatusStr := make(map[IndexKey]string)
	operState := make(map[IndexKey]int64)
	serials := make(map[IndexKey]string)
	rxPower := make(map[IndexKey]int64)
	txPower := make(map[IndexKey]int64)
	distances := make(map[IndexKey]int64)
	descs := make(map[IndexKey]string)

	for _, out := range walkResults {
		if out.err != nil {
			continue
		}
		// Determine the base OID for suffix extraction
		var baseOID string
		switch out.key {
		case "regStatus":
			baseOID = oidRegStatus
		case "operState":
			baseOID = oidOperState
		case "serial":
			baseOID = oidSerial
		case "rxPower":
			baseOID = oidRxPower
		case "txPower":
			baseOID = oidTxPower
		case "distance":
			baseOID = oidDistance
		case "desc":
			baseOID = oidDesc
		}

		skippedCount := 0
		for _, r := range out.results {
			// Extract suffix: ".ponIndex.onuId"
			suffix := snmputil.OIDSuffix(r.OID, baseOID)
			ponIdx, onuID := parsePonOnuSuffix(suffix)
			if ponIdx == 0 || onuID <= 0 {
				skippedCount++
				if skippedCount <= 5 {
					log.Debug().
						Str("oid", r.OID).
						Str("base", baseOID).
						Str("suffix", suffix).
						Msg("zte: failed to parse ponIndex/onuID from OID")
				}
				continue
			}
			k := IndexKey{PonIndex: ponIdx, OnuID: onuID}

			switch out.key {
			case "operState":
				if v, ok := snmputil.ToInt(r.Value); ok {
					operState[k] = v
				}
			case "regStatus":
				if v, ok := snmputil.ToInt(r.Value); ok {
					regStatus[k] = v
					if skippedCount < 5 {
						log.Debug().Int64("regVal", v).Str("suffix", snmputil.OIDSuffix(r.OID, baseOID)).Msg("zte: regStatus INT")
						skippedCount++
					}
				} else {
					strVal := snmputil.ToString(r.Value)
					regStatusStr[k] = strVal
					if skippedCount < 5 {
						log.Debug().Str("regStr", strVal).Str("suffix", snmputil.OIDSuffix(r.OID, baseOID)).Msg("zte: regStatus STR")
						skippedCount++
					}
				}
			case "rxPower":
				if v, ok := snmputil.ToInt(r.Value); ok {
					rxPower[k] = v
				}
			case "txPower":
				if v, ok := snmputil.ToInt(r.Value); ok {
					txPower[k] = v
				}
			case "distance":
				if v, ok := snmputil.ToInt(r.Value); ok {
					distances[k] = v
				}
			case "serial":
				serials[k] = ParseSerial(r.Value)
			case "desc":
				descs[k] = snmputil.ToString(r.Value)
			}
		}
	}

	// Build ONU list
	onuMap := make(map[IndexKey]*ONUInfo)

	// Process based on unified serials keys to catch everything
	for k, sn := range serials {
		frame, slot, port, onuID := decodePonIndex(k.PonIndex, k.OnuID)

		info := &ONUInfo{
			Frame:        frame,
			Slot:         slot,
			Port:         port,
			OnuID:        onuID,
			SerialNumber: sn,
			Status:       models.OnuOffline, // Default to offline
		}

		// Track PhaseState as an integer or string
		var isOnline bool
		if regVal, ok := regStatus[k]; ok {
			// Integer check: 3 or 4 (depending on firmware/GPON vs EPON)
			if regVal == 3 || regVal == 4 {
				isOnline = true
			}
		} else if regStr, ok := regStatusStr[k]; ok {
			// Some ZTE firmwares return a literal string instead of an integer
			strLower := strings.ToLower(regStr)
			if strings.Contains(strLower, "working") || strings.Contains(strLower, "online") {
				isOnline = true
			}
		}

		if isOnline {
			info.Status = models.OnuOnline
		} else {
			info.Status = models.OnuOffline
		}

		if rxRaw, ok := rxPower[k]; ok && rxRaw != 0 && rxRaw != 65535 {
			dbm := float64(rxRaw)*0.002 - 30.0
			info.RxPower = &dbm
		}
		if txRaw, ok := txPower[k]; ok && txRaw != 0 && txRaw != 65535 {
			dbm := float64(txRaw)*0.002 - 30.0
			info.TxPower = &dbm
		}
		if dist, ok := distances[k]; ok && dist > 0 {
			d := int(dist)
			info.Distance = &d
		}
		if desc, ok := descs[k]; ok {
			info.Description = desc
		}

		onuMap[k] = info
	}

	result := make([]*ONUInfo, 0, len(onuMap))
	for _, v := range onuMap {
		result = append(result, v)
	}

	log.Info().Int("total_onus", len(result)).Msg("zte: discovery complete")
	return result, nil
}

// parsePonOnuSuffix parses an OID suffix like ".268501248.5" or ".268501248.5.1" into (ponIndex, onuID).
// We take the first component as ponIndex and the SECOND component as onuID.
func parsePonOnuSuffix(suffix string) (ponIndex int64, onuID int) {
	suffix = strings.TrimPrefix(suffix, ".")
	parts := strings.Split(suffix, ".")
	if len(parts) < 2 {
		return 0, 0
	}
	pi, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0
	}
	oi, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0
	}
	return pi, oi
}

// ─── Serial parsing ──────────────────────────────────────────────────────────

// ParseSerial converts the SNMP Hex-STRING serial value to a human-readable serial.
// SNMP returns 8 bytes: first 4 are ASCII vendor prefix, last 4 are hex SN.
// Result example: "ZTEGDA5918AC"
func ParseSerial(value interface{}) string {
	b, ok := value.([]byte)
	if !ok {
		// Try string representation like "5a 54 45 47 da 59 18 ac"
		s := snmputil.ToString(value)
		b = hexStringToBytes(s)
		if len(b) == 0 {
			return s
		}
	}

	if len(b) < 8 {
		return fmt.Sprintf("%X", b)
	}

	vendor := string(b[0:4])
	// Replace non-printable chars in vendor portion
	var vb strings.Builder
	for _, c := range vendor {
		if c >= 0x20 && c <= 0x7e {
			vb.WriteRune(c)
		} else {
			fmt.Fprintf(&vb, "%02X", c)
		}
	}

	snHex := fmt.Sprintf("%02X%02X%02X%02X", b[4], b[5], b[6], b[7])
	return vb.String() + snHex
}

// hexStringToBytes parses space-separated hex strings like "5a 54 45 47 da 59 18 ac".
func hexStringToBytes(s string) []byte {
	parts := strings.Fields(s)
	out := make([]byte, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.ParseUint(p, 16, 8)
		if err != nil {
			return nil
		}
		out = append(out, byte(v))
	}
	return out
}

// ─── Unregistered ONU (Telnet) ───────────────────────────────────────────────

// UnregisteredONU holds data about an ONU that is seen but not registered.
type UnregisteredONU struct {
	// Port string like "gpon-onu_1/1/1:2"
	PortString   string
	Frame        int
	Slot         int
	Port         int
	OnuID        int
	SerialNumber string
	State        string
}

// GetUnregisteredONUs fetches unregistered ONUs via Telnet.
// It uses "show gpon onu uncfg" (global, all ports) as the authoritative source.
//
// Format: gpon-onu_1/1/1:2  ZTEGDA5918AC  unknown   (3 columns, SN at index 1)
func GetUnregisteredONUs(pool *telnet.Pool) ([]*UnregisteredONU, error) {
	output, err := pool.Execute("show gpon onu uncfg")
	if err != nil {
		return nil, fmt.Errorf("show gpon onu uncfg: %w", err)
	}
	return ParseUncfgOutput(output), nil
}

// ParseUncfgOutput parses the output of "show gpon onu uncfg".
// Line format: gpon-onu_FRAME/SLOT/PORT:ONUID  SERIAL  STATE
func ParseUncfgOutput(output string) []*UnregisteredONU {
	var results []*UnregisteredONU
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "gpon-onu_") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		portStr := fields[0] // gpon-onu_1/1/1:2
		sn := fields[1]      // ZTEGDA5918AC — SN always at index 1
		state := fields[2]

		frame, slot, port, onuID, err := parsePortString(portStr)
		if err != nil {
			log.Warn().Str("port", portStr).Err(err).Msg("zte: failed to parse unregistered ONU port string")
			continue
		}

		results = append(results, &UnregisteredONU{
			PortString:   portStr,
			Frame:        frame,
			Slot:         slot,
			Port:         port,
			OnuID:        onuID,
			SerialNumber: sn,
			State:        state,
		})
	}
	return results
}

// parsePortString parses "gpon-onu_1/1/2:3" → frame=1, slot=1, port=2, onuID=3.
func parsePortString(s string) (frame, slot, port, onuID int, err error) {
	s = strings.TrimPrefix(s, "gpon-onu_")
	// format: frame/slot/port:onuID
	colonIdx := strings.LastIndex(s, ":")
	if colonIdx < 0 {
		return 0, 0, 0, 0, fmt.Errorf("no colon in port string %q", s)
	}
	onuIDStr := s[colonIdx+1:]
	portPart := s[:colonIdx]
	parts := strings.Split(portPart, "/")
	if len(parts) != 3 {
		return 0, 0, 0, 0, fmt.Errorf("expected 3 parts in %q", portPart)
	}
	frame, _ = strconv.Atoi(parts[0])
	slot, _ = strconv.Atoi(parts[1])
	port, _ = strconv.Atoi(parts[2])
	onuID, _ = strconv.Atoi(onuIDStr)
	return
}

// ─── ONU Registration ────────────────────────────────────────────────────────

// RegisterParams holds the parameters for registering a new ONU.
type RegisterParams struct {
	Frame        int
	Slot         int
	Port         int
	OnuID        int
	SerialNumber string
	OnuType      string
	TcontProfile string
	VLAN         int
}

// RegisterONU sends the ZTE basic registration command sequence via Telnet.
// It returns an error if any CLI error is detected (lines starting with "%").
func RegisterONU(pool *telnet.Pool, p RegisterParams) error {
	cmds := []string{
		"configure terminal",
		fmt.Sprintf("interface gpon-olt_%d/%d/%d", p.Frame, p.Slot, p.Port),
		fmt.Sprintf("onu %d type %s sn %s", p.OnuID, p.OnuType, p.SerialNumber),
		"exit",
		fmt.Sprintf("interface gpon-onu_%d/%d/%d:%d", p.Frame, p.Slot, p.Port, p.OnuID),
		fmt.Sprintf("tcont 1 profile %s", p.TcontProfile),
		"gemport 1 tcont 1",
		fmt.Sprintf("service-port 1 vport 1 user-vlan %d vlan %d", p.VLAN, p.VLAN),
		"exit",
		"end",
	}

	output, err := pool.ExecuteMultiple(cmds)
	if err != nil {
		return fmt.Errorf("register ONU: %w", err)
	}

	// Detect CLI errors: only match lines that START with "%"
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "%") {
			return fmt.Errorf("OLT CLI error: %s", line)
		}
		lower := strings.ToLower(line)
		// Additional known error patterns (must not match normal OLT MOTD)
		if strings.Contains(lower, "invalid input") ||
			strings.Contains(lower, "invalid command") ||
			strings.Contains(lower, "already exist") {
			return fmt.Errorf("OLT CLI error: %s", line)
		}
	}

	return nil
}

// DeregisterONU removes an ONU from a PON port via Telnet.
func DeregisterONU(pool *telnet.Pool, frame, slot, port, onuID int) error {
	cmds := []string{
		"configure terminal",
		fmt.Sprintf("interface gpon-olt_%d/%d/%d", frame, slot, port),
		fmt.Sprintf("no onu %d", onuID),
		"exit",
		"end",
	}
	output, err := pool.ExecuteMultiple(cmds)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "%") {
			return fmt.Errorf("OLT CLI error during deregister: %s", strings.TrimSpace(line))
		}
	}
	return nil
}

// ─── ONU Types ───────────────────────────────────────────────────────────────

// ONUType holds metadata about a supported ONU type on this OLT.
type ONUType struct {
	Name string
}

// GetONUTypes fetches the list of registered ONU types from the OLT.
// NOTE: The correct command for ZTE C320 V2.1 is "show onu-type" — NOT "show gpon onu-type".
func GetONUTypes(pool *telnet.Pool) ([]ONUType, error) {
	output, err := pool.Execute("show onu-type")
	if err != nil {
		return nil, err
	}
	return parseONUTypes(output), nil
}

func parseONUTypes(output string) []ONUType {
	var types []ONUType
	seen := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "ONU") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		types = append(types, ONUType{Name: name})
	}
	return types
}

// ─── TCONT Profiles ──────────────────────────────────────────────────────────

// TcontProfile holds a bandwidth profile name.
type TcontProfile struct {
	Name string
}

// GetTcontProfiles fetches available TCONT profiles.
func GetTcontProfiles(pool *telnet.Pool) ([]TcontProfile, error) {
	output, err := pool.Execute("show gpon traffic-profile")
	if err != nil {
		return nil, err
	}
	return parseTcontProfiles(output), nil
}

func parseTcontProfiles(output string) []TcontProfile {
	var profiles []TcontProfile
	seen := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "Profile") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if seen[name] {
			continue
		}
		seen[name] = true
		profiles = append(profiles, TcontProfile{Name: name})
	}
	return profiles
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// decodeOperState converts the raw ZTE OperState integer to our OltOnuStatus.
// 5 or 4 = online, 0 = unknown → offline, anything else = offline.
func decodeOperState(v int64) models.OltOnuStatus {
	switch v {
	case 4, 5:
		return models.OnuOnline
	default:
		return models.OnuOffline
	}
}

// lastOIDComponent extracts the last numeric component from an OID string.
// e.g. ".1.3.6.1.4.1.3902.1012.3.50.12.1.1.6.268501248.1" → 1
func lastOIDComponent(oid string) int {
	parts := strings.Split(oid, ".")
	for i := len(parts) - 1; i >= 0; i-- {
		if v, err := strconv.Atoi(parts[i]); err == nil && v > 0 {
			return v
		}
	}
	return 0
}

// decodePonIndex decodes a ponIndex back to (frame, slot, port).
// ZTE GPON ponIndex format: 0x10000000 (type) | (slot << 16) | (port << 8)
func decodePonIndex(ponIdx int64, onuID int) (frame, slot, port, onu int) {
	onu = onuID
	frame = 1 // C320 is usually frame/chassis 1
	slot = int((ponIdx >> 16) & 0xFF)
	
	actualPort := int((ponIdx >> 8) & 0xFF)
	// The DB and frontend expect port to be 0-indexed because the UI adds 1: `${onu.port + 1}`
	port = actualPort - 1
	if port < 0 {
		port = 0
	}
	return
}

// decodePowerDBM converts raw SNMP RxPower to dBm.
func decodePowerDBM(raw int64) float64 {
	if raw == 0 {
		return 0
	}
	return -math.Abs(float64(raw)) / 1000.0
}

// ─── Concurrent PON Discovery (exported) ─────────────────────────────────────

// DiscoverResult is the full result of a PON discovery run.
type DiscoverResult struct {
	RegisteredONUs   []*ONUInfo
	UnregisteredONUs []*UnregisteredONU
	Errors           []error
}

// DiscoverAll runs SNMP discovery and (if Telnet pool is provided) fetches unregistered ONUs.
// This is the main entry point called by the poller.
func DiscoverAll(ctx context.Context, snmpCfg snmputil.Config, telnetPool *telnet.Pool, ponPorts [][2]int) *DiscoverResult {
	result := &DiscoverResult{}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// SNMP discovery
	wg.Add(1)
	go func() {
		defer wg.Done()
		onus, err := DiscoverONUsSNMP(ctx, snmpCfg, ponPorts)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("SNMP discovery: %w", err))
		}
		result.RegisteredONUs = onus
	}()

	// Telnet: unregistered ONUs (only if Telnet is available)
	if telnetPool != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			uncfg, err := GetUnregisteredONUs(telnetPool)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("telnet uncfg: %w", err))
				return
			}
			// Telnet is authoritative: discard seen-table SNMP ghosts
			result.UnregisteredONUs = uncfg
		}()
	}

	wg.Wait()
	return result
}
