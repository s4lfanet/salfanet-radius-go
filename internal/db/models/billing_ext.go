package models

import "time"

// ─── PackageChangeLog ────────────────────────────────────────────────────────

type PackageChangeLog struct {
	ID             string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	UserID         string    `gorm:"index;column:userId" json:"userId"`
	Username       string    `gorm:"column:username" json:"username"`
	OldProfileID   *string   `gorm:"column:oldProfileId" json:"oldProfileId"`
	OldProfileName *string   `gorm:"column:oldProfileName" json:"oldProfileName"`
	NewProfileID   *string   `gorm:"column:newProfileId" json:"newProfileId"`
	NewProfileName *string   `gorm:"column:newProfileName" json:"newProfileName"`
	ChangedBy      string    `gorm:"column:changedBy" json:"changedBy"`
	ChangedByName  *string   `gorm:"column:changedByName" json:"changedByName"`
	Reason         *string   `gorm:"type:text" json:"reason"`
	ChangedAt      time.Time `gorm:"column:changedAt;autoCreateTime" json:"changedAt"`
}

func (PackageChangeLog) TableName() string { return "package_change_logs" }

// ─── InstallationLog ─────────────────────────────────────────────────────────

type InstallationLog struct {
	ID             string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	UserID         string    `gorm:"index;column:userId" json:"userId"`
	Username       string    `gorm:"column:username" json:"username"`
	CustomerID     *string   `gorm:"column:customerId" json:"customerId"`
	Fullname       *string   `gorm:"column:fullname" json:"fullname"`
	Phone          *string   `gorm:"column:phone" json:"phone"`
	Address        *string   `gorm:"type:text" json:"address"`
	IdentityNumber *string   `gorm:"column:identityNumber" json:"identityNumber"`
	ProfileName    *string   `gorm:"column:profileName" json:"profileName"`
	TerritoryName  *string   `gorm:"column:territoryName" json:"territoryName"`
	InstallerID    string    `gorm:"index;column:installerId" json:"installerId"`
	InstallerName  *string   `gorm:"column:installerName" json:"installerName"`
	InstallDate    time.Time `gorm:"index;column:installDate" json:"installDate"`
	Latitude       *float64  `gorm:"column:latitude" json:"latitude"`
	Longitude      *float64  `gorm:"column:longitude" json:"longitude"`
	CreatedAt      time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
}

func (InstallationLog) TableName() string { return "installation_logs" }
