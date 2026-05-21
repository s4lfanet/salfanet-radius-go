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

### v2.52.63 — 2026-05-23

### Fixed
- **Port map uplink AdminStatus/LinkStatus wrong** — `parseUplinkPortStatus` was reading column 7 (Pause) and column 8 (FlowControl) instead of column 9 (AdminStatus) and column 10 (LinkStatus). All uplink ports showed `DIS`/down incorrectly on the chassis port map.
- **Minimum column count check** — updated from `len(parts) < 8` to `len(parts) < 11` so rows with fewer than 11 columns are skipped correctly.
- **Uplink description via SNMP** — when Telnet-parsed uplink states are available, SNMP `ifAlias` descriptions are now merged in, so uplink port hover tooltips show descriptions even when Telnet is the primary data source.
- **GetUplink handler** — replaced stub with real Telnet-based implementation for all four tabs: `status` (parses `show interface port-status` + `show interface` with SNMP fallback), `vlan` (parses `show running-config interface`), `config` (returns raw running-config), `optical` (parses `show interface optical-module-info` + `show ddmi interface`).
- **CreateUplink handler** — replaced stub with real Telnet config-mode implementation: `addVlan`, `removeVlan`, `enable`, `disable`, `setPvid`, `removePvid`, `setDescription`.
### Files
- `internal/api/handlers/olt_chassis.go` — fix `parseUplinkPortStatus` column indices (9, 10); merge SNMP description into Telnet-parsed uplink states
- `internal/api/handlers/olt.go` — implement real `GetUplink` + `CreateUplink` handlers with full parser helpers; add `context`, `regexp`, `strings`, snmputil imports

### v2.52.62 — 2026-05-23

### Fixed
- **ONU polling — missing ONUs from new/empty PON ports** — removed DB-based `knownPONPorts` lookup. PON ports are now discovered dynamically by walking the ZTE V2.1 PON port table (`oidPONPortTable = 1.3.6.1.4.1.3902.1012.3.11.3.1.1`). All provisioned PON ports are found regardless of DB state. Falls back to 2×8 default only when the walk returns nothing.
- **Unregistered ONU detection via SNMP** — replaced Telnet `show gpon onu uncfg` with a walk of the SNMP seen-ONU table (`zxAnGponOnuDiscoveredInfoTable = 1.3.6.1.4.1.3902.1012.3.27.4.1.1`). ONUs present in the seen table but absent from `regStatus` walk are recorded with `status=unregistered`. No Telnet required for read operations; Telnet remains available for config/registration write operations.
- **SNMP BulkWalk** — replaced GetNext-based `Walk` with GetBulk-based `BulkWalk` for all ONU data collection (8 parallel BulkWalks per PON port). Auto-falls back to Walk when the agent rejects GetBulk.
- **RxPower validation** — added upper bound check (`rxRaw < 50000`) alongside the existing `rxRaw > 0` check to filter invalid raw values. Same fix applied to new `TxPower` field.
- **TxPower** — added OLT TX power toward ONU (`oidTxPower = .3.50.12.1.1.11`) to per-ONU data.
- **Unregistered ONU DB upsert** — poller now saves unregistered ONUs to `olt_onu_status` with a separate upsert that only updates `status` + `lastSeenAt`, preserving any previously known serial number/description.
### Files
- `internal/olt/snmp/snmp.go` — add `BulkWalk` function (GetBulk with auto Walk fallback)
- `internal/olt/vendors/zte/zte.go` — add `discoverPONPorts` (dynamic SNMP PON table walk); rewrite `walkPONPort` to use `BulkWalk` + seen-ONU table; add `txPower` OID; fix `DiscoverAll` signature (no more `telnetPool`/`ponPorts` params); fix RxPower/TxPower upper bound
- `internal/olt/poller/poller.go` — remove `knownPONPorts`; update `DiscoverAll` call; split upsert into registered (full) and unregistered (status only); add `unregistered` count to broadcast

### v2.52.61 — 2026-05-22

### Added
- **GetChassis — real Telnet + SNMP chassis data (ported from Next.js)** — `GET /api/olt/:id/chassis` now fires Telnet (`show card` + `show interface port-status`) and 5 SNMP IF-MIB walks (ifDescr, ifAdminStatus, ifOperStatus, ifHighSpeed, ifAlias) plus ZTE PON table walk **in parallel**. Real card types (GTGO/GTGH/GTGQ/SMXA/MCUD) come from Telnet `show card`; uplink port states (admin/link status, speed, description) come from `show interface port-status` with SNMP IF-MIB fallback. Falls back to SNMP board presence + DB ONU port data when Telnet is unavailable. Response includes `source: "telnet" | "snmp+db"`.
### Files
- `internal/api/handlers/olt_chassis.go` — new file: full chassis handler with `parseShowCard`, `classifyCard`, `smxaUplinkIfaces`, `parseUplinkPortStatus`, `buildUplinkStatesFromSNMP`, SNMP IF-MIB walk helpers, and `GetChassis`
- `internal/api/handlers/olt.go` — remove old inline `GetChassis` implementation (now delegated to `olt_chassis.go`)

### v2.52.60 — 2026-05-22

### Fixed
- **ZTE SNMP poller — wrong frame/slot mapping** — `decodePonIndex` stored board2 ONUs as `frame=2, slot=1` instead of `frame=1, slot=2`. Fixed: `frame` is always `1` (ZTE C320 single chassis), `slot = board` (1 or 2), `port = PON number (1-based)`. Matches ZTE CLI notation `gpon-olt_1/slot/port`.
- **`knownPONPorts` — used wrong column** — Was selecting `DISTINCT frame, port` and passing `[frame, port]` as `[board, pon]` to `PonIndex()`. Fixed to select `DISTINCT slot, port` and pass `[slot, port]`.
- **`GET /api/olt/:id/chassis` — wrong response shape** — Returned `{"ports": [...]}` but frontend expects `{"success": true, "chassis": [ApiChassisSlot]}`. Rebuilt handler to aggregate ONU port data from DB into proper slot-level chassis structure with card type inference (GTGO/GTGH/GTGQ) and uplink slot (SMXA at index 15).
- **OLT Management "0 ONUs"** — `GET /api/network/olts` returned raw `NetworkOLT` records without `_count` or `onu_stats`. Frontend reads `olt._count?.olt_onu_status`. Fixed by adding per-OLT status aggregation query; response now includes `_count.olt_onu_status` and `onu_stats` with per-status breakdown.
### Files
- `internal/olt/vendors/zte/zte.go` — fix `decodePonIndex`: `frame=1` always, `slot=board`, `port=pon`
- `internal/olt/poller/poller.go` — fix `knownPONPorts`: select `slot, port` not `frame, port`
- `internal/api/handlers/olt.go` — rewrite `GetChassis` to return `{success, chassis}` with ApiChassisSlot format
- `internal/api/handlers/network_ext.go` — `ListOLTs` now returns `_count` and `onu_stats` per OLT

### v2.52.59 — 2026-05-21

### Fixed
- **OLT detail page crash — TypeError: Cannot read properties of undefined (reading 'length')** — GORM JSON omits empty arrays (due to `omitempty`) so `onuStatuses`, `alerts`, `routers`, `monitoringLogs`, `performanceMetrics` are `undefined` on the frontend when empty. Added `?? []` normalization after `data.olt` is received in `fetchOLT`.
### Files
- `src/app/admin/olt/[id]/page.tsx` — normalize all relation arrays to `[]` after fetch

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
