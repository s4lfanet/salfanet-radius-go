package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	ros "github.com/go-routeros/routeros/v3"
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

	tplHeaders := []string{"username", "password", "name", "phone", "email", "address", "latitude", "longitude", "ipAddress", "macAddress", "profileName", "routerName", "areaName", "subscriptionType", "billingDay", "expiredAt", "comment"}
	tplExample := [][]string{{"user001", "pass123", "John Doe", "08123456789", "", "Jl. Merdeka No. 1", "-6.200000", "106.816666", "", "", "Paket 10 Mbps", "Router 1", "", "POSTPAID", "1", "2025-12-31", ""}}

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

	expHeaders := []string{"username", "password", "name", "phone", "email", "address", "latitude", "longitude", "ipAddress", "macAddress", "profileName", "routerName", "areaName", "subscriptionType", "billingDay", "status", "expiredAt", "comment"}
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
		areaName2 := ""
		if u.Area != nil {
			areaName2 = u.Area.Name
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
		latStr := ""
		if u.Latitude != nil {
			latStr = strconv.FormatFloat(*u.Latitude, 'f', 6, 64)
		}
		lngStr := ""
		if u.Longitude != nil {
			lngStr = strconv.FormatFloat(*u.Longitude, 'f', 6, 64)
		}
		macStr := ""
		if u.MACAddress != nil {
			macStr = *u.MACAddress
		}
		billingDayStr := "1"
		if u.BillingDay != nil {
			billingDayStr = strconv.Itoa(*u.BillingDay)
		}
		commentStr := ""
		if u.Comment != nil {
			commentStr = *u.Comment
		}
		expRows = append(expRows, []string{u.Username, u.Password, u.Name, u.Phone, emailStr, addrStr, latStr, lngStr, ipStr, macStr, u.Profile.Name, routerName, areaName2, string(u.SubscriptionType), billingDayStr, u.Status, expStr, commentStr})
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

	// Optional fallback profileId from form field (like old Next.js system)
	var fallbackProfile *models.PppoeProfile
	if pids, has := mf.Value["profileId"]; has && len(pids) > 0 && pids[0] != "" {
		var fp models.PppoeProfile
		if h.db.Where("id = ?", pids[0]).First(&fp).Error == nil {
			fallbackProfile = &fp
		}
	}

	f, err := fileHeader.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal membuka file"})
	}
	defer f.Close()

	var records [][]string
	if strings.HasSuffix(filename, ".xlsx") || strings.HasSuffix(filename, ".xls") {
		// Read all bytes so we can fallback to CSV if needed
		data, err2 := io.ReadAll(f)
		if err2 != nil {
			return c.Status(500).JSON(fiber.Map{"error": "gagal membaca file Excel"})
		}
		exf, err2 := excelize.OpenReader(bytes.NewReader(data))
		if err2 != nil {
			// Fallback: file might be CSV content disguised as .xlsx (old export format)
			if strings.Contains(err2.Error(), "zip") || strings.Contains(err2.Error(), "ZIP") {
				r := csv.NewReader(bytes.NewReader(data))
				r.FieldsPerRecord = -1
				records, err = r.ReadAll()
				if err != nil {
					return c.Status(400).JSON(fiber.Map{"error": "file tidak valid (bukan Excel maupun CSV yang bisa dibaca)"})
				}
			} else {
				return c.Status(400).JSON(fiber.Map{"error": "file Excel tidak valid: " + err2.Error()})
			}
		} else {
			defer exf.Close()
			records, err2 = exf.GetRows(exf.GetSheetName(0))
			if err2 != nil {
				return c.Status(400).JSON(fiber.Map{"error": "gagal membaca baris Excel"})
			}
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
	// Accept column aliases from ExportUsers-format files
	aliases := map[string]string{
		"profile": "profilename",
		"router":  "routername",
		"area":    "areaname",
	}
	for src, dst := range aliases {
		if v, ok := idx[src]; ok {
			if _, has := idx[dst]; !has {
				idx[dst] = v
			}
		}
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
		Line     int    `json:"line"`
		Username string `json:"username"`
		Error    string `json:"error"`
	}
	var failures []failRow

	for rowIdx, row := range records[1:] {
		// Skip blank rows
		if len(row) == 0 {
			continue
		}
		username := col(row, "username")
		password := col(row, "password")
		name := col(row, "name")
		phone := col(row, "phone")
		if username == "" || name == "" || phone == "" {
			if username == "" {
				continue // skip empty rows
			}
			failedCount++
			failures = append(failures, failRow{Line: rowIdx + 2, Username: username, Error: "username/name/phone wajib diisi"})
			continue
		}
		// Auto-generate password if not provided
		if password == "" {
			pfx := username
			if len(pfx) > 6 {
				pfx = pfx[:6]
			}
			password = pfx + "123"
		}

		// Profile lookup (case-insensitive; required)
		profileName := col(row, "profilename")
		var profile models.PppoeProfile
		if profileName != "" {
			h.db.Where("LOWER(name) = LOWER(?)", profileName).First(&profile)
		}
		// Fallback to the profile selected in UI if file profile not found
		if profile.ID == "" && fallbackProfile != nil {
			profile = *fallbackProfile
		}
		if profile.ID == "" {
			failedCount++
			failures = append(failures, failRow{Line: rowIdx + 2, Username: username, Error: fmt.Sprintf("profile '%s' tidak ditemukan — pilih Profile Default di dialog import", profileName)})
			continue
		}

		// Router lookup (case-insensitive; optional)
		routerName := col(row, "routername")
		var router models.Router
		routerID := (*string)(nil)
		if routerName != "" {
			if err2 := h.db.Where("LOWER(name) = LOWER(?)", routerName).First(&router).Error; err2 == nil {
				routerID = &router.ID
			}
		}

		// Area lookup (case-insensitive; optional)
		areaNameStr := col(row, "areaname")
		var area models.PppoeArea
		areaID := (*string)(nil)
		if areaNameStr != "" {
			if err2 := h.db.Where("LOWER(name) = LOWER(?)", areaNameStr).First(&area).Error; err2 == nil {
				areaID = &area.ID
			}
		}

		// Subscription type
		subType := models.Postpaid
		if strings.ToUpper(col(row, "subscriptiontype")) == "PREPAID" {
			subType = models.Prepaid
		}

		emailStr := col(row, "email")
		addrStr := col(row, "address")
		ipStr := col(row, "ipaddress")
		latStr := col(row, "latitude")
		lngStr := col(row, "longitude")
		macStr := col(row, "macaddress")
		billingDayStr := col(row, "billingday")
		commentStr := col(row, "comment")
		expiredAtStr := col(row, "expiredat")

		user := models.PppoeUser{
			ID:               generateID(),
			Username:         username,
			Password:         password,
			Name:             name,
			Phone:            phone,
			Status:           "active",
			ProfileID:        profile.ID,
			RouterID:         routerID,
			AreaID:           areaID,
			SubscriptionType: subType,
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
		if macStr != "" {
			user.MACAddress = &macStr
		}
		if commentStr != "" {
			user.Comment = &commentStr
		}
		if latStr != "" {
			if lat, err2 := strconv.ParseFloat(latStr, 64); err2 == nil {
				user.Latitude = &lat
			}
		}
		if lngStr != "" {
			if lng, err2 := strconv.ParseFloat(lngStr, 64); err2 == nil {
				user.Longitude = &lng
			}
		}
		if billingDayStr != "" {
			if bd, err2 := strconv.Atoi(billingDayStr); err2 == nil && bd >= 1 && bd <= 31 {
				user.BillingDay = &bd
			}
		}
		if expiredAtStr != "" {
			if t2, err2 := time.Parse("2006-01-02", expiredAtStr); err2 == nil {
				user.ExpiredAt = &t2
			}
		}

		if err2 := h.db.Create(&user).Error; err2 != nil {
			failedCount++
			failures = append(failures, failRow{Line: rowIdx + 2, Username: username, Error: err2.Error()})
		} else {
			successCount++
		}
	}

	// Cap errors returned to 20 to keep response payload small
	errorsToReturn := failures
	if len(errorsToReturn) > 20 {
		errorsToReturn = errorsToReturn[:20]
	}

	return c.JSON(fiber.Map{
		"success": true,
		"results": fiber.Map{
			"success": successCount,
			"failed":  failedCount,
			"errors":  errorsToReturn,
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

// POST /api/pppoe/users/bulk — bulk create users with RADIUS sync
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
			// Sync to FreeRADIUS radcheck
			rc := models.Radcheck{
				Username:  body[i].Username,
				Attribute: "Cleartext-Password",
				Op:        ":=",
				Value:     body[i].Password,
			}
			h.db.Where("username = ? AND attribute = ?", body[i].Username, "Cleartext-Password").
				Assign(rc).FirstOrCreate(&rc)
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

// PUT /api/pppoe/profiles/sync-mikrotik — test koneksi ke MikroTik router
func (h *PppoeExtHandler) TestMikrotikConnection(c fiber.Ctx) error {
	var body struct {
		RouterID string `json:"routerId"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.RouterID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "routerId wajib diisi"})
	}

	var router models.Router
	if err := h.db.First(&router, "id = ?", body.RouterID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Router tidak ditemukan"})
	}

	// Decrypt password
	routerPass := decryptVPNPassword(router.Password)

	type portResult struct {
		Port          int    `json:"port"`
		Success       bool   `json:"success"`
		Identity      string `json:"identity"`
		Error         string `json:"error,omitempty"`
		PPPRead       bool   `json:"pppRead"`
		PPPWrite      bool   `json:"pppWrite"`
		PPPReadError  string `json:"pppReadError,omitempty"`
		PPPWriteError string `json:"pppWriteError,omitempty"`
	}

	tryPort := func(port int) portResult {
		addr := fmt.Sprintf("%s:%d", router.IPAddress, port)
		client, err := ros.DialTimeout(addr, router.Username, routerPass, 8*time.Second)
		if err != nil {
			return portResult{Port: port, Success: false, Error: err.Error()}
		}
		defer client.Close()

		identity := ""
		if reply, err := client.Run("/system/identity/print"); err == nil && len(reply.Re) > 0 {
			identity = reply.Re[0].Map["name"]
		}

		// Test PPP profile read
		pppRead := false
		pppReadErr := ""
		if _, err := client.Run("/ppp/profile/print"); err != nil {
			pppReadErr = err.Error()
		} else {
			pppRead = true
			pppReadErr = "OK"
		}

		// Test PPP profile write (add temporary test profile then remove)
		pppWrite := false
		pppWriteErr := ""
		testName := "salfanet-test-tmp"
		if _, err := client.Run("/ppp/profile/add", "=name="+testName); err != nil {
			pppWriteErr = err.Error()
		} else {
			pppWrite = true
			// cleanup
			if rep, err := client.Run("/ppp/profile/print", "?name="+testName); err == nil && len(rep.Re) > 0 {
				_, _ = client.Run("/ppp/profile/remove", "=.id="+rep.Re[0].Map[".id"])
			}
		}

		return portResult{Port: port, Success: true, Identity: identity, PPPRead: pppRead, PPPReadError: pppReadErr, PPPWrite: pppWrite, PPPWriteError: pppWriteErr}
	}

	// Try non-SSL port first (8728), fallback to apiPort
	ports := []int{router.Port, router.APIPort}
	if router.Port == 0 {
		ports[0] = 8728
	}
	if router.APIPort == 0 {
		ports[1] = 8729
	}

	results := make([]portResult, 0, len(ports))
	for _, p := range ports {
		r := tryPort(p)
		results = append(results, r)
		if r.Success {
			break
		}
	}

	hint := ""
	ok := len(results) > 0 && results[len(results)-1].Success
	if !ok {
		hint = "Pastikan API port MikroTik (8728/8729) dapat diakses dari server VPS dan kredensial admin benar."
	}

	return c.JSON(fiber.Map{
		"success":    ok,
		"routerName": router.Name,
		"user":       router.Username,
		"host":       router.IPAddress,
		"results":    results,
		"hint":       hint,
	})
}

// POST /api/pppoe/profiles/sync-mikrotik — sync PPPoE profile ke MikroTik router(s)
func (h *PppoeExtHandler) SyncProfilesMikrotik(c fiber.Ctx) error {
	var body struct {
		ID           string   `json:"id"`
		RouterIDs    []string `json:"routerIds"`
		IPPoolName   string   `json:"ipPoolName"`
		LocalAddress string   `json:"localAddress"`
		PoolRanges   string   `json:"poolRanges"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.ID == "" || len(body.RouterIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "id dan routerIds wajib diisi"})
	}

	var profile models.PppoeProfile
	if err := h.db.First(&profile, "id = ?", body.ID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Profile tidak ditemukan"})
	}

	// Build rate limit string
	rateLimit := ""
	if profile.RateLimit != nil && *profile.RateLimit != "" {
		rateLimit = *profile.RateLimit
	} else {
		rateLimit = fmt.Sprintf("%dM/%dM", profile.UploadSpeed, profile.DownloadSpeed)
	}

	debugLines := []string{}
	successCount := 0

	for _, routerID := range body.RouterIDs {
		var router models.Router
		if err := h.db.First(&router, "id = ?", routerID).Error; err != nil {
			debugLines = append(debugLines, fmt.Sprintf("[%s] router tidak ditemukan di database", routerID))
			continue
		}

		pass := decryptVPNPassword(router.Password)
		port := router.Port
		if port == 0 {
			port = 8728
		}
		addr := fmt.Sprintf("%s:%d", router.IPAddress, port)
		client, err := ros.DialTimeout(addr, router.Username, pass, 10*time.Second)
		if err != nil {
			// fallback ke apiPort
			apiPort := router.APIPort
			if apiPort == 0 {
				apiPort = 8729
			}
			addr2 := fmt.Sprintf("%s:%d", router.IPAddress, apiPort)
			client, err = ros.DialTimeout(addr2, router.Username, pass, 10*time.Second)
			if err != nil {
				debugLines = append(debugLines, fmt.Sprintf("[%s] gagal konek: %s", router.Name, err.Error()))
				continue
			}
		}

		// Buat / update IP pool jika poolRanges diisi
		if body.PoolRanges != "" && body.IPPoolName != "" {
			poolReply, _ := client.Run("/ip/pool/print", "?name="+body.IPPoolName)
			if len(poolReply.Re) > 0 {
				_, _ = client.Run("/ip/pool/set", "=.id="+poolReply.Re[0].Map[".id"], "=ranges="+body.PoolRanges)
				debugLines = append(debugLines, fmt.Sprintf("[%s] pool '%s' diupdate", router.Name, body.IPPoolName))
			} else {
				if _, perr := client.Run("/ip/pool/add", "=name="+body.IPPoolName, "=ranges="+body.PoolRanges); perr != nil {
					debugLines = append(debugLines, fmt.Sprintf("[%s] gagal buat pool: %s", router.Name, perr.Error()))
				} else {
					debugLines = append(debugLines, fmt.Sprintf("[%s] pool '%s' dibuat", router.Name, body.IPPoolName))
				}
			}
		}

		// Args tambahan untuk PPP profile
		extraArgs := []string{"=rate-limit=" + rateLimit}
		if body.IPPoolName != "" {
			extraArgs = append(extraArgs, "=remote-address="+body.IPPoolName)
		}
		if body.LocalAddress != "" {
			extraArgs = append(extraArgs, "=local-address="+body.LocalAddress)
		}

		// Cek apakah PPP profile sudah ada
		profReply, _ := client.Run("/ppp/profile/print", "?name="+profile.GroupName)
		if len(profReply.Re) > 0 {
			// Update existing
			setArgs := append([]string{"/ppp/profile/set", "=.id=" + profReply.Re[0].Map[".id"]}, extraArgs...)
			if _, serr := client.Run(setArgs...); serr != nil {
				debugLines = append(debugLines, fmt.Sprintf("[%s] gagal update PPP profile: %s", router.Name, serr.Error()))
			} else {
				debugLines = append(debugLines, fmt.Sprintf("[%s] PPP profile '%s' diupdate ✓", router.Name, profile.GroupName))
				successCount++
			}
		} else {
			// Buat baru
			addArgs := append([]string{"/ppp/profile/add", "=name=" + profile.GroupName}, extraArgs...)
			if _, aerr := client.Run(addArgs...); aerr != nil {
				debugLines = append(debugLines, fmt.Sprintf("[%s] gagal buat PPP profile: %s", router.Name, aerr.Error()))
			} else {
				debugLines = append(debugLines, fmt.Sprintf("[%s] PPP profile '%s' dibuat ✓", router.Name, profile.GroupName))
				successCount++
			}
		}
		client.Close()
	}

	// Simpan data pool/localAddress + rateLimit ke profile di DB
	updates := map[string]interface{}{"rateLimit": rateLimit}
	if body.IPPoolName != "" {
		updates["ipPoolName"] = body.IPPoolName
	}
	if body.PoolRanges != "" {
		updates["ipPoolRange"] = body.PoolRanges
	}
	if body.LocalAddress != "" {
		updates["localAddress"] = body.LocalAddress
	}
	h.db.Model(&profile).Updates(updates)
	h.db.First(&profile, "id = ?", body.ID) // reload

	success := successCount > 0
	msg := fmt.Sprintf("Berhasil sync ke %d dari %d router", successCount, len(body.RouterIDs))
	if !success {
		msg = "Gagal sync ke semua router"
	}

	return c.JSON(fiber.Map{
		"success": success,
		"message": msg,
		"debug":   debugLines,
		"savedProfile": fiber.Map{
			"ipPoolName":   profile.IPPoolName,
			"ipPoolRange":  profile.IPPoolRange,
			"localAddress": profile.LocalAddress,
		},
	})
}

// POST /api/pppoe/profiles/sync-radius — sync profiles rate limits to FreeRADIUS radgroupreply
func (h *PppoeExtHandler) SyncProfilesRadius(c fiber.Ctx) error {
	// Support syncing a single profile by ID (from action button)
	var reqBody struct {
		ID string `json:"id"`
	}
	_ = c.Bind().JSON(&reqBody)

	var profiles []models.PppoeProfile
	q := h.db.Where("isActive = ?", true)
	if reqBody.ID != "" {
		q = q.Where("id = ?", reqBody.ID)
	}
	q.Find(&profiles)

	synced := 0
	for _, p := range profiles {
		if p.GroupName == "" {
			continue
		}
		// Build rate limit string: prefer stored rateLimit, fallback to speed fields
		rateLimit := ""
		if p.RateLimit != nil && *p.RateLimit != "" {
			rateLimit = *p.RateLimit
		} else if p.DownloadSpeed > 0 && p.UploadSpeed > 0 {
			rateLimit = fmt.Sprintf("%dM/%dM", p.UploadSpeed, p.DownloadSpeed)
		}
		if rateLimit == "" {
			continue
		}
		// Delete existing then insert fresh (no UNIQUE constraint on radgroupreply)
		h.db.Exec(`DELETE FROM radgroupreply WHERE groupname = ? AND attribute = 'Mikrotik-Rate-Limit'`, p.GroupName)
		h.db.Exec(`INSERT INTO radgroupreply (groupname, attribute, op, value) VALUES (?, 'Mikrotik-Rate-Limit', ':=', ?)`, p.GroupName, rateLimit)
		// Update syncedToRadius flag in pppoe_profiles table
		h.db.Model(&p).Update("syncedToRadius", true)
		synced++
	}
	return c.JSON(fiber.Map{"success": true, "synced": synced, "message": fmt.Sprintf("synced %d profiles to FreeRADIUS radgroupreply", synced)})
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
