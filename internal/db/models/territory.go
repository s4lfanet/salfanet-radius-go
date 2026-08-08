package models

import "time"

// ─── Territory ───────────────────────────────────────────────────────────────

type Territory struct {
	ID          string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name        string     `gorm:"uniqueIndex;not null" json:"name"`
	Description *string    `json:"description"`
	CollectorID *string    `gorm:"index;column:collectorId" json:"collectorId"`
	IsActive    bool       `gorm:"default:true" json:"isActive"`
	CreatedAt   time.Time  `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`

	Collector   *User            `gorm:"foreignKey:CollectorID" json:"collector,omitempty"`
	Areas       []TerritoryArea  `gorm:"foreignKey:TerritoryID" json:"areas,omitempty"`
}

func (Territory) TableName() string { return "territories" }

// ─── TerritoryArea ───────────────────────────────────────────────────────────

type TerritoryArea struct {
	ID             string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	TerritoryID    string    `gorm:"index;column:territoryId" json:"territoryId"`
	KelurahanKode  *string   `gorm:"column:kelurahanKode;index" json:"kelurahanKode"`
	KelurahanNama  *string   `gorm:"column:kelurahanNama" json:"kelurahanNama"`
	KecamatanNama  *string   `gorm:"column:kecamatanNama" json:"kecamatanNama"`
	KabupatenNama  *string   `gorm:"column:kabupatenNama" json:"kabupatenNama"`
	ProvinsiNama   *string   `gorm:"column:provinsiNama" json:"provinsiNama"`
	DusunNama      *string   `gorm:"column:dusunNama" json:"dusunNama"`
	CollectorID    *string   `gorm:"index;column:collectorId" json:"collectorId"`
	CreatedAt      time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`

	Territory *Territory `gorm:"foreignKey:TerritoryID" json:"territory,omitempty"`
}

func (TerritoryArea) TableName() string { return "territory_areas" }

// ─── Settlement ──────────────────────────────────────────────────────────────

type Settlement struct {
	ID           string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	CollectorID  string     `gorm:"index;column:collectorId" json:"collectorId"`
	PeriodDate   time.Time  `gorm:"index;column:periodDate" json:"periodDate"`
	TotalAmount  int        `gorm:"column:totalAmount" json:"totalAmount"`
	InvoiceCount int        `gorm:"column:invoiceCount" json:"invoiceCount"`
	Status       string     `gorm:"default:pending;index" json:"status"`
	ConfirmedBy  *string    `gorm:"column:confirmedBy" json:"confirmedBy"`
	ConfirmedAt  *time.Time `gorm:"column:confirmedAt" json:"confirmedAt"`
	CreatedAt    time.Time  `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`

	Collector *User `gorm:"foreignKey:CollectorID" json:"collector,omitempty"`
}

func (Settlement) TableName() string { return "settlements" }
