package models

import "time"

// ─── Enums ───────────────────────────────────────────────────────────────────

type OltOnuStatus string

const (
	OnuOnline       OltOnuStatus = "online"
	OnuOffline      OltOnuStatus = "offline"
	OnuDyingGasp    OltOnuStatus = "dying_gasp"
	OnuLOS          OltOnuStatus = "los"
	OnuAuthFailed   OltOnuStatus = "auth_failed"
	OnuUnregistered OltOnuStatus = "unregistered"
)

type OltAlertType string

const (
	AlertOLTOffline      OltAlertType = "olt_offline"
	AlertOLTHighTemp     OltAlertType = "olt_high_temp"
	AlertONUOffline      OltAlertType = "onu_offline"
	AlertLowSignal       OltAlertType = "low_signal"
	AlertHighErrors      OltAlertType = "high_errors"
	AlertDyingGasp       OltAlertType = "dying_gasp"
	AlertUnauthorizedONU OltAlertType = "unauthorized_onu"
)

type OltAlertSeverity string

const (
	SeverityInfo     OltAlertSeverity = "info"
	SeverityWarning  OltAlertSeverity = "warning"
	SeverityCritical OltAlertSeverity = "critical"
)

// ─── NetworkOLT ──────────────────────────────────────────────────────────────

type NetworkOLT struct {
	ID         string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name       string    `gorm:"not null" json:"name"`
	IPAddress  string    `gorm:"not null;column:ipAddress" json:"ipAddress"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Status     string    `gorm:"default:active" json:"status"`
	FollowRoad bool      `gorm:"default:false;column:followRoad" json:"followRoad"`
	CreatedAt  time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`

	// Monitoring fields
	Vendor            *string    `gorm:"default:huawei" json:"vendor"`
	Model             *string    `json:"model"`
	FirmwareVersion   *string    `gorm:"column:firmwareVersion" json:"firmwareVersion"`
	SNMPEnabled       bool       `gorm:"default:true;column:snmpEnabled" json:"snmpEnabled"`
	SNMPCommunity     string     `gorm:"default:public;column:snmpCommunity" json:"snmpCommunity"`
	SNMPPort          int        `gorm:"default:161;column:snmpPort" json:"snmpPort"`
	TelnetEnabled     bool       `gorm:"default:false;column:telnetEnabled" json:"telnetEnabled"`
	TelnetPort        int        `gorm:"default:23;column:telnetPort" json:"telnetPort"`
	SSHEnabled        bool       `gorm:"default:false;column:sshEnabled" json:"sshEnabled"`
	SSHPort           int        `gorm:"default:22;column:sshPort" json:"sshPort"`
	Username          *string    `json:"username"`
	Password          *string    `json:"-"`
	MonitoringEnabled bool       `gorm:"default:false;column:monitoringEnabled" json:"monitoringEnabled"`
	PollingInterval   int        `gorm:"default:300;column:pollingInterval" json:"pollingInterval"`
	LastPollAt        *time.Time `gorm:"column:lastPollAt" json:"lastPollAt"`
	IsOnline          bool       `gorm:"default:false;column:isOnline" json:"isOnline"`
	Uptime            int64      `gorm:"default:0" json:"uptime"`
	Temperature       *float64   `json:"temperature"`
	TotalONU          int        `gorm:"default:0;column:totalOnu" json:"totalOnu"`
	OnlineONU         int        `gorm:"default:0;column:onlineOnu" json:"onlineOnu"`
	OfflineONU        int        `gorm:"default:0;column:offlineOnu" json:"offlineOnu"`

	// Relations
	ONUStatuses        []OLTONUStatus         `gorm:"foreignKey:OltID" json:"onuStatuses,omitempty"`
	Alerts             []OLTAlert             `gorm:"foreignKey:OltID" json:"alerts,omitempty"`
	Routers            []NetworkOLTRouter     `gorm:"foreignKey:OltID" json:"routers,omitempty"`
	MonitoringLogs     []OLTMonitoringLog     `gorm:"foreignKey:OltID" json:"monitoringLogs,omitempty"`
	PerformanceMetrics []OLTPerformanceMetric `gorm:"foreignKey:OltID" json:"performanceMetrics,omitempty"`
}

func (NetworkOLT) TableName() string { return "network_olts" }

// ─── OLTONUStatus ─────────────────────────────────────────────────────────────

type OLTONUStatus struct {
	ID              string       `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	OltID           string       `gorm:"not null;index;column:oltId" json:"oltId"`
	OnuIndex        int          `gorm:"default:0;column:onuIndex" json:"onuIndex"`
	Frame           int          `gorm:"default:0;column:frame" json:"frame"`
	Slot            int          `gorm:"default:0;column:slot" json:"slot"`
	Port            int          `gorm:"column:port" json:"port"`
	OnuID           int          `gorm:"column:onuId" json:"onuId"`
	MACAddress      *string      `gorm:"column:macAddress" json:"macAddress"`
	SerialNumber    *string      `gorm:"index;column:serialNumber" json:"serialNumber"`
	Description     *string      `gorm:"type:text;column:description" json:"description"`
	Status          OltOnuStatus `gorm:"default:offline;column:status" json:"status"`
	RxPower         *float64     `gorm:"column:rxPower" json:"rxPower"`
	TxPower         *float64     `gorm:"column:txPower" json:"txPower"`
	Distance        *int         `gorm:"column:distance" json:"distance"`
	Temperature     *float64     `gorm:"column:temperature" json:"temperature"`
	Voltage         *float64     `gorm:"column:voltage" json:"voltage"`
	BiasCurrent     *float64     `gorm:"column:biasCurrent" json:"biasCurrent"`
	LastDeregReason *string      `gorm:"column:lastDeregReason" json:"lastDeregReason"`
	IPAddress       *string      `gorm:"column:ipAddress" json:"ipAddress"`
	VlanID          *int         `gorm:"column:vlanId" json:"vlanId"`
	BandwidthUp     int64        `gorm:"default:0;column:bandwidthUp" json:"bandwidthUp"`
	BandwidthDown   int64        `gorm:"default:0;column:bandwidthDown" json:"bandwidthDown"`
	CustomerID      *string      `gorm:"index;column:customerId" json:"customerId"`
	FirstSeenAt     time.Time    `gorm:"autoCreateTime;column:firstSeenAt" json:"firstSeenAt"`
	LastSeenAt      *time.Time   `gorm:"column:lastSeenAt" json:"lastSeenAt"`
	LastOfflineAt   *time.Time   `gorm:"column:lastOfflineAt" json:"lastOfflineAt"`
	CreatedAt       time.Time    `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt       time.Time    `gorm:"column:updatedAt" json:"updatedAt"`

	OLT      NetworkOLT `gorm:"foreignKey:OltID" json:"-"`
	Customer *PppoeUser `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
}

func (OLTONUStatus) TableName() string { return "olt_onu_status" }

// ─── OLTAlert ────────────────────────────────────────────────────────────────

type OLTAlert struct {
	ID                  string           `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	OltID               *string          `gorm:"index;column:oltId" json:"oltId"`
	OnuID               *string          `gorm:"index;column:onuId" json:"onuId"`
	AlertType           OltAlertType     `gorm:"default:onu_offline;column:alertType" json:"alertType"`
	Severity            OltAlertSeverity `gorm:"default:warning;column:severity" json:"severity"`
	Message             string           `gorm:"type:text;column:message" json:"message"`
	IsResolved          bool             `gorm:"default:false;column:isResolved" json:"isResolved"`
	ResolvedAt          *time.Time       `gorm:"column:resolvedAt" json:"resolvedAt"`
	NotifiedViaEmail    bool             `gorm:"default:false;column:notifiedViaEmail" json:"notifiedViaEmail"`
	NotifiedViaWhatsapp bool             `gorm:"default:false;column:notifiedViaWhatsapp" json:"notifiedViaWhatsapp"`
	CreatedAt           time.Time        `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt           time.Time        `gorm:"column:updatedAt" json:"updatedAt"`
}

func (OLTAlert) TableName() string { return "olt_alerts" }

// ─── OLTPerformanceMetric ─────────────────────────────────────────────────────

type OLTPerformanceMetric struct {
	ID          string    `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	OltID       string    `gorm:"not null;index;column:oltId" json:"oltId"`
	CPUUsage    *float64  `gorm:"column:cpuUsage" json:"cpuUsage"`
	MemoryUsage *float64  `gorm:"column:memoryUsage" json:"memoryUsage"`
	Temperature *float64  `gorm:"column:temperature" json:"temperature"`
	Uptime      *int64    `gorm:"column:uptime" json:"uptime"`
	TotalONU    int       `gorm:"default:0;column:totalOnu" json:"totalOnu"`
	OnlineONU   int       `gorm:"default:0;column:onlineOnu" json:"onlineOnu"`
	OfflineONU  int       `gorm:"default:0;column:offlineOnu" json:"offlineOnu"`
	RxBytes     int64     `gorm:"default:0;column:rxBytes" json:"rxBytes"`
	TxBytes     int64     `gorm:"default:0;column:txBytes" json:"txBytes"`
	RxErrors    int64     `gorm:"default:0;column:rxErrors" json:"rxErrors"`
	TxErrors    int64     `gorm:"default:0;column:txErrors" json:"txErrors"`
	RecordedAt  time.Time `gorm:"autoCreateTime;column:recordedAt" json:"recordedAt"`
}

func (OLTPerformanceMetric) TableName() string { return "olt_performance_metrics" }

// ─── NetworkOLTRouter ─────────────────────────────────────────────────────────

type NetworkOLTRouter struct {
	ID         string    `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	OltID      string    `gorm:"not null;index;column:oltId" json:"oltId"`
	RouterID   string    `gorm:"not null;index;column:routerId" json:"routerId"`
	UplinkPort *string   `gorm:"column:uplinkPort" json:"uplinkPort"`
	Priority   int       `gorm:"default:0;column:priority" json:"priority"`
	IsActive   bool      `gorm:"default:true;column:isActive" json:"isActive"`
	CreatedAt  time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updatedAt" json:"updatedAt"`
	Router     *Router   `gorm:"foreignKey:RouterID;references:ID" json:"router,omitempty"`
}

func (NetworkOLTRouter) TableName() string { return "network_olt_routers" }

// ─── OLTMonitoringLog ─────────────────────────────────────────────────────────

type OLTMonitoringLog struct {
	ID        string    `gorm:"primaryKey;type:varchar(191);column:id" json:"id"`
	OltID     string    `gorm:"not null;index;column:oltId" json:"oltId"`
	LogType   string    `gorm:"default:poll;column:logType" json:"logType"`
	Message   *string   `gorm:"type:text;column:message" json:"message"`
	Severity  string    `gorm:"default:info;column:severity" json:"severity"`
	CreatedAt time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
}

func (OLTMonitoringLog) TableName() string { return "olt_monitoring_logs" }
