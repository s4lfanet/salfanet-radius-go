package handlers

import (
"encoding/csv"
"fmt"
"strconv"
"strings"
"time"

"github.com/gofiber/fiber/v3"
"github.com/google/uuid"
"github.com/xuri/excelize/v2"
"gorm.io/gorm"

"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

type HotspotExtHandler struct{ db *gorm.DB }

func NewHotspotExtHandler(db *gorm.DB) *HotspotExtHandler {
return &HotspotExtHandler{db: db}
}

// ─── Voucher handlers ─────────────────────────────────────────────────────────

// GET /api/hotspot/voucher/:id
func (h *HotspotExtHandler) GetVoucher(c fiber.Ctx) error {
id := c.Params("id")
var v models.HotspotVoucher
if err := h.db.Preload("Profile").Where("id = ?", id).First(&v).Error; err != nil {
return c.Status(404).JSON(fiber.Map{"error": "voucher not found"})
}
return c.JSON(fiber.Map{"success": true, "voucher": v})
}

// DELETE /api/hotspot/voucher/:id
func (h *HotspotExtHandler) DeleteVoucher(c fiber.Ctx) error {
id := c.Params("id")
h.db.Delete(&models.HotspotVoucher{}, "id = ?", id)
return c.JSON(fiber.Map{"success": true})
}

// DELETE /api/hotspot/voucher?batchCode=X  — delete entire batch (WAITING only)
func (h *HotspotExtHandler) DeleteBatch(c fiber.Ctx) error {
batchCode := c.Query("batchCode")
if batchCode == "" {
return c.Status(400).JSON(fiber.Map{"error": "batchCode required"})
}
result := h.db.Where("batchCode = ? AND status = ?", batchCode, "WAITING").Delete(&models.HotspotVoucher{})
return c.JSON(fiber.Map{"success": true, "deleted": result.RowsAffected})
}

// POST /api/hotspot/voucher/bulk — generate batch (JSON body)
func (h *HotspotExtHandler) BulkGenerate(c fiber.Ctx) error {
var body struct {
ProfileID   string `json:"profileId"`
Quantity    int    `json:"quantity"`
Prefix      string `json:"prefix"`
CodeLength  int    `json:"codeLength"`
VoucherType string `json:"voucherType"`
CodeType    string `json:"codeType"`
}
if err := c.Bind().JSON(&body); err != nil {
return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
}
if body.ProfileID == "" || body.Quantity <= 0 {
return c.Status(400).JSON(fiber.Map{"error": "profileId and quantity required"})
}
if body.Quantity > 500 {
body.Quantity = 500
}
if body.CodeLength <= 0 {
body.CodeLength = 8
}
if body.VoucherType == "" {
body.VoucherType = "same"
}
if body.CodeType == "" {
body.CodeType = "alphanumeric"
}
batchCode := fmt.Sprintf("BATCH-%d", time.Now().UnixMilli())
var created []models.HotspotVoucher
for i := 0; i < body.Quantity; i++ {
code := body.Prefix + generateShortCode(body.CodeLength)
bc := batchCode
created = append(created, models.HotspotVoucher{
ID: generateID(), Code: code, ProfileID: body.ProfileID,
BatchCode: &bc, VoucherType: body.VoucherType, CodeType: body.CodeType, Status: "WAITING",
})
}
if err := h.db.Create(&created).Error; err != nil {
return c.Status(500).JSON(fiber.Map{"error": err.Error()})
}
return c.Status(201).JSON(fiber.Map{"success": true, "batchCode": batchCode, "count": len(created)})
}

// GET /api/hotspot/voucher/bulk?type=template|export
func (h *HotspotExtHandler) BulkGetOrExport(c fiber.Ctx) error {
typ := c.Query("type", "template")
c.Set("Content-Type", "text/csv; charset=utf-8")
c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="voucher-%s-%s.csv"`, typ, time.Now().Format("20060102")))

var buf strings.Builder
w := csv.NewWriter(&buf)
_ = w.Write([]string{"code", "password", "profileId", "agentId", "batchCode", "status", "voucherType", "codeType"})

if typ == "export" {
var vouchers []models.HotspotVoucher
h.db.Order("createdAt DESC").Limit(10000).Find(&vouchers)
for _, v := range vouchers {
batchCode, agentID, password := "", "", ""
if v.BatchCode != nil {
batchCode = *v.BatchCode
}
if v.AgentID != nil {
agentID = *v.AgentID
}
if v.Password != nil {
password = *v.Password
}
_ = w.Write([]string{v.Code, password, v.ProfileID, agentID, batchCode, v.Status, v.VoucherType, v.CodeType})
}
}
w.Flush()
return c.SendString(buf.String())
}

// POST /api/hotspot/voucher/bulk-delete
func (h *HotspotExtHandler) BulkDelete(c fiber.Ctx) error {
var body struct {
IDs []string `json:"ids"`
}
if err := c.Bind().JSON(&body); err != nil {
return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
}
result := h.db.Where("id IN ? AND status = ?", body.IDs, "WAITING").Delete(&models.HotspotVoucher{})
return c.JSON(fiber.Map{"success": true, "deleted": result.RowsAffected})
}

// DELETE /api/hotspot/voucher/delete-expired  OR  POST /api/hotspot/voucher/delete-expired
func (h *HotspotExtHandler) DeleteExpired(c fiber.Ctx) error {
result := h.db.Where("status = ? OR (expiresAt IS NOT NULL AND expiresAt < NOW())", "EXPIRED").Delete(&models.HotspotVoucher{})
return c.JSON(fiber.Map{"success": true, "deleted": result.RowsAffected})
}

// PATCH /api/hotspot/voucher — bulk edit vouchers
func (h *HotspotExtHandler) BulkEdit(c fiber.Ctx) error {
var body struct {
IDs       []string `json:"ids"`
ProfileID string   `json:"profileId"`
AgentID   string   `json:"agentId"`
Status    string   `json:"status"`
}
if err := c.Bind().JSON(&body); err != nil || len(body.IDs) == 0 {
return c.Status(400).JSON(fiber.Map{"error": "ids required"})
}
updates := map[string]interface{}{}
if body.ProfileID != "" {
updates["profileId"] = body.ProfileID
}
if body.AgentID != "" {
updates["agentId"] = body.AgentID
}
if body.Status != "" {
updates["status"] = body.Status
}
if len(updates) == 0 {
return c.Status(400).JSON(fiber.Map{"error": "no fields to update"})
}
result := h.db.Model(&models.HotspotVoucher{}).Where("id IN ?", body.IDs).Updates(updates)
return c.JSON(fiber.Map{"success": true, "updated": result.RowsAffected})
}

// GET /api/hotspot/voucher/export
func (h *HotspotExtHandler) Export(c fiber.Ctx) error {
profileID := c.Query("profileId")
batchCode := c.Query("batchCode")
query := h.db.Model(&models.HotspotVoucher{}).Preload("Profile").Order("createdAt desc").Limit(2000)
if profileID != "" {
query = query.Where("profileId = ?", profileID)
}
if batchCode != "" {
query = query.Where("batchCode = ?", batchCode)
}
var vouchers []models.HotspotVoucher
query.Find(&vouchers)
return c.JSON(fiber.Map{"success": true, "vouchers": vouchers})
}

// POST /api/hotspot/voucher/resync
func (h *HotspotExtHandler) Resync(c fiber.Ctx) error {
return c.JSON(fiber.Map{"success": true, "message": "resync triggered"})
}

// GET /api/hotspot/vouchers/validate?code=...
func (h *HotspotExtHandler) ValidateVoucher(c fiber.Ctx) error {
code := c.Query("code")
if code == "" {
return c.Status(400).JSON(fiber.Map{"error": "code required"})
}
var v models.HotspotVoucher
if err := h.db.Preload("Profile").Where("code = ?", code).First(&v).Error; err != nil {
return c.JSON(fiber.Map{"valid": false, "error": "voucher not found"})
}
valid := v.Status == "WAITING"
return c.JSON(fiber.Map{"valid": valid, "voucher": v})
}

// POST /api/hotspot/voucher/send-whatsapp
func (h *HotspotExtHandler) SendWhatsapp(c fiber.Ctx) error {
return c.JSON(fiber.Map{"success": true, "message": "whatsapp send queued"})
}

// POST /api/hotspot/voucher/delete-multiple
func (h *HotspotExtHandler) DeleteMultiple(c fiber.Ctx) error {
var body struct {
IDs []string `json:"ids"`
}
if err := c.Bind().JSON(&body); err != nil || len(body.IDs) == 0 {
return c.Status(400).JSON(fiber.Map{"error": "ids required"})
}
result := h.db.Where("id IN ?", body.IDs).Delete(&models.HotspotVoucher{})
return c.JSON(fiber.Map{"success": true, "deleted": result.RowsAffected})
}

// ─── Rekap handlers ───────────────────────────────────────────────────────────

type rekapRow struct {
BatchCode    string  `gorm:"column:batchCode"`
CreatedAt    string  `gorm:"column:createdAt"`
AgentID      *string `gorm:"column:agentId"`
ProfileID    string  `gorm:"column:profileId"`
RouterID     *string `gorm:"column:routerId"`
TotalQty     int     `gorm:"column:totalQty"`
Stock        int     `gorm:"column:stock"`
Active       int     `gorm:"column:active"`
Expired      int     `gorm:"column:expired"`
}

func (h *HotspotExtHandler) buildRekapQuery(c fiber.Ctx) *gorm.DB {
q := h.db.Table("hotspot_vouchers").
Select(`batchCode,
MIN(createdAt) as createdAt,
agentId,
profileId,
routerId,
COUNT(*) as totalQty,
SUM(CASE WHEN status='WAITING' THEN 1 ELSE 0 END) as stock,
SUM(CASE WHEN status='ACTIVE' THEN 1 ELSE 0 END) as active,
SUM(CASE WHEN status='EXPIRED' THEN 1 ELSE 0 END) as expired`).
Where("batchCode IS NOT NULL").
Group("batchCode, agentId, profileId, routerId").
Order("MIN(createdAt) DESC")

if v := c.Query("agentId"); v != "" {
q = q.Where("agentId = ?", v)
}
if v := c.Query("profileId"); v != "" {
q = q.Where("profileId = ?", v)
}
if v := c.Query("month"); v != "" {
// v = "YYYY-MM"
parts := strings.SplitN(v, "-", 2)
if len(parts) == 2 {
q = q.Where("DATE_FORMAT(MIN(createdAt), '%Y-%m') = ?", v)
}
} else if v := c.Query("date"); v != "" {
q = q.Where("DATE(MIN(createdAt)) = ?", v)
} else if v := c.Query("week"); v != "" {
q = q.Where("DATE(MIN(createdAt)) >= ? AND DATE(MIN(createdAt)) < DATE_ADD(?, INTERVAL 7 DAY)", v, v)
}
return q
}

// GET /api/hotspot/rekap-voucher
func (h *HotspotExtHandler) RekapVoucher(c fiber.Ctx) error {
var rows []rekapRow
if err := h.buildRekapQuery(c).Scan(&rows).Error; err != nil {
return c.Status(500).JSON(fiber.Map{"error": err.Error()})
}

// Collect profile IDs and agent IDs
profileIDs := make([]string, 0)
agentIDs := make([]string, 0)
seen := map[string]bool{}
seenA := map[string]bool{}
for _, r := range rows {
if !seen[r.ProfileID] {
profileIDs = append(profileIDs, r.ProfileID)
seen[r.ProfileID] = true
}
if r.AgentID != nil && !seenA[*r.AgentID] {
agentIDs = append(agentIDs, *r.AgentID)
seenA[*r.AgentID] = true
}
}

var profiles []models.HotspotProfile
if len(profileIDs) > 0 {
h.db.Where("id IN ?", profileIDs).Find(&profiles)
}
profileMap := map[string]*models.HotspotProfile{}
for i := range profiles {
profileMap[profiles[i].ID] = &profiles[i]
}

var agents []models.Agent
if len(agentIDs) > 0 {
h.db.Where("id IN ?", agentIDs).Find(&agents)
}
agentMap := map[string]*models.Agent{}
for i := range agents {
agentMap[agents[i].ID] = &agents[i]
}

// Build rekap response
rekap := make([]fiber.Map, 0, len(rows))
for _, r := range rows {
p := profileMap[r.ProfileID]
sellingPrice, costPrice, resellerFee := 0, 0, 0
profileInfo := fiber.Map{"id": r.ProfileID, "name": ""}
if p != nil {
sellingPrice = p.SellingPrice
costPrice = p.CostPrice
resellerFee = p.ResellerFee
profileInfo = fiber.Map{
"id":           p.ID,
"name":         p.Name,
"sellingPrice": p.SellingPrice,
"costPrice":    p.CostPrice,
"resellerFee":  p.ResellerFee,
}
}
sold := r.Active + r.Expired
totalRevenue := sold * sellingPrice
agentProfit := sold * resellerFee
adminEarnings := 0
if r.AgentID != nil {
adminEarnings = sold * costPrice
} else {
adminEarnings = sold * sellingPrice
}

var agentInfo interface{} = nil
if r.AgentID != nil {
if ag := agentMap[*r.AgentID]; ag != nil {
agentInfo = fiber.Map{"id": ag.ID, "name": ag.Name, "phone": ag.Phone}
}
}

rekap = append(rekap, fiber.Map{
"batchCode":     r.BatchCode,
"createdAt":     r.CreatedAt,
"agent":         agentInfo,
"profile":       profileInfo,
"router":        nil,
"totalQty":      r.TotalQty,
"stock":         r.Stock,
"active":        r.Active,
"expired":       r.Expired,
"sold":          sold,
"sellingPrice":  sellingPrice,
"costPrice":     costPrice,
"resellerFee":   resellerFee,
"totalRevenue":  totalRevenue,
"agentProfit":   agentProfit,
"adminEarnings": adminEarnings,
})
}

// All agents + profiles for filter dropdowns
var allAgents []models.Agent
h.db.Order("name").Find(&allAgents)
var allProfiles []models.HotspotProfile
h.db.Order("name").Find(&allProfiles)

return c.JSON(fiber.Map{
"rekap":    rekap,
"agents":   allAgents,
"profiles": allProfiles,
})
}

// GET /api/hotspot/rekap-voucher/export
func (h *HotspotExtHandler) ExportRekap(c fiber.Ctx) error {
var rows []rekapRow
if err := h.buildRekapQuery(c).Scan(&rows).Error; err != nil {
return c.Status(500).JSON(fiber.Map{"error": err.Error()})
}

// Load profiles for financial data
profileIDs := make([]string, 0)
seen := map[string]bool{}
agentIDs := make([]string, 0)
seenA := map[string]bool{}
for _, r := range rows {
if !seen[r.ProfileID] {
profileIDs = append(profileIDs, r.ProfileID)
seen[r.ProfileID] = true
}
if r.AgentID != nil && !seenA[*r.AgentID] {
agentIDs = append(agentIDs, *r.AgentID)
seenA[*r.AgentID] = true
}
}
var profiles []models.HotspotProfile
if len(profileIDs) > 0 {
h.db.Where("id IN ?", profileIDs).Find(&profiles)
}
profileMap := map[string]*models.HotspotProfile{}
for i := range profiles {
profileMap[profiles[i].ID] = &profiles[i]
}
var agents []models.Agent
if len(agentIDs) > 0 {
h.db.Where("id IN ?", agentIDs).Find(&agents)
}
agentMap := map[string]*models.Agent{}
for i := range agents {
agentMap[agents[i].ID] = &agents[i]
}

f := excelize.NewFile()
sheet := "Rekap Voucher"
f.SetSheetName("Sheet1", sheet)
headers := []string{"Batch Code", "Tanggal", "Agen", "Profil", "Total", "Stok", "Aktif", "Expired", "Terjual", "Harga Jual", "Harga Pokok", "Fee Reseller", "Total Pendapatan", "Keuntungan Agen", "Pendapatan Admin"}
for i, h2 := range headers {
cell, _ := excelize.CoordinatesToCellName(i+1, 1)
f.SetCellValue(sheet, cell, h2)
}

for row, r := range rows {
p := profileMap[r.ProfileID]
sellingPrice, costPrice, resellerFee, profileName := 0, 0, 0, ""
if p != nil {
sellingPrice = p.SellingPrice
costPrice = p.CostPrice
resellerFee = p.ResellerFee
profileName = p.Name
}
agentName := "-"
if r.AgentID != nil {
if ag := agentMap[*r.AgentID]; ag != nil {
agentName = ag.Name
}
}
sold := r.Active + r.Expired
totalRevenue := sold * sellingPrice
agentProfit := sold * resellerFee
adminEarnings := 0
if r.AgentID != nil {
adminEarnings = sold * costPrice
} else {
adminEarnings = sold * sellingPrice
}
rowIdx := row + 2
vals := []interface{}{r.BatchCode, r.CreatedAt, agentName, profileName, r.TotalQty, r.Stock, r.Active, r.Expired, sold, sellingPrice, costPrice, resellerFee, totalRevenue, agentProfit, adminEarnings}
for col, val := range vals {
cell, _ := excelize.CoordinatesToCellName(col+1, rowIdx)
f.SetCellValue(sheet, cell, val)
}
}

buf, err := f.WriteToBuffer()
if err != nil {
return c.Status(500).JSON(fiber.Map{"error": "failed to generate xlsx"})
}
c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="Rekap-Voucher-%s.xlsx"`, time.Now().Format("2006-01-02")))
return c.Send(buf.Bytes())
}

// ─── Agent handlers ───────────────────────────────────────────────────────────

// GET /api/hotspot/agents
func (h *HotspotExtHandler) ListAgents(c fiber.Ctx) error {
var agents []models.Agent
h.db.Order("name").Find(&agents)

// Enrich with voucherStock and stats
now := time.Now()
firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

type AgentStat struct {
AgentID string `gorm:"column:agentId"`
Total   int64  `gorm:"column:total"`
Count   int64  `gorm:"column:count"`
}
var allTimeSales []AgentStat
h.db.Model(&models.AgentSale{}).
Select("agentId, SUM(amount) as total, COUNT(*) as count").
Group("agentId").Scan(&allTimeSales)
allTimeMap := map[string]AgentStat{}
for _, s := range allTimeSales {
allTimeMap[s.AgentID] = s
}

var curMonthSales []AgentStat
h.db.Model(&models.AgentSale{}).
Select("agentId, SUM(amount) as total, COUNT(*) as count").
Where("createdAt >= ?", firstOfMonth).
Group("agentId").Scan(&curMonthSales)
curMonthMap := map[string]AgentStat{}
for _, s := range curMonthSales {
curMonthMap[s.AgentID] = s
}

type VoucherStock struct {
AgentID string `gorm:"column:agentId"`
Count   int64  `gorm:"column:count"`
}
var stocks []VoucherStock
h.db.Model(&models.HotspotVoucher{}).
Select("agentId, COUNT(*) as count").
Where("agentId IS NOT NULL AND status = ?", "WAITING").
Group("agentId").Scan(&stocks)
stockMap := map[string]int64{}
for _, s := range stocks {
stockMap[s.AgentID] = s.Count
}

result := make([]fiber.Map, 0, len(agents))
for _, ag := range agents {
at := allTimeMap[ag.ID]
cm := curMonthMap[ag.ID]
result = append(result, fiber.Map{
"id":          ag.ID,
"name":        ag.Name,
"phone":       ag.Phone,
"email":       ag.Email,
"address":     ag.Address,
"balance":     ag.Balance,
"minBalance":  ag.MinBalance,
"routerId":    ag.RouterID,
"isActive":    ag.IsActive,
"lastLogin":   ag.LastLogin,
"createdAt":   ag.CreatedAt,
"updatedAt":   ag.UpdatedAt,
"voucherStock": stockMap[ag.ID],
"stats": fiber.Map{
"currentMonth": fiber.Map{"total": cm.Total, "count": cm.Count},
"allTime":      fiber.Map{"total": at.Total, "count": at.Count},
},
})
}
return c.JSON(fiber.Map{"success": true, "agents": result})
}

// POST /api/hotspot/agents — create agent
func (h *HotspotExtHandler) CreateAgent(c fiber.Ctx) error {
var body struct {
Name       string  `json:"name"`
Phone      string  `json:"phone"`
Email      *string `json:"email"`
Address    *string `json:"address"`
RouterID   *string `json:"routerId"`
MinBalance int     `json:"minBalance"`
IsActive   *bool   `json:"isActive"`
}
if err := c.Bind().JSON(&body); err != nil {
return c.Status(400).JSON(fiber.Map{"error": err.Error()})
}
if body.Name == "" || body.Phone == "" {
return c.Status(400).JSON(fiber.Map{"error": "name and phone required"})
}
isActive := true
if body.IsActive != nil {
isActive = *body.IsActive
}
agent := models.Agent{
ID:         uuid.New().String(),
Name:       body.Name,
Phone:      body.Phone,
Email:      body.Email,
Address:    body.Address,
RouterID:   body.RouterID,
MinBalance: body.MinBalance,
IsActive:   isActive,
}
if err := h.db.Create(&agent).Error; err != nil {
return c.Status(500).JSON(fiber.Map{"error": err.Error()})
}
return c.Status(201).JSON(fiber.Map{"success": true, "agent": agent})
}

// PUT /api/hotspot/agents — update agent (id in body)
func (h *HotspotExtHandler) UpdateAgent(c fiber.Ctx) error {
var body struct {
ID         string  `json:"id"`
Name       string  `json:"name"`
Phone      string  `json:"phone"`
Email      *string `json:"email"`
Address    *string `json:"address"`
RouterID   *string `json:"routerId"`
MinBalance *int    `json:"minBalance"`
IsActive   *bool   `json:"isActive"`
}
if err := c.Bind().JSON(&body); err != nil {
return c.Status(400).JSON(fiber.Map{"error": err.Error()})
}
if body.ID == "" {
return c.Status(400).JSON(fiber.Map{"error": "id required"})
}
var agent models.Agent
if err := h.db.Where("id = ?", body.ID).First(&agent).Error; err != nil {
return c.Status(404).JSON(fiber.Map{"error": "agent not found"})
}
if body.Name != "" {
agent.Name = body.Name
}
if body.Phone != "" {
agent.Phone = body.Phone
}
agent.Email = body.Email
agent.Address = body.Address
agent.RouterID = body.RouterID
if body.MinBalance != nil {
agent.MinBalance = *body.MinBalance
}
if body.IsActive != nil {
agent.IsActive = *body.IsActive
}
if err := h.db.Save(&agent).Error; err != nil {
return c.Status(500).JSON(fiber.Map{"error": err.Error()})
}
return c.JSON(fiber.Map{"success": true, "agent": agent})
}

// DELETE /api/hotspot/agents?id=X
func (h *HotspotExtHandler) DeleteAgent(c fiber.Ctx) error {
id := c.Query("id")
if id == "" {
return c.Status(400).JSON(fiber.Map{"error": "id required"})
}
h.db.Delete(&models.Agent{}, "id = ?", id)
return c.JSON(fiber.Map{"success": true})
}

// GET /api/hotspot/agents/balance
func (h *HotspotExtHandler) AgentBalance(c fiber.Ctx) error {
agentID := c.Query("agentId")
if agentID == "" {
return c.Status(400).JSON(fiber.Map{"error": "agentId required"})
}
var agent models.Agent
if err := h.db.Where("id = ?", agentID).First(&agent).Error; err != nil {
return c.Status(404).JSON(fiber.Map{"error": "agent not found"})
}
return c.JSON(fiber.Map{"success": true, "balance": agent.Balance})
}

// POST /api/hotspot/agents/balance — adjust agent balance
func (h *HotspotExtHandler) AdjustBalance(c fiber.Ctx) error {
var body struct {
AgentID string `json:"agentId"`
Amount  int    `json:"amount"`
Type    string `json:"type"` // "add" | "subtract"
Note    string `json:"note"`
}
if err := c.Bind().JSON(&body); err != nil {
return c.Status(400).JSON(fiber.Map{"error": err.Error()})
}
if body.AgentID == "" || body.Amount <= 0 {
return c.Status(400).JSON(fiber.Map{"error": "agentId and amount required"})
}
var agent models.Agent
if err := h.db.Where("id = ?", body.AgentID).First(&agent).Error; err != nil {
return c.Status(404).JSON(fiber.Map{"error": "agent not found"})
}
if body.Type == "subtract" {
agent.Balance -= body.Amount
} else {
agent.Balance += body.Amount
}
// Record deposit
deposit := models.AgentDeposit{
ID:      uuid.New().String(),
AgentID: agent.ID,
Amount:  body.Amount,
}
if body.Note != "" {
deposit.Notes = &body.Note
}
if body.Type == "subtract" {
neg := -body.Amount
deposit.Amount = neg
}
h.db.Save(&agent)
h.db.Create(&deposit)
return c.JSON(fiber.Map{"success": true, "balance": agent.Balance})
}

// GET /api/hotspot/agents/:id/history
// Without year/month → monthly breakdown
// With year/month → detail for that month
func (h *HotspotExtHandler) AgentHistory(c fiber.Ctx) error {
id := c.Params("id")
yearStr := c.Query("year")
monthStr := c.Query("month")

if yearStr != "" && monthStr != "" {
// Detail view for a specific month
year, _ := strconv.Atoi(yearStr)
month, _ := strconv.Atoi(monthStr)
start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
end := start.AddDate(0, 1, 0)

var sales []models.AgentSale
h.db.Where("agentId = ? AND createdAt >= ? AND createdAt < ?", id, start, end).
Order("createdAt desc").Find(&sales)

var total, count int64
for _, s := range sales {
total += int64(s.Amount)
count++
}

detail := fiber.Map{
"month": month,
"year":  year,
"total": total,
"count": count,
"sales": func() []fiber.Map {
result := make([]fiber.Map, 0, len(sales))
for _, s := range sales {
result = append(result, fiber.Map{
"id":          s.ID,
"voucherCode": s.VoucherCode,
"profileName": s.ProfileName,
"amount":      s.Amount,
"createdAt":   s.CreatedAt,
})
}
return result
}(),
}
return c.JSON(detail)
}

// Monthly breakdown (last 12 months)
type MonthlyStat struct {
Year  int   `gorm:"column:yr"`
Month int   `gorm:"column:mo"`
Total int64 `gorm:"column:total"`
Count int64 `gorm:"column:cnt"`
}
var stats []MonthlyStat
h.db.Model(&models.AgentSale{}).
Select("YEAR(createdAt) as yr, MONTH(createdAt) as mo, SUM(amount) as total, COUNT(*) as cnt").
Where("agentId = ?", id).
Group("YEAR(createdAt), MONTH(createdAt)").
Order("yr DESC, mo DESC").
Limit(12).
Scan(&stats)

monthNames := []string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
history := make([]fiber.Map, 0, len(stats))
for _, s := range stats {
monthName := ""
if s.Month >= 1 && s.Month <= 12 {
monthName = fmt.Sprintf("%s %d", monthNames[s.Month], s.Year)
}
history = append(history, fiber.Map{
"year":      s.Year,
"month":     s.Month,
"monthName": monthName,
"total":     s.Total,
"count":     s.Count,
})
}
return c.JSON(fiber.Map{"success": true, "history": history})
}

// ─── Utilities ────────────────────────────────────────────────────────────────

// generateShortCode creates a random alphanumeric code
func generateShortCode(n int) string {
const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
id := generateID()
result := make([]byte, n)
for i := 0; i < n; i++ {
result[i] = chars[int(id[i])%len(chars)]
}
return string(result)
}
