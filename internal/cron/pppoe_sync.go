package cron

// pppoe_sync.go — Port dari src/server/jobs/pppoe-sync.ts
//
// Runs hourly. Enhanced auto-isolation:
// 1. Enforce blocked/stop users have Auth-Type=Reject in radcheck
// 2. Disconnect active sessions for blocked/stop users
// 3. Find expired active users (with grace period from company settings)
// 4. Isolate: update status, keep password, move to 'isolir' group, remove static IP
// 5. Close radacct sessions
//
// IMPORTANT: isolated users CAN LOGIN (restricted access via isolir group)
//            blocked/stop users CANNOT LOGIN (Auth-Type=Reject)

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/notify"
)

var pppoeSyncMu sync.Mutex

// jobPPPoEAutoIsolir is the enhanced version of jobAutoIsolate.
// It replaces the simpler jobAutoIsolate with full pppoe-sync.ts logic.
func (s *Scheduler) jobPPPoEAutoIsolir() {
	if !pppoeSyncMu.TryLock() {
		log.Info().Msg("cron: pppoe_auto_isolir already running, skipping")
		return
	}
	defer pppoeSyncMu.Unlock()

	h := s.startHistory("pppoe_auto_isolir")
	defer func() { s.completeHistory(h, recover()) }()

	// Get grace period from company settings
	var company models.Company
	_ = s.db.First(&company)

	grace := 0
	if company.GracePeriodDays != nil {
		grace = *company.GracePeriodDays
	}

	// 1. Enforce blocked/stop users have Auth-Type=Reject
	s.db.Exec(`DELETE rc FROM radcheck rc
		INNER JOIN pppoe_users pu ON pu.username = rc.username
		WHERE pu.status IN ('blocked','stop') AND rc.attribute = 'Auth-Type'`)
	s.db.Exec(`INSERT INTO radcheck (username, attribute, op, value)
		SELECT pu.username, 'Auth-Type', ':=', 'Reject'
		FROM pppoe_users pu WHERE pu.status IN ('blocked','stop')`)

	// Remove old reply messages and set blocked message
	s.db.Exec(`DELETE rr FROM radreply rr
		INNER JOIN pppoe_users pu ON pu.username = rr.username
		WHERE pu.status IN ('blocked','stop') AND rr.attribute = 'Reply-Message'`)
	s.db.Exec(`INSERT INTO radreply (username, attribute, op, value)
		SELECT pu.username, 'Reply-Message', ':=', 'Akun Diblokir - Hubungi Admin'
		FROM pppoe_users pu WHERE pu.status IN ('blocked','stop')`)

	// 2. Close active sessions for blocked/stop users
	s.db.Exec(`UPDATE radacct ra
		INNER JOIN pppoe_users pu ON pu.username = ra.username
		SET ra.acctstoptime = NOW(), ra.acctterminatecause = 'Admin-Reset'
		WHERE pu.status IN ('blocked','stop') AND ra.acctstoptime IS NULL`)

	// 3. Find expired active users
	cutoff := time.Now().AddDate(0, 0, -grace)
	var users []models.PppoeUser
	s.db.Where(`status = 'active' AND expiredAt < ? AND autoIsolationEnabled = true`, cutoff).
		Find(&users)

	if len(users) == 0 {
		s.finishHistory(h, "No expired users found")
		return
	}

	isolatedCount := 0
	for _, user := range users {
		if err := s.isolateUser(&user); err != nil {
			log.Error().Err(err).Str("username", user.Username).Msg("cron: pppoe_auto_isolir isolate failed")
			continue
		}
		isolatedCount++
	}

	msg := fmt.Sprintf("Isolated %d/%d users", isolatedCount, len(users))
	s.finishHistory(h, msg)
	log.Info().Int("isolated", isolatedCount).Int("total", len(users)).Msg("cron: pppoe_auto_isolir done")
}

// isolateUser performs the full isolation sequence for a single user.
func (s *Scheduler) isolateUser(user *models.PppoeUser) error {
	// 1. Update status to isolated
	if err := s.db.Model(user).Update("status", "isolated").Error; err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	// 2. Keep password in radcheck (allow login for isolation)
	s.db.Exec(`INSERT INTO radcheck (username, attribute, op, value)
		VALUES (?, 'Cleartext-Password', ':=', ?)
		ON DUPLICATE KEY UPDATE value = VALUES(value)`,
		user.Username, user.Password)

	// 2b. Remove Auth-Type Reject (allow login for isolation)
	s.db.Exec(`DELETE FROM radcheck WHERE username = ? AND attribute = 'Auth-Type'`, user.Username)

	// Remove reject message
	s.db.Exec(`DELETE FROM radreply WHERE username = ? AND attribute = 'Reply-Message'`, user.Username)

	// 3. Move to isolir group
	s.db.Exec(`DELETE FROM radusergroup WHERE username = ?`, user.Username)
	s.db.Exec(`INSERT INTO radusergroup (username, groupname, priority) VALUES (?, 'isolir', 1)`, user.Username)

	// 4. Remove static IP (user will get IP from pool-isolir)
	s.db.Exec(`DELETE FROM radreply WHERE username = ? AND attribute = 'Framed-IP-Address'`, user.Username)

	// 5. Close session in radacct
	s.db.Exec(`UPDATE radacct SET acctstoptime = NOW(), acctterminatecause = 'Admin-Reset'
		WHERE username = ? AND acctstoptime IS NULL`, user.Username)

	// 6. Send WhatsApp notification (best effort)
	_ = notify.SendIsolationNotice(user.Phone, user.Name)

	return nil
}
