// Package api wires up the Fiber router with all handlers and middleware.
package api

import (
	"time"

	fws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/valyala/fasthttp"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/api/handlers"
	"github.com/s4lfanet/salfanet-radius-go/internal/api/middleware"
	"github.com/s4lfanet/salfanet-radius-go/internal/config"
	"github.com/s4lfanet/salfanet-radius-go/internal/cron"
	"github.com/s4lfanet/salfanet-radius-go/internal/olt/poller"
	"github.com/s4lfanet/salfanet-radius-go/internal/radius"
	"github.com/s4lfanet/salfanet-radius-go/internal/ws"
)

var wsUpgrader = fws.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
}

// New builds and returns the configured Fiber app.
func New(db *gorm.DB, p *poller.Poller, hub *ws.Hub, rad *radius.Service, sched *cron.Scheduler) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Salfanet RADIUS API",
		IdleTimeout:  150 * time.Second, // must be > nginx keepalive_timeout (120s) to avoid race with Cloudflare
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: config.C.CORSOrigins,
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
	}))

	// ─── Handlers ────────────────────────────────────────────────────────────
	authH := handlers.NewAuthHandler(db)
	adminH := handlers.NewAdminHandler(db)
	oltH := handlers.NewOLTHandler(db, p, hub)
	pppoeH := handlers.NewPPPoEHandler(db, rad)
	billingH := handlers.NewBillingHandler(db)
	radiusH := handlers.NewRadiusHandler(db, rad)
	hotspotH := handlers.NewHotspotHandler(db)
	agentH := handlers.NewAgentHandler(db)
	networkH := handlers.NewNetworkHandler(db)
	waH := handlers.NewWhatsappHandler(db)
	ticketH := handlers.NewTicketHandler(db)
	companyH := handlers.NewCompanyHandler(db)
	cronH := handlers.NewCronHandler(db, sched)
	customerH := handlers.NewCustomerPortalHandler(db)
	sessionsH := handlers.NewSessionsHandler(db)
	settingsH := handlers.NewSettingsHandler(db)
	permsH := handlers.NewPermissionsHandler(db)
	inventoryH := handlers.NewInventoryHandler(db)
	keuanganH := handlers.NewKeuanganHandler(db)
	manualPayH := handlers.NewManualPaymentHandler(db)
	jobH := handlers.NewJobHandler(db)
	empAdminH := handlers.NewEmployeeAdminHandler(db)
	genieacsH := handlers.NewGenieacsHandler(db)

	// New handlers (instantiated early so public routes can be registered before auth group)
	pubH := handlers.NewPublicHandler(db)
	notifH := handlers.NewNotificationHandler(db)
	frH := handlers.NewFreeradiusHandler(db)
	invExtH := handlers.NewInvoiceExtHandler(db)
	refH := handlers.NewReferralHandler(db)
	adminUserH := handlers.NewAdminUserHandler(db)
	techAdminH := handlers.NewTechnicianAdminHandler(db)
	actH := handlers.NewActivityLogHandler(db)
	hotspotExtH := handlers.NewHotspotExtHandler(db)
	voucherTplH := handlers.NewVoucherTemplateHandler(db)
	ticketExtH := handlers.NewTicketExtHandler(db)
	analyticsH := handlers.NewAnalyticsHandler(db)
	settingsExtH := handlers.NewSettingsExtHandler(db)
	backupH := handlers.NewBackupHandler(db)
	telegramH := handlers.NewTelegramHandler(db)
	pushH := handlers.NewPushHandler(db)
	oltExtH := handlers.NewOltExtHandler(db)
	pppoeExtH := handlers.NewPppoeExtHandler(db)
	techPortalH := handlers.NewTechnicianPortalHandler(db)
	uploadH := handlers.NewUploadHandler(db)
	waExtH := handlers.NewWhatsappExtHandler(db)
	pushExtH := handlers.NewPushExtHandler(db)
	settingsGnH := handlers.NewSettingsGenieacsHandler(db)

	// ─── Batch 7 handlers ────────────────────────────────────────────────────
	custExtH := handlers.NewCustomerExtHandler(db)
	waCrudH := handlers.NewWhatsappCrudHandler(db)
	adminJobsH := handlers.NewAdminJobsHandler(db)
	miscH := handlers.NewMiscHandler(db, p)

	// ─── Batch 8 handlers ────────────────────────────────────────────────────
	custExt2H := handlers.NewCustomerPortalExt2Handler(db)
	paymentH := handlers.NewPaymentHandler(db)

	// ─── Batch 9 handlers ────────────────────────────────────────────────────
	paymentsApprovalH := handlers.NewPaymentsApprovalHandler(db)
	invoiceTplH := handlers.NewInvoiceTemplateHandler(db)
	payrollTplH := handlers.NewPayrollTemplateHandler(db)
	troubleshootH := handlers.NewTroubleshootingHandler(db)
	evoucherH := handlers.NewEvoucherHandler(db)

	// ─── Batch 10 handlers ───────────────────────────────────────────────────
	adminVPNH := handlers.NewAdminVPNHandler(db)
	adminPayrollH := handlers.NewAdminPayrollHandler(db)
	adminHRH := handlers.NewAdminHRHandler(db)
	fcmH := handlers.NewFCMHandler(db)

	// ─── Batch 11 handlers ───────────────────────────────────────────────────
	adminMiscH := handlers.NewAdminMiscHandler(db)
	networkVPNH := handlers.NewNetworkVPNHandler(db)
	// NOTE: networkInfraH is instantiated near batch 12 routes (after olt group)

	// ─── Public routes (NO auth — must be before the api group) ──────────────
	app.Get("/api/system/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/api/system/version", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"version": "2.0.0-go", "engine": "Go"})
	})
	app.Get("/api/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "engine": "go"})
	})

	// Public API (registration portal, etc.)
	app.Get("/api/public/company", pubH.GetCompany)
	app.Get("/api/public/areas", pubH.GetAreas)
	app.Get("/api/public/profiles", pubH.GetProfiles)
	app.Get("/api/public/stats", pubH.GetStats)
	app.Get("/api/public/payment-gateways", pubH.GetPaymentGateways)
	app.Post("/api/public/upload-registration", pubH.UploadRegistration)

	// Admin pre-login (public — must be before the api group to avoid auth middleware)
	app.Post("/api/admin/auth/pre-login", adminMiscH.PreLogin)

	// Technician auth (public — uses its own JWT, not admin JWT)
	techAuth := app.Group("/api/technician/auth")
	techAuth.Post("/request-otp", techPortalH.RequestOTP)
	techAuth.Post("/verify-otp", techPortalH.VerifyOTP)
	techAuth.Post("/login", techPortalH.Login)
	techAuth.Post("/logout", techPortalH.Logout)
	techAuth.Get("/session", techPortalH.Session)

	// WhatsApp webhook (public — verified by WA service)
	app.Post("/api/whatsapp/webhook", waExtH.Webhook)

	// PWA
	app.Get("/api/pwa/icon", handlers.PwaIcon)

	// Static uploads
	app.Get("/api/uploads/logos/:filename", uploadH.ServeLogoFile)

	// Auth (public)
	auth := app.Group("/api/auth")
	auth.Post("/login", authH.Login)
	auth.Post("/logout", authH.Logout)
	auth.Post("/refresh", authH.Refresh)
	auth.Post("/customer/login", authH.CustomerLogin)
	auth.Post("/customer/verify-otp", authH.CustomerVerifyOTP)
	auth.Post("/agent/login", authH.AgentLogin)

	// Payment gateway webhooks (public — verified by signature)
	webhooks := app.Group("/api/billing/payment-gateway/webhook")
	webhooks.Post("/midtrans", billingH.WebhookMidtrans)
	webhooks.Post("/xendit", billingH.WebhookXendit)
	webhooks.Post("/duitku", billingH.WebhookDuitku)
	webhooks.Post("/tripay", billingH.WebhookTripay)

	// ─── Protected routes (JWT or NextAuth session required) ─────────────────
	api := app.Group("/api", middleware.CombinedAuthMiddleware)

	// Session
	api.Get("/auth/session", authH.Session)

	// Admin dashboard
	admin := api.Group("/admin")
	admin.Get("/stats", adminH.Stats)
	admin.Get("/revenue-chart", adminH.RevenueChart)
	admin.Get("/activity", adminH.Activity)
	admin.Get("/isolated-users", adminH.IsolatedUsers)
	admin.Get("/topup-requests", adminH.TopupRequests)
	admin.Post("/topup-requests/:id/approve", adminH.ApproveTopup)
	admin.Post("/topup-requests/:id/reject", adminH.RejectTopup)
	admin.Get("/suspend-requests", adminH.SuspendRequests)
	admin.Post("/suspend-requests/:id/approve", adminH.ApproveSuspend)
	admin.Post("/suspend-requests/:id/reject", adminH.RejectSuspend)

	// Dashboard alias (same stats, different path)
	api.Get("/dashboard/stats", adminH.Stats)
	api.Get("/dashboard/revenue-chart", adminH.RevenueChart)

	// OLT management
	olt := api.Group("/olt")
	olt.Get("/", oltH.ListOLTs)
	olt.Post("/", oltH.CreateOLT)
	// Monitoring dashboard & global alerts (must be before /:id)
	olt.Post("/test-connection", oltH.TestConnection) // before /:id
	olt.Get("/monitoring", oltH.MonitoringList)
	olt.Post("/monitoring", oltH.MonitoringPoll)
	olt.Get("/alerts", oltH.ListAllAlerts)
	olt.Put("/alerts/:id", oltH.ResolveAlert)
	olt.Get("/:id/onus/register", oltH.GetRegisterMetadata) // before /:id
	olt.Get("/:id", oltH.GetOLT)
	olt.Put("/:id", oltH.UpdateOLT)
	olt.Delete("/:id", oltH.DeleteOLT)
	olt.Post("/:id/sync", oltH.SyncOLT)
	olt.Get("/:id/onus", oltH.ListONUs)
	olt.Get("/:id/onus/:onuId", oltH.GetONU)
	olt.Post("/:id/onus/:onuId/register", oltH.RegisterONU)
	olt.Delete("/:id/onus/:onuId", oltH.DeregisterONU)
	olt.Get("/:id/onus/:onuId/assign", oltH.GetAssignONUCandidates)
	olt.Post("/:id/onus/:onuId/assign", oltH.AssignONU)
	olt.Get("/:id/alerts", oltH.ListAlerts)
	olt.Get("/:id/performance", oltH.ListPerformance)
	olt.Get("/:id/chassis", oltH.GetChassis)
	olt.Get("/:id/pon-stat", oltH.GetPONStat)
	olt.Post("/:id/pon", oltH.PONPortAction)
	olt.Patch("/:id/onus/:onuId", oltH.UpdateONU)

	// PPPoE
	pppoe := api.Group("/pppoe")
	pppoe.Get("/areas", pppoeH.ListAreas)
	pppoe.Post("/areas", pppoeH.CreateArea)
	pppoe.Put("/areas/:id", pppoeH.UpdateArea)
	pppoe.Delete("/areas/:id", pppoeH.DeleteArea)

	pppoe.Get("/profiles", pppoeH.ListProfiles)
	pppoe.Post("/profiles", pppoeH.CreateProfile)
	pppoe.Post("/profiles/sync-mikrotik", pppoeExtH.SyncProfilesMikrotik) // before :id
	pppoe.Post("/profiles/sync-radius", pppoeExtH.SyncProfilesRadius)     // before :id
	pppoe.Put("/profiles/:id", pppoeH.UpdateProfile)
	pppoe.Delete("/profiles/:id", pppoeH.DeleteProfile)

	pppoe.Get("/customers", pppoeH.ListCustomers)
	pppoe.Post("/customers", pppoeH.CreateCustomer)
	pppoe.Get("/customers/export", pppoeExtH.ExportCustomers)           // before :id
	pppoe.Post("/customers/bulk-create", pppoeExtH.BulkCreateCustomers) // before :id
	pppoe.Get("/customers/:id", pppoeH.GetCustomer)
	pppoe.Put("/customers/:id", pppoeH.UpdateCustomer)

	pppoe.Get("/users", pppoeExtH.ListUsersWithFilters)
	pppoe.Post("/users", pppoeH.CreateUser)
	// Static sub-paths — MUST be before /users/:id to prevent wildcard capture in Fiber v3 beta
	pppoe.Get("/users/export", pppoeExtH.ExportUsers)
	pppoe.Get("/users/bulk", pppoeExtH.BulkGet)
	pppoe.Post("/users/bulk", pppoeExtH.BulkImport)
	pppoe.Delete("/users/bulk-delete", pppoeExtH.BulkDelete)
	pppoe.Post("/users/bulk-create", pppoeExtH.BulkCreateUsers)
	pppoe.Get("/users/bulk-status", pppoeExtH.BulkStatus)
	pppoe.Put("/users/bulk-status", pppoeExtH.BulkStatus)
	pppoe.Post("/users/bulk-status", pppoeExtH.BulkStatus)
	pppoe.Get("/users/check-isolation", miscH.CheckIsolationGlobal)
	pppoe.Post("/users/status", miscH.PppoeBatchStatus)
	pppoe.Put("/users/status", miscH.PppoeBatchStatus)
	pppoe.Post("/users/send-notification", miscH.PppoeBatchNotification)
	pppoe.Post("/users/sync-mikrotik", miscH.SyncAllMikrotik)
	pppoe.Get("/users/search", miscH.PppoeSearch)
	// Parameterized user routes — after all static routes
	pppoe.Get("/users/:id", pppoeH.GetUser)
	pppoe.Put("/users/:id", pppoeH.UpdateUser)
	pppoe.Delete("/users/:id", pppoeH.DeleteUser)
	pppoe.Post("/users/:id/suspend", pppoeH.SuspendUser)
	pppoe.Post("/users/:id/activate", pppoeH.ActivateUser)
	pppoe.Post("/users/:id/isolate", pppoeH.IsolateUser)
	pppoe.Post("/users/:id/unisolate", pppoeH.UnisolateUser)
	pppoe.Get("/users/:id/sessions", pppoeH.GetUserSessions)
	pppoe.Get("/users/:id/invoices", pppoeH.GetUserInvoices)
	pppoe.Post("/users/:id/sync-radius", pppoeExtH.SyncUserRadius)
	pppoe.Get("/users/:id/status", pppoeExtH.UserStatus)
	pppoe.Get("/users/:id/check-isolation", pppoeExtH.CheckIsolation)
	pppoe.Post("/users/:id/send-notification", pppoeExtH.SendNotification)
	pppoe.Post("/users/:id/sync-mikrotik", pppoeExtH.SyncMikrotik)
	pppoe.Get("/users/:id/activity", pppoeExtH.UserActivity)
	pppoe.Post("/users/:id/extend", pppoeExtH.ExtendUser)
	pppoe.Post("/users/:id/mark-paid", pppoeExtH.MarkPaid)
	pppoe.Get("/users/:id/available-profiles", miscH.PppoeAvailableProfiles)
	pppoe.Get("/users/:id/traffic", miscH.PppoeUserTraffic)

	pppoe.Get("/registrations", pppoeH.ListRegistrations)
	pppoe.Post("/registrations/:id/approve", pppoeH.ApproveRegistration)
	pppoe.Post("/registrations/:id/reject", pppoeH.RejectRegistration)

	// Registrations (root-level alias — matches TypeScript path /api/registrations)
	regs := api.Group("/registrations")
	regs.Get("/", pppoeH.ListRegistrations)
	regs.Get("/:id", pppoeH.GetRegistration)
	regs.Put("/:id", pppoeH.UpdateRegistration)
	regs.Delete("/:id", pppoeH.DeleteRegistration)
	regs.Post("/:id/approve", pppoeH.ApproveRegistration)
	regs.Post("/:id/reject", pppoeH.RejectRegistration)

	// Billing
	billing := api.Group("/billing")
	billing.Get("/invoices", billingH.ListInvoices)
	billing.Post("/invoices", billingH.CreateInvoice)
	billing.Get("/invoices/:id", billingH.GetInvoice)
	billing.Put("/invoices/:id", billingH.UpdateInvoice)
	billing.Delete("/invoices/:id", billingH.DeleteInvoice)
	billing.Post("/invoices/:id/pay", billingH.PayInvoice)
	billing.Post("/invoices/:id/send-wa", billingH.SendReminderWA)
	billing.Post("/generate-monthly", billingH.GenerateMonthlyInvoices)
	billing.Get("/manual-payments", billingH.ListManualPayments)
	billing.Get("/transactions", billingH.ListTransactions)
	billing.Post("/transactions", billingH.CreateTransaction)
	billing.Get("/transaction-categories", billingH.ListTransactionCategories)
	billing.Post("/transaction-categories", billingH.CreateTransactionCategory)

	// FreeRADIUS
	radiusGrp := api.Group("/radius")
	radiusGrp.Get("/users", radiusH.ListUsers)
	radiusGrp.Post("/users", radiusH.UpsertUser)
	radiusGrp.Delete("/users/:username", radiusH.DeleteUser)
	radiusGrp.Get("/sessions", radiusH.ActiveSessions)
	radiusGrp.Get("/stats", radiusH.Stats)
	radiusGrp.Post("/disconnect/:username", radiusH.Disconnect)

	// Hotspot
	hotspot := api.Group("/hotspot")
	hotspot.Get("/profiles", hotspotH.ListProfiles)
	hotspot.Post("/profiles", hotspotH.CreateProfile)
	hotspot.Put("/profiles/:id", hotspotH.UpdateProfile)
	hotspot.Delete("/profiles/:id", hotspotH.DeleteProfile)
	hotspot.Get("/vouchers", hotspotH.ListVouchers)
	hotspot.Post("/vouchers/generate", hotspotH.GenerateVouchers)
	hotspot.Delete("/vouchers/:id", hotspotH.DeleteVoucher)

	// Agents
	agent := api.Group("/agents")
	agent.Get("/", agentH.ListAgents)
	agent.Post("/", agentH.CreateAgent)
	agent.Get("/vouchers", agentH.ListAgentVouchers)
	agent.Get("/:id", agentH.GetAgent)
	agent.Put("/:id", agentH.UpdateAgent)
	agent.Delete("/:id", agentH.DeleteAgent)
	agent.Get("/:id/sales", agentH.GetAgentSales)
	agent.Get("/:id/deposits", agentH.GetAgentDeposits)
	agent.Post("/:id/topup-balance", agentH.TopupBalance)

	// Network Map
	network := api.Group("/network")
	network.Get("/olts", networkH.ListOLTs)
	network.Post("/olts", networkH.CreateOLT)
	network.Put("/olts", networkH.UpdateOLT)
	network.Delete("/olts", networkH.DeleteOLT)
	network.Get("/odcs", networkH.ListODCs)
	network.Post("/odcs", networkH.CreateODC)
	network.Put("/odcs/:id", networkH.UpdateODC)
	network.Delete("/odcs/:id", networkH.DeleteODC)
	network.Get("/odps", networkH.ListODPs)
	network.Post("/odps", networkH.CreateODP)
	network.Put("/odps/:id", networkH.UpdateODP)
	network.Delete("/odps/:id", networkH.DeleteODP)
	network.Get("/otbs", networkH.ListOTBs)
	network.Post("/otbs", networkH.CreateOTB)
	network.Get("/routers", networkH.ListRouters)
	network.Post("/routers", networkH.CreateRouter)

	// WhatsApp
	wa := api.Group("/whatsapp")
	wa.Get("/providers", waH.ListProviders)
	wa.Post("/providers", waH.CreateProvider)
	wa.Put("/providers/:id", waH.UpdateProvider)
	wa.Get("/templates", waH.ListTemplates)
	wa.Put("/templates/:type", waH.UpdateTemplate)
	wa.Post("/send", waH.SendMessage)
	wa.Get("/history", waH.ListHistory)
	wa.Get("/reminder-settings", waH.GetReminderSettings)
	wa.Put("/reminder-settings", waH.UpdateReminderSettings)

	// Tickets
	tickets := api.Group("/tickets")
	tickets.Get("/", ticketH.ListTickets)
	tickets.Post("/", ticketH.CreateTicket)
	// Ticket extensions — must be before /:id to avoid wildcard capture
	tickets.Get("/categories", ticketExtH.ListCategories)
	tickets.Post("/categories", ticketExtH.CreateCategory)
	tickets.Get("/stats", ticketExtH.Stats)
	tickets.Get("/messages", ticketExtH.ListMessages)
	tickets.Get("/dispatch", ticketExtH.ListDispatch)
	tickets.Post("/dispatch", ticketExtH.Dispatch)
	tickets.Get("/dispatch-data", miscH.TicketDispatchData)
	// Wildcard routes after specific ones
	tickets.Get("/:id", ticketH.GetTicket)
	tickets.Put("/:id", ticketH.UpdateTicket)
	tickets.Post("/:id/reply", ticketH.ReplyTicket)
	tickets.Post("/:id/close", ticketH.CloseTicket)

	// Sessions
	sessions := api.Group("/sessions")
	sessions.Get("/", sessionsH.ListSessions)
	sessions.Post("/disconnect", sessionsH.DisconnectSession)
	sessions.Post("/sync", sessionsH.SyncSessions)
	sessions.Get("/export", sessionsH.ExportSessions)

	// Company
	api.Get("/company", companyH.GetCompany)
	api.Put("/company", companyH.UpdateCompany)
	api.Post("/company", companyH.UpdateCompany) // frontend uses POST

	// Settings
	settings := api.Group("/settings")
	settings.Get("/email", settingsH.GetEmailSettings)
	settings.Post("/email", settingsH.UpdateEmailSettings)
	settings.Get("/isolation", settingsH.GetIsolationSettings)
	settings.Put("/isolation", settingsH.UpdateIsolationSettings)
	// settings/company → alias to company endpoint
	settings.Get("/company", companyH.GetCompany)
	settings.Put("/company", companyH.UpdateCompany)
	// GenieACS settings
	settings.Get("/genieacs", genieacsH.GetSettings)
	settings.Post("/genieacs", genieacsH.SaveSettings)

	// Permissions (RBAC)
	perms := api.Group("/permissions")
	perms.Get("/", permsH.GetPermissions)
	perms.Get("/role-templates", permsH.GetRoleTemplates)
	perms.Get("/role/:role", permsH.GetRolePermissions)
	perms.Put("/role/:role", permsH.UpdateRolePermissions)

	// Inventory
	inv := api.Group("/inventory")
	inv.Get("/categories", inventoryH.ListCategories)
	inv.Post("/categories", inventoryH.CreateCategory)
	inv.Put("/categories/:id", inventoryH.UpdateCategory)
	inv.Delete("/categories/:id", inventoryH.DeleteCategory)
	inv.Get("/suppliers", inventoryH.ListSuppliers)
	inv.Post("/suppliers", inventoryH.CreateSupplier)
	inv.Put("/suppliers/:id", inventoryH.UpdateSupplier)
	inv.Delete("/suppliers/:id", inventoryH.DeleteSupplier)
	inv.Get("/items", inventoryH.ListItems)
	inv.Post("/items", inventoryH.CreateItem)
	inv.Put("/items/:id", inventoryH.UpdateItem)
	inv.Delete("/items/:id", inventoryH.DeleteItem)
	inv.Get("/movements", inventoryH.ListMovements)
	inv.Post("/movements", inventoryH.CreateMovement)

	// Keuangan
	keu := api.Group("/keuangan")
	keu.Get("/transactions", keuanganH.ListTransactions)
	keu.Post("/transactions", keuanganH.CreateTransaction)
	keu.Delete("/transactions/:id", keuanganH.DeleteTransaction)
	keu.Get("/categories", keuanganH.ListCategories)
	keu.Post("/categories", keuanganH.CreateCategory)
	keu.Get("/export", keuanganH.Export)

	// Manual Payments
	manPay := api.Group("/manual-payments")
	manPay.Get("", manualPayH.List)
	manPay.Post("", manualPayH.Create)
	manPay.Put("/:id", manualPayH.Review)
	manPay.Delete("/:id", manualPayH.Delete)

	// Jobs (Admin)
	jobs := api.Group("/admin/jobs")
	jobs.Get("", jobH.List)
	jobs.Get("/stats", jobH.Stats)
	jobs.Post("", jobH.Create)
	jobs.Get("/:id", jobH.Get)
	jobs.Patch("/:id/status", jobH.UpdateStatus)

	// Employees (for job assignment dropdown)
	api.Get("/admin/employees", jobH.ListEmployees)

	// Admin Employees full CRUD
	api.Post("/admin/employees", empAdminH.Create)
	api.Put("/admin/employees/:id", empAdminH.Update)
	api.Delete("/admin/employees/:id", empAdminH.Delete)

	// Job Assignments (alias for jobs with dedicated delete path)
	api.Get("/admin/job-assignments", jobH.List)
	api.Delete("/admin/job-assignments/:id", jobH.DeleteJob)

	// GenieACS proxy
	api.Post("/genieacs/devices/:deviceId/wifi", genieacsH.UpdateWifi)
	api.Post("/genieacs/devices/:deviceId/connection-request", genieacsH.ConnectionRequest)
	api.Get("/genieacs/tasks", genieacsH.ListTasks)
	api.Delete("/genieacs/tasks/:taskId", genieacsH.DeleteTask)

	// Users list (with ODP/ODC filters)
	api.Get("/users/list", pppoeH.ListUsersForSelect)

	// Cron
	cronGrp := api.Group("/cron")
	cronGrp.Get("/history", cronH.ListHistory)
	cronGrp.Post("/trigger/:job", cronH.TriggerJob)

	// Customer Portal (JWT-less, uses customer session token validated against DB)
	customer := app.Group("/api/customer", middleware.NewCustomerAuthMiddleware(db))
	customer.Get("/profile", customerH.GetProfile)
	customer.Get("/me", customerH.GetMe)
	customer.Get("/dashboard", customerH.GetDashboard)
	customer.Get("/packages", customerH.GetPackages)
	customer.Post("/auto-renewal", customerH.ToggleAutoRenewal)
	customer.Get("/notifications", customerH.GetNotifications)
	customer.Get("/payment-history", customerH.GetPaymentHistory)
	customer.Get("/invoices", customerH.GetInvoices)
	customer.Post("/invoices/:id/pay", customerH.PayInvoice)
	customer.Get("/usage", customerH.GetUsage)
	customer.Post("/topup-request", customerH.CreateTopupRequest)
	customer.Get("/suspend-request", customerH.GetSuspendRequest)
	customer.Post("/suspend-request", customerH.CreateSuspendRequest)
	customer.Delete("/suspend-request", customerH.CancelSuspendRequest)
	customer.Get("/tickets", customerH.GetCustomerTickets)
	customer.Post("/tickets", customerH.CreateCustomerTicket)
	customer.Post("/push-subscribe", customerH.PushSubscribe)

	// ─── WebSocket ────────────────────────────────────────────────────────────
	app.Get("/ws/olt/:id", func(c fiber.Ctx) error {
		oltID := c.Params("id")
		type rawFasthttpCtx interface {
			RequestCtx() *fasthttp.RequestCtx
		}
		rc, ok := c.(rawFasthttpCtx)
		if !ok {
			return fiber.ErrUpgradeRequired
		}
		return wsUpgrader.Upgrade(rc.RequestCtx(), func(conn *fws.Conn) {
			client := hub.Register(conn, oltID)
			defer hub.Unregister(client)
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		})
	})

	// ─── NEW HANDLERS (Batch 5) ──────────────────────────────────────────────
	// (handlers instantiated earlier, before auth group)

	// Notifications
	api.Get("/notifications", notifH.List)
	api.Put("/notifications", notifH.MarkRead)
	api.Delete("/notifications/:id", notifH.Delete)

	// FreeRADIUS management
	freeradius := api.Group("/freeradius")
	freeradius.Get("/status", frH.GetStatus)
	freeradius.Post("/start", frH.Start)
	freeradius.Post("/stop", frH.Stop)
	freeradius.Post("/restart", frH.Restart)
	freeradius.Get("/logs", frH.GetLogs)
	freeradius.Get("/radcheck", frH.GetRadcheck)
	freeradius.Post("/radcheck", frH.CreateRadcheck)
	freeradius.Delete("/radcheck", frH.DeleteRadcheck)
	freeradius.Post("/radtest", frH.RunRadtest)
	freeradius.Get("/config/list", frH.ListConfigs)
	freeradius.Post("/config/read", frH.ReadConfig)
	freeradius.Post("/config/save", frH.SaveConfig)

	// Root-level invoices (separate from /billing/invoices)
	invoicesGrp := api.Group("/invoices")
	invoicesGrp.Get("/export", invExtH.Export)
	invoicesGrp.Get("/counts", invExtH.Counts)
	invoicesGrp.Post("/generate", invExtH.Generate)
	invoicesGrp.Post("/send-reminder", invExtH.SendReminder)
	invoicesGrp.Post("/send-reminders-bulk", invExtH.SendRemindersBulk)
	invoicesGrp.Get("/by-token/:token", invExtH.GetByToken)
	invoicesGrp.Get("/:id/pdf", invExtH.GetPDF)
	invoicesGrp.Get("/", invExtH.List)
	invoicesGrp.Post("/", invExtH.Create)
	invoicesGrp.Delete("/", invExtH.Delete)

	// Referrals
	api.Get("/admin/referrals/config", refH.GetConfig)
	api.Put("/admin/referrals/config", refH.UpdateConfig)
	api.Get("/admin/referrals", refH.List)
	api.Put("/admin/referrals/:id", refH.UpdateStatus)
	api.Post("/admin/referrals/:id", refH.UpdateStatus) // frontend sends POST
	api.Delete("/admin/referrals/:id", refH.Delete)

	// Admin Users
	api.Get("/admin/users", adminUserH.List)
	api.Post("/admin/users", adminUserH.Create)
	api.Get("/admin/users/:id/permissions", adminUserH.GetPermissions)
	api.Put("/admin/users/:id/permissions", adminUserH.SetPermissions)
	api.Get("/admin/users/:id", adminUserH.Get)
	api.Put("/admin/users/:id", adminUserH.Update)
	api.Delete("/admin/users/:id", adminUserH.Delete)

	// Admin Technicians
	api.Get("/admin/technicians", techAdminH.List)
	api.Post("/admin/technicians", techAdminH.Create)
	api.Get("/admin/technicians/:id", techAdminH.Get)
	api.Put("/admin/technicians/:id", techAdminH.Update)
	api.Delete("/admin/technicians/:id", techAdminH.Delete)

	// Activity Log
	api.Get("/admin/activity-logs", actH.List)

	// Hotspot extensions (singular /voucher for individual ops)
	hotspot.Get("/voucher/export", hotspotExtH.Export)
	hotspot.Get("/voucher/bulk", hotspotExtH.BulkGetOrExport) // GET ?type=template|export
	hotspot.Post("/voucher/bulk", hotspotExtH.BulkGenerate)
	hotspot.Post("/voucher/bulk-delete", hotspotExtH.BulkDelete)
	hotspot.Post("/voucher/resync", hotspotExtH.Resync)
	hotspot.Post("/voucher/send-whatsapp", hotspotExtH.SendWhatsapp)
	hotspot.Delete("/voucher/delete-expired", hotspotExtH.DeleteExpired)
	hotspot.Post("/voucher/delete-expired", hotspotExtH.DeleteExpired) // frontend uses POST
	hotspot.Get("/vouchers/validate", hotspotExtH.ValidateVoucher)
	hotspot.Post("/vouchers/validate", hotspotExtH.ValidateVoucher) // frontend sends POST
	hotspot.Get("/voucher/:id", hotspotExtH.GetVoucher)
	hotspot.Delete("/voucher/:id", hotspotExtH.DeleteVoucher)
	hotspot.Get("/voucher", hotspotH.ListVouchers)      // GET /api/hotspot/voucher?... — list vouchers
	hotspot.Post("/voucher", hotspotH.GenerateVouchers) // POST /api/hotspot/voucher
	hotspot.Delete("/voucher", hotspotExtH.DeleteBatch) // DELETE /api/hotspot/voucher?batchCode=X
	hotspot.Patch("/voucher", hotspotExtH.BulkEdit)     // PATCH /api/hotspot/voucher — bulk edit
	hotspot.Get("/rekap-voucher/export", hotspotExtH.ExportRekap)
	hotspot.Get("/rekap-voucher", hotspotExtH.RekapVoucher)
	hotspot.Get("/agents/balance", hotspotExtH.AgentBalance)
	hotspot.Post("/agents/balance", hotspotExtH.AdjustBalance) // POST balance adjust
	hotspot.Get("/agents/:id/history", hotspotExtH.AgentHistory)
	hotspot.Get("/agents", hotspotExtH.ListAgents)
	hotspot.Post("/agents", hotspotExtH.CreateAgent)   // create agent
	hotspot.Put("/agents", hotspotExtH.UpdateAgent)    // update agent (id in body)
	hotspot.Delete("/agents", hotspotExtH.DeleteAgent) // delete agent ?id=X

	// ─── Batch 12: Hotspot extras ─────────────────────────────────────────────
	hotspot.Post("/voucher/delete-multiple", hotspotExtH.DeleteMultiple)

	// Voucher Templates
	api.Get("/voucher-templates", voucherTplH.List)
	api.Post("/voucher-templates", voucherTplH.Create)
	api.Get("/voucher-templates/:id", voucherTplH.Get)
	api.Put("/voucher-templates/:id", voucherTplH.Update)
	api.Delete("/voucher-templates/:id", voucherTplH.Delete)

	// Analytics
	api.Get("/admin/analytics", analyticsH.GetAnalytics)
	api.Get("/dashboard/analytics", analyticsH.GetAnalytics)
	api.Get("/dashboard/traffic", analyticsH.GetTraffic)

	// Settings extensions
	api.Get("/settings/email/templates", settingsExtH.ListEmailTemplates)
	api.Put("/settings/email/templates/:type", settingsExtH.UpdateEmailTemplate)
	api.Post("/settings/email/test", settingsExtH.TestEmail)
	api.Get("/settings/timezone", settingsExtH.GetTimezone)
	api.Post("/settings/timezone", settingsExtH.SetTimezone) // save timezone
	api.Get("/settings/map", settingsExtH.GetMapSettings)
	api.Put("/settings/map", settingsExtH.UpdateMapSettings)
	api.Get("/email/history", settingsExtH.EmailHistory)

	// Backup
	api.Get("/backup/history", backupH.History)
	api.Post("/backup/create", backupH.Create)
	api.Get("/backup/download/:id", backupH.Download)
	api.Post("/backup/restore", backupH.Restore)
	api.Get("/backup/telegram/settings", backupH.GetTelegramSettings)
	api.Put("/backup/telegram/settings", backupH.UpdateTelegramSettings)
	api.Delete("/backup/delete/:id", backupH.Delete)

	// Telegram
	api.Get("/telegram/settings", telegramH.GetSettings)
	api.Put("/telegram/settings", telegramH.UpdateSettings)
	api.Post("/telegram/settings", telegramH.UpdateSettings) // frontend sends POST
	api.Post("/telegram/test", telegramH.Test)
	api.Post("/telegram/send-backup", telegramH.SendBackup)
	api.Post("/telegram/test-backup", telegramH.TestBackup)
	api.Post("/telegram/send-health", telegramH.SendHealth)

	// Push Notifications
	api.Get("/admin/push-notifications", pushH.ListBroadcasts)
	api.Post("/push/send", pushH.Send)
	api.Post("/push/subscribe", pushH.Subscribe)
	api.Delete("/push/unsubscribe", pushH.Unsubscribe)
	api.Post("/push/unsubscribe", pushH.Unsubscribe) // frontend sends POST
	api.Get("/push/vapid-public-key", pushH.GetVapidKey)

	// Push extended (agent & technician subscriptions)
	api.Post("/push/agent-subscribe", pushExtH.AgentSubscribe)
	api.Delete("/push/agent-unsubscribe", pushExtH.AgentUnsubscribe)
	api.Post("/push/agent-unsubscribe", pushExtH.AgentUnsubscribe) // frontend sends POST
	api.Post("/push/technician-subscribe", pushExtH.TechnicianSubscribe)
	api.Delete("/push/technician-unsubscribe", pushExtH.TechnicianUnsubscribe)
	api.Post("/push/technician-unsubscribe", pushExtH.TechnicianUnsubscribe) // frontend sends POST

	// OLT extensions (alert management, monitoring, metrics)
	api.Get("/olt/alerts", oltExtH.ListAlerts)
	api.Get("/olt/monitoring", oltExtH.Monitoring)
	api.Get("/olt/metrics", oltExtH.Metrics)
	api.Get("/olt/alerts/:id", oltExtH.GetAlert)
	api.Put("/olt/alerts/:id/resolve", oltExtH.ResolveAlert)

	// PPPoE extended routes — now consolidated in the pppoe group above (order-safe)

	// Technician portal — protected routes with own JWT check (techFromHeader)
	tech := app.Group("/api/technician")
	tech.Get("/profile", techPortalH.GetProfile)
	tech.Get("/work-orders", techPortalH.ListWorkOrders)
	tech.Get("/tasks", techPortalH.ListTasks)
	tech.Get("/customers", techPortalH.ListCustomers)
	tech.Post("/customers/create", techPortalH.CreateCustomer)
	tech.Get("/form-data", techPortalH.FormData)
	tech.Get("/isolated", techPortalH.IsolatedUsers)
	tech.Get("/offline", techPortalH.OfflineUsers)
	tech.Get("/sessions", techPortalH.Sessions)
	tech.Get("/tickets", techPortalH.ListTickets)
	tech.Get("/monitor", techPortalH.Monitor)
	tech.Get("/genieacs", techPortalH.GenieacsSummary)
	tech.Get("/genieacs/devices", techPortalH.GenieacsDevices)
	tech.Get("/genieacs/devices/:deviceId", techPortalH.GenieacsDevice)
	tech.Post("/upload", techPortalH.Upload)

	// Upload endpoints
	api.Post("/upload/logo", uploadH.UploadLogo)
	api.Post("/upload/payment-proof", uploadH.UploadPaymentProof)
	api.Post("/upload/pppoe-customer", uploadH.UploadCustomerPhoto)

	// WhatsApp extended (broadcast, provider actions)
	api.Post("/whatsapp/broadcast", waExtH.Broadcast)
	api.Post("/whatsapp/broadcast-invoice", waExtH.BroadcastInvoice)
	api.Get("/whatsapp/providers/:id/status", waExtH.ProviderStatus)
	api.Get("/whatsapp/providers/:id/qr", waExtH.ProviderQR)
	api.Post("/whatsapp/providers/:id/restart", waExtH.ProviderRestart)
	api.Post("/whatsapp/providers/:id/test", waExtH.ProviderTest)

	// GenieACS settings
	api.Get("/settings/genieacs/devices", settingsGnH.ListDevices)
	api.Get("/settings/genieacs/devices/:deviceId", settingsGnH.GetDevice)
	api.Get("/settings/genieacs/devices/:deviceId/detail", settingsGnH.DeviceDetail)
	api.Get("/settings/genieacs/devices/:deviceId/parameters", settingsGnH.DeviceParameters)
	api.Post("/settings/genieacs/devices/:deviceId/reboot", settingsGnH.RebootDevice)
	api.Post("/settings/genieacs/devices/:deviceId/refresh", settingsGnH.RefreshDevice)
	api.Get("/settings/genieacs/tasks", settingsGnH.ListTasks)
	api.Post("/settings/genieacs/test", settingsGnH.TestConnection)
	api.Get("/settings/genieacs/parameter-display", settingsGnH.ListParameterDisplay)
	api.Put("/settings/genieacs/parameter-display/:id", settingsGnH.UpdateParameterDisplay)
	api.Post("/settings/genieacs/parameter-display/reset", settingsGnH.ResetParameterDisplay)
	api.Get("/settings/genieacs/virtual-parameters", settingsGnH.ListVirtualParameters)
	api.Get("/settings/genieacs/virtual-parameters/:id", settingsGnH.GetVirtualParameter)

	// Isolation templates
	api.Get("/settings/isolation/templates", settingsGnH.ListIsolationTemplates)
	api.Get("/settings/isolation/templates/:id", settingsGnH.GetIsolationTemplate)
	api.Put("/settings/isolation/templates/:id", settingsGnH.UpdateIsolationTemplate)

	// System & services
	api.Post("/settings/restart-services", settingsGnH.RestartServices)
	api.Get("/sessions/realtime", settingsGnH.RealtimeSessions)
	api.Get("/system/radius", settingsGnH.SystemRadius)
	api.Get("/sse/voucher-updates", settingsGnH.SSEVoucherUpdates)

	// ─────────────────────────────────────────────────────────────────────────

	// ─── Batch 7: Customer Extended ──────────────────────────────────────────
	// Public customer OTP (before api group, so declared as app route)
	app.Post("/api/customer/auth/send-otp", custExtH.AuthSendOTP)

	// Customer portal extended (customer auth group)
	customer.Get("/invoices-ext", custExtH.GetInvoices)
	customer.Post("/cash-payment", custExtH.CashPayment)
	customer.Post("/manual-payment", custExtH.ManualPayment)
	customer.Get("/products", custExtH.GetProducts)
	customer.Post("/profile/send-otp", custExtH.ProfileSendOTP)
	customer.Post("/profile/verify-otp", custExtH.ProfileVerifyOTP)
	customer.Post("/renewal", custExtH.Renewal)
	customer.Get("/sessions-list", custExtH.GetSessions)
	customer.Post("/extend", custExtH.Extend)
	customer.Get("/ont", custExtH.GetONT)
	customer.Post("/ont/update-wifi", custExtH.UpdateWifi)
	customer.Get("/diagnostics/ping", custExtH.DiagnosticsPing)
	customer.Get("/diagnostics/speedtest", custExtH.DiagnosticsSpeedtest)
	customer.Get("/diagnostics/traceroute", custExtH.DiagnosticsTraceroute)

	// ─── Batch 7: WhatsApp CRUD ───────────────────────────────────────────────
	api.Get("/whatsapp/providers-list", waCrudH.ListProviders)
	api.Post("/whatsapp/providers-create", waCrudH.CreateProvider)
	api.Get("/whatsapp/providers/:id/detail", waCrudH.GetProvider)
	api.Put("/whatsapp/providers/:id/update", waCrudH.UpdateProvider)
	api.Delete("/whatsapp/providers/:id/remove", waCrudH.DeleteProvider)
	api.Delete("/whatsapp/providers/:id", waCrudH.DeleteProvider) // REST alias: frontend calls DELETE /providers/:id
	api.Get("/whatsapp/history", waCrudH.ListHistory)             // primary route used by frontend
	api.Get("/whatsapp/history-list", waCrudH.ListHistory)        // legacy alias
	api.Delete("/whatsapp/history/delete", waCrudH.DeleteHistory)
	api.Get("/whatsapp/templates-list", waCrudH.ListTemplates)
	api.Get("/whatsapp/templates", waCrudH.ListTemplates) // REST alias: overrides waH.ListTemplates (wrong format)
	api.Post("/whatsapp/templates-create", waCrudH.CreateTemplate)
	api.Get("/whatsapp/templates/:id/detail", waCrudH.GetTemplate)
	api.Put("/whatsapp/templates/:id/update", waCrudH.UpdateTemplate)
	api.Put("/whatsapp/templates/:id", waCrudH.UpdateTemplate) // REST alias: overrides waH.UpdateTemplate (type vs id mismatch)
	api.Delete("/whatsapp/templates/:id/remove", waCrudH.DeleteTemplate)
	api.Delete("/whatsapp/templates/:id", waCrudH.DeleteTemplate) // REST alias
	api.Get("/whatsapp/reminder-settings-ext", waCrudH.GetReminderSettings)
	api.Put("/whatsapp/reminder-settings-ext", waCrudH.UpdateReminderSettings)
	api.Post("/whatsapp/send-ext", waCrudH.Send)

	// ─── Batch 7: Network Extended ────────────────────────────────────────────
	api.Get("/network/routers/status", networkH.RouterStatus)
	api.Post("/network/routers/status", networkH.RouterStatus) // frontend sends POST
	api.Post("/network/routers/import", networkH.ImportRouters)
	api.Get("/network/routers/template", networkH.RouterImportTemplate)
	api.Get("/network/routers/:id/detail", networkH.GetRouter)
	api.Put("/network/routers/:id", networkH.UpdateRouter)
	api.Delete("/network/routers/:id", networkH.DeleteRouter)
	api.Post("/network/routers/:id/test-connection", networkH.TestRouterConnection)
	api.Post("/network/routers/:id/detect-public-ip", networkH.DetectPublicIP)
	api.Get("/network/routers/:id/interfaces", networkH.RouterInterfaces)
	api.Get("/network/routers/:id/isolation-settings", networkH.RouterIsolationSettings)
	api.Post("/network/routers/:id/ping-olt", networkH.PingOLT)
	api.Post("/network/routers/:id/setup-isolir", networkH.SetupIsolir)
	api.Get("/network/routers/:id/uplinks", networkH.RouterUplinks)
	api.Get("/network/olts-list", networkH.ListOLTs)
	api.Post("/network/olts/import", networkH.ImportOLTs)
	api.Get("/network/olts/template", networkH.OLTImportTemplate)
	api.Get("/network/olt-routers", networkH.ListOLTRouters)
	api.Post("/network/odcs/import", networkH.ImportODCs)
	api.Get("/network/odcs/template", networkH.ODCImportTemplate)
	api.Post("/network/odps/import", networkH.ImportODPs)
	api.Get("/network/odps/template", networkH.ODPImportTemplate)
	api.Get("/network/otbs/stats", networkH.OTBStats)
	api.Post("/network/otbs/import", networkH.ImportOTBs)
	api.Get("/network/otbs/:id", networkH.GetOTB)
	api.Get("/network/fiber-paths", networkH.ListFiberPaths)
	api.Post("/network/fiber-paths", networkH.CreateFiberPath)
	api.Get("/network/fiber-paths/trace", networkH.TraceFiberPath)
	api.Get("/network/fiber-paths/:id", networkH.GetFiberPath)
	api.Put("/network/fiber-paths/:id", networkH.UpdateFiberPath)
	api.Delete("/network/fiber-paths/:id", networkH.DeleteFiberPath)
	api.Get("/network/joint-closures", networkH.ListJointClosures)
	api.Post("/network/joint-closures", networkH.CreateJointClosure)
	api.Post("/network/joint-closures/import", networkH.ImportJointClosures)
	api.Get("/network/joint-closures/:id", networkH.GetJointClosure)
	api.Put("/network/joint-closures/:id", networkH.UpdateJointClosure)
	api.Delete("/network/joint-closures/:id", networkH.DeleteJointClosure)
	api.Get("/network/nodes", networkH.ListNodes)
	api.Post("/network/nodes", networkH.CreateNode)
	api.Get("/network/nodes/:id", networkH.GetNode)
	api.Put("/network/nodes/:id", networkH.UpdateNode)
	api.Delete("/network/nodes/:id", networkH.DeleteNode)
	api.Get("/network/servers", networkH.ListServers)
	api.Get("/network/paths", networkH.ListPaths)
	api.Post("/network/detect-nas", networkH.DetectNAS)
	api.Post("/network/customers/assign", networkH.AssignCustomer)
	api.Get("/customers/with-location", networkH.CustomersWithLocation)

	// ─── Batch 7: Admin Jobs & Registrations ─────────────────────────────────
	api.Get("/admin/registrations", adminJobsH.ListRegistrations)
	api.Get("/admin/registrations/:id", adminJobsH.GetRegistration)
	api.Post("/admin/registrations/:id/approve", adminJobsH.ApproveRegistration)
	api.Post("/admin/registrations/:id/reject", adminJobsH.RejectRegistration)
	api.Post("/admin/registrations/:id/mark-installed", adminJobsH.MarkInstalled)
	api.Post("/admin/registrations/:id/request-info", adminJobsH.RequestInfo)
	api.Post("/admin/registrations/:id/tech-survey", adminJobsH.TechSurvey)
	api.Get("/customer-registrations", adminJobsH.ListCustomerRegistrations)
	api.Get("/customer-registrations/:id", adminJobsH.GetCustomerRegistration)
	api.Post("/customer-registrations/:id/activate", adminJobsH.ActivateCustomerRegistration)
	api.Post("/customer-registrations/:id/admin-approve", adminJobsH.AdminApproveRegistration)
	api.Post("/customer-registrations/:id/admin-reject", adminJobsH.AdminRejectRegistration)
	api.Post("/customer-registrations/:id/install", adminJobsH.InstallRegistration)
	api.Post("/customer-registrations/:id/request-info", adminJobsH.CustomerRequestInfo)
	api.Post("/customer-registrations/:id/tech-survey", adminJobsH.CustomerTechSurvey)
	api.Get("/admin/jobs/approvals", adminJobsH.ListApprovals)
	api.Get("/admin/jobs/stats-ext", adminJobsH.JobStats)
	api.Get("/admin/recurring-jobs", adminJobsH.ListRecurringJobs)
	api.Post("/admin/jobs-ext", adminJobsH.CreateJob)
	api.Get("/admin/jobs-ext/:id", adminJobsH.GetJob)
	api.Put("/admin/jobs-ext/:id", adminJobsH.UpdateJob)
	api.Post("/admin/jobs/:id/approve", adminJobsH.ApproveJob)
	api.Post("/admin/jobs/:id/reject", adminJobsH.RejectJob)
	api.Post("/admin/jobs/:id/escalate", adminJobsH.EscalateJob)
	api.Post("/admin/jobs/:id/submit-approval", adminJobsH.SubmitApproval)
	api.Get("/admin/jobs/:id/materials", adminJobsH.JobMaterials)
	api.Get("/admin/jobs/:id/approval-history", adminJobsH.ApprovalHistory)
	tech.Get("/jobs", adminJobsH.TechListJobs)
	tech.Get("/jobs/:id", adminJobsH.TechGetJob)
	tech.Post("/jobs/:id/complete", adminJobsH.TechCompleteJob)
	tech.Get("/jobs/:id/customer-data", adminJobsH.TechCustomerData)
	tech.Post("/jobs/:id/generate-credentials", adminJobsH.TechGenCredentials)
	api.Get("/jobs/team", adminJobsH.TeamJobs)

	// ─── Batch 7: Misc ───────────────────────────────────────────────────────
	app.Post("/api/coordinator/auth/request-otp", miscH.CoordinatorRequestOTP)
	app.Post("/api/coordinator/auth/verify-otp", miscH.CoordinatorVerifyOTP)
	app.Post("/api/coordinator/auth/logout", miscH.CoordinatorLogout)
	app.Get("/api/coordinator/auth/session", miscH.CoordinatorSession)
	api.Get("/coordinator/stats", miscH.CoordinatorStats)
	api.Get("/coordinator/tasks", miscH.CoordinatorTasks)
	api.Get("/health/db", miscH.HealthDB)
	api.Get("/health/radius", miscH.HealthRadius)
	app.Post("/api/radius/authorize", miscH.RadiusAuthorize)
	app.Post("/api/radius/post-auth", miscH.RadiusPostAuth)
	app.Post("/api/radius/coa", miscH.RadiusCOA)
	api.Post("/pppoe/upload-photo", miscH.PppoeUploadPhoto)
	api.Get("/public/homepage", miscH.PublicHomepage)
	api.Get("/company/info", miscH.CompanyInfo)
	api.Get("/admin/nas", miscH.ListNAS)
	api.Post("/email/broadcast-invoice", miscH.EmailBroadcastInvoice)
	api.Post("/notifications/generate", miscH.GenerateNotifications)
	api.Post("/notifications/job-reassigned", miscH.NotifyJobReassigned)
	api.Post("/notifications/need-support", miscH.NotifyNeedSupport)
	api.Post("/notifications/support-resolved", miscH.NotifySupportResolved)
	app.Get("/api/pay/:token", miscH.PayByToken)
	api.Get("/payment-gateway/config", miscH.PaymentGatewayConfig)
	api.Post("/payment-gateway/config", miscH.PaymentGatewaySaveConfig)
	api.Get("/payment-gateway/webhook-logs", miscH.PaymentGatewayWebhookLogs)
	api.Get("/inventory/items/available", inventoryH.ListItems)
	api.Post("/inventory/stock-in", inventoryH.CreateMovement)
	api.Post("/inventory/stock-out", inventoryH.CreateMovement)
	api.Post("/inventory/consume", inventoryH.CreateMovement)
	api.Post("/inventory/reserve", inventoryH.CreateMovement)
	api.Get("/inventory/variance", miscH.InventoryVariance)
	api.Post("/inventory/reorder", miscH.InventoryReorder)

	// ─── Batch 8: Company bank ────────────────────────────────────────────────
	api.Get("/company/bank", miscH.CompanyBank)
	api.Post("/company/bank", miscH.UpdateCompanyBank)

	// ─── Batch 8: OLT uplink ─────────────────────────────────────────────────
	olt.Get("/:id/uplink", oltH.GetUplink)
	olt.Post("/:id/uplink", oltH.CreateUplink)

	// ─── Batch 8: Cron schedule management ───────────────────────────────────
	api.Get("/cron/schedules", cronH.ListSchedules)
	api.Put("/cron/schedules/:job", cronH.UpdateSchedule)
	api.Delete("/cron/schedules/:job", cronH.DeleteSchedule)

	// ─── Batch 8: Payment routes ──────────────────────────────────────────────
	// Webhook is public (no admin auth, verified by signature)
	app.Post("/api/payment/webhook", paymentH.Webhook)
	// QRIS Mandiri: Android app notify + frontend polling (public, auth by device_key)
	app.Post("/api/payment/qris-notify", paymentH.QrisNotify)
	app.Get("/api/payment/qris-status", paymentH.QrisStatus)
	// QRIS test: admin-only simulation endpoint
	api.Post("/payment/qris-test", paymentH.QrisTest)
	api.Post("/payment/create", paymentH.CreatePayment)
	api.Get("/payment/check-order", paymentH.CheckOrder)

	// ─── Batch 8: Customer Portal Ext 2 ──────────────────────────────────────
	// Public: bypass-login (generates session from admin token)
	app.Post("/api/customer/auth/bypass-login", custExt2H.BypassLogin)
	// Authenticated customer routes
	customer.Get("/payments", custExt2H.GetPayments)
	customer.Post("/payments", custExt2H.CreatePayment)
	customer.Post("/payments/:id/proof", custExt2H.UploadPaymentProof)
	customer.Get("/payment-methods", custExt2H.GetPaymentMethods)
	customer.Post("/notifications/:id/read", custExt2H.MarkNotificationRead)
	customer.Post("/topup-direct", custExt2H.TopupDirect)
	customer.Post("/upgrade", custExt2H.UpgradePackage)
	customer.Post("/upgrade-package", custExt2H.UpgradePackageAlt)
	customer.Get("/referral", custExt2H.GetReferral)
	customer.Post("/referral", custExt2H.CreateReferral)
	customer.Get("/referral/rewards", custExt2H.GetReferralRewards)
	customer.Post("/invoices/:id/manual-payment", custExt2H.PayInvoiceManual)

	// ─── Batch 8: GenieACS extended routes ───────────────────────────────────
	genieacs := api.Group("/genieacs")
	// Devices
	genieacs.Get("/devices", genieacsH.ListDevices)
	genieacs.Get("/devices/:deviceId", genieacsH.GetDevice)
	genieacs.Delete("/devices/:deviceId", genieacsH.DeleteDevice)
	genieacs.Get("/devices/:deviceId/all-parameters", genieacsH.DeviceAllParameters)
	genieacs.Post("/devices/:deviceId/download", genieacsH.DeviceDownload)
	genieacs.Get("/devices/:deviceId/parameters", genieacsH.GetDeviceParameters)
	genieacs.Post("/devices/:deviceId/parameters", genieacsH.SetDeviceParameters)
	genieacs.Get("/devices/:deviceId/tasks", genieacsH.GetDeviceTasks)
	genieacs.Post("/devices/:deviceId/tasks", genieacsH.CreateDeviceTask)
	genieacs.Post("/devices/:deviceId/wan", genieacsH.DeviceWAN)
	genieacs.Put("/devices/:deviceId/wan", genieacsH.DeviceWAN)
	genieacs.Delete("/devices/:deviceId/wan", genieacsH.DeviceWAN)
	genieacs.Get("/devices/:deviceId/wifi", genieacsH.GetDeviceWifi)
	genieacs.Post("/devices/:deviceId/reboot", genieacsH.RebootDevice)
	genieacs.Post("/devices/:deviceId/refresh", genieacsH.RefreshDevice)
	genieacs.Post("/devices/:deviceId/factory-reset", genieacsH.FactoryResetDevice)
	// Tasks
	genieacs.Post("/tasks/:taskId/retry", genieacsH.RetryTask)
	// Sync
	genieacs.Post("/sync", genieacsH.SyncDevices)
	// Presets
	genieacs.Get("/presets", genieacsH.ListPresets)
	genieacs.Post("/presets", genieacsH.CreatePreset)
	genieacs.Get("/presets/:presetId", genieacsH.GetPreset)
	genieacs.Put("/presets/:presetId", genieacsH.UpdatePreset)
	genieacs.Delete("/presets/:presetId", genieacsH.DeletePreset)
	// Provisions
	genieacs.Get("/provisions", genieacsH.ListProvisions)
	genieacs.Post("/provisions", genieacsH.CreateProvision)
	genieacs.Get("/provisions/:provisionId", genieacsH.GetProvision)
	genieacs.Put("/provisions/:provisionId", genieacsH.UpdateProvision)
	genieacs.Delete("/provisions/:provisionId", genieacsH.DeleteProvision)
	// Virtual Parameters
	genieacs.Get("/virtual-parameters", genieacsH.ListVirtualParameters)
	genieacs.Post("/virtual-parameters", genieacsH.CreateVirtualParameter)
	genieacs.Get("/virtual-parameters/:vpId", genieacsH.GetVirtualParameter)
	genieacs.Put("/virtual-parameters/:vpId", genieacsH.UpdateVirtualParameter)
	genieacs.Delete("/virtual-parameters/:vpId", genieacsH.DeleteVirtualParameter)
	// Files
	genieacs.Get("/files", genieacsH.ListFiles)
	genieacs.Post("/files", genieacsH.UploadFile)
	genieacs.Delete("/files", genieacsH.DeleteFile)
	// Faults
	genieacs.Get("/faults", genieacsH.ListFaults)
	genieacs.Delete("/faults/:faultId", genieacsH.DeleteFault)
	// Config
	genieacs.Get("/config", genieacsH.ListConfig)
	genieacs.Put("/config", genieacsH.UpdateConfig)
	genieacs.Delete("/config", genieacsH.DeleteConfig)
	// Backup
	genieacs.Get("/backup", genieacsH.GetBackup)
	genieacs.Post("/backup", genieacsH.CreateBackup)
	// Auto-provision
	genieacs.Get("/auto-provision", genieacsH.ListAutoProvision)
	genieacs.Post("/auto-provision", genieacsH.CreateAutoProvision)
	genieacs.Delete("/auto-provision", genieacsH.DeleteAutoProvision)

	// ─── Batch 9: Backup info routes ─────────────────────────────────────────
	api.Get("/backup", backupH.ListBackups)
	api.Get("/backup/health", backupH.Health)
	api.Post("/backup/telegram/test", telegramH.TestBackup)

	// ─── Batch 9: Cron info routes ───────────────────────────────────────────
	api.Get("/cron", cronH.Info)
	api.Get("/cron/status", cronH.Status)
	api.Get("/cron/status", cronH.Status)

	// ─── Batch 9: Invoice extras ─────────────────────────────────────────────
	api.Post("/invoices/:id/void", invExtH.Void)
	api.Post("/invoices/bulk-delete", invExtH.BulkDelete)

	// ─── Batch 9: Manual payments bulk-delete ────────────────────────────────
	api.Post("/manual-payments/bulk-delete", manualPayH.BulkDelete)

	// ─── Batch 9: Tickets create-job ─────────────────────────────────────────
	api.Post("/tickets/:id/create-job", ticketExtH.CreateJob)

	// ─── Batch 9: Jobs photos ────────────────────────────────────────────────
	api.Get("/jobs/:id/photos", jobH.ListPhotos)

	// ─── Batch 9: Agent self-service portal ──────────────────────────────────
	api.Get("/agent/dashboard", agentH.Dashboard)
	api.Post("/agent/deposit/create", agentH.CreateDeposit)
	app.Post("/api/agent/deposit/webhook", agentH.DepositWebhook) // public
	api.Post("/agent/generate-voucher", agentH.GenerateVoucher)
	api.Post("/agent/record-sales", agentH.RecordSales)

	// ─── Batch 9: Payments approval ──────────────────────────────────────────
	api.Get("/payments", paymentsApprovalH.List)
	api.Post("/payments/:id/approve", paymentsApprovalH.Approve)
	api.Post("/payments/:id/reject", paymentsApprovalH.Reject)
	api.Get("/payments/manual", paymentsApprovalH.ListManual)
	api.Post("/payments/manual", paymentsApprovalH.CreateManual)

	// ─── Batch 9: Payment gateways public list ───────────────────────────────
	app.Get("/api/payment/gateways", paymentH.ListGateways)

	// ─── Batch 9: Invoice templates ──────────────────────────────────────────
	api.Get("/invoice-templates", invoiceTplH.List)
	api.Post("/invoice-templates", invoiceTplH.Create)
	api.Get("/invoice-templates/:id", invoiceTplH.Get)
	api.Put("/invoice-templates/:id", invoiceTplH.Update)
	api.Delete("/invoice-templates/:id", invoiceTplH.Delete)
	api.Post("/invoice-templates/:id/default", invoiceTplH.SetDefault)

	// ─── Batch 9: Payroll templates ──────────────────────────────────────────
	api.Get("/payroll-templates", payrollTplH.List)
	api.Post("/payroll-templates", payrollTplH.Create)
	api.Get("/payroll-templates/:id", payrollTplH.Get)
	api.Put("/payroll-templates/:id", payrollTplH.Update)
	api.Delete("/payroll-templates/:id", payrollTplH.Delete)
	api.Post("/payroll-templates/:id/default", payrollTplH.SetDefault)

	// ─── Batch 9: Troubleshooting ────────────────────────────────────────────
	api.Get("/troubleshooting/checklists", troubleshootH.ListChecklists)
	api.Post("/troubleshooting/checklists", troubleshootH.CreateChecklist)
	api.Get("/troubleshooting/jobs", troubleshootH.ListJobs)
	api.Get("/troubleshooting/jobs/:id", troubleshootH.GetJob)
	api.Get("/troubleshooting/jobs/:id/materials", troubleshootH.JobMaterials)

	// ─── Batch 9: E-Voucher public portal ────────────────────────────────────
	app.Get("/api/evoucher/profiles", evoucherH.ListProfiles)
	app.Post("/api/evoucher/purchase", evoucherH.Purchase)
	app.Get("/api/evoucher/order/:token", evoucherH.GetOrder)
	// E-Voucher admin
	api.Get("/admin/evoucher/orders", evoucherH.AdminListOrders)
	api.Post("/admin/evoucher/orders/:id/cancel", evoucherH.AdminCancelOrder)
	api.Post("/admin/evoucher/orders/:id/resend", evoucherH.AdminResendOrder)
	api.Delete("/admin/evoucher/orders/bulk-delete", evoucherH.AdminBulkDelete)

	// ─── Batch 10: Admin VPN ─────────────────────────────────────────────────
	api.Get("/admin/vpn/clients", adminVPNH.ListClients)
	api.Post("/admin/vpn/clients", adminVPNH.CreateClient)
	api.Get("/admin/vpn/clients/:id", adminVPNH.GetClient)
	api.Put("/admin/vpn/clients/:id", adminVPNH.UpdateClient)
	api.Delete("/admin/vpn/clients/:id", adminVPNH.DeleteClient)
	api.Post("/admin/vpn/clients/:id/approve", adminVPNH.ApproveClient)
	api.Post("/admin/vpn/clients/:id/reject", adminVPNH.RejectClient)
	api.Get("/admin/vpn/clients/:id/config", adminVPNH.GetClientConfig)
	api.Get("/admin/vpn/clients/:id/qr", adminVPNH.GetClientQR)
	api.Post("/admin/vpn/generate-keys", adminVPNH.GenerateKeys)
	api.Get("/admin/vpn/service", adminVPNH.GetService)
	api.Put("/admin/vpn/service", adminVPNH.UpdateService)
	api.Get("/admin/vpn/settings", adminVPNH.GetSettings)
	api.Put("/admin/vpn/settings", adminVPNH.UpdateSettings)
	api.Get("/admin/vpn/sites", adminVPNH.ListSites)
	api.Post("/admin/vpn/sites", adminVPNH.CreateSite)
	api.Get("/admin/vpn/sites/:id", adminVPNH.GetSite)
	api.Put("/admin/vpn/sites/:id", adminVPNH.UpdateSite)
	api.Delete("/admin/vpn/sites/:id", adminVPNH.DeleteSite)
	api.Get("/admin/vpn/sites/:id/config", adminVPNH.GetSiteConfig)

	// ─── Batch 10: Admin Payroll ─────────────────────────────────────────────
	api.Get("/admin/payroll", adminPayrollH.List)
	api.Get("/admin/payroll/:id", adminPayrollH.Get)
	api.Put("/admin/payroll/:id", adminPayrollH.Update)
	api.Delete("/admin/payroll/:id", adminPayrollH.Delete)
	api.Post("/admin/payroll/generate", adminPayrollH.Generate)
	api.Get("/admin/payroll/overtime", adminPayrollH.ListOvertime)
	api.Post("/admin/payroll/overtime", adminPayrollH.CreateOvertime)
	api.Post("/admin/payroll/pay/:id", adminPayrollH.Pay)

	// ─── Batch 10: Admin HR (Attendance) ─────────────────────────────────────
	api.Get("/admin/attendance", adminHRH.ListAttendance)
	api.Post("/admin/attendance", adminHRH.CreateAttendance)
	api.Post("/admin/attendance/bulk-delete", adminHRH.BulkDeleteAttendance)
	api.Get("/admin/attendance-locations", adminHRH.ListLocations)
	api.Post("/admin/attendance-locations", adminHRH.CreateLocation)

	// ─── Batch 10: Admin HR (Cash Advances) ──────────────────────────────────
	api.Get("/admin/cash-advances", adminHRH.ListCashAdvances)
	api.Post("/admin/cash-advances", adminHRH.CreateCashAdvance)
	api.Get("/admin/cash-advances/:id", adminHRH.GetCashAdvance)
	api.Put("/admin/cash-advances/:id", adminHRH.UpdateCashAdvance)
	api.Delete("/admin/cash-advances/:id", adminHRH.DeleteCashAdvance)
	api.Post("/admin/cash-advances/pay/:id", adminHRH.PayCashAdvance)

	// ─── Batch 10: Admin HR (Commissions) ────────────────────────────────────
	api.Get("/admin/commissions", adminHRH.ListCommissions)
	api.Post("/admin/commissions", adminHRH.CreateCommission)
	api.Get("/admin/commissions/:id", adminHRH.GetCommission)
	api.Put("/admin/commissions/:id", adminHRH.UpdateCommission)
	api.Delete("/admin/commissions/:id", adminHRH.DeleteCommission)
	api.Post("/admin/commissions/:id/approve", adminHRH.ApproveCommission)
	api.Post("/admin/commissions/:id/reject", adminHRH.RejectCommission)

	// ─── Batch 10: FCM device tokens ─────────────────────────────────────────
	app.Post("/api/fcm/token", fcmH.RegisterToken)
	app.Post("/api/fcm/test", fcmH.Test)

	// ─── Batch 11: Admin misc ─────────────────────────────────────────────────
	// APK
	api.Get("/admin/apk/env", adminMiscH.ApkEnv)
	api.Get("/admin/apk/status", adminMiscH.ApkStatus)
	api.Post("/admin/apk/trigger", adminMiscH.ApkTrigger)
	api.Post("/admin/apk/build", adminMiscH.ApkBuild)
	api.Get("/admin/apk/file", adminMiscH.ApkFile)
	api.Get("/admin/download-apk", adminMiscH.DownloadApk)
	// Cloudflare tunnel
	api.Get("/admin/cloudflare-tunnel", adminMiscH.GetCloudflareTunnel)
	api.Put("/admin/cloudflare-tunnel", adminMiscH.UpdateCloudflareTunnel)
	api.Post("/admin/cloudflare-tunnel", adminMiscH.UpdateCloudflareTunnel) // frontend sends POST
	// System info
	api.Get("/admin/system/info", adminMiscH.SystemInfo)
	// FreeRADIUS backup
	api.Get("/admin/system/freeradius-backup", adminMiscH.ListFreeradiusBackups)
	api.Post("/admin/system/freeradius-backup", adminMiscH.CreateFreeradiusBackup)
	api.Get("/admin/system/freeradius-backup/download", adminMiscH.DownloadFreeradiusBackup)
	api.Post("/admin/system/freeradius-backup/restore", adminMiscH.RestoreFreeradiusBackup)
	api.Post("/admin/system/freeradius-backup/upload", adminMiscH.UploadFreeradiusBackup)
	// Admin profile 2FA
	api.Get("/admin/profile/2fa", adminMiscH.Get2FA)
	api.Post("/admin/profile/2fa", adminMiscH.Update2FA)
	// PPPoE admin
	api.Post("/admin/pppoe/sync-all-radius", adminMiscH.SyncAllRadius)
	api.Post("/admin/pppoe/users/:id/deposit", adminMiscH.PPPoEUserDeposit)
	// Invoice import
	api.Post("/admin/invoices/import", adminMiscH.ImportInvoices)
	// Reports / Laporan
	api.Get("/admin/laporan", adminMiscH.Laporan)
	// OLT model profiles
	api.Get("/admin/olt/model-profiles", adminMiscH.ListOLTModelProfiles)
	api.Post("/admin/olt/model-profiles", adminMiscH.CreateOLTModelProfile)
	api.Post("/admin/olt/test-connection", adminMiscH.TestOLTConnection)

	// ─── Batch 11: Network VPN/VPS ────────────────────────────────────────────
	api.Get("/network/vpn-server", networkVPNH.GetVPNServer)
	api.Post("/network/vpn-server", networkVPNH.UpdateVPNServer)
	api.Post("/network/vpn-server/setup", networkVPNH.SetupVPNServer)
	api.Post("/network/vpn-server/test", networkVPNH.TestVPNServer)
	api.Post("/network/vpn-server/l2tp-control", networkVPNH.L2TPControl)
	api.Post("/network/vpn-server/pptp-control", networkVPNH.PPTPControl)
	api.Post("/network/vpn-server/sstp-control", networkVPNH.SSTPControl)
	api.Get("/network/vpn-client", networkVPNH.ListVPNClients)
	api.Post("/network/vpn-client", networkVPNH.CreateVPNClient)
	api.Patch("/network/vpn-client", networkVPNH.PatchVPNClient)
	api.Put("/network/vpn-client", networkVPNH.PutVPNClient)
	api.Delete("/network/vpn-client", networkVPNH.DeleteVPNClient)
	api.Get("/network/vpn-routing", networkVPNH.ListVPNRouting)
	api.Post("/network/vpn-routing", networkVPNH.CreateVPNRoute)
	api.Get("/network/vps-info", networkVPNH.GetVPSInfo)
	api.Get("/network/vps-l2tp-info", networkVPNH.GetVPSL2TPInfo)
	api.Get("/network/vps-l2tp-peer", networkVPNH.ListL2TPPeers)
	api.Post("/network/vps-l2tp-peer", networkVPNH.CreateL2TPPeer)
	api.Get("/network/vps-wg-peer", networkVPNH.ListWGPeers)
	api.Post("/network/vps-wg-peer", networkVPNH.CreateWGPeer)
	api.Patch("/network/vps-wg-peer", networkVPNH.PatchWGServerConfig)
	api.Patch("/network/vps-l2tp-peer", networkVPNH.PatchL2TPServerConfig)

	// ─── Batch 11: Agent portal extras ────────────────────────────────────────
	api.Get("/agent/deposit/check", agentH.DepositCheck)
	api.Post("/agent/deposit/manual-request", agentH.DepositManualRequest)
	api.Get("/agent/deposit/payment-methods", agentH.ListDepositPaymentMethods)
	api.Get("/agent/notifications", agentH.GetAgentNotifications)
	api.Get("/agent/sessions", agentH.GetAgentSessions)
	api.Get("/agent/tickets", agentH.GetAgentTickets)
	api.Get("/agent/tickets/:id", agentH.GetAgentTicket)

	// ─── Batch 12 handlers ───────────────────────────────────────────────────
	networkInfraH := handlers.NewNetworkInfraHandler(db)

	// ─── Batch 12: Auth ───────────────────────────────────────────────────────
	app.Post("/api/auth/logout-log", miscH.LogoutLog)

	// ─── Batch 12: Admin ─────────────────────────────────────────────────────
	api.Get("/admin/agent-deposits", miscH.AdminAgentDeposits)
	api.Post("/admin/isolate-user", miscH.AdminIsolateUser)
	api.Get("/admin/settings/isolation", miscH.AdminGetIsolationSettings)
	api.Put("/admin/settings/isolation", miscH.AdminUpdateIsolationSettings)
	api.Get("/admin/settings/isolation/mikrotik-script", miscH.AdminGetMikrotikScript)
	api.Get("/admin/users/:id/renewal", pppoeExtH.ExtendUser)
	api.Post("/admin/users/:id/renewal", pppoeExtH.ExtendUser) // frontend sends POST

	// ─── Batch 12: Cron extras ────────────────────────────────────────────────
	api.Get("/cron/olt-poll", miscH.CronOLTPoll)
	api.Post("/cron/olt-poll", miscH.CronOLTPoll)
	api.Get("/cron/telegram", miscH.CronTelegram)
	api.Post("/cron/telegram", miscH.CronTelegram)

	// ─── Batch 12: Invoices check ─────────────────────────────────────────────
	api.Get("/invoices/check", miscH.CheckInvoice)

	// ─── Batch 12: Payment extras ─────────────────────────────────────────────
	app.Post("/api/pay/manual", miscH.PayManual)
	api.Get("/payment/duitku-methods", miscH.DuitkuMethods)

	// ─── Batch 12: RADIUS accounting ─────────────────────────────────────────
	app.Get("/api/radius/accounting", miscH.RadiusAccounting)
	app.Post("/api/radius/accounting", miscH.RadiusAccounting)

	// ─── Batch 12: Network routers extras ────────────────────────────────────
	api.Post("/network/routers/:id/setup-radius", miscH.SetupRadiusOnRouter)
	api.Post("/network/routers/test", miscH.TestRouterGeneric)
	api.Post("/network/routers/test-gateway", miscH.TestGateway)

	// ─── Batch 12: OLT extras ─────────────────────────────────────────────────
	olt.Post("/:id/onus/:onuId/reboot", miscH.RebootONU)
	olt.Post("/:id/onus/batch-reboot", miscH.BatchRebootONUs)
	olt.Get("/:id/onus/:onuId/detail", miscH.ONUDetail)
	olt.Post("/:id/onus/:onuId/clean-config", miscH.CleanONUConfig)
	// Frontend calls DELETE /onus/:onuId/delete — map to DeregisterONU
	olt.Delete("/:id/onus/:onuId/delete", oltH.DeregisterONU)

	// ─── Batch 12: Network infrastructure ────────────────────────────────────
	api.Get("/network/cables", networkInfraH.ListCables)
	api.Post("/network/cables", networkInfraH.CreateCable)
	api.Get("/network/cables/:id", networkInfraH.GetCable)
	api.Put("/network/cables/:id", networkInfraH.UpdateCable)
	api.Delete("/network/cables/:id", networkInfraH.DeleteCable)
	api.Get("/network/connections", networkInfraH.ListConnections)
	api.Post("/network/connections", networkInfraH.CreateConnection)
	api.Get("/network/cores", networkInfraH.ListCores)
	api.Post("/network/cores", networkInfraH.CreateCore)
	api.Get("/network/joint-closures/:id/segments", networkInfraH.ListJointClosureSegments)
	api.Post("/network/joint-closures/:id/segments", networkInfraH.CreateJointClosureSegment)
	api.Get("/network/joint-closures/:id/splices", networkInfraH.ListJointClosureSplices)
	api.Post("/network/joint-closures/:id/splices", networkInfraH.CreateJointClosureSplice)
	api.Get("/network/joint-closures/template", networkInfraH.JointClosureTemplate)
	api.Get("/network/otbs/:id/feeder-cables", networkInfraH.ListFeederCables)
	api.Post("/network/otbs/:id/feeder-cables", networkInfraH.CreateFeederCable)
	api.Get("/network/otbs/:id/segments", networkInfraH.ListOTBSegments)
	api.Post("/network/otbs/:id/segments", networkInfraH.CreateOTBSegment)
	api.Get("/network/splices", networkInfraH.ListSplices)
	api.Post("/network/splices", networkInfraH.CreateSplice)
	api.Get("/network/splices/:id", networkInfraH.GetSplice)
	api.Put("/network/splices/:id", networkInfraH.UpdateSplice)
	api.Delete("/network/splices/:id", networkInfraH.DeleteSplice)
	api.Get("/network/trace", networkInfraH.NetworkTrace)
	api.Post("/network/trace", networkInfraH.NetworkTrace)
	api.Post("/network/auto-connect", networkInfraH.AutoConnect)

	// ─── Batch 12: Hotspot extras ─────────────────────────────────────────────
	hotspot.Post("/voucher/delete-multiple", hotspotExtH.DeleteMultiple)

	// ─── Batch 12: Customer portal extras ────────────────────────────────────
	customer.Get("/wifi", custExt2H.GetWifi)
	customer.Put("/wifi", custExt2H.UpdateWifiSettings)
	customer.Post("/wifi", custExt2H.UpdateWifiSettings) // frontend sends POST
	customer.Post("/ont/reboot", custExt2H.RebootONT)
	customer.Post("/invoice/regenerate-payment", custExt2H.RegeneratePayment)
	customer.Get("/invoices/payment", custExt2H.InvoicePayment)
	customer.Post("/invoices/payment", custExt2H.InvoicePayment)

	// ─── Batch 13: Auth path aliases ─────────────────────────────────────────
	// Next.js frontend calls these paths; alias to existing auth handlers
	app.Post("/api/customer/auth/login", authH.CustomerLogin)
	app.Post("/api/customer/auth/verify-otp", authH.CustomerVerifyOTP)
	app.Post("/api/customer/login", authH.CustomerLogin)
	app.Post("/api/agent/login", authH.AgentLogin)

	// ─── Batch 13: Hotspot voucher singular path ──────────────────────────────
	hotspot.Get("/voucher", hotspotH.ListVouchers)
	hotspot.Post("/voucher", hotspotH.GenerateVouchers)

	// ─── Batch 13: Network OLTs status ───────────────────────────────────────
	api.Get("/network/olts/status", miscH.NetworkOLTStatus)
	api.Post("/network/olts/status", miscH.NetworkOLTStatus) // frontend sends POST

	// ─── Batch 14: Audit fixes ────────────────────────────────────────────────
	// POST /api/olt/test-connection (non-admin alias; same handler as /admin/olt/test-connection)
	olt.Post("/test-connection", adminMiscH.TestOLTConnection)
	// GET /api/pppoe/customers/bulk (template download) + POST alias for bulk-create
	api.Get("/pppoe/customers/bulk", pppoeExtH.BulkCustomersTemplate)
	api.Post("/pppoe/customers/bulk", pppoeExtH.BulkCreateCustomers)
	// PUT /api/admin/suspend-requests/:id (unified action: APPROVE | REJECT)
	admin.Put("/suspend-requests/:id", adminH.SuspendRequestAction)

	// ─────────────────────────────────────────────────────────────────────────

	return app
}
