// Package poller manages per-OLT polling goroutines.
//
// Each OLT runs its own polling goroutine at the configured interval.
// Results are upserted into olt_onu_status and broadcast to the WebSocket hub.
package poller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/notify"
	snmputil "github.com/s4lfanet/salfanet-radius-go/internal/olt/snmp"
	sshclient "github.com/s4lfanet/salfanet-radius-go/internal/olt/ssh"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/telnet"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/vendors/zte"
)

// BroadcastFn is a function that broadcasts ONU status updates to WebSocket clients.
type BroadcastFn func(oltID string, data interface{})

// Poller manages polling goroutines for all OLTs.
type Poller struct {
	db        *gorm.DB
	broadcast BroadcastFn

	mu       sync.Mutex
	workers  map[string]context.CancelFunc // oltID → cancel
	pools    map[string]*telnet.Pool       // oltID → telnet pool (legacy)
	sshPools map[string]*sshclient.Pool    // oltID → SSH pool (preferred)
}

// New creates a new Poller.
func New(db *gorm.DB, broadcast BroadcastFn) *Poller {
	return &Poller{
		db:        db,
		broadcast: broadcast,
		workers:   make(map[string]context.CancelFunc),
		pools:     make(map[string]*telnet.Pool),
		sshPools:  make(map[string]*sshclient.Pool),
	}
}

// Start begins polling for the given OLT. It stops any existing poll loop first.
func (p *Poller) Start(olt *models.NetworkOLT) {
	p.Stop(olt.ID) // Stop existing loop if any

	ctx, cancel := context.WithCancel(context.Background())
	p.mu.Lock()
	p.workers[olt.ID] = cancel
	p.mu.Unlock()

	// Build SSH pool (preferred — more reliable than Telnet for ZTE C320).
	// SSH CLI state is the ground truth; SNMP OperState lags significantly.
	if olt.SSHEnabled && olt.Username != nil && olt.Password != nil {
		cfg := sshclient.DefaultConfig(olt.IPAddress, olt.SSHPort, *olt.Username, *olt.Password)
		sshPool := sshclient.New(cfg)
		p.mu.Lock()
		p.sshPools[olt.ID] = sshPool
		p.mu.Unlock()
		log.Info().Str("olt", olt.ID).Int("port", olt.SSHPort).Msg("poller: SSH CLI pool created")
	}

	// Build Telnet pool as fallback when SSH is not configured.
	if !olt.SSHEnabled && olt.TelnetEnabled && olt.Username != nil && olt.Password != nil {
		cfg := telnet.DefaultConfig(olt.IPAddress, olt.TelnetPort, *olt.Username, *olt.Password)
		pool := telnet.NewPool(cfg)
		p.mu.Lock()
		p.pools[olt.ID] = pool
		p.mu.Unlock()
		log.Info().Str("olt", olt.ID).Int("port", olt.TelnetPort).Msg("poller: Telnet CLI pool created (SSH not configured)")
	}

	interval := time.Duration(olt.PollingInterval) * time.Second
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}

	log.Info().Str("olt", olt.ID).Str("ip", olt.IPAddress).Dur("interval", interval).Msg("poller: starting")

	go func() {
		// First poll immediately
		p.poll(ctx, olt)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info().Str("olt", olt.ID).Msg("poller: stopped")
				return
			case <-ticker.C:
				p.poll(ctx, olt)
			}
		}
	}()
}

// Stop cancels the polling goroutine for the given OLT.
func (p *Poller) Stop(oltID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if cancel, ok := p.workers[oltID]; ok {
		cancel()
		delete(p.workers, oltID)
	}
	if pool, ok := p.pools[oltID]; ok {
		pool.Close()
		delete(p.pools, oltID)
	}
	if sshPool, ok := p.sshPools[oltID]; ok {
		sshPool.Close()
		delete(p.sshPools, oltID)
	}
}

// StopAll cancels all running pollers.
func (p *Poller) StopAll() {
	p.mu.Lock()
	ids := make([]string, 0, len(p.workers))
	for id := range p.workers {
		ids = append(ids, id)
	}
	p.mu.Unlock()
	for _, id := range ids {
		p.Stop(id)
	}
}

// StartAll loads all enabled OLTs from the DB and starts their pollers.
func (p *Poller) StartAll() {
	var olts []models.NetworkOLT
	if err := p.db.Where("monitoringEnabled = ?", true).Find(&olts).Error; err != nil {
		log.Error().Err(err).Msg("poller: failed to load OLTs")
		return
	}
	log.Info().Int("count", len(olts)).Msg("poller: starting all OLT pollers")
	for i := range olts {
		p.Start(&olts[i])
	}
}

// GetPool returns the persistent Telnet pool for the given OLT, or nil if none exists.
// Callers must NOT close the returned pool — it is managed by the Poller.
func (p *Poller) GetPool(oltID string) *telnet.Pool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pools[oltID]
}

// GetCLIPool returns the best available CLI pool for the given OLT.
// SSH pool is preferred when available; Telnet pool is returned as fallback.
// Returns nil when neither is configured.
// Callers must NOT close the returned pool — it is managed by the Poller.
func (p *Poller) GetCLIPool(oltID string) zte.CLIPool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if sp, ok := p.sshPools[oltID]; ok {
		return sp
	}
	if tp, ok := p.pools[oltID]; ok {
		return tp
	}
	return nil
}

// TriggerPoll triggers an immediate poll for the given OLT (used by manual sync endpoint).
func (p *Poller) TriggerPoll(oltID string) error {
	var olt models.NetworkOLT
	if err := p.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return fmt.Errorf("OLT not found: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	p.poll(ctx, &olt)
	return nil
}

// ─── Core poll logic ──────────────────────────────────────────────────────────

func (p *Poller) poll(ctx context.Context, olt *models.NetworkOLT) {
	start := time.Now()
	log.Debug().Str("olt", olt.ID).Str("ip", olt.IPAddress).Msg("poller: poll start")

	snmpCfg := snmputil.DefaultConfig(olt.IPAddress, olt.SNMPCommunity, olt.SNMPPort)

	// Dynamic PON port discovery + BulkWalk all ONU tables (registered + unregistered via SNMP).
	onus, err := zte.DiscoverAll(ctx, snmpCfg)
	if err != nil {
		log.Error().Err(err).Str("olt", olt.ID).Msg("poller: ONU discovery failed")
	}

	// ── CLI enrichment (SSH preferred, Telnet fallback) ─────────────────────
	// ZTE C320 SNMP OperState lags significantly — the CLI "show gpon onu state"
	// command reflects the real state immediately and is the authoritative source.
	// SSH is preferred because some Mikrotik DST-NAT configs map the "Telnet" port
	// to the OLT's SSH service, causing Telnet login to fail on the SSH banner.
	p.mu.Lock()
	sshPool := p.sshPools[olt.ID]
	telnetPool := p.pools[olt.ID]
	p.mu.Unlock()

	var cliPool zte.CLIPool
	cliProto := ""
	if sshPool != nil {
		cliPool = sshPool
		cliProto = "ssh"
	} else if telnetPool != nil {
		cliPool = telnetPool
		cliProto = "telnet"
	}

	if cliPool != nil && len(onus) > 0 {
		// Enrich ONU distances via CLI (more accurate than SNMP equalization delay).
		if cliDist := zte.FetchTelnetDistances(cliPool, onus); len(cliDist) > 0 {
			for _, onu := range onus {
				key := fmt.Sprintf("%d/%d/%d:%d", onu.Frame, onu.Slot, onu.Port, onu.OnuID)
				if d, ok := cliDist[key]; ok {
					onu.Distance = &d
				}
			}
			log.Debug().Str("olt", olt.ID).Str("proto", cliProto).Int("distances", len(cliDist)).Msg("poller: CLI distances enriched")
		}

		// Override SNMP-derived status with authoritative CLI state.
		cliStates := zte.FetchTelnetONUStates(cliPool, onus)
		if len(cliStates) > 0 {
			// Sanity check: if ALL registered ONUs that SNMP reports as online
			// would be overridden to non-online, it's almost certainly a CLI parse
			// error (garbled banner, session contamination) — skip the override.
			snmpOnlineCount := 0
			cliWouldDropOnline := 0
			for _, onu := range onus {
				if !onu.Registered {
					continue
				}
				if onu.Status == models.OnuOnline {
					snmpOnlineCount++
					key := fmt.Sprintf("%d/%d/%d:%d", onu.Frame, onu.Slot, onu.Port, onu.OnuID)
					if state, ok := cliStates[key]; ok && state != models.OnuOnline {
						cliWouldDropOnline++
					}
				}
			}
			if snmpOnlineCount > 0 && cliWouldDropOnline == snmpOnlineCount {
				log.Warn().Str("olt", olt.ID).Str("proto", cliProto).
					Int("snmpOnline", snmpOnlineCount).
					Msg("poller: CLI override skipped — all SNMP-online ONUs would go offline (likely CLI parse error)")
			} else {
				overrideCount := 0
				for _, onu := range onus {
					if !onu.Registered {
						continue
					}
					key := fmt.Sprintf("%d/%d/%d:%d", onu.Frame, onu.Slot, onu.Port, onu.OnuID)
					if state, ok := cliStates[key]; ok && state != onu.Status {
						onu.Status = state
						overrideCount++
					}
				}
				if overrideCount > 0 {
					log.Info().Str("olt", olt.ID).Str("proto", cliProto).Int("overridden", overrideCount).Msg("poller: CLI ONU states applied")
				}
			}
		} else {
			// CLI returned 0 states — connection failed or output unparseable.
			// We trust SNMP as-is (DO NOT fall back to DB — that causes a death
			// spiral where stale offline statuses accumulate indefinitely).
			log.Warn().Str("olt", olt.ID).Str("proto", cliProto).Msg("poller: CLI returned 0 ONU states — using SNMP status as-is")
		}
	}

	// Fetch sysUpTime via SNMP (OID 1.3.6.1.2.1.1.3.0, value in centiseconds)
	uptimeSeconds := int64(0)
	if uptimeResults, uptimeErr := snmputil.Get(ctx, snmpCfg, []string{"1.3.6.1.2.1.1.3.0"}); uptimeErr == nil && len(uptimeResults) > 0 {
		if v, ok := snmputil.ToInt(uptimeResults[0].Value); ok && v > 0 {
			uptimeSeconds = v / 100 // centiseconds → seconds
		}
	}

	now := time.Now()
	var registeredStatuses []models.OLTONUStatus
	var unregisteredStatuses []models.OLTONUStatus
	onlineCount := 0
	offlineCount := 0
	unregisteredCount := 0

	for _, onu := range onus {
		base := models.OLTONUStatus{
			ID:         uuid.NewString(),
			OltID:      olt.ID,
			OnuIndex:   onu.Frame*100000 + onu.Slot*10000 + onu.Port*100 + onu.OnuID,
			Frame:      onu.Frame,
			Slot:       onu.Slot,
			Port:       onu.Port,
			OnuID:      onu.OnuID,
			Status:     onu.Status,
			LastSeenAt: &now,
			UpdatedAt:  now,
		}

		if onu.Registered {
			if onu.SerialNumber != "" {
				base.SerialNumber = &onu.SerialNumber
			}
			if onu.Description != "" {
				base.Description = &onu.Description
			}
			base.RxPower = onu.RxPower
			base.TxPower = onu.TxPower
			base.Distance = onu.Distance
			base.LastDeregReason = onu.LastDeregReason
			if onu.Status == models.OnuOnline {
				onlineCount++
			} else {
				offlineCount++
			}
			registeredStatuses = append(registeredStatuses, base)
		} else {
			unregisteredCount++
			unregisteredStatuses = append(unregisteredStatuses, base)
		}
	}

	// Upsert registered ONUs with full column set.
	// COALESCE ensures we never overwrite an existing description/serialNumber
	// with NULL when the OLT didn't return one for that poll cycle.
	// Same for optical data (rxPower, txPower, distance): if SNMP/Telnet
	// returns null this cycle (OLT busy, timing, etc.), keep the last valid
	// value so the UI never shows stale-then-blank flicker.
	if len(registeredStatuses) > 0 {
		if e := p.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "oltId"}, {Name: "frame"}, {Name: "slot"}, {Name: "port"}, {Name: "onuId"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"serialNumber":    gorm.Expr("COALESCE(VALUES(serialNumber), serialNumber)"),
				"description":     gorm.Expr("COALESCE(VALUES(description), description)"),
				"status":          gorm.Expr("VALUES(status)"),
				"rxPower":         gorm.Expr("COALESCE(VALUES(rxPower), rxPower)"),
				"txPower":         gorm.Expr("COALESCE(VALUES(txPower), txPower)"),
				"distance":        gorm.Expr("COALESCE(VALUES(distance), distance)"),
				"lastDeregReason": gorm.Expr("COALESCE(VALUES(lastDeregReason), lastDeregReason)"),
				"lastSeenAt":      gorm.Expr("VALUES(lastSeenAt)"),
				"updatedAt":       gorm.Expr("VALUES(updatedAt)"),
			}),
		}).CreateInBatches(registeredStatuses, 100).Error; e != nil {
			log.Error().Err(e).Str("olt", olt.ID).Msg("poller: upsert registered ONUs failed")
		}
	}

	// Upsert unregistered ONUs — only update status and timestamps,
	// preserving any previously known serialNumber/description in the DB.
	if len(unregisteredStatuses) > 0 {
		if e := p.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "oltId"}, {Name: "frame"}, {Name: "slot"}, {Name: "port"}, {Name: "onuId"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"status", "lastSeenAt", "updatedAt",
			}),
		}).CreateInBatches(unregisteredStatuses, 100).Error; e != nil {
			log.Error().Err(e).Str("olt", olt.ID).Msg("poller: upsert unregistered ONUs failed")
		}
	}

	// ── Ghost ONU cleanup ─────────────────────────────────────────────────────
	// Registered ONUs that vanished from the SNMP walk entirely (e.g. ZTE removed
	// them from the reg table after a clean shutdown) still hold their old 'online'
	// status in the DB.  We identify them by updatedAt < start: the upsert above
	// sets updatedAt=now for every ONU visible in this poll; anything older was
	// not touched and is therefore a "ghost".
	//
	// IMPORTANT: Only run when SNMP discovery returned at least 1 ONU. If the
	// walk returned 0, it almost certainly means the OLT is temporarily
	// unreachable (network blip, SNMP timeout, etc.) — NOT that all ONUs
	// disappeared. Running ghost cleanup on an empty result would incorrectly
	// mark every previously-online ONU as offline.
	//
	// NOTE: We use Table("olt_onu_status") — NOT Model(&OLTONUStatus{}) — because
	// GORM would silently append "WHERE id = ''" when the primary-key field is empty,
	// making the UPDATE a no-op.
	allStatuses := append(registeredStatuses, unregisteredStatuses...)

	snmpDiscoveredAny := len(registeredStatuses)+len(unregisteredStatuses) > 0
	var ghostOnus []models.OLTONUStatus
	if snmpDiscoveredAny {
		if err := p.db.WithContext(ctx).
			Where("oltId = ? AND status NOT IN (?, ?) AND updatedAt < ?",
				olt.ID, string(models.OnuOffline), string(models.OnuUnregistered), start).
			Find(&ghostOnus).Error; err == nil && len(ghostOnus) > 0 {

			p.db.WithContext(ctx).Table("olt_onu_status").
				Where("oltId = ? AND status NOT IN (?, ?) AND updatedAt < ?",
					olt.ID, string(models.OnuOffline), string(models.OnuUnregistered), start).
				Updates(map[string]interface{}{
					"status":    string(models.OnuOffline),
					"updatedAt": now,
				})

			log.Debug().Str("olt", olt.ID).Int("count", len(ghostOnus)).Msg("poller: ghost ONUs marked offline")
			for i := range ghostOnus {
				ghostOnus[i].Status = models.OnuOffline
				// Ghost ONUs are NOT counted in offlineCount — they are stale DB entries
				// that vanished from SNMP, not live offline events. Counting them would
				// inflate the network_olts.offlineOnu summary counter incorrectly.
			}
			// Include ghost ONUs in allStatuses so checkAlerts can still raise offline alerts.
			allStatuses = append(allStatuses, ghostOnus...)
		}
		// Extended ghost cleanup: remove persistent ghost ONUs that have been offline
		// for >2 hours and have no customer assigned. These are stale DB rows from
		// ONUs that were physically removed from the OLT. Keeping them indefinitely
		// inflates the ONU list with phantom entries.
		// We ONLY delete if:
		//   - status is already offline (handled by the ghost cleanup above OR previous polls)
		//   - customerId IS NULL (safe to delete — not assigned to any customer)
		//   - lastSeenAt < 2 hours ago OR lastSeenAt IS NULL (truly gone)
		//   - updatedAt < start (not touched in this poll cycle)
		oldThreshold := start.Add(-2 * time.Hour)
		var deletedGhosts int64
		p.db.WithContext(ctx).Table("olt_onu_status").
			Where("oltId = ? AND status = ? AND customerId IS NULL AND (lastSeenAt IS NULL OR lastSeenAt < ?) AND updatedAt < ?",
				olt.ID, string(models.OnuOffline), oldThreshold, start).
			Count(&deletedGhosts)
		if deletedGhosts > 0 {
			p.db.WithContext(ctx).Table("olt_onu_status").
				Where("oltId = ? AND status = ? AND customerId IS NULL AND (lastSeenAt IS NULL OR lastSeenAt < ?) AND updatedAt < ?",
					olt.ID, string(models.OnuOffline), oldThreshold, start).
				Delete(nil)
			log.Info().Str("olt", olt.ID).Int64("deleted", deletedGhosts).Msg("poller: stale ghost ONUs deleted (offline >2h, no customer)")
		}
	} else {
		log.Warn().Str("olt", olt.ID).Msg("poller: SNMP returned 0 ONUs — skipping ghost cleanup to avoid false offline marking")
	}

	// totalONU: only count SNMP-discovered ONUs (registered + unregistered).
	// Ghost ONUs are in allStatuses for alert purposes but must NOT inflate the
	// live summary counter — they are stale DB entries, not real ONUs on the OLT.
	totalONU := len(registeredStatuses) + len(unregisteredStatuses)

	// Update OLT summary.
	// isOnline: true when SNMP returned at least 1 ONU OR when uptime > 0.
	// When SNMP returns 0 ONUs and uptime is 0, the OLT is unreachable — mark offline.
	oltIsOnline := snmpDiscoveredAny || uptimeSeconds > 0
	pollTime := now
	p.db.Model(olt).Updates(map[string]interface{}{
		"lastPollAt": pollTime,
		"totalOnu":   totalONU,
		"onlineOnu":  onlineCount,
		"offlineOnu": offlineCount,
		"isOnline":   oltIsOnline,
		"uptime":     uptimeSeconds,
	})

	// Generate alerts for newly offline ONUs
	p.checkAlerts(ctx, olt, allStatuses)

	duration := time.Since(start)
	log.Debug().
		Str("olt", olt.ID).
		Int("total", totalONU).
		Int("online", onlineCount).
		Int("offline", offlineCount).
		Int("unregistered", unregisteredCount).
		Dur("took", duration).
		Msg("poller: poll done")

	// Broadcast to WebSocket clients
	if p.broadcast != nil {
		p.broadcast(olt.ID, map[string]interface{}{
			"type":         "olt_status",
			"oltId":        olt.ID,
			"total":        totalONU,
			"online":       onlineCount,
			"offline":      offlineCount,
			"unregistered": unregisteredCount,
			"polledAt":     now,
		})
	}
}

// checkAlerts creates/resolves alert records and sends WA+Telegram notifications.
func (p *Poller) checkAlerts(ctx context.Context, olt *models.NetworkOLT, statuses []models.OLTONUStatus) {
	// Fetch real DB IDs for all registered ONUs on this OLT.
	// (The upsert generates a new UUID per poll; the DB may keep the original.)
	type onuKey struct{ frame, slot, port, onuID int }
	var dbONUs []models.OLTONUStatus
	p.db.WithContext(ctx).
		Where("oltId = ?", olt.ID).
		Select("id, frame, slot, port, onuId").
		Find(&dbONUs)
	realIDs := make(map[onuKey]string, len(dbONUs))
	for _, o := range dbONUs {
		realIDs[onuKey{o.Frame, o.Slot, o.Port, o.OnuID}] = o.ID
	}

	// Track per-port Rx power for bulk degradation check.
	type portStat struct {
		total    int
		degraded int // Rx < -27 dBm
	}
	portRx := make(map[int]*portStat) // key: port number

	for _, s := range statuses {
		realID, ok := realIDs[onuKey{s.Frame, s.Slot, s.Port, s.OnuID}]
		if !ok {
			continue // ONU not yet committed to DB
		}

		switch s.Status {
		case models.OnuOffline:
			// Resolve any open dying gasp alert — if the ONU transitions back to a plain
			// offline state, dying gasp is no longer the active condition.
			resolvOffline := time.Now()
			var openDyingGasp models.OLTAlert
			if err := p.db.WithContext(ctx).Where(
				"oltId = ? AND onuId = ? AND alertType = ? AND isResolved = ?",
				olt.ID, realID, models.AlertDyingGasp, false,
			).First(&openDyingGasp).Error; err == nil {
				p.db.WithContext(ctx).Model(&openDyingGasp).Updates(map[string]interface{}{
					"isResolved": true,
					"resolvedAt": &resolvOffline,
				})
			}

			// Create a new offline alert only when there is no open one.
			var existing models.OLTAlert
			err := p.db.WithContext(ctx).Where(
				"oltId = ? AND onuId = ? AND alertType = ? AND isResolved = ?",
				olt.ID, realID, models.AlertONUOffline, false,
			).First(&existing).Error
			if err == nil {
				continue // Already open
			}
			message := fmt.Sprintf("ONU %s (port %d/%d/%d:%d) offline",
				strPtr(s.SerialNumber), s.Frame, s.Slot, s.Port, s.OnuID)
			alert := models.OLTAlert{
				ID:        uuid.NewString(),
				OltID:     &olt.ID,
				OnuID:     &realID,
				AlertType: models.AlertONUOffline,
				Severity:  models.SeverityWarning,
				Message:   message,
			}
			if waErr := p.notifyAlert(olt.Name, message, false); waErr == nil {
				alert.NotifiedViaWhatsapp = true
			}
			p.db.WithContext(ctx).Create(&alert)

		case models.OnuDyingGasp:
			// Resolve any open offline alert for this ONU — dying gasp is more specific
			// than offline, so we don't want both alerts open simultaneously.
			resolvTime := time.Now()
			var openOffline models.OLTAlert
			if err := p.db.WithContext(ctx).Where(
				"oltId = ? AND onuId = ? AND alertType = ? AND isResolved = ?",
				olt.ID, realID, models.AlertONUOffline, false,
			).First(&openOffline).Error; err == nil {
				p.db.WithContext(ctx).Model(&openOffline).Updates(map[string]interface{}{
					"isResolved": true,
					"resolvedAt": &resolvTime,
				})
			}

			// Create a dying gasp alert if none open.
			var existing models.OLTAlert
			err := p.db.WithContext(ctx).Where(
				"oltId = ? AND onuId = ? AND alertType = ? AND isResolved = ?",
				olt.ID, realID, models.AlertDyingGasp, false,
			).First(&existing).Error
			if err == nil {
				continue // Already open
			}
			message := fmt.Sprintf("ONU %s (port %d/%d/%d:%d) dying gasp — kemungkinan listrik mati mendadak",
				strPtr(s.SerialNumber), s.Frame, s.Slot, s.Port, s.OnuID)
			alert := models.OLTAlert{
				ID:        uuid.NewString(),
				OltID:     &olt.ID,
				OnuID:     &realID,
				AlertType: models.AlertDyingGasp,
				Severity:  models.SeverityCritical,
				Message:   message,
			}
			if waErr := p.notifyAlert(olt.Name, message, false); waErr == nil {
				alert.NotifiedViaWhatsapp = true
			}
			p.db.WithContext(ctx).Create(&alert)

		case models.OnuOnline:
			// Resolve any open offline or dying-gasp alert and send recovery notification.
			resolvTime := time.Now()
			for _, alertType := range []models.OltAlertType{models.AlertONUOffline, models.AlertDyingGasp} {
				var existing models.OLTAlert
				if err := p.db.WithContext(ctx).Where(
					"oltId = ? AND onuId = ? AND alertType = ? AND isResolved = ?",
					olt.ID, realID, alertType, false,
				).First(&existing).Error; err == nil {
					p.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
						"isResolved": true,
						"resolvedAt": &resolvTime,
					})
					message := fmt.Sprintf("ONU %s (port %d/%d/%d:%d) kembali online",
						strPtr(s.SerialNumber), s.Frame, s.Slot, s.Port, s.OnuID)
					go p.notifyAlert(olt.Name, message, true) //nolint:errcheck
				}
			}
		}

		// ── Rx power degradation check (online ONUs only) ──────────────────
		if s.Status == models.OnuOnline && s.RxPower != nil {
			const rxThreshold = -27.0 // dBm — "Poor" signal boundary
			if _, ok2 := portRx[s.Port]; !ok2 {
				portRx[s.Port] = &portStat{}
			}
			portRx[s.Port].total++
			if *s.RxPower < rxThreshold {
				portRx[s.Port].degraded++

				// Single-ONU Rx degradation alert (if no open one already).
				var existing models.OLTAlert
				err := p.db.WithContext(ctx).Where(
					"oltId = ? AND onuId = ? AND alertType = ? AND isResolved = ?",
					olt.ID, realID, models.AlertRxDegradation, false,
				).First(&existing).Error
				if err != nil { // no open alert → create one
					msg := fmt.Sprintf("ONU %s (port %d/%d/%d:%d) sinyal lemah: Rx %.2f dBm (ambang -27 dBm)",
						strPtr(s.SerialNumber), s.Frame, s.Slot, s.Port, s.OnuID, *s.RxPower)
					alert := models.OLTAlert{
						ID:        uuid.NewString(),
						OltID:     &olt.ID,
						OnuID:     &realID,
						AlertType: models.AlertRxDegradation,
						Severity:  models.SeverityWarning,
						Message:   msg,
					}
					if waErr := p.notifyAlert(olt.Name, msg, false); waErr == nil {
						alert.NotifiedViaWhatsapp = true
					}
					p.db.WithContext(ctx).Create(&alert)
				}
			} else {
				// Rx recovered — resolve any open single-ONU Rx alert.
				var existing models.OLTAlert
				if err := p.db.WithContext(ctx).Where(
					"oltId = ? AND onuId = ? AND alertType = ? AND isResolved = ?",
					olt.ID, realID, models.AlertRxDegradation, false,
				).First(&existing).Error; err == nil {
					now := time.Now()
					p.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
						"isResolved": true, "resolvedAt": &now,
					})
				}
			}
		}
	}

	// ── Bulk Rx degradation check per PON port ─────────────────────────────
	// If >= 3 ONUs or >= 50% of online ONUs on a port have poor Rx → bulk alert.
	for port, ps := range portRx {
		if ps.total == 0 {
			continue
		}
		pct := float64(ps.degraded) / float64(ps.total)
		isBulk := ps.degraded >= 3 || pct >= 0.5

		// Build a portKey-level OltID to use as onuId placeholder (nil for bulk).
		var existing models.OLTAlert
		err := p.db.WithContext(ctx).Where(
			"oltId = ? AND alertType = ? AND isResolved = ? AND message LIKE ?",
			olt.ID, models.AlertBulkRxDegrade, false, fmt.Sprintf("%%port %d%%", port),
		).First(&existing).Error

		if isBulk && err != nil { // no open bulk alert → create one
			msg := fmt.Sprintf("ODP/Port %d: %d dari %d ONU sinyal lemah (Rx < -27 dBm) — kemungkinan gangguan fiber atau splitter",
				port, ps.degraded, ps.total)
			alert := models.OLTAlert{
				ID:        uuid.NewString(),
				OltID:     &olt.ID,
				AlertType: models.AlertBulkRxDegrade,
				Severity:  models.SeverityCritical,
				Message:   msg,
			}
			if waErr := p.notifyAlert(olt.Name, msg, false); waErr == nil {
				alert.NotifiedViaWhatsapp = true
			}
			p.db.WithContext(ctx).Create(&alert)
		} else if !isBulk && err == nil { // bulk resolved
			now := time.Now()
			p.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
				"isResolved": true, "resolvedAt": &now,
			})
		}
	}
}

// notifyAlert sends an OLT/ONU alert or recovery message via WhatsApp and Telegram.
// isRecovery=true sends a green recovery message; false sends a red alert.
// Returns non-nil error only if WhatsApp sending failed (callers may use this to set NotifiedViaWhatsapp).
func (p *Poller) notifyAlert(oltName, message string, isRecovery bool) error {
	emoji := "🔴"
	label := "ALERT"
	if isRecovery {
		emoji = "🟢"
		label = "PULIH"
	}

	// WhatsApp — send to admin phone stored in Company settings.
	var company models.Company
	var waErr error
	if err := p.db.Select("adminPhone").First(&company).Error; err == nil &&
		company.AdminPhone != nil && *company.AdminPhone != "" {
		waMsg := fmt.Sprintf("%s *[%s] %s*\n%s", emoji, oltName, label, message)
		waErr = notify.Send(*company.AdminPhone, waMsg)
		if waErr != nil {
			log.Warn().Err(waErr).Msg("poller: WA alert send failed")
		}
	}

	// Telegram — send to Telegram chat configured in backup settings.
	var tgs models.TelegramBackupSettings
	if err := p.db.First(&tgs).Error; err == nil &&
		tgs.Enabled && tgs.BotToken != "" && tgs.ChatId != "" {
		tgMsg := fmt.Sprintf("%s <b>[%s] %s</b>\n%s", emoji, oltName, label, message)
		if tgErr := notify.SendTelegramMessage(tgs.BotToken, tgs.ChatId, tgMsg); tgErr != nil {
			log.Warn().Err(tgErr).Msg("poller: Telegram alert send failed")
		}
	}

	return waErr
}

func strPtr(s *string) string {
	if s == nil {
		return "N/A"
	}
	return *s
}
