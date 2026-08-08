# Roadmap Implementasi Salfanet-Radius
## Menuju ISP Management Platform Lengkap (Billing + NMS + Automation)

> Berdasarkan analisis mendalam billing.pmynet.id vs salfanet-radius-go  
> Tanggal: 8 Agustus 2026  
> Status: Ready for execution

---

## Daftar Isi

1. [Quick Wins (Mulai dari Sini)](#1-quick-wins-mulai-dari-sini)
2. [Phase 1: Foundation & Stabilization](#phase-1-foundation--stabilization)
3. [Phase 2: Billing Core](#phase-2-billing-core)
4. [Phase 3: Integration (OLT/ONU + Customer + Billing)](#phase-3-integration-oltonu--customer--billing)
5. [Phase 4: Automation & Smart Monitoring](#phase-4-automation--smart-monitoring)
6. [Phase 5: Scaling & Optimization](#phase-5-scaling--optimization)
7. [Dependency Graph](#dependency-graph)
8. [Rekomendasi Teknis](#rekomendasi-teknis)
9. [Database Schema Changes](#database-schema-changes)

---

## Status Legend

| Simbol | Arti |
|---|---|
| [✓] | Sudah ada di salfanet-radius, berfungsi baik |
| [~] | Sudah ada tapi perlu improvement/extension |
| [ ] | Belum ada, harus dibuat dari nol |

---

## 1. Quick Wins (Mulai dari Sini)

> Estimasi: 1-2 minggu  
> Impact: Langsung dirasakan operasional ISP  
> Risiko: Minimal (tidak mengubah flow yang sudah berjalan)

### 1.1 Invoice Discount

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Admin bisa berikan diskon ke invoice tertentu dengan alasan. Amount invoice berkurang, selisih dicatat untuk audit. |
| Dampak | **Bisnis**: fleksibilitas negosiasi pelanggan, retensi. **Operasional**: tidak perlu hapus+buat invoice baru. |
| Kompleksitas | **Low** |
| Prioritas | **High** |

**Checklist:**

- [ ] **DB**: Tambah kolom di `invoices` table:
  - `discountAmount` INT DEFAULT 0
  - `discountReason` TEXT NULL
  - `originalAmount` INT NULL (simpan amount sebelum diskon)
- [ ] **Backend**: `PUT /api/invoices/:id/discount` — body: `{ amount, reason }`, update invoice, log ke activity log
- [ ] **Backend**: Validasi: discount tidak boleh > amount, invoice harus status PENDING
- [ ] **Frontend**: Tombol "Diskon" di invoice detail modal
- [ ] **Frontend**: Modal input diskon + alasan + preview amount setelah diskon
- [ ] **Frontend**: Tampilkan diskon info di invoice card (badge "Diskon Rp X")

### 1.2 Cancel Invoice

| Item | Detail |
|---|---|
| Status | [~] Model sudah ada `InvoiceCancelled`, tapi tidak ada endpoint cancel |
| Deskripsi | Admin bisa batalkan invoice dengan alasan. Invoice berubah status CANCELLED, tidak muncul di tagihan aktif. |
| Dampak | **Operasional**: bersihkan invoice salah/void tanpa hapus permanen. |
| Kompleksitas | **Low** |
| Prioritas | **High** |

**Checklist:**

- [ ] **DB**: Tambah kolom di `invoices`:
  - `cancelledAt` TIMESTAMP NULL
  - `cancelledBy` VARCHAR(191) NULL (user ID admin)
  - `cancelReason` TEXT NULL
- [ ] **Backend**: `POST /api/invoices/:id/cancel` — body: `{ reason }`, set status=CANCELLED, cancelledAt, cancelledBy
- [ ] **Backend**: Validasi: hanya PENDING/OVERDUE yang bisa dibatalkan, PAID tidak bisa
- [ ] **Frontend**: Tombol "Batalkan" di invoice detail (dengan confirm dialog)
- [ ] **Frontend**: Filter "Dibatalkan" di invoice list
- [ ] **Frontend**: Badge status "DIBATALKAN" dengan alasan

### 1.3 Edit Payment Method Post-Lunas

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Admin bisa ubah metode pembayaran (cash/transfer/gateway) setelah invoice lunas, untuk koreksi data. |
| Dampak | **Operasional**: koreksi laporan keuangan yang salah input. |
| Kompleksitas | **Low** |
| Prioritas | **Medium** |

**Checklist:**

- [ ] **DB**: Tambah kolom `paymentMethodEditCount` INT DEFAULT 0 di `payments` table (audit)
- [ ] **Backend**: `PUT /api/payments/:id/method` — body: `{ method }`, update payment, increment editCount, log ke activity
- [ ] **Backend**: Validasi: method harus valid (cash/transfer/midtrans/xendit/tripay/qris)
- [ ] **Frontend**: Tombol "Edit Metode" di payment detail (hanya untuk admin)
- [ ] **Frontend**: Dropdown pilih metode baru + confirm

### 1.4 Skip Minggu untuk Auto-Isolir

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Auto-isolir cron skip hari Minggu (admin libur). Pelanggan yang jatuh tempo Minggu akan terisolir Senin karena query pakai `dueDate <= now`. |
| Dampak | **Operasional**: tidak ada isolir di hari libur, kurangi komplain weekend. |
| Kompleksitas | **Low** |
| Prioritas | **High** |

**Checklist:**

- [ ] **Backend**: Modifikasi `internal/cron/auto_isolate.go` (atau `suspend_check.go`):
  ```go
  loc, _ := time.LoadLocation("Asia/Jakarta")
  now := time.Now().In(loc)
  if now.Weekday() == time.Sunday {
      log.Info().Msg("auto-isolir: skip hari Minggu")
      return
  }
  ```
- [ ] **Backend**: Tambah log "Skip Minggu, akan diproses Senin"
- [ ] **Frontend**: Tampilkan info "Auto-isolir skip Minggu" di settings page
- [ ] **Settings**: Tambah toggle `skipSundayIsolate` di Company settings (default: true)

### 1.5 Package Change Log

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Setiap perubahan paket pelanggan dicatat: paket lama, paket baru, siapa yang ubah, kapan, alasan. |
| Dampak | **Operasional**: audit trail untuk dispute pelanggan. **Bisnis**: analisis upgrade/downgrade pattern. |
| Kompleksitas | **Low** |
| Prioritas | **Medium** |

**Checklist:**

- [ ] **DB**: Buat table `package_change_logs`:
  ```sql
  CREATE TABLE package_change_logs (
    id VARCHAR(191) PRIMARY KEY,
    userId VARCHAR(191) NOT NULL,
    username VARCHAR(191) NOT NULL,
    oldProfileId VARCHAR(191) NULL,
    oldProfileName VARCHAR(191) NULL,
    newProfileId VARCHAR(191) NULL,
    newProfileName VARCHAR(191) NULL,
    changedBy VARCHAR(191) NOT NULL,
    changedByName VARCHAR(191) NULL,
    reason TEXT NULL,
    changedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user (userId),
    INDEX idx_date (changedAt)
  );
  ```
- [ ] **Backend**: Buat model `PackageChangeLog` di `internal/db/models/`
- [ ] **Backend**: Modifikasi `pppoe_ext.go` — saat ganti paket, insert log
- [ ] **Backend**: `GET /api/users/:id/package-logs` — list riwayat perubahan paket
- [ ] **Frontend**: Tab "Riwayat Paket" di customer detail page
- [ ] **Frontend**: Tabel: tanggal, paket lama → paket baru, oleh, alasan

### 1.6 Installation Log

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Log permanen pemasangan baru: siapa, kapan, di mana, koordinat GPS, territory, installer. |
| Dampak | **Operasional**: audit pemasangan, data untuk laporan PSB. |
| Kompleksitas | **Low** |
| Prioritas | **Medium** |

**Checklist:**

- [ ] **DB**: Buat table `installation_logs`:
  ```sql
  CREATE TABLE installation_logs (
    id VARCHAR(191) PRIMARY KEY,
    userId VARCHAR(191) NOT NULL,
    username VARCHAR(191) NOT NULL,
    customerId VARCHAR(20) NULL,
    fullname VARCHAR(150) NULL,
    phone VARCHAR(20) NULL,
    address TEXT NULL,
    identityNumber VARCHAR(50) NULL,
    profileName VARCHAR(100) NULL,
    territoryName VARCHAR(150) NULL,
    installerId VARCHAR(191) NOT NULL,
    installerName VARCHAR(150) NULL,
    installDate DATE NOT NULL,
    latitude DECIMAL(10,7) NULL,
    longitude DECIMAL(10,7) NULL,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user (userId),
    INDEX idx_date (installDate),
    INDEX idx_installer (installerId)
  );
  ```
- [ ] **Backend**: Buat model `InstallationLog`
- [ ] **Backend**: Modifikasi PSB endpoint (`pppoe_ext.go` CreateUser) — auto-insert installation log
- [ ] **Backend**: `GET /api/installation-logs` — list dengan filter periode, installer
- [ ] **Frontend**: Page "Laporan Pemasangan" di admin → laporan
- [ ] **Frontend**: Filter periode + installer + export PDF/Excel

### 1.7 External API (User Status by API Key)

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Endpoint publik (API key-based) yang return status semua user: online/aktif/isolir. Untuk integrasi sistem eksternal. |
| Dampak | **Bisnis**: memungkinkan integrasi dengan sistem lain (monitoring, dashboard eksternal). |
| Kompleksitas | **Low** |
| Prioritas | **Low** |

**Checklist:**

- [ ] **DB**: Tambah table `api_keys`:
  ```sql
  CREATE TABLE api_keys (
    id VARCHAR(191) PRIMARY KEY,
    keyHash VARCHAR(255) NOT NULL,
    label VARCHAR(100) NOT NULL,
    isActive BOOLEAN DEFAULT true,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    lastUsedAt TIMESTAMP NULL
  );
  ```
- [ ] **Backend**: `GET /api/external/users/status` — header `X-API-Key`, return array `{ username, status, profileName, routerName }`
- [ ] **Backend**: Middleware validasi API key, log last used
- [ ] **Backend**: `POST /api/api-keys` (admin) — generate new API key
- [ ] **Backend**: `GET /api/api-keys` (admin) — list API keys
- [ ] **Frontend**: Page "API Keys" di admin → settings
- [ ] **Frontend**: Generate key button + copy key + revoke

---

## Phase 1: Foundation & Stabilization

> Estimasi: 2-4 minggu  
> Goal: Stabilkan fondasi sebelum fitur kompleks  
> Prinsip: Tidak merusak yang sudah berjalan

### 1.8 Territory & Collector Management

| Item | Detail |
|---|---|
| Status | [~] Ada `PppoeArea`, tapi tidak ada hierarchy wilayah/collector |
| Deskripsi | Sistem territory berharki: provinsi → kabupaten → kecamatan → kelurahan → dusun. Setiap territory punya kolektor. Data pelanggan difilter per territory untuk role collector. |
| Dampak | **Operasional**: manajemen wilayah terstruktur, kolektor lihat data sendiri. **Bisnis**: laporan per wilayah. |
| Kompleksitas | **High** |
| Prioritas | **High** |

**Checklist:**

- [ ] **DB**: Buat table `territories`:
  ```sql
  CREATE TABLE territories (
    id VARCHAR(191) PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    description TEXT NULL,
    collectorId VARCHAR(191) NULL,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_collector (collectorId)
  );
  ```
- [ ] **DB**: Buat table `territory_areas`:
  ```sql
  CREATE TABLE territory_areas (
    id VARCHAR(191) PRIMARY KEY,
    territoryId VARCHAR(191) NOT NULL,
    kelurahanKode VARCHAR(20) NULL,
    kelurahanNama VARCHAR(150) NULL,
    kecamatanNama VARCHAR(150) NULL,
    kabupatenNama VARCHAR(150) NULL,
    provinsiNama VARCHAR(150) NULL,
    dusunNama VARCHAR(150) NULL,
    collectorId VARCHAR(191) NULL,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_territory (territoryId),
    INDEX idx_kelurahan (kelurahanKode),
    INDEX idx_collector (collectorId)
  );
  ```
- [ ] **DB**: Tambah kolom di `pppoe_users`: `territoryId VARCHAR(191) NULL`, `territoryAreaId VARCHAR(191) NULL`
- [ ] **DB**: Buat table `wilayah` (BPS codes): provinsi/kabupaten/kecamatan/kelurahan
- [ ] **Backend**: Model `Territory`, `TerritoryArea`, `Wilayah`
- [ ] **Backend**: CRUD territory endpoints (admin only)
- [ ] **Backend**: `GET /api/wilayah/provinsi`, `GET /api/wilayah/kabupaten/:kode`, `GET /api/wilayah/kecamatan/:kode`, `GET /api/wilayah/kelurahan/:kode`
- [ ] **Backend**: `POST /api/territories/:id/areas` — tambah kelurahan/dusun ke territory
- [ ] **Backend**: Modifikasi PSB endpoint — auto-assign territory berdasarkan kelurahan/dusun
- [ ] **Backend**: Modifikasi semua list endpoints — filter by territory jika role=collector
- [ ] **Backend**: Tambah role `collector` ke UsersRole enum
- [ ] **Frontend**: Page "Manajemen Wilayah" di admin
- [ ] **Frontend**: Form territory: nama, deskripsi, pilih kolektor
- [ ] **Frontend**: Form area: pilih provinsi → kabupaten → kecamatan → kelurahan (cascading dropdown)
- [ ] **Frontend**: Filter data pelanggan per territory di dashboard
- [ ] **Frontend**: PSB form — auto-fill territory dari pilihan kelurahan

### 1.9 Settlement Report per Kolektor

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Rekap setoran harian/range per kolektor: total invoice lunas, jumlah pelanggan, detail per invoice. Admin bisa konfirmasi settlement. |
| Dampak | **Bisnis**: transparansi keuangan, rekonsiliasi kolektor. |
| Kompleksitas | **Medium** |
| Prioritas | **Medium** |
| Depedensi | **1.8 Territory & Collector** (kolektor harus ada dulu) |

**Checklist:**

- [ ] **DB**: Buat table `settlements`:
  ```sql
  CREATE TABLE settlements (
    id VARCHAR(191) PRIMARY KEY,
    collectorId VARCHAR(191) NOT NULL,
    periodDate DATE NOT NULL,
    totalAmount INT NOT NULL,
    invoiceCount INT NOT NULL,
    status ENUM('pending','confirmed') DEFAULT 'pending',
    confirmedBy VARCHAR(191) NULL,
    confirmedAt TIMESTAMP NULL,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_collector_date (collectorId, periodDate)
  );
  ```
- [ ] **Backend**: `GET /api/settlements?collectorId=&date=` — rekap setoran
- [ ] **Backend**: `GET /api/settlements/range?from=&to=&collectorId=` — rekap range
- [ ] **Backend**: `POST /api/settlements/:id/confirm` — admin konfirmasi
- [ ] **Frontend**: Page "Setoran Kolektor" di admin → keuangan
- [ ] **Frontend**: Filter periode (harian/range) + pilih kolektor
- [ ] **Frontend**: Tabel detail: invoice number, pelanggan, amount, metode, tanggal
- [ ] **Frontend**: Tombol "Konfirmasi Settlement"
- [ ] **Frontend**: Export PDF/Excel

### 1.10 MikroTik Connection Pool Improvement

| Item | Detail |
|---|---|
| Status | [~] Ada MikroTik API, tapi tidak ada connection pool yang reusable |
| Deskripsi | Pool koneksi MikroTik per router, reuse koneksi, cleanup idle >5 menit. Kurangi overhead TCP handshake. |
| Dampak | **Performa**: response lebih cepat untuk operasi MikroTik, kurangi latency. |
| Kompleksitas | **Medium** |
| Prioritas | **Medium** |

**Checklist:**

- [ ] **Backend**: Buat `internal/mikrotik/pool.go` — connection pool struct:
  ```go
  type Pool struct {
      mu    sync.Mutex
      conns map[string]*poolEntry // key: routerID
  }
  type poolEntry struct {
      client     *ros.Client
      lastUsedAt time.Time
  }
  ```
- [ ] **Backend**: `GetClient(routerID string) (*ros.Client, error)` — reuse atau buat baru
- [ ] **Backend**: Cleanup goroutine — tutup koneksi idle >5 menit
- [ ] **Backend**: Ganti semua `dialMikrotik` calls di handlers dengan pool
- [ ] **Backend**: Graceful close semua koneksi saat server shutdown

---

## Phase 2: Billing Core

> Estimasi: 1-2 bulan  
> Goal: Billing system sekuat billing.pmynet.id  
> Depedensi: Phase 1 selesai (territory, collector)

### 2.1 PSB 24-Jam Deadline

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Pelanggan baru dapat 24 jam konfirmasi pembayaran awal sejak first connect. Jika tidak konfirmasi, isolir otomatis. |
| Dampak | **Bisnis**: kurangi rugi dari pelanggan pasang → pakai gratis → kabur. |
| Kompleksitas | **Medium** |
| Prioritas | **High** |

**Checklist:**

- [ ] **DB**: Tambah kolom di `pppoe_users`:
  - `initialPaymentPending` BOOL DEFAULT false
  - `initialPaymentDeadline` TIMESTAMP NULL
  - `firstConnectAt` TIMESTAMP NULL
- [ ] **Backend**: Modifikasi PSB endpoint — set `initialPaymentPending=true` saat create user baru
- [ ] **Backend**: Modifikasi session sync cron (`pppoe_session_sync.go`) — detect first UP:
  ```go
  // Saat user pertama kali muncul di active sessions:
  // Set firstConnectAt = now, deadline = now + 24h
  ```
- [ ] **Backend**: Modifikasi `auto_isolate.go` — cek deadline PSB:
  ```go
  // SELECT users WHERE initialPaymentPending=true 
  //   AND initialPaymentDeadline < NOW()
  //   AND status = 'active'
  // → isolir
  ```
- [ ] **Backend**: Saat invoice PSB dilunasi → set `initialPaymentPending=false`
- [ ] **Backend**: Notifikasi: admin + pelanggan saat diisolir
- [ ] **Frontend**: Badge "PSB Pending" di user list untuk user yang masih dalam 24h
- [ ] **Frontend**: Countdown timer di customer detail
- [ ] **Settings**: Tambah `psbDeadlineHours` (default: 24) di Company settings

### 2.2 Payment Promise (Janji Bayar)

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Pelanggan bisa ajukan janji bayar dengan tanggal. Sistem skip isolir selama janji aktif. Cron tengah malam cek: jika tanggal lewat dan belum bayar → re-isolir. |
| Dampak | **Bisnis**: kurangi churn, beri waktu grace kepada pelanggan. **Operasional**: kurangi komplain isolir. |
| Kompleksitas | **Medium** |
| Prioritas | **High** |
| Depedensi: | **2.1 PSB Deadline** (cron isolir harus sudah handle promise skip) |

**Checklist:**

- [ ] **DB**: Buat table `payment_promises`:
  ```sql
  CREATE TABLE payment_promises (
    id VARCHAR(191) PRIMARY KEY,
    userId VARCHAR(191) NOT NULL,
    username VARCHAR(191) NOT NULL,
    promiseDate DATE NOT NULL,
    status ENUM('active','fulfilled','broken') DEFAULT 'active',
    createdBy VARCHAR(191) NOT NULL,
    createdByName VARCHAR(191) NULL,
    notes TEXT NULL,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user (userId),
    INDEX idx_status (status),
    INDEX idx_date (promiseDate)
  );
  ```
- [ ] **Backend**: Model `PaymentPromise`
- [ ] **Backend**: `POST /api/users/:id/promise` — body: `{ promiseDate, notes }`, set status=active
- [ ] **Backend**: `GET /api/users/promises/active` — semua promise aktif (bulk, 1 request)
- [ ] **Backend**: `GET /api/portal/promise` — promise aktif untuk pelanggan yang login
- [ ] **Backend**: Modifikasi `auto_isolate.go` — skip isolir jika ada promise aktif:
  ```go
  hasPromise := s.db.Where("userId = ? AND status = 'active' AND promiseDate >= ?", userID, today).First(&promise).Error == nil
  if hasPromise { continue }
  ```
- [ ] **Backend**: Cron tengah malam `check_expired_promises`:
  ```go
  // Ambil semua promise active yang promiseDate < today
  // Cek apakah invoice sudah lunas
  // Jika lunas → status=fulfilled
  // Jika belum → status=broken, re-isolir user
  ```
- [ ] **Backend**: Saat invoice dilunasi → cek & set promise=fulfilled
- [ ] **Backend**: Tambah cron job di `scheduler.go`:
  ```go
  s.cron.AddFunc("0 30 0 * * *", s.jobCheckExpiredPromises) // 00:30 daily
  ```
- [ ] **Frontend**: Tombol "Janji Bayar" di invoice list (admin) dan portal pelanggan
- [ ] **Frontend**: Modal: pilih tanggal janji + notes
- [ ] **Frontend**: Badge "Janji Bayar" di user card
- [ ] **Frontend**: List promise aktif di dashboard admin

### 2.3 Profile Overrides per NAS (MikroTik)

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | 1 paket (profile) bisa punya MikroTik profile berbeda per router. Contoh: paket "10Mbps" → MikroTik profile "10M" di router A, "10M_RX" di router B. |
| Dampak | **Operasional**: fleksibilitas untuk router dengan konfigurasi berbeda. |
| Kompleksitas | **Medium** |
| Prioritas | **Medium** |

**Checklist:**

- [ ] **DB**: Buat table `profile_router_map`:
  ```sql
  CREATE TABLE profile_router_map (
    id VARCHAR(191) PRIMARY KEY,
    profileId VARCHAR(191) NOT NULL,
    routerId VARCHAR(191) NOT NULL,
    mikrotikProfile VARCHAR(100) NOT NULL,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_profile_router (profileId, routerId)
  );
  ```
- [ ] **Backend**: Model `ProfileRouterMap`
- [ ] **Backend**: `GET /api/profiles/:id/overrides` — list overrides per profile
- [ ] **Backend**: `PUT /api/profiles/:id/overrides` — set override per router
- [ ] **Backend**: Modifikasi `syncPPPProfileToRouters` — gunakan override jika ada
- [ ] **Backend**: Modifikasi PSB — saat create PPP Secret, gunakan override profile
- [ ] **Frontend**: Di profile edit page → tab "Override per Router"
- [ ] **Frontend**: Tabel: router → MikroTik profile name (editable)

### 2.4 Waiting List (Antrian Pemasangan)

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Form pendaftaran antrian pemasangan: nama, telepon, alamat, NIK, foto KTP, pilih paket, koordinat GPS. Assign ke teknisi. Status: waiting → installed → cancelled. |
| Dampak | **Operasional**: manajemen antrian PSB, tracking progress instalasi. |
| Kompleksitas | **Medium** |
| Prioritas | **Medium** |
| Depedensi | **1.8 Territory** (untuk auto-assign wilayah) |

**Checklist:**

- [ ] **DB**: Buat table `waiting_list`:
  ```sql
  CREATE TABLE waiting_list (
    id VARCHAR(191) PRIMARY KEY,
    fullname VARCHAR(150) NOT NULL,
    phone VARCHAR(20) NULL,
    address TEXT NULL,
    identityNumber VARCHAR(50) NULL,
    ktpPhoto TEXT NULL,
    notes TEXT NULL,
    territoryId VARCHAR(191) NULL,
    territoryAreaId VARCHAR(191) NULL,
    kelurahanKode VARCHAR(20) NULL,
    profileId VARCHAR(191) NULL,
    sales VARCHAR(100) NULL,
    latitude DECIMAL(10,7) NULL,
    longitude DECIMAL(10,7) NULL,
    status ENUM('waiting','installed','cancelled') DEFAULT 'waiting',
    createdBy VARCHAR(191) NOT NULL,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_territory (territoryId)
  );
  ```
- [ ] **DB**: Buat table `waiting_list_assignments`:
  ```sql
  CREATE TABLE waiting_list_assignments (
    id VARCHAR(191) PRIMARY KEY,
    waitingListId VARCHAR(191) NOT NULL,
    technicianUsername VARCHAR(191) NOT NULL,
    assignedBy VARCHAR(191) NOT NULL,
    assignedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_wl (waitingListId)
  );
  ```
- [ ] **Backend**: Model `WaitingList`, `WaitingListAssignment`
- [ ] **Backend**: `GET /api/waiting-list` — list dengan filter status, territory
- [ ] **Backend**: `POST /api/waiting-list` — create entry (admin/noc/collector)
- [ ] **Backend**: `PUT /api/waiting-list/:id` — update entry
- [ ] **Backend**: `DELETE /api/waiting-list/:id` — cancel (set status=cancelled)
- [ ] **Backend**: `POST /api/waiting-list/:id/assign` — assign teknisi
- [ ] **Backend**: `POST /api/waiting-list/:id/convert` — convert ke PSB (create pppoe_user)
- [ ] **Backend**: Notifikasi teknisi saat di-assign
- [ ] **Frontend**: Page "Waiting List" di admin
- [ ] **Frontend**: Form pendaftaran dengan foto KTP upload, map picker
- [ ] **Frontend**: List dengan filter status, territory
- [ ] **Frontend**: Assign teknisi modal (multi-select)
- [ ] **Frontend**: Tombol "Convert ke PSB" → pre-fill PSB form
- [ ] **Frontend**: Teknisi page: list WL yang di-assign ke mereka

### 2.5 ONT Removal Tasks

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Task pencabutan ONT untuk pelanggan berhenti. Teknisi di-assign, upload foto bukti, admin konfirmasi. Auto-cancel saat pelanggan dilunasi. |
| Dampak | **Operasional**: tracking pencabutan perangkat, akuntabilitas inventaris. |
| Kompleksitas | **Medium** |
| Prioritas | **Low** |
| Depedensi | **2.4 Waiting List** (sistem task assignment serupa) |

**Checklist:**

- [ ] **DB**: Buat table `ont_removal_tasks`:
  ```sql
  CREATE TABLE ont_removal_tasks (
    id VARCHAR(191) PRIMARY KEY,
    userId VARCHAR(191) NOT NULL,
    username VARCHAR(191) NOT NULL,
    customerId VARCHAR(20) NULL,
    fullname VARCHAR(150) NULL,
    address TEXT NULL,
    territoryName VARCHAR(150) NULL,
    latitude DECIMAL(10,7) NULL,
    longitude DECIMAL(10,7) NULL,
    assignedTo VARCHAR(191) NOT NULL,
    assignedBy VARCHAR(191) NOT NULL,
    status ENUM('pending','done','confirmed','cancelled') DEFAULT 'pending',
    proofPhoto TEXT NULL,
    notes TEXT NULL,
    cancelReason TEXT NULL,
    cancelledBy VARCHAR(191) NULL,
    cancelledAt TIMESTAMP NULL,
    confirmedBy VARCHAR(191) NULL,
    confirmedAt TIMESTAMP NULL,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_assigned (assignedTo)
  );
  ```
- [ ] **Backend**: Model `OntRemovalTask`
- [ ] **Backend**: `GET /api/ont-removal-tasks` — list dengan filter status, teknisi
- [ ] **Backend**: `POST /api/ont-removal-tasks` — create task (admin)
- [ ] **Backend**: `POST /api/ont-removal-tasks/:id/complete` — teknisi upload proof photo
- [ ] **Backend**: `POST /api/ont-removal-tasks/:id/confirm` — admin konfirmasi
- [ ] **Backend**: `POST /api/ont-removal-tasks/:id/cancel` — cancel dengan alasan
- [ ] **Backend**: Auto-create task saat user set status=berhenti
- [ ] **Backend**: Auto-cancel task saat invoice dilunasi
- [ ] **Backend**: Notifikasi teknisi saat di-assign
- [ ] **Frontend**: Page "Task Cabut ONT" di admin
- [ ] **Frontend**: Teknisi page: list task yang di-assign
- [ ] **Frontend**: Upload foto bukti pencabutan
- [ ] **Frontend**: Admin: konfirmasi/cancel task

### 2.6 MikroTik Hybrid Auth Mode

| Item | Detail |
|---|---|
| Status | [ ] Belum ada (salfanet pakai RADIUS only) |
| Deskripsi | Setiap router bisa set auth mode: RADIUS atau Local (PPP Secret). Saat RADIUS: PPP Secret dibuat DISABLED (backup). Saat Local: PPP Secret ENABLED, RADIUS tidak handle auth. Isolir: Auth-Type Reject + disable PPP Secret (safety net). |
| Dampak | **Operasional**: fallback jika RADIUS down, migrasi gradual ke local auth. |
| Kompleksitas | **High** |
| Prioritas | **Low** (nice to have, RADIUS sudah stabil) |
| Depedensi | **1.10 MikroTik Connection Pool** |

**Checklist:**

- [ ] **DB**: Tambah kolom di `nas` table: `authMode ENUM('radius','local') NULL DEFAULT 'radius'`
- [ ] **Backend**: Modifikasi PSB — create PPP Secret:
  - Jika authMode=radius → disabled=true (backup)
  - Jika authMode=local → disabled=false (aktif)
- [ ] **Backend**: Modifikasi isolir — disable PPP Secret selalu (safety net)
- [ ] **Backend**: Modifikasi activate — enable PPP Secret jika authMode=local
- [ ] **Backend**: `POST /api/mikrotik/:routerId/enable-all-secrets` — bulk enable
- [ ] **Backend**: `POST /api/mikrotik/:routerId/sync-all-secrets` — sync semua user
- [ ] **Backend**: `POST /api/users/migrate-local-auth` — bulk migrate radius→local
- [ ] **Backend**: `GET /api/mikrotik/sync-status` — rekap user belum sinkron per NAS
- [ ] **Frontend**: Di router edit → pilih "Mode Autentikasi": RADIUS / Local
- [ ] **Frontend**: Button "Sync All Secrets" per router
- [ ] **Frontend**: Button "Enable All Secrets" per router (untuk migrasi)
- [ ] **Frontend**: Sync status dashboard: rekap user belum sinkron

---

## Phase 3: Integration (OLT/ONU + Customer + Billing)

> Estimasi: 2-4 bulan  
> Goal: Sinkronisasi data antara device, customer, dan billing  
> Depedensi: Phase 2 selesai

### 3.1 ONU ↔ Customer Linking

| Item | Detail |
|---|---|
| Status | [~] Ada `OdpCustomerAssignment`, tapi belum link ke ONU/GeriatricACS device |
| Deskripsi | Setiap pelanggan di-link ke ONU spesifik via GenieACS device ID. Saat lihat customer detail → tampilkan status ONU (online, RX power, firmware, dll). |
| Dampak | **Operasional**: troubleshooting cepat, tahu ONU mana milik pelanggan mana. |
| Kompleksitas | **Medium** |
| Prioritas | **High** |

**Checklist:**

- [ ] **DB**: Tambah kolom di `pppoe_users`: `genieacsDeviceId VARCHAR(191) NULL`, `onuSerialNumber VARCHAR(100) NULL`
- [ ] **Backend**: Modifikasi PSB — field baru: pilih ONU dari list GenieACS devices (atau input serial number)
- [ ] **Backend**: `GET /api/users/:id/onu-status` — fetch dari GenieACS:
  - Online/offline status
  - RX power
  - TX power
  - Temperature
  - Firmware version
  - Uptime
- [ ] **Backend**: Saat ONU offline >30 menit → create ticket otomatis
- [ ] **Frontend**: Di customer detail → tab "Perangkat ONU"
- [ ] **Frontend**: Tampilkan status ONU real-time (poll tiap 30 detik)
- [ ] **Frontend**: PSB form → dropdown pilih ONU (atau scan QR serial number)

### 3.2 Auto-Provisioning ONU via GenieACS

| Item | Detail |
|---|---|
| Status | [~] Ada GenieACS integration, tapi tidak ada auto-provisioning saat PSB |
| Deskripsi | Saat PSB baru, otomatis configure ONU via GenieACS TR-069: set VLAN, PPPoE credentials, WiFi SSID/password. |
| Dampak | **Operasional**: eliminasi manual config ONU, kurangi waktu instalasi. |
| Kompleksitas | **High** |
| Prioritas | **Medium** |
| Depedensi | **3.1 ONU ↔ Customer Linking** |

**Checklist:**

- [ ] **Backend**: Buat `internal/provisioning/auto.go`:
  ```go
  func AutoProvisionONU(deviceId string, config ProvisionConfig) error {
      // 1. Set PPPoE username/password via TR-069
      // 2. Set VLAN ID
      // 3. Set WiFi SSID + password
      // 4. Reboot ONU
  }
  ```
- [ ] **Backend**: Modifikasi PSB endpoint — setelah create user, trigger auto-provisioning
- [ ] **Backend**: Job queue untuk provisioning (async, tidak block PSB response)
- [ ] **Backend**: Retry logic: 3x dengan backoff jika GenieACS tidak reachable
- [ ] **Backend**: Status tracking: provisioning_pending → provisioning_success → provisioning_failed
- [ ] **Frontend**: Di PSB form → opsi "Auto-provision ONU" (checkbox, default on)
- [ ] **Frontend**: Provisioning status badge di user detail
- [ ] **Frontend**: Button "Re-provision" jika gagal

### 3.3 RX Power Monitoring → Alert + Ticket

| Item | Detail |
|---|---|
| Status | [~] Ada OLT PON stats, tapi tidak ada alerting otomatis |
| Deskripsi | Cron polling RX power ONU via GenieACS. Jika RX drop below threshold (contoh: -28 dBm) → create alert + ticket + notifikasi. |
| Dampak | **Operasional**: deteksi proaktif degradasi fiber sebelum pelanggan komplain. |
| Kompleksitas | **Medium** |
| Prioritas | **High** |
| Depedensi | **3.1 ONU ↔ Customer Linking** |

**Checklist:**

- [ ] **DB**: Tambah table `rx_power_alerts`:
  ```sql
  CREATE TABLE rx_power_alerts (
    id VARCHAR(191) PRIMARY KEY,
    userId VARCHAR(191) NULL,
    deviceId VARCHAR(191) NOT NULL,
    rxPower DECIMAL(5,2) NOT NULL,
    threshold DECIMAL(5,2) NOT NULL,
    status ENUM('active','resolved') DEFAULT 'active',
    ticketId VARCHAR(191) NULL,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolvedAt TIMESTAMP NULL,
    INDEX idx_status (status),
    INDEX idx_device (deviceId)
  );
  ```
- [ ] **DB**: Tambah setting `rxPowerThreshold` (default: -28.00) di Company settings
- [ ] **Backend**: Cron job `rx_power_monitor` — tiap 15 menit:
  ```go
  // 1. Fetch all ONU RX power via GenieACS bulk API
  // 2. Bandingkan dengan threshold
  // 3. Jika below threshold → create alert + ticket + notifikasi
  // 4. Jika alert sudah ada dan RX recovered → resolve alert
  ```
- [ ] **Backend**: Tambah cron di `scheduler.go`:
  ```go
  s.cron.AddFunc("0 */15 * * * *", s.jobRxPowerMonitor)
  ```
- [ ] **Backend**: Notifikasi: admin + NOC + pelanggan (jika terdampak)
- [ ] **Frontend**: Alert feed di dashboard — "RX Power Alert" dengan severity
- [ ] **Frontend**: Di ONU detail → grafik RX power history (recharts)
- [ ] **Frontend**: Settings → threshold configuration

### 3.4 Isolir via ONU (GenieACS)

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Alternatif isolir: disable ONU port via GenieACS (TR-069) sebagai backup jika RADIUS/PoD gagal. |
| Dampak | **Operasional**: isolir lebih reliable, multiple layer. |
| Kompleksitas | **Medium** |
| Prioritas | **Low** |
| Depedensi | **3.1 ONU ↔ Customer Linking** |

**Checklist:**

- [ ] **Backend**: `func IsolateONU(deviceId string) error` — disable WAN PPPoE via TR-069
- [ ] **Backend**: `func ActivateONU(deviceId string) error` — enable WAN PPPoE
- [ ] **Backend**: Modifikasi isolir endpoint — jalankan paralel: RADIUS + PoD + ONU disable
- [ ] **Backend**: Modifikasi activate endpoint — jalankan: hapus Reject + PoD + ONU enable
- [ ] **Frontend**: Toggle "Isolir via ONU" di settings (default: off, opt-in)

### 3.5 Real-Time Operational Dashboard

| Item | Detail |
|---|---|
| Status | [~] Ada dashboard, tapi tidak real-time (manual refresh) |
| Deskripsi | Dashboard dengan WebSocket/SSE untuk update real-time: user online/offline, invoice lunas, isolir, alert. |
| Dampak | **Operasional**: respons cepat ke event, monitoring proaktif. |
| Kompleksitas | **High** |
| Prioritas | **Medium** |

**Checklist:**

- [ ] **Backend**: Buat `internal/realtime/hub.go` — WebSocket hub:
  ```go
  type Hub struct {
      clients    map[*Client]bool
      broadcast  chan Event
      register   chan *Client
      unregister chan *Client
  }
  type Event struct {
      Type string      `json:"type"` // "user_online", "user_offline", "invoice_paid", etc.
      Data interface{} `json:"data"`
  }
  ```
- [ ] **Backend**: `GET /api/ws` — WebSocket upgrade, register client
- [ ] **Backend**: Emit events dari: session sync, payment webhook, isolir cron, alert system
- [ ] **Backend**: Filter events by role (collector hanya lihat territory sendiri)
- [ ] **Frontend**: WebSocket client hook `useRealtimeEvents`
- [ ] **Frontend**: Dashboard: live counter (total online, revenue today, alerts)
- [ ] **Frontend**: Activity feed: event stream real-time
- [ ] **Frontend**: Toast notification saat event penting (invoice lunas, user isolir)

---

## Phase 4: Automation & Smart Monitoring

> Estimasi: 2-3 bulan  
> Goal: Otomasi penuh, notifikasi pintar  
> Depedensi: Phase 3 selesai

### 4.1 Event-Driven Notification System

| Item | Detail |
|---|---|
| Status | [~] Ada notifikasi WA/Email/Telegram/Push, tapi tidak event-driven |
| Deskripsi | Internal event bus: `user.isolated`, `user.activated`, `invoice.paid`, `onu.offline`, `rx.drop`. Setiap event trigger notifikasi multi-channel dengan template engine. |
| Dampak | **Operasional**: notifikasi konsisten, tidak ada yang terlewat. **Bisnis**: pelanggan lebih informed. |
| Kompleksitas | **High** |
| Prioritas | **High** |

**Checklist:**

- [ ] **Backend**: Buat `internal/eventbus/bus.go`:
  ```go
  type EventBus struct {
      subscribers map[string][]EventHandler
  }
  type EventHandler func(event Event) error
  type Event struct {
      Type    string
      Payload map[string]interface{}
  }
  ```
- [ ] **Backend**: Register handlers:
  - `user.isolated` → WA + Email + Push + Portal notif
  - `user.activated` → WA + Push + Portal notif
  - `invoice.paid` → WA + Push + Portal notif
  - `invoice.overdue` → WA + Email + Push
  - `onu.offline` → Telegram + Push (admin only)
  - `rx.drop` → Telegram + Push (admin only)
  - `psb.deadline` → WA + Push (admin + customer)
- [ ] **Backend**: Template engine dengan variabel substitution:
  - `{customer_name}`, `{amount}`, `{due_date}`, `{rx_power}`, `{package_name}`
- [ ] **DB**: Buat table `notification_templates`:
  ```sql
  CREATE TABLE notification_templates (
    id VARCHAR(191) PRIMARY KEY,
    eventType VARCHAR(50) NOT NULL,
    channel ENUM('wa','email','push','telegram','portal') NOT NULL,
    template TEXT NOT NULL,
    isEnabled BOOLEAN DEFAULT true,
    UNIQUE KEY uniq_event_channel (eventType, channel)
  );
  ```
- [ ] **Backend**: Smart timing — quiet hours (configurable, default 22:00-07:00 WIB)
- [ ] **Backend**: Deduplication — tidak kirim notif sama 2x dalam 1 jam
- [ ] **Frontend**: Settings → "Template Notifikasi" — edit template per event per channel
- [ ] **Frontend**: Preview template dengan sample data
- [ ] **Frontend**: Toggle enable/disable per channel per event

### 4.2 Auto-Activation (Invoice Lunas → Internet Aktif)

| Item | Detail |
|---|---|
| Status | [~] Ada auto-isolir, tapi activate masih manual |
| Deskripsi | Saat invoice lunas via payment gateway webhook → otomatis: hapus Auth-Type Reject + PoD + enable PPP Secret + notifikasi pelanggan. |
| Dampak | **Operasional**: pelanggan langsung aktif tanpa intervensi admin. **Bisnis**: better UX. |
| Kompleksitas | **Medium** |
| Prioritas | **High** |
| Depedensi | **4.1 Event-Driven Notification** |

**Checklist:**

- [ ] **Backend**: Modifikasi payment webhook handler (`payment_handler.go`):
  ```go
  // Setelah invoice.Status = PAID:
  // 1. Hapus Auth-Type Reject dari radcheck
  // 2. Enable PPP Secret (jika local mode)
  // 3. Send PoD untuk force reconnect
  // 4. Emit event "user.activated"
  // 5. Notifikasi pelanggan
  ```
- [ ] **Backend**: Modifikasi manual payment confirmation — same flow
- [ ] **Backend**: Modifikasi bukti transfer approval — same flow
- [ ] **Backend**: Logging: catat siapa/apa yang trigger aktivasi (auto-gateway / admin / proof)
- [ ] **Frontend**: Settings → toggle "Auto-Activation on Payment" (default: on)
- [ ] **Frontend**: Badge "Auto-Activated" di invoice detail jika aktivasi otomatis

### 4.3 Auto-Provisioning Pipeline (PSB → ONU + PPP Secret + Invoice)

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Pipeline otomatis saat PSB: create user → create PPP Secret → configure ONU via GenieACS → generate invoice → send welcome notification. |
| Dampak | **Operasional**: eliminasi manual steps, waktu instalasi turun dari 30 menit → 5 menit. |
| Kompleksitas | **High** |
| Prioritas | **Medium** |
| Depedensi | **3.2 Auto-Provisioning ONU**, **4.1 Event-Driven Notification** |

**Checklist:**

- [ ] **Backend**: Buat `internal/provisioning/pipeline.go`:
  ```go
  func RunProvisioningPipeline(userID string) error {
      // Step 1: Create PPP Secret di MikroTik
      // Step 2: Configure ONU via GenieACS
      // Step 3: Generate invoice (INSTALLATION type)
      // Step 4: Send welcome notification (WA + Email)
      // Step 5: Set initialPaymentPending = true
      // Step 6: Emit event "user.provisioned"
  }
  ```
- [ ] **Backend**: Job queue untuk pipeline (async, retry 3x)
- [ ] **Backend**: Status tracking per step: `provisioning_status` table
- [ ] **DB**: Buat table `provisioning_status`:
  ```sql
  CREATE TABLE provisioning_status (
    id VARCHAR(191) PRIMARY KEY,
    userId VARCHAR(191) NOT NULL,
    step VARCHAR(50) NOT NULL,
    status ENUM('pending','running','success','failed') DEFAULT 'pending',
    error TEXT NULL,
    startedAt TIMESTAMP NULL,
    completedAt TIMESTAMP NULL,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user (userId),
    INDEX idx_status (status)
  );
  ```
- [ ] **Frontend**: PSB form → checkbox "Auto-Provisioning" (default: on)
- [ ] **Frontend**: Provisioning progress tracker di user detail
- [ ] **Frontend**: Retry button untuk step yang gagal

### 4.4 Smart Alert Rules Engine

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Rule-based alerting: `IF condition THEN action`. Contoh: `IF rx_power < -28 AND user.status = 'active' THEN create_ticket + notify_admin + notify_customer`. |
| Dampak | **Operasional**: deteksi proaktif, response otomatis ke masalah jaringan. |
| Kompleksitas | **High** |
| Prioritas | **Low** |
| Depedensi | **4.1 Event-Driven Notification**, **3.3 RX Power Monitoring** |

**Checklist:**

- [ ] **DB**: Buat table `alert_rules`:
  ```sql
  CREATE TABLE alert_rules (
    id VARCHAR(191) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    triggerEvent VARCHAR(50) NOT NULL,
    conditions JSON NOT NULL,
    actions JSON NOT NULL,
    isEnabled BOOLEAN DEFAULT true,
    priority INT DEFAULT 0,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
  );
  ```
- [ ] **Backend**: Rule engine `internal/alerts/engine.go`:
  ```go
  func EvaluateRule(rule AlertRule, event Event) bool {
      // Parse conditions JSON
      // Evaluate against event payload
      // Return true if all conditions match
  }
  func ExecuteActions(rule AlertRule, event Event) {
      // Parse actions JSON
      // Execute: create_ticket, notify_admin, notify_customer, isolate_user, etc.
  }
  ```
- [ ] **Backend**: Pre-built rules:
  - RX drop → ticket + notify admin
  - ONU offline >30min → ticket + notify admin
  - Invoice overdue 7 days → WA + email
  - PSB deadline 2h → notify admin
- [ ] **Backend**: `GET /api/alert-rules` — list rules
- [ ] **Backend**: `POST /api/alert-rules` — create rule
- [ ] **Backend**: `PUT /api/alert-rules/:id` — update rule
- [ ] **Frontend**: Page "Aturan Alert" di admin → settings
- [ ] **Frontend**: Visual rule builder: trigger → conditions → actions
- [ ] **Frontend**: Enable/disable toggle per rule

---

## Phase 5: Scaling & Optimization

> Estimasi: 3-6 bulan  
> Goal: Scale untuk 5000+ pelanggan, multi-tenant ready  
> Depedensi: Phase 4 selesai

### 5.1 Redis Caching

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Redis untuk: session cache, dashboard stats cache, rate limiting, job queue backend. |
| Dampak | **Performa**: response API 3-5x lebih cepat untuk query berat. |
| Kompleksitas | **Medium** |
| Prioritas | **Medium** |

**Checklist:**

- [ ] **Infra**: Add Redis to `docker-compose.yml` / VPS install
- [ ] **Backend**: `internal/cache/redis.go` — Redis client wrapper
- [ ] **Backend**: Cache dashboard stats (TTL: 30s)
- [ ] **Backend**: Cache user list (TTL: 60s, invalidate on create/update/delete)
- [ ] **Backend**: Cache MikroTik active sessions (TTL: 60s)
- [ ] **Backend**: Cache ONU status (TTL: 30s)
- [ ] **Backend**: Fallback: jika Redis down, query DB (graceful degradation)
- [ ] **Settings**: `redisUrl` di environment config

### 5.2 Job Queue (Asynq)

| Item | Detail |
|---|---|
| Status | [ ] Belum ada (cron only) |
| Deskripsi | Asynq (Redis-backed) untuk: MikroTik API calls, notification sending, billing generation, ONU provisioning. Async, retry, dead letter queue. |
| Dampak | **Performa**: API response tidak blocking. **Reliability**: retry otomatis, tidak lose task. |
| Kompleksitas | **High** |
| Prioritas | **Medium** |
| Depedensi | **5.1 Redis** |

**Checklist:**

- [ ] **Backend**: `go get github.com/hibiken/asynq`
- [ ] **Backend**: `internal/queue/client.go` — Asynq client
- [ ] **Backend**: `internal/queue/worker.go` — worker process
- [ ] **Backend**: Task definitions:
  - `task:mikrotik:sync_secret` — sync PPP Secret
  - `task:mikrotik:kick_user` — kick active session
  - `task:notify:send` — send notification multi-channel
  - `task:billing:generate_invoice` — generate invoice for user
  - `task:provisioning:onu` — auto-provision ONU
  - `task:alert:rx_check` — check RX power for device
- [ ] **Backend**: Modifikasi semua handler — enqueue task instead of synchronous call
- [ ] **Backend**: Retry policy: 3x with exponential backoff
- [ ] **Backend**: Dead letter queue: failed tasks after max retry
- [ ] **Backend**: Dashboard: `GET /api/queue/status` — queue depth, failed tasks
- [ ] **Frontend**: Queue monitor page di admin → settings
- [ ] **Frontend**: Retry failed tasks button

### 5.3 Captive Portal (DST-NAT)

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | User isolir diarahkan ke halaman tagihan via MikroTik DST-NAT. User bisa lihat tagihan + bayar langsung dari halaman captive. |
| Dampak | **Bisnis**: tingkatkan collection rate, pelanggan bisa self-service bayar. |
| Kompleksitas | **High** |
| Prioritas | **Low** |
| Depedensi | **4.2 Auto-Activation** |

**Checklist:**

- [ ] **Frontend**: Next.js route `/captive` (tanpa login):
  - Deteksi IP request → cari username via radacct/ppp_active_cache
  - Tampilkan: nama pelanggan, tagihan unpaid, tombol bayar
  - Redirect ke payment gateway
- [ ] **Backend**: `GET /api/captive/identify?ip=` — return user info + unpaid invoices berdasarkan IP
- [ ] **Backend**: `POST /api/captive/pay` — create payment link untuk invoice
- [ ] **MikroTik**: DST-NAT config:
  - IP Pool isolir: `173.16.20.2-173.16.20.254`
  - Profile isolir: `profile-isolir` (rate-limit 1M/1M)
  - Address-list `isolir` → redirect HTTP/HTTPS ke VPS
  - Bypass masquerade ke VPS agar IP asli terdeteksi
  - Bypass payment gateway domains (Midtrans, Xendit, Duitku, Tripay)
- [ ] **Backend**: Modifikasi isolir — set profile ke `profile-isolir` (alternatif Auth-Type Reject)
- [ ] **Backend**: Modifikasi activate — set profile kembali ke normal
- [ ] **Frontend**: Captive portal page: responsive, sederhana, bahasa Indonesia
- [ ] **Frontend**: Status: "Pembayaran Diterima, internet akan aktif dalam 1-2 menit"

### 5.4 Multi-Tenant Support

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Multiple ISP/company dalam 1 instance. Setiap tenant punya: pelanggan sendiri, router sendiri, branding sendiri, billing sendiri. |
| Dampak | **Bisnis**: SaaS model, jual ke multiple ISP. |
| Kompleksitas | **High** |
| Prioritas | **Low** (strategic, long-term) |

**Checklist:**

- [ ] **DB**: Tambah `tenantId` ke semua table utama (pppoe_users, invoices, nas, tickets, dll)
- [ ] **DB**: Buat table `tenants`:
  ```sql
  CREATE TABLE tenants (
    id VARCHAR(191) PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    domain VARCHAR(150) NULL,
    logo TEXT NULL,
    settings JSON NULL,
    subscriptionPlanId VARCHAR(191) NULL,
    isActive BOOLEAN DEFAULT true,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
  );
  ```
- [ ] **DB**: Buat table `subscription_plans`:
  ```sql
  CREATE TABLE subscription_plans (
    id VARCHAR(191) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    maxUsers INT NOT NULL,
    maxRouters INT NOT NULL,
    price INT NOT NULL,
    features JSON NULL,
    isActive BOOLEAN DEFAULT true
  );
  ```
- [ ] **Backend**: Middleware `TenantMiddleware` — inject tenantId dari JWT/domain
- [ ] **Backend**: Modifikasi semua query — filter by tenantId
- [ ] **Backend**: Tenant-scoped RADIUS config (per-tenant FreeRADIUS virtual server)
- [ ] **Frontend**: Login page → detect tenant from domain
- [ ] **Frontend**: Superadmin page: manage tenants, subscription plans
- [ ] **Frontend**: Per-tenant branding: logo, color, company name

### 5.5 API Rate Limiting

| Item | Detail |
|---|---|
| Status | [ ] Belum ada |
| Deskripsi | Rate limit API per IP/user untuk mencegah abuse. |
| Dampak | **Keamanan**: proteksi brute force, API abuse. |
| Kompleksitas | **Low** |
| Prioritas | **Medium** |

**Checklist:**

- [ ] **Backend**: Fiber rate limiter middleware:
  ```go
  app.Use(limiter.New(limiter.Config{
      Max:        100,
      Expiration: 1 * time.Minute,
      KeyGenerator: func(c fiber.Ctx) string {
          return c.IP() // or utils.ExtractIP(c)
      },
  }))
  ```
- [ ] **Backend**: Stricter limit untuk auth endpoints (5 req/min)
- [ ] **Backend**: Stricter limit for portal login (10 req/min)
- [ ] **Backend**: Redis-backed rate limiter (if Redis available)

---

## Dependency Graph

```
Quick Wins (1-2 minggu)
├── 1.1 Invoice Discount          [ ] → no dependency
├── 1.2 Cancel Invoice            [ ] → no dependency
├── 1.3 Edit Payment Method       [ ] → no dependency
├── 1.4 Skip Minggu Isolir        [ ] → no dependency
├── 1.5 Package Change Log        [ ] → no dependency
├── 1.6 Installation Log          [ ] → no dependency
└── 1.7 External API              [ ] → no dependency

Phase 1: Foundation (2-4 minggu)
├── 1.8 Territory & Collector     [ ] → no dependency (foundation)
├── 1.9 Settlement Report         [ ] → depends on 1.8
└── 1.10 MikroTik Pool            [~] → no dependency

Phase 2: Billing Core (1-2 bulan)
├── 2.1 PSB 24h Deadline          [ ] → no dependency
├── 2.2 Payment Promise           [ ] → depends on 2.1 (cron skip promise)
├── 2.3 Profile Overrides         [ ] → no dependency
├── 2.4 Waiting List              [ ] → depends on 1.8 (territory)
├── 2.5 ONT Removal Tasks         [ ] → depends on 2.4 (task pattern)
└── 2.6 MikroTik Hybrid Auth      [ ] → depends on 1.10 (connection pool)

Phase 3: Integration (2-4 bulan)
├── 3.1 ONU ↔ Customer            [~] → no dependency
├── 3.2 Auto-Provisioning ONU     [ ] → depends on 3.1
├── 3.3 RX Power Monitoring       [~] → depends on 3.1
├── 3.4 Isolir via ONU            [ ] → depends on 3.1
└── 3.5 Real-Time Dashboard       [~] → no dependency (can start parallel)

Phase 4: Automation (2-3 bulan)
├── 4.1 Event-Driven Notif        [~] → no dependency (can start parallel)
├── 4.2 Auto-Activation           [~] → depends on 4.1
├── 4.3 Auto-Provisioning Pipeline[ ] → depends on 3.2, 4.1
└── 4.4 Smart Alert Rules         [ ] → depends on 4.1, 3.3

Phase 5: Scaling (3-6 bulan)
├── 5.1 Redis Caching             [ ] → no dependency
├── 5.2 Job Queue (Asynq)         [ ] → depends on 5.1
├── 5.3 Captive Portal            [ ] → depends on 4.2
├── 5.4 Multi-Tenant              [ ] → depends on all above (strategic)
└── 5.5 API Rate Limiting         [ ] → no dependency (can start anytime)
```

---

## Rekomendasi Teknis

### 1. Redis / Queue untuk Automation

**Kapan implementasi:**
- **Redis**: Phase 5 (atau lebih awal jika dashboard sudah lambat)
- **Asynq**: Phase 5 (atau saat MikroTik API calls mulai blocking)

**Kenapa bukan sekarang:**
- Salfanet saat ini pakai cron + synchronous, masih cukup untuk <1000 pelanggan
- Redis menambah complexity infra (1 service lagi untuk maintain)
- Implementasi incremental: Redis dulu, baru Asynq

**Best practice:**
```
Redis → Cache layer (dashboard stats, session cache, ONU status)
  ↓
Asynq → Job queue (MikroTik API, notification, provisioning)
  ↓
Worker → Process jobs async (separate goroutine or process)
```

### 2. Struktur Database untuk Billing + Pelanggan

**Prinsip:**
- **Single source of truth**: `pppoe_users` sebagai pivot table
- **Soft delete**: tidak pernah DELETE, gunakan status flag
- **Audit trail**: semua perubahan state dicatat
- **Index optimization**: index di kolom yang sering di-query

**Schema yang sudah ada (tidak perlu ubah):**
- `pppoe_users` → pelanggan utama
- `invoices` → tagihan
- `payments` → pembayaran
- `nas` → router MikroTik
- `radcheck/radreply/radusergroup` → RADIUS auth
- `radacct` → RADIUS accounting

**Schema yang perlu ditambah (urut prioritas):**
1. `package_change_logs` — audit perubahan paket
2. `installation_logs` — audit pemasangan
3. `payment_promises` — janji bayar
4. `territories` + `territory_areas` — wilayah
5. `waiting_list` + `waiting_list_assignments` — antrian PSB
6. `ont_removal_tasks` — task cabut ONT
7. `rx_power_alerts` — alert RX power
8. `provisioning_status` — tracking auto-provisioning
9. `notification_templates` — template notif
10. `alert_rules` — rule engine
11. `api_keys` — external API
12. `settlements` — setoran kolektor

### 3. Best Practice Integrasi NMS + Radius + Billing

**Arsitektur:**
```
┌─────────────────────────────────────────────────────┐
│                    Go API (Fiber)                     │
├──────────┬──────────┬──────────┬─────────────────────┤
│  Billing │  RADIUS  │   NMS    │   Provisioning      │
│  Module  │  Module  │  Module  │   Module            │
├──────────┼──────────┼──────────┼─────────────────────┤
│ Postgres │ FreeRAD- │  GenieACS │   MikroTik API     │
│ (GORM)   │   ius    │  (TR-069)│   (routeros-api)   │
└──────────┴──────────┴──────────┴─────────────────────┘
```

**Prinsip integrasi:**

1. **Event-driven, bukan polling**
   - Saat invoice lunas → emit `invoice.paid` → trigger activate user
   - Saat user diisolir → emit `user.isolated` → trigger ONU disable + notif
   - Saat ONU offline → emit `onu.offline` → trigger alert + ticket

2. **Idempotent operations**
   - Isolir user yang sudah isolir → skip, tidak error
   - Activate user yang sudah aktif → skip, tidak error
   - Generate invoice yang sudah ada → skip, tidak duplicate

3. **Graceful degradation**
   - MikroTik tidak reachable → log error, lanjutkan, retry later
   - GenieACS tidak reachable → log error, lanjutkan, retry later
   - Redis tidak reachable → fallback ke DB query

4. **Reconciliation cron**
   - Daily: bandingkan PPP Secret di MikroTik vs radcheck → report mismatch
   - Daily: bandingkan ONU di GenieACS vs pppoe_users → report orphan
   - Daily: bandingkan invoice status vs payment status → fix inconsistency

5. **Connection management**
   - MikroTik: connection pool per router, reuse, cleanup idle
   - GenieACS: HTTP client with keep-alive, timeout 10s
   - PostgreSQL: GORM connection pool (max 50, idle 10)
   - Redis: connection pool (max 10)

6. **Multi-tenant ready (future-proofing)**
   - Tambah `tenantId` di semua model baru dari awal (nullable, default null = single tenant)
   - Query helper: `db.Scopes(TenantScope(tenantID))` untuk auto-filter
   - RADIUS config per-tenant: virtual server atau realm prefix

---

## Urutan Implementasi (Starting Point)

### Sprint 1 (Minggu 1-2): Quick Wins
```
Hari 1-2:  1.4 Skip Minggu Isolir        (0.5 hari)
Hari 2-4:  1.1 Invoice Discount          (2 hari)
Hari 4-6:  1.2 Cancel Invoice            (1.5 hari)
Hari 6-8:  1.5 Package Change Log        (2 hari)
Hari 8-10: 1.6 Installation Log          (2 hari)
Hari 10:   1.3 Edit Payment Method       (1 hari)
Hari 11-12:1.7 External API              (1.5 hari)
```

### Sprint 2 (Minggu 3-4): Foundation
```
Minggu 3:  1.8 Territory & Collector     (5 hari)
Minggu 4:  1.9 Settlement Report         (3 hari) + 1.10 MikroTik Pool (2 hari)
```

### Sprint 3-4 (Bulan 2): Billing Core Part 1
```
Minggu 5-6: 2.1 PSB 24h Deadline         (5 hari)
Minggu 7:   2.2 Payment Promise          (5 hari)
Minggu 8:   2.3 Profile Overrides        (3 hari) + testing (2 hari)
```

### Sprint 5-6 (Bulan 3): Billing Core Part 2
```
Minggu 9-10:  2.4 Waiting List           (5 hari)
Minggu 11:    2.5 ONT Removal Tasks      (5 hari)
Minggu 12:    2.6 MikroTik Hybrid Auth   (5 hari, optional)
```

### Sprint 7-10 (Bulan 4-5): Integration
```
Minggu 13: 3.1 ONU ↔ Customer Linking    (5 hari)
Minggu 14: 3.3 RX Power Monitoring       (5 hari)
Minggu 15: 3.2 Auto-Provisioning ONU     (5 hari)
Minggu 16: 3.5 Real-Time Dashboard       (5 hari)
Minggu 17: 3.4 Isolir via ONU            (3 hari, optional)
```

### Sprint 11-13 (Bulan 6-7): Automation
```
Minggu 18: 4.1 Event-Driven Notif        (5 hari)
Minggu 19: 4.2 Auto-Activation           (3 hari)
Minggu 20: 4.3 Auto-Provisioning Pipeline(5 hari)
Minggu 21: 4.4 Smart Alert Rules         (5 hari, optional)
```

### Sprint 14+ (Bulan 8+): Scaling
```
Minggu 22: 5.1 Redis Caching             (5 hari)
Minggu 23: 5.2 Job Queue (Asynq)         (5 hari)
Minggu 24: 5.5 API Rate Limiting         (2 hari)
Minggu 25-26: 5.3 Captive Portal         (10 hari)
Minggu 27+:  5.4 Multi-Tenant            (ongoing, strategic)
```

---

## Summary: Prioritas Implementasi

### HIGH (Dampak langsung ke operasional ISP)
| # | Fitur | Sprint | Estimasi |
|---|---|---|---|
| 1.4 | Skip Minggu Isolir | Sprint 1 | 0.5 hari |
| 1.1 | Invoice Discount | Sprint 1 | 2 hari |
| 1.2 | Cancel Invoice | Sprint 1 | 1.5 hari |
| 1.8 | Territory & Collector | Sprint 2 | 5 hari |
| 2.1 | PSB 24h Deadline | Sprint 3 | 5 hari |
| 2.2 | Payment Promise | Sprint 4 | 5 hari |
| 3.1 | ONU ↔ Customer Linking | Sprint 7 | 5 hari |
| 3.3 | RX Power Monitoring | Sprint 8 | 5 hari |
| 4.1 | Event-Driven Notif | Sprint 11 | 5 hari |
| 4.2 | Auto-Activation | Sprint 12 | 3 hari |

### MEDIUM (Meningkatkan efisiensi)
| # | Fitur | Sprint | Estimasi |
|---|---|---|---|
| 1.5 | Package Change Log | Sprint 1 | 2 hari |
| 1.6 | Installation Log | Sprint 1 | 2 hari |
| 1.3 | Edit Payment Method | Sprint 1 | 1 hari |
| 1.9 | Settlement Report | Sprint 2 | 3 hari |
| 1.10 | MikroTik Pool | Sprint 2 | 2 hari |
| 2.3 | Profile Overrides | Sprint 4 | 3 hari |
| 2.4 | Waiting List | Sprint 5 | 5 hari |
| 3.2 | Auto-Provisioning ONU | Sprint 9 | 5 hari |
| 3.5 | Real-Time Dashboard | Sprint 10 | 5 hari |
| 4.3 | Auto-Provisioning Pipeline | Sprint 13 | 5 hari |
| 5.1 | Redis Caching | Sprint 14 | 5 hari |
| 5.2 | Job Queue (Asynq) | Sprint 15 | 5 hari |
| 5.5 | API Rate Limiting | Sprint 16 | 2 hari |

### LOW (Nice to have)
| # | Fitur | Sprint | Estimasi |
|---|---|---|---|
| 1.7 | External API | Sprint 1 | 1.5 hari |
| 2.5 | ONT Removal Tasks | Sprint 6 | 5 hari |
| 2.6 | MikroTik Hybrid Auth | Sprint 6 | 5 hari |
| 3.4 | Isolir via ONU | Sprint 10 | 3 hari |
| 4.4 | Smart Alert Rules | Sprint 13 | 5 hari |
| 5.3 | Captive Portal | Sprint 17+ | 10 hari |
| 5.4 | Multi-Tenant | Sprint 27+ | Ongoing |

---

## Catatan Penting

1. **Incremental improvement** — setiap sprint menghasilkan fitur yang langsung bisa dipakai, tidak perlu tunggu semua selesai
2. **Tidak merusak yang sudah berjalan** — semua perubahan DB pakai migration (GORM AutoMigrate), tidak drop table
3. **Test sebelum deploy** — setiap fitur di-test di local sebelum push ke VPS
4. **Deploy command**: `sudo bash /var/www/salfanet-radius/vps-install/updater.sh --branch master`
5. **Backup DB sebelum migration** besar (Phase 1 territory, Phase 5 multi-tenant)
6. **Monitor performa** setelah Redis + Asynq — pastikan tidak ada memory leak
7. **Dokumentasi API** — update Swagger/OpenAPI spec setiap endpoint baru
8. **Code review** — setiap PR di-review sebelum merge ke master
