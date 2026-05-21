package handlers

// olt_chassis.go — GET /api/olt/:id/chassis
//
// Ported from Next.js: src/app/api/olt/[id]/chassis/route.ts
// Execution strategy: Telnet (show card + show interface port-status) and SNMP
// IF-MIB walks fire in PARALLEL. Real card data from Telnet takes precedence;
// SNMP + DB are used as fallback when Telnet is unavailable.

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	snmputil "github.com/s4lfanet/salfanet-radius-go/internal/olt/snmp"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/telnet"
)

// ── IF-MIB OIDs ───────────────────────────────────────────────────────────────
const (
	oidIfDescr     = "1.3.6.1.2.1.2.2.1.2"              // ifDescr — interface name
	oidIfAdmin     = "1.3.6.1.2.1.2.2.1.7"              // ifAdminStatus (1=up, 2=down)
	oidIfOper      = "1.3.6.1.2.1.2.2.1.8"              // ifOperStatus  (1=up, 2=down)
	oidIfHighSpeed = "1.3.6.1.2.1.31.1.1.1.15"          // ifHighSpeed Mbps
	oidIfAlias     = "1.3.6.1.2.1.31.1.1.1.18"          // ifAlias
	oidZtePONTable = "1.3.6.1.4.1.3902.1012.3.11.3.1.1" // PON index table
)

// ── ZTE chassis constants ─────────────────────────────────────────────────────
const (
	b1BaseInt int64 = 268500992
	b2BaseInt int64 = 268509184
	ponIncInt int64 = 256
)

// ── Types ─────────────────────────────────────────────────────────────────────

type slotType string

const (
	slotMcud    slotType = "mcud"
	slotService slotType = "service"
	slotUplink  slotType = "uplink"
	slotEmpty   slotType = "empty"
)

type cardInfo struct {
	Slot      int
	CardType  string
	CfgType   string
	HardVer   string
	SoftVer   string
	Status    string
	SlotType  slotType
	PortCount int
}

type uplinkPortState struct {
	Iface        string
	AdminStatus  string
	LinkStatus   string
	Speed        string
	PhysicalType string
	Description  string
	IsEnabled    bool
	IsLinked     bool
}

type chassisPort struct {
	Port        int    `json:"port"`
	Iface       string `json:"iface,omitempty"`
	OnuCount    int    `json:"onuCount"`
	OnlineCount int    `json:"onlineCount"`
	HasOnus     bool   `json:"hasOnus"`
	AdminStatus string `json:"adminStatus,omitempty"`
	LinkStatus  string `json:"linkStatus,omitempty"`
	Speed       string `json:"speed,omitempty"`
	PhysType    string `json:"physicalType,omitempty"`
	Description string `json:"description,omitempty"`
	IsEnabled   *bool  `json:"isEnabled,omitempty"`
	IsLinked    *bool  `json:"isLinked,omitempty"`
}

type chassisSlotOut struct {
	Index        int           `json:"index"`
	Label        string        `json:"label"`
	Type         string        `json:"type"`
	Description  string        `json:"description,omitempty"`
	Present      bool          `json:"present"`
	CardType     string        `json:"cardType"`
	HardVer      string        `json:"hardVer,omitempty"`
	SoftVer      string        `json:"softVer,omitempty"`
	CardStatus   string        `json:"cardStatus,omitempty"`
	PortCount    int           `json:"portCount"`
	Ports        []chassisPort `json:"ports"`
	UplinkIfaces []string      `json:"uplinkIfaces,omitempty"`
}

// ── Card classification ───────────────────────────────────────────────────────

func classifyCard(cardType string) slotType {
	ct := strings.ToUpper(cardType)
	if strings.HasPrefix(ct, "MCUD") || strings.HasPrefix(ct, "MCUA") || ct == "MCU" {
		return slotMcud
	}
	if strings.HasPrefix(ct, "SMXA") || strings.HasPrefix(ct, "GICF") ||
		strings.HasPrefix(ct, "GISF") || strings.HasPrefix(ct, "UPLINK") {
		return slotUplink
	}
	return slotService
}

func isOperationalCard(status string) bool {
	if status == "" {
		return true
	}
	re := regexp.MustCompile(`(?i)not\s*install|not\s*present|absent|empty`)
	return !re.MatchString(status)
}

// ── "show card" parser ────────────────────────────────────────────────────────
// Ported from parseShowCard() in Next.js chassis/route.ts.
// Handles both rack-shelf format (ZTE C320 V2.1) and legacy format.
//
// Sample V2.1 output:
//   Rack Shelf Slot CfgType RealType Port HardVer SoftVer Status
//   1    1     1    GTGH    GTGHG    16   V1.0.0  V2.1.0  INSERVICE
//   1    1     4    SMXA             3                     OFFLINE
//
// Sample legacy output:
//   Slot  CardType  HardVer  SoftVer  Status
//   0     MCUD1     V3.0     V2.1.0   Normal
//   1     GTGHG     V1.0     V2.1.0   Normal

func parseShowCard(output string) []cardInfo {
	var cards []cardInfo
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "---") ||
			regexp.MustCompile(`Slot\s+CardType|Rack\s+Shelf\s+Slot|^-`).MatchString(line) {
			continue
		}
		parts := strings.Fields(line)

		// Rack-Shelf format: first 3 tokens are all digits → Rack Shelf Slot
		if len(parts) >= 6 &&
			isDigit(parts[0]) && isDigit(parts[1]) && isDigit(parts[2]) {
			slot, _ := strconv.Atoi(parts[2])
			cfgType := parts[3]
			cardType := cfgType
			portCount := 0
			hardVer, softVer, status := "", "", ""

			if len(parts) > 4 && isDigit(parts[4]) {
				// RealType column is blank; Port shifted into parts[4]
				portCount, _ = strconv.Atoi(parts[4])
				if len(parts) > 5 {
					status = parts[len(parts)-1]
				}
			} else {
				if len(parts) > 4 && parts[4] != "" {
					cardType = parts[4]
				}
				if len(parts) > 5 && isDigit(parts[5]) {
					portCount, _ = strconv.Atoi(parts[5])
				}
				if len(parts) > 6 {
					hardVer = parts[6]
				}
				if len(parts) > 7 {
					softVer = parts[7]
				}
				if len(parts) > 8 {
					status = strings.Join(parts[8:], " ")
				} else {
					status = parts[len(parts)-1]
				}
			}
			cards = append(cards, cardInfo{
				Slot: slot, CardType: cardType, CfgType: cfgType,
				HardVer: hardVer, SoftVer: softVer, Status: status,
				SlotType: classifyCard(cardType), PortCount: portCount,
			})
			continue
		}

		// Legacy format: Slot CardType HardVer SoftVer Status
		if len(parts) < 5 {
			continue
		}
		slot, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		ct := parts[1]
		cards = append(cards, cardInfo{
			Slot: slot, CardType: ct, CfgType: ct,
			HardVer: parts[2], SoftVer: parts[3],
			Status:   strings.Join(parts[4:], " "),
			SlotType: classifyCard(ct),
		})
	}
	return cards
}

func isDigit(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ── SMXA uplink interface names ───────────────────────────────────────────────
// Ported from smxaUplinkPorts() in Next.js chassis/route.ts.

func smxaUplinkIfaces(slot int, cardType string) []string {
	ct := strings.ToUpper(cardType)
	switch ct {
	case "SMXA-B":
		return []string{
			fmt.Sprintf("gei_1/%d/1", slot), fmt.Sprintf("gei_1/%d/2", slot), fmt.Sprintf("gei_1/%d/3", slot),
			fmt.Sprintf("xgei_1/%d/1", slot), fmt.Sprintf("xgei_1/%d/2", slot),
		}
	case "SMXA-A":
		return []string{
			fmt.Sprintf("xgei_1/%d/1", slot), fmt.Sprintf("xgei_1/%d/2", slot),
		}
	case "GICF", "GICF-B":
		return []string{
			fmt.Sprintf("gei_1/%d/1", slot), fmt.Sprintf("gei_1/%d/2", slot),
			fmt.Sprintf("xgei_1/%d/1", slot), fmt.Sprintf("xgei_1/%d/2", slot),
		}
	case "SMXA":
		// ZTE C320 SMXA: 1 GE (no port suffix) + 2 XGE
		return []string{
			fmt.Sprintf("gei_1/%d", slot),
			fmt.Sprintf("xgei_1/%d/1", slot), fmt.Sprintf("xgei_1/%d/2", slot),
		}
	default:
		return []string{
			fmt.Sprintf("gei_1/%d", slot),
			fmt.Sprintf("xgei_1/%d/1", slot), fmt.Sprintf("xgei_1/%d/2", slot),
		}
	}
}

// ── "show interface port-status" parser ──────────────────────────────────────
// Ported from parseUplinkPortStatusTable() in Next.js chassis/route.ts.

func parseUplinkPortStatus(output string, ifaces []string) map[string]uplinkPortState {
	wanted := make(map[string]bool, len(ifaces))
	for _, iface := range ifaces {
		wanted[strings.ToLower(iface)] = true
	}
	result := map[string]uplinkPortState{}
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || regexp.MustCompile(`^-+$|^Port\s+|^Status\s+`).MatchString(line) {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 11 {
			continue
		}
		iface := parts[0]
		if !wanted[strings.ToLower(iface)] {
			continue
		}
		// Column layout (0-indexed):
		// [0]=Port [1]=PhType [2]=Speed [3]=Duplex [4]=ActualSpeed [5]=FEC
		// [6]=CRC16 [7]=Pause [8]=FlowCtrl [9]=AdminStatus [10]=LinkStatus
		adminRaw := parts[9]
		linkRaw := parts[10]
		adminStatus := adminRaw
		if strings.EqualFold(adminRaw, "activate") {
			adminStatus = "Up"
		} else if strings.EqualFold(adminRaw, "deactivate") {
			adminStatus = "Down"
		}
		linkStatus := linkRaw
		if strings.EqualFold(linkRaw, "up") {
			linkStatus = "Up"
		} else if strings.EqualFold(linkRaw, "down") {
			linkStatus = "Down"
		}
		speed := ""
		if len(parts) > 4 && parts[4] != "N/A" {
			speed = parts[4] + "M"
		}
		result[iface] = uplinkPortState{
			Iface:        iface,
			AdminStatus:  adminStatus,
			LinkStatus:   linkStatus,
			Speed:        speed,
			PhysicalType: parts[1],
			IsEnabled:    regexp.MustCompile(`(?i)^(up|enable|activate)$`).MatchString(adminStatus),
			IsLinked:     regexp.MustCompile(`(?i)^(up|online)$`).MatchString(linkStatus),
		}
	}
	return result
}

// ── SNMP IF-MIB walks ─────────────────────────────────────────────────────────

type ifMibData struct {
	descr map[string]string // oid suffix → interface name
	admin map[string]string
	oper  map[string]string
	speed map[string]string
	alias map[string]string
}

func walkToMap(ctx context.Context, cfg snmputil.Config, oid string) map[string]string {
	results, err := snmputil.BulkWalk(ctx, cfg, oid)
	if err != nil {
		return nil
	}
	m := make(map[string]string, len(results))
	for _, r := range results {
		parts := strings.Split(r.OID, ".")
		suffix := parts[len(parts)-1]
		m[suffix] = snmputil.ToString(r.Value)
	}
	return m
}

func fetchIfMib(ctx context.Context, cfg snmputil.Config) *ifMibData {
	type walkJob struct {
		oid string
		out *map[string]string
	}
	data := &ifMibData{}
	jobs := []walkJob{
		{oidIfDescr, &data.descr},
		{oidIfAdmin, &data.admin},
		{oidIfOper, &data.oper},
		{oidIfHighSpeed, &data.speed},
		{oidIfAlias, &data.alias},
	}
	var wg sync.WaitGroup
	for _, j := range jobs {
		j := j
		wg.Add(1)
		go func() {
			defer wg.Done()
			*j.out = walkToMap(ctx, cfg, j.oid)
		}()
	}
	wg.Wait()
	if data.descr == nil {
		return nil
	}
	return data
}

// buildUplinkStatesFromSNMP derives uplink port states from pre-walked IF-MIB data.
// Ported from buildUplinkStatesFromSNMP() in Next.js chassis/route.ts.
func buildUplinkStatesFromSNMP(data *ifMibData, ifaces []string) map[string]uplinkPortState {
	result := map[string]uplinkPortState{}
	if data == nil {
		return result
	}
	normalize := func(s string) string {
		return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, "-", "_"), " ", "_"))
	}
	// Build ifName → ifIndex from ifDescr walk
	nameToIdx := map[string]string{}
	for suffix, name := range data.descr {
		nameToIdx[normalize(name)] = suffix
	}
	for _, iface := range ifaces {
		idx, ok := nameToIdx[normalize(iface)]
		if !ok {
			continue
		}
		adminVal, _ := strconv.Atoi(data.admin[idx])
		operVal, _ := strconv.Atoi(data.oper[idx])
		speed := ""
		if s := data.speed[idx]; s != "" && s != "0" {
			speed = s + "M"
		}
		desc := data.alias[idx]
		isEnabled := adminVal == 1
		isLinked := operVal == 1
		adminStr := "Down"
		if isEnabled {
			adminStr = "Up"
		}
		linkStr := "Down"
		if isLinked {
			linkStr = "Up"
		}
		result[iface] = uplinkPortState{
			Iface:       iface,
			AdminStatus: adminStr,
			LinkStatus:  linkStr,
			Speed:       speed,
			Description: desc,
			IsEnabled:   isEnabled,
			IsLinked:    isLinked,
		}
	}
	return result
}

// ── PON table walk → board presence ──────────────────────────────────────────

func snmpBoardsPresent(ctx context.Context, cfg snmputil.Config) map[int]bool {
	results, err := snmputil.Walk(ctx, cfg, oidZtePONTable)
	if err != nil {
		return nil
	}
	boards := map[int]bool{}
	for _, r := range results {
		// OID contains ponIndex encoded as large integer
		for _, part := range strings.Split(r.OID, ".") {
			n, err := strconv.ParseInt(part, 10, 64)
			if err != nil || n <= 268000000 {
				continue
			}
			if n > b1BaseInt && n < b1BaseInt+128*ponIncInt {
				pon := (n - b1BaseInt) / ponIncInt
				if (n-b1BaseInt)%ponIncInt == 0 && pon > 0 {
					boards[1] = true
				}
			}
			if n > b2BaseInt && n < b2BaseInt+128*ponIncInt {
				pon := (n - b2BaseInt) / ponIncInt
				if (n-b2BaseInt)%ponIncInt == 0 && pon > 0 {
					boards[2] = true
				}
			}
		}
	}
	return boards
}

// ── GetChassis handler ────────────────────────────────────────────────────────

// GetChassis godoc
// GET /api/olt/:id/chassis
// Returns the physical chassis slot layout.
// Strategy (mirrors Next.js):
//  1. Fire Telnet (show card + show interface port-status) and all SNMP walks IN PARALLEL.
//  2. If Telnet succeeds → use real card data as authoritative source.
//  3. Else → fall back to SNMP board presence + DB ONU port data.
func (h *OLTHandler) GetChassis(c fiber.Ctx) error {
	id := c.Params("id")

	// ── Step 0: fetch OLT from DB ─────────────────────────────────────────────
	var olt models.NetworkOLT
	if err := h.db.First(&olt, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "OLT not found"})
	}

	// Load ONU statuses
	type onuRow struct {
		Slot   int
		Port   int
		Status string
	}
	var onuRows []onuRow
	h.db.Raw("SELECT slot, port, status FROM olt_onu_status WHERE oltId = ?", id).Scan(&onuRows)

	// ── Step 1: Build SNMP and Telnet configs ─────────────────────────────────
	var snmpCfg *snmputil.Config
	if olt.SNMPEnabled {
		community := "public"
		if olt.SNMPCommunity != "" {
			community = olt.SNMPCommunity
		}
		port := 161
		if olt.SNMPPort > 0 {
			port = olt.SNMPPort
		}
		cfg := snmputil.DefaultConfig(olt.IPAddress, community, port)
		snmpCfg = &cfg
	}

	var telnetPool *telnet.Pool
	if (olt.TelnetEnabled || olt.SSHEnabled) && olt.Username != nil && olt.Password != nil {
		tport := olt.TelnetPort
		if tport == 0 {
			tport = 23
		}
		tcfg := telnet.DefaultConfig(olt.IPAddress, tport, *olt.Username, *olt.Password)
		tcfg.CommandTimeout = 20 * time.Second
		telnetPool = telnet.NewPool(tcfg)
	}
	defer func() {
		if telnetPool != nil {
			telnetPool.Close()
		}
	}()

	// ── Step 2: Fire Telnet + all SNMP walks IN PARALLEL ─────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	type telnetResult struct {
		showCard   string
		portStatus string
		err        error
	}
	telnetCh := make(chan telnetResult, 1)
	snmpIfCh := make(chan *ifMibData, 1)
	snmpBoardCh := make(chan map[int]bool, 1)

	// Telnet: show card + show interface port-status
	go func() {
		if telnetPool == nil {
			telnetCh <- telnetResult{err: fmt.Errorf("telnet disabled")}
			return
		}
		out, err := telnetPool.ExecuteMultiple([]string{"show card", "show interface port-status"})
		if err != nil {
			log.Warn().Err(err).Str("olt", id).Msg("chassis: telnet failed")
			telnetCh <- telnetResult{err: err}
			return
		}
		// Split output at the second command boundary (prompt between commands)
		parts := splitAtPrompt(out)
		showCard, portStatus := "", ""
		if len(parts) >= 1 {
			showCard = parts[0]
		}
		if len(parts) >= 2 {
			portStatus = parts[1]
		}
		telnetCh <- telnetResult{showCard: showCard, portStatus: portStatus}
	}()

	// SNMP: IF-MIB parallel walks
	go func() {
		if snmpCfg == nil {
			snmpIfCh <- nil
			return
		}
		snmpIfCh <- fetchIfMib(ctx, *snmpCfg)
	}()

	// SNMP: ZTE PON table walk → board presence
	go func() {
		if snmpCfg == nil {
			snmpBoardCh <- nil
			return
		}
		snmpBoardCh <- snmpBoardsPresent(ctx, *snmpCfg)
	}()

	telnetRes := <-telnetCh
	ifMib := <-snmpIfCh
	snmpBoards := <-snmpBoardCh

	// ── Step 3: Build DB slot data ────────────────────────────────────────────
	type slotDBData struct {
		maxPort int
		ports   map[int]struct{ onuCount, onlineCount int }
	}
	slotDB := map[int]*slotDBData{}
	for _, onu := range onuRows {
		if _, ok := slotDB[onu.Slot]; !ok {
			slotDB[onu.Slot] = &slotDBData{ports: map[int]struct{ onuCount, onlineCount int }{}}
		}
		d := slotDB[onu.Slot]
		p := d.ports[onu.Port]
		p.onuCount++
		if onu.Status == "online" {
			p.onlineCount++
		}
		d.ports[onu.Port] = p
		if onu.Port > d.maxPort {
			d.maxPort = onu.Port
		}
	}
	// Determine card type from maxPort for DB-only fallback
	inferCardType := func(maxPort int) (string, int) {
		if maxPort <= 4 {
			return "GTGO", 4
		} else if maxPort <= 8 {
			return "GTGH", 8
		}
		return "GTGQ", 16
	}

	buildServicePorts := func(slot int, portCount int) []chassisPort {
		ports := make([]chassisPort, portCount+1) // 0-based slice; port indices 0..portCount
		for i := range ports {
			ports[i] = chassisPort{Port: i}
		}
		if d, ok := slotDB[slot]; ok {
			for portIdx, p := range d.ports {
				if portIdx < len(ports) {
					e := false
					ports[portIdx] = chassisPort{
						Port:        portIdx,
						OnuCount:    p.onuCount,
						OnlineCount: p.onlineCount,
						HasOnus:     p.onuCount > 0,
						IsEnabled:   &e,
					}
				}
			}
		}
		return ports
	}

	// ── Step 4: Build chassis ─────────────────────────────────────────────────

	// 4a. Telnet path: real "show card" data available
	if telnetRes.err == nil && len(telnetRes.showCard) > 20 {
		telnetCards := parseShowCard(telnetRes.showCard)
		if len(telnetCards) > 0 {
			cardMap := map[int]cardInfo{}
			for _, c := range telnetCards {
				cardMap[c.Slot] = c
			}

			// Collect uplink ifaces per slot
			uplinkIfacesBySlot := map[int][]string{}
			for _, card := range telnetCards {
				if card.SlotType == slotUplink {
					uplinkIfacesBySlot[card.Slot] = smxaUplinkIfaces(card.Slot, card.CardType)
				}
			}
			allIfaces := []string{}
			for _, ifaces := range uplinkIfacesBySlot {
				allIfaces = append(allIfaces, ifaces...)
			}

			// Parse port-status Telnet output first; fall back to SNMP IF-MIB
			uplinkStates := parseUplinkPortStatus(telnetRes.portStatus, allIfaces)
			if len(uplinkStates) == 0 && len(allIfaces) > 0 && ifMib != nil {
				uplinkStates = buildUplinkStatesFromSNMP(ifMib, allIfaces)
			} else if ifMib != nil {
				// Merge SNMP ifAlias descriptions into Telnet-parsed states
				snmpStates := buildUplinkStatesFromSNMP(ifMib, allIfaces)
				for iface, st := range uplinkStates {
					if snmpSt, ok := snmpStates[iface]; ok && snmpSt.Description != "" {
						st.Description = snmpSt.Description
						uplinkStates[iface] = st
					}
				}
			}

			// Determine slot range
			maxSlot := 17
			for _, c := range telnetCards {
				if c.Slot > maxSlot {
					maxSlot = c.Slot
				}
			}

			var chassis []chassisSlotOut
			allSlots := map[int]bool{0: true, 17: true}
			for _, c := range telnetCards {
				allSlots[c.Slot] = true
			}
			for slotIdx := 0; slotIdx <= maxSlot; slotIdx++ {
				if !allSlots[slotIdx] {
					continue
				}
				card, hasCard := cardMap[slotIdx]
				isMCU := slotIdx == 0 || slotIdx == 17
				operational := hasCard && isOperationalCard(card.Status)
				var sType slotType
				if operational {
					sType = card.SlotType
				} else if isMCU {
					sType = slotMcud
				} else if _, inDB := slotDB[slotIdx]; inDB {
					sType = slotService
				} else {
					sType = slotEmpty
				}

				cardType := "empty"
				label := fmt.Sprintf("S%d", slotIdx)
				if isMCU {
					cardType = "MCUD1"
					if slotIdx == 0 {
						label = "MCU-A"
					} else {
						label = "MCU-B"
					}
				} else if operational {
					cardType = card.CardType
					if sType == slotUplink {
						label = fmt.Sprintf("UPL-%d", slotIdx)
					}
				}

				var ports []chassisPort
				portCount := 0

				switch sType {
				case slotService:
					if dbD, ok := slotDB[slotIdx]; ok {
						_, stdPorts := inferCardType(dbD.maxPort)
						portCount = stdPorts + 1
						ports = buildServicePorts(slotIdx, stdPorts)
					} else if card.PortCount > 0 {
						portCount = card.PortCount
					} else {
						portCount = 16
					}
				case slotUplink:
					ifaces := uplinkIfacesBySlot[slotIdx]
					if len(ifaces) == 0 {
						ifaces = smxaUplinkIfaces(slotIdx, cardType)
					}
					portCount = len(ifaces)
					ports = make([]chassisPort, portCount)
					for pi, iface := range ifaces {
						p := chassisPort{Port: pi, Iface: iface}
						if st, ok := uplinkStates[iface]; ok {
							p.AdminStatus = st.AdminStatus
							p.LinkStatus = st.LinkStatus
							p.Speed = st.Speed
							p.PhysType = st.PhysicalType
							p.Description = st.Description
							t1, t2 := st.IsEnabled, st.IsLinked
							p.IsEnabled = &t1
							p.IsLinked = &t2
						}
						ports[pi] = p
					}
				}

				cardStatus := ""
				hardVer, softVer := "", ""
				desc := cardType
				if hasCard {
					cardStatus = card.Status
					hardVer = card.HardVer
					softVer = card.SoftVer
					desc = fmt.Sprintf("%s (%s)", card.CardType, card.Status)
				}

				chassis = append(chassis, chassisSlotOut{
					Index:       slotIdx,
					Label:       label,
					Type:        string(sType),
					Description: desc,
					Present:     operational || (isMCU && slotIdx == 0),
					CardType:    cardType,
					HardVer:     hardVer,
					SoftVer:     softVer,
					CardStatus:  cardStatus,
					PortCount:   portCount,
					Ports:       ports,
					UplinkIfaces: func() []string {
						if sType == slotUplink {
							return uplinkIfacesBySlot[slotIdx]
						}
						return nil
					}(),
				})
			}

			// Sort by slot index
			sortChassisSlots(chassis)
			vendorStr, modelStr := derefStr(olt.Vendor), derefStr(olt.Model)
			return c.JSON(fiber.Map{
				"success": true, "chassis": chassis,
				"vendor": vendorStr, "model": modelStr, "source": "telnet",
			})
		}
	}

	// ── 4b. Fallback: SNMP board presence + DB data ───────────────────────────
	usedSlots := map[int]bool{0: true, 17: true}
	for slotIdx := range slotDB {
		usedSlots[slotIdx] = true
	}
	for board := range snmpBoards {
		usedSlots[board] = true
	}

	// Add standard uplink slots if we have GPON boards
	if len(slotDB) > 0 || len(snmpBoards) > 0 {
		usedSlots[15] = true // standard ZTE C320 uplink slot
	}

	sortedSlots := sortedKeys(usedSlots)
	var chassis []chassisSlotOut
	for _, slotIdx := range sortedSlots {
		isMCU := slotIdx == 0 || slotIdx == 17
		isUplinkSlot := slotIdx == 15 || slotIdx == 16
		dbD, hasDB := slotDB[slotIdx]
		_, hasBoard := snmpBoards[slotIdx]

		var sType slotType
		var cardType, label string
		present := false

		if isMCU {
			sType = slotMcud
			cardType = "MCUD1"
			present = slotIdx == 0
			if slotIdx == 0 {
				label = "MCU-A"
			} else {
				label = "MCU-B"
			}
		} else if isUplinkSlot {
			sType = slotUplink
			cardType = "GICF"
			label = fmt.Sprintf("UPL-%d", slotIdx)
			present = true
		} else {
			sType = slotService
			if hasDB {
				ct, _ := inferCardType(dbD.maxPort)
				cardType = ct
			} else if hasBoard {
				cardType = "GTGQ"
			} else {
				cardType = "GTGQ"
			}
			label = fmt.Sprintf("S%d", slotIdx)
			present = hasDB || hasBoard
		}

		var ports []chassisPort
		portCount := 0

		switch sType {
		case slotService:
			if hasDB {
				_, stdPorts := inferCardType(dbD.maxPort)
				portCount = stdPorts + 1
				ports = buildServicePorts(slotIdx, stdPorts)
			}
		case slotUplink:
			// Build uplink port states from SNMP if available
			ifaces := smxaUplinkIfaces(slotIdx, cardType)
			portCount = len(ifaces)
			ports = make([]chassisPort, portCount)
			var uplinkStates map[string]uplinkPortState
			if ifMib != nil {
				uplinkStates = buildUplinkStatesFromSNMP(ifMib, ifaces)
			}
			for pi, iface := range ifaces {
				p := chassisPort{Port: pi, Iface: iface}
				if st, ok := uplinkStates[iface]; ok {
					p.AdminStatus = st.AdminStatus
					p.LinkStatus = st.LinkStatus
					p.Speed = st.Speed
					t1, t2 := st.IsEnabled, st.IsLinked
					p.IsEnabled = &t1
					p.IsLinked = &t2
				}
				ports[pi] = p
			}
		}

		chassis = append(chassis, chassisSlotOut{
			Index:       slotIdx,
			Label:       label,
			Type:        string(sType),
			Description: cardType,
			Present:     present,
			CardType:    cardType,
			PortCount:   portCount,
			Ports:       ports,
		})
	}

	return c.JSON(fiber.Map{
		"success": true, "chassis": chassis,
		"vendor": derefStr(olt.Vendor), "model": derefStr(olt.Model), "source": "snmp+db",
	})
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// splitAtPrompt splits Telnet multi-command output at ZTE shell prompts (#).
// Returns one string per command output section.
func splitAtPrompt(output string) []string {
	promptRe := regexp.MustCompile(`(?m)\bZXAN#|\b[A-Z0-9_-]+#\s*$`)
	parts := promptRe.Split(output, -1)
	if len(parts) > 0 && strings.TrimSpace(parts[0]) == "" {
		parts = parts[1:] // skip leader before first prompt
	}
	return parts
}

func sortChassisSlots(slots []chassisSlotOut) {
	for i := 0; i < len(slots); i++ {
		for j := i + 1; j < len(slots); j++ {
			if slots[j].Index < slots[i].Index {
				slots[i], slots[j] = slots[j], slots[i]
			}
		}
	}
}

func sortedKeys(m map[int]bool) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
