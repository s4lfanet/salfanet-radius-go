package cron

// payment_promise_cron.go — Check expired payment promises
//
// Runs daily at 08:00 WIB. Checks active promises where promiseDate < today
// and invoice still unpaid → mark as broken, re-isolir user.

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

func (s *Scheduler) jobPaymentPromiseCheck() {
	h := s.startHistory("payment_promise_check")
	defer func() { s.completeHistory(h, recover()) }()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	broken := 0

	// Find expired active promises
	var promises []models.PaymentPromise
	s.db.Where("status = ? AND promiseDate < ?", "active", today).Find(&promises)

	for _, promise := range promises {
		// Check if user has any paid invoice after promise date
		var paidCount int64
		s.db.Table("invoices").
			Where("userId = ? AND status = ? AND updatedAt >= ?", promise.UserID, "PAID", promise.PromiseDate).
			Count(&paidCount)

		if paidCount == 0 {
			// Promise broken — re-isolir
			s.db.Model(&promise).Update("status", "broken")

			// Isolir user
			s.db.Model(&models.PppoeUser{}).Where("id = ?", promise.UserID).Update("status", "isolated")
			s.db.Exec("INSERT INTO radcheck (username, attribute, op, value) VALUES (?, 'Auth-Type', ':=', 'Reject') ON DUPLICATE KEY UPDATE value = 'Reject'", promise.Username)

			broken++
			log.Info().Str("username", promise.Username).Msg("payment_promise: broken, user re-isolated")
		} else {
			// Promise fulfilled
			s.db.Model(&promise).Update("status", "fulfilled")
			log.Info().Str("username", promise.Username).Msg("payment_promise: fulfilled")
		}
	}

	msg := fmt.Sprintf("Payment promise check: %d broken, %d total checked", broken, len(promises))
	s.finishHistory(h, msg)
	log.Info().Int("broken", broken).Int("checked", len(promises)).Msg("cron: payment_promise_check done")
}
