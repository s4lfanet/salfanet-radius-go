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

### v2.52.70 — 2026-05-21

### Fixed
- **Uptime N/A di monitoring OLT** — Poller tidak pernah mengambil data uptime dari SNMP. Tambah SNMP GET untuk OID `1.3.6.1.2.1.1.3.0` (sysUpTime, centiseconds) di setiap siklus poll; konversi ke detik dan simpan ke field `uptime` di `network_olts`.
- **ONU count = 0 di OLT Management list** — GORM raw SQL scan per-OLT menggunakan field `Count int64` tanpa tag → nama kolom ambigu. Ganti pendekatan: satu query agregat dengan alias eksplisit (`SELECT oltId AS olt_id, status, COUNT(*) AS cnt FROM olt_onu_status GROUP BY oltId, status`) dan gunakan map untuk distribusikan ke tiap OLT. Eliminasi N+1 queries sekaligus.
- **Status OLT hanya tampil SSH** — `NetworkOLTStatus` handler mengecek Telnet hanya jika SSH gagal (`if !sshOK`). Fix: cek SSH dan Telnet secara paralel (goroutine terpisah), keduanya selalu dicek.
- **Badge SNMP tidak muncul di status** — Tambah field `SNMPEnabled` ke `oltRow` query dan tambah badge SNMP (orange) di frontend status details.
- **Tidak ada notifikasi saat Poll OLT** — Tambah CyberToast notification di `handleManualPoll` (poll selesai / gagal) dan `handlePollAll` (mulai polling + selesai) di halaman OLT Monitoring.
### Files
- `internal/olt/poller/poller.go` — tambah SNMP GET sysUpTime; update field `uptime` di Updates map
- `internal/api/handlers/network_ext.go` — ganti N+1 per-OLT query dengan single aggregated query + explicit column aliases
- `internal/api/handlers/misc_handler.go` — SSH+Telnet checked in parallel; tambah SNMPEnabled field + SNMP badge logic
- `src/app/admin/network/olts/page.tsx` — tambah `snmp` ke `OLTStatus.details` interface; tampilkan badge SNMP; urutan badge SSH > TEL > SNMP
- `src/app/admin/olt/monitoring/page.tsx` — tambah `useToast`; notifikasi di `handleManualPoll` dan `handlePollAll`

### v2.52.69 — 2026-05-21

### Fixed
- **Password OLT tidak tampil di list** — `Password *string json:"-"` di model mencegah password dikembalikan di API (by design untuk keamanan). Tambah field `hasPassword bool` di response `ListOLTs` yang bernilai `true` jika password sudah tersimpan. Frontend kini menampilkan `••••••••` jika `hasPassword=true`, `-` jika belum diset.
- **Stats card teks terlalu mepet ke border** — Padding kiri kartu stats (Status/Uptime/ONUs) diperbesar dari `p-4` ke `pl-6 pr-4 py-4` agar teks tidak terlalu dekat dengan garis `border-l-4`.
### Files
- `internal/api/handlers/network_ext.go` — tambah `HasPassword bool json:"hasPassword"` ke `oltWithStats`, populate dari `o.Password != nil && *o.Password != ""`
- `src/app/admin/network/olts/page.tsx` — tambah `hasPassword?: boolean` ke interface OLT; gunakan `olt.hasPassword` untuk display password di tabel dan detail panel
- `src/app/admin/olt/[id]/page.tsx` — ubah padding stats card dari `p-4` ke `pl-6 pr-4 py-4`

### v2.52.68 — 2026-05-21

### Fixed
- **ONU Detail Modal kosong** — ZTE C320 menggunakan `--More--` paging untuk output panjang (show gpon onu detail-info). `readUntilPrompt` tidak pernah melihat prompt `#` karena pager menginterupsi output → timeout 10 detik → output kosong. Fix: kirim `terminal length 0` setelah login berhasil untuk menonaktifkan paging di seluruh sesi telnet.
- **Jarak ONU semua sama 328m** — OID `.21` (zxAnGponOnuOptDistance) mengembalikan nilai fixed 328 untuk semua ONU (equalization delay provisioning, bukan jarak sebenarnya). Ganti ke OID `.19` (zxAnPonAniOptRtt) yang mengembalikan RTT aktual dalam nanoseconds per ONU. Konversi: `jarak_m = RTT_ns / 10` (sesuai kecepatan cahaya di fiber ≈ 2×10⁸ m/s). Contoh: ONU1=11627 ns → 1163m, ONU9=15367 ns → 1537m.
- **Uplink SMXA card status** — Setelah fix telnet paging, `show interface port-status` sekarang mengembalikan output lengkap → status active/disable port uplink SMXA ditampilkan dengan benar.
- **Stats card ONU list** — Perbaiki padding kartu (`p-4` konsisten), ukuran font lebih jelas (`text-2xl`), label uppercase tracking, dan breakdown status ONU (offline, LOS, DyingGasp) ditampilkan lebih detail.
### Files
- `internal/olt/telnet/telnet.go` — tambah `terminal length 0` setelah login berhasil untuk disable paging
- `internal/olt/vendors/zte/zte.go` — ubah `oidDistance` dari `.21` ke `.19`; konversi RTT → jarak `int(float64(dist)/10.0)`; filter batas atas dari 100000 ke 1000000
- `src/app/admin/olt/[id]/page.tsx` — stats cards: `p-4` konsisten, `text-2xl`, label uppercase, grid `grid-cols-3`, breakdown LOS/DyingGasp

### v2.52.67 — 2026-05-21

### Fixed
- **RxPower formula salah (lagi)** — Formula `10*log10(raw)-60` (nW asumsi) masih salah. Setelah menganalisis raw SNMP dari live device vs CSV referensi, ditemukan rumus yang benar: **`dBm = raw/500.0 - 30.0`**. Verifikasi: raw=6751 → -16.50 dBm ✓, raw=5085 → -19.83 dBm ✓, raw=5910 → -18.18 dBm ✓ (cocok dengan CSV). ZTE C320 menggunakan encoding linear: 0 raw = -30 dBm, step 0.002 dBm per unit.
- **ONU Name tidak muncul untuk FiberHome ONUs** — ONU FiberHome (prefix FHTTC/FHTT) mengembalikan `regStatus=2` dari SNMP, tapi kode hanya memproses `regStatus=1`. ONU regStatus=2 masuk ke loop fallback yang tidak membaca `description` dan masih menggunakan formula lama. Fix: terima regStatus=1 dan 2 sebagai "registered", baca description di kedua loop, gunakan formula yang benar di kedua loop.
- **Signal quality threshold** — Dikalibrasi ulang sesuai nilai aktual dari live device: rentang -10 s/d -20 dBm. Threshold baru: ≥ -14 Excellent, ≥ -18 Good, ≥ -22 Fair, < -22 Poor.
### Files
- `internal/olt/vendors/zte/zte.go` — formula rxPower & txPower → `raw/500.0 - 30.0`; filter regStatus menerima 1 dan 2; loop fallback tambah description + formula benar; hapus import `math`
- `src/app/admin/olt/[id]/page.tsx` — signal thresholds dikalibrasi

### v2.52.66 — 2026-05-21

### Fixed
- **RxPower formula salah** — ZTE C320 SNMP OID `.3.50.12.1.1.10` mengembalikan nilai integer dalam satuan **nanowatt (nW)**, bukan milli-dBm. Formula lama `dBm = -raw/1000` menghasilkan nilai seperti -7 dBm (salah). Formula benar: `dBm = 10 × log10(raw) - 60`. Contoh: raw=7540 nW → -21.2 dBm ✓. Nilai rxPower lama di DB di-reset agar sync berikutnya menulis nilai yang benar.
- **ONU count tidak muncul di OLT Management list** — `ListOLTs` hanya mengembalikan data OLT mentah tanpa jumlah ONU. Frontend mengakses `olt._count.olt_onu_status` dan `olt.onu_stats` yang selalu `undefined`, sehingga tampil "0 ONU". Handler sekarang menjalankan satu query GROUP BY untuk mendapatkan jumlah ONU per OLT per status, lalu menyertakan `_count` dan `onu_stats` di setiap respons OLT.
- **Signal quality threshold** — Dikalibrasi ulang untuk ONU downstream RX power (nW formula baru): ≥ -20 dBm Excellent, ≥ -24 Good, ≥ -27 Fair, < -27 Poor.
### Files
- `internal/olt/vendors/zte/zte.go` — rxPower dan txPower: formula diubah ke `10*math.Log10(raw) - 60`; import `math` ditambahkan; komentar OID diupdate
- `internal/api/handlers/olt.go` — `ListOLTs`: tambah GROUP BY query untuk `onu_stats` dan `_count`; tambah `Preload("Routers.Router")`
- `src/app/admin/olt/[id]/page.tsx` — signal quality thresholds untuk downstream ONU power

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
