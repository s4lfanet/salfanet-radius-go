package handlers

import (
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// CompanyHandler handles company settings endpoints.
type CompanyHandler struct {
	db *gorm.DB
}

func NewCompanyHandler(db *gorm.DB) *CompanyHandler { return &CompanyHandler{db: db} }

func (h *CompanyHandler) GetCompany(c fiber.Ctx) error {
	var company models.Company
	if err := h.db.First(&company).Error; err != nil {
		// Fresh install: no company record yet — return safe defaults instead of 404
		defaultBaseURL := "http://localhost:3000"
		defaultTimezone := "Asia/Jakarta"
		defaultPoweredBy := "SALFANET RADIUS"
		defaultInvoiceDays := 7
		defaultGracePeriod := 0
		defaultIsolation := true
		defaultAllowDns := true
		defaultAllowPayment := true
		defaultNotifyWA := false
		defaultNotifyEmail := false
		defaultReferral := false
		defaultReferralAmount := 10000
		return c.JSON(models.Company{
			Name:                    "SALFANET RADIUS",
			BaseURL:                 &defaultBaseURL,
			Timezone:                &defaultTimezone,
			PoweredBy:               &defaultPoweredBy,
			InvoiceGenerateDays:     &defaultInvoiceDays,
			GracePeriodDays:         &defaultGracePeriod,
			IsolationEnabled:        &defaultIsolation,
			IsolationAllowDns:       &defaultAllowDns,
			IsolationAllowPayment:   &defaultAllowPayment,
			IsolationNotifyWhatsapp: &defaultNotifyWA,
			IsolationNotifyEmail:    &defaultNotifyEmail,
			ReferralEnabled:         &defaultReferral,
			ReferralRewardAmount:    &defaultReferralAmount,
		})
	}
	return c.JSON(company)
}

func (h *CompanyHandler) UpdateCompany(c fiber.Ctx) error {
	var company models.Company
	h.db.First(&company)

	if err := c.Bind().JSON(&company); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if company.ID == "" {
		h.db.Create(&company)
	} else {
		h.db.Save(&company)
	}
	return c.JSON(company)
}
