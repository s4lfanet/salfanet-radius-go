// Package zte implements OLT monitoring and management for ZTE C320 V2.1.
//
// OID reference (verified from live device — ZTE_OID_TABLE.md):
//
//	Base:           1.3.6.1.4.1.3902.1012
//	Description:    .3.28.1.1.2   (zxAnGponOnuCfgTable, indexed .col.ponIndex.onuId)
//	Serial:         .3.28.1.1.5   (Hex-STRING: 4 ASCII vendor + 4 hex SN)
//	RegStatus:      .3.50.12.1.1.1 (1 = registered; indexed .col.ponIndex.onuSlot.onuId)
//	OperState:      .3.50.12.1.1.6 (5|4=online, 0=unknown, else=offline)
//	RxPower:        .3.50.12.1.1.10 (raw int; dBm = raw/500.0 - 30; e.g. raw=6751 → -16.50 dBm)
//	TxPower:        .3.50.12.1.1.11 (same encoding as RxPower)
//	Distance:       .3.50.12.1.1.18 (equalization delay; meters = raw × 0.112)
//	SeenONU table:  1.3.6.1.4.1.3902.1012.3.27.4.1.1  (ALL seen ONUs incl. unregistered)
//	PON port table: 1.3.6.1.4.1.3902.1012.3.11.3.1.1  (one entry per provisioned PON port)
package zte

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	snmputil "github.com/s4lfanet/salfanet-radius-go/internal/olt/snmp"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/telnet"
)

// ─── OID constants ────────────────────────────────────────────────────────────

const (
	oidBase = "1.3.6.1.4.1.3902.1012"

	// zxAnGponOnuCfgTable — indexed .col.ponIndex.onuId (2 index components)
	oidDescription = oidBase + ".3.28.1.1.2"
	oidSerial      = oidBase + ".3.28.1.1.5"

	// zxAnGponOnuRegTable — indexed .col.ponIndex.onuSlot.onuId (3 index components)
	oidRegStatus   = oidBase + ".3.50.12.1.1.1"
	oidOperState   = oidBase + ".3.50.12.1.1.6"
	oidDeregReason = oidBase + ".3.50.12.1.1.7"  // last deregistration reason (int → PowerOff/LOS/etc.)
	oidRxPower     = oidBase + ".3.50.12.1.1.10" // raw int; dBm = raw/500.0 - 30.0
	oidTxPower     = oidBase + ".3.50.12.1.1.11" // OLT TX power toward ONU
	oidDistance    = oidBase + ".3.50.12.1.1.18" // ONU equalization delay; distance (m) = raw × 0.112

	// zxAnGponOnuDiscoveredInfoTable — ALL seen ONUs incl. unregistered; indexed .ponIndex.onuSlot.onuId
	oidSeenONUTable = oidBase + ".3.27.4.1.1"

	// ZTE C320 V2.1 PON port provisioning table — one entry per active PON port; indexed by ponIndex
	oidPONPortTable = oidBase + ".3.11.3.1.1"

	// Index format notes (verified via live SNMP walk against ZTE C320 V2.1):
	//   zxAnGponOnuRegTable  (.3.50.12.1.1.*): suffix = .ponIndex.onuId.1   (onuId is second-to-last)
	//   zxAnGponOnuCfgTable  (.3.28.1.1.*):    suffix = .ponIndex.onuId      (onuId is last)
	//   zxAnGponOnuDiscoveredInfoTable (.3.27.4.1.1): suffix = .ponIndex.onuId.colIdx (onuId is second-to-last)

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
	Frame           int
	Slot            int
	Port            int
	OnuID           int
	SerialNumber    string
	Description     string
	Status          models.OltOnuStatus
	RxPower         *float64 // dBm (negative, e.g. -23.5)
	TxPower         *float64 // dBm (OLT TX toward ONU)
	Distance        *int     // meters
	LastDeregReason *string  // last deregistration reason from SNMP (e.g. "PowerOff", "LOS", "Reboot")
	Registered      bool
}

// IndexKey is a compact representation of the ONU index used in lookup maps.
type IndexKey struct {
	PonIndex int64
	OnuID    int
}

// ─── Dynamic PON port discovery ───────────────────────────────────────────────

// discoverPONPorts walks the ZTE C320 V2.1 PON port provisioning table and
// returns all active (board, pon) pairs found via SNMP.
// Falls back to the traditional 2×8 default when the walk returns nothing.
func discoverPONPorts(ctx context.Context, cfg snmputil.Config) [][2]int {
	results, err := snmputil.BulkWalk(ctx, cfg, oidPONPortTable)
	seenPonIndexes := map[int64]bool{}
	if err == nil {
		for _, r := range results {
			for _, part := range strings.Split(r.OID, ".") {
				n, e := strconv.ParseInt(part, 10, 64)
				if e != nil || n <= 268000000 {
					continue
				}
				seenPonIndexes[n] = true
			}
		}
	}

	var ports [][2]int
	for ponIdx := range seenPonIndexes {
		if ponIdx > board1Base && ponIdx < board1Base+128*ponIncrement {
			if pon := int((ponIdx - board1Base) / ponIncrement); pon >= 1 && (ponIdx-board1Base)%ponIncrement == 0 {
				ports = append(ports, [2]int{1, pon})
			}
		} else if ponIdx > board2Base && ponIdx < board2Base+128*ponIncrement {
			if pon := int((ponIdx - board2Base) / ponIncrement); pon >= 1 && (ponIdx-board2Base)%ponIncrement == 0 {
				ports = append(ports, [2]int{2, pon})
			}
		}
	}

	if len(ports) == 0 {
		log.Debug().Msg("zte: PON port table walk returned nothing — using 2×8 fallback")
		for b := 1; b <= 2; b++ {
			for p := 1; p <= 8; p++ {
				ports = append(ports, [2]int{b, p})
			}
		}
	}

	sort.Slice(ports, func(i, j int) bool {
		if ports[i][0] != ports[j][0] {
			return ports[i][0] < ports[j][0]
		}
		return ports[i][1] < ports[j][1]
	})
	return ports
}

// ─── ONU discovery ───────────────────────────────────────────────────────────

// DiscoverONUsSNMP walks all PON ports concurrently and returns merged ONU data.
// ponPorts is a list of (board, pon) pairs; obtain via discoverPONPorts().
//
// 8 OID trees are BulkWalked in parallel per PON port.
// Unregistered ONUs are detected via the SNMP seen-ONU table (no Telnet required).
func DiscoverONUsSNMP(ctx context.Context, snmpCfg snmputil.Config, ponPorts [][2]int) ([]*ONUInfo, error) {
	resultsCh := make(chan ponResult, len(ponPorts))

	for _, bp := range ponPorts {
		bp := bp
		go func() {
			ponIdx := PonIndex(bp[0], bp[1])
			res := walkPONPort(ctx, snmpCfg, ponIdx)
			resultsCh <- res
		}()
	}

	// Merge results from all PON ports
	merged := &ponResult{
		regStatus:   make(map[IndexKey]int64),
		operState:   make(map[IndexKey]int64),
		deregReason: make(map[IndexKey]int64),
		serials:     make(map[IndexKey]string),
		rxPower:     make(map[IndexKey]int64),
		txPower:     make(map[IndexKey]int64),
		distances:   make(map[IndexKey]int64),
		descs:       make(map[IndexKey]string),
		seenONUs:    make(map[IndexKey]bool),
	}

	for range ponPorts {
		r := <-resultsCh
		if r.err != nil {
			log.Warn().Err(r.err).Msg("zte: PON walk error (partial results may be available)")
		}
		for k, v := range r.regStatus {
			merged.regStatus[k] = v
		}
		for k, v := range r.operState {
			merged.operState[k] = v
		}
		for k, v := range r.deregReason {
			merged.deregReason[k] = v
		}
		for k, v := range r.serials {
			merged.serials[k] = v
		}
		for k, v := range r.rxPower {
			merged.rxPower[k] = v
		}
		for k, v := range r.txPower {
			merged.txPower[k] = v
		}
		for k, v := range r.distances {
			merged.distances[k] = v
		}
		for k, v := range r.descs {
			merged.descs[k] = v
		}
		for k, v := range r.seenONUs {
			merged.seenONUs[k] = v
		}
	}

	onuMap := make(map[IndexKey]*ONUInfo)

	// ── Registered ONUs (regStatus=1 normal; regStatus=2 = SB mode, used by FiberHome ONUs) ───
	for k, regVal := range merged.regStatus {
		if regVal != 1 && regVal != 2 {
			continue
		}
		frame, slot, port, onuID := decodePonIndex(k.PonIndex, k.OnuID)
		info := &ONUInfo{
			Frame:      frame,
			Slot:       slot,
			Port:       port,
			OnuID:      onuID,
			Registered: true,
		}
		if sn, ok := merged.serials[k]; ok {
			info.SerialNumber = sn
		}
		if desc, ok := merged.descs[k]; ok {
			info.Description = desc
		}
		info.Status = decodeOperState(merged.operState[k])

		// RxPower: ZTE C320 encodes optical power as raw = (dBm + 30) * 500.
		// Formula: dBm = raw/500.0 - 30.0
		// Verified from live device: raw=6751 → -16.50 dBm; raw=5085 → -19.83 dBm.
		// 0xFFFF (65535) is a ZTE sentinel meaning "no data" → must be excluded.
		if rxRaw, ok := merged.rxPower[k]; ok && rxRaw > 0 && rxRaw != 0xFFFF {
			dbm := float64(rxRaw)/500.0 - 30.0
			info.RxPower = &dbm
		}
		// TxPower: same encoding and same sentinel
		if txRaw, ok := merged.txPower[k]; ok && txRaw > 0 && txRaw != 0xFFFF {
			dbm := float64(txRaw)/500.0 - 30.0
			info.TxPower = &dbm
		}
		if dist, ok := merged.distances[k]; ok && dist > 0 && dist < 1000000 {
			d := int(float64(dist) * 0.112)
			info.Distance = &d
		}
		if reason, ok := merged.deregReason[k]; ok {
			info.LastDeregReason = decodeDeregReason(reason)
			// ZTE MIB: DeregReason=1 means notApplicable (ONU currently registered).
			// DeregReason>=2 means ONU is currently DEREGISTERED (offline).
			// OperState can lag behind in SNMP cache — trust DeregReason over OperState.
			if reason >= 2 && info.Status == models.OnuOnline {
				info.Status = models.OnuOffline
			}
		}
		onuMap[k] = info
	}

	// ── Also include ONUs visible via OperState but missing from regStatus ────
	for k, opState := range merged.operState {
		if _, exists := onuMap[k]; exists {
			continue
		}
		frame, slot, port, onuID := decodePonIndex(k.PonIndex, k.OnuID)
		info := &ONUInfo{
			Frame:      frame,
			Slot:       slot,
			Port:       port,
			OnuID:      onuID,
			Status:     decodeOperState(opState),
			Registered: true,
		}
		if sn, ok := merged.serials[k]; ok {
			info.SerialNumber = sn
		}
		if desc, ok := merged.descs[k]; ok {
			info.Description = desc
		}
		if rxRaw, ok := merged.rxPower[k]; ok && rxRaw > 0 && rxRaw != 0xFFFF {
			dbm := float64(rxRaw)/500.0 - 30.0
			info.RxPower = &dbm
		}
		if txRaw, ok := merged.txPower[k]; ok && txRaw > 0 && txRaw != 0xFFFF {
			dbm := float64(txRaw)/500.0 - 30.0
			info.TxPower = &dbm
		}
		if dist, ok := merged.distances[k]; ok && dist > 0 && dist < 1000000 {
			d := int(float64(dist) * 0.112)
			info.Distance = &d
		}
		if reason, ok := merged.deregReason[k]; ok {
			info.LastDeregReason = decodeDeregReason(reason)
			// ZTE MIB: DeregReason=1 means notApplicable (ONU currently registered).
			// DeregReason>=2 means ONU is currently DEREGISTERED (offline).
			// OperState can lag behind in SNMP cache — trust DeregReason over OperState.
			if reason >= 2 && info.Status == models.OnuOnline {
				info.Status = models.OnuOffline
			}
		}
		onuMap[k] = info
	}

	// ── Unregistered ONUs (seen-ONU table minus registered set) ──────────────
	// The SNMP seen-ONU table contains ALL ONUs visible on each PON port,
	// including those not yet provisioned/registered. No Telnet required.
	for k := range merged.seenONUs {
		if _, exists := onuMap[k]; exists {
			continue // already accounted for as registered
		}
		frame, slot, port, onuID := decodePonIndex(k.PonIndex, k.OnuID)
		info := &ONUInfo{
			Frame:      frame,
			Slot:       slot,
			Port:       port,
			OnuID:      onuID,
			Status:     models.OnuUnregistered,
			Registered: false,
		}
		// Serial number may be available from the cfg table even for unregistered ONUs
		if sn, ok := merged.serials[k]; ok {
			info.SerialNumber = sn
		}
		onuMap[k] = info
	}

	result := make([]*ONUInfo, 0, len(onuMap))
	for _, v := range onuMap {
		result = append(result, v)
	}
	return result, nil
}

// walkPONPort fetches all ONU data for a single PON port index via 8 parallel BulkWalks.
//
// OID index structure:
//
//	zxAnGponOnuCfgTable (.3.28.1.1.*):      .col.ponIndex.onuId           (2-component suffix)
//	zxAnGponOnuRegTable (.3.50.12.1.1.*):   .col.ponIndex.onuSlot.onuId   (3-component suffix)
//	zxAnGponOnuDiscoveredInfoTable (.3.27.4.1.1): .ponIndex.onuSlot.onuId
//
// For all tables, onuId is always the LAST OID component, so lastOIDComponent() is correct.
type ponResult struct {
	regStatus   map[IndexKey]int64
	operState   map[IndexKey]int64
	deregReason map[IndexKey]int64
	serials     map[IndexKey]string
	rxPower     map[IndexKey]int64
	txPower     map[IndexKey]int64
	distances   map[IndexKey]int64
	descs       map[IndexKey]string
	seenONUs    map[IndexKey]bool // all ONU IDs visible on this PON (registered + unregistered)
	err         error
}

func walkPONPort(ctx context.Context, cfg snmputil.Config, ponIdx int64) ponResult {
	type oidWalk struct {
		oid string
		key string
	}

	oids := []oidWalk{
		{fmt.Sprintf("%s.%d", oidRegStatus, ponIdx), "regStatus"},
		{fmt.Sprintf("%s.%d", oidOperState, ponIdx), "operState"},
		{fmt.Sprintf("%s.%d", oidDeregReason, ponIdx), "deregReason"},
		{fmt.Sprintf("%s.%d", oidSerial, ponIdx), "serial"},
		{fmt.Sprintf("%s.%d", oidRxPower, ponIdx), "rxPower"},
		{fmt.Sprintf("%s.%d", oidTxPower, ponIdx), "txPower"},
		{fmt.Sprintf("%s.%d", oidDistance, ponIdx), "distance"},
		{fmt.Sprintf("%s.%d", oidDescription, ponIdx), "desc"},
		{fmt.Sprintf("%s.%d", oidSeenONUTable, ponIdx), "seen"},
	}

	type walkOut struct {
		key     string
		baseOID string // full walked OID (oid + "." + ponIdx)
		results []snmputil.WalkResult
		err     error
	}

	ch := make(chan walkOut, len(oids))
	for _, o := range oids {
		o := o
		go func() {
			// BulkWalk (GetBulk PDUs) is far more efficient for large tables.
			// snmputil.BulkWalk automatically falls back to Walk on error.
			res, err := snmputil.BulkWalk(ctx, cfg, o.oid)
			ch <- walkOut{key: o.key, baseOID: o.oid, results: res, err: err}
		}()
	}

	pr := ponResult{
		regStatus:   make(map[IndexKey]int64),
		operState:   make(map[IndexKey]int64),
		deregReason: make(map[IndexKey]int64),
		serials:     make(map[IndexKey]string),
		rxPower:     make(map[IndexKey]int64),
		txPower:     make(map[IndexKey]int64),
		distances:   make(map[IndexKey]int64),
		descs:       make(map[IndexKey]string),
		seenONUs:    make(map[IndexKey]bool),
	}

	for range oids {
		out := <-ch
		if out.err != nil {
			pr.err = out.err
			continue
		}

		for _, r := range out.results {
			// Validate OID belongs to the walked subtree.
			// BulkWalk may return the first entry of the next sibling subtree
			// when the last GetBulk batch overshoots. Strip a leading dot that
			// gosnmp sometimes adds, then check for the expected prefix.
			oidNorm := strings.TrimPrefix(r.OID, ".")
			if !strings.HasPrefix(oidNorm, out.baseOID+".") {
				continue
			}

			// ZTE C320 GPON OID index structure:
			//  zxAnGponOnuRegTable  (.3.50.12.1.1.*): .col.ponIndex.onuId.1  → use second-to-last
			//  zxAnGponOnuCfgTable  (.3.28.1.1.*):    .col.ponIndex.onuId    → use last
			//  zxAnGponOnuDiscoveredInfoTable (.3.27.4.1.1): .ponIndex.onuId.colIdx → use second-to-last
			var onuID int
			switch out.key {
			case "serial", "desc":
				onuID = lastOIDComponent(r.OID)
			default:
				onuID = secondToLastOIDComponent(r.OID)
			}
			if onuID <= 0 || onuID > 128 {
				continue
			}
			k := IndexKey{PonIndex: ponIdx, OnuID: onuID}

			switch out.key {
			case "regStatus", "operState", "deregReason", "rxPower", "txPower", "distance":
				v, ok := snmputil.ToInt(r.Value)
				if !ok {
					break
				}
				switch out.key {
				case "regStatus":
					pr.regStatus[k] = v
				case "operState":
					pr.operState[k] = v
				case "deregReason":
					pr.deregReason[k] = v
				case "rxPower":
					pr.rxPower[k] = v
				case "txPower":
					pr.txPower[k] = v
				case "distance":
					pr.distances[k] = v
				}
			case "serial":
				pr.serials[k] = ParseSerial(r.Value)
			case "desc":
				pr.descs[k] = snmputil.ToString(r.Value)
			case "seen":
				pr.seenONUs[k] = true
			}
		}
	}

	return pr
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

// CleanONUConfig resets an ONU's service configuration to factory defaults via
// "restore default" on the gpon-onu interface.  The ONU stays registered on the
// PON port — only its service configuration (VLANs, profiles, etc.) is cleared.
func CleanONUConfig(pool *telnet.Pool, frame, slot, port, onuID int) error {
	cmds := []string{
		"configure terminal",
		fmt.Sprintf("interface gpon-onu_%d/%d/%d:%d", frame, slot, port, onuID),
		"restore default",
		"exit",
		"end",
	}
	output, err := pool.ExecuteMultiple(cmds)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "%") {
			return fmt.Errorf("OLT CLI error during clean config: %s", strings.TrimSpace(line))
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

// TcontProfile holds a DBA/TCONT profile name and bandwidth info.
type TcontProfile struct {
	Name   string
	BwType int
	FBW    int
	ABW    int
	MBW    int
}

// TrafficProfile holds a downstream traffic profile name and rate info.
type TrafficProfile struct {
	Name string
	SIR  int
	PIR  int
}

// GetTcontProfiles fetches available TCONT/DBA profiles via "show gpon profile tcont".
func GetTcontProfiles(pool *telnet.Pool) ([]TcontProfile, error) {
	output, err := pool.Execute("show gpon profile tcont")
	if err != nil {
		return nil, err
	}
	return parseTcontProfiles(output), nil
}

func parseTcontProfiles(output string) []TcontProfile {
	var profiles []TcontProfile
	var current *TcontProfile
	dataExpected := false

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "Profile name") {
			if current != nil {
				profiles = append(profiles, *current)
			}
			if idx := strings.LastIndex(trimmed, ":"); idx >= 0 {
				name := strings.TrimSpace(trimmed[idx+1:])
				current = &TcontProfile{Name: name}
				dataExpected = false
			}
			continue
		}
		if strings.Contains(trimmed, "FBW") || strings.Contains(trimmed, "ABW") {
			dataExpected = true
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "ZXAN") {
			continue
		}
		if dataExpected && current != nil {
			fields := strings.Fields(trimmed)
			if len(fields) >= 4 {
				t, _ := strconv.Atoi(fields[0])
				fbw, _ := strconv.Atoi(fields[1])
				abw, _ := strconv.Atoi(fields[2])
				mbw, _ := strconv.Atoi(fields[3])
				current.BwType = t
				current.FBW = fbw
				current.ABW = abw
				current.MBW = mbw
				dataExpected = false
			}
		}
	}
	if current != nil {
		profiles = append(profiles, *current)
	}
	return profiles
}

// GetTrafficProfiles fetches available downstream traffic profiles via "show gpon profile traffic".
func GetTrafficProfiles(pool *telnet.Pool) ([]TrafficProfile, error) {
	output, err := pool.Execute("show gpon profile traffic")
	if err != nil {
		return nil, err
	}
	return parseTrafficProfiles(output), nil
}

func parseTrafficProfiles(output string) []TrafficProfile {
	var profiles []TrafficProfile
	var current *TrafficProfile
	awaitData := false

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "Profile name") {
			if current != nil {
				profiles = append(profiles, *current)
			}
			if idx := strings.LastIndex(trimmed, ":"); idx >= 0 {
				name := strings.TrimSpace(trimmed[idx+1:])
				current = &TrafficProfile{Name: name}
				awaitData = false
			}
			continue
		}
		if strings.Contains(trimmed, "SIR") || strings.Contains(trimmed, "PIR") {
			awaitData = true
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "ZXAN") {
			continue
		}
		if awaitData && current != nil && current.SIR == 0 {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				sir, e1 := strconv.Atoi(fields[0])
				pir, e2 := strconv.Atoi(fields[1])
				if e1 == nil && e2 == nil {
					current.SIR = sir
					current.PIR = pir
					awaitData = false
				}
			}
		}
	}
	if current != nil {
		profiles = append(profiles, *current)
	}
	return profiles
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// decodeOperState converts the raw ZTE OperState integer to our OltOnuStatus.
// ZTE C320 zxAnGponOnuRegOperStatus values (verified from ZTE MIB):
//
//	1=notPresent, 2=inactive, 3=activating, 4=working, 5=active, 6=dyingGasp
func decodeOperState(v int64) models.OltOnuStatus {
	switch v {
	case 4, 5:
		return models.OnuOnline
	case 6:
		return models.OnuDyingGasp
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

// secondToLastOIDComponent returns the second-to-last numeric OID component.
// Used for ZTE C320 tables with 2-component row index suffix (e.g. .onuId.1 for RegTable,
// or .onuId.colIdx for SeenONU), where the actual ONU-ID is the second-to-last component.
func secondToLastOIDComponent(oid string) int {
	parts := strings.Split(oid, ".")
	foundLast := false
	for i := len(parts) - 1; i >= 0; i-- {
		v, err := strconv.Atoi(parts[i])
		if err != nil || v <= 0 {
			continue
		}
		if !foundLast {
			foundLast = true
			continue // skip the trailing sub-index
		}
		return v
	}
	return 0
}

// decodePonIndex decodes a ponIndex back to (frame, slot, port).
// ZTE C320: frame is always 1 (single chassis). slot = board card (1 or 2). port = 1-based PON port.
// ponIndex = boardBase + port * ponIncrement
func decodePonIndex(ponIdx int64, onuID int) (frame, slot, port, onu int) {
	onu = onuID
	frame = 1 // ZTE C320: single chassis, frame is always 1
	var offset int64

	if ponIdx >= board2Base {
		slot = 2
		offset = ponIdx - board2Base
	} else {
		slot = 1
		offset = ponIdx - board1Base
	}

	if ponIncrement > 0 {
		port = int(offset / ponIncrement)
	}
	return
}

// ─── Concurrent PON Discovery (exported) ─────────────────────────────────────

// DiscoverAll runs full SNMP-based ONU discovery for the OLT.
// It first discovers active PON ports dynamically via the ZTE PON table,
// then BulkWalks all ONU data tables in parallel per port.
// Unregistered ONUs are detected via the SNMP seen-ONU table — no Telnet required.
//
// Telnet (pool) remains available for write operations (config, register, deregister)
// through the package-level Telnet helper functions.
func DiscoverAll(ctx context.Context, snmpCfg snmputil.Config) ([]*ONUInfo, error) {
	ponPorts := discoverPONPorts(ctx, snmpCfg)
	log.Debug().Int("ports", len(ponPorts)).Msg("zte: discovered PON ports")
	return DiscoverONUsSNMP(ctx, snmpCfg, ponPorts)
}

// ─── Telnet Distance Collection ───────────────────────────────────────────────

// FetchTelnetDistances retrieves ONU fiber distances via Telnet
// "show gpon onu detail-info gpon-onu_F/S/P:N" for each registered ONU.
// Returns a map of "F/S/P:N" → distance in meters.
// All commands are batched in a single Telnet session for efficiency.
func FetchTelnetDistances(pool *telnet.Pool, onus []*ONUInfo) map[string]int {
	var cmds []string
	for _, onu := range onus {
		if onu.Registered {
			cmds = append(cmds, fmt.Sprintf(
				"show gpon onu detail-info gpon-onu_%d/%d/%d:%d",
				onu.Frame, onu.Slot, onu.Port, onu.OnuID,
			))
		}
	}
	if len(cmds) == 0 {
		return nil
	}

	output, err := pool.ExecuteMultiple(cmds)
	if err != nil {
		log.Warn().Err(err).Int("onus", len(cmds)).Msg("zte: telnet distance fetch failed")
		return nil
	}

	return parseTelnetDistances(output)
}

// parseTelnetDistances scans the combined Telnet output from multiple
// "show gpon onu detail-info" commands and extracts ONU Distance values.
// Looks for: "ONU interface: gpon-onu_F/S/P:N" followed by "ONU Distance: Xm".
func parseTelnetDistances(raw string) map[string]int {
	distances := map[string]int{}
	var currentKey string

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Match: "ONU interface:         gpon-onu_1/1/1:1"
		if strings.HasPrefix(line, "ONU interface:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "ONU interface:"))
			if idx := strings.Index(val, "gpon-onu_"); idx >= 0 {
				currentKey = val[idx+len("gpon-onu_"):]
			} else {
				currentKey = ""
			}
			continue
		}

		// Match: "ONU Distance:        583m"
		if currentKey != "" && strings.HasPrefix(line, "ONU Distance:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "ONU Distance:"))
			val = strings.TrimSuffix(strings.TrimSpace(val), "m")
			val = strings.TrimSpace(val)
			if d, err := strconv.Atoi(val); err == nil && d > 0 {
				distances[currentKey] = d
			}
		}
	}

	return distances
}

// ─── Telnet ONU State Collection ─────────────────────────────────────────────

// FetchTelnetONUStates fetches the authoritative operational state for every
// registered ONU by running "show gpon onu state gpon-olt_F/S/P" once per
// unique PON port in a single batched Telnet session.
//
// ZTE C320's SNMP OperState agent may lag (reporting a dying-gasp or LOS ONU
// as still "working") while the CLI reflects the real state immediately.
// Callers should override the SNMP-derived Status field with this map.
//
// Returns a map of "F/S/P:ID" → OltOnuStatus.
func FetchTelnetONUStates(pool *telnet.Pool, onus []*ONUInfo) map[string]models.OltOnuStatus {
	// Collect unique PON ports from registered ONUs.
	type portKey struct{ frame, slot, port int }
	seen := make(map[portKey]bool)
	for _, o := range onus {
		if o.Registered {
			seen[portKey{o.Frame, o.Slot, o.Port}] = true
		}
	}
	if len(seen) == 0 {
		return nil
	}

	// Build one command per unique port, execute in a single Telnet session.
	cmds := make([]string, 0, len(seen))
	for pk := range seen {
		cmds = append(cmds, fmt.Sprintf("show gpon onu state gpon-olt_%d/%d/%d", pk.frame, pk.slot, pk.port))
	}

	output, err := pool.ExecuteMultiple(cmds)
	if err != nil {
		log.Warn().Err(err).Int("ports", len(cmds)).Msg("zte: telnet ONU state fetch failed")
		return nil
	}

	return parseONUStateOutput(output)
}

// parseONUStateOutput parses the combined Telnet output of one or more
// "show gpon onu state gpon-olt_F/S/P" commands.
//
// ZTE C320 column format (verified on firmware V2.1):
//
//	OnuIndex  Admin State  OMCC State  Phase State  Channel
//	1/1/1:1   enable       enable      working      1(GPON)
//	1/1/1:38  enable       enable      dying-gasp   1(GPON)
//
// The operational state ("Phase State") is at column index 3. To handle
// firmware variations without hardcoding a column index, we scan ALL
// fields after the ONU index for a known state keyword. If no state is
// recognized the ONU is omitted so the SNMP-derived status is preserved.
func parseONUStateOutput(output string) map[string]models.OltOnuStatus {
	result := make(map[string]models.OltOnuStatus)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		// Strip optional "gpon-onu_" prefix (some firmware versions include it)
		line = strings.TrimPrefix(line, "gpon-onu_")
		// Must start with a digit (frame number)
		if len(line) == 0 || line[0] < '0' || line[0] > '9' {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		onuIdx := fields[0] // "F/S/P:ID"
		if !strings.Contains(onuIdx, "/") || !strings.Contains(onuIdx, ":") {
			continue
		}
		// Scan all remaining fields for a recognisable oper-state keyword.
		// ZTE C320 V2.1: OnuIndex | Admin State | OMCC State | Phase State | Channel
		// Scanning avoids brittle hardcoded column indices across firmware versions.
		//
		// Known ZTE C320 Phase State values:
		//   working / active / online / up    → online
		//   dying-gasp / dyinggasp            → dying_gasp (ONU sent dying-gasp signal)
		//   power-off / power_off / poweroff  → offline (ONU powered off without dying-gasp)
		//   los / lofi / losi                 → los (Loss of Signal / Frame / inner)
		//   auth-failed / auth_failed         → auth_failed
		//   inactive / not-present / offline  → offline
		var status models.OltOnuStatus
		found := false
		for _, f := range fields[1:] {
			f = strings.ToLower(f)
			// Normalize dashes/underscores so "power-off" == "power_off" == "poweroff"
			fNorm := strings.ReplaceAll(strings.ReplaceAll(f, "-", ""), "_", "")
			switch {
			case f == "working" || f == "active" || f == "online" || f == "up":
				status, found = models.OnuOnline, true
			case strings.HasPrefix(f, "dying") || strings.HasPrefix(fNorm, "dyinggasp"):
				status, found = models.OnuDyingGasp, true
			case fNorm == "poweroff" || fNorm == "powerdown":
				status, found = models.OnuOffline, true
			case f == "los" || f == "lofi" || f == "losi":
				status, found = models.OnuLOS, true
			case fNorm == "authfailed":
				status, found = models.OnuAuthFailed, true
			case f == "inactive" || fNorm == "notpresent" || f == "offline" || f == "down":
				status, found = models.OnuOffline, true
			}
			if found {
				break
			}
		}
		// Only override when we positively identified the state.
		if found {
			result[onuIdx] = status
		}
	}
	return result
}

// decodeDeregReason converts the raw ZTE zxAnGponOnuRegDeregReason integer
// (OID .3.50.12.1.1.7) into a human-readable string.
// Returns nil for 0 (no data) and 1 (Unknown) to avoid surfacing noise.
func decodeDeregReason(v int64) *string {
	var s string
	switch v {
	case 2:
		s = "LOS"
	case 3:
		s = "LOSi"
	case 4:
		s = "LOFi"
	case 5:
		s = "SFi"
	case 6:
		s = "LOAi"
	case 7:
		s = "LOAMi"
	case 8:
		s = "AuthFail"
	case 9:
		s = "PowerOff"
	case 10:
		s = "DeactiveSucc"
	case 11:
		s = "DeactiveFail"
	case 12:
		s = "Reboot"
	case 13:
		s = "Shutdown"
	default:
		return nil // 0 = no data, 1 = Unknown → omit
	}
	return &s
}
