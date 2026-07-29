package cron

// suspend_check.go — Port dari jobs.config.ts entry: suspend_check
//
// Runs hourly. Two actions:
// 1. Activate pending suspends (startDate <= now, status = APPROVED, user still active)
// 2. Restore users whose suspend endDate has passed

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// jobSuspendCheck processes pending suspend requests and restores expired ones.
func (s *Scheduler) jobSuspendCheck() {
	h := s.startHistory("suspend_check")
	defer func() { s.completeHistory(h, recover()) }()

	now := time.Now()
	suspended := 0
	restored := 0

	// 1. Activate pending suspends (startDate <= now, status = APPROVED, user still active)
	var toSuspend []models.SuspendRequest
	s.db.Where("status = ? AND startDate <= ?", "APPROVED", now).
		Preload("User").
		Find(&toSuspend)

	for _, sr := range toSuspend {
		if sr.User != nil && sr.User.Status == "active" {
			s.db.Model(&models.PppoeUser{}).Where("id = ?", sr.UserID).Update("status", "stopped")
			suspended++
		}
	}

	// 2. Restore users whose suspend endDate has passed
	var toRestore []models.SuspendRequest
	s.db.Where("status = ? AND endDate <= ?", "APPROVED", now).
		Preload("User").
		Find(&toRestore)

	for _, sr := range toRestore {
		if sr.User != nil && sr.User.Status == "stopped" {
			s.db.Model(&models.PppoeUser{}).Where("id = ?", sr.UserID).Update("status", "active")
			restored++
		}
		s.db.Model(&sr).Update("status", "COMPLETED")
	}

	msg := fmt.Sprintf("Suspend check: %d suspended, %d restored", suspended, restored)
	s.finishHistory(h, msg)
	log.Info().Int("suspended", suspended).Int("restored", restored).Msg("cron: suspend_check done")
}
