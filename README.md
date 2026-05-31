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

### v2.52.98 — 2026-05-31

### Fixed / Improved
- **PageSpeed Accessibility** — Hapus `userScalable: false` & `maximumScale: 1` dari viewport di 3 layout (root, agent, technician) agar pinch-zoom tidak diblokir
- **PageSpeed Accessibility** — Tambah `aria-label` pada tombol ikon tanpa teks: show/hide password (admin login) dan toggle tema (customer login)
- **PageSpeed Accessibility** — Ubah wrapper root admin login dari `<div>` menjadi `<main>` landmark
- **PageSpeed Best Practices** — Tambah header `Cross-Origin-Opener-Policy: same-origin-allow-popups` dan `Cross-Origin-Resource-Policy: same-site` di `next.config.ts`
- **PageSpeed SEO** — Perbaiki `robots.txt`: izinkan `/customer/login` diindex (halaman produk utama), blokir hanya path sensitif
- **PageSpeed SEO** — Perbaiki metadata customer layout: tambah `description`, `robots: index`, `openGraph`
- **Performance** — Kurangi weight Outfit font dari 5 ke 4 (hapus weight 300 yang tidak terpakai)
- **Dependency** — Tambah `qrcode.react` yang sebelumnya missing dari `package.json`
### Files
- `src/app/layout.tsx` — Hapus userScalable, kurangi Outfit weight ke 400/500/600/700
- `src/app/agent/layout.tsx` — Hapus userScalable dari viewport
- `src/app/technician/layout.tsx` — Hapus userScalable dari viewport
- `src/app/admin/login/page.tsx` — aria-label show/hide password, wrap dalam `<main>`
- `src/app/customer/login/page.tsx` — aria-label theme toggle
- `src/app/customer/layout.tsx` — Metadata description + openGraph + robots
- `next.config.ts` — Tambah COOP + CORP header
- `public/robots.txt` — Allow customer portal, block sensitive paths saja

### v2.52.97 — 2026-06-01

### Fixed
- **ONUDetail: telnet pool race condition → ONU count naik 55→66, banyak offline** — Handler `ONUDetail` sebelumnya memakai shared telnet pool milik poller (`h.poller.GetPool`). Pool `acquire()` tidak menahan lock per-batch, sehingga dua goroutine bisa mendapat sesi yang sama dan command saling interleave. Akibatnya `FetchTelnetONUStates` mendapat output kacau → override state gagal → SNMP state yang lag dipakai → ONU tampil offline; sesi telnet crash di tengah poll → ghost ONU cleanup salah menandai ONU sebagai offline dan jumlah ONU melonjak. Solusi: ONUDetail **selalu membuat private telnet pool** (bukan reuse shared pool), agar poller tidak terganggu.
### Files
- `internal/api/handlers/misc_handler.go` — Ganti `h.poller.GetPool(oltID)` + conditional `ownPool` menjadi selalu `telnet.NewPool(tcfg)` + `defer pool.Close()`

### v2.52.96 — 2026-06-01

### Added
- **ONU detail: TCONT/DBA profile bandwidth** — Panel "TCONT / DBA Profile (Upload)" kini menampilkan bandwidth type dan max bandwidth (Mbps) dari profil DBA yang digunakan ONU, diambil via `show gpon profile tcont`.
- **ONU detail: Traffic Profile (Download) nama + bandwidth** — Panel "Traffic Profile (Download)" menampilkan nama profil (misal DOWN-PPPOE) beserta SIR/PIR dalam Mbps, diambil via `show gpon profile traffic`. Sebelumnya selalu tampil "(dari TCONT)".
- **ONU detail: Build Script lengkap** — Section baru "Build Script (Reproducible Config)" menampilkan skrip CLI ZTE lengkap 3 step: Step 1 registrasi ONU di OLT port, Step 2 konfigurasi interface ONU (tcont/gemport/service-port), Step 3 pon-onu-mng OMCI (service-gemport-vlan + TR-069 jika GenieACS aktif).
- **GetTrafficProfiles** — Fungsi baru di zte.go untuk mengambil daftar downstream traffic profile dari OLT.
- **GetRegisterMetadata: trafficProfiles** — API `/api/olt/:id/register-metadata` sekarang juga mengembalikan `trafficProfiles[]` untuk keperluan form registrasi.
### Fixed
- **GetTcontProfiles command invalid** — Sebelumnya menggunakan `show gpon traffic-profile` (tidak valid di ZTE C320 V2.1), diganti ke `show gpon profile tcont` yang benar.
### Files
- `internal/api/handlers/misc_handler.go` — Tambah `oltIface`, 3 command baru ke `ExecuteMultiple`, tipe `GponTcontProfile`/`GponTrafficProfile`, fungsi `parseGponTcontProfiles`, `parseGponTrafficProfiles`, `parseOltRegistrationLine`, `generateONUBuildScript`; update response `oltProfiles` dan `buildScript`
- `internal/olt/vendors/zte/zte.go` — Perbaiki `GetTcontProfiles` + `parseTcontProfiles`, tambah `TrafficProfile` type + `GetTrafficProfiles` + `parseTrafficProfiles`
- `internal/api/handlers/olt.go` — Tambah `GetTrafficProfiles` call dan `trafficProfiles` ke response metadata
- `src/app/admin/olt/[id]/page.tsx` — Tambah `oltProfiles`, `buildScript`, `usedTcontProfile`, `activeTrafficProfiles`; update card TCONT + Traffic Profile; tambah section Build Script

### v2.52.95 — 2026-05-31

### Added
- **ONU detail: Traffic Profile dari PPPoE** — Karena ZTE C320 tidak mengekspos DBA/bandwidth profile via CLI, limit bandwidth sekarang diambil dari profil PPPoE customer yang terhubung. Menampilkan nama profil, download limit, dan upload limit (dalam Kbps/Mbps) di section "ONT Build Configuration".
- **ONU detail: TR-069 / GenieACS status** — Section baru menampilkan apakah GenieACS sudah dikonfigurasi di sistem. Jika ya, tampilkan host dan username; jika tidak, tampilkan pesan panduan konfigurasi.
- **ONU detail: preload Customer.Profile** — Backend sekarang melakukan `Preload("Customer.Profile")` sehingga data profil PPPoE (nama, kecepatan download/upload) tersedia di respons API.
### Files
- `internal/api/handlers/misc_handler.go` — Ganti `Preload("Customer")` ke `Preload("Customer.Profile")`, tambah GenieACS settings check, tambah `bandwidth` dan `tr069` ke respons
- `src/app/admin/olt/[id]/page.tsx` — Tambah `bandwidth` dan `tr069` dari respons API, tambah "Traffic Profile (PPPoE Radius)" cards di ONT Build Config, tambah section "TR-069 / GenieACS"

### v2.52.94 — 2026-05-31

### Added
- **ONU detail modal: traffic statistics realtime** — Tambah command `show interface gpon-onu_F/S/P:N` ke Telnet fetch. Downstream/upstream rate, bandwidth throughput %, peak rate, dan total bytes sekarang tampil di section "Traffic Statistics (Realtime)".
- **ONU detail modal: service ports table** — Running-config parser sekarang mengekstrak setiap `service-port` sebagai objek `{servicePort, vport, userVlan, vlan}` dan ditampilkan sebagai tabel di section "ONT Build Configuration".
- **ONU detail modal: TCONT & GEM port tables** — Setiap `tcont N profile <name>` dan `gemport N tcont M` diparsing dan ditampilkan sebagai tabel terpisah.
- **ONU detail modal: riwayat koneksi ONU** — Tabel auth history dari `show gpon onu detail-info` (AuthpassTime / OfflineTime / Cause) sekarang diparsing dan ditampilkan di section "Riwayat Koneksi ONU".
- **ONU detail modal: field tambahan** — Tambah FEC, ONU Status, Multicast Encryption, dan Description ke technical items. TX Power juga ditampilkan di basic info.
### Fixed
- **Description di summary** — Sebelumnya menggunakan `kv["Name"]` (nama pendek), sekarang menggunakan `kv["Description"]` (deskripsi lengkap) dari ZTE output.
- **Empty string OMCI BW Profile menampilkan blank** — Ganti `??` ke `||` di frontend agar string kosong juga di-fallback ke 'N/A'.
- **Downstream Profile selalu N/A** — ZTE C320 running-config tidak memiliki keyword "downstream" untuk ONU tanpa traffic-limit config. Label diubah menjadi info yang lebih akurat `(dari TCONT)`.
### Files
- `internal/api/handlers/misc_handler.go` — Tambah `show interface` command, `onuParseInterfaceStats()`, perbaikan `onuParseDetailInfo()` (auth history, lebih banyak field), perbaikan `onuParseRunningConfig()` (servicePorts, gemports, tconts, name, description)
- `src/app/admin/olt/[id]/page.tsx` — Update `ONUDetailModal`: traffic stats section, build config tables (service ports, TCONT, GEM), auth history table, field tambahan

### Fixed
- **RX Power ONU 1/1/1:1 (MARLINARINA) selalu tampil "—"** — Root cause: ONU 1/1/1:1 secara *intermittent* mengembalikan `rxRaw=65535` (0xFFFF = sentinel ZTE "no measurement") via SNMP, terjadi bergantian dengan nilai valid (misal 7465 → -15.07 dBm). Sebelum v2.52.92, setiap kali SNMP mengembalikan 65535, upsert menimpa nilai DB dengan NULL (karena `VALUES(rxPower)` tanpa COALESCE). Akibatnya rxPower selalu NULL karena siklus overwrite terus berulang. Fix v2.52.92 (COALESCE) sudah menyelesaikan masalah ini — dikonfirmasi via debug log bahwa DB sekarang menyimpan -15.07 dBm untuk ONU 1. Debug log dihapus setelah analisis selesai.
### Files
- `internal/olt/vendors/zte/zte.go` — Hapus debug logging sementara (tidak ada perubahan logic)

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
