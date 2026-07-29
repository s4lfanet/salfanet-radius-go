package cron

// auto_renewal.go — Port dari src/server/jobs/auto-renewal.ts
//
// Runs daily at 8 AM WIB. Auto-renews PREPAID users from balance if:
// 1. autoRenewal = true
// 2. expiredAt within next 3 days
// 3. balance >= package price (with PPN if enabled)
//
// Creates RENEWAL invoice, deducts balance, extends expiry, restores RADIUS if isolated.

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

func (s *Scheduler) jobAutoRenewal() {
	h := s.startHistory("auto_renewal")
	defer func() { s.completeHistory(h, recover()) }()

	threeDaysFromNow := time.Now().AddDate(0, 0, 3)

	var users []models.PppoeUser
	s.db.Where(`status IN ('active','isolated') AND subscriptionType = 'PREPAID'
		AND autoRenewal = true
		AND expiredAt IS NOT NULL AND expiredAt >= ? AND expiredAt <= ?`,
		time.Now(), threeDaysFromNow).
		Preload("Profile").
		Find(&users)

	successCount := 0
	failedCount := 0

	for _, user := range users {
		if err := s.processAutoRenewalUser(&user); err != nil {
			log.Error().Err(err).Str("username", user.Username).Msg("cron: auto-renewal failed")
			failedCount++
		} else {
			successCount++
		}
	}

	msg := fmt.Sprintf("Processed %d users, paid %d, failed %d", len(users), successCount, failedCount)
	s.finishHistory(h, msg)
	log.Info().Int("success", successCount).Int("failed", failedCount).Msg("cron: auto_renewal done")
}

func (s *Scheduler) processAutoRenewalUser(user *models.PppoeUser) error {
	profile := user.Profile
	if profile.ID == "" {
		return fmt.Errorf("no profile")
	}

	// Calculate price with PPN if enabled
	packagePrice := profile.Price
	if profile.PPNActive && profile.PPNRate > 0 {
		packagePrice = packagePrice + (packagePrice * profile.PPNRate / 100)
	}

	if user.Balance < packagePrice {
		return fmt.Errorf("insufficient balance (%d < %d)", user.Balance, packagePrice)
	}

	// Check for existing pending/overdue renewal invoice
	var existingInv models.Invoice
	hasExisting := s.db.Where("userId = ? AND invoiceType = ? AND status IN ?",
		user.ID, models.InvoiceRenewal, []string{"PENDING", "OVERDUE"}).First(&existingInv).Error == nil

	var invoice models.Invoice
	if hasExisting {
		invoice = existingInv
	} else {
		// Create renewal invoice
		invoice = models.Invoice{
			ID:               uuid.NewString(),
			InvoiceNumber:    fmt.Sprintf("INV-%s-%s", time.Now().Format("200601"), uuid.NewString()[:8]),
			UserID:           &user.ID,
			Amount:           packagePrice,
			BaseAmount:       &profile.Price,
			DueDate:          *user.ExpiredAt,
			Status:           models.InvoicePending,
			InvoiceType:      models.InvoiceRenewal,
			CustomerName:     &user.Name,
			CustomerPhone:    &user.Phone,
			CustomerUsername: &user.Username,
		}
		if err := s.db.Create(&invoice).Error; err != nil {
			return fmt.Errorf("create invoice: %w", err)
		}
	}

	// Pay from balance in a transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Deduct balance
		if err := tx.Model(&models.PppoeUser{}).Where("id = ?", user.ID).
			Update("balance", gorm.Expr("balance - ?", packagePrice)).Error; err != nil {
			return fmt.Errorf("deduct balance: %w", err)
		}

		// 2. Mark invoice as paid
		now := time.Now()
		if err := tx.Model(&invoice).Updates(map[string]interface{}{
			"status": "PAID",
			"paidAt": now,
		}).Error; err != nil {
			return fmt.Errorf("mark paid: %w", err)
		}

		// 3. Extend expiredAt
		newExpiredAt := *user.ExpiredAt
		validity := profile.ValidityValue
		switch profile.ValidityUnit {
		case "MONTHS":
			newExpiredAt = newExpiredAt.AddDate(0, validity, 0)
		case "DAYS":
			newExpiredAt = newExpiredAt.AddDate(0, 0, validity)
		}

		if err := tx.Model(&models.PppoeUser{}).Where("id = ?", user.ID).
			Updates(map[string]interface{}{
				"expiredAt": newExpiredAt,
				"status":    "active",
			}).Error; err != nil {
			return fmt.Errorf("extend expiry: %w", err)
		}

		// 4. Create transaction record
		var category models.TransactionCategory
		if err := tx.Where("name = ?", "Pembayaran Langganan").First(&category).Error; err != nil {
			category = models.TransactionCategory{
				ID:   uuid.NewString(),
				Name: "Pembayaran Langganan",
				Type: "INCOME",
			}
			tx.Create(&category)
		}

		desc := fmt.Sprintf("Auto-payment dari saldo untuk invoice %s", invoice.InvoiceNumber)
		notes := fmt.Sprintf("User: %s, Balance payment", user.Username)
		createdBy := "system"
		txn := models.Transaction{
			ID:          uuid.NewString(),
			Date:        now,
			Type:        "INCOME",
			CategoryID:  category.ID,
			Description: desc,
			Amount:      packagePrice,
			Reference:   &invoice.InvoiceNumber,
			Notes:       &notes,
			CreatedBy:   &createdBy,
		}
		if err := tx.Create(&txn).Error; err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}

		return nil
	})

	// Restore RADIUS if user was isolated (outside transaction — best effort)
	if err == nil {
		s.restoreRADIUS(user)
	}
	return err
}

// restoreRADIUS restores user in RADIUS if they were isolated
func (s *Scheduler) restoreRADIUS(user *models.PppoeUser) {
	if user.Status != "isolated" {
		return
	}
	profile := user.Profile
	groupName := profile.GroupName
	if err := s.radius.RestoreUser(user.Username, groupName, user.IPAddress); err != nil {
		log.Error().Err(err).Str("username", user.Username).Msg("cron: auto-renewal RADIUS restore failed")
	}
}
