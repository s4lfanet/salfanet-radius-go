# SALFANET RADIUS - Billing System for ISP/RTRW.NET

Modern, full-stack billing & RADIUS management system for ISP/RTRW.NET with FreeRADIUS integration supporting PPPoE and Hotspot authentication.

> **Latest:** v2.51.5 — Fix PM2 auto-start pada fresh install: hapus `pkill` yang bunuh installer, PM2 di bawah systemd supervision (May 19, 2026)

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
| **Roles & Permissions** | 53 permissions, 5 portals (Admin/Customer/Agent/Technician + SuperAdmin) |
| **Activity Log** | Audit trail with auto-cleanup (30 days) |
| **Security** | Session timeout 30 min, idle warning, RBAC, HTTPS/SSL |
| **Bahasa** | Bahasa Indonesia (full) |
| **PWA** | Installable di semua portal (admin, customer, agent, technician), offline fallback, service worker cache |
| **Web Push** | VAPID-based browser push notifications, subscribe/unsubscribe toggle, admin broadcast |
| **System Update** | Update via SSH menggunakan `updater.sh`, tidak ada web-based update |
| **Mobile App** | Flutter customer portal (WiFi control, invoice, payment) |
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
| Database | MySQL 8.0 + Prisma ORM |
| RADIUS | FreeRADIUS 3.0.26 |
| API Backend | Go (Fiber) — port 8080, systemd `salfanet-api.service` |
| Process Manager | PM2 (fork mode, 3 processes) + systemd |
| Session Tracking | FreeRADIUS radacct (real-time) |
| Maps | Leaflet / OpenStreetMap |

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

**Roles:** SUPER_ADMIN · FINANCE · CUSTOMER_SERVICE · TECHNICIAN · MARKETING · VIEWER

---

## 📝 Changelog

Bagian ini otomatis sinkron dari `CHANGELOG.md` saat file changelog berubah di GitHub.

<!-- AUTO-CHANGELOG:START -->

### v2.54.23 — 2026-07-30

### Added
- **Docker installer** — Full Docker Compose stack (`docker-compose.full.yml`) with 6 services: MySQL 8, Go API, Next.js Frontend, FreeRADIUS, Nginx, WhatsApp Service.
  - `docker/Dockerfile.go` — Multi-stage Go backend build (alpine, stripped binary).
  - `docker/Dockerfile.nextjs` — Multi-stage Next.js build with standalone output.
  - `docker/Dockerfile.freeradius` — Custom FreeRADIUS with SQL + REST modules.
  - `docker/nginx/salfanet.conf` — Nginx reverse proxy config for Docker.
  - `docker/freeradius/` — FreeRADIUS config templates (clients, sql, rest).
  - `docker-install.sh` — One-command Docker installer with auto-generated secrets.
  - `.env.docker.example` — Template for Docker environment configuration.

### Fixed
- **VPS installer: DB passwords** (`vps-install/common.sh`) — Replaced hardcoded default passwords with random generated ones.
- **VPS installer: CORS_ORIGINS** (`vps-install/install-app.sh`) — Fixed malformed `.env` output from shell conditional inside heredoc.
- **VPS installer: DATABASE_URL** (`vps-install/install-go.sh`) — Fixed standalone `.env` template to use proper `mysql://` URL format.

### v2.54.22 — 2026-07-30

### Fixed
- **Production cleanup** — Removed tracked binaries (`salfanet-api-linux`, `salfanet-api-win.exe`, `server.exe`), user data exports (`*.xlsx`, `*.csv`), and debug scripts (`dbquery.py`, `dbschema.py`, `debug_rx.py`, `json_keys.txt`, `used_keys.txt`, `step1.ps1`, `step2.ps1`, `audit_i18n.ps1`, `deploy-v2.52.84.sh`) from git. Updated `.gitignore` to prevent re-adding.
- **package.json** — Set `private: true` (this is not an npm package).
- **docker-compose.yml** — Removed deprecated `version: '3.8'` key (Docker Compose v2+).

### Added
- **`.env.production.example`** — Added missing Go backend env vars (`JWT_SECRET`, `PORT`, `CORS_ORIGINS`, `APP_BASE_URL`, `APP_TIMEZONE`, `WA_SERVICE_URL`, `UPLOAD_DIR`, `GO_API_URL`, Tripay/Duitku/Xendit keys, VAPID keys for Go backend).

### v2.54.21 — 2026-07-30

### Fixed
- **Security: Admin RBAC** (`internal/api/middleware/auth.go`) — Added `AdminPathGuard` middleware on the `api` group to enforce `RequireAdmin` on all `/admin/`, `/cron/`, `/backup/` paths. Updated `RequireAdmin` to accept both `ADMIN` and `SUPER_ADMIN` roles. Added generic `RequireRole` middleware.
- **Security: Rate limiting** (`internal/api/middleware/auth.go`) — Added `LoginRateLimit` middleware (10 attempts per 15 min per IP) on `/api/auth` and `/api/technician/auth` groups to prevent brute-force attacks.
- **Security: Password masking** (`internal/api/handlers/network_ext.go`) — `GetRouter` now returns `hasPassword`/`hasSecret` booleans instead of raw values. `UpdateRouter` skips empty password/secret to preserve existing values.
- **Duplicate route** (`internal/api/router.go`) — Removed duplicate `/api/cron/status` route registration.
- **Hardcoded API URL** (`src/app/*/layout.tsx`) — Replaced hardcoded `127.0.0.1:8080` with `GO_API_URL` env var in admin, customer, agent, and technician layouts.
- **Menu placement** (`src/app/admin/AdminClientLayout.tsx`) — Moved "Permintaan Top Up" from PPPoE to Hotspot category (it's for hotspot agents, not PPPoE).

### Added
- **Env var** (`.env.example`) — Added `GO_API_URL` for Next.js → Go backend communication.

### v2.54.20 — 2026-07-30

### Added
- **Cron: hotspot-sync** (`internal/cron/hotspot_sync.go`) — Full hotspot voucher lifecycle: WAITING→ACTIVE on first login detection, ACTIVE→EXPIRED on validity/usage duration expiry, Phase 3 cleanup of stale EXPIRED vouchers from RADIUS tables (radcheck, radreply, radusergroup, radgroupreply). Agent notifications on activation and expiry. MikroTik API disconnect with CoA fallback.
- **Cron: notification_check** (`internal/cron/notification_check.go`) — Every 6h: creates in-app notifications for overdue invoices, users expiring today, and pending registrations. Deduplication by type/link/time-window.
- **Cron: voucher_reconcile** (`internal/cron/voucher_reconcile.go`) — Daily 4 AM: creates financial transactions for used vouchers without orders (manually sold), plus agent commission expenses for vouchers with reseller fee.
- **Cron: auto-renewal** (`internal/cron/auto_renewal.go`) — Daily 8 AM WIB: auto-renew prepaid users with sufficient balance.
- **Cron: pppoe-sync** (`internal/cron/pppoe_sync.go`) — Enhanced auto-isolir for expired PPPoE users with grace period support.
- **Cron: telegram-cron** (`internal/cron/telegram_cron.go`) — Telegram backup (dynamic schedule from DB settings) + health check (hourly).
- **Cron: invoice-status-updater + cleanup + suspend-check** (`internal/cron/invoice_status.go`, `cleanup_jobs.go`, `suspend_check.go`) — Invoice overdue updates, activity log cleanup, webhook log cleanup, suspended user checking.
- **Service: isolation** (`internal/isolation/isolation.go`) — Isolation settings with 5-min cache, IP pool CIDR checking, CIDR range calculation for MikroTik pool config.
- **Service: push-notification** (`internal/push/push.go`) — Full Web Push service using `SherClockHolmes/webpush-go` with VAPID auth. Per-role sends (customer/agent/technician/admin), broadcast with target filtering (all/active/expired/area/selected), expired subscription auto-deactivation, dashboard stats.
- **Model: AgentNotification** — For voucher activation/expiry agent notifications.
- **Model: AdminPushSubscription** — Admin push subscription support.
- **Extended push subscription models** — Added `isActive`, `userAgent`, `expirationTime`, `lastUsedAt` fields to `PushSubscription`, `AgentPushSubscription`, `TechnicianPushSubscription`.

### Changed
- **Scheduler** (`internal/cron/scheduler.go`) — 17 jobs registered. Replaced basic `jobSyncVoucherExpiry` with enhanced `jobHotspotSync`. All jobs have manual trigger support via `TriggerJob`.
- **Push handler** (`internal/api/handlers/push_handler.go`) — Now uses `push.SendBroadcast` for actual Web Push delivery instead of just recording a notification. Stats endpoint uses `push.GetDashboardStats`.

### Fixed
- **Inefficient string concatenation** in `admin_misc_handler.go` and `misc_handler.go` — Replaced `WriteString` with `+` concatenation to `fmt.Fprintf` with format verbs.

### Files
- `internal/cron/hotspot_sync.go` — **NEW** hotspot voucher lifecycle cron
- `internal/cron/notification_check.go` — **NEW** notification check cron
- `internal/cron/voucher_reconcile.go` — **NEW** voucher transaction reconciliation cron
- `internal/cron/auto_renewal.go` — **NEW** auto-renewal cron
- `internal/cron/pppoe_sync.go` — **NEW** PPPoE sync + auto-isolir cron
- `internal/cron/telegram_cron.go` — **NEW** Telegram backup + health cron
- `internal/cron/invoice_status.go` — **NEW** invoice status updater cron
- `internal/cron/cleanup_jobs.go` — **NEW** activity log + webhook log cleanup cron
- `internal/cron/suspend_check.go` — **NEW** suspend check cron
- `internal/isolation/isolation.go` — **NEW** isolation service
- `internal/push/push.go` — **NEW** Web Push notification service
- `internal/cron/scheduler.go` — Updated with all new cron registrations
- `internal/db/models/extra.go` — New models + extended push subscription fields
- `internal/api/handlers/push_handler.go` — Updated to use push service
- `internal/api/handlers/admin_misc_handler.go` — Lint fixes
- `internal/api/handlers/misc_handler.go` — Lint fixes

### v2.54.19 — 2026-06-03

### Fixed
- **ONU list menampilkan data salah (11 online padahal 53 online)** — Root cause: port 9006 di Mikrotik DST-NAT diarahkan ke SSH ZTE (bukan Telnet), sehingga Telnet selalu gagal dengan SSH banner. v2.54.15 DB fallback kemudian mengunci 44 ONU sebagai offline secara permanen (death spiral). Fix 1: buat SSH pool (`internal/olt/ssh`) dengan API identik ke Telnet pool — poller sekarang pakai SSH (port 9005) untuk fetch ONU state, bukan Telnet. Fix 2: **hapus seluruh DB fallback** di poller — ketika CLI return 0 state, percayai SNMP apa adanya, tidak lagi "preserve DB offline status" yang menjadi sumber death spiral.
- **DB fallback death spiral dihapus** — Logika "jika CLI gagal, ambil status offline dari DB dan override SNMP online" dihapus permanen. Alasan: SNMP bisa saja memberikan partial data yang benar; mengoverride dengan data DB lama justru mempertahankan status salah selamanya.

### Added
- **SSH CLI pool** (`internal/olt/ssh/ssh.go`) — Package baru untuk koneksi SSH interaktif ke ZTE OLT. Implements same `Execute`/`ExecuteMultiple` API sebagai `telnet.Pool`. Stateless: setiap `ExecuteMultiple` buka koneksi SSH baru → tidak ada stale connection issue. Auth: password + keyboard-interactive fallback. PTY requested dengan ECHO=0.
- **`CLIPool` interface** di `internal/olt/vendors/zte/zte.go` — Interface `Execute(string)(string,error)` + `ExecuteMultiple([]string)(string,error)` yang diimplementasi oleh both `*telnet.Pool` dan `*ssh.Pool`. `FetchTelnetONUStates` dan `FetchTelnetDistances` sekarang menerima `CLIPool` bukan `*telnet.Pool`.
- **`GetCLIPool(oltID)`** di poller — Mengembalikan SSH pool jika tersedia, Telnet pool sebagai fallback. Handlers bisa pakai ini untuk management commands juga.

### Changed
- **Poller sekarang prefer SSH untuk CLI** — Jika `SSHEnabled=true` di OLT settings, SSH pool dibuat pada `SSHPort`. Telnet pool hanya dibuat jika SSH tidak dikonfigurasi. SSH adalah sumber kebenaran untuk ONU state di ZTE C320.

### Files
- `internal/olt/ssh/ssh.go` — **NEW** SSH CLI pool
- `internal/olt/vendors/zte/zte.go` — `CLIPool` interface; update `FetchTelnetONUStates` + `FetchTelnetDistances` signatures
- `internal/olt/poller/poller.go` — SSH pool support; hapus DB fallback; `GetCLIPool()`

<!-- AUTO-CHANGELOG:END -->

See full changelog: [docs/getting-started/CHANGELOG.md](docs/getting-started/CHANGELOG.md)

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
