package models

import "time"

// APIKey stores hashed API keys for external integrations.
type APIKey struct {
	ID         string     `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	KeyHash    string     `gorm:"not null;column:keyHash" json:"-"`
	Label      string     `gorm:"not null;column:label" json:"label"`
	IsActive   bool       `gorm:"default:true;column:isActive" json:"isActive"`
	CreatedAt  time.Time  `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	LastUsedAt *time.Time `gorm:"column:lastUsedAt" json:"lastUsedAt"`
}

func (APIKey) TableName() string { return "api_keys" }

// ProfileRouterMap maps a PPPoE profile to a MikroTik profile per router.
type ProfileRouterMap struct {
	ID              string    `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	ProfileID       string    `gorm:"not null;column:profileId" json:"profileId"`
	RouterID        string    `gorm:"not null;column:routerId" json:"routerId"`
	MikrotikProfile string    `gorm:"not null;column:mikrotikProfile" json:"mikrotikProfile"`
	CreatedAt       time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
}

func (ProfileRouterMap) TableName() string { return "profile_router_map" }

// WaitingList entry for installation queue.
type WaitingList struct {
	ID              string     `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	Fullname        string     `gorm:"not null;column:fullname" json:"fullname"`
	Phone           *string    `gorm:"column:phone" json:"phone"`
	Address         *string    `gorm:"type:text;column:address" json:"address"`
	IdentityNumber  *string    `gorm:"column:identityNumber" json:"identityNumber"`
	KtpPhoto        *string    `gorm:"type:text;column:ktpPhoto" json:"ktpPhoto"`
	Notes           *string    `gorm:"type:text;column:notes" json:"notes"`
	TerritoryID     *string    `gorm:"column:territoryId" json:"territoryId"`
	TerritoryAreaID *string    `gorm:"column:territoryAreaId" json:"territoryAreaId"`
	KelurahanKode   *string    `gorm:"column:kelurahanKode" json:"kelurahanKode"`
	ProfileID       *string    `gorm:"column:profileId" json:"profileId"`
	Sales           *string    `gorm:"column:sales" json:"sales"`
	Latitude        *float64   `gorm:"type:decimal(10,7);column:latitude" json:"latitude"`
	Longitude       *float64   `gorm:"type:decimal(10,7);column:longitude" json:"longitude"`
	Status          string     `gorm:"default:waiting;column:status" json:"status"` // waiting, installed, cancelled
	CreatedBy       string     `gorm:"not null;column:createdBy" json:"createdBy"`
	CreatedAt       time.Time  `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime;column:updatedAt" json:"updatedAt"`
}

func (WaitingList) TableName() string { return "waiting_list" }

// WaitingListAssignment tracks technician assignments for waiting list entries.
type WaitingListAssignment struct {
	ID                 string    `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	WaitingListID      string    `gorm:"not null;column:waitingListId" json:"waitingListId"`
	TechnicianUsername string    `gorm:"not null;column:technicianUsername" json:"technicianUsername"`
	AssignedBy         string    `gorm:"not null;column:assignedBy" json:"assignedBy"`
	AssignedAt         time.Time `gorm:"autoCreateTime;column:assignedAt" json:"assignedAt"`
}

func (WaitingListAssignment) TableName() string { return "waiting_list_assignments" }

// OntRemovalTask tracks ONT/ONU removal tasks for stopped customers.
type OntRemovalTask struct {
	ID            string     `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	UserID        string     `gorm:"not null;column:userId" json:"userId"`
	Username      string     `gorm:"not null;column:username" json:"username"`
	CustomerID    *string    `gorm:"column:customerId" json:"customerId"`
	Fullname      *string    `gorm:"column:fullname" json:"fullname"`
	Address       *string    `gorm:"type:text;column:address" json:"address"`
	TerritoryName *string    `gorm:"column:territoryName" json:"territoryName"`
	Latitude      *float64   `gorm:"type:decimal(10,7);column:latitude" json:"latitude"`
	Longitude     *float64   `gorm:"type:decimal(10,7);column:longitude" json:"longitude"`
	AssignedTo    string     `gorm:"not null;column:assignedTo" json:"assignedTo"`
	AssignedBy    string     `gorm:"not null;column:assignedBy" json:"assignedBy"`
	Status        string     `gorm:"default:pending;column:status" json:"status"` // pending, done, confirmed, cancelled
	ProofPhoto    *string    `gorm:"type:text;column:proofPhoto" json:"proofPhoto"`
	Notes         *string    `gorm:"type:text;column:notes" json:"notes"`
	CancelReason  *string    `gorm:"type:text;column:cancelReason" json:"cancelReason"`
	CancelledBy   *string    `gorm:"column:cancelledBy" json:"cancelledBy"`
	CancelledAt   *time.Time `gorm:"column:cancelledAt" json:"cancelledAt"`
	ConfirmedBy   *string    `gorm:"column:confirmedBy" json:"confirmedBy"`
	ConfirmedAt   *time.Time `gorm:"column:confirmedAt" json:"confirmedAt"`
	CreatedAt     time.Time  `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
}

func (OntRemovalTask) TableName() string { return "ont_removal_tasks" }
