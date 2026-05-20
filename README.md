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

### v2.52.58 — 2026-05-21

### Fixed
- **GET /api/olt/:id — 404** — Root cause: `GetOLT` handler uses `Preload("MonitoringLogs", Order("created_at DESC"))` dan `Preload("PerformanceMetrics", Order("recorded_at DESC"))`. DB columns are camelCase (`createdAt`, `recordedAt`), bukan snake_case. MySQL error "Unknown column 'created_at'" membuat seluruh query gagal → handler return 404.
- **`NetworkOLT` model — tidak ada column tags** — Semua field compound (`ipAddress`, `snmpEnabled`, `isOnline`, `totalOnu`, dst.) dipetakan GORM sebagai snake_case (`ip_address`, `snmp_enabled`, `is_online`, `total_onu`, dst.) tapi DB pakai camelCase. CREATE/UPDATE/SELECT semua salah.
- **`olt_ext.go` Monitoring SELECT — snake_case columns** — `Select("id,name,ip_address,status,is_online,total_onu,online_onu,offline_onu")` → MySQL error "Unknown column". Diganti ke `"id,name,ipAddress,status,isOnline,totalOnu,onlineOnu,offlineOnu"`.
### Files
- `internal/db/models/olt.go` — tambah `gorm:"column:..."` camelCase ke semua field `NetworkOLT`: `ipAddress`, `followRoad`, `firmwareVersion`, `snmpEnabled`, `snmpCommunity`, `snmpPort`, `telnetEnabled`, `telnetPort`, `sshEnabled`, `sshPort`, `monitoringEnabled`, `pollingInterval`, `lastPollAt`, `isOnline`, `totalOnu`, `onlineOnu`, `offlineOnu`, `createdAt`, `updatedAt`
- `internal/api/handlers/olt.go` — fix `GetOLT` preload order: `created_at` → `createdAt`, `recorded_at` → `recordedAt`
- `internal/api/handlers/olt_ext.go` — fix `Monitoring` SELECT ke camelCase column names

### v2.52.57 — 2026-05-22

### Fixed
- **TSC Errors (9 errors → 0)** — Semua TypeScript compile error diperbaiki:
  - `invoice-templates/page.tsx` — `variant="ghost"` → `variant="secondary"` (2 tempat, `ghost` tidak valid di `ModalButton`)
  - `laporan/analitik/page.tsx` — `s.avgChurnRate` possibly undefined → `(s.avgChurnRate ?? 0)`
  - `pay/[token]/page.tsx` — tambah `hasListener?: boolean` ke tipe state `qrisOwn`
  - `cron/runner.ts` — `(prisma as any).cronScheduleConfig.findMany()` + tambah type untuk callback parameter
  - `lib/genieacs/api-client.ts` — import dari route yang tidak ada → pindahkan `getGenieACSCredentials` ke `src/lib/genieacs/credentials.ts`
  - `qrcode.react` — buat type declaration `src/types/qrcode.react.d.ts` (package tidak tersedia lokal)
- **i18n — 72 missing translation keys** — Tambah semua key yang hilang ke `src/locales/id.json`:
  - Top-level `payment.*` namespace (38 keys dari `customer.payment.*`)
  - `pppoe.balance.*` (10 keys baru untuk halaman riwayat saldo)
  - `pppoe.eWallet`, `pppoe.monthlyDueDateDesc`, `pppoe.profileMikrotik`
  - `invoices.pdfBillTo/HeaderPrice/HeaderQty/HeaderTotal`
  - `keuangan.pdfAmount/Category/Description/Type`
  - `network.common.nodes`, `network.otb.output`, `network.routerCreated`
  - `ticket.replySent/statusUpdated/priorityUpdated`
  - `common.coordinates/done/subtract`
  - `hotspot.autoGenerate/of`
  - `settings.pageWillReload`
- **Go Model Column Tags — Komprehensif** — Semua model Go di `internal/db/models/models.go` dan `olt.go` kini memiliki explicit `gorm:"column:camelCase"` tags agar INSERT/UPDATE/CREATE menggunakan nama kolom yang benar (Prisma default = camelCase, GORM default = snake_case):
  - `PppoeProfile` — `downloadSpeed`, `uploadSpeed`, `rateLimit`, `groupName`, `mikrotikProfileName`, `ipPoolName`, `ipPoolRange`, `localAddress`, `ppnActive`, `ppnRate`, `isActive`, `validityUnit`, `validityValue`, `sharedUser`, `createdAt`, `updatedAt`
  - `PppoeUser` — `profileId`, `areaId`, `ipAddress`, `macAddress`, `expiredAt`, `routerId`, `subscriptionType`, `lastPaymentDate`, `billingDay`, `autoIsolationEnabled`, `autoRenewal`, `connectionType`, `referralCode`, `syncedToRadius`, `createdAt`, `updatedAt`
  - `PppoeCustomer` — `customerId`, `idCardNumber`, `idCardPhoto`, `isActive`, `areaId`, `createdAt`, `updatedAt`
  - `Invoice` — `invoiceNumber`, `userId`, `dueDate`, `paidAt`, `paymentLink`, `paymentToken`, `customerName`, `customerPhone`, `customerEmail`, `customerUsername`, `sentReminders`, `invoiceType`, `baseAmount`, `createdAt`, `updatedAt`
  - `Router` — `ipAddress`, `apiPort`, `isActive`, `createdAt`, `updatedAt`
  - `Company` — semua compound fields: `adminPhone`, `baseUrl`, `poweredBy`, `customerIdPrefix`, `invoiceGenerateDays`, `gracePeriodDays`, `isolationEnabled`, dll. + `createdAt`, `updatedAt`
  - `QrisPending` — `invoiceId`, `userId`, `orderId`, `baseAmount`, `uniqueAmount`, `qrString`, `sourceApp`, `expiresAt`, `paidAt`, `createdAt`, `updatedAt`
  - `CustomerSession` — `userId`, `otpCode`, `otpExpiry`, `expiresAt`, `createdAt`, `updatedAt`
  - `WhatsappProvider` — `apiKey`, `apiUrl`, `senderNumber`, `isActive`, `createdAt`, `updatedAt`
  - `WhatsappTemplate` — `isActive`, `createdAt`, `updatedAt`
  - `ManualPayment` — `createdAt`, `updatedAt`
  - `Payment` — `invoiceId`, `gatewayId`, `paidAt`, `createdAt`
  - `Ticket` — `ticketNumber`, `customerId`, `customerName`, `customerEmail`, `customerPhone`, `categoryId`, `assignedToId`, `assignedToType`, `closedAt`, `resolvedAt`, `createdAt`, `updatedAt`
### Added
- `src/lib/genieacs/credentials.ts` — standalone helper `getGenieACSCredentials()` yang membaca dari tabel `genieacsSettings` via Prisma
- `src/types/qrcode.react.d.ts` — type declaration untuk `qrcode.react` (QRCodeSVG, QRCodeCanvas)
### Files
- `src/app/admin/invoice-templates/page.tsx` — ghost→secondary
- `src/app/admin/laporan/analitik/page.tsx` — null coalescing untuk avgChurnRate
- `src/app/pay/[token]/page.tsx` — tambah hasListener ke qrisOwn type
- `src/cron/runner.ts` — type cast untuk cronScheduleConfig
- `src/lib/genieacs/api-client.ts` — fix import credentials
- `src/lib/genieacs/credentials.ts` — file baru
- `src/types/qrcode.react.d.ts` — file baru
- `src/locales/id.json` — tambah 72+ missing translation keys
- `internal/db/models/models.go` — komprehensif camelCase column tags
- `internal/db/models/olt.go` — sudah di-fix di [2.52.56]

### v2.52.56 — 2026-05-22

### Fixed
- **GET /api/olt/:id — 404 meskipun route terdaftar** — `GetOLT` handler melakukan preload `ONUStatuses`, `Alerts`, `MonitoringLogs`, dan `PerformanceMetrics` dengan GORM auto-naming. GORM mengubah `OltID` → `olt_id`, `OnuID` → `onu_id`, dst., padahal kolom DB (Prisma) adalah camelCase (`oltId`, `onuId`). MySQL melempar error "Unknown column 'olt_id'" yang diperlakukan sebagai 404. Fix: tambah explicit `gorm:"column:..."` camelCase tags ke semua field di `OLTONUStatus`, `OLTAlert`, `OLTPerformanceMetric`, `OLTMonitoringLog`.
- **ONU data selalu 0/0 di monitoring** — Poller `CreateInBatches` OLT ONU Status gagal karena INSERT menggunakan nama kolom snake_case (`olt_id`, `onu_id`, `mac_address`, dst.) yang tidak ada di DB. Dengan adanya column tags, INSERT sekarang menggunakan nama kolom yang benar (`oltId`, `onuId`, `macAddress`, dst.).
- **Poller `knownPONPorts` — WHERE clause salah** — `Where("olt_id = ?", oltID)` diganti ke `Where("oltId = ?", oltID)`.
- **Poller `checkAlerts` — WHERE clause salah** — raw SQL dengan `olt_id`, `onu_id`, `alert_type`, `is_resolved` diganti ke camelCase `oltId`, `onuId`, `alertType`, `isResolved`.
### Files
- `internal/db/models/olt.go` — `OLTONUStatus`, `OLTAlert`, `OLTPerformanceMetric`, `OLTMonitoringLog`: tambah `gorm:"column:camelCase"` tags ke semua field
- `internal/olt/poller/poller.go` — fix `knownPONPorts` dan `checkAlerts` WHERE clause ke camelCase

### v2.52.55 — 2026-05-22

### Fixed
- **OLT detail page — JS crash `Cannot read properties of undefined (reading 'length')`** — `olt.monitoringLogs` undefined karena `GetOLT` tidak me-preload `MonitoringLogs` / `PerformanceMetrics`. Fix: tambah preload dengan ORDER + LIMIT 100, serta null-safety `??[]` di frontend.
- **OLT detail page — password terhapus saat Save Settings** — `OLTHandler.UpdateOLT` menggunakan `h.db.Save(&body)` dengan struct `NetworkOLT`; field `Password *string json:"-"` tidak ter-bind oleh Fiber sehingga nilainya `nil`, lalu GORM Save menghapus password di DB. Fix: ganti ke map-based update (`map[string]interface{}`), skip key `password` jika kosong.
- **OLT router associations — kolom DB camelCase vs GORM snake_case mismatch** — GORM default naming strategy menghasilkan `olt_id`, `router_id` dst., tapi Prisma membuat kolom dengan nama camelCase (`oltId`, `routerId`). Fix: tambah explicit `gorm:"column:camelCase"` tags ke model `NetworkOLTRouter`.
- **OLT status check — kolom DB salah** — `NetworkOLTStatus` menggunakan `SELECT id, ip_address, ssh_enabled...` dan `UPDATE is_online` dengan nama snake_case. Fix: ganti ke `ipAddress`, `sshEnabled`, `sshPort`, `telnetEnabled`, `telnetPort`, `isOnline` (camelCase sesuai skema Prisma).
- **OLT router delete — WHERE clause salah** — `h.db.Where("olt_id = ?", id)` seharusnya `"oltId = ?"` agar sesuai kolom DB camelCase. Fix diterapkan di `olt.go` dan `network_ext.go`.
### Files
- `internal/db/models/olt.go` — `NetworkOLTRouter`: tambah explicit `gorm:"column:..."` camelCase tags + relasi `MonitoringLogs`/`PerformanceMetrics` ke `NetworkOLT`
- `internal/api/handlers/olt.go` — `GetOLT`: preload semua relasi; `UpdateOLT`: map-based update, skip empty password, fix `oltId` WHERE clause
- `internal/api/handlers/network_ext.go` — fix `WHERE "oltId = ?"` untuk delete router associations
- `internal/api/handlers/misc_handler.go` — `NetworkOLTStatus`: fix semua kolom DB ke camelCase
- `src/app/admin/olt/[id]/page.tsx` — null-safety `??[]` pada `olt.monitoringLogs`

### v2.52.54 — 2026-05-22

### Fixed
- **OLT Edit — router tidak tersimpan** — `UpdateOLT` menghapus `routerIds` dari body sebelum update sehingga relasi ke `network_olt_routers` tidak pernah disimpan. Fix: ekstrak `routerIds` terlebih dahulu, hapus baris lama, lalu insert ulang ke `network_olt_routers`.
- **OLT Create — router tidak tersimpan** — `CreateOLT` tidak memiliki field `routerIds` di struct body. Fix: tambah `RouterIDs []string` ke struct dan create `NetworkOLTRouter` records setelah OLT dibuat.
- **OLT Edit — password terhapus jika dikosongkan** — GORM `Updates` dengan map menyertakan empty string sebagai nilai update. Jika user tidak mengisi ulang password di form edit, password lama terhapus. Fix: hapus key `password` dari map jika nilainya kosong.
- **OLT list — Status selalu "Checking" / tidak muncul** — `NetworkOLTStatus` query tabel `olts` lama (legacy) dan mengembalikan `{olts, count}` sedangkan frontend mengharapkan `{statusMap: {[id]: {online, details}}}`. Fix: query tabel `network_olts`, lakukan TCP check ke SSH/Telnet port secara concurrent, kembalikan format yang benar.
- **OLT Edit form — router tidak ter-preload** — `ListOLTsForMap` tidak preload router associations, sehingga form Edit selalu menampilkan checklist kosong. Fix: tambah `Preload("Routers.Router")` dan tambah relasi `Routers` + nested `Router` ke model.
### Files
- `internal/api/handlers/network_ext.go` — `UpdateOLT`: save router associations, skip empty password; `CreateOLT`: tambah `RouterIDs` + create associations
- `internal/api/handlers/misc_handler.go` — `NetworkOLTStatus`: rewrite untuk query `network_olts`, TCP check concurrent, return `{statusMap}`
- `internal/api/handlers/network.go` — `ListOLTsForMap`: tambah `Preload("Routers.Router")`
- `internal/db/models/olt.go` — `NetworkOLT`: tambah relasi `Routers`; `NetworkOLTRouter`: tambah nested `Router`

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
