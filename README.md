# SALFANET RADIUS - Billing System for ISP/RTRW.NET

Modern, full-stack billing & RADIUS management system for ISP/RTRW.NET with FreeRADIUS integration supporting PPPoE and Hotspot authentication.

> **Latest:** v2.25.2 — Native Baileys WhatsApp gateway built-in di VPS, QR modal auto-retry, auto-reconnect setelah device disconnect (Apr 26, 2026)

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
| `salfanet-radius` | cluster | 3000 | Next.js app |
| `salfanet-wa` | fork | 4000 (internal) | Baileys WA service |
| `salfanet-cron` | fork | — | Background jobs |

### Auth Session

Session WhatsApp tersimpan di `/var/data/salfanet/baileys_auth/` dan persist meski PM2 restart. Untuk logout/scan ulang, klik **Restart Session** di admin panel.

---

## 🚀 Tech Stack

| Component | Technology |
|-----------|------------|
| Framework | Next.js 16 (App Router, standalone output) |
| Language | TypeScript |
| Styling | Tailwind CSS |
| Database | MySQL 8.0 + Prisma ORM |
| RADIUS | FreeRADIUS 3.0.26 |
| Process Manager | PM2 (cluster × 2) |
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

git clone https://github.com/s4lfanet/salfanet-radius.git /root/salfanet-radius
cd /root/salfanet-radius
bash vps-install/vps-installer.sh
```

Installer akan berjalan **interaktif** — mendeteksi environment otomatis, memandu konfigurasi, lalu menjalankan semua step.

---

### Metode 2 — Upload Manual via SCP (Tanpa Akses Internet di Server)

```bash
# Jalankan di terminal LOKAL (bukan di server)
scp -r ./salfanet-radius root@YOUR_VPS_IP:/root/salfanet-radius

# SSH ke server, lalu jalankan installer
ssh root@YOUR_VPS_IP
cd /root/salfanet-radius
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
npm install --legacy-peer-deps
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

### v2.46.0 — 2026-05-15

### Added
- **Uninstaller `--unattended` mode** — `vps-uninstaller.sh` kini mendukung `--unattended` (alias `--force`/`-y`) untuk menghapus semua komponen tanpa interaksi manual; flag tambahan `--keep-nodejs`, `--keep-mysql`, `--keep-pm2`
- **Go backend removal** — Uninstaller sekarang menghentikan dan menghapus service `salfanet-api` (systemd unit), serta Go runtime (`/usr/local/go`) secara otomatis
- **Port 8080 cleanup** — Tambah port 8080 ke daftar port yang dibersihkan oleh uninstaller

### Fixed
- **Semua `read` prompts interactive** — `remove_freeradius`, `remove_pm2`, `remove_nodejs`, `remove_mysql`, `clean_firewall` kini skip konfirmasi saat mode `--unattended`
- **`clean_logs` error** — Ubah `rm -rf` yang salah ke `rm -f` untuk file (bukan direktori)
- **Log cleanup** — Tambah `/var/log/salfanet-install.log` ke daftar file yang dihapus

### Verified
- **Uninstall → fresh install cycle tested** — Uninstaller `--unattended` berhasil menghapus semua komponen; fresh install berhasil kembali; semua services aktif: nginx, mysql, freeradius, salfanet-api, fail2ban, PM2 (3 apps)

### Files
- `vps-install/vps-uninstaller.sh` — `--unattended` mode, Go backend removal, port 8080, fix semua prompts interactive


### Added
- **`--unattended` flag** — `vps-installer.sh` mendukung flag `--unattended`, `--env`, `--ip`, `--domain`, `--db-pass` untuk instalasi otomatis tanpa interaksi manual (CI/CD, test fresh install)

### Fixed
- **TERM env crash** — `print_banner()` di `common.sh` menggunakan `clear 2>/dev/null || true`; tambah `export TERM="${TERM:-xterm}"` di `vps-installer.sh` agar tidak crash di sesi non-interactive
- **dpkg interactive prompts** — Set `DEBIAN_FRONTEND=noninteractive` + `DEBCONF_NONINTERACTIVE_SEEN=true` secara global di `common.sh` dan apt-get upgrade/install di `install-system.sh` menggunakan `--force-confdef --force-confold` agar tidak ada prompt config file conflict
- **`initialize_user_selection` read prompt** — Skip `read` prompt untuk pilihan user saat mode `--unattended`
- **Go backend `connection_limit` crash** — `convertDSN()` di `internal/db/db.go` sekarang strip Prisma-specific URL params (`connection_limit`, `pool_timeout`, dll) sebelum build MySQL DSN. MySQL tidak mengenal params ini sebagai session variable.

### Changed
- **Repo cleanup** — Hapus file non-produksi dari GitHub: `oltc320_v2.1.1_linux/`, `baileys_whatsapp_patch/`, `bin/server.exe`, `nginx-frontend.conf`, `ZTE_OID_TABLE.md`, `.air.toml`, debug scripts; perbarui `.gitignore`

### Verified
- **Fresh install test passed** — Instalasi dari nol di Ubuntu 22.04 LTS berhasil: MySQL, Node.js v20, FreeRADIUS, nginx, PM2 (salfanet-radius + salfanet-cron + salfanet-wa), Go backend (port 8080, 820 handlers), 101 tabel Prisma schema, seed admin user — semua services `active`

### Files
- `vps-install/vps-installer.sh` — Tambah flag `--unattended`, `--env`, `--ip`, `--domain`, `--db-pass`; export TERM default
- `vps-install/common.sh` — `DEBIAN_FRONTEND=noninteractive` global; `clear` toleran error; `--unattended` skip user selection prompt
- `vps-install/install-system.sh` — apt-get upgrade + install dengan `--force-confdef --force-confold`
- `internal/db/db.go` — Strip Prisma URL params sebelum build MySQL DSN
- `.gitignore` — Tambah entri untuk file/folder non-produksi

### v2.44.0 — 2026-05-14

### Changed
- **nginx /api/ routing** — All server blocks now route `/api/` → Go backend (port 8080) instead of Next.js (port 3000)
- **install-nginx.sh** — Updated `_proxy_locations()` & `_proxy_locations_https_domain()` helpers; added `salfanet_api` upstream (port 8080, keepalive 32)
- **install-go.sh** — Refactored into proper module with `install_go_runtime()`, `build_go_binary()`, `setup_go_systemd_service()`, `start_go_service()`, `install_go_backend()` functions; standalone mode preserved
- **updater.sh** — Added Go binary rebuild + systemd service restart step before Node.js install
- **vps-installer.sh** — Added Step 4.5: Go backend build & systemd service after app setup; added Go version + service status to install summary
- **ecosystem.config.js** — `salfanet-cron` `API_URL` updated from port 3000 → port 8080 (Go backend)
- **install-app.sh** — `.env` template extended with Go-specific vars (`PORT`, `APP_ENV`, `JWT_SECRET`, `CORS_ORIGINS`, `WA_SERVICE_URL`, `GO_API_URL`)
- **production/nginx-salfanet-radius.conf** — `/api/` → port 8080 in all 4 server blocks; SSL cert paths updated (Let's Encrypt for domain blocks, self-signed comment for IP blocks)

### Files
- `vps-install/install-nginx.sh` — Go upstream + /api/ proxy to port 8080
- `vps-install/install-go.sh` — Full module refactor with proper functions
- `vps-install/updater.sh` — Go binary build step added
- `vps-install/vps-installer.sh` — Step 4.5 + summary info for Go service
- `vps-install/install-app.sh` — Go env vars in .env template
- `production/ecosystem.config.js` — cron API_URL → 8080
- `production/nginx-salfanet-radius.conf` — /api/ → Go:8080 + LE cert paths

### v2.43.0 — 2026-05-14

### Added
- **Troubleshooting DB tables** — Created `troubleshooting_checklists`, `troubleshooting_jobs`, `troubleshooting_materials` tables in MySQL
- **Payroll DB tables** — Created `payroll_templates`, `payroll_records`, `payroll_overtime` tables in MySQL
- **Admin page: Troubleshooting Checklists** — `/admin/troubleshooting` — CRUD checklist panduan troubleshooting per kategori
- **Admin page: Troubleshooting Jobs** — `/admin/troubleshooting/jobs` — Lacak pekerjaan troubleshooting aktif (stats: open/in-progress/resolved)
- **Admin page: Absensi** — `/admin/attendance` — Manajemen kehadiran karyawan/teknisi dengan bulk delete
- **Admin page: Kasbon** — `/admin/cash-advances` — Pengajuan & approval kasbon dengan tombol bayar
- **Admin page: Komisi** — `/admin/commissions` — Manajemen komisi instalasi/sales/referral dengan approve/reject
- **Admin page: Template Payroll** — `/admin/payroll-templates` — Template perhitungan gaji dengan preview langsung
- **Admin page: Payroll** — `/admin/payroll` — Generate & manajemen slip gaji bulanan dengan tombol lunas
### Fixed
- **troubleshooting_handler.go** — `"job_id = ?"` → `"jobId = ?"` (camelCase consistency)
### Files
- `internal/api/handlers/troubleshooting_handler.go` — fixed jobId column reference
- `scripts/create-missing-tables.sql` — SQL untuk 6 tabel baru (dieksekusi di VPS)
- `src/app/admin/troubleshooting/page.tsx` — halaman baru
- `src/app/admin/troubleshooting/jobs/page.tsx` — halaman baru
- `src/app/admin/attendance/page.tsx` — halaman baru
- `src/app/admin/cash-advances/page.tsx` — halaman baru
- `src/app/admin/commissions/page.tsx` — halaman baru
- `src/app/admin/payroll-templates/page.tsx` — halaman baru
- `src/app/admin/payroll/page.tsx` — halaman baru

### v2.42.0 — 2026-05-14

### Fixed
- **DB camelCase 100% complete** — Fixed final 8 remaining snake_case column references across 5 files
- **TechnicianOtp** — `Update("used_at", now)` → `Update("isUsed", true)` (model uses `bool`, not `*time.Time`)
- **Technician auth** — `Update("last_login_at", now)` → `Update("lastLoginAt", now)` in both login paths
- **PPPoE sync** — `Update("synced_to_radius", true)` → `Update("syncedToRadius", true)`
- **PPPoE expiry** — `Update("expired_at", newExpiry)` → `Update("expiredAt", newExpiry)`
- **Invoice generator** — `"subscription_type = ?"` → `"subscriptionType = ?"` and `"invoice_type = ?"` → `"invoiceType = ?"`
- **Auth 2FA** — removed dead `"otp_code": nil` key (column doesn't exist in `admin_two_factor_pending`)
### Files
- `internal/api/handlers/technician_portal.go` — isUsed, lastLoginAt
- `internal/api/handlers/pppoe.go` — syncedToRadius
- `internal/api/handlers/pppoe_ext.go` — expiredAt
- `internal/api/handlers/invoices_ext.go` — subscriptionType, invoiceType
- `internal/api/handlers/auth.go` — removed dead otp_code key

### v2.41.0 — 2026-05-14

### Fixed
- **HR tables migration** — Created missing DB tables: `attendance_records`, `attendance_locations`, `cash_advances`, `commissions` with correct camelCase column names
- **registration_requests migration** — Added missing `processedAt datetime(3)` column to `registration_requests` table
- **processedAt column** — Fixed `"processed_at"` → `"processedAt"` in Updates maps across `admin_jobs.go` and `pppoe.go`
- **HR handler column names** — Fixed `"employee_id = ?"` → `"employeeId = ?"` and `"check_in desc"` → `"checkIn desc"` in `admin_hr_handler.go`
### Files
- `internal/api/handlers/admin_jobs.go` — processedAt in registration approve/reject/install
- `internal/api/handlers/pppoe.go` — processedAt in registration approve/reject
- `internal/api/handlers/admin_hr_handler.go` — employeeId WHERE, checkIn ORDER BY

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
