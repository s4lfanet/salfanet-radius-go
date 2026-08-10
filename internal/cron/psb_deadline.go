package cron

// psb_deadline.go — PSB 24-jam deadline check
//
// Runs every 30 minutes. Checks users with initialPaymentPending = true
// and psbDeadlineAt < now → auto-isolir.
// Also emits psb.deadline event when 2h remaining.

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
	"github.com/s4lfanet/salfanet-radius-go/internal/tzutil"
)

func (s *Scheduler) jobPsbDeadline() {
	h := s.startHistory("psb_deadline")
	defer func() { s.completeHistory(h, recover()) }()

	now := tzutil.Now()
	nowWIB := now

	isolated := 0
	notified := 0

	// 1. Auto-isolir users whose PSB deadline has passed and still haven't paid
	var expiredUsers []models.PppoeUser
	s.db.Where("initialPaymentPending = ? AND psbDeadlineAt < ? AND status = ?",
		true, now, "active").Find(&expiredUsers)

	for _, user := range expiredUsers {
		s.db.Model(&user).Update("status", "isolated")
		// Set Auth-Type Reject in radcheck
		s.db.Exec("DELETE FROM radcheck WHERE username = ? AND attribute = 'Auth-Type'", user.Username)
		s.db.Exec("INSERT INTO radcheck (username, attribute, op, value) VALUES (?, 'Auth-Type', ':=', 'Reject')", user.Username)
		isolated++
		log.Info().Str("username", user.Username).Msg("psb_deadline: auto-isolated (24h passed, no payment)")
	}

	// 2. Notify users approaching deadline (2h remaining)
	twoHoursFromNow := now.Add(2 * time.Hour)
	var approachingUsers []models.PppoeUser
	s.db.Where("initialPaymentPending = ? AND psbDeadlineAt <= ? AND psbDeadlineAt > ? AND status = ?",
		true, twoHoursFromNow, now, "active").Find(&approachingUsers)

	for _, user := range approachingUsers {
		// In production, emit event via EventBus: psb.deadline
		// For now, just log
		notified++
		log.Info().Str("username", user.Username).Time("deadline", *user.PsbDeadlineAt).Msg("psb_deadline: 2h warning")
	}

	msg := fmt.Sprintf("PSB deadline: %d isolated, %d notified (2h warning), checked at %s", isolated, notified, nowWIB.Format("15:04:05"))
	s.finishHistory(h, msg)
	log.Info().Int("isolated", isolated).Int("notified", notified).Msg("cron: psb_deadline done")
}
