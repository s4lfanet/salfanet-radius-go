package cron

// cleanup_jobs.go — Port dari jobs.config.ts entries:
//   - activity_log_cleanup (daily 2 AM)
//   - webhook_log_cleanup (daily 3 AM)

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// jobActivityLogCleanup deletes activity logs older than 30 days.
func (s *Scheduler) jobActivityLogCleanup() {
	h := s.startHistory("activity_log_cleanup")
	defer func() { s.completeHistory(h, recover()) }()

	cutoff := time.Now().AddDate(0, 0, -30)
	result := s.db.Table("activity_logs").Where("createdAt < ?", cutoff).Delete(nil)
	if result.Error != nil {
		s.failHistory(h, result.Error)
		log.Error().Err(result.Error).Msg("cron: activity_log_cleanup error")
		return
	}

	msg := fmt.Sprintf("Deleted %d activity logs older than 30 days", result.RowsAffected)
	s.finishHistory(h, msg)
	log.Info().Int64("count", result.RowsAffected).Msg("cron: activity_log_cleanup done")
}

// jobWebhookLogCleanup deletes webhook logs older than 30 days.
func (s *Scheduler) jobWebhookLogCleanup() {
	h := s.startHistory("webhook_log_cleanup")
	defer func() { s.completeHistory(h, recover()) }()

	cutoff := time.Now().AddDate(0, 0, -30)
	result := s.db.Table("webhook_logs").Where("createdAt < ?", cutoff).Delete(nil)
	if result.Error != nil {
		s.failHistory(h, result.Error)
		log.Error().Err(result.Error).Msg("cron: webhook_log_cleanup error")
		return
	}

	msg := fmt.Sprintf("Deleted %d webhook logs older than 30 days", result.RowsAffected)
	s.finishHistory(h, msg)
	log.Info().Int64("count", result.RowsAffected).Msg("cron: webhook_log_cleanup done")
}
