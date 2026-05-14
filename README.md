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

### v2.46.5 — 2026-05-15

### Fixed
- **500: `/api/network/cables`** — Relasi Prisma salah: `cable_segments` → `segments`; status enum salah: `ASSIGNED` → `USED`
- **404: `/api/admin/attendance`** — Dibuat Next.js API route `GET/POST` dengan `$queryRawUnsafe` ke tabel `attendance_records`
- **404: `/api/admin/cash-advances`** — Dibuat Next.js API route `GET/POST` ke tabel `cash_advances`
- **404: `/api/admin/commissions`** — Dibuat Next.js API route `GET/POST` ke tabel `commissions`
- **404: `/api/admin/payroll`** — Dibuat Next.js API route `GET/POST` ke tabel `payroll_records`

### Added
- **DB migration HR tables** — `scripts/migrate-hr-tables.sql` membuat tabel: `attendance_locations`, `attendance_records`, `cash_advances`, `commissions`, `payroll_records`, `payroll_overtime`

### Files
- `src/app/api/network/cables/route.ts` — Fix relasi `segments` dan enum `USED`
- `src/app/api/admin/attendance/route.ts` — BARU: GET/POST attendance
- `src/app/api/admin/cash-advances/route.ts` — BARU: GET/POST cash advances
- `src/app/api/admin/commissions/route.ts` — BARU: GET/POST commissions
- `src/app/api/admin/payroll/route.ts` — BARU: GET/POST payroll records
- `scripts/migrate-hr-tables.sql` — BARU: SQL migration HR tables

### v2.46.4 — 2026-05-14

### Fixed
- **Sidebar: Hapus menu "Manajemen VPN"** — Submenu duplikat (Klien VPN, Site VPN, Pengaturan VPN) dihapus dari sidebar karena VPN Client sudah ada di menu Router
- **CSP: Leaflet CSS dari cdnjs diblokir** — Ditambahkan `https://cdnjs.cloudflare.com`, `https://fonts.googleapis.com`, `https://cdn.jsdelivr.net` ke `style-src` di `next.config.ts`
- **404: `/api/payroll-templates`** — Endpoint hanya ada di Go (butuh Bearer token); dibuat Next.js API route `GET/POST` + `[id]` `GET/PUT/DELETE` + `[id]/default POST` menggunakan `prisma.$queryRaw`
- **500: `/api/network/cables`** — Tabel `fiber_cables` beserta relasi tidak ada di DB VPS; dibuat SQL migration `scripts/migrate-fiber-payroll-tables.sql`

### Added
- **DB migration SQL** — `scripts/migrate-fiber-payroll-tables.sql` membuat tabel: `payroll_templates`, `fiber_cables`, `fiber_tubes`, `fiber_cores`, `splice_points`, `cable_segments`, `core_assignment_history`

### Files
- `src/app/admin/AdminClientLayout.tsx` — Hapus blok `vpnManagement` dari sidebar
- `next.config.ts` — Tambah CDN ke `style-src` CSP
- `src/app/api/payroll-templates/route.ts` — Baru: GET list, POST create
- `src/app/api/payroll-templates/[id]/route.ts` — Baru: GET, PUT, DELETE
- `src/app/api/payroll-templates/[id]/default/route.ts` — Baru: POST set-default
- `scripts/migrate-fiber-payroll-tables.sql` — Baru: SQL migration fiber + payroll

### v2.46.3 — 2026-05-14

### Fixed
- **404: `/admin/invoice-templates`** — Halaman frontend tidak ada; dibuat `src/app/admin/invoice-templates/page.tsx` lengkap dengan tabel, modal create/edit, preview HTML, set-default
- **404: `/admin/logs/activity`** — File `page.tsx` ada di lokal tapi belum di-commit ke git; sekarang sudah dicommit
- **404: `/api/troubleshooting/checklists`** — Endpoint hanya ada di Go (butuh Bearer token), sedangkan admin panel tidak mengirim token. Dibuat Next.js API route (`src/app/api/troubleshooting/checklists/route.ts` dan `[id]/route.ts`) menggunakan Prisma `$queryRaw` langsung ke tabel MySQL
- **404: `/api/invoice-templates`** — Sama dengan troubleshooting; dibuat Next.js API route lengkap (GET/POST list, GET/PUT/DELETE per-id, POST set-default)

### Added
- **14 missing nav translation keys** — Ditambahkan ke `src/locales/id.json`: `invoiceTemplates`, `troubleshooting`, `troubleshootingChecklists`, `troubleshootingJobs`, `hrManagement`, `attendance`, `cashAdvances`, `commissions`, `payroll`, `payrollTemplates`, `vpnManagement`, `vpnClients`, `vpnSites`, `vpnSettings`
- **DB migration SQL** — `scripts/migrate-missing-tables.sql` membuat tabel `troubleshooting_checklists`, `troubleshooting_jobs`, `troubleshooting_materials`, `invoice_templates`

### Files
- `src/app/admin/invoice-templates/page.tsx` — Halaman baru (dibuat)
- `src/app/admin/logs/activity/page.tsx` — Commit ke git (sebelumnya untracked)
- `src/app/api/troubleshooting/checklists/route.ts` — GET + POST
- `src/app/api/troubleshooting/checklists/[id]/route.ts` — PUT + DELETE
- `src/app/api/invoice-templates/route.ts` — GET + POST
- `src/app/api/invoice-templates/[id]/route.ts` — GET + PUT + DELETE
- `src/app/api/invoice-templates/[id]/default/route.ts` — POST (set default)
- `src/locales/id.json` — Tambah 14 nav keys
- `scripts/migrate-missing-tables.sql` — SQL migration baru

### v2.46.2 — 2026-05-14

### Fixed
- **Nginx smart API routing** — Routing `/api/` sebelumnya mengarahkan semua ke Go backend, menyebabkan `GET /api/company` dan `POST /api/admin/auth/pre-login` menghasilkan 401 "missing authorization header". Sekarang ada routing granular: Go JWT auth (`/api/auth/login`, `/api/auth/logout`, `/api/auth/refresh`, `/api/auth/customer/`, `/api/auth/agent/`), NextAuth (`/api/auth/callback/`, `/api/auth/session`, `/api/auth/csrf`, `/api/auth/signout`), portal API (`/api/customer/`, `/api/agent/`, `/api/technician/`) ke Go, dan catch-all `/api/` → Next.js untuk admin panel.
- **Admin login 401 resolved** — `POST /api/admin/auth/pre-login` kini mengarah ke Next.js (handler Prisma+2FA yang sesungguhnya), bukan Go stub yang dilindungi AuthMiddleware

### Files
- `vps-install/install-nginx.sh` — Ganti 2 location block lama (`/api/auth/` + `/api/`) dengan routing granular komprehensif di kedua fungsi `_proxy_locations()` dan `_proxy_locations_https_domain()`

### v2.46.1 — 2026-05-14

### Fixed
- **NextAuth 401 error** — Nginx mengarahkan semua `/api/*` ke Go backend (port 8080), termasuk `/api/auth/*` yang dikelola Next.js. Tambah `location /api/auth/` → port 3000 **sebelum** block `location /api/` di kedua server block (HTTP + HTTPS). Sebelumnya semua `GET /api/auth/session` dan `POST /api/auth/_log` menghasilkan 401 dari Go backend.

### Files
- `vps-install/install-nginx.sh` — Tambah `location /api/auth/` → Next.js (port 3000) sebelum `location /api/` di HTTP dan HTTPS server block


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
