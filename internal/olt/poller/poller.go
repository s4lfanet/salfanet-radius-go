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
	snmputil "github.com/s4lfanet/salfanet-radius-go/internal/olt/snmp"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/telnet"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/vendors/zte"
)

// BroadcastFn is a function that broadcasts ONU status updates to WebSocket clients.
type BroadcastFn func(oltID string, data interface{})

// Poller manages polling goroutines for all OLTs.
type Poller struct {
	db        *gorm.DB
	broadcast BroadcastFn

	mu      sync.Mutex
	workers map[string]context.CancelFunc // oltID → cancel
	pools   map[string]*telnet.Pool       // oltID → telnet pool
}

// New creates a new Poller.
func New(db *gorm.DB, broadcast BroadcastFn) *Poller {
	return &Poller{
		db:        db,
		broadcast: broadcast,
		workers:   make(map[string]context.CancelFunc),
		pools:     make(map[string]*telnet.Pool),
	}
}

// Start begins polling for the given OLT. It stops any existing poll loop first.
func (p *Poller) Start(olt *models.NetworkOLT) {
	p.Stop(olt.ID) // Stop existing loop if any

	ctx, cancel := context.WithCancel(context.Background())
	p.mu.Lock()
	p.workers[olt.ID] = cancel
	p.mu.Unlock()

	// Build Telnet pool if Telnet is enabled
	var pool *telnet.Pool
	if olt.TelnetEnabled && olt.Username != nil && olt.Password != nil {
		cfg := telnet.DefaultConfig(olt.IPAddress, olt.TelnetPort, *olt.Username, *olt.Password)
		pool = telnet.NewPool(cfg)
		p.mu.Lock()
		p.pools[olt.ID] = pool
		p.mu.Unlock()
	}

	interval := time.Duration(olt.PollingInterval) * time.Second
	if interval < 30*time.Second {
		interval = 60 * time.Second
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

// TriggerPoll triggers an immediate poll for the given OLT (used by manual sync endpoint).
func (p *Poller) TriggerPoll(oltID string) error {
	var olt models.NetworkOLT
	if err := p.db.First(&olt, "id = ?", oltID).Error; err != nil {
		return fmt.Errorf("OLT not found: %w", err)
	}

	p.mu.Lock()
	pool := p.pools[oltID]
	p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var _ *telnet.Pool = pool
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

	// Enrich ONU distances via Telnet (more accurate than SNMP equalization delay).
	// Runs "show gpon onu detail-info" per ONU and parses "ONU Distance: Xm".
	p.mu.Lock()
	pool := p.pools[olt.ID]
	p.mu.Unlock()
	if pool != nil && len(onus) > 0 {
		if telnetDist := zte.FetchTelnetDistances(pool, onus); len(telnetDist) > 0 {
			for _, onu := range onus {
				key := fmt.Sprintf("%d/%d/%d:%d", onu.Frame, onu.Slot, onu.Port, onu.OnuID)
				if d, ok := telnetDist[key]; ok {
					onu.Distance = &d
				}
			}
			log.Debug().Str("olt", olt.ID).Int("distances", len(telnetDist)).Msg("poller: telnet distances enriched")
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
			base.Distance = onu.Distance
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

	// Upsert registered ONUs with full column set
	if len(registeredStatuses) > 0 {
		if e := p.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "oltId"}, {Name: "frame"}, {Name: "slot"}, {Name: "port"}, {Name: "onuId"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"serialNumber", "description", "status", "rxPower",
				"distance", "lastSeenAt", "updatedAt",
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

	allStatuses := append(registeredStatuses, unregisteredStatuses...)
	totalONU := len(allStatuses)

	// Update OLT summary
	pollTime := now
	p.db.Model(olt).Updates(map[string]interface{}{
		"lastPollAt": pollTime,
		"totalOnu":   totalONU,
		"onlineOnu":  onlineCount,
		"offlineOnu": offlineCount,
		"isOnline":   true,
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

// checkAlerts creates alert records for ONUs that went offline.
func (p *Poller) checkAlerts(ctx context.Context, olt *models.NetworkOLT, statuses []models.OLTONUStatus) {
	for _, s := range statuses {
		if s.Status == models.OnuOffline {
			// Check if an unresolved alert already exists for this ONU
			var existing models.OLTAlert
			err := p.db.WithContext(ctx).Where(
				"oltId = ? AND onuId = ? AND alertType = ? AND isResolved = ?",
				olt.ID, s.ID, models.AlertONUOffline, false,
			).First(&existing).Error
			if err == nil {
				continue // Alert already open
			}

			onuID := s.ID
			alert := models.OLTAlert{
				ID:        uuid.NewString(),
				OltID:     &olt.ID,
				OnuID:     &onuID,
				AlertType: models.AlertONUOffline,
				Severity:  models.SeverityWarning,
				Message:   fmt.Sprintf("ONU %s (port %d/%d/%d:%d) went offline", strPtr(s.SerialNumber), s.Frame, s.Slot, s.Port, s.OnuID),
			}
			p.db.WithContext(ctx).Create(&alert)
		}
	}
}

func strPtr(s *string) string {
	if s == nil {
		return "N/A"
	}
	return *s
}
