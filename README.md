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

### v2.52.78 — 2026-05-22

### Fixed
- **Remove VLAN 500 error** — Perintah `no switchport vlan X tag` tidak valid di ZTE C320 (mengembalikan `%Error 20201: Invalid command key word`). Perintah yang benar adalah `no switchport vlan X` (tanpa suffix `tag`). Sebelumnya juga ada dua commandSet — fallback kedua (`no switchport default vlan`) salah karena menghapus PVID bukan tagged VLAN. Kini hanya satu commandSet dengan perintah yang benar.
- **API response 500 → 422 saat CLI error** — Ketika perintah Telnet ditolak OLT (CLI error seperti `%Error`), handler mengembalikan HTTP 500 "Uplink action failed" padahal ini bukan server error. Fix: kembalikan HTTP 422 Unprocessable Entity dengan detail pesan error dari OLT, sehingga frontend dapat menampilkan pesan yang tepat.
### Changed
- **Toggle Enable/Disable port** — Dua tombol terpisah (Enable + Disable) di status tab uplink diganti dengan satu tombol kontekstual: jika port sedang enabled tampil tombol "Disable Port" (merah), jika port disabled tampil tombol "Enable Port" (hijau).
### Files
- `internal/api/handlers/olt.go` — Fix `removeVlan` command dari `no switchport vlan X tag` → `no switchport vlan X`; remove wrong fallback commandSet; return 422 instead of 500 on CLI errors
- `src/app/admin/olt/[id]/page.tsx` — Replace Enable+Disable buttons with single contextual toggle button

### v2.52.77 — 2026-05-22

### Fixed
- **Uplink status "Unknown"** — `uplinkParsePortStatus` membaca kolom index yang salah dari output `show interface port-status xgei_1/3/2`. ZTE C320 mengembalikan 9 field (index 0–8): port, hybridStatus, nativeVlan, negotiation, speed, duplex, flowCtrl, adminStatus, linkStatus. Kode lama salah baca duplex dari `parts[3]` (seharusnya `parts[5]`), flowCtrl dari `parts[8]` (seharusnya `parts[6]`), adminStatus dari `parts[9]` (OOB), linkStatus dari `parts[10]` (OOB). Akibatnya `adminRaw` selalu kosong → status "Unknown".
- **Uplink status "Unknown" (fallback)** — `uplinkParseInterfaceStatus` memiliki regex `stateRe` yang hanya menangani `activate|deactivate`. Namun ZTE C320 untuk interface uplink (`xgei`) melaporkan `xgei_1/3/2 is up, line protocol is up` bukan `activate`. Fix: tambahkan `up|down` ke pattern. Update pemetaan: `up` → "Up", `down` → "Down".
- **CONFIG tab menampilkan `%Error 20202`** — Handler `case "config"` menetapkan `raw = out` sebelum memeriksa CLI error, sehingga teks error mentah tampil di UI. Fix: hanya set `raw = out` ketika output tidak mengandung CLI error.
- **Dark/light theme uplink modal** — Status indicator menggunakan warna hex hardcoded (`#111827`, `#052e16`, `#451a03`) via inline style tanpa dukungan light mode. Diganti dengan Tailwind classes `dark:` variants (`bg-gray-100 dark:bg-gray-900`, `bg-green-50 dark:bg-green-950/40`, dll).
- **PON expansion bg invalid** — `dark:bg-gray-850` bukan class Tailwind valid (Tailwind hanya memiliki 100–900). Diganti ke `dark:bg-gray-900`.
### Files
- `internal/api/handlers/olt.go` — Fix `uplinkParsePortStatus` column indices; fix `uplinkParseInterfaceStatus` stateRe + admin mapping; fix `case "config"` raw error handling
- `src/app/admin/olt/[id]/page.tsx` — Fix `statusTone` dari inline styles ke Tailwind dark: classes; fix `dark:bg-gray-850` → `dark:bg-gray-900`

### v2.52.76 — 2026-05-22

### Fixed
- **Distance via Telnet** — SNMP OID `.18` (equalization delay × 0.112) memberikan nilai yang tidak akurat untuk sebagian ONU (contoh: ONU 1/1/1:12 menampilkan "101.07 dBm" karena nilai raw OID terpetakan ke RxPower ONU lain akibat bug index). Fix: ambil jarak dari Telnet `show gpon onu detail-info gpon-onu_1/{slot}/{port}:{onuId}` yang langsung melaporkan `ONU Distance: Xm` (diukur dari proses ranging OLT). Semua ONU terdaftar di-query dalam satu sesi Telnet via `ExecuteMultiple`. Hasil Telnet override nilai SNMP; jika Telnet gagal, nilai SNMP tetap dipakai sebagai fallback.
- **Frontend tidak update** — Next.js berjalan via PM2 (`salfanet-radius`), bukan `systemctl salfanet-api`. Deploy sebelumnya hanya restart Go binary tanpa restart PM2, sehingga UI lama masih terbuffer. Fix: `rm -rf .next && npm run build && pm2 restart salfanet-radius`.
### Files
- `internal/olt/vendors/zte/zte.go` — Tambah `FetchTelnetDistances(pool, onus)` dan `parseTelnetDistances(raw)` untuk parsing `ONU Distance: Xm` dari combined Telnet output
- `internal/olt/poller/poller.go` — Di `poll()`: retrieve pool dari `p.pools[olt.ID]`; panggil `zte.FetchTelnetDistances` setelah SNMP discovery; override `onu.Distance` untuk setiap ONU yang mendapat data Telnet

### v2.52.75 — 2026-05-22

### Added
- **Per-PON Live Stats** — Setiap port PON di section "Detail Per Port PON" kini bisa di-klik untuk expand dan menampilkan data live dari Telnet OLT: Temperature (°C), TX Power (dBm), Voltage (V), Bias Current (mA), Upstream/Downstream rate (Mbps) dan bandwidth usage (%). Data diambil via `show interface gpon-olt_1/{slot}/{port}` dan `show interface optical-module-info gpon-olt_1/{slot}/{port}`.
### Fixed
- **Chassis port squares** — Port PON menunjukkan index 0 (0/1/0) pada tooltip dan green box di posisi ke-2; seharusnya 1-based. Fix: rebuild frontend (source sudah benar dengan `i+1`, VPS belum di-build ulang).
### Files
- `internal/api/handlers/olt_pon_stat.go` — Handler baru `GetPONStat`; parser `parsePONInterfaceStat`, `parsePONBps`, `parsePONPct`, `parsePONFloatFromUnit`
- `internal/api/router.go` — Route baru `GET /api/olt/:id/pon-stat`
- `src/app/admin/olt/[id]/page.tsx` — Tambah type `PONPortStat`; state `expandedPON`, `ponStatCache`, `loadingPON`; fungsi `fetchPONStat`; "Detail Per Port PON" expandable cards dengan Temperature, TX Power, Voltage, Upstream/Downstream

### v2.52.74 — 2026-05-22

### Fixed
- **Distance ONU semua NULL** — OID `.3.50.12.1.1.21` adalah nilai konstan per-port (bukan per-ONU), sehingga tidak ter-mapping ke masing-masing ONU dan distance tetap NULL di DB. Fix: ganti ke OID `.3.50.12.1.1.18` (equalization delay per-ONU, terindex per onuId) dengan formula `raw × 0.112` (diverifikasi: ONU 28 OID18=5000 → 5000×0.112=560m sesuai data Telnet).
### Files
- `internal/olt/vendors/zte/zte.go` — `oidDistance` dari OID `.21` ke OID `.18`; formula `int(dist)` → `int(float64(dist) * 0.112)` (2 tempat)

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
