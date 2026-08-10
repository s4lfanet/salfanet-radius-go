package models

import "time"

// ─── Enums ───────────────────────────────────────────────────────────────────

type UsersRole string

const (
	RoleAdmin     UsersRole = "ADMIN"
	RoleAgent     UsersRole = "AGENT"
	RoleUser      UsersRole = "USER"
	RoleCollector UsersRole = "COLLECTOR"
)

type InvoiceStatus string

const (
	InvoicePending   InvoiceStatus = "PENDING"
	InvoicePaid      InvoiceStatus = "PAID"
	InvoiceOverdue   InvoiceStatus = "OVERDUE"
	InvoiceCancelled InvoiceStatus = "CANCELLED"
)

type InvoiceType string

const (
	InvoiceMonthly      InvoiceType = "MONTHLY"
	InvoiceInstallation InvoiceType = "INSTALLATION"
	InvoiceAddon        InvoiceType = "ADDON"
	InvoiceTopup        InvoiceType = "TOPUP"
	InvoiceRenewal      InvoiceType = "RENEWAL"
)

type SubscriptionType string

const (
	Postpaid SubscriptionType = "POSTPAID"
	Prepaid  SubscriptionType = "PREPAID"
)

type ConnectionType string

const (
	ConnPPPoE    ConnectionType = "PPPOE"
	ConnHotspot  ConnectionType = "HOTSPOT"
	ConnStaticIP ConnectionType = "STATIC_IP"
)

// ─── User (admin) ─────────────────────────────────────────────────────────────

type User struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	Name      string    `json:"name"`
	Role      UsersRole `gorm:"default:ADMIN" json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (User) TableName() string { return "users" }

// ─── PppoeArea ───────────────────────────────────────────────────────────────

type PppoeArea struct {
	ID          string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description *string   `json:"description"`
	IsActive    bool      `gorm:"default:true" json:"isActive"`
	TerritoryID *string   `gorm:"column:territoryId;index" json:"territoryId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (PppoeArea) TableName() string { return "pppoe_areas" }

// ─── PppoeProfile ────────────────────────────────────────────────────────────

type PppoeProfile struct {
	ID                  string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name                string    `gorm:"uniqueIndex;not null" json:"name"`
	Price               int       `json:"price"`
	Description         *string   `json:"description"`
	DownloadSpeed       int       `gorm:"column:downloadSpeed" json:"downloadSpeed"`
	UploadSpeed         int       `gorm:"column:uploadSpeed" json:"uploadSpeed"`
	RateLimit           *string   `gorm:"column:rateLimit" json:"rateLimit"`
	GroupName           string    `gorm:"column:groupName" json:"groupName"`
	MikrotikProfileName *string   `gorm:"column:mikrotikProfileName" json:"mikrotikProfileName"`
	IPPoolName          *string   `gorm:"column:ipPoolName" json:"ipPoolName"`
	IPPoolRange         *string   `gorm:"column:ipPoolRange" json:"ipPoolRange"`
	LocalAddress        *string   `gorm:"column:localAddress" json:"localAddress"`
	HPP                 *int      `json:"hpp"`
	PPNActive           bool      `gorm:"default:false;column:ppnActive" json:"ppnActive"`
	PPNRate             int       `gorm:"default:11;column:ppnRate" json:"ppnRate"`
	IsActive            bool      `gorm:"default:true;column:isActive" json:"isActive"`
	SyncedToRadius      bool      `gorm:"default:false;column:syncedToRadius" json:"syncedToRadius"`
	ValidityUnit        string    `gorm:"default:MONTHS;column:validityUnit" json:"validityUnit"`
	ValidityValue       int       `gorm:"default:1;column:validityValue" json:"validityValue"`
	SharedUser          bool      `gorm:"default:true;column:sharedUser" json:"sharedUser"`
	CreatedAt           time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt           time.Time `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
}

func (PppoeProfile) TableName() string { return "pppoe_profiles" }

// ─── PppoeCustomer ───────────────────────────────────────────────────────────

type PppoeCustomer struct {
	ID           string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	CustomerID   string    `gorm:"uniqueIndex;type:varchar(10);column:customerId" json:"customerId"`
	Name         string    `json:"name"`
	Phone        string    `gorm:"index" json:"phone"`
	Email        *string   `json:"email"`
	Address      *string   `gorm:"type:text" json:"address"`
	IDCardNumber *string   `gorm:"type:varchar(50);column:idCardNumber" json:"idCardNumber"`
	IDCardPhoto  *string   `gorm:"type:varchar(500);column:idCardPhoto" json:"idCardPhoto"`
	IsActive     bool      `gorm:"default:true;column:isActive" json:"isActive"`
	AreaID       *string   `gorm:"column:areaId" json:"areaId"`
	CreatedAt    time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`

	Area       *PppoeArea  `gorm:"foreignKey:AreaID" json:"area,omitempty"`
	PPPoEUsers []PppoeUser `gorm:"foreignKey:PppoeCustomerID" json:"pppoeUsers,omitempty"`
}

func (PppoeCustomer) TableName() string { return "pppoe_customers" }

// ─── PppoeUser ───────────────────────────────────────────────────────────────

type PppoeUser struct {
	ID                    string           `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Username              string           `gorm:"uniqueIndex;not null" json:"username"`
	CustomerID            *string          `gorm:"uniqueIndex;column:customer_id;type:varchar(20)" json:"customerId"`
	PppoeCustomerID       *string          `gorm:"column:pppoe_customer_id;index" json:"pppoeCustomerId"`
	Password              string           `gorm:"not null" json:"-"`
	ProfileID             string           `gorm:"index;column:profileId" json:"profileId"`
	AreaID                *string          `gorm:"index;column:areaId" json:"areaId"`
	Status                string           `gorm:"default:active;index" json:"status"`
	IPAddress             *string          `gorm:"column:ipAddress" json:"ipAddress"`
	MACAddress            *string          `gorm:"column:macAddress" json:"macAddress"`
	Comment               *string          `json:"comment"`
	CreatedAt             time.Time        `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt             time.Time        `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
	ExpiredAt             *time.Time       `gorm:"index;column:expiredAt" json:"expiredAt"`
	Address               *string          `json:"address"`
	Latitude              *float64         `json:"latitude"`
	Longitude             *float64         `json:"longitude"`
	Email                 *string          `json:"email"`
	Name                  string           `json:"name"`
	Phone                 string           `gorm:"index" json:"phone"`
	RouterID              *string          `gorm:"index;column:routerId" json:"routerId"`
	SubscriptionType      SubscriptionType `gorm:"default:POSTPAID;index;column:subscriptionType" json:"subscriptionType"`
	LastPaymentDate       *time.Time       `gorm:"column:lastPaymentDate" json:"lastPaymentDate"`
	BillingDay            *int             `gorm:"default:1;column:billingDay" json:"billingDay"`
	AutoIsolationEnabled  bool             `gorm:"default:true;column:autoIsolationEnabled" json:"autoIsolationEnabled"`
	Balance               int              `gorm:"default:0" json:"balance"`
	AutoRenewal           bool             `gorm:"default:false;column:autoRenewal" json:"autoRenewal"`
	ConnectionType        ConnectionType   `gorm:"default:PPPOE;column:connectionType" json:"connectionType"`
	ReferralCode          *string          `gorm:"uniqueIndex;type:varchar(10);column:referralCode" json:"referralCode"`
	ReferredByID          *string          `gorm:"column:referred_by_id" json:"referredById"`
	SyncedToRadius        bool             `gorm:"default:false;column:syncedToRadius" json:"syncedToRadius"`
	TerritoryID           *string          `gorm:"index;column:territoryId" json:"territoryId"`
	TerritoryAreaID       *string          `gorm:"index;column:territoryAreaId" json:"territoryAreaId"`
	InitialPaymentPending bool             `gorm:"default:false;column:initialPaymentPending" json:"initialPaymentPending"`
	PsbDeadlineAt         *time.Time       `gorm:"index;column:psbDeadlineAt" json:"psbDeadlineAt"`
	IdCardNumber          *string          `gorm:"column:idCardNumber;type:varchar(50)" json:"idCardNumber"`
	IdCardPhoto           *string          `gorm:"column:idCardPhoto;type:varchar(500)" json:"idCardPhoto"`
	InstallationPhotos    string           `gorm:"column:installationPhotos;type:json" json:"installationPhotos"`
	FollowRoad            bool             `gorm:"default:false;column:followRoad" json:"followRoad"`

	Profile       PppoeProfile           `gorm:"foreignKey:ProfileID" json:"profile,omitempty"`
	Territory     *Territory             `gorm:"foreignKey:TerritoryID" json:"territory,omitempty"`
	Area          *PppoeArea             `gorm:"foreignKey:AreaID" json:"area,omitempty"`
	Router        *Router                `gorm:"foreignKey:RouterID" json:"router,omitempty"`
	ODPAssignment *OdpCustomerAssignment `gorm:"foreignKey:CustomerID;references:ID" json:"odpAssignment,omitempty"`
	Invoices      []Invoice              `gorm:"foreignKey:UserID" json:"invoices,omitempty"`
}

func (PppoeUser) TableName() string { return "pppoe_users" }

// ─── Invoice ─────────────────────────────────────────────────────────────────

type Invoice struct {
	ID               string        `gorm:"primaryKey;type:varchar(191)" json:"id"`
	InvoiceNumber    string        `gorm:"uniqueIndex;not null;column:invoiceNumber" json:"invoiceNumber"`
	UserID           *string       `gorm:"index;column:userId" json:"userId"`
	Amount           int           `json:"amount"`
	Status           InvoiceStatus `gorm:"default:PENDING;index" json:"status"`
	DueDate          time.Time     `gorm:"index;column:dueDate" json:"dueDate"`
	PaidAt           *time.Time    `gorm:"column:paidAt" json:"paidAt"`
	CreatedAt        time.Time     `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt        time.Time     `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
	PaymentLink      *string       `gorm:"column:paymentLink" json:"paymentLink"`
	PaymentToken     *string       `gorm:"uniqueIndex;column:paymentToken" json:"paymentToken"`
	CustomerName     *string       `gorm:"column:customerName" json:"customerName"`
	CustomerPhone    *string       `gorm:"column:customerPhone" json:"customerPhone"`
	CustomerEmail    *string       `gorm:"column:customerEmail" json:"customerEmail"`
	CustomerUsername *string       `gorm:"column:customerUsername" json:"customerUsername"`
	SentReminders    *string       `gorm:"type:text;column:sentReminders" json:"sentReminders"`
	Notes            *string       `gorm:"type:text" json:"notes"`
	InvoiceType      InvoiceType   `gorm:"default:MONTHLY;index;column:invoiceType" json:"invoiceType"`
	BaseAmount       *int          `gorm:"column:baseAmount" json:"baseAmount"`
	DiscountAmount   *int          `gorm:"column:discountAmount" json:"discountAmount"`
	DiscountReason   *string       `gorm:"type:text;column:discountReason" json:"discountReason"`
	OriginalAmount   *int          `gorm:"column:originalAmount" json:"originalAmount"`
	CancelledAt      *time.Time    `gorm:"column:cancelledAt" json:"cancelledAt"`
	CancelledBy      *string       `gorm:"column:cancelledBy" json:"cancelledBy"`
	CancelReason     *string       `gorm:"type:text;column:cancelReason" json:"cancelReason"`

	User *PppoeUser `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Invoice) TableName() string { return "invoices" }

// ─── Router (NAS) ────────────────────────────────────────────────────────────

type Router struct {
	ID          string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name        string    `json:"name"`
	NASName     string    `gorm:"column:nasname" json:"nasname"`
	ShortName   string    `gorm:"column:shortname;index" json:"shortname"`
	Type        string    `gorm:"default:mikrotik" json:"type"`
	IPAddress   string    `gorm:"column:ipAddress" json:"ipAddress"`
	Username    string    `json:"username"`
	Password    string    `json:"-"` // omit from responses; use routerBody to read from requests
	Port        int       `gorm:"default:8728" json:"port"`
	APIPort     int       `gorm:"default:8729;column:apiPort" json:"apiPort"`
	Secret      string    `gorm:"default:secret123" json:"-"` // omit from responses
	Ports       int       `gorm:"default:1812" json:"ports"`
	Server      *string   `json:"server"`
	Community   *string   `json:"community"`
	VpnClientId *string   `gorm:"column:vpnClientId" json:"vpnClientId"`
	AuthMode    string    `gorm:"column:auth_mode;default:radius" json:"authMode"` // 'local' = PPP Secret, 'radius' = FreeRADIUS
	Description *string   `json:"description"`
	Latitude    *float64  `json:"latitude"`
	Longitude   *float64  `json:"longitude"`
	IsActive    bool      `gorm:"default:true;column:isActive" json:"isActive"`
	CreatedAt   time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
}

func (Router) TableName() string { return "nas" }

// ─── Company ─────────────────────────────────────────────────────────────────

type Company struct {
	ID                          string  `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name                        string  `json:"name"`
	Address                     *string `json:"address"`
	Phone                       *string `json:"phone"`
	Email                       *string `json:"email"`
	Logo                        *string `json:"logo"`
	AdminPhone                  *string `gorm:"column:adminPhone" json:"adminPhone"`
	BaseURL                     *string `gorm:"default:http://localhost:3000;column:baseUrl" json:"baseUrl"`
	Timezone                    *string `gorm:"default:Asia/Jakarta" json:"timezone"`
	PoweredBy                   *string `gorm:"default:SALFANET RADIUS;column:poweredBy" json:"poweredBy"`
	CustomerIDPrefix            *string `gorm:"type:varchar(10);column:customerIdPrefix" json:"customerIdPrefix"`
	FooterAdmin                 *string `gorm:"type:text;column:footerAdmin" json:"footerAdmin"`
	FooterCustomer              *string `gorm:"type:text;column:footerCustomer" json:"footerCustomer"`
	FooterTechnician            *string `gorm:"type:text;column:footerTechnician" json:"footerTechnician"`
	FooterAgent                 *string `gorm:"type:text;column:footerAgent" json:"footerAgent"`
	InvoiceGenerateDays         *int    `gorm:"default:7;column:invoiceGenerateDays" json:"invoiceGenerateDays"`
	GracePeriodDays             *int    `gorm:"default:0;column:gracePeriodDays" json:"gracePeriodDays"`
	IsolationEnabled            *bool   `gorm:"default:true;column:isolationEnabled" json:"isolationEnabled"`
	IsolationIpPool             *string `gorm:"column:isolationIpPool" json:"isolationIpPool"`
	IsolationServerIp           *string `gorm:"column:isolationServerIp" json:"isolationServerIp"`
	IsolationRateLimit          *string `gorm:"column:isolationRateLimit" json:"isolationRateLimit"`
	IsolationRedirectUrl        *string `gorm:"column:isolationRedirectUrl" json:"isolationRedirectUrl"`
	IsolationMessage            *string `gorm:"type:text;column:isolationMessage" json:"isolationMessage"`
	IsolationAllowDns           *bool   `gorm:"default:true;column:isolationAllowDns" json:"isolationAllowDns"`
	IsolationAllowPayment       *bool   `gorm:"default:true;column:isolationAllowPayment" json:"isolationAllowPayment"`
	IsolationNotifyWhatsapp     *bool   `gorm:"default:false;column:isolationNotifyWhatsapp" json:"isolationNotifyWhatsapp"`
	IsolationNotifyEmail        *bool   `gorm:"default:false;column:isolationNotifyEmail" json:"isolationNotifyEmail"`
	IsolationWhatsappTemplateId *string `gorm:"type:varchar(191);column:isolationWhatsappTemplateId" json:"isolationWhatsappTemplateId"`
	IsolationEmailTemplateId    *string `gorm:"type:varchar(191);column:isolationEmailTemplateId" json:"isolationEmailTemplateId"`
	IsolationHtmlTemplateId     *string `gorm:"type:varchar(191);column:isolationHtmlTemplateId" json:"isolationHtmlTemplateId"`
	PppoeRenewalAnytime         *bool   `gorm:"default:false;column:pppoeRenewalAnytime" json:"pppoeRenewalAnytime"`
	PppoeRenewalDaysBefore      *int    `gorm:"default:7;column:pppoeRenewalDaysBefore" json:"pppoeRenewalDaysBefore"`
	BankAccounts                *string `gorm:"type:text;column:bankAccounts" json:"bankAccounts"`
	ReferralEnabled             *bool   `gorm:"default:false;column:referralEnabled" json:"referralEnabled"`
	ReferralRewardAmount        *int    `gorm:"default:10000;column:referralRewardAmount" json:"referralRewardAmount"`
	ReferralRewardType          *string `gorm:"default:FIRST_PAYMENT;column:referralRewardType" json:"referralRewardType"`
	ReferralRewardBoth          *bool   `gorm:"default:false;column:referralRewardBoth" json:"referralRewardBoth"`
	ReferralReferredAmount      *int    `gorm:"default:0;column:referralReferredAmount" json:"referralReferredAmount"`
	// QRIS Mandiri — konversi QRIS statis dari bank ke dinamis
	QrisStaticCode   *string `gorm:"type:longtext;column:qrisStaticCode" json:"qrisStaticCode"`
	QrisMerchantName *string `gorm:"type:varchar(191);column:qrisMerchantName" json:"qrisMerchantName"`
	QrisEnabled      *bool   `gorm:"default:false;column:qrisEnabled" json:"qrisEnabled"`
	// Device key untuk autentikasi Android QrisListener app
	QrisDeviceKey *string   `gorm:"type:varchar(100);column:qrisDeviceKey" json:"qrisDeviceKey"`
	CreatedAt     time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
}

func (Company) TableName() string { return "companies" }

// ─── QrisPending ─────────────────────────────────────────────────────────────

// QrisPending menyimpan QRIS transaksi yang sedang menunggu pembayaran.
// Sistem pencocokan: Android app mengirim nominal → cocok dengan uniqueAmount.
type QrisPending struct {
	ID           string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	InvoiceID    string     `gorm:"index;type:varchar(191);column:invoiceId" json:"invoiceId"`
	UserID       *string    `gorm:"index;type:varchar(191);column:userId" json:"userId"`
	OrderID      string     `gorm:"uniqueIndex;type:varchar(191);column:orderId" json:"orderId"`
	BaseAmount   int        `gorm:"column:baseAmount" json:"baseAmount"`
	UniqueAmount int        `gorm:"index;column:uniqueAmount" json:"uniqueAmount"` // BaseAmount + random 1-999 (untuk pencocokan notif)
	QrString     string     `gorm:"type:longtext;column:qrString" json:"qrString"`
	Status       string     `gorm:"default:pending;index" json:"status"`                 // pending | paid | expired
	SourceApp    string     `gorm:"type:varchar(100);column:sourceApp" json:"sourceApp"` // id.dana, com.gojek.app, dll
	ExpiresAt    time.Time  `gorm:"index;column:expiresAt" json:"expiresAt"`
	PaidAt       *time.Time `gorm:"column:paidAt" json:"paidAt"`
	CreatedAt    time.Time  `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
}

func (QrisPending) TableName() string { return "qris_pendings" }

// ─── CronHistory ─────────────────────────────────────────────────────────────

type CronHistory struct {
	ID          string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	JobType     string     `gorm:"column:jobType;index" json:"jobType"`
	Status      string     `gorm:"column:status;index" json:"status"`
	StartedAt   time.Time  `gorm:"column:startedAt;autoCreateTime;index" json:"startedAt"`
	CompletedAt *time.Time `gorm:"column:completedAt" json:"completedAt"`
	Duration    *int       `gorm:"column:duration" json:"duration"`
	Result      *string    `gorm:"column:result;type:text" json:"result"`
	Error       *string    `gorm:"column:error;type:text" json:"error"`
}

func (CronHistory) TableName() string { return "cron_history" }

// ─── Radcheck ────────────────────────────────────────────────────────────────

type Radcheck struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string `gorm:"type:varchar(64);index" json:"username"`
	Attribute string `gorm:"type:varchar(64)" json:"attribute"`
	Op        string `gorm:"type:char(2);default:=" json:"op"`
	Value     string `gorm:"type:varchar(253)" json:"value"`
}

func (Radcheck) TableName() string { return "radcheck" }

// ─── Radreply ────────────────────────────────────────────────────────────────

type Radreply struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string `gorm:"type:varchar(64);index" json:"username"`
	Attribute string `gorm:"type:varchar(64)" json:"attribute"`
	Op        string `gorm:"type:char(2);default:=" json:"op"`
	Value     string `gorm:"type:varchar(253)" json:"value"`
}

func (Radreply) TableName() string { return "radreply" }

// ─── Radusergroup ─────────────────────────────────────────────────────────────

type Radusergroup struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string `gorm:"type:varchar(64);index" json:"username"`
	Groupname string `gorm:"type:varchar(64);index" json:"groupname"`
	Priority  int    `gorm:"default:1" json:"priority"`
}

func (Radusergroup) TableName() string { return "radusergroup" }

// ─── Radacct ─────────────────────────────────────────────────────────────────

type Radacct struct {
	RadacctID          int64      `gorm:"primaryKey;autoIncrement;column:radacctid" json:"radacctid"`
	AcctSessionID      string     `gorm:"type:varchar(64);column:acctsessionid" json:"acctsessionid"`
	AcctUniqueID       string     `gorm:"uniqueIndex;type:varchar(32);column:acctuniqueid" json:"acctuniqueid"`
	Username           string     `gorm:"type:varchar(64);index;column:username" json:"username"`
	NASIPAddress       string     `gorm:"type:varchar(15);index;column:nasipaddress" json:"nasipaddress"`
	AcctStartTime      *time.Time `gorm:"column:acctstarttime" json:"acctstarttime"`
	AcctUpdateTime     *time.Time `gorm:"column:acctupdatetime" json:"acctupdatetime"`
	AcctStopTime       *time.Time `gorm:"index;column:acctstoptime" json:"acctstoptime"`
	AcctSessionTime    *int64     `gorm:"column:acctsessiontime" json:"acctsessiontime"`
	FramedIPAddress    string     `gorm:"type:varchar(15);index;column:framedipaddress" json:"framedipaddress"`
	AcctInputOctets    *int64     `gorm:"column:acctinputoctets" json:"acctinputoctets"`
	AcctOutputOctets   *int64     `gorm:"column:acctoutputoctets" json:"acctoutputoctets"`
	CalledStationID    string     `gorm:"type:varchar(50);column:calledstationid" json:"calledstationid"`
	CallingStationID   string     `gorm:"type:varchar(50);column:callingstationid" json:"callingstationid"`
	AcctTerminateCause string     `gorm:"type:varchar(32);column:acctterminatecause" json:"acctterminatecause"`
}

func (Radacct) TableName() string { return "radacct" }

// ─── CustomerSession ─────────────────────────────────────────────────────────

type CustomerSession struct {
	ID        string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	UserID    string     `gorm:"index;column:userId" json:"userId"`
	Phone     string     `gorm:"index" json:"phone"`
	OTPCode   *string    `gorm:"column:otpCode" json:"-"`
	OTPExpiry *time.Time `gorm:"column:otpExpiry" json:"-"`
	Token     *string    `gorm:"uniqueIndex" json:"-"`
	ExpiresAt *time.Time `gorm:"column:expiresAt" json:"expiresAt"`
	Verified  bool       `gorm:"default:false" json:"verified"`
	CreatedAt time.Time  `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
}

func (CustomerSession) TableName() string { return "customer_sessions" }

// ─── WhatsappProvider ────────────────────────────────────────────────────────

type WhatsappProvider struct {
	ID           string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	APIKey       string    `gorm:"column:apiKey" json:"-"`
	APIURL       string    `gorm:"column:apiUrl" json:"apiUrl"`
	SenderNumber *string   `gorm:"column:senderNumber" json:"senderNumber"`
	IsActive     bool      `gorm:"default:true;column:isActive" json:"isActive"`
	Priority     int       `gorm:"default:0" json:"priority"`
	Description  *string   `gorm:"type:text" json:"description"`
	CreatedAt    time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
}

func (WhatsappProvider) TableName() string { return "whatsapp_providers" }

// ─── WhatsappTemplate ────────────────────────────────────────────────────────

type WhatsappTemplate struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name      string    `json:"name"`
	Type      string    `gorm:"uniqueIndex" json:"type"`
	Message   string    `gorm:"type:text" json:"message"`
	IsActive  bool      `gorm:"default:true;column:isActive" json:"isActive"`
	CreatedAt time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
}

func (WhatsappTemplate) TableName() string { return "whatsapp_templates" }

// ─── ManualPayment ────────────────────────────────────────────────────────────
// NOTE: DB columns use original Prisma camelCase names; explicit tags needed where
// the Go field name would otherwise map to the wrong column name.

type ManualPayment struct {
	ID            string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	InvoiceID     string     `gorm:"column:invoiceId;index" json:"invoiceId"`
	PppoeUserID   string     `gorm:"column:userId;index" json:"pppoeUserId"` // DB column: userId
	Amount        float64    `json:"amount"`
	BankName      string     `gorm:"column:bankName" json:"bankName"`
	AccountName   string     `gorm:"column:accountName" json:"accountName"`
	AccountNumber *string    `gorm:"column:accountNumber" json:"accountNumber"`
	TransferDate  time.Time  `gorm:"column:paymentDate" json:"transferDate"` // DB column: paymentDate
	ProofImage    *string    `gorm:"column:receiptImage" json:"proofImage"`  // DB column: receiptImage
	Notes         *string    `gorm:"type:text" json:"notes"`
	Status        string     `gorm:"default:PENDING;index" json:"status"`
	ReviewedBy    *string    `gorm:"column:approvedBy" json:"reviewedBy"`                 // DB column: approvedBy
	ReviewedAt    *time.Time `gorm:"column:approvedAt" json:"reviewedAt"`                 // DB column: approvedAt
	ReviewNotes   *string    `gorm:"column:rejectionReason;type:text" json:"reviewNotes"` // DB column: rejectionReason
	CreatedAt     time.Time  `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time  `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`

	Invoice   *Invoice   `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
	PppoeUser *PppoeUser `gorm:"foreignKey:PppoeUserID" json:"pppoeUser,omitempty"`
}

func (ManualPayment) TableName() string { return "manual_payments" }

// ─── Payment (gateway payments table) ────────────────────────────────────────

type Payment struct {
	ID                     string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	InvoiceID              string    `gorm:"index;column:invoiceId" json:"invoiceId"`
	Amount                 int       `json:"amount"`
	Method                 string    `json:"method"`
	GatewayID              *string   `gorm:"index;column:gatewayId" json:"gatewayId"`
	Status                 string    `gorm:"default:PENDING;index" json:"status"`
	Notes                  *string   `gorm:"type:text" json:"notes"`
	PaymentMethodEditCount int       `gorm:"default:0;column:paymentMethodEditCount" json:"paymentMethodEditCount"`
	PaidAt                 time.Time `gorm:"column:paidAt" json:"paidAt"`
	CreatedAt              time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`

	Invoice *Invoice `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
}

func (Payment) TableName() string { return "payments" }

// ─── Ticket ───────────────────────────────────────────────────────────────────

type Ticket struct {
	ID             string     `gorm:"primaryKey;type:varchar(191)" json:"id"`
	TicketNumber   string     `gorm:"uniqueIndex;column:ticketNumber" json:"ticketNumber"`
	CustomerID     *string    `gorm:"index;column:customerId" json:"customerId"`
	CustomerName   string     `gorm:"column:customerName" json:"customerName"`
	CustomerEmail  *string    `gorm:"column:customerEmail" json:"customerEmail"`
	CustomerPhone  string     `gorm:"column:customerPhone" json:"customerPhone"`
	Subject        string     `json:"subject"`
	Description    string     `gorm:"type:text" json:"description"`
	CategoryID     *string    `gorm:"index;column:categoryId" json:"categoryId"`
	Priority       string     `gorm:"default:MEDIUM" json:"priority"`
	Status         string     `gorm:"default:OPEN;index" json:"status"`
	AssignedToID   *string    `gorm:"column:assignedToId" json:"assignedToId"`
	AssignedToType *string    `gorm:"column:assignedToType" json:"assignedToType"`
	CreatedAt      time.Time  `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
	ClosedAt       *time.Time `gorm:"column:closedAt" json:"closedAt"`
	ResolvedAt     *time.Time `gorm:"column:resolvedAt" json:"resolvedAt"`

	Customer *PppoeUser      `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Category *TicketCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

func (Ticket) TableName() string { return "tickets" }
