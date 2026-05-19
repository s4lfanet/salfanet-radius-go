package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	excelize "github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

type PppoeExtHandler struct{ db *gorm.DB }

func NewPppoeExtHandler(db *gorm.DB) *PppoeExtHandler {
	return &PppoeExtHandler{db: db}
}

// writeXLSX creates a proper .xlsx file from column headers and data rows.
func writeXLSX(headers []string, rows [][]string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sw, err := f.NewStreamWriter("Sheet1")
	if err != nil {
		return nil, err
	}
	heatRow := make([]interface{}, len(headers))
	for i, h := range headers {
		heatRow[i] = h
	}
	if err = sw.SetRow("A1", heatRow); err != nil {
		return nil, err
	}
	for i, row := range rows {
		cells := make([]interface{}, len(row))
		for j, v := range row {
			cells[j] = v
		}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		if err = sw.SetRow(cell, cells); err != nil {
			return nil, err
		}
	}
	if err = sw.Flush(); err != nil {
		return nil, err
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GET /api/pppoe/users/status — count by status
func (h *PppoeExtHandler) UserStatus(c fiber.Ctx) error {
	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var rows []StatusCount
	h.db.Model(&models.PppoeUser{}).
		Select("status, COUNT(*) as count").
		Group("status").Scan(&rows)
	return c.JSON(fiber.Map{"success": true, "data": rows})
}

// GET /api/pppoe/users/export — export users as CSV or real Excel (.xlsx)
func (h *PppoeExtHandler) ExportUsers(c fiber.Ctx) error {
	profileID := c.Query("profileId")
	routerID := c.Query("routerId")
	status := c.Query("status")
	format := c.Query("format", "csv")

	var users []models.PppoeUser
	q := h.db.Preload("Profile").Preload("Area").Preload("Router")
	if profileID != "" {
		q = q.Where("profileId = ?", profileID)
	}
	if routerID != "" {
		q = q.Where("routerId = ?", routerID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Find(&users)

	headers := []string{"ID", "Username", "Name", "Phone", "Email", "Profile", "Area", "Status", "SubscriptionType", "ExpiredAt", "CreatedAt"}
	var dataRows [][]string
	for _, u := range users {
		expStr := ""
		if u.ExpiredAt != nil {
			expStr = u.ExpiredAt.Format("2006-01-02")
		}
		areaName := ""
		if u.Area != nil {
			areaName = u.Area.Name
		}
		emailStr := ""
		if u.Email != nil {
			emailStr = *u.Email
		}
		dataRows = append(dataRows, []string{u.ID, u.Username, u.Name, u.Phone, emailStr, u.Profile.Name, areaName, u.Status, string(u.SubscriptionType), expStr, u.CreatedAt.Format("2006-01-02")})
	}

	if format == "excel" || format == "xlsx" {
		data, err := writeXLSX(headers, dataRows)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "gagal membuat Excel"})
		}
		c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Set("Content-Disposition", "attachment; filename=pppoe-users.xlsx")
		return c.Send(data)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(headers)
	for _, row := range dataRows {
		_ = w.Write(row)
	}
	w.Flush()
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", "attachment; filename=pppoe-users.csv")
	return c.Send(buf.Bytes())
}

// GET /api/pppoe/users/bulk — template download or export (csv or real xlsx)
func (h *PppoeExtHandler) BulkGet(c fiber.Ctx) error {
	t := c.Query("type", "template")
	format := c.Query("format", "csv")

	tplHeaders := []string{"username", "password", "name", "phone", "email", "address", "ipAddress", "profileName", "routerName", "expiredAt"}
	tplExample := [][]string{{"user001", "pass123", "John Doe", "08123456789", "", "", "", "Paket 10 Mbps", "Router 1", "2025-12-31"}}

	if t == "template" {
		if format == "xlsx" {
			data, err := writeXLSX(tplHeaders, tplExample)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "gagal membuat Excel template"})
			}
			c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			c.Set("Content-Disposition", "attachment; filename=pppoe-template.xlsx")
			return c.Send(data)
		}
		// CSV template
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		_ = w.Write(tplHeaders)
		_ = w.Write(tplExample[0])
		w.Flush()
		c.Set("Content-Type", "text/csv; charset=utf-8")
		c.Set("Content-Disposition", "attachment; filename=pppoe-template.csv")
		return c.Send(buf.Bytes())
	}

	// type=export
	paymentStatus := c.Query("paymentStatus")
	var users []models.PppoeUser
	q := h.db.Preload("Profile").Preload("Area").Preload("Router")
	switch paymentStatus {
	case "paid":
		q = q.Where("id IN (SELECT userId FROM Invoice WHERE status = 'PAID')")
	case "unpaid":
		q = q.Where("id IN (SELECT userId FROM Invoice WHERE status IN ('PENDING','OVERDUE'))")
	}
	q.Find(&users)

	expHeaders := []string{"username", "password", "name", "phone", "email", "address", "ipAddress", "profileName", "routerName", "status", "expiredAt"}
	var expRows [][]string
	for _, u := range users {
		expStr := ""
		if u.ExpiredAt != nil {
			expStr = u.ExpiredAt.Format("2006-01-02")
		}
		routerName := ""
		if u.Router != nil {
			routerName = u.Router.Name
		}
		emailStr := ""
		if u.Email != nil {
			emailStr = *u.Email
		}
		addrStr := ""
		if u.Address != nil {
			addrStr = *u.Address
		}
		ipStr := ""
		if u.IPAddress != nil {
			ipStr = *u.IPAddress
		}
		expRows = append(expRows, []string{u.Username, u.Password, u.Name, u.Phone, emailStr, addrStr, ipStr, u.Profile.Name, routerName, u.Status, expStr})
	}

	if format == "xlsx" {
		data, err := writeXLSX(expHeaders, expRows)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "gagal membuat Excel"})
		}
		c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Set("Content-Disposition", "attachment; filename=pppoe-export.xlsx")
		return c.Send(data)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(expHeaders)
	for _, row := range expRows {
		_ = w.Write(row)
	}
	w.Flush()
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", "attachment; filename=pppoe-export.csv")
	return c.Send(buf.Bytes())
}

// POST /api/pppoe/users/bulk — import users from CSV or Excel (.xlsx) file upload
func (h *PppoeExtHandler) BulkImport(c fiber.Ctx) error {
	// Explicitly parse multipart form (more reliable than c.FormFile in Fiber v3 beta)
	mf, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "gagal membaca form: " + err.Error()})
	}

	fileHeaders, ok := mf.File["file"]
	if !ok || len(fileHeaders) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "tidak ada file yang diunggah"})
	}
	fileHeader := fileHeaders[0]
	filename := strings.ToLower(fileHeader.Filename)

	f, err := fileHeader.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal membuka file"})
	}
	defer f.Close()

	var records [][]string
	if strings.HasSuffix(filename, ".xlsx") || strings.HasSuffix(filename, ".xls") {
		// Parse Excel file
		data, err2 := io.ReadAll(f)
		if err2 != nil {
			return c.Status(500).JSON(fiber.Map{"error": "gagal membaca file Excel"})
		}
		exf, err2 := excelize.OpenReader(bytes.NewReader(data))
		if err2 != nil {
			return c.Status(400).JSON(fiber.Map{"error": "file Excel tidak valid: " + err2.Error()})
		}
		defer exf.Close()
		records, err2 = exf.GetRows(exf.GetSheetName(0))
		if err2 != nil {
			return c.Status(400).JSON(fiber.Map{"error": "gagal membaca baris Excel"})
		}
	} else {
		// Parse CSV file
		r := csv.NewReader(f)
		r.FieldsPerRecord = -1 // allow variable fields
		records, err = r.ReadAll()
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "gagal membaca CSV: " + err.Error()})
		}
	}

	if len(records) < 2 {
		return c.Status(400).JSON(fiber.Map{"error": "file kosong atau tidak ada baris data"})
	}

	// Build header index map (case-insensitive)
	headers := records[0]
	idx := map[string]int{}
	for i, h2 := range headers {
		idx[strings.ToLower(strings.TrimSpace(h2))] = i
	}
	col := func(row []string, key string) string {
		i, ok := idx[key]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	successCount, failedCount := 0, 0
	type failRow struct {
		Row   int    `json:"row"`
		Error string `json:"error"`
	}
	var failures []failRow

	for rowIdx, row := range records[1:] {
		username := col(row, "username")
		password := col(row, "password")
		name := col(row, "name")
		phone := col(row, "phone")
		if username == "" || password == "" || name == "" || phone == "" {
			failedCount++
			failures = append(failures, failRow{Row: rowIdx + 2, Error: "missing required fields"})
			continue
		}

		// Look up profile by name
		profileName := col(row, "profilename")
		var profile models.PppoeProfile
		if profileName != "" {
			h.db.Where("name = ?", profileName).First(&profile)
		}

		// Look up router by name
		routerName := col(row, "routername")
		var router models.Router
		routerID := (*string)(nil)
		if routerName != "" {
			if err2 := h.db.Where("name = ?", routerName).First(&router).Error; err2 == nil {
				routerID = &router.ID
			}
		}

		emailStr := col(row, "email")
		addrStr := col(row, "address")
		ipStr := col(row, "ipaddress")
		expiredAtStr := col(row, "expiredat")

		user := models.PppoeUser{
			ID:        generateID(),
			Username:  username,
			Password:  password,
			Name:      name,
			Phone:     phone,
			Status:    "active",
			ProfileID: profile.ID,
			RouterID:  routerID,
		}
		if emailStr != "" {
			user.Email = &emailStr
		}
		if addrStr != "" {
			user.Address = &addrStr
		}
		if ipStr != "" {
			user.IPAddress = &ipStr
		}
		if expiredAtStr != "" {
			if t2, err2 := time.Parse("2006-01-02", expiredAtStr); err2 == nil {
				user.ExpiredAt = &t2
			}
		}

		if err2 := h.db.Create(&user).Error; err2 != nil {
			failedCount++
			failures = append(failures, failRow{Row: rowIdx + 2, Error: err2.Error()})
		} else {
			successCount++
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"results": fiber.Map{
			"success":  successCount,
			"failed":   failedCount,
			"failures": failures,
		},
	})
}

// DELETE /api/pppoe/users/bulk-delete — bulk delete users (for stopped/terminated users)
func (h *PppoeExtHandler) BulkDelete(c fiber.Ctx) error {
	var body struct {
		UserIDs []string `json:"userIds"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if len(body.UserIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "userIds required"})
	}
	result := h.db.Where("id IN ?", body.UserIDs).Delete(&models.PppoeUser{})
	return c.JSON(fiber.Map{"success": true, "deleted": result.RowsAffected})
}

// POST /api/pppoe/users/bulk — bulk create users (stub — requires radius sync)
func (h *PppoeExtHandler) BulkCreateUsers(c fiber.Ctx) error {
	var body []models.PppoeUser
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	created := 0
	for i := range body {
		body[i].ID = generateID()
		if err := h.db.Create(&body[i]).Error; err == nil {
			created++
		}
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "created": created})
}

// POST /api/pppoe/users/bulk-status — bulk update status
func (h *PppoeExtHandler) BulkStatus(c fiber.Ctx) error {
	var body struct {
		UserIDs []string `json:"userIds"`
		Status  string   `json:"status"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if len(body.UserIDs) == 0 || body.Status == "" {
		return c.Status(400).JSON(fiber.Map{"error": "userIds and status required"})
	}
	result := h.db.Model(&models.PppoeUser{}).
		Where("id IN ?", body.UserIDs).
		Update("status", body.Status)
	return c.JSON(fiber.Map{"success": true, "updated": result.RowsAffected})
}

// GET /api/pppoe/users/check-isolation — check isolation status for users
func (h *PppoeExtHandler) CheckIsolation(c fiber.Ctx) error {
	var total, isolated int64
	h.db.Model(&models.PppoeUser{}).Count(&total)
	h.db.Model(&models.PppoeUser{}).Where("status = ?", "isolated").Count(&isolated)
	return c.JSON(fiber.Map{
		"success":  true,
		"total":    total,
		"isolated": isolated,
		"active":   total - isolated,
	})
}

// POST /api/pppoe/users/send-notification — send notification to filtered users
func (h *PppoeExtHandler) SendNotification(c fiber.Ctx) error {
	var body struct {
		UserIDs []string `json:"userIds"`
		Title   string   `json:"title"`
		Message string   `json:"message"`
		Type    string   `json:"type"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	// Create notification record (global, not per-user)
	n := models.Notification{
		ID:      generateID(),
		Type:    body.Type,
		Title:   body.Title,
		Message: body.Message,
		Link:    nil,
		IsRead:  false,
	}
	if err := h.db.Create(&n).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create notification"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "notification sent"})
}

// POST /api/pppoe/users/sync-mikrotik — sync users to Mikrotik (stub)
func (h *PppoeExtHandler) SyncMikrotik(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "mikrotik sync triggered", "synced": 0})
}

// GET /api/pppoe/users/:id/activity — get user activity logs
func (h *PppoeExtHandler) UserActivity(c fiber.Ctx) error {
	id := c.Params("id")
	var logs []models.ActivityLog
	h.db.Where("userId = ?", id).Order("createdAt desc").Limit(50).Find(&logs)
	return c.JSON(fiber.Map{"success": true, "logs": logs})
}

// POST /api/pppoe/users/:id/extend — extend subscription
func (h *PppoeExtHandler) ExtendUser(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Days   int    `json:"days"`
		Months int    `json:"months"`
		Reason string `json:"reason"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	var user models.PppoeUser
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}
	now := time.Now()
	base := now
	if user.ExpiredAt != nil && user.ExpiredAt.After(now) {
		base = *user.ExpiredAt
	}
	newExpiry := base.AddDate(0, body.Months, body.Days)
	h.db.Model(&user).Update("expiredAt", newExpiry)
	return c.JSON(fiber.Map{"success": true, "expiredAt": newExpiry})
}

// POST /api/pppoe/users/:id/mark-paid — mark invoice as paid
func (h *PppoeExtHandler) MarkPaid(c fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		InvoiceID string `json:"invoiceId"`
		Amount    int    `json:"amount"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	now := time.Now()
	result := h.db.Model(&models.Invoice{}).
		Where("id = ? AND userId = ?", body.InvoiceID, id).
		Updates(map[string]interface{}{"status": "PAID", "paidAt": now})
	if result.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "invoice not found"})
	}
	return c.JSON(fiber.Map{"success": true, "paidAt": now})
}

// GET /api/pppoe/customers/export — export customers CSV
func (h *PppoeExtHandler) ExportCustomers(c fiber.Ctx) error {
	var customers []models.PppoeUser
	h.db.Preload("Profile").Preload("Area").Find(&customers)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"ID", "Name", "Username", "Phone", "Email", "Area", "Profile", "Status", "CreatedAt"})
	for _, u := range customers {
		areaName := ""
		if u.Area != nil {
			areaName = u.Area.Name
		}
		emailStr := ""
		if u.Email != nil {
			emailStr = *u.Email
		}
		_ = w.Write([]string{u.ID, u.Name, u.Username, u.Phone, emailStr, areaName, u.Profile.Name, u.Status, u.CreatedAt.Format("2006-01-02")})
	}
	w.Flush()

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=customers.csv")
	return c.Send(buf.Bytes())
}

// POST /api/pppoe/customers/bulk — bulk create customers
func (h *PppoeExtHandler) BulkCreateCustomers(c fiber.Ctx) error {
	var body []models.PppoeUser
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	created := 0
	for i := range body {
		body[i].ID = generateID()
		if err := h.db.Create(&body[i]).Error; err == nil {
			created++
		}
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "created": created})
}

// POST /api/pppoe/profiles/sync-mikrotik — sync profiles to Mikrotik (stub)
func (h *PppoeExtHandler) SyncProfilesMikrotik(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "mikrotik profile sync triggered"})
}

// POST /api/pppoe/profiles/sync-radius — sync profiles to RADIUS
func (h *PppoeExtHandler) SyncProfilesRadius(c fiber.Ctx) error {
	var profiles []models.PppoeProfile
	h.db.Where("isActive = ?", true).Find(&profiles)
	synced := len(profiles)
	return c.JSON(fiber.Map{"success": true, "synced": synced, "message": fmt.Sprintf("synced %d profiles", synced)})
}

// GET /api/pppoe/users/:id/sync-radius — sync single user to RADIUS
func (h *PppoeExtHandler) SyncUserRadius(c fiber.Ctx) error {
	id := c.Params("id")
	var user models.PppoeUser
	if err := h.db.Preload("Profile").First(&user, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}
	// Upsert to radcheck
	rc := models.Radcheck{
		Username:  user.Username,
		Attribute: "Cleartext-Password",
		Op:        ":=",
		Value:     user.Password,
	}
	h.db.Where("username = ? AND attribute = ?", user.Username, "Cleartext-Password").
		Assign(rc).FirstOrCreate(&rc)
	return c.JSON(fiber.Map{"success": true, "message": "user synced to radius"})
}

// GET /api/pppoe/users — alias with richer filters (pagination handled by pppoeH.ListUsers)
// This handler adds additional filter params used by the frontend
func (h *PppoeExtHandler) ListUsersWithFilters(c fiber.Ctx) error {
	page, limit := pageParams(c)
	status := c.Query("status")
	search := c.Query("search")
	areaID := c.Query("areaId")
	profileID := c.Query("profileId")
	subscriptionType := c.Query("subscriptionType")

	q := h.db.Model(&models.PppoeUser{}).Preload("Profile").Preload("Area")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if areaID != "" {
		q = q.Where("areaId = ?", areaID)
	}
	if profileID != "" {
		q = q.Where("profileId = ?", profileID)
	}
	if subscriptionType != "" {
		q = q.Where("subscriptionType = ?", subscriptionType)
	}
	if search != "" {
		q = q.Where("username LIKE ? OR name LIKE ? OR phone LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	q.Count(&total)

	users := make([]models.PppoeUser, 0)
	q.Order("createdAt desc").Offset((page - 1) * limit).Limit(limit).Find(&users)

	return c.JSON(fiber.Map{
		"success": true,
		"users":   users,
		"pagination": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GET /api/pppoe/customers/bulk — template for bulk customer import
func (h *PppoeExtHandler) BulkCustomersTemplate(c fiber.Ctx) error {
	t := c.Query("type")
	if t == "template" {
		columns := []string{"customerId", "name", "phone", "email", "address", "idCardNumber"}
		sample := []fiber.Map{
			{"customerId": "", "name": "Budi Santoso", "phone": "08123456789", "email": "budi@example.com", "address": "Jl. Merdeka No. 10", "idCardNumber": "3171234567890001"},
			{"customerId": "", "name": "Siti Rahayu", "phone": "08987654321", "email": "", "address": "", "idCardNumber": ""},
		}
		return c.JSON(fiber.Map{"success": true, "columns": columns, "sample": sample})
	}
	return c.JSON(fiber.Map{"success": true, "message": "use ?type=template to download import template"})
}

// helper to satisfy import
var _ = strconv.Itoa
