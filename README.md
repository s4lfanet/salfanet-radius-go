# SALFANET RADIUS - Billing System for ISP/RTRW.NET

Modern, full-stack billing & RADIUS management system for ISP/RTRW.NET with FreeRADIUS integration supporting PPPoE and Hotspot authentication.

> **Latest:** v2.54.31 — Backend ↔ Database ↔ Frontend consistency audit: removed legacy WhatsappReminderSetting model, fixed Settlement.Collector relation, aligned Go structs with Prisma schema (missing Company fields, Go-managed columns), fixed frontend VpnClient/VpnServer types, removed unused genieacs sync method. (Aug 10, 2026)

---

## 🤖 AI Development Assistant

**READ FIRST:** [docs/AI_PROJECT_MEMORY.md](docs/AI_PROJECT_MEMORY.md) — contains full architecture, VPS details, DB schema, known issues, and proven solutions.

---

## 🎯 Features

| Category | Key Capabilities |
|----------|-----------------|
| **RADIUS / Auth** | FreeRADIUS 3.0.26, PAP/CHAP/MS-CHAP, VPN L2TP/IPSec, PPPoE & Hotspot, CoA real-time speed/disconnect |
| **VPN Management** | MikroTik CHR via API, VPS built-in WireGuard & L2TP/IPsec peer management, configurable IP pool & gateway per protocol, auto-generated RouterOS scripts |
| **PPPoE Management** | Customer accounts, profile-based bandwidth, isolation, IP assignment, MikroTik auto-sync, foto KTP+instalasi via kamera HP, GPS otomatis |
| **Hotspot Voucher** | 8 code types, batch up to 25,000, agent distribution, auto-sync with RADIUS, print templates |
| **Billing** | Postpaid/prepaid invoices, auto-generation, payment reminders, balance/deposit, auto-renewal |
| **Payment** | Manual upload (bukti transfer), Midtrans/Xendit/Duitku gateway, approval workflow, 0–5 bank accounts |
| **Notifications** | WhatsApp (Fonnte/WAHA/GOWA/MPWA/Wablas/WABlast/**Kirimi.id**/**Baileys native**), Email SMTP, broadcast (outage/invoice/payment), webhook pesan masuk |
| **Agent/Reseller** | Balance-based voucher generation, commission tracking, sales stats |
| **Financial** | Income/expense tracking with categories, keuangan reconciliation |
| **Network (FTTH)** | OLT/ODC/ODP management, customer port assignment, network map, distance calculation |
| **GenieACS TR-069** | CPE/ONT management, device list with online/offline status (60-min threshold), auto-sync every 5 min with in-memory cache, WiFi config (SSID/password), device parameters (flattened tree), presets, provisions, virtual parameters, faults, files, config, tasks, auto-provision |
| **Isolation** | Auto-isolate expired customers, customizable WhatsApp/Email/HTML landing page templates |
| **Cron Jobs** | 20 automated background jobs (tsx runner via PM2 fork), history, distributed locking, manual trigger, GenieACS auto-sync |
| **Roles & Permissions** | 53 permissions, 6 roles (SUPER_ADMIN/FINANCE/CUSTOMER_SERVICE/TECHNICIAN/MARKETING/VIEWER/COLLECTOR), 5 portals (Admin/Customer/Agent/Technician + SuperAdmin) |
| **Activity Log** | Audit trail with auto-cleanup (30 days) |
| **Security** | Session timeout 30 min, idle warning, RBAC, HTTPS/SSL |
| **Bahasa** | Bahasa Indonesia (full) |
| **PWA** | Installable di semua portal (admin, customer, agent, technician), offline fallback, service worker cache |
| **Web Push** | VAPID-based browser push notifications, subscribe/unsubscribe toggle, admin broadcast |
| **System Update** | Update via SSH menggunakan `updater.sh`, tidak ada web-based update |
| **Mobile App** | Flutter customer portal (WiFi control, invoice, payment) |
| **Captive Portal** | IP-based identification, invoice display, payment redirect |
| **Collector Management** | Territory/collector assignment, area multi-select from Kelola Area, settlement reports |
| **WhatsApp Baileys** | Native WhatsApp gateway built-in VPS via `@whiskeysockets/baileys`, PM2 proses terpisah, scan QR langsung di admin panel, auto-reconnect |

---

## 📱 WhatsApp Baileys (Native Gateway)

Provider WhatsApp bawaan tanpa layanan pihak ketiga. Berjalan sebagai proses PM2 terpisah (`salfanet-wa`) di VPS.

### Setup

Provider Baileys otomatis di-setup saat menjalankan `updater.sh`. Tidak ada konfigurasi tambahan.

```bash
# Cek status wa-service
pm2 status
pm2 logs salfanet-wa --lines 20
```

### Cara Pakai

1. Buka **Admin → Pengaturan → WhatsApp → Penyedia**
2. Klik **+ Tambah Provider**, pilih tipe **Baileys**
3. Klik **QR Code** → scan dengan HP (WhatsApp → Linked Devices)
4. Setelah scan berhasil, modal menampilkan centang hijau konfirmasi
5. Provider siap digunakan untuk kirim notifikasi

### PM2 Processes

| Process | Mode | Port | Purpose |
|---------|------|------|---------|
| `salfanet-radius` | fork | 3000 | Next.js app |
| `salfanet-wa` | fork | 4000 (internal) | Baileys WA service |
| `salfanet-cron` | fork | — | Background jobs |

### Auth Session

Session WhatsApp tersimpan di `/var/data/salfanet/baileys_auth/` dan persist meski PM2 restart. Untuk logout/scan ulang, klik **Restart Session** di admin panel.

---

## 🚀 Tech Stack

| Component | Technology |
|-----------|------------|
| Framework | Next.js 16 (App Router, standalone output) |
| Language | TypeScript + Go |
| Styling | Tailwind CSS |
| Database | MySQL 8.0 + Prisma ORM (frontend) + GORM (Go backend) |
| RADIUS | FreeRADIUS 3.0.26 |
| API Backend | Go (Fiber) — port 8080, systemd `salfanet-api.service` |
| Cache | Redis 6.0+ (HybridCache: Redis primary + memory fallback) |
| Process Manager | PM2 (fork mode, 3 processes) + systemd |
| Session Tracking | FreeRADIUS radacct (real-time) |
| Maps | Leaflet / OpenStreetMap |

---

## 🏗️ Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Browser     │────▶│   Nginx      │────▶│  Next.js    │
│  (Admin/     │     │  (reverse    │     │  (port 3000)│
│   Customer/  │     │   proxy)     │     │  PM2 fork   │
│   Agent/     │     │              │     └──────┬──────┘
│   Technician)│     │              │            │
└─────────────┘     │              │     ┌──────▼──────┐
                     │              │────▶│  Go API     │
                     │              │     │  (port 8080)│
                     │              │     │  systemd    │
                     │              │     └──────┬──────┘
                     │              │            │
                     │              │     ┌──────▼──────┐
                     │              │     │  Redis 6.0  │
                     │              │     │  (cache +   │
                     │              │     │   rate limit)│
                     │              │     └─────────────┘
                     │              │            │
                     │              │     ┌──────▼──────┐
                     │              │     │  MySQL 8.0  │
                     │              │     │  (Prisma +  │
                     │              │     │   GORM)     │
                     │              │     └─────────────┘
                     │              │            │
                     │              │     ┌──────▼──────┐
                     │              │────▶│ FreeRADIUS  │
                     │              │     │  (port 1812)│
                     └──────────────┘     └─────────────┘
                                          ┌─────────────┐
                                          │  WA Service │
                                          │  (Baileys)  │
                                          │  PM2 fork   │
                                          └─────────────┘
```

### Dual ORM Architecture

The project uses **two ORMs** that share the same MySQL database:

- **Prisma** (Node.js/TypeScript) — manages schema migrations via `npx prisma db push`. Owns most tables (users, customers, invoices, etc.).
- **GORM** (Go) — manages Go-specific tables via `runMigrations()` in `internal/db/db.go`. These are tables not covered by Prisma (cron history, backup history, telegram settings, etc.).

**Important:** When adding a new field to a Go model, ensure the corresponding column exists in the database. GORM does not auto-migrate — you must add an `ALTER TABLE` statement to `internal/db/db.go`.

### Authentication

The Go API uses `CombinedAuthMiddleware` which supports two auth methods:
1. **JWT Bearer token** — for mobile/API clients
2. **NextAuth session cookie** — for admin panel browser requests (validated by calling Next.js `/api/auth/session` internally)

This allows Nginx to route all `/api/` traffic to the Go backend without the frontend needing to change how it authenticates.

---

## 📁 Project Structure

```
salfanet-radius/
├── src/
│   ├── app/
│   │   ├── admin/          # Admin panel
│   │   ├── agent/          # Agent/reseller portal
│   │   ├── api/            # API route handlers
│   │   ├── customer/       # Customer self-service portal
│   │   └── technician/     # Technician portal
│   ├── server/             # DB, services, jobs, cache, auth
│   ├── features/           # Vertical slices (queries, schemas, types)
│   ├── components/         # Shared React components
│   ├── locales/            # i18n translations (id, en)
│   └── types/              # Shared TypeScript types
├── prisma/
│   ├── schema.prisma       # Database schema (~45 models)
│   └── seeds/              # Seed scripts
├── freeradius-config/      # FreeRADIUS config (deployed by installer)
├── vps-install/            # One-command VPS installer scripts
├── production/             # PM2 & Nginx config templates
├── mobile-app/             # Flutter customer app
├── scripts/                # Utility & tuning scripts
└── docs/                   # Documentation & AI memory
```

---

## ⚙️ Installation

### Prerequisites

- Ubuntu 22.04+ / Debian 12+ VPS
- Root access
- 2GB RAM minimum (4GB recommended)
- 20GB disk space
- Go 1.23+ (for building API binary)
- Node.js 20+ (for building Next.js frontend)
- MySQL 8.0
- FreeRADIUS 3.0.26

### Metode 1 — Git Clone (Recommended)

```bash
ssh root@YOUR_VPS_IP

git clone https://github.com/s4lfanet/salfanet-radius-go.git /root/salfanet-radius
cd /root/salfanet-radius
bash vps-install/vps-installer.sh
```

Installer akan berjalan **interaktif** — mendeteksi environment otomatis, memandu konfigurasi, lalu menjalankan semua step.

#### Unattended install (tanpa prompt)

```bash
bash vps-install/vps-installer.sh \
  --env vps \
  --domain yourdomain.com \
  --db-pass YourDbPassword \
  --unattended
```

---

### Metode 2 — Upload Manual via SCP (Tanpa Akses Internet di Server)

```bash
# Di komputer LOKAL — buat zip dari git HEAD
git archive --format=zip HEAD -o salfanet-fresh.zip

# Upload ke VPS
scp salfanet-fresh.zip root@YOUR_VPS_IP:/root/salfanet-fresh.zip

# SSH ke VPS, unzip, lalu jalankan installer
ssh root@YOUR_VPS_IP
mkdir -p /root/salfanet-radius && cd /root/salfanet-radius
unzip -q /root/salfanet-fresh.zip
bash vps-install/vps-installer.sh
```

---

### Environment yang Didukung

| Environment | Flag | Akses |
|------------|------|-------|
| **Public VPS** (DigitalOcean, Vultr, Hetzner, AWS) | `--env vps` | Internet |
| **Proxmox LXC** | `--env lxc` | LAN/VLAN |
| **Proxmox VM / VirtualBox** | `--env vm` | LAN |
| **Bare Metal / Server Fisik** | `--env bare` | LAN |

```bash
# Contoh: paksa environment + IP
bash vps-install/vps-installer.sh --env lxc --ip 192.168.1.50

# Semua flag yang tersedia:
# --env vps|lxc|vm|bare   Tipe environment (default: auto-detect)
# --ip IP                 IP server (default: auto-detect)
# --domain DOMAIN         Domain untuk SSL/nginx (lewati prompt)
# --db-pass PASS          Password database MySQL
# --unattended            Non-interactive, pakai semua default
```

---

### Environment Configuration (.env)

The project uses two `.env` files:

| File | Location | Purpose |
|------|----------|---------|
| `.env` (root) | `/var/www/salfanet-radius/.env` | Next.js frontend (Prisma, NextAuth, WA service) |
| `.env` (Go) | `/var/www/salfanet-radius/.env` | Go API (shared with root .env) |

Key environment variables:

```bash
# Database
DATABASE_URL="mysql://salfanet_user:PASSWORD@localhost:3306/salfanet_radius"

# Go API
PORT=8080
JWT_SECRET=your-secret-key
CORS_ORIGINS=http://localhost:3000
APP_BASE_URL=http://localhost:3000
APP_TIMEZONE=Asia/Jakarta
UPLOAD_DIR=/var/data/salfanet/uploads

# NextAuth
NEXTAUTH_SECRET=your-nextauth-secret
NEXTAUTH_URL=http://localhost:3000

# WhatsApp Service
WA_SERVICE_URL=http://127.0.0.1:4000

# Payment Gateways (optional)
MIDTRANS_SERVER_KEY=
XENDIT_API_KEY=
DUITKU_API_KEY=
TRIPAY_API_KEY=
TRIPAY_PRIVATE_KEY=

# Web Push (optional)
VAPID_PUBLIC_KEY=
VAPID_PRIVATE_KEY=

# Redis Cache
REDIS_ENABLED=true
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_PREFIX=salfanet
```

### Database Setup

The database uses **dual migration systems**:

1. **Prisma migrations** (frontend-managed tables):
```bash
cd /var/www/salfanet-radius
npx prisma db push
```

2. **Go migrations** (Go-managed tables) — run automatically on API startup:
- Tables: `cron_history`, `backup_history`, `telegram_backup_settings`, `payment_gateways`, `cloudflare_settings`, `map_settings`, `promise_payments`, `installation_logs`, etc.
- Schema additions: `ALTER TABLE` statements for columns needed by Go models but not in Prisma schema (e.g., `qrisDeviceKey` on `companies`, `territoryId` on `pppoe_areas`).
- Migrations are idempotent — duplicate column/key errors are silently ignored.

### Services & Workers

| Service | Manager | Port | Purpose |
|---------|---------|------|---------|
| `salfanet-radius` | PM2 | 3000 | Next.js frontend |
| `salfanet-api` | systemd | 8080 | Go API backend |
| `redis-server` | systemd | 6379 | Redis cache & rate limiting |
| `salfanet-wa` | PM2 | 4000 | WhatsApp Baileys service |
| `salfanet-cron` | PM2 | — | Background cron jobs |
| `nginx` | systemd | 80/443 | Reverse proxy |
| `freeradius` | systemd | 1812/1813 | RADIUS authentication |

### Updating Existing Installation

Cara paling aman. **Semua data upload (logo, foto KTP pelanggan, bukti bayar) otomatis dipreservasi.**

```bash
bash /var/www/salfanet-radius/vps-install/updater.sh
```

Atau update dari branch terbaru secara manual:

```bash
cd /var/www/salfanet-radius
git pull origin master
npm install --omit=dev
npx prisma db push
npm run build
pm2 reload all
```

Lihat detail lengkap di [vps-install/README.md](vps-install/README.md).

---

### Data yang Aman Saat Update

| Data | Status |
|------|--------|
| Logo perusahaan (`public/uploads/logos/`) | ✅ Dipreservasi |
| Foto KTP & dokumen pelanggan | ✅ Dipreservasi |
| Bukti pembayaran | ✅ Dipreservasi |
| File `.env` (database, secrets) | ✅ Tidak disentuh |
| **Database MySQL (semua data pelanggan)** | ✅ Tidak disentuh |

---

### Default Credentials

| | |
|--|--|
| Admin URL | `http://YOUR_VPS_IP/admin/login` |
| Username | `superadmin` |
| Password | `admin123` |

⚠️ **Ganti password segera setelah login pertama!**

---

## 🔌 FreeRADIUS

Key config files at `/etc/freeradius/3.0/`:

| File | Purpose |
|------|---------|
| `mods-enabled/sql` | MySQL connection for user auth |
| `mods-enabled/rest` | REST API for voucher management |
| `sites-enabled/default` | Main auth logic (PPPoE realm support) |
| `clients.conf` | NAS/router clients (+ `$INCLUDE clients.d/`) |
| `sites-enabled/coa` | CoA/Disconnect-Request virtual server |

Config backup in `freeradius-config/` is auto-deployed by the installer.

### Auth Flow

**PPPoE:** `MikroTik → FreeRADIUS → MySQL (radcheck/radusergroup/radgroupreply)` → Access-Accept with Mikrotik-Rate-Limit

**Hotspot Voucher:** Same RADIUS path + `REST /api/radius/post-auth` → sets firstLoginAt, expiresAt, syncs keuangan

### RADIUS Tables

| Table | Purpose |
|-------|---------|
| `radcheck` | User credentials |
| `radreply` | User-specific reply attrs |
| `radusergroup` | User → Group mapping |
| `radgroupreply` | Group reply (bandwidth, session timeout) |
| `radacct` | Session accounting |
| `nas` | NAS/Router clients (dynamic) |

---

## ⏰ Cron Jobs (16 automated)

| Job | Schedule | Function |
|-----|----------|----------|
| Voucher Sync | Every 5 min | Sync voucher status with RADIUS |
| Disconnect Sessions | Every 5 min | CoA disconnect expired vouchers |
| Auto Isolir (PPPoE) | Every hour | Suspend overdue customers |
| FreeRADIUS Health | Every 5 min | Auto-restart if down |
| PPPoE Session Sync | Every 10 min | Sync radacct sessions |
| Agent Sales | Daily 1 AM | Update sales statistics |
| Invoice Generate | Daily 2 AM | Generate monthly invoices |
| Activity Log Cleanup | Daily 2 AM | Delete logs >30 days |
| Invoice Reminder | Daily 8 AM | Send payment reminders |
| Invoice Status | Daily 9 AM | Mark overdue invoices |
| Notification Check | Every 10 min | Process notification queue |
| Auto Renewal | Daily 8 AM | Prepaid auto-renew from balance |
| Webhook Log Cleanup | Daily 3 AM | Delete webhook logs >30 days |
| Session Monitor | Every 5 min | Security session monitoring |
| Cron History Cleanup | Daily 4 AM | Keep last 50 per job type |
| Suspend Check | Every hour | Activate/restore suspend requests |

All jobs can be triggered manually from **Settings → Cron** in the admin panel.

---

## � Android APK Builder

Buat APK Android (WebView wrapper) untuk 4 portal langsung di server VPS — tanpa GitHub Actions, tanpa Android Studio.

### 1) Setup Android SDK (satu kali via SSH)

```bash
apt-get update && apt-get install -y openjdk-17-jdk wget unzip && \
mkdir -p /opt/android/cmdline-tools && \
wget -q https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip -O /tmp/cmdtools.zip && \
unzip -q /tmp/cmdtools.zip -d /opt/android/cmdline-tools && \
mv /opt/android/cmdline-tools/cmdline-tools /opt/android/cmdline-tools/latest && \
yes | /opt/android/cmdline-tools/latest/bin/sdkmanager --licenses && \
/opt/android/cmdline-tools/latest/bin/sdkmanager "platforms;android-34" "build-tools;34.0.0" && \
echo 'export ANDROID_HOME=/opt/android' >> /etc/environment && \
echo 'Selesai!'
```

> **Perkiraan waktu:** ~5–10 menit (download ~500MB). Disk yang dibutuhkan: ~2GB.

### 2) Build APK via Admin Panel

Buka **Admin → Download Aplikasi Android** → klik **Build APK** pada role yang diinginkan.

- Build berjalan di background (tidak timeout meski butuh beberapa menit)
- Status diperbarui otomatis setiap 3 detik
- Setelah selesai, tombol **Download APK** muncul

### 3) Build via API (opsional)

```bash
# Cek environment
curl http://YOUR_VPS/api/admin/apk/trigger

# Mulai build (role: admin | customer | technician | agent)
curl -X POST http://YOUR_VPS/api/admin/apk/trigger?role=customer \
  -H "Cookie: next-auth.session-token=..."

# Cek status
curl http://YOUR_VPS/api/admin/apk/status?role=customer

# Download APK
curl -OJ http://YOUR_VPS/api/admin/apk/file?role=customer \
  -H "Cookie: next-auth.session-token=..."
```

### Storage APK

| Path | Keterangan |
|------|------------|
| `/var/data/salfanet/apk/{role}/app.apk` | File APK hasil build |
| `/var/data/salfanet/apk/{role}/status.json` | Status & metadata build |
| `/var/data/salfanet/apk/{role}/build.log` | Log Gradle |
| `/var/data/salfanet/gradle-cache` | Cache Gradle (mempercepat build berikutnya) |

### Paket Aplikasi

| Role | Package ID | Warna |
|------|-----------|-------|
| Admin | `net.salfanet.admin` | Biru |
| Customer | `net.salfanet.customer` | Cyan |
| Technician | `net.salfanet.technician` | Hijau |
| Agent | `net.salfanet.agent` | Ungu |

---

## �🛠️ Common Commands

```bash
# PM2
pm2 status ; pm2 logs salfanet-radius
pm2 restart ecosystem.config.js --update-env

# FreeRADIUS
systemctl restart freeradius
freeradius -XC    # Test config
radtest 'user@realm' password 127.0.0.1 0 testing123

# Database
mysql -u salfanet_user -psalfanetradius123 salfanet_radius
mysqldump -u salfanet_user -psalfanetradius123 salfanet_radius > backup.sql
```

---

## 🧯 Troubleshooting Cepat

### 1) Website tidak bisa diakses dari IP VPS

Jika `Nginx` dan app sudah jalan di server tapi dari internet tetap tidak bisa akses, biasanya masalah ada di layer jaringan (NAT/forwarding/firewall external), bukan di aplikasi.

```bash
# Di VM/VPS guest
ss -tulpn | grep -E ':80|:443|:3000'
curl -I http://127.0.0.1:3000
curl -I http://127.0.0.1
systemctl status nginx --no-pager
pm2 status
```

Jika semua check local di atas OK, cek mapping di host Proxmox/router/cloud firewall:

1. `Public:2020 -> VM:22` (SSH)
2. `Public:80 -> VM:80` (HTTP)
3. `Public:443 -> VM:443` (HTTPS)

Catatan: `IP:2020` adalah port SSH, bukan URL web aplikasi.

### 2) PM2 jalan tapi web tetap blank/error

```bash
pm2 status
pm2 logs salfanet-radius --lines 100
cd /var/www/salfanet-radius
npm run build
pm2 restart ecosystem.config.js --update-env
```

### 4) Jalankan diagnosa Nginx otomatis dari installer

Installer Nginx terbaru menambahkan self-check internal (`127.0.0.1:3000`, `127.0.0.1`) dan best-effort check publik (HTTP/HTTPS).

```bash
cd /var/www/salfanet-radius
bash vps-install/install-nginx.sh
```

Jika warning menunjukkan HTTP publik tidak reachable, fokus perbaikan di NAT/port-forward/security-group, bukan di Next.js.

---

## 🔐 Security

```bash
# Firewall
ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp
ufw allow 1812/udp && ufw allow 1813/udp && ufw allow 3799/udp
```

1. Change default admin password on first login
2. Change MySQL passwords in `.env`
3. Configure SSL (Let's Encrypt or Cloudflare)
4. Enable UFW

---

## 📡 CoA (Change of Authorization)

Sends real-time speed/disconnect commands to MikroTik without dropping PPPoE connections.

**MikroTik requirement:** `/radius incoming set accept=yes port=3799`

**API:** `POST /api/radius/coa` — actions: `disconnect`, `update`, `sync-profile`, `test`

Auto-triggered when: PPPoE profile speed is edited (syncs all active sessions).

---

## 📲 WhatsApp Providers

| Provider | Base URL | Auth |
|----------|----------|------|
| Fonnte | `https://api.fonnte.com/send` | Token |
| WAHA | `http://IP:PORT` | API Key |
| GOWA | `http://IP:PORT` | `user:pass` |
| MPWA | `http://IP:PORT` | API Key |
| Wablas | `https://pati.wablas.com` | Token |

---

## ⏱️ Timezone

| Layer | Timezone | Note |
|-------|----------|------|
| Database (Prisma) | UTC | Prisma default |
| FreeRADIUS | WIB (UTC+7) | Server local time |
| PM2 env | WIB | `TZ: 'Asia/Jakarta'` in ecosystem.config.js |
| API / Frontend | WIB | Auto-converts UTC ↔ WIB |

For WITA (UTC+8) or WIT (UTC+9): change `TZ` in `.env`, `ecosystem.config.js`, and `src/lib/timezone.ts`.

---

## 📋 Admin Modules

Dashboard · PPPoE · Hotspot · Agent · Invoice · Payment · Keuangan · Sessions · WhatsApp · Network (OLT/ODC/ODP) · GenieACS · Settings

**Roles:** SUPER_ADMIN · FINANCE · CUSTOMER_SERVICE · TECHNICIAN · MARKETING · VIEWER · COLLECTOR

---

## 📝 Changelog

Bagian ini otomatis sinkron dari `CHANGELOG.md` saat file changelog berubah di GitHub.

<!-- AUTO-CHANGELOG:START -->

### v2.54.31 — 2026-08-10

### Fixed — Backend ↔ Database ↔ Frontend Consistency Audit
- **WhatsappReminderSetting legacy model conflict** (`internal/db/models/extra.go`, `internal/cron/scheduler.go`, `internal/api/handlers/whatsapp_crud.go`, `internal/api/router.go`) — Removed deprecated `WhatsappReminderSetting` struct that conflicted with `WhatsappGlobalSettings` on the same `whatsapp_reminder_settings` table. Updated cron scheduler to use `WhatsappGlobalSettings` with `reminderDays` JSON array parsing. Removed legacy CRUD handlers and route registrations for `-ext` endpoints.
- **Settlement.Collector relation** (`internal/db/models/territory.go`) — Changed `Collector` field from `*User` to `*AdminUser` to correctly reference `admin_users` table.
- **Radcheck.Op default value** (`internal/db/models/models.go`) — Fixed invalid GORM tag `default::=` to `default:=`.
- **PppoeCustomer.CustomerID type** (`internal/db/models/models.go`) — Changed varchar length from 20 to 10 to match Prisma schema `@db.VarChar(10)`.
- **Missing Company fields in Go struct** (`internal/db/models/models.go`) — Added `IsolationWhatsappTemplateId`, `IsolationEmailTemplateId`, `IsolationHtmlTemplateId`, `PppoeRenewalAnytime`, `PppoeRenewalDaysBefore`, `ReferralRewardType`, `ReferralRewardBoth`, `ReferralReferredAmount` to match Prisma schema.
- **Go-managed columns missing from Prisma schema** (`prisma/schema.prisma`) — Added `territoryId` to `pppoeArea`; `territoryId`, `territoryAreaId`, `initialPaymentPending`, `psbDeadlineAt` to `pppoeUser`; `discountAmount`, `discountReason`, `originalAmount`, `cancelledAt`, `cancelledBy`, `cancelReason` to `invoice`; `qrisDeviceKey` to `company`; `paymentMethodEditCount` to `payment`. Prevents `prisma db push` from dropping Go-managed columns.
- **GetRouter response missing fields** (`internal/api/handlers/network_ext.go`) — Added `latitude`, `longitude`, and `updatedAt` to the router detail response for API consistency.
- **Frontend VpnClient/VpnServer types** (`src/features/network/types.ts`) — Fixed `VpnClient` type: removed non-existent `privateKey`, added `endpoint`, `description`, `updatedAt`. Fixed `VpnServer` type: added `description` field. Both now match Go struct definitions exactly.
- **Unused genieacs sync method** (`internal/cron/genieacs_sync.go`) — Removed dead `jobGenieacsSyncSingle` method (never called; `RefreshDevice` handler does its own API call).

### Files
- `internal/db/models/extra.go` — **EDIT** — Removed legacy `WhatsappReminderSetting` struct
- `internal/db/models/territory.go` — **EDIT** — Fixed `Settlement.Collector` relation type
- `internal/db/models/models.go` — **EDIT** — Fixed `Radcheck.Op` default, `PppoeCustomer.CustomerID` type, added missing Company fields
- `internal/cron/scheduler.go` — **EDIT** — Use `WhatsappGlobalSettings` instead of `WhatsappReminderSetting`
- `internal/api/handlers/whatsapp_crud.go` — **EDIT** — Removed legacy reminder settings CRUD methods
- `internal/api/router.go` — **EDIT** — Removed legacy `-ext` reminder settings routes
- `internal/api/handlers/network_ext.go` — **EDIT** — Added missing fields to `GetRouter` response
- `internal/cron/genieacs_sync.go` — **EDIT** — Removed unused `jobGenieacsSyncSingle` method
- `prisma/schema.prisma` — **EDIT** — Added Go-managed columns to prevent schema drift
- `src/features/network/types.ts` — **EDIT** — Fixed `VpnClient` and `VpnServer` types to match Go structs
- `src/components/network/SplicePointsSection.tsx` — **EDIT** — TSC fix

### v2.54.30 — 2026-08-10

### Fixed — GenieACS Timezone & Telegram Cron Timezone
- **GenieACS timestamps 7-hour offset** (`src/lib/timezone.ts`, `src/app/admin/genieacs/devices/page.tsx`, `src/app/admin/genieacs/devices/[deviceId]/page.tsx`, `src/app/admin/genieacs/tasks/page.tsx`, `src/app/admin/genieacs/faults/page.tsx`) — GenieACS returns `_lastInform` and task `timestamp` as real UTC ISO 8601 strings (with `Z` suffix). The frontend used `formatWIB()` which treats UTC values as already-WIB (Prisma/MySQL convention), causing no timezone conversion and a 7-hour display offset. Added new `formatFromUTC()` function in `src/lib/timezone.ts` that performs real UTC→system timezone conversion using `date-fns-tz/formatInTimeZone`. Applied to all GenieACS timestamp displays: device list `lastInform`, device detail `lastInform` + `task.timestamp`, tasks page (card + table views), faults page (was raw string, now properly formatted).
- **Telegram cron hardcoded timezone** (`internal/cron/telegram_cron.go`) — Replaced `time.Now().Format("2006-01-02 15:04:05 WIB")` with `tzutil.FormatNow("2006-01-02 15:04:05 WIB")` for consistent timezone from company settings. Added `tzutil` import.
- **VPN server page `toLocaleString`** (`src/app/admin/network/vpn-server/page.tsx`) — Replaced browser-local `toLocaleString` with `formatWIB` for `peer.lastHandshake` display, ensuring consistent timezone formatting.

### Added — GenieACS Auto-Sync Cronjob & In-Memory Cache
- **GenieACS device cache** (`internal/cache/genieacs_cache.go`) — New specialized in-memory cache with 5-minute TTL for GenieACS device data. Thread-safe with `sync.RWMutex`. Stores flat device list and per-device map for O(1) lookups. Cache invalidation on manual refresh. More efficient than Redis for structured data (no serialization overhead).
- **GenieACS auto-sync cronjob** (`internal/cron/genieacs_sync.go`) — New cronjob runs every 5 minutes, fetches all devices from GenieACS NBI with projection (`_id,_lastInform,_lastBoot,_registered,_deviceId,VirtualParameters`), maps to flat structure, and populates cache. Also supports single-device sync (`jobGenieacsSyncSingle`) for immediate cache update after manual refresh. Standalone `mapGenieacsDevice` function (no fiber dependency).
- **Scheduler registration** (`internal/cron/scheduler.go`) — Registered `genieacs_sync` job at `0 */5 * * * *` (every 5 min). Added to `TriggerJob` switch for manual invocation. Total jobs: 20.

### Improved — GenieACS API Performance
- **ListDevices/GetDevice cache-first** (`internal/api/handlers/settings_genieacs.go`) — `ListDevices` and `GetDevice` now serve from cache first. On cache miss (cold start or expired TTL), falls back to direct GenieACS NBI call and populates cache. `RefreshDevice` invalidates cache for the refreshed device so next read fetches fresh data. Response includes `cached` boolean and `updatedAt` timestamp. This eliminates repeated GenieACS HTTP calls on every page load, reducing CPU/RAM usage.
- **Status/rxPower/pppoeStatus always actual** — With cron auto-sync every 5 minutes, device status (Online/Offline), rxPower, PPPoE username/IP, PON mode, and uptime are always fresh without manual intervention.

### Files
- `src/lib/timezone.ts` — **EDIT** — Added `formatFromUTC()` function for real UTC→timezone conversion
- `src/app/admin/genieacs/devices/page.tsx` — **EDIT** — `lastInform` uses `formatFromUTC` instead of `formatWIB`
- `src/app/admin/genieacs/devices/[deviceId]/page.tsx` — **EDIT** — `lastInform` + `task.timestamp` use `formatFromUTC`
- `src/app/admin/genieacs/tasks/page.tsx` — **EDIT** — Task timestamps (card + table) use `formatFromUTC`
- `src/app/admin/genieacs/faults/page.tsx` — **EDIT** — Fault timestamp uses `formatFromUTC` (was raw string)
- `src/app/admin/network/vpn-server/page.tsx` — **EDIT** — `peer.lastHandshake` uses `formatWIB` instead of `toLocaleString`
- `internal/cron/telegram_cron.go` — **EDIT** — Replaced `time.Now().Format` with `tzutil.FormatNow`, added `tzutil` import
- `internal/cache/genieacs_cache.go` — **NEW** — In-memory cache for GenieACS device data with 5-min TTL
- `internal/cron/genieacs_sync.go` — **NEW** — Auto-sync cronjob + standalone `mapGenieacsDevice`
- `internal/cron/scheduler.go` — **EDIT** — Registered `genieacs_sync` job, added to `TriggerJob`
- `internal/api/handlers/settings_genieacs.go` — **EDIT** — Cache-first `ListDevices`/`GetDevice`, cache invalidation on `RefreshDevice`, added `cache` import

### v2.54.29 — 2026-08-09

### Fixed — WiFi Configuration Audit & GenieACS Query Fixes
- **WiFi security mapping mismatch** (`internal/api/handlers/genieacs.go`) — Huawei ONT/ONU devices require `BeaconType` as the primary security indicator. For "None" (Open) security, `IEEE11iAuthenticationMode` and `IEEE11iEncryptionModes` must NOT be sent (Huawei rejects with faultCode 9007). Now sets `BasicAuthenticationMode=None` and `BasicEncryptionModes=None` for Open security to clear previous security settings. Security mapping: `None→BeaconType=None`, `WPA-PSK→WPA+TKIP`, `WPA2-PSK→11i+AES`, `WPA/WPA2 Mixed→WPAand11i+TKIP+AES`.
- **Security display in DeviceDetail** (`internal/api/handlers/settings_genieacs.go`) — Was showing raw `IEEE11iAuthenticationMode` (e.g. "PSKAuthentication") instead of user-friendly label. Now maps `BeaconType` to security labels: `11i→WPA2-PSK`, `WPA→WPA-PSK`, `WPAand11i→WPA-WPA2-PSK`, `None→None`, `Basic→WEP`.
- **GenieACS device queries returning all devices** (`internal/api/handlers/genieacs_ext.go`) — 6 handlers still used broken `?_id=xxx` query format. GenieACS NBI ignores `_id` as a query param and returns all devices. Fixed all to use `genieacsIDQuery()` with proper `?query={"_id":"xxx"}` format.
- **Connected devices display** (`internal/api/handlers/settings_genieacs.go`) — Huawei devices don't populate `Layer2Interface._value`. Switched to `InterfaceType` for WiFi/LAN detection (802.11/WiFi). Default `ssidIndex=1` for WiFi if not parseable. Default `active=true` if IP present. Added `X_HW_RSSI`/`X_HW_RSI` for signal strength.
- **WiFi update logging** (`internal/api/handlers/genieacs.go`) — Added zerolog logging for setParameterValues task payload (device, wlanIndex, ssid, securityMode, beaconType, paramCount) and GenieACS response status code. Error responses now logged with full details.

### Added — Bulk Delete Faults & Redis Cache
- **Bulk delete faults** (`internal/api/handlers/genieacs_ext.go`, `internal/api/router.go`, `src/app/admin/genieacs/faults/page.tsx`) — New `POST /api/genieacs/faults/bulk-delete` endpoint accepts `{ids: [...]}` array. Frontend faults page now has select-all checkbox, per-row checkboxes, and "Delete N selected" button with confirmation dialog.
- **Redis cache implementation** (`internal/cache/redis.go`, `internal/config/config.go`, `cmd/server/main.go`, `internal/api/router.go`, `internal/api/handlers/scaling_handler.go`) — Full Redis integration with `HybridCache` (Redis primary + Memory fallback). Auto-fallback to memory-only if Redis unavailable. New env vars: `REDIS_ENABLED`, `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`, `REDIS_PREFIX`. Redis-based sliding window `RateLimiter` using Sorted Sets. `CacheInterface` abstraction for swappable cache backends. Cache stats endpoint now reports Redis mode.

### Files
- `internal/api/handlers/genieacs.go` — **EDIT** — WiFi security mapping fix, BasicAuth/BasicEnc for None, detailed logging
- `internal/api/handlers/settings_genieacs.go` — **EDIT** — Security display from BeaconType, connected devices parsing fix
- `internal/api/handlers/genieacs_ext.go` — **EDIT** — All `?_id=` queries replaced with `genieacsIDQuery()`, added `DeleteFaultsBulk` handler
- `internal/api/router.go` — **EDIT** — Added `POST /faults/bulk-delete` route, HybridCache wiring
- `internal/api/handlers/scaling_handler.go` — **EDIT** — `CacheInterface` abstraction, Redis mode in rate-limit status
- `internal/cache/redis.go` — **NEW** — `RedisCache`, `HybridCache`, `RateLimiter` implementations
- `internal/config/config.go` — **EDIT** — Added Redis config fields
- `cmd/server/main.go` — **EDIT** — Redis/HybridCache initialization, graceful shutdown
- `src/app/admin/genieacs/faults/page.tsx` — **EDIT** — Bulk delete UI with checkboxes

### v2.54.28 — 2026-08-09

### Fixed — GenieACS API Audit & Device Status Threshold
- **GenieACS path-based GET returning 405** (`internal/api/handlers/genieacs_ext.go`) — Root cause: GenieACS NBI API does not support path-based GET for single resources (e.g. `GET /presets/:id`). All such requests returned 405 Method Not Allowed. Fixed by switching to query-based filtering: `GET /presets?_id=:id`. Affected endpoints: GetDevice, DeviceAllParameters, GetDeviceParameters, GetDeviceWifi, GetPreset, GetProvision.
- **List endpoints returning raw arrays** (`internal/api/handlers/genieacs_ext.go`) — Frontend expected `{success: true, data: [...]}` but backend returned raw JSON arrays. Fixed all list endpoints (presets, provisions, faults, files, config, virtual-parameters) to wrap responses in standard format.
- **Single-item GET returning raw object** (`internal/api/handlers/genieacs_ext.go`) — Frontend expected `{data: item}` but backend returned raw object. Fixed all single-item GET endpoints to wrap in `{data: item}` format.
- **Device detail page response format** (`src/app/admin/genieacs/devices/[deviceId]/page.tsx`) — Page expected `devJson.data` and `taskJson.data` but API returned different structure. Fixed to match frontend expectations.
- **GetDeviceTasks query filtering** (`internal/api/handlers/genieacs_ext.go`) — GenieACS `/tasks` endpoint does not support query filtering by device. Fixed by fetching all tasks and filtering in Go by `device` field matching `deviceId`.
- **DELETE body parsing** (`internal/api/handlers/genieacs_ext.go`, `internal/api/router.go`) — Frontend sends DELETE requests with JSON body `{id: "..."}` but backend expected path params. Fixed faults, config, and files DELETE handlers to read IDs from request body. Added `DELETE /faults` route (body-based) alongside existing `/faults/:faultId`.
- **DeviceAllParameters raw device response** (`internal/api/handlers/settings_genieacs.go`) — Endpoint returned raw nested device object. Fixed to flatten parameter tree into a flat list with metadata (writable, object, timestamp fields).
- **Status case mismatch** (`internal/api/handlers/settings_genieacs.go`) — Backend returned lowercase `online`/`offline` but frontend expected capitalized `Online`/`Offline`. Fixed `mapDevice` to return capitalized values.
- **Device status threshold too strict** (`internal/api/handlers/settings_genieacs.go`, `src/components/genieacs/DeviceStatusBadge.tsx`) — Threshold was 15 minutes, causing devices to appear offline when GenieACS server restarts or NAT timeouts cause inform gaps >15min. Increased to 60 minutes to match GenieACS UI behavior. Devices with `PeriodicInformInterval=300s` (5 min) can have legitimate gaps up to 45+ minutes.

### Improved
- **GenieACS API response consistency** — All GenieACS proxy endpoints now return consistent JSON formats: `{success: true, data: [...]}` for lists, `{data: item}` for single items, matching frontend expectations.
- **Device parameters flattening** — `DeviceAllParameters` now returns a flat list of parameters with metadata (path, value, writable, object, timestamp), making it easier for the frontend to render parameter trees.
- **Device projection optimization** — `mapDevice` now uses targeted projection parameters to reduce GenieACS API response size.

### Files
- `internal/api/handlers/genieacs_ext.go` — **EDIT** — Fixed all path-based GETs to query-based, response formats, DELETE body parsing, GetDeviceTasks filtering, DeviceAllParameters flattening
- `internal/api/handlers/settings_genieacs.go` — **EDIT** — Status threshold 15→60min, capitalized status, flattenParameters enhancement, query filter for device detail
- `internal/api/router.go` — **EDIT** — Added DELETE /faults route (body-based)
- `src/components/genieacs/DeviceStatusBadge.tsx` — **EDIT** — Default threshold 15→60min
- `src/app/admin/genieacs/devices/[deviceId]/page.tsx` — **EDIT** — Fixed response format expectations

### v2.54.27 — 2026-08-08

### Fixed — Footer Login, Cronjob Emoji, GenieACS Test Connection
- **Footer Login not saving** (`internal/db/models/models.go`) — Root cause: Go `Company` struct was missing `footerAdmin`, `footerCustomer`, `footerTechnician`, `footerAgent` fields. These fields exist in Prisma schema and MySQL table, but were absent from the Go model. When the frontend sent footer values via `POST /api/company`, `json.Unmarshal` silently dropped them and `db.Save` never persisted them to the database. Added the 4 missing fields with proper GORM column tags.
- **Cronjob emoji not rendering** (`internal/db/db.go`) — Root cause: `cron_history` table may have been created with `utf8` charset instead of `utf8mb4`, causing 4-byte emoji characters to be corrupted. Added `ALTER TABLE cron_history CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci` migration. Also added same conversion for `companies` table (footer text may contain emoji).
- **GenieACS test connection always failing** (`internal/api/handlers/settings_genieacs.go`) — Root cause: `TestConnection` handler was a stub that always returned `{"success": false, "message": "GenieACS connection test not configured"}`. Implemented real connection test: makes HTTP GET to `{host}/devices` with Basic Auth, parses response to count devices, returns success with device count.
- **Missing qrisDeviceKey migration** (`internal/db/db.go`) — Added `ALTER TABLE companies ADD COLUMN qrisDeviceKey VARCHAR(100) NULL` migration that was referenced in v2.54.26 changelog but not actually added to db.go.

### Files
- `internal/db/models/models.go` — **EDIT** — Added FooterAdmin, FooterCustomer, FooterTechnician, FooterAgent fields to Company struct
- `internal/db/db.go` — **EDIT** — Added utf8mb4 conversion migrations for cron_history and companies, qrisDeviceKey column migration
- `internal/api/handlers/settings_genieacs.go` — **EDIT** — Implemented real TestConnection handler with HTTP request to GenieACS NBI API

<!-- AUTO-CHANGELOG:END -->

See full changelog: [docs/getting-started/CHANGELOG.md](docs/getting-started/CHANGELOG.md)

---

## ⚠️ Known Issues

| Issue | Status | Workaround |
|-------|--------|------------|
| **Go model ↔ DB schema drift** | Fixed (v2.54.26) | GORM migrations in `internal/db/db.go` now run on startup. Always add `ALTER TABLE` when adding new Go model fields. |
| **Silent GORM failures** | Fixed (v2.54.26) | All settings handlers now check `.Error` on GORM operations and return HTTP 500 on failure. |
| **Dual ORM complexity** | Monitored | Prisma and GORM share the same MySQL DB. Prisma owns most tables; GORM manages Go-specific tables. Coordinate schema changes carefully. |
| **Upload directory missing** | Fixed | systemd service `ReadWritePaths` requires `/var/www/salfanet-radius/uploads` to exist. Created by installer/updater. |

---

## 📚 Documentation

| File | Description |
|------|-------------|
| [docs/INSTALLATION-GUIDE.md](docs/INSTALLATION-GUIDE.md) | Complete VPS installation |
| [docs/GENIEACS-GUIDE.md](docs/GENIEACS-GUIDE.md) | GenieACS TR-069 setup & WiFi management |
| [docs/AGENT_DEPOSIT_SYSTEM.md](docs/AGENT_DEPOSIT_SYSTEM.md) | Agent balance & deposit |
| [docs/RADIUS-CONNECTIVITY.md](docs/RADIUS-CONNECTIVITY.md) | RADIUS architecture |
| [docs/FREERADIUS-SETUP.md](docs/FREERADIUS-SETUP.md) | FreeRADIUS configuration guide |

## 📝 License

MIT License - Free for commercial and personal use

## 👨‍💻 Development

Built with ❤️ for Indonesian ISPs

**Important**: Always use `formatWIB()` and `toWIB()` functions when displaying dates to users.
