package db

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// prismaStyleNamer converts GORM's default snake_case column names to Prisma-style camelCase.
// GORM converts Go PascalCase to snake_case (e.g. UpdatedAt → updated_at),
// but Prisma stores columns in camelCase (e.g. updatedAt).
// This namer wraps GORM's default and converts snake_case → camelCase as the last step.
type prismaStyleNamer struct {
	schema.NamingStrategy
}

func (n prismaStyleNamer) ColumnName(table, column string) string {
	snake := n.NamingStrategy.ColumnName(table, column)
	return snakeToCamel(snake)
}

func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		return s
	}
	result := parts[0]
	for _, p := range parts[1:] {
		if len(p) > 0 {
			result += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return result
}

// Init connects to MySQL using the given DSN and configures a connection pool.
// The DSN should be in the format: user:pass@tcp(host:port)/dbname?parseTime=True&loc=Local
func Init(databaseURL string) (*gorm.DB, error) {
	// Convert Prisma-style URL to Go MySQL DSN
	// Prisma: mysql://user:pass@localhost:3306/dbname
	// GORM:   user:pass@tcp(localhost:3306)/dbname?parseTime=True&loc=Local
	dsn, err := convertDSN(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Warn),
		NamingStrategy: prismaStyleNamer{},
		NowFunc: func() time.Time {
			return time.Now()
		},
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Run Go-managed table migrations (tables not managed by Prisma)
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("db migrations failed: %w", err)
	}

	DB = db
	return db, nil
}

// runMigrations creates tables that are managed by Go (not by Prisma).
// Uses raw *sql.DB (not GORM Exec) to bypass PrepareStmt=true which
// blocks DDL statements on some MySQL setups.
func runMigrations(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("runMigrations: get sql.DB: %w", err)
	}
	statements := []string{
		// vpn_servers: managed by Go (@@ignore in Prisma schema) — Prisma will NOT touch this table
		`CREATE TABLE IF NOT EXISTS vpn_servers (
			id             VARCHAR(191) NOT NULL,
			name           VARCHAR(255) NOT NULL DEFAULT '',
			host           VARCHAR(255) NOT NULL DEFAULT '',
			username       VARCHAR(255) NOT NULL DEFAULT '',
			password       VARCHAR(255) NOT NULL DEFAULT '',
			apiPort        INT          NOT NULL DEFAULT 8728,
			subnet         VARCHAR(45)  NOT NULL DEFAULT '',
			poolStart      INT          NOT NULL DEFAULT 10,
			poolEnd        INT          NOT NULL DEFAULT 254,
			gateway        VARCHAR(45)  NULL,
			l2tpEnabled    TINYINT(1)   NOT NULL DEFAULT 0,
			sstpEnabled    TINYINT(1)   NOT NULL DEFAULT 0,
			pptpEnabled    TINYINT(1)   NOT NULL DEFAULT 0,
			wgEnabled      TINYINT(1)   NOT NULL DEFAULT 0,
			wgPublicKey    TEXT         NULL,
			wgPort         INT          NULL DEFAULT 51820,
			openVpnEnabled TINYINT(1)   NOT NULL DEFAULT 0,
			openVpnPort    INT          NULL DEFAULT 1194,
			isActive       TINYINT(1)   NOT NULL DEFAULT 1,
			createdAt      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			updatedAt      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		// vpn_clients: managed by Go (@@ignore in Prisma schema) — Prisma will NOT touch this table
		`CREATE TABLE IF NOT EXISTS vpn_clients (
			id               VARCHAR(191) NOT NULL,
			name             VARCHAR(255) NOT NULL DEFAULT '',
			vpnServerId      VARCHAR(191) NOT NULL,
			vpnIp            VARCHAR(45)  NOT NULL DEFAULT '',
			username         VARCHAR(255) NOT NULL DEFAULT '',
			password         VARCHAR(255) NOT NULL DEFAULT '',
			vpnType          VARCHAR(50)  NOT NULL DEFAULT 'L2TP',
			description      TEXT         NULL,
			winboxPort       INT          NULL,
			apiUsername      VARCHAR(255) NULL,
			apiPassword      VARCHAR(255) NULL,
			clientPublicKey  TEXT         NULL,
			clientPrivateKey TEXT         NULL,
			isActive         TINYINT(1)   NOT NULL DEFAULT 1,
			isRadiusServer   TINYINT(1)   NOT NULL DEFAULT 0,
			createdAt        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			updatedAt        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			UNIQUE KEY vpn_clients_username_key (username),
			UNIQUE KEY vpn_clients_vpnServerId_vpnIp_key (vpnServerId, vpnIp),
			KEY vpn_clients_vpnServerId_fkey (vpnServerId),
			CONSTRAINT vpn_clients_vpnServerId_fkey FOREIGN KEY (vpnServerId) REFERENCES vpn_servers (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS vps_peers (
			id                VARCHAR(191) NOT NULL,
			type              VARCHAR(50)  NOT NULL DEFAULT 'wireguard',
			peer_name         VARCHAR(255) NOT NULL DEFAULT '',
			peer_ip           VARCHAR(45)  NOT NULL DEFAULT '',
			local_ip          VARCHAR(45)  NOT NULL DEFAULT '',
			public_key        TEXT         NULL,
			nas_secret        VARCHAR(64)  NULL,
			api_username      VARCHAR(255) NULL,
			api_password      VARCHAR(255) NULL,
			client_private_key TEXT        NULL,
			is_active         TINYINT(1)   NOT NULL DEFAULT 1,
			created_at        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			updated_at        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			INDEX idx_vps_peers_type (type)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		// topup_requests: customer top-up/balance requests submitted via the customer portal
		`CREATE TABLE IF NOT EXISTS topup_requests (
			id            VARCHAR(191) NOT NULL,
			userId        VARCHAR(191) NOT NULL,
			amount        INT          NOT NULL DEFAULT 0,
			paymentMethod VARCHAR(255) NOT NULL DEFAULT '',
			description   TEXT         NULL,
			status        VARCHAR(50)  NOT NULL DEFAULT 'PENDING',
			proofUrl      VARCHAR(500) NULL,
			metadata      TEXT         NULL,
			createdAt     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			processedAt   DATETIME(3)  NULL,
			PRIMARY KEY (id),
			INDEX idx_topup_requests_userId (userId),
			INDEX idx_topup_requests_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		// territories: ISP territory/collector management (Go-managed)
		`CREATE TABLE IF NOT EXISTS territories (
			id             VARCHAR(191) NOT NULL,
			name           VARCHAR(150) NOT NULL,
			description    TEXT         NULL,
			collectorId    VARCHAR(191) NULL,
			isActive       TINYINT(1)   NOT NULL DEFAULT 1,
			createdAt      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			updatedAt      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			UNIQUE KEY territories_name_key (name),
			INDEX idx_territories_collectorId (collectorId)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		// territory_areas: kelurahan/dusun within a territory (Go-managed)
		`CREATE TABLE IF NOT EXISTS territory_areas (
			id             VARCHAR(191) NOT NULL,
			territoryId    VARCHAR(191) NOT NULL,
			kelurahanKode  VARCHAR(20)  NULL,
			kelurahanNama  VARCHAR(150) NULL,
			kecamatanNama  VARCHAR(150) NULL,
			kabupatenNama  VARCHAR(150) NULL,
			provinsiNama   VARCHAR(150) NULL,
			dusunNama      VARCHAR(150) NULL,
			collectorId    VARCHAR(191) NULL,
			createdAt      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			INDEX idx_territory_areas_territoryId (territoryId),
			INDEX idx_territory_areas_kelurahanKode (kelurahanKode),
			INDEX idx_territory_areas_collectorId (collectorId)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		// settlements: collector settlement reports (Go-managed)
		`CREATE TABLE IF NOT EXISTS settlements (
			id             VARCHAR(191) NOT NULL,
			collectorId    VARCHAR(191) NOT NULL,
			periodDate     DATE         NOT NULL,
			totalAmount    INT          NOT NULL DEFAULT 0,
			invoiceCount   INT          NOT NULL DEFAULT 0,
			status         VARCHAR(50)  NOT NULL DEFAULT 'pending',
			confirmedBy    VARCHAR(191) NULL,
			confirmedAt    DATETIME(3)  NULL,
			createdAt      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			INDEX idx_settlements_collectorId (collectorId),
			INDEX idx_settlements_periodDate (periodDate),
			INDEX idx_settlements_status (status),
			UNIQUE KEY uniq_settlement_collector_date (collectorId, periodDate)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		// Add territoryId and territoryAreaId columns to pppoe_users (Prisma-managed table, Go adds columns)
		`ALTER TABLE pppoe_users ADD COLUMN territoryId VARCHAR(191) NULL`,
		`ALTER TABLE pppoe_users ADD COLUMN territoryAreaId VARCHAR(191) NULL`,
		`ALTER TABLE pppoe_users ADD INDEX idx_pppoe_users_territoryId (territoryId)`,
		`ALTER TABLE pppoe_users ADD INDEX idx_pppoe_users_territoryAreaId (territoryAreaId)`,
		// Phase 2: Invoice discount & cancel columns
		`ALTER TABLE invoices ADD COLUMN discountAmount INT NULL`,
		`ALTER TABLE invoices ADD COLUMN discountReason TEXT NULL`,
		`ALTER TABLE invoices ADD COLUMN originalAmount INT NULL`,
		`ALTER TABLE invoices ADD COLUMN cancelledAt DATETIME(3) NULL`,
		`ALTER TABLE invoices ADD COLUMN cancelledBy VARCHAR(191) NULL`,
		`ALTER TABLE invoices ADD COLUMN cancelReason TEXT NULL`,
		// Phase 2: Package change logs
		`CREATE TABLE IF NOT EXISTS package_change_logs (
			id             VARCHAR(191) NOT NULL,
			userId         VARCHAR(191) NOT NULL,
			username       VARCHAR(191) NOT NULL,
			oldProfileId   VARCHAR(191) NULL,
			oldProfileName VARCHAR(100) NULL,
			newProfileId   VARCHAR(191) NULL,
			newProfileName VARCHAR(100) NULL,
			changedBy      VARCHAR(191) NOT NULL,
			changedByName  VARCHAR(150) NULL,
			reason         TEXT         NULL,
			changedAt      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			INDEX idx_pcl_user (userId),
			INDEX idx_pcl_date (changedAt)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		// Phase 2: Installation logs
		`CREATE TABLE IF NOT EXISTS installation_logs (
			id             VARCHAR(191) NOT NULL,
			userId         VARCHAR(191) NOT NULL,
			username       VARCHAR(191) NOT NULL,
			customerId     VARCHAR(20)  NULL,
			fullname       VARCHAR(150) NULL,
			phone          VARCHAR(20)  NULL,
			address        TEXT         NULL,
			identityNumber VARCHAR(50)  NULL,
			profileName    VARCHAR(100) NULL,
			territoryName  VARCHAR(150) NULL,
			installerId    VARCHAR(191) NOT NULL,
			installerName  VARCHAR(150) NULL,
			installDate    DATE         NOT NULL,
			latitude       DECIMAL(10,7) NULL,
			longitude      DECIMAL(10,7) NULL,
			createdAt      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			INDEX idx_il_user (userId),
			INDEX idx_il_date (installDate),
			INDEX idx_il_installer (installerId)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		// ─── Phase 4: Automation — notification templates, provisioning status, alert rules, payment promises ───
		`CREATE TABLE IF NOT EXISTS notification_templates (
			id          VARCHAR(191) NOT NULL,
			eventType   VARCHAR(50)  NOT NULL,
			channel     VARCHAR(20)  NOT NULL,
			template    TEXT         NOT NULL,
			isEnabled   BOOLEAN      NOT NULL DEFAULT TRUE,
			createdAt   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			updatedAt   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			UNIQUE KEY uniq_nt_event_channel (eventType, channel)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS provisioning_status (
			id          VARCHAR(191) NOT NULL,
			userId      VARCHAR(191) NOT NULL,
			step        VARCHAR(50)  NOT NULL,
			status      VARCHAR(20)  NOT NULL DEFAULT 'pending',
			error       TEXT         NULL,
			startedAt   DATETIME(3)  NULL,
			completedAt DATETIME(3)  NULL,
			createdAt   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			INDEX idx_ps_user (userId),
			INDEX idx_ps_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS alert_rules (
			id            VARCHAR(191) NOT NULL,
			name          VARCHAR(100) NOT NULL,
			triggerEvent  VARCHAR(50)  NOT NULL,
			conditions    JSON         NOT NULL,
			actions       JSON         NOT NULL,
			isEnabled     BOOLEAN      NOT NULL DEFAULT TRUE,
			priority      INT          NOT NULL DEFAULT 0,
			createdAt     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			updatedAt     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS payment_promises (
			id             VARCHAR(191) NOT NULL,
			userId         VARCHAR(191) NOT NULL,
			username       VARCHAR(191) NOT NULL,
			promiseDate    DATE         NOT NULL,
			status         VARCHAR(20)  NOT NULL DEFAULT 'active',
			createdBy      VARCHAR(191) NOT NULL,
			createdByName  VARCHAR(150) NULL,
			notes          TEXT         NULL,
			createdAt      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			INDEX idx_pp_user (userId),
			INDEX idx_pp_status (status),
			INDEX idx_pp_date (promiseDate)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		// ─── Link pppoe_areas to territories (collector area assignment uses existing area data) ───
		`ALTER TABLE pppoe_areas ADD COLUMN territoryId VARCHAR(191) NULL`,
		`ALTER TABLE pppoe_areas ADD INDEX idx_pppoe_areas_territoryId (territoryId)`,

		// ─── Seed COLLECTOR role permissions (view customers, dashboard, invoices, sessions, notifications) ───
		// First, add COLLECTOR to the role ENUM in both tables (Prisma created them without COLLECTOR)
		`ALTER TABLE admin_users MODIFY COLUMN role ENUM('SUPER_ADMIN','FINANCE','CUSTOMER_SERVICE','TECHNICIAN','MARKETING','COLLECTOR','VIEWER') NOT NULL DEFAULT 'CUSTOMER_SERVICE'`,
		`ALTER TABLE role_permissions MODIFY COLUMN role ENUM('SUPER_ADMIN','FINANCE','CUSTOMER_SERVICE','TECHNICIAN','MARKETING','COLLECTOR','VIEWER') NOT NULL`,
		// Delete any bad rows with empty role (from previous failed inserts)
		`DELETE FROM role_permissions WHERE role = ''`,
		// Insert COLLECTOR permissions using INSERT IGNORE for idempotency
		`INSERT IGNORE INTO role_permissions (id, role, permissionId, createdAt)
		 SELECT UUID(), 'COLLECTOR', id, NOW(3) FROM permissions WHERE name = 'View Customers'`,
		`INSERT IGNORE INTO role_permissions (id, role, permissionId, createdAt)
		 SELECT UUID(), 'COLLECTOR', id, NOW(3) FROM permissions WHERE name = 'View Dashboard'`,
		`INSERT IGNORE INTO role_permissions (id, role, permissionId, createdAt)
		 SELECT UUID(), 'COLLECTOR', id, NOW(3) FROM permissions WHERE name = 'View Invoices'`,
		`INSERT IGNORE INTO role_permissions (id, role, permissionId, createdAt)
		 SELECT UUID(), 'COLLECTOR', id, NOW(3) FROM permissions WHERE name = 'View Sessions'`,
		`INSERT IGNORE INTO role_permissions (id, role, permissionId, createdAt)
		 SELECT UUID(), 'COLLECTOR', id, NOW(3) FROM permissions WHERE name = 'View Notifications'`,

		// ─── Remaining roadmap items: payment method edit count, API keys, PSB deadline, profile overrides, waiting list ───
		`ALTER TABLE payments ADD COLUMN paymentMethodEditCount INT NOT NULL DEFAULT 0`,

		`ALTER TABLE pppoe_users ADD COLUMN initialPaymentPending BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE pppoe_users ADD COLUMN psbDeadlineAt DATETIME(3) NULL`,
		`ALTER TABLE pppoe_users ADD INDEX idx_psb_deadline (psbDeadlineAt)`,

		`CREATE TABLE IF NOT EXISTS api_keys (
			id          VARCHAR(191) NOT NULL,
			keyHash     VARCHAR(255) NOT NULL,
			label       VARCHAR(100) NOT NULL,
			isActive    BOOLEAN      NOT NULL DEFAULT TRUE,
			createdAt   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			lastUsedAt  DATETIME(3)  NULL,
			PRIMARY KEY (id),
			INDEX idx_ak_hash (keyHash(255))
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS profile_router_map (
			id              VARCHAR(191) NOT NULL,
			profileId       VARCHAR(191) NOT NULL,
			routerId        VARCHAR(191) NOT NULL,
			mikrotikProfile VARCHAR(100) NOT NULL,
			createdAt       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			UNIQUE KEY uniq_prm_profile_router (profileId, routerId)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS waiting_list (
			id              VARCHAR(191) NOT NULL,
			fullname        VARCHAR(150) NOT NULL,
			phone           VARCHAR(20)  NULL,
			address         TEXT         NULL,
			identityNumber  VARCHAR(50)  NULL,
			ktpPhoto        TEXT         NULL,
			notes           TEXT         NULL,
			territoryId     VARCHAR(191) NULL,
			territoryAreaId VARCHAR(191) NULL,
			kelurahanKode   VARCHAR(20)  NULL,
			profileId       VARCHAR(191) NULL,
			sales           VARCHAR(100) NULL,
			latitude        DECIMAL(10,7) NULL,
			longitude       DECIMAL(10,7) NULL,
			status          VARCHAR(20)  NOT NULL DEFAULT 'waiting',
			createdBy       VARCHAR(191) NOT NULL,
			createdAt       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			updatedAt       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			INDEX idx_wl_status (status),
			INDEX idx_wl_territory (territoryId)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS waiting_list_assignments (
			id                  VARCHAR(191) NOT NULL,
			waitingListId       VARCHAR(191) NOT NULL,
			technicianUsername  VARCHAR(191) NOT NULL,
			assignedBy          VARCHAR(191) NOT NULL,
			assignedAt          DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			INDEX idx_wla_wl (waitingListId)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS ont_removal_tasks (
			id              VARCHAR(191) NOT NULL,
			userId          VARCHAR(191) NOT NULL,
			username        VARCHAR(191) NOT NULL,
			customerId      VARCHAR(20)  NULL,
			fullname        VARCHAR(150) NULL,
			address         TEXT         NULL,
			territoryName   VARCHAR(150) NULL,
			latitude        DECIMAL(10,7) NULL,
			longitude       DECIMAL(10,7) NULL,
			assignedTo      VARCHAR(191) NOT NULL,
			assignedBy      VARCHAR(191) NOT NULL,
			status          VARCHAR(20)  NOT NULL DEFAULT 'pending',
			proofPhoto      TEXT         NULL,
			notes           TEXT         NULL,
			cancelReason    TEXT         NULL,
			cancelledBy     VARCHAR(191) NULL,
			cancelledAt     DATETIME(3)  NULL,
			confirmedBy     VARCHAR(191) NULL,
			confirmedAt     DATETIME(3)  NULL,
			createdAt       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			INDEX idx_ort_status (status),
			INDEX idx_ort_assigned (assignedTo)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}
	for _, stmt := range statements {
		if _, err := sqlDB.Exec(stmt); err != nil {
			// Ignore duplicate column (1060) and duplicate key/index (1061) errors
			// since MySQL doesn't support ADD COLUMN IF NOT EXISTS
			if isIgnorableMigrationError(err) {
				continue
			}
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

// isIgnorableMigrationError returns true for MySQL errors that indicate
// a column or index already exists (1060 / 1061), which are safe to skip
// since we run migrations idempotently.
func isIgnorableMigrationError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "1060") || strings.Contains(msg, "1061") ||
		strings.Contains(msg, "Duplicate column") || strings.Contains(msg, "Duplicate key")
}

// convertDSN converts a Prisma-style mysql:// URL to a GORM DSN.
// Example: mysql://user:pass@localhost:3306/dbname
//
//	→ user:pass@tcp(localhost:3306)/dbname?parseTime=True&loc=Local&charset=utf8mb4
func convertDSN(url string) (string, error) {
	// Strip scheme
	const scheme = "mysql://"
	if len(url) <= len(scheme) {
		return "", fmt.Errorf("too short")
	}
	rest := url[len(scheme):]

	// user:pass@host:port/dbname[?params]
	atIdx := -1
	for i := len(rest) - 1; i >= 0; i-- {
		if rest[i] == '@' {
			atIdx = i
			break
		}
	}
	if atIdx < 0 {
		return "", fmt.Errorf("missing @ in URL")
	}
	userInfo := rest[:atIdx]
	hostDB := rest[atIdx+1:]

	// hostDB → host:port/dbname
	slashIdx := -1
	for i, c := range hostDB {
		if c == '/' {
			slashIdx = i
			break
		}
	}
	if slashIdx < 0 {
		return "", fmt.Errorf("missing / after host")
	}
	host := hostDB[:slashIdx]
	dbname := hostDB[slashIdx+1:]

	// Strip Prisma-specific query params (connection_limit, pool_timeout, socket_timeout, etc.)
	// These are not valid MySQL session variables. We append our own MySQL-compatible params below.
	if qIdx := strings.Index(dbname, "?"); qIdx >= 0 {
		dbname = dbname[:qIdx]
	}

	dsn := fmt.Sprintf("%s@tcp(%s)/%s?parseTime=True&loc=Local&charset=utf8mb4&collation=utf8mb4_unicode_ci", userInfo, host, dbname)
	return dsn, nil
}
