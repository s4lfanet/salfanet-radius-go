package cron

// hotspot_sync.go — Port dari src/server/jobs/hotspot-sync.ts
//
// Runs every minute. Hotspot voucher sync with RADIUS:
// 1. Check WAITING vouchers for first login in radacct → set ACTIVE + calculate expiry
// 2. Check ACTIVE vouchers for expiry → set EXPIRED, disconnect session, cleanup RADIUS tables
// 3. Send agent notifications for activated/expired vouchers

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

var hotspotSyncMu sync.Mutex

// jobHotspotSync is the enhanced version of jobSyncVoucherExpiry.
func (s *Scheduler) jobHotspotSync() {
	if !hotspotSyncMu.TryLock() {
		log.Info().Msg("cron: hotspot_sync already running, skipping")
		return
	}
	defer hotspotSyncMu.Unlock()

	h := s.startHistory("hotspot_sync")
	defer func() { s.completeHistory(h, recover()) }()

	activatedCount := 0
	expiredCount := 0
	cleanedUpCount := 0

	// ── PART 1: WAITING → ACTIVE (check first login) ──────────────────────────
	var waitingVouchers []models.HotspotVoucher
	s.db.Where("status = 'WAITING'").Preload("Profile").Find(&waitingVouchers)

	for _, voucher := range waitingVouchers {
		// Check radacct for first login
		var session models.Radacct
		if err := s.db.Where("username = ? AND acctstarttime IS NOT NULL", voucher.Code).
			Order("acctstarttime asc").First(&session).Error; err != nil {
			continue // No session found
		}

		firstLoginAt := *session.AcctStartTime
		expiresAt := calculateVoucherExpiry(firstLoginAt, voucher.Profile.ValidityValue, voucher.Profile.ValidityUnit)

		// Update voucher to ACTIVE
		s.db.Model(&voucher).Updates(map[string]interface{}{
			"status":       "ACTIVE",
			"firstLoginAt": firstLoginAt,
			"expiresAt":    expiresAt,
		})

		// Send agent notification
		if voucher.AgentID != nil {
			s.createAgentNotification(*voucher.AgentID, "voucher_activated",
				"Voucher Digunakan",
				fmt.Sprintf("Voucher %s (%s) telah digunakan. Aktif hingga %s.",
					voucher.Code, voucher.Profile.Name, expiresAt.Format("02 Jan 2006 15:04")))
		}

		activatedCount++
		log.Debug().Str("code", voucher.Code).Time("expires", expiresAt).Msg("hotspot_sync: activated")
	}

	// ── PART 2: ACTIVE → EXPIRED ──────────────────────────────────────────────
	now := time.Now()
	var activeVouchers []models.HotspotVoucher
	s.db.Where("status = 'ACTIVE' AND expiresAt IS NOT NULL AND expiresAt <= ?", now).
		Preload("Profile").Find(&activeVouchers)

	// Also check vouchers with usageDuration limit (Method B)
	var durationVouchers []models.HotspotVoucher
	s.db.Joins("JOIN hotspot_profiles ON hotspot_profiles.id = hotspot_vouchers.profileId").
		Where("hotspot_vouchers.status = 'ACTIVE' AND hotspot_profiles.usageDuration IS NOT NULL").
		Preload("Profile").Find(&durationVouchers)

	// Track which voucher IDs are already expired by validity
	expiredIDs := make(map[string]bool)
	for _, v := range activeVouchers {
		expiredIDs[v.ID] = true
	}

	// Check duration-based expiry
	for _, voucher := range durationVouchers {
		if expiredIDs[voucher.ID] {
			continue
		}
		if voucher.Profile.UsageDuration == nil || *voucher.Profile.UsageDuration <= 0 {
			continue
		}
		maxDurationSec := int64(*voucher.Profile.UsageDuration * 3600)
		var totalUsage struct{ Total *int64 }
		s.db.Model(&models.Radacct{}).
			Where("username = ?", voucher.Code).
			Select("COALESCE(SUM(acctsessiontime), 0) as total").
			Scan(&totalUsage)
		if totalUsage.Total != nil && *totalUsage.Total >= maxDurationSec {
			activeVouchers = append(activeVouchers, voucher)
			expiredIDs[voucher.ID] = true
			log.Debug().Str("code", voucher.Code).Int64("used", *totalUsage.Total).Int64("max", maxDurationSec).
				Msg("hotspot_sync: expired by duration")
		}
	}

	for _, voucher := range activeVouchers {
		// Update status to EXPIRED
		s.db.Model(&voucher).Update("status", "EXPIRED")

		// Send agent notification
		if voucher.AgentID != nil {
			s.createAgentNotification(*voucher.AgentID, "voucher_expired",
				"Voucher Kadaluarsa",
				fmt.Sprintf("Voucher %s (%s) telah kadaluarsa.", voucher.Code, voucher.Profile.Name))
		}

		// Close active session in radacct
		s.db.Exec(`UPDATE radacct SET acctstoptime = NOW(), acctterminatecause = 'Session-Timeout'
			WHERE username = ? AND acctstoptime IS NULL`, voucher.Code)

		// Cleanup FreeRADIUS tables:
		// 1. Set password to EXPIRED in radcheck (prevents login)
		s.db.Model(&models.Radcheck{}).
			Where("username = ? AND attribute = 'Cleartext-Password'", voucher.Code).
			Update("value", "EXPIRED")

		// 2. Add/update Reply-Message in radreply
		s.db.Exec(`INSERT INTO radreply (username, attribute, op, value)
			VALUES (?, 'Reply-Message', '=', 'Kode Voucher Kadaluarsa')
			ON DUPLICATE KEY UPDATE value = VALUES(value)`, voucher.Code)

		// 3. Remove from radusergroup (removes bandwidth limits)
		s.db.Where("username = ?", voucher.Code).Delete(&models.Radusergroup{})

		expiredCount++
		log.Debug().Str("code", voucher.Code).Msg("hotspot_sync: expired")
	}

	// ── PART 3: Cleanup stale EXPIRED vouchers still in RADIUS ────────────────
	var expiredVoucherCodes []string
	s.db.Model(&models.HotspotVoucher{}).Where("status = 'EXPIRED'").Pluck("code", &expiredVoucherCodes)

	for _, code := range expiredVoucherCodes {
		// Check if still in radcheck
		var inRadcheck int64
		s.db.Model(&models.Radcheck{}).Where("username = ?", code).Count(&inRadcheck)
		if inRadcheck > 0 {
			// Get group name before deleting
			var ug models.Radusergroup
			s.db.Where("username = ?", code).First(&ug)

			s.db.Where("username = ?", code).Delete(&models.Radcheck{})
			s.db.Where("username = ?", code).Delete(&models.Radusergroup{})
			if ug.Groupname != "" {
				s.db.Exec("DELETE FROM radgroupreply WHERE groupname = ?", ug.Groupname)
			}
			cleanedUpCount++
			log.Debug().Str("code", code).Msg("hotspot_sync: cleaned stale")
		}

		// Close stale active sessions
		s.db.Exec(`UPDATE radacct SET acctstoptime = NOW(), acctterminatecause = 'Admin-Reset'
			WHERE username = ? AND acctstoptime IS NULL`, code)
	}

	msg := fmt.Sprintf("Activated: %d, Expired: %d, Cleaned: %d", activatedCount, expiredCount, cleanedUpCount)
	s.finishHistory(h, msg)
	log.Info().Int("activated", activatedCount).Int("expired", expiredCount).Int("cleaned", cleanedUpCount).
		Msg("cron: hotspot_sync done")
}

// calculateVoucherExpiry calculates the expiry time based on first login and validity.
func calculateVoucherExpiry(firstLogin time.Time, validityValue int, validityUnit string) time.Time {
	switch validityUnit {
	case "MINUTES":
		return firstLogin.Add(time.Duration(validityValue) * time.Minute)
	case "HOURS":
		return firstLogin.Add(time.Duration(validityValue) * time.Hour)
	case "DAYS":
		return firstLogin.AddDate(0, 0, validityValue)
	case "MONTHS":
		return firstLogin.AddDate(0, validityValue, 0)
	default:
		return firstLogin.AddDate(0, 0, validityValue)
	}
}

// createAgentNotification creates a notification for an agent (best effort).
func (s *Scheduler) createAgentNotification(agentID, notifType, title, message string) {
	notif := models.AgentNotification{
		ID:        uuid.NewString(),
		AgentID:   agentID,
		Type:      notifType,
		Title:     title,
		Message:   message,
		CreatedAt: time.Now(),
	}
	if err := s.db.Create(&notif).Error; err != nil {
		log.Error().Err(err).Str("agentId", agentID).Msg("hotspot_sync: create notification failed")
	}
}
