# SALFANET RADIUS - Billing System for ISP/RTRW.NET

Modern, full-stack billing & RADIUS management system for ISP/RTRW.NET with FreeRADIUS integration supporting PPPoE and Hotspot authentication.

> **Latest:** v2.54.26 — Settings silent failure audit: fixed GORM Create/Save/Updates calls across 10 settings handlers that silently failed and returned HTTP 200 with unsaved data. Added missing `qrisDeviceKey` column migration. (Aug 8, 2026)

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
| **GenieACS TR-069** | CPE/ONT management, WiFi config (SSID/password), device status & uptime |
| **Isolation** | Auto-isolate expired customers, customizable WhatsApp/Email/HTML landing page templates |
| **Cron Jobs** | 16 automated background jobs (tsx runner via PM2 fork), history, distributed locking, manual trigger |
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

### v2.54.26 — 2026-08-08

### Fixed — Settings Silent Failure Audit
- **Company settings not saving** (`internal/api/handlers/company.go`) — Root cause: Go model `Company` had `QrisDeviceKey` field but DB table `companies` lacked the `qrisDeviceKey` column. GORM `Create`/`Save` silently failed with "Unknown column" error. Added error checking on `db.Create` and `db.Save` calls with proper 500 response + `log.Error()` logging.
- **Database migration** (`internal/db/db.go`) — `ALTER TABLE companies ADD COLUMN qrisDeviceKey VARCHAR(100) NULL` — adds missing column referenced by Go model.
- **Email settings silent save** (`internal/api/handlers/settings.go`) — `UpdateEmailSettings`: `db.Model().Updates()` had no error check. Added error handling + logging.
- **Telegram settings silent save** (`internal/api/handlers/telegram_handler.go`) — `UpdateSettings`: `db.Create()` and `db.Model().Updates()` had no error check. Added error handling + logging.
- **Backup Telegram settings silent save** (`internal/api/handlers/backup_handler.go`) — `UpdateTelegramSettings`: `db.Create()` and `db.Model().Updates()` had no error check. Added error handling + logging.
- **WhatsApp reminder settings silent save** (`internal/api/handlers/whatsapp.go`) — `UpdateReminderSettings`: `db.Create()` and `db.Save()` had no error check. Added error handling + logging.
- **Cloudflare tunnel settings silent save** (`internal/api/handlers/admin_misc_handler.go`) — `UpdateCloudflareTunnel`: `db.Model().Updates()` had no error check. Added error handling + logging.
- **Map settings silent save** (`internal/api/handlers/admin_misc_handler.go`) — `UpdateMapSettings`: `db.Model().Updates()` had no error check. Added error handling + logging.
- **VPN settings silent save** (`internal/api/handlers/admin_vpn_handler.go`) — `UpdateSettings`: `db.Model().Updates()` had no error check. Added error handling + logging.
- **GenieACS settings silent save** (`internal/api/handlers/genieacs.go`) — `SaveSettings`: `db.Create()` and `db.Model().Updates()` had no error check. Added error handling + logging.
- **Payment gateway settings silent save** (`internal/api/handlers/misc_handler.go`) — `UpdatePaymentGateway`: `db.Create()` and `db.Save()` had no error check. Added error handling + logging.

### Changed
- **Pattern fix across all settings handlers** — All GORM `Create`/`Save`/`Updates` calls in settings handlers now check `.Error` and return HTTP 500 with error message on failure. Previously, these calls silently failed, returning HTTP 200 with unsaved data to the frontend, causing the "data reverts to empty" bug.

### Files
- `internal/api/handlers/company.go` — **EDIT** — Error checking on Create/Save
- `internal/api/handlers/settings.go` — **EDIT** — Error checking on Updates (email settings)
- `internal/api/handlers/telegram_handler.go` — **EDIT** — Error checking on Create/Updates
- `internal/api/handlers/backup_handler.go` — **EDIT** — Error checking on Create/Updates
- `internal/api/handlers/whatsapp.go` — **EDIT** — Error checking on Create/Save
- `internal/api/handlers/admin_misc_handler.go` — **EDIT** — Error checking on Updates (cloudflare + map)
- `internal/api/handlers/admin_vpn_handler.go` — **EDIT** — Error checking on Updates
- `internal/api/handlers/genieacs.go` — **EDIT** — Error checking on Create/Updates
- `internal/api/handlers/misc_handler.go` — **EDIT** — Error checking on Create/Save (payment gateway)
- `internal/db/db.go` — **EDIT** — Migration: qrisDeviceKey column

### v2.54.25 — 2026-08-08

### Changed — Collector Area Management Refactor
- **Menu rename** (`src/locales/id.json`, `src/app/admin/territories/page.tsx`) — "Manajemen Wilayah" → "Manajemen Kolektor". All UI labels updated: page title, modal title, buttons, search placeholder, detail modal.
- **Auto-populate collector name** (`src/app/admin/territories/page.tsx`) — Removed manual name input field. Collector name auto-fills from dropdown selection. Dropdown is now required.
- **Area multi-select in modal** (`src/app/admin/territories/page.tsx`) — Added checkbox list for area selection directly in create/edit collector modal. Fetches all areas from `GET /api/territories/all-areas`. Areas assigned to other collectors shown greyed out with "(wilayah lain)" label. Empty state links to Kelola Area page.
- **Backend: areaIds in create/update** (`internal/api/handlers/territory_handler.go`) — `CreateTerritory` and `UpdateTerritory` now accept `areaIds[]` in request body. Areas are assigned/unassigned by updating `territoryId` on `pppoe_areas` in a single request. Removed need for separate area assignment calls.

### Added
- **COLLECTOR role permissions** (`internal/api/handlers/permissions.go`) — Added "COLLECTOR" to `GetRoleTemplates` role list. Role permissions now auto-generate when COLLECTOR role is selected in user management.
- **Database migration** (`internal/db/db.go`) — `ALTER TABLE pppoe_areas ADD COLUMN territoryId VARCHAR(191) NULL` + index. Links areas to territories. `ALTER TABLE admin_users MODIFY COLUMN role ENUM(..., 'COLLECTOR')` — adds COLLECTOR to role enum.
- **API endpoint** — `GET /api/territories/all-areas` — Returns all areas (assigned + unassigned) for multi-select UI.
- **Upload: BodyLimit** (`internal/api/router.go`) — Set Fiber BodyLimit to 10MB (default 4MB) to support file uploads.
- **Upload: file types** (`internal/api/handlers/upload.go`) — Added `.gif` and `.avif` support to match frontend accept types. Added error logging for debugging 500 errors.

### Files
- `internal/api/handlers/territory_handler.go` — **EDIT** — CreateTerritory & UpdateTerritory accept areaIds
- `internal/api/handlers/permissions.go` — **EDIT** — Add COLLECTOR to GetRoleTemplates
- `internal/db/db.go` — **EDIT** — Migration: territoryId column, COLLECTOR role enum
- `internal/api/router.go` — **EDIT** — BodyLimit 10MB
- `internal/api/handlers/upload.go` — **EDIT** — Error logging, .gif/.avif support
- `src/app/admin/territories/page.tsx` — **EDIT** — Rename to Manajemen Kolektor, auto-fill name, area multi-select
- `src/locales/id.json` — **EDIT** — nav.territoryManage: "Manajemen Kolektor"

### v2.54.24 — 2026-07-30

### Changed — UI/UX Consistency Audit
- **Theme tokens** — Replaced all hardcoded neon hex colors (`#00f7ff`, `#bc13fe`, `#0a0520`, `#1a0f35`, `#e0d0ff`, `#ff44cc`, `#00ff88`, `#ff4466`, `#ff6b8a`, `#ff8c00`, `#1e1b2e`, `#0f0a1e`) with CSS variable tokens (`brand-400`, `brand-600`, `input`, `secondary`, `muted-foreground`, `accent-foreground`, `success`, `destructive`, etc.) across all 124 TSX files.
- **Border radius** — Standardized `rounded-2xl` (16px) and `rounded-3xl` (24px) → `rounded-xl` (14px) across 53 files to align with the `--radius-xl` design token.
- **Typography** — Fixed body font size (13px→14px), html root (14px→16px, mobile 15px), and larger heading sizes for better visual hierarchy.
- **SweetAlert** — Replaced all neon hex colors in SweetAlert theme with brand tokens. Removed neon glow shadows in favor of clean elevation shadows.
- **Utility classes** — Standardized `.neon-glow`, `.btn-cyber`, `.badge-*`, `.compact-card`, `.glass`, `.cyber-gradient`, `.table-container` to use theme tokens.
- **Decorative elements** — Removed 261 decorative `blur-3xl` glow blob elements across 70 files, plus grid overlay patterns and their container divs.
- **CSS cleanup** — Removed ~630 lines of redundant CSS overrides in `globals.css` targeting legacy hex colors and `slate-900/800` classes no longer present in TSX files. Fixed 2FA input border color from neon cyan to brand blue.
- **Unused palette** — Removed unreferenced color variables from `globals.css`.

### Stats
- 124 files changed, 2,912 insertions, 3,787 deletions
- `globals.css` reduced from ~2,481 to 1,850 lines

### v2.54.23 — 2026-07-30

### Fixed
- **VPS installer: DB passwords** (`vps-install/common.sh`) — Replaced hardcoded default passwords with random generated ones.
- **VPS installer: CORS_ORIGINS** (`vps-install/install-app.sh`) — Fixed malformed `.env` output from shell conditional inside heredoc.
- **VPS installer: DATABASE_URL** (`vps-install/install-go.sh`) — Fixed standalone `.env` template to use proper `mysql://` URL format.

### v2.54.22 — 2026-07-30

### Fixed
- **Production cleanup** — Removed tracked binaries (`salfanet-api-linux`, `salfanet-api-win.exe`, `server.exe`), user data exports (`*.xlsx`, `*.csv`), and debug scripts (`dbquery.py`, `dbschema.py`, `debug_rx.py`, `json_keys.txt`, `used_keys.txt`, `step1.ps1`, `step2.ps1`, `audit_i18n.ps1`, `deploy-v2.52.84.sh`) from git. Updated `.gitignore` to prevent re-adding.
- **package.json** — Set `private: true` (this is not an npm package).

### Added
- **`.env.production.example`** — Added missing Go backend env vars (`JWT_SECRET`, `PORT`, `CORS_ORIGINS`, `APP_BASE_URL`, `APP_TIMEZONE`, `WA_SERVICE_URL`, `UPLOAD_DIR`, `GO_API_URL`, Tripay/Duitku/Xendit keys, VAPID keys for Go backend).

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
