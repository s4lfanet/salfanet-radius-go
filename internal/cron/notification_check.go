package cron

// notification_check.go — Port dari dispatcher.service.ts runNotificationCheck()
//
// Runs every 6 hours. Creates in-app notifications for:
// 1. Overdue invoices (PENDING + past dueDate, deduped within 24h)
// 2. Users expiring today (active + expiredAt is today, deduped per day)
// 3. Pending registration requests (deduped by phone number)

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

func (s *Scheduler) jobNotificationCheck() {
	h := s.startHistory("notification_check")
	defer func() { s.completeHistory(h, recover()) }()

	overdue := s.checkOverdueInvoices()
	expired := s.checkExpiredUsers()
	pending := s.checkPendingRegistrations()
	total := overdue + expired + pending

	s.finishHistory(h, fmt.Sprintf("Overdue: %d, Expired: %d, Pending: %d, Total: %d",
		overdue, expired, pending, total))
	log.Info().Int("overdue", overdue).Int("expired", expired).Int("pending", pending).
		Msg("cron: notification_check done")
}

func (s *Scheduler) checkOverdueInvoices() int {
	now := time.Now()
	twentyFourHoursAgo := now.Add(-24 * time.Hour)

	var invoices []models.Invoice
	s.db.Where("status = 'PENDING' AND dueDate < ?", now).Find(&invoices)

	count := 0
	for _, inv := range invoices {
		link := fmt.Sprintf("/admin/invoices?id=%s", inv.ID)
		// Check if notification already exists in last 24h
		var existing int64
		s.db.Model(&models.Notification{}).
			Where("type = ? AND link = ? AND createdAt >= ?", "invoice_overdue", link, twentyFourHoursAgo).
			Count(&existing)
		if existing > 0 {
			continue
		}

		customerName := ""
		if inv.CustomerName != nil {
			customerName = *inv.CustomerName
		} else if inv.CustomerUsername != nil {
			customerName = *inv.CustomerUsername
		}
		s.db.Create(&models.Notification{
			ID:        uuid.NewString(),
			Type:      "invoice_overdue",
			Title:     "Invoice Overdue",
			Message:   fmt.Sprintf("Invoice %s for %s is overdue", inv.InvoiceNumber, customerName),
			Link:      &link,
			CreatedAt: now,
		})
		count++
	}
	return count
}

func (s *Scheduler) checkExpiredUsers() int {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)

	var users []models.PppoeUser
	s.db.Where("expiredAt >= ? AND expiredAt < ? AND status = 'active'", today, tomorrow).Find(&users)

	count := 0
	for _, user := range users {
		link := fmt.Sprintf("/admin/pppoe/users?id=%s", user.ID)
		// Check if notification already exists today
		var existing int64
		s.db.Model(&models.Notification{}).
			Where("type = ? AND link = ? AND createdAt >= ?", "user_expired", link, today).
			Count(&existing)
		if existing > 0 {
			continue
		}

		s.db.Create(&models.Notification{
			ID:        uuid.NewString(),
			Type:      "user_expired",
			Title:     "User Expiring Today",
			Message:   fmt.Sprintf("User %s (%s) is expiring today", user.Username, user.Name),
			Link:      &link,
			CreatedAt: now,
		})
		count++
	}
	return count
}

func (s *Scheduler) checkPendingRegistrations() int {
	var registrations []models.RegistrationRequest
	s.db.Where("status = 'PENDING'").Find(&registrations)

	count := 0
	for _, reg := range registrations {
		// Check if notification already exists for this phone
		var existing int64
		s.db.Model(&models.Notification{}).
			Where("type = ? AND message LIKE ?", "new_registration", "%"+reg.Phone+"%").
			Count(&existing)
		if existing > 0 {
			continue
		}

		link := "/admin/pppoe/registrations"
		s.db.Create(&models.Notification{
			ID:        uuid.NewString(),
			Type:      "new_registration",
			Title:     "New Registration Request",
			Message:   fmt.Sprintf("%s (%s) requested service registration", reg.Name, reg.Phone),
			Link:      &link,
			CreatedAt: time.Now(),
		})
		count++
	}
	return count
}
