package cron

// invoice_status.go — Port dari src/server/jobs/invoice-status-updater.ts
//
// Runs hourly. Updates PENDING invoices to OVERDUE when due date has passed.

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// jobInvoiceStatusUpdate updates PENDING invoices to OVERDUE if past due date.
func (s *Scheduler) jobInvoiceStatusUpdate() {
	h := s.startHistory("invoice_status_update")
	defer func() { s.completeHistory(h, recover()) }()

	now := time.Now()
	result := s.db.Table("invoices").
		Where("status = ? AND dueDate < ?", "PENDING", now).
		Update("status", "OVERDUE")

	if result.Error != nil {
		s.failHistory(h, result.Error)
		log.Error().Err(result.Error).Msg("cron: invoice_status_update error")
		return
	}

	msg := fmt.Sprintf("Updated %d invoices from PENDING to OVERDUE", result.RowsAffected)
	s.finishHistory(h, msg)
	log.Info().Int64("count", result.RowsAffected).Msg("cron: invoice_status_update done")
}
