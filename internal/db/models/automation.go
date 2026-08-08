package models

import "time"

// NotificationTemplate stores per-event per-channel message templates.
type NotificationTemplate struct {
	ID        string    `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	EventType string    `gorm:"not null;column:eventType" json:"eventType"`
	Channel   string    `gorm:"not null;column:channel" json:"channel"` // wa, email, push, telegram, portal
	Template  string    `gorm:"type:text;not null;column:template" json:"template"`
	IsEnabled bool      `gorm:"default:true;column:isEnabled" json:"isEnabled"`
	CreatedAt time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;column:updatedAt" json:"updatedAt"`
}

func (NotificationTemplate) TableName() string { return "notification_templates" }

// ProvisioningStatus tracks the auto-provisioning pipeline progress per user.
type ProvisioningStatus struct {
	ID          string     `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	UserID      string     `gorm:"not null;index;column:userId" json:"userId"`
	Step        string     `gorm:"not null;column:step" json:"step"` // create_secret, configure_onu, generate_invoice, send_welcome
	Status      string     `gorm:"default:pending;column:status" json:"status"` // pending, running, success, failed
	Error       *string    `gorm:"type:text;column:error" json:"error"`
	StartedAt   *time.Time `gorm:"column:startedAt" json:"startedAt"`
	CompletedAt *time.Time `gorm:"column:completedAt" json:"completedAt"`
	CreatedAt   time.Time  `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
}

func (ProvisioningStatus) TableName() string { return "provisioning_status" }

// AlertRule defines a rule-based alert condition and actions.
type AlertRule struct {
	ID            string    `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	Name          string    `gorm:"not null;column:name" json:"name"`
	TriggerEvent  string    `gorm:"not null;column:triggerEvent" json:"triggerEvent"`
	Conditions    string    `gorm:"type:json;not null;column:conditions" json:"conditions"`
	Actions       string    `gorm:"type:json;not null;column:actions" json:"actions"`
	IsEnabled     bool      `gorm:"default:true;column:isEnabled" json:"isEnabled"`
	Priority      int       `gorm:"default:0;column:priority" json:"priority"`
	CreatedAt     time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime;column:updatedAt" json:"updatedAt"`
}

func (AlertRule) TableName() string { return "alert_rules" }

// PaymentPromise tracks customer payment promises to skip isolation.
type PaymentPromise struct {
	ID             string    `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	UserID         string    `gorm:"not null;index;column:userId" json:"userId"`
	Username       string    `gorm:"not null;column:username" json:"username"`
	PromiseDate    time.Time `gorm:"not null;column:promiseDate" json:"promiseDate"`
	Status         string    `gorm:"default:active;column:status" json:"status"` // active, fulfilled, broken
	CreatedBy      string    `gorm:"not null;column:createdBy" json:"createdBy"`
	CreatedByName  *string   `gorm:"column:createdByName" json:"createdByName"`
	Notes          *string   `gorm:"type:text;column:notes" json:"notes"`
	CreatedAt      time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
}

func (PaymentPromise) TableName() string { return "payment_promises" }
