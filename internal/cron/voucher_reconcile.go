package cron

// voucher_reconcile.go — Port dari voucher-sync.ts reconcileVoucherTransactions()
//
// Runs daily. Creates financial transactions for used vouchers that don't have
// a corresponding Transaction record. Also creates agent commission expenses.

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

func (s *Scheduler) jobReconcileVoucherTransactions() {
	h := s.startHistory("voucher_reconcile")
	defer func() { s.completeHistory(h, recover()) }()

	// Find used vouchers without order (manually sold)
	var vouchers []models.HotspotVoucher
	s.db.Preload("Profile").Preload("Agent").
		Where("status IN ('ACTIVE','EXPIRED') AND firstLoginAt IS NOT NULL AND orderId IS NULL").
		Find(&vouchers)

	if len(vouchers) == 0 {
		s.finishHistory(h, "No vouchers to reconcile")
		return
	}

	// Get hotspot income category
	var hotspotCat models.TransactionCategory
	if err := s.db.Where("name = ? AND type = ?", "Pembayaran Hotspot", "INCOME").First(&hotspotCat).Error; err != nil {
		s.finishHistory(h, "Hotspot income category not found")
		return
	}

	// Get agent commission category (best effort)
	var agentCat models.TransactionCategory
	s.db.Where("name = ? AND type = ?", "Komisi Agent", "EXPENSE").First(&agentCat)

	count := 0
	for _, voucher := range vouchers {
		// Check if income transaction already exists
		ref := fmt.Sprintf("VOUCHER-%s", voucher.Code)
		var existing int64
		s.db.Model(&models.Transaction{}).Where("reference = ?", ref).Count(&existing)
		if existing > 0 {
			continue
		}

		if voucher.FirstLoginAt == nil {
			continue
		}

		incomeAmount := voucher.Profile.SellingPrice
		isAgent := voucher.AgentID != nil

		desc := fmt.Sprintf("Voucher %s - %s", voucher.Profile.Name, voucher.Code)
		if isAgent {
			desc += " (Agent)"
		}
		notes := fmt.Sprintf("[Rekonsiliasi] Pendapatan voucher hotspot (Harga Jual: Rp %d, Harga Modal: Rp %d)",
			incomeAmount, voucher.Profile.CostPrice)

		s.db.Create(&models.Transaction{
			ID:          uuid.NewString(),
			Date:        *voucher.FirstLoginAt,
			Type:        "INCOME",
			CategoryID:  hotspotCat.ID,
			Description: desc,
			Amount:      incomeAmount,
			Reference:   &ref,
			Notes:       &notes,
			CreatedAt:   time.Now(),
		})
		count++

		// Create agent commission expense if applicable
		if isAgent && voucher.Profile.ResellerFee > 0 && agentCat.ID != "" {
			commRef := fmt.Sprintf("COMMISSION-%s", voucher.Code)
			var existingComm int64
			s.db.Model(&models.Transaction{}).Where("reference = ?", commRef).Count(&existingComm)
			if existingComm == 0 {
				agentName := "Unknown"
				if voucher.Agent != nil {
					agentName = voucher.Agent.Name
				}
				commDesc := fmt.Sprintf("Komisi Agent %s - Voucher %s", agentName, voucher.Code)
				commNotes := fmt.Sprintf("[Rekonsiliasi] Komisi agent untuk voucher %s", voucher.Profile.Name)
				s.db.Create(&models.Transaction{
					ID:          uuid.NewString(),
					Date:        *voucher.FirstLoginAt,
					Type:        "EXPENSE",
					CategoryID:  agentCat.ID,
					Description: commDesc,
					Amount:      voucher.Profile.ResellerFee,
					Reference:   &commRef,
					Notes:       &commNotes,
					CreatedAt:   time.Now(),
				})
			}
		}
	}

	s.finishHistory(h, fmt.Sprintf("Reconciled %d voucher transactions", count))
	log.Info().Int("count", count).Msg("cron: voucher_reconcile done")
}
