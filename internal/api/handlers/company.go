package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// CompanyHandler handles company settings endpoints.
type CompanyHandler struct {
	db *gorm.DB
}

func NewCompanyHandler(db *gorm.DB) *CompanyHandler { return &CompanyHandler{db: db} }

// companyResp embeds Company but overrides bankAccounts with parsed JSON.
type companyResp struct {
	models.Company
	BankAccounts json.RawMessage `json:"bankAccounts"`
}

func buildCompanyResp(company models.Company) companyResp {
	resp := companyResp{Company: company}
	if company.BankAccounts != nil && *company.BankAccounts != "" {
		resp.BankAccounts = json.RawMessage(*company.BankAccounts)
	} else {
		resp.BankAccounts = json.RawMessage("[]")
	}
	return resp
}

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
		defaultQrisEnabled := false
		return c.JSON(companyResp{
			Company: models.Company{
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
				QrisEnabled:             &defaultQrisEnabled,
			},
			BankAccounts: json.RawMessage("[]"),
		})
	}
	return c.JSON(buildCompanyResp(company))
}

func (h *CompanyHandler) UpdateCompany(c fiber.Ctx) error {
	// Parse body as a generic map to handle type mismatches.
	// The frontend sends bankAccounts as a JSON array, but the DB stores it as a JSON string.
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Convert bankAccounts from array → JSON string if needed
	if ba, ok := body["bankAccounts"]; ok {
		switch v := ba.(type) {
		case []interface{}:
			baJSON, _ := json.Marshal(v)
			body["bankAccounts"] = string(baJSON)
		case nil:
			body["bankAccounts"] = "[]"
		}
	}

	// Load existing company record (if any)
	var company models.Company
	h.db.First(&company)

	// Re-encode body with converted fields, then decode into company struct
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	if err := json.Unmarshal(bodyJSON, &company); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if company.ID == "" {
		company.ID = uuid.New().String()
		if err := h.db.Create(&company).Error; err != nil {
			log.Error().Err(err).Msg("company: failed to create")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save company: " + err.Error()})
		}
	} else {
		if err := h.db.Save(&company).Error; err != nil {
			log.Error().Err(err).Msg("company: failed to save")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save company: " + err.Error()})
		}
	}
	return c.JSON(buildCompanyResp(company))
}
