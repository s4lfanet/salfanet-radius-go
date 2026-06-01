# Changelog

All notable changes to Salfanet RADIUS are documented in this file.  
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).  
Versioning follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [2.54.15] — 2026-06-02
### Fixed
- **ONU offline kembali online setelah polling** — Saat Telnet tersedia (pool ada) tapi `FetchTelnetONUStates` mengembalikan 0 state (output tidak terbaca: timeout, format CLI tidak dikenali, dsb.), sebelumnya tidak ada log dan tidak ada fallback — SNMP OperState yang ter-cache (lag) langsung menang, semua ONU menjadi online. Fix: jika Telnet return 0 state, ambil status terakhir dari DB sebagai fallback. Setiap ONU yang SNMP-nya bilang `online` tapi DB mencatat `offline/dying_gasp/los` → status SNMP diabaikan dan status DB dipertahankan. Log warning ditambahkan untuk visibilitas.

### Files
- `internal/olt/poller/poller.go` — DB fallback ketika Telnet return 0 states; log warning untuk Telnet failure

---

## [2.54.14] — 2026-06-02
### Fixed
- **Revert v2.54.13** — Fix DeregReason>=2 → force offline ternyata salah. ZTE C320 `DeregReason` menyimpan alasan TERAKHIR ONU offline (historis), bukan status saat ini. ONU yang sudah kembali online tetap punya `DeregReason=PowerOff` dari event sebelumnya, sehingga semua 55 ONU dipaksa ke offline. Revert: `lastDeregReason` hanya dipakai sebagai informasi display, status tetap dari `OperState` SNMP + Telnet override.

### Files
- `internal/olt/vendors/zte/zte.go` — Revert DeregReason status override (v2.54.13)

---

## [2.54.13] — 2026-06-02
### Fixed
- **ONU offline ditampilkan sebagai online** — ZTE C320 SNMP `OperState` bisa tertinggal (cached) dan masih melaporkan nilai `working/active` (4/5) meski ONU sudah deregistrasi (PowerOff, LOS, dsb.). MIB ZTE mendefinisikan `DeregReason=1` = `notApplicable` (ONU sedang online), dan `DeregReason>=2` = ONU sedang offline. Fix: jika `DeregReason>=2` tetapi `OperState` di-decode sebagai `OnuOnline`, status di-override ke `OnuOffline`. Berlaku untuk kedua blok ONU (registered via regStatus dan registered via OperState).

### Files
- `internal/olt/vendors/zte/zte.go` — Override status ke offline jika DeregReason>=2 (ONU deregistered per ZTE MIB)

---

## [2.54.12] — 2026-06-02
### Fixed
- **totalONU tidak termasuk ghost ONU** — `totalONU` di header OLT (contoh "0/64") sebelumnya ter-inflate karena ghost ONU dari siklus sebelumnya ikut dihitung. Sekarang hanya ONU yang ditemukan SNMP siklus ini (registered + unregistered) yang masuk ke `totalOnu` di `network_olts`. Ghost ONU tetap di-pass ke `checkAlerts` untuk generate alert, tapi tidak mempengaruhi angka summary.
- **Ghost ONU inflate offlineOnu** — Ghost ONU yang ditandai offline sebelumnya ikut ditambahkan ke `offlineCount`, membuat angka `offlineOnu` di summary salah. Dihapus karena ghost ONU bukan live-offline event.
- **Semua ONU jadi offline setelah restart PM2** — Saat Telnet pool membuka koneksi baru (setelah restart), ZTE C320 kadang mengembalikan output pertama yang belum bersih. `parseONUStateOutput` salah parse → semua ONU di-override ke offline. Fix: tambah sanity check — jika Telnet ingin men-set SEMUA SNMP-online ONU menjadi non-online, skip Telnet override untuk siklus itu dan pertahankan status SNMP.

### Files
- `internal/olt/poller/poller.go` — Telnet sanity check; totalONU = SNMP count only; hapus offlineCount++ untuk ghost ONU

---

## [2.54.11] — 2026-06-02
### Added
- **LastDeregReason ONU dari SNMP** — Alasan terakhir ONU offline (PowerOff, LOS, Reboot, AuthFail, dll.) sekarang dibaca langsung dari SNMP OID `.3.50.12.1.1.7` (zxAnGponOnuRegTable kolom 7 — DeregReason) dan disimpan ke DB field `lastDeregReason`. Upsert menggunakan COALESCE agar nilai tidak tertimpa NULL saat OLT tidak mengembalikan data.

### Files
- `internal/olt/vendors/zte/zte.go` — tambah `oidDeregReason`, field `LastDeregReason` di `ONUInfo`, `deregReason` di `ponResult`, BulkWalk OID baru, decode & populate `LastDeregReason`, fungsi `decodeDeregReason()`
- `internal/olt/poller/poller.go` — set `base.LastDeregReason` dari ONU info; tambah `lastDeregReason` ke COALESCE upsert assignments

---

## [2.54.10] — 2026-06-02
### Fixed
- **Semua ONU jadi offline saat SNMP gagal** — Ghost-cleanup poller menandai semua ONU sebagai offline ketika SNMP BulkWalk mengembalikan 0 ONU (misalnya OLT sementara tidak bisa diakses). Seharusnya ghost-cleanup hanya dijalankan bila SNMP berhasil mendeteksi minimal 1 ONU di siklus polling ini. Fix: tambah guard `snmpDiscoveredAny` sebelum ghost-cleanup.
- **OLT tampil "Online" meski SNMP gagal** — `isOnline` selalu di-set `true` bahkan ketika SNMP gagal. Fix: `isOnline = snmpDiscoveredAny || uptimeSeconds > 0`.

### Files
- `internal/olt/poller/poller.go` — ghost-cleanup hanya jalan jika SNMP return > 0 ONU; `isOnline` sekarang refleksikan konektivitas nyata

---

## [2.54.9] — 2026-06-01
### Fixed
- **ONU power-off tampil "online"** — `parseONUStateOutput` tidak menangani Phase State `power-off` dari CLI ZTE C320 (`show gpon onu state`). Saat ONU mati lampu/power-off, CLI mengembalikan `power-off` tapi karena tidak ada case-nya, Telnet override tidak terjadi dan status SNMP yang stale (`working`/`active`) tetap dipakai. Fix: tambah `power-off`, `power_off`, `poweroff`, `powerdown` → `offline`; juga tambah `losi` → `los` dan `auth-failed` → `auth_failed`.

### Files
- `internal/olt/vendors/zte/zte.go` — `parseONUStateOutput` + komentar lengkap Phase State ZTE C320

---

## [2.54.8] — 2026-06-01
### Fixed
- **`toLocaleString` crash** — `deleteOverlay.count` bisa undefined saat hapus expired, crash dengan `TypeError: Cannot read properties of undefined`. Fix: tambah `?? 0` guard.
- **Hapus voucher (selected) gagal** — Frontend kirim `voucherIds` tapi backend `DeleteMultiple` hanya baca `ids`. Fix: backend sekarang baca keduanya.
- **`delete-expired` count undefined** — Backend return `deleted` tapi frontend baca `data.count`. Fix: frontend baca `data.deleted ?? data.count ?? 0`; backend juga tambah alias `count`.
- **Pagination per-page tidak berubah** — `pageParams` di backend baca `pageSize` tapi frontend kirim `limit`. Fix: baca keduanya, cap dinaikkan ke 2000.

### Files
- `internal/api/handlers/helpers.go` — `pageParams` terima `limit` alias + cap 2000
- `internal/api/handlers/hotspot_ext.go` — `DeleteMultiple` terima `voucherIds`; `DeleteExpired` tambah alias `count`
- `src/app/admin/hotspot/voucher/page.tsx` — guard `toLocaleString`, baca `data.deleted` dari delete-expired

---

## [2.54.7] — 2026-06-01
### Fixed
- **Generate voucher error Duplicate entry** — `generateVoucherCode` menggunakan rumus deterministik berbasis waktu (`t + i*31337`) sehingga mudah tabrakan dengan kode yang sudah ada. Fix: (1) ganti ke `crypto/rand` untuk randomness yang sesungguhnya, (2) fetch semua kode existing sebelum generate dan lakukan retry max 20x jika ada collision.

### Files
- `internal/api/handlers/hotspot.go` — `generateVoucherCode` pakai `crypto/rand`; `GenerateVouchers` precheck existing codes + retry loop

---

## [2.54.6] — 2026-06-01
### Fixed
- **Kolom Router tabel agen selalu "Belum ditugaskan"** — `GET /api/hotspot/agents` hanya mengirim `routerId` tapi tidak menyertakan object router. Frontend mengecek `agent.router?.name` yang selalu null. Fix: backend sekarang fetch semua router, lookup berdasarkan `routerId`, dan sertakan `{ id, name, nasname, shortname }` di setiap item response.

### Files
- `internal/api/handlers/hotspot_ext.go` — `ListAgents`: tambah lookup router map + sertakan `router` object di response

---

## [2.54.5] — 2026-06-01
### Fixed
- **Error 500 saat tambah agent voucher** — Go struct `Agent` memiliki field `PIN string` tanpa tag `gorm:"-"`, menyebabkan GORM mencoba INSERT ke kolom `pin` yang tidak ada di tabel `agents` database. Fix: tambah `gorm:"-"` ke field `PIN`.

### Files
- `internal/db/models/extra.go` — Tambah `gorm:"-"` ke field `PIN` di struct `Agent`

---

## [2.54.4] — 2026-06-01
### Fixed
- **Status RADIUS tetap "Pending" setelah sync** — Root cause: Go struct `PppoeProfile` tidak memiliki field `SyncedToRadius`, padahal kolom `syncedToRadius` ada di database (Prisma schema). Akibatnya `h.db.Model(&p).Update("syncedToRadius", true)` silently fail karena GORM tidak menemukan field di struct. Fix: tambah `SyncedToRadius bool \`gorm:"default:false;column:syncedToRadius"\`` ke struct + ganti update dengan `h.db.Exec("UPDATE pppoe_profiles SET syncedToRadius = 1 WHERE id = ?", p.ID)` agar pasti berhasil.

### Files
- `internal/db/models/models.go` — Tambah `SyncedToRadius` ke `PppoeProfile` struct
- `internal/api/handlers/pppoe_ext.go` — Ganti `Model(&p).Update` dengan `Exec` langsung

---

## [2.54.3] — 2026-06-01
### Added / Fixed
- **Implementasi Sync PPPoE Profile ke MikroTik** — Handler `SyncProfilesMikrotik` sebelumnya selalu return 501 Not Implemented. Sekarang: connect ke setiap router via RouterOS API (port 8728, fallback 8729), buat/update IP pool (jika `poolRanges` diisi), buat/update PPP profile dengan `rate-limit`, `remote-address` (pool), `local-address`. Simpan `ipPoolName`, `ipPoolRange`, `localAddress`, `rateLimit` ke database. Return `savedProfile` agar frontend update state tanpa reload.

### Files
- `internal/api/handlers/pppoe_ext.go` — Implementasi penuh `SyncProfilesMikrotik`

---

## [2.54.2] — 2026-06-01
### Fixed
- **Test Koneksi MikroTik 404** — Frontend mengirim `PUT /api/pppoe/profiles/sync-mikrotik` tapi Go router tidak punya handler untuk method `PUT` di path tersebut → 404. Fix: tambah handler `TestMikrotikConnection` yang connect ke MikroTik via RouterOS API, test identity + PPP profile read/write, return detail hasil per port (8728/8729).

### Files
- `internal/api/handlers/pppoe_ext.go` — Tambah `TestMikrotikConnection` handler (PUT)
- `internal/api/router.go` — Register `PUT /profiles/sync-mikrotik → TestMikrotikConnection`

---

## [2.54.1] — 2026-06-01
### Fixed
- **Router list kosong di modal "Sync ke MikroTik"** — `GET /api/network/routers` mengembalikan `{ routers: [...], vpnClients: [...] }` tapi frontend PPPoE profiles melakukan `Array.isArray(data)` yang selalu false → `setRouters([])`. Fix: gunakan `data.routers` sebagai sumber list router.
- **Status RADIUS tetap "Pending" saat profile belum pernah sync MikroTik** — `SyncProfilesRadius` hanya sync profil yang punya field `rateLimit` terisi (diisi saat sync MikroTik). Profil baru tanpa rateLimit selalu di-skip. Fix: bangun `rateLimit` otomatis dari `uploadSpeed`/`downloadSpeed` dalam format `{up}M/{down}M` jika `rateLimit` nil.

### Files
- `src/app/admin/pppoe/profiles/page.tsx` — `loadRouterList` & `handleSyncMikrotik`: parse `data.routers` bukan `data`
- `internal/api/handlers/pppoe_ext.go` — `SyncProfilesRadius`: fallback build rateLimit dari speed fields

---

## [2.54.0] — 2026-06-01
### Fixed
- **Status RADIUS tetap "Pending" setelah sync** — `SyncProfilesRadius` melakukan sync ke `radgroupreply` tapi tidak meng-update field `syncedToRadius` di tabel `pppoe_profiles`. Fix: tambah `h.db.Model(&p).Update("syncedToRadius", true)` setelah setiap profil berhasil di-sync.

### Files
- `internal/api/handlers/pppoe_ext.go` — Update `syncedToRadius = true` setelah sync radius berhasil

---

## [2.53.9] — 2026-06-01
### Fixed
- **Hotspot profile Harga Jual selalu Rp 0** — Field `sellingPrice` tidak disertakan dalam request body saat create/update. Frontend hanya menghitung `sellingPrice` untuk tampilan form saja, tidak mengirimkannya ke API. Fix: tambah `sellingPrice: costPrice + resellerFee` ke body request.
- **Sync PPPoE ke FreeRADIUS tidak berfungsi** — Tabel `radgroupreply` tidak memiliki UNIQUE constraint pada `(groupname, attribute)`, hanya index biasa. `ON DUPLICATE KEY UPDATE` tidak pernah trigger sehingga setiap sync insert baris baru (duplikat) tanpa update. Fix: ganti dengan `DELETE WHERE groupname+attribute` lalu `INSERT`. Sekaligus support sync per-profile (berdasarkan `id` dari body request).

### Files
- `src/app/admin/hotspot/profile/page.tsx` — Tambah `sellingPrice` ke body API
- `internal/api/handlers/pppoe_ext.go` — Fix `SyncProfilesRadius`: DELETE+INSERT, support sync by `id`

---

## [2.53.8] — 2026-06-01
### Fixed
- **PPPoE Paket Layanan tidak tampil setelah tambah** — `ListProfiles` Go handler mengembalikan array langsung (`[...]`) sedangkan frontend menggunakan `data.profiles || []`. Fix: bungkus response dalam `{profiles: [...]}` agar konsisten dengan HotspotHandler.
- **Hotspot profile error 400 saat tambah** — Frontend mengirim `costPrice`, `resellerFee`, `sharedUsers`, `validityValue` sebagai string dari formData, tapi Go model ekspektasi `int`. Fix: parse dengan `parseInt()` sebelum dikirim.
- **PPPoE & Hotspot profile edit (PUT) gagal** — URL PUT tidak menyertakan `:id` di path (`/api/pppoe/profiles` instead of `/api/pppoe/profiles/:id`). Fix: ubah URL dynamis ke `/api/pppoe/profiles/${id}` dan `/api/hotspot/profiles/${id}` saat edit.

### Files
- `internal/api/handlers/pppoe.go` — Wrap `ListProfiles` response dalam `fiber.Map{"profiles": profiles}`
- `src/app/admin/hotspot/profile/page.tsx` — Parse costPrice, resellerFee, sharedUsers, validityValue ke int; fix PUT URL
- `src/app/admin/pppoe/profiles/page.tsx` — Fix PUT URL menyertakan `/:id`

---

## [2.53.7] — 2026-06-01
### Fixed
- **CSP violation — Leaflet CSS dari `cdnjs.cloudflare.com` diblokir** — `src/proxy.ts` (Next.js middleware) set `Content-Security-Policy` header tanpa `https://cdnjs.cloudflare.com` di `style-src` dan `font-src`. Halaman `/admin/network/map` dan `/admin/network/unified-map` menggunakan Leaflet CSS dari CDN tersebut sehingga stylesheet diblokir browser. Fix: tambah `https://cdnjs.cloudflare.com` ke `style-src` dan `font-src` di `proxy.ts`.

### Files
- `src/proxy.ts` — Tambah `https://cdnjs.cloudflare.com` ke `style-src` dan `font-src` di CSP header

---

## [2.53.6] — 2026-06-01
### Fixed
- **Error 401 pada Peta Jaringan — `GET /api/customers/with-location`** — Cloudflare (Flexible SSL) dapat memproses request ke Go backend tanpa meneruskan cookie `__Secure-next-auth.session-token` dengan benar, sehingga `CombinedAuthMiddleware` gagal validasi dan return 401. Fix: pindahkan route ke Next.js sebagai API route (`src/app/api/customers/with-location/route.ts`) menggunakan `getServerSession` (server-side, tidak bergantung pada cookie forwarding). Nginx perlu diupdate: tambahkan `location /api/customers/` → port 3000 sebelum catch-all `location /api/`.
- **Response format `GET /api/customers/with-location` salah (Go handler)** — Go handler mengembalikan `{success, customers:[]}` tapi frontend menggunakan `customersData.data`. Fix: response format diubah ke `{success: true, data: [...], count: N}`. Tambah juga support query param `?limit=` (default 2000, max 5000).

### Files
- `src/app/api/customers/with-location/route.ts` — **NEW** Next.js API route handler dengan Prisma query dan `checkAuth()`
- `internal/api/handlers/network_ext.go` — Fix response format `customers` → `data`, tambah `count`, support `?limit`

---

## [2.53.5] — 2026-06-01
### Fixed
- **`GET /api/settings/isolation` 500** — Handler memanggil `db.First(&company)` dan return 500 saat tabel `companies` kosong. Sekarang `ErrRecordNotFound` ditangani: return `{success:true, data:{}}` (default kosong). `UpdateIsolationSettings` juga difix untuk tidak 500 saat tidak ada company.
- **Menu Isolir (IsolatedUsers) format response salah** — Handler lama: field names salah (`profile`/`price`/`unpaidAmt`/`unpaidCnt` bukan `profileName`/`profilePrice`/`totalUnpaid`/`unpaidInvoicesCount`), tidak ada `success:true`, tidak ada `stats`, tidak ada field `email`, `customerId`, `isOnline`, `ipAddress`, `loginTime`, `nasIp`, `unpaidInvoices[]`. Handler ditulis ulang lengkap dengan JOIN radacct untuk status online dan batch query unpaid invoices.

### Files
- `internal/api/handlers/settings.go` — Fix `GetIsolationSettings` dan `UpdateIsolationSettings` handle `ErrRecordNotFound`
- `internal/api/handlers/admin.go` — Rewrite `IsolatedUsers` handler dengan response format lengkap

---

## [2.53.4] — 2026-05-31
### Fixed
- **Hotspot — Model Go tidak cocok dengan skema DB nyata** — Model `HotspotProfile` memakai field lama (`Price`, `Duration`, `BandwidthDown`, dll) yang tidak ada di DB. Model sekarang memakai kolom DB yang benar: `costPrice`, `sellingPrice`, `resellerFee`, `validityValue`, `validityUnit`, `speed`, `groupProfile`, `sharedUsers`, `agentAccess`, `eVoucherAccess`. Model `HotspotVoucher` difix: kolom `batchId` → `batchCode`, status `UNUSED` → `WAITING`, ditambah `routerId`, `voucherType`, `codeType`, `firstLoginAt`. Model `Agent` difix: ditambah `minBalance`, `routerId`, `lastLogin`.
- **`ListProfiles` response salah** — Mengembalikan array biasa `[]`, frontend expect `{profiles:[]}`. Sekarang dibungkus dengan `fiber.Map{"profiles": profiles}`.
- **`ListVouchers` rusak total** — Query param `batchId` (tidak ada di frontend), response `{data,total}` (frontend expect `{vouchers,batches,totalPages,total,stats}`). Ditulis ulang: filter `profileId`, `batchCode`, `status`, `agentId`, `routerId`; response lengkap dengan distinct batch list dan stats (total/waiting/active/expired).
- **`GenerateVouchers` field salah** — Body `count` → `quantity`, tidak ada `batchCode`/`codeLength`/`voucherType`/`codeType`. Response `batchId` → `batchCode`. Status `UNUSED` → `WAITING`.
- **`RekapVoucher` struktur salah** — Return `{success, data:[{profileId,profileName,total,used,unused}]}`. Ditulis ulang: per-batch SQL GROUP BY, join profil & agen, kalkulasi finansial (`sold`, `totalRevenue`, `agentProfit`, `adminEarnings`). Response `{rekap:[...], agents:[], profiles:[]}`.
- **`ExportRekap` return JSON bukan blob** — Sekarang generate XLSX menggunakan excelize, Content-Type `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`.
- **`AgentHistory` abaikan year/month filter** — Frontend kirim `?year=X&month=X` untuk detail bulan, `GET` tanpa param untuk rekap bulanan. Sekarang keduanya didukung. Tanpa param → monthly breakdown 12 bulan terakhir. Dengan param → detail sales bulan tersebut.
- **9 route hilang** — Ditambahkan: `POST /hotspot/agents`, `PUT /hotspot/agents`, `DELETE /hotspot/agents`, `POST /hotspot/agents/balance`, `DELETE /hotspot/voucher`, `POST /hotspot/voucher/delete-expired`, `GET /hotspot/voucher/bulk`, `PATCH /hotspot/voucher`, `GET /hotspot/voucher`.
- **`ListAgents` tidak ada `voucherStock` dan `stats`** — Diperkaya dengan `voucherStock` (count WAITING voucher per agen) dan `stats.currentMonth`/`stats.allTime` dari tabel `agent_sales`.
- **BulkGenerate menggunakan `BatchID` lama** — Difix ke `BatchCode`, status `WAITING`, support `codeLength`/`voucherType`/`codeType`.
- **Sisa bug**: `BulkDelete` & `ValidateVoucher` status `UNUSED` → `WAITING`, `DeleteExpired` kolom `expires_at` → `expiresAt`.

### Files
- `internal/db/models/extra.go` — Rewrite `HotspotProfile`, `HotspotVoucher`, `Agent`, `AgentSale` sesuai DB schema
- `internal/api/handlers/hotspot.go` — Rewrite `ListProfiles`, `ListVouchers`, `GenerateVouchers`
- `internal/api/handlers/hotspot_ext.go` — Rewrite total: semua handler fix, tambah `BulkGetOrExport`, `BulkEdit`, `DeleteBatch`, `CreateAgent`, `UpdateAgent`, `DeleteAgent`, `AdjustBalance`
- `internal/api/router.go` — Tambah 9 route yang hilang
- `internal/cron/pppoe_session_sync.go` — Fix `agent.Commission` (field tidak ada di DB)
- `internal/api/handlers/agent.go` — Fix `AgentSale` struct literal (`VoucherID` tidak ada)
- `internal/api/handlers/evoucher_handler.go` — Fix `profile.price` → `profile.SellingPrice`


### Fixed
- **Template Excel import PPPoE tidak punya field koordinat peta & field lainnya** — Template hanya punya 12 kolom, tidak ada `latitude`, `longitude`, `macAddress`, `billingDay`, `comment`. Form tambah pelanggan sudah punya semua field ini tapi file Excel template/export tidak bisa mengisinya. Ditambahkan ke template (17 kolom), export, dan parser import.
### Files
- `internal/api/handlers/pppoe_ext.go` — `BulkGet` template headers & example row, export headers & per-row output; `BulkImport` parser field baru

---

## [2.53.2] — 2026-05-31
### Fixed
- **Suspend Requests handler** — Response format tidak cocok dengan frontend: response key `data` → `rows`, status uppercase mismatch, fields lengkap (startDate, endDate, adminNotes, approvedAt, approvedBy). Rewrite handler pakai GORM Preload bukan raw SQL custom struct.
- **Topup Requests — model salah** — Backend query `agent_deposits` (deposit agen) padahal frontend menampilkan top-up PPPoE customer. Dibuat model baru `TopupRequest` + tabel `topup_requests` (CREATE TABLE IF NOT EXISTS di runMigrations), customer portal `CreateTopupRequest` sekarang menyimpan ke tabel baru, admin handler query tabel yang benar, status `PENDING`/`SUCCESS`/`FAILED`.
- **BulkCreateUsers tanpa RADIUS sync** — User dibuat di DB tapi tidak disync ke FreeRADIUS `radcheck`. Ditambahkan upsert `Cleartext-Password` ke `radcheck` setelah setiap user berhasil dibuat.
- **SyncProfilesRadius adalah stub** — Hanya menghitung jumlah profil. Sekarang benar-benar upsert `Mikrotik-Rate-Limit` ke tabel `radgroupreply` untuk setiap profil aktif yang memiliki `rateLimit` dan `groupName`.
- **SyncProfilesMikrotik return fake 200** — Diganti return `501 Not Implemented` dengan pesan jelas bahwa fitur ini belum diimplementasi.
### TypeScript
- TSC check: **0 errors** (npx tsc --noEmit exit 0)
### Files
- `internal/api/handlers/admin.go` — Rewrite `TopupRequests`, `ApproveTopup`, `RejectTopup`, `SuspendRequests`
- `internal/api/handlers/customer_portal.go` — `CreateTopupRequest` simpan ke `topup_requests` bukan `transactions`
- `internal/api/handlers/pppoe_ext.go` — `BulkCreateUsers` + RADIUS sync, `SyncProfilesRadius` implementasi nyata, `SyncProfilesMikrotik` → 501
- `internal/db/models/extra.go` — Tambah model `TopupRequest`
- `internal/db/db.go` — Tambah `CREATE TABLE IF NOT EXISTS topup_requests` di `runMigrations`

---

## [2.53.1] — 2026-05-31
### Fixed
- **FreeRADIUS backup/restore/download/upload semua stub** — Semua 5 handler di `admin_misc_handler.go` hanya menyimpan ke DB atau return dummy response tanpa melakukan operasi nyata. Diimplementasi ulang sepenuhnya:
  - `ListFreeradiusBackups` → scan filesystem `/var/www/salfanet-radius/backups/freeradius/*.tar.gz` + baca `/tmp/salfanet-fr-backup.log`
  - `CreateFreeradiusBackup` → jalankan `scripts/backup-freeradius-local.sh` di background goroutine, log ke `/tmp/salfanet-fr-backup.log`
  - `DownloadFreeradiusBackup` → `c.Download()` file langsung dari disk (bukan return URL JSON)
  - `RestoreFreeradiusBackup` → extract tar.gz → copy ke `/etc/freeradius/3.0/` → fix ownership `freerad:freerad` → `systemctl restart freeradius`
  - `UploadFreeradiusBackup` → `c.SaveFile()` ke backup dir → return `{savedAs: filename}` sesuai kontrak frontend
- Script backup `scripts/backup-freeradius-local.sh` diperbaiki permission (sebelumnya `-rw-r--r--`, tidak executable)
### Files
- `internal/api/handlers/admin_misc_handler.go` — Implementasi penuh semua 5 FreeRADIUS backup handler

---

## [2.53.0] — 2026-05-31
### Fixed
- **FreeRADIUS tidak merespons RADIUS request dari MikroTik (`radius timeout`)** — Root cause: `systemctl reload-or-restart` (SIGHUP) tidak me-reload `clients.d` di FreeRADIUS 3.x; client list hanya terbaca saat full restart. Akibatnya meski `nas-from-db.conf` sudah berisi entry NAS yang benar, FreeRADIUS tetap menganggap MikroTik sebagai unknown client dan drop packet tanpa respons. Fix: ganti ke `systemctl restart freeradius` di `nas_sync.go`.
### Files
- `internal/radius/nas_sync.go` — `reload-or-restart` → `restart` agar clients.d selalu ter-load

---

## [2.52.99] — 2026-05-31
### Fixed
- **FreeRADIUS integration: NAS tidak ter-trigger ke `clients.d/nas-from-db.conf`** — Saat router/NAS ditambah via UI, handler Go `CreateRouter/UpdateRouter/DeleteRouter` hanya menulis ke DB tanpa men-sync ke file FreeRADIUS clients. File `nas-from-db.conf` di VPS kosong (hanya header), akibatnya MikroTik tidak dikenali sebagai client RADIUS sah → autentikasi PPPoE tidak bisa terjadi. Sync sebelumnya hanya ada di TypeScript service yang sudah tidak dipanggil.
- **Manual immediate fix di VPS** — File `/etc/freeradius/3.0/clients.d/nas-from-db.conf` ditulis manual untuk NAS `DST-PASKA` (10.201.0.10) + reload FreeRADIUS. Sekarang FreeRADIUS aktif listen di `0.0.0.0:1812` dengan NAS dikenali.
### Added
- **`internal/radius/nas_sync.go` — `SyncNASClients(db)`** — Helper Go yang membaca tabel `nas` (LEFT JOIN `vpn_clients` + `vpn_servers`), generate file `nas-from-db.conf` (per-NAS client + CHR VPN gateway entries untuk kasus masquerade), tulis hanya jika berubah, lalu `systemctl reload-or-restart freeradius`. Mutex + cooldown 3 detik untuk anti-burst.
- **Auto-trigger di handler** — `CreateRouter`, `UpdateRouter`, `DeleteRouter` sekarang memanggil `radius.SyncNASClients(h.db)` setelah perubahan DB.
- **Cron `freeradius_health` regenerate file** — Cron tetap berfungsi sebagai self-healing kalau ada drift atau file dihapus.
### Files
- `internal/radius/nas_sync.go` — File baru: NAS → clients.d sync + reload
- `internal/api/handlers/network.go` — `CreateRouter` panggil `SyncNASClients`
- `internal/api/handlers/network_ext.go` — `UpdateRouter` & `DeleteRouter` panggil `SyncNASClients`
- `internal/cron/pppoe_session_sync.go` — `jobFreeRADIUSHealth` panggil `SyncNASClients` per run

## [2.52.98] — 2026-05-31
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

## [2.52.97] — 2026-06-01
### Fixed
- **ONUDetail: telnet pool race condition → ONU count naik 55→66, banyak offline** — Handler `ONUDetail` sebelumnya memakai shared telnet pool milik poller (`h.poller.GetPool`). Pool `acquire()` tidak menahan lock per-batch, sehingga dua goroutine bisa mendapat sesi yang sama dan command saling interleave. Akibatnya `FetchTelnetONUStates` mendapat output kacau → override state gagal → SNMP state yang lag dipakai → ONU tampil offline; sesi telnet crash di tengah poll → ghost ONU cleanup salah menandai ONU sebagai offline dan jumlah ONU melonjak. Solusi: ONUDetail **selalu membuat private telnet pool** (bukan reuse shared pool), agar poller tidak terganggu.
### Files
- `internal/api/handlers/misc_handler.go` — Ganti `h.poller.GetPool(oltID)` + conditional `ownPool` menjadi selalu `telnet.NewPool(tcfg)` + `defer pool.Close()`

## [2.52.96] — 2026-06-01
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

## [2.52.95] — 2026-05-31
### Added
- **ONU detail: Traffic Profile dari PPPoE** — Karena ZTE C320 tidak mengekspos DBA/bandwidth profile via CLI, limit bandwidth sekarang diambil dari profil PPPoE customer yang terhubung. Menampilkan nama profil, download limit, dan upload limit (dalam Kbps/Mbps) di section "ONT Build Configuration".
- **ONU detail: TR-069 / GenieACS status** — Section baru menampilkan apakah GenieACS sudah dikonfigurasi di sistem. Jika ya, tampilkan host dan username; jika tidak, tampilkan pesan panduan konfigurasi.
- **ONU detail: preload Customer.Profile** — Backend sekarang melakukan `Preload("Customer.Profile")` sehingga data profil PPPoE (nama, kecepatan download/upload) tersedia di respons API.
### Files
- `internal/api/handlers/misc_handler.go` — Ganti `Preload("Customer")` ke `Preload("Customer.Profile")`, tambah GenieACS settings check, tambah `bandwidth` dan `tr069` ke respons
- `src/app/admin/olt/[id]/page.tsx` — Tambah `bandwidth` dan `tr069` dari respons API, tambah "Traffic Profile (PPPoE Radius)" cards di ONT Build Config, tambah section "TR-069 / GenieACS"

## [2.52.94] — 2026-05-31
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

## [2.52.92] — 2026-05-31
### Fixed
- **RX Power / TX Power / Distance menjadi kosong (—) setelah poll** — Upsert menggunakan `VALUES(rxPower)` yang selalu menimpa nilai DB, termasuk dengan NULL. Jika SNMP tidak berhasil mendapat data optik dalam satu siklus poll (karena OLT sibuk, Telnet overlap, dll), nilai yang sebelumnya valid terhapus. Fix: ubah ke `COALESCE(VALUES(rxPower), rxPower)` — sama seperti yang sudah dipakai untuk `serialNumber`/`description`. Berlaku juga untuk `txPower` dan `distance`.
### Files
- `internal/olt/poller/poller.go` — `rxPower`, `txPower`, `distance` di DoUpdates upsert kini pakai COALESCE

## [2.52.91] — 2026-05-31
### Fixed
- **ONU dying-gasp/offline tampil sebagai "online"** — Root cause: ZTE C320 SNMP OperState agent **tidak segera update** saat ONU dying gasp; SNMP masih return `working` (4/5) meski ONU sudah mati. Fix: tambah `FetchTelnetONUStates()` yang menjalankan `show gpon onu state gpon-olt_F/S/P` via Telnet CLI per PON port (batch dalam 1 sesi), lalu override status SNMP dengan kolom "Phase State" dari output ZTE.
- **Bug regresi: semua ONU offline** — Fix pertama memiliki bug parser: ZTE C320 V2.1 menggunakan 5 kolom (`OnuIndex | Admin State | OMCC State | Phase State | Channel`), oper state ada di kolom ke-4 (fields[3]), bukan fields[2]. Parser lama membaca "OMCC State" = "enable" → jatuh ke default offline. Fix: scan SEMUA fields setelah ONU index untuk kata state yang dikenal; jika tidak dikenal, status tidak dioverride (SNMP preserved).
  Kata state yang dikenali: `working`/`active`/`online`/`up` → online, `dying*` → dying_gasp, `los`/`lofi` → los, `inactive`/`not-present`/`offline`/`down` → offline.
### Files
- `internal/olt/vendors/zte/zte.go` — Tambah `FetchTelnetONUStates()` + `parseONUStateOutput()` (scan semua fields)
- `internal/olt/poller/poller.go` — Panggil `FetchTelnetONUStates` setelah distance enrichment, override `onu.Status` hanya saat state dikenali

## [2.52.90] — 2026-05-31
### Fixed
- **Ghost ONU cleanup tidak berjalan (GORM bug)** — `Model(&models.OLTONUStatus{})` dengan primary key kosong menyebabkan GORM menambah kondisi `WHERE id = ''` secara diam-diam, membuat batch UPDATE tidak pernah mengeksekusi satu baris pun. Fix: ganti ke `.Table("olt_onu_status")` yang tidak punya kondisi PK implisit.
- **Ghost ONUs tidak masuk `checkAlerts`** — Karena ghost ONUs di-append setelah `allStatuses` dibuat, mereka tidak diproses oleh alert engine → tidak ada alert offline yang dibuat. Fix: load ghost ONUs terlebih dulu via `Find`, lalu append ke `allStatuses` SEBELUM `checkAlerts` dipanggil.
- **`totalONU` tidak menghitung ghost ONUs** — Summary OLT (`onlineOnu`, `offlineOnu`, `totalOnu`) tidak mencerminkan ONUs yang hilang dari SNMP walk. Sekarang ghost ONUs masuk ke `allStatuses` sehingga `totalONU` dan `offlineOnu` dihitung dengan benar.
### Files
- `internal/olt/poller/poller.go` — Rewrite ghost ONU cleanup: `Find` + `Table("olt_onu_status")` + append ke `allStatuses` sebelum `totalONU` dihitung

## [2.52.89] — 2026-05-31
### Fixed
- **ONU DyingGasp tidak terdeteksi** — `decodeOperState` hanya mengenal nilai 4 dan 5 (online) dan semua nilai lainnya sebagai offline, sehingga nilai 6 (DyingGasp dari ZTE MIB: `zxAnGponOnuRegOperStatus`) ikut dibaca sebagai offline biasa. Fix: tambah `case 6 → dying_gasp`.
- **ONU offline tetapi terbaca online di tabel** — Jika ONU mati mendadak (tanpa dying gasp), ZTE bisa menghapusnya dari tabel SNMP `zxAnGponOnuRegTable` sepenuhnya. Poller tidak memproses ONU tersebut sama sekali → status lama di DB (`online`) tidak diupdate → ONU kelihatan online padahal sudah mati. Fix: setelah upsert, semua ONU yang `updatedAt < poll_start` (tidak tersentuh upsert = tidak ada di SNMP walk) ditandai offline secara otomatis.
- **Alert DyingGasp belum ada** — `checkAlerts` hanya menangani `OnuOffline` dan `OnuOnline`, tidak ada penanganan `OnuDyingGasp`. Fix: tambah case DyingGasp → create alert dengan severity `critical` dan pesan "kemungkinan listrik mati mendadak". Alert DyingGasp juga di-resolve saat ONU kembali online.
- **Recovery message (online kembali) lebih akurat** — Sebelumnya hanya resolve alert `onu_offline`. Sekarang juga resolve `dying_gasp` alert saat ONU recovery.
### Files
- `internal/olt/vendors/zte/zte.go` — `decodeOperState`: tambah `case 6 → OnuDyingGasp`
- `internal/olt/poller/poller.go` — Ghost ONU cleanup setelah upsert; DyingGasp case di `checkAlerts`; resolve loop untuk `onu_offline` + `dying_gasp` di case online

## [2.52.88] — 2026-05-31
### Fixed
- **RX Power / TX Power tampil 101.07 dBm** — ZTE C320 mengembalikan `0xFFFF` (65535) via SNMP sebagai nilai sentinel "no data". Sebelumnya nilai ini lolos filter `rxRaw > 0` dan dihitung: `65535/500 - 30 = 101.07 dBm`. Fix: tambah filter `rxRaw != 0xFFFF` di parser ZTE SNMP. Fix juga di layer SQL `ListONUs`: `CASE WHEN rxPower <= 30 THEN rxPower ELSE NULL END` agar nilai lama di DB tidak tampil sampai poller update berikutnya.
- **Data ONU lambat aktual** — Default minimum poller interval diubah dari 60s → 30s. Sebelumnya jika `PollingInterval` tidak dikonfigurasi (0), interval fallback ke 60s. Sekarang fallback ke 30s sehingga data DB maksimal 30s dari kondisi aktual OLT.
### Files
- `internal/olt/vendors/zte/zte.go` — Filter `rxRaw != 0xFFFF` dan `txRaw != 0xFFFF` di dua blok ONUInfo builder
- `internal/olt/poller/poller.go` — Default minimum interval: 60s → 30s
- `internal/api/handlers/olt.go` — Filter SQL `CASE WHEN rxPower/txPower <= 30` di `ListONUs`

## [2.52.87] — 2026-05-31
### Added
- **Clean Config ONU dari OLT** — Tombol "Clean" (kuning) di tabel ONU list mengirim `restore default` ke interface `gpon-onu_1/{slot}/{port}:{onuId}` via Telnet. ONU tetap terdaftar di PON port, hanya konfigurasi service (VLAN, profile) yang di-reset. Endpoint: `POST /api/olt/:id/onus/:onuId/clean-config`.
- **Fix Delete ONU endpoint** — Frontend memanggil `DELETE /api/olt/:id/onus/:onuId/delete` tapi route belum ada. Route baru ditambahkan, mengarah ke handler `DeregisterONU` yang sudah ada.
### Fixed
- **TxPower tidak tersimpan saat poll** — Poller SNMP sudah memparse `TxPower` dari ZTE via SNMP (`oidTxPower`) tapi nilai tidak di-assign ke `base.TxPower` dan tidak masuk ke upsert `DoUpdates`. Fix: tambah `base.TxPower = onu.TxPower` dan `txPower: COALESCE(...)` ke upsert.
- **Table ONU lambat update setelah mutasi** — Setelah delete ONU, sekarang memanggil `fetchLiveOnus()` segera (bukan hanya `fetchOLT()`). Interval auto-refresh dipercepat dari 30s → 15s.
### Files
- `internal/olt/vendors/zte/zte.go` — Tambah `CleanONUConfig()`
- `internal/api/handlers/misc_handler.go` — Tambah `CleanONUConfig` handler; import `zte`
- `internal/api/router.go` — Tambah route `POST /:id/onus/:onuId/clean-config` dan `DELETE /:id/onus/:onuId/delete`
- `internal/olt/poller/poller.go` — Tambah `TxPower` ke base struct dan upsert `DoUpdates`
- `src/app/admin/olt/[id]/page.tsx` — State `cleaningConfigOnu`; handler `handleCleanConfigOnu`; tombol "Clean" di action column; refresh 15s; import `Eraser`

## [2.52.86] — 2026-05-31
### Fixed
- **Fix adminStatus PON port selalu "Enabled"** — Parser `parsePONInterfaceStat` di `olt_pon_stat.go` menggunakan `strings.Contains(lower, "activate")` yang juga match kata "**de**activate" → semua port disabled terbaca sebagai enabled. Fix: cek "deactivate" lebih dulu sebelum "activate".
- **Fix ONU name/description terhapus saat poll** — Upsert poller menggunakan `AssignmentColumns` yang selalu update kolom `description` dengan nilai baru (termasuk NULL jika OLT tidak mengembalikan deskripsi). Akibatnya nama ONU yang sudah di-set manual terhapus saat poll berikutnya. Fix: gunakan `COALESCE(VALUES(description), description)` agar nilai lama dipertahankan jika nilai baru NULL. Berlaku juga untuk `serialNumber`.
### Files
- `internal/api/handlers/olt_pon_stat.go` — Fix urutan cek `deactivate` vs `activate` di `parsePONInterfaceStat`
- `internal/olt/poller/poller.go` — Ganti `AssignmentColumns` ke `clause.Assignments` dengan `COALESCE` untuk `description` dan `serialNumber`

## [2.52.85] — 2026-05-31
### Added
- **Enable/Disable PON Port dari Rack Diagram** — Klik titik PON port di diagram rack ZTE C320 membuka modal PON port. Modal menampilkan status admin (Enabled/Disabled), link proto (UP/DOWN), statistik ONU (Total/Online/Offline), suhu dan TX power optik. Tombol **Disable Port** mengirim `shutdown` ke interface `gpon-olt_1/{slot}/{port}` via Telnet; tombol **Enable Port** mengirim `no shutdown`. Tidak perlu login CLI lagi.
### Files
- `src/app/admin/olt/[id]/page.tsx` — PON dot `<div>` → `<button>` klikable; state `selectedPON`; modal render `PONPortModal`; komponen `PONPortModal`

## [2.52.84] — 2026-05-31
### Added
- **Assign ODP ke ONU dari tabel ONU list** — Klik kolom ODP (menampilkan "— assign" jika belum ada) di tabel ONU list membuka modal pilih ODP. Modal menampilkan daftar semua ODP (`GET /api/network/odps`) dengan pencarian, preview ODP aktif, dan opsi hapus link. Simpan via `PATCH /api/olt/:id/onus/:onuId` dengan body `{odpId: "..."}` atau `{clearOdp: true}`. Setelah simpan, tabel live-refresh otomatis.
- **Field `odpId` pada `olt_onu_status`** — Tambah kolom `odpId VARCHAR(191)` ke tabel `olt_onu_status` untuk link langsung per-ONU ke ODP (sebelumnya join by port — tidak akurat karena 1 port bisa punya banyak ODP).
- **Perbaikan SQL JOIN `ListONUs`** — JOIN dari `ponPort`-based (tidak akurat) diganti ke direct `o.odpId = odp.id` sehingga setiap ONU menampilkan ODP yang tepat.
### Files
- `internal/db/models/olt.go` — `OLTONUStatus`: tambah `OdpID *string \`gorm:"index;column:odpId"\``
- `internal/api/handlers/olt.go` — `ListONUs`: SQL JOIN ke `network_odps` via `o.odpId`; `UpdateONU`: tambah handling `odpId` + `clearOdp`
- `src/app/admin/olt/[id]/page.tsx` — Kolom ODP kini clickable → `ONUOdpAssignModal`; state `assigningOdpToOnu`; komponen `ONUOdpAssignModal`

## [2.52.83] — 2026-05-25
### Added
- **Rx power degradation alerts (WA + Telegram)** — Poller kini mendeteksi sinyal ONU lemah (Rx < -27 dBm) dan membuat alert tipe `rx_degradation`. Saat Rx membaik (>= -27 dBm), alert otomatis di-resolve. Juga mendeteksi degradasi massal: jika ≥ 3 atau ≥ 50% ONU online pada satu PON port memiliki sinyal lemah, dibuat alert tipe `bulk_rx_degrade` (severity: critical) dengan notifikasi WA + Telegram.
- **Real-time auto-refresh tabel ONU** — List ONU di tab detail OLT kini refresh otomatis setiap 30 detik via `GET /api/olt/:id/onus?all=true`. Ditampilkan indikator "Updated Xs ago" di header tabel. Data awal langsung dimuat saat halaman dibuka.
- **Kolom ODP di tabel ONU** — Kolom baru "ODP" di tabel list ONU, diisi dari join `network_odps` berdasarkan `ponPort = onu.port` (query SQL di `ListONUs`). Menampilkan nama ODP yang terhubung ke port tersebut.
- **Fix PON port enable/disable (stale closure)** — Tombol Enable/Disable PON port kini bekerja dengan benar untuk port yang sedang enabled maupun disabled. Fix stale closure di `handlePONAction`: gunakan `useRef` (`fetchPONStatRef`) agar `fetchPONStat` dengan cache terbaru selalu dipanggil setelah aksi.
### Files
- `internal/db/models/olt.go` — Tambah `AlertRxDegradation` + `AlertBulkRxDegrade` ke `OltAlertType`
- `internal/api/handlers/olt.go` — `ListONUs` ditulis ulang: raw SQL dengan LEFT JOIN ke `network_odps` + support `?all=true` untuk polling tanpa pagination; fix `const baseSQL` → `var baseSQL`
- `internal/olt/poller/poller.go` — `checkAlerts`: tambah Rx power degradation check (single ONU + bulk per PON port) dengan create/resolve alert + WA+Telegram notifikasi
- `src/app/admin/olt/[id]/page.tsx` — `liveOnus` state + `fetchLiveOnus` (30s interval); ODP column di tabel ONU; refresh indicator; fix PON stale closure via `fetchPONStatRef`

## [2.52.82] — 2026-05-24
### Fixed
- **Sinkronisasi alert OLT detail vs halaman global** — Badge "Alerts" di tab detail OLT menghitung semua alert (termasuk yang sudah resolved) karena `Preload("Alerts")` tanpa filter. Kini backend hanya preload alert `isResolved = false`, dan frontend juga memfilter `!a.isResolved` sehingga badge dan konten tab hanya menampilkan alert aktif — konsisten dengan halaman global OLT Alerts.
### Files
- `internal/api/handlers/olt.go` — `GetOLT`: tambah filter `isResolved = false` + `Limit(50)` pada `Preload("Alerts")`
- `src/app/admin/olt/[id]/page.tsx` — Badge tab "Alerts" dan konten tab kini filter `!a.isResolved`

## [2.52.81] — 2026-05-23
### Added
- **PON port enable/disable & description edit** — Tambahkan handler `POST /api/olt/:id/pon` dengan action `enable`, `disable`, `setDescription`. Frontend: di panel detail port PON (expanded card), muncul tombol Enable/Disable dan "Edit Desc" dengan inline input.
- **ONU name/description edit** — Tambahkan handler `PATCH /api/olt/:id/onus/:onuId` untuk update `description` di DB dan perintah `name` + `description` via Telnet ke OLT. Frontend: tombol **Edit** di list ONU membuka modal `ONUEditModal`.
- **WA + Telegram alert saat ONU offline** — Poller `checkAlerts` kini mengirim notifikasi WhatsApp (ke `Company.AdminPhone`) dan Telegram (ke `TelegramBackupSettings`) saat ONU baru go offline. Set `notifiedViaWhatsapp = true` pada record alert.
- **Auto-resolve alert + notifikasi recovery** — Saat ONU yang sebelumnya offline kembali online, alert `onu_offline` di-resolve (`isResolved = true`, `resolvedAt = now`) dan dikirim pesan 🟢 recovery via WA + Telegram.
- **Fix alert deduplication bug** — `checkAlerts` sebelumnya menggunakan `s.ID` (UUID baru tiap poll) alih-alih DB ID ONU nyata, sehingga duplikat alert dibuat setiap siklus poll. Fix: fetch real DB IDs by (frame, slot, port, onuId) terlebih dahulu.
- **`internal/notify/telegram.go`** — Fungsi paket-level `notify.SendTelegramMessage(botToken, chatId, text)` yang dapat digunakan oleh poller (di luar package `handlers`).
### Files
- `internal/notify/telegram.go` — New: `SendTelegramMessage` helper
- `internal/api/handlers/olt.go` — Add `PONPortAction` + `UpdateONU` handlers
- `internal/api/router.go` — Register `POST /:id/pon` + `PATCH /:id/onus/:onuId`
- `internal/olt/poller/poller.go` — Fix `checkAlerts` deduplication; add recovery detection; add WA+Telegram notifications via `notifyAlert()`
- `src/app/admin/olt/[id]/page.tsx` — PON port action buttons + inline desc editor; ONU Edit button + `ONUEditModal`

---

## [2.52.80] — 2026-05-22
### Fixed
- **Assign Customer 405 Method Not Allowed** — Frontend memanggil `GET /api/olt/:id/onus/:onuId/assign` untuk memuat daftar pelanggan di assign modal, namun backend hanya mendaftarkan `POST` untuk route tersebut. Fix: tambahkan `GET` handler `GetAssignONUCandidates` yang mengembalikan daftar `PppoeUser` (bisa dicari via `?q=`) dan customer yang sedang di-assign ke ONU.
- **Unassign ONU tidak berfungsi** — `AssignONU` POST handler menggunakan `CustomerID string` sehingga nilai `null` dari frontend di-decode sebagai string kosong, lalu ditolak dengan error "customerId required". Fix: ubah ke `*string` agar `null` diterima dan GORM meng-update kolom ke NULL (unassign).
### Files
- `internal/api/handlers/olt.go` — Add `GetAssignONUCandidates` GET handler; fix `AssignONU` to accept null customerId for unassign
- `internal/api/router.go` — Register `GET /:id/onus/:onuId/assign` route

---

## [2.52.79] — 2026-05-22
### Fixed
- **Reboot ONU tidak berfungsi** — `RebootONU` handler sebelumnya hanya stub (return success tanpa melakukan apa-apa). Perintah `reset gpon-onu_F/S/P:N` tidak valid di ZTE C320 V2.1 (error 20200/20204). Fix: implementasi nyata via Telnet dengan `shutdown` + `no shutdown` pada interface ONU, yang memaksa ONU offline dan re-registrasi (terbukti dari log: `Online Duration: 0h 00m 02s` setelah reboot).
- **Batch Reboot ONU tidak berfungsi** — `BatchRebootONUs` handler juga stub. Fix: implementasi nyata — query ONUs dari DB, build commands `configure terminal` → `interface gpon-onu_F/S/P:N` → `shutdown` → `no shutdown` → `exit` untuk setiap ONU, kemudian `end`.
- **ONU Detail modal banyak field N/A** — `onuParseDetailInfo` menggunakan alias field yang salah (`SN` alih-alih `Serial number`, `Match mode` alih-alih `Authentication mode`, `Control flag` alih-alih `Admin state`, dll) dan skip nilai kosong (`if val == "" { continue }`) sehingga `OMCI BW Profile` selalu ter-skip. Fix: hapus alias mapping (frontend menggunakan nama field ZTE C320 asli langsung), izinkan empty values, dan perbaiki summary map dengan nama field yang benar (`Authentication mode`, `SN Bind`, `Admin state`, `Current channel`, `Configured channel`, `DBA Mode`, `Vport mode`, `Line Profile`, `Service Profile`, `OMCI BW Profile`, `Serial number`).
### Files
- `internal/api/handlers/misc_handler.go` — Implement `RebootONU` via shutdown/no-shutdown; implement `BatchRebootONUs`; fix `onuParseDetailInfo` aliases and summary fields

---

## [2.52.78] — 2026-05-22
### Fixed
- **Remove VLAN 500 error** — Perintah `no switchport vlan X tag` tidak valid di ZTE C320 (mengembalikan `%Error 20201: Invalid command key word`). Perintah yang benar adalah `no switchport vlan X` (tanpa suffix `tag`). Sebelumnya juga ada dua commandSet — fallback kedua (`no switchport default vlan`) salah karena menghapus PVID bukan tagged VLAN. Kini hanya satu commandSet dengan perintah yang benar.
- **API response 500 → 422 saat CLI error** — Ketika perintah Telnet ditolak OLT (CLI error seperti `%Error`), handler mengembalikan HTTP 500 "Uplink action failed" padahal ini bukan server error. Fix: kembalikan HTTP 422 Unprocessable Entity dengan detail pesan error dari OLT, sehingga frontend dapat menampilkan pesan yang tepat.
### Changed
- **Toggle Enable/Disable port** — Dua tombol terpisah (Enable + Disable) di status tab uplink diganti dengan satu tombol kontekstual: jika port sedang enabled tampil tombol "Disable Port" (merah), jika port disabled tampil tombol "Enable Port" (hijau).
### Files
- `internal/api/handlers/olt.go` — Fix `removeVlan` command dari `no switchport vlan X tag` → `no switchport vlan X`; remove wrong fallback commandSet; return 422 instead of 500 on CLI errors
- `src/app/admin/olt/[id]/page.tsx` — Replace Enable+Disable buttons with single contextual toggle button

---

## [2.52.77] — 2026-05-22
### Fixed
- **Uplink status "Unknown"** — `uplinkParsePortStatus` membaca kolom index yang salah dari output `show interface port-status xgei_1/3/2`. ZTE C320 mengembalikan 9 field (index 0–8): port, hybridStatus, nativeVlan, negotiation, speed, duplex, flowCtrl, adminStatus, linkStatus. Kode lama salah baca duplex dari `parts[3]` (seharusnya `parts[5]`), flowCtrl dari `parts[8]` (seharusnya `parts[6]`), adminStatus dari `parts[9]` (OOB), linkStatus dari `parts[10]` (OOB). Akibatnya `adminRaw` selalu kosong → status "Unknown".
- **Uplink status "Unknown" (fallback)** — `uplinkParseInterfaceStatus` memiliki regex `stateRe` yang hanya menangani `activate|deactivate`. Namun ZTE C320 untuk interface uplink (`xgei`) melaporkan `xgei_1/3/2 is up, line protocol is up` bukan `activate`. Fix: tambahkan `up|down` ke pattern. Update pemetaan: `up` → "Up", `down` → "Down".
- **CONFIG tab menampilkan `%Error 20202`** — Handler `case "config"` menetapkan `raw = out` sebelum memeriksa CLI error, sehingga teks error mentah tampil di UI. Fix: hanya set `raw = out` ketika output tidak mengandung CLI error.
- **Dark/light theme uplink modal** — Status indicator menggunakan warna hex hardcoded (`#111827`, `#052e16`, `#451a03`) via inline style tanpa dukungan light mode. Diganti dengan Tailwind classes `dark:` variants (`bg-gray-100 dark:bg-gray-900`, `bg-green-50 dark:bg-green-950/40`, dll).
- **PON expansion bg invalid** — `dark:bg-gray-850` bukan class Tailwind valid (Tailwind hanya memiliki 100–900). Diganti ke `dark:bg-gray-900`.
### Files
- `internal/api/handlers/olt.go` — Fix `uplinkParsePortStatus` column indices; fix `uplinkParseInterfaceStatus` stateRe + admin mapping; fix `case "config"` raw error handling
- `src/app/admin/olt/[id]/page.tsx` — Fix `statusTone` dari inline styles ke Tailwind dark: classes; fix `dark:bg-gray-850` → `dark:bg-gray-900`

---

## [2.52.76] — 2026-05-22
### Fixed
- **Distance via Telnet** — SNMP OID `.18` (equalization delay × 0.112) memberikan nilai yang tidak akurat untuk sebagian ONU (contoh: ONU 1/1/1:12 menampilkan "101.07 dBm" karena nilai raw OID terpetakan ke RxPower ONU lain akibat bug index). Fix: ambil jarak dari Telnet `show gpon onu detail-info gpon-onu_1/{slot}/{port}:{onuId}` yang langsung melaporkan `ONU Distance: Xm` (diukur dari proses ranging OLT). Semua ONU terdaftar di-query dalam satu sesi Telnet via `ExecuteMultiple`. Hasil Telnet override nilai SNMP; jika Telnet gagal, nilai SNMP tetap dipakai sebagai fallback.
- **Frontend tidak update** — Next.js berjalan via PM2 (`salfanet-radius`), bukan `systemctl salfanet-api`. Deploy sebelumnya hanya restart Go binary tanpa restart PM2, sehingga UI lama masih terbuffer. Fix: `rm -rf .next && npm run build && pm2 restart salfanet-radius`.
### Files
- `internal/olt/vendors/zte/zte.go` — Tambah `FetchTelnetDistances(pool, onus)` dan `parseTelnetDistances(raw)` untuk parsing `ONU Distance: Xm` dari combined Telnet output
- `internal/olt/poller/poller.go` — Di `poll()`: retrieve pool dari `p.pools[olt.ID]`; panggil `zte.FetchTelnetDistances` setelah SNMP discovery; override `onu.Distance` untuk setiap ONU yang mendapat data Telnet

---

## [2.52.75] — 2026-05-22
### Added
- **Per-PON Live Stats** — Setiap port PON di section "Detail Per Port PON" kini bisa di-klik untuk expand dan menampilkan data live dari Telnet OLT: Temperature (°C), TX Power (dBm), Voltage (V), Bias Current (mA), Upstream/Downstream rate (Mbps) dan bandwidth usage (%). Data diambil via `show interface gpon-olt_1/{slot}/{port}` dan `show interface optical-module-info gpon-olt_1/{slot}/{port}`.
### Fixed
- **Chassis port squares** — Port PON menunjukkan index 0 (0/1/0) pada tooltip dan green box di posisi ke-2; seharusnya 1-based. Fix: rebuild frontend (source sudah benar dengan `i+1`, VPS belum di-build ulang).
### Files
- `internal/api/handlers/olt_pon_stat.go` — Handler baru `GetPONStat`; parser `parsePONInterfaceStat`, `parsePONBps`, `parsePONPct`, `parsePONFloatFromUnit`
- `internal/api/router.go` — Route baru `GET /api/olt/:id/pon-stat`
- `src/app/admin/olt/[id]/page.tsx` — Tambah type `PONPortStat`; state `expandedPON`, `ponStatCache`, `loadingPON`; fungsi `fetchPONStat`; "Detail Per Port PON" expandable cards dengan Temperature, TX Power, Voltage, Upstream/Downstream

---

## [2.52.74] — 2026-05-22
### Fixed
- **Distance ONU semua NULL** — OID `.3.50.12.1.1.21` adalah nilai konstan per-port (bukan per-ONU), sehingga tidak ter-mapping ke masing-masing ONU dan distance tetap NULL di DB. Fix: ganti ke OID `.3.50.12.1.1.18` (equalization delay per-ONU, terindex per onuId) dengan formula `raw × 0.112` (diverifikasi: ONU 28 OID18=5000 → 5000×0.112=560m sesuai data Telnet).
### Files
- `internal/olt/vendors/zte/zte.go` — `oidDistance` dari OID `.21` ke OID `.18`; formula `int(dist)` → `int(float64(dist) * 0.112)` (2 tempat)

---

## [2.52.73] — 2026-05-21
### Fixed
- **Distance ONU salah di ONU List** — SNMP OID `.3.50.12.1.1.19` (equalization delay / bukan jarak fiber) menghasilkan nilai jarak yang salah (contoh: 1228m padahal actual 560m). Fix: ganti ke OID `.3.50.12.1.1.21` (direct fiber distance dalam meter) dan hapus konversi `raw/10` sehingga nilai dipakai langsung.
- **Counter DyingGasp tidak muncul di OLT Detail** — Filter status menggunakan `'dyingGasp'` (camelCase) padahal API mengembalikan `'dying_gasp'`. Fix: ganti ke `'dying_gasp'` agar counter DyingGasp tampil di header OLT detail page.
### Files
- `internal/olt/vendors/zte/zte.go` — `oidDistance` diubah ke OID `.3.50.12.1.1.21`; konversi `raw/10.0` diganti `int(dist)` langsung (2 tempat)
- `src/app/admin/olt/[id]/page.tsx` — filter `o.status === 'dyingGasp'` diubah ke `'dying_gasp'`

---

## [2.52.72] — 2026-05-21
### Fixed
- **Port 1 tidak menyala di ZTE C320 Rack Diagram** — `buildServicePorts` menyertakan port 0 (dummy) sebagai elemen pertama array (`ports[0] = {Port:0, HasOnus:false}`), sehingga offset index menggeser tampilan port. Fix: array ports kini dimulai dari port 1 (1-based), `ports[i] = {Port: i+1}`, dan lookup DB menggunakan `arrayIdx = portIdx - 1`. Juga perbaiki `portCount`: tidak lagi menggunakan `stdPorts + 1` melainkan `stdPorts` (atau `card.PortCount` jika lebih besar, misal 16 untuk GTGHG dari Telnet).
### Files
- `internal/api/handlers/olt_chassis.go` — `buildServicePorts`: 1-based port array; Telnet & SNMP service case: `portCount = stdPorts` (bukan `stdPorts+1`), gunakan `card.PortCount` jika lebih besar

---

## [2.52.71] — 2026-05-21
### Fixed
- **Total ONU = 0 di halaman OLT Management** — `GET /api/network/olts` diarahkan ke handler `ListOLTsForMap` yang mengembalikan data OLT polos tanpa ONU stats. Fix: ubah route ke `ListOLTs` (handler yang mengembalikan `_count.olt_onu_status` dan `onu_stats`). Tambah `Preload("Routers.Router")` ke `ListOLTs` agar kolom ROUTER/NAS tetap muncul.
### Files
- `internal/api/router.go` — `network.Get("/olts")` → `networkH.ListOLTs` (bukan `ListOLTsForMap`)
- `internal/api/handlers/network_ext.go` — tambah `Preload("Routers.Router")` ke `ListOLTs`

---

## [2.52.70] — 2026-05-21
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

---

## [2.52.69] — 2026-05-21
### Fixed
- **Password OLT tidak tampil di list** — `Password *string json:"-"` di model mencegah password dikembalikan di API (by design untuk keamanan). Tambah field `hasPassword bool` di response `ListOLTs` yang bernilai `true` jika password sudah tersimpan. Frontend kini menampilkan `••••••••` jika `hasPassword=true`, `-` jika belum diset.
- **Stats card teks terlalu mepet ke border** — Padding kiri kartu stats (Status/Uptime/ONUs) diperbesar dari `p-4` ke `pl-6 pr-4 py-4` agar teks tidak terlalu dekat dengan garis `border-l-4`.
### Files
- `internal/api/handlers/network_ext.go` — tambah `HasPassword bool json:"hasPassword"` ke `oltWithStats`, populate dari `o.Password != nil && *o.Password != ""`
- `src/app/admin/network/olts/page.tsx` — tambah `hasPassword?: boolean` ke interface OLT; gunakan `olt.hasPassword` untuk display password di tabel dan detail panel
- `src/app/admin/olt/[id]/page.tsx` — ubah padding stats card dari `p-4` ke `pl-6 pr-4 py-4`

---

## [2.52.68] — 2026-05-21
### Fixed
- **ONU Detail Modal kosong** — ZTE C320 menggunakan `--More--` paging untuk output panjang (show gpon onu detail-info). `readUntilPrompt` tidak pernah melihat prompt `#` karena pager menginterupsi output → timeout 10 detik → output kosong. Fix: kirim `terminal length 0` setelah login berhasil untuk menonaktifkan paging di seluruh sesi telnet.
- **Jarak ONU semua sama 328m** — OID `.21` (zxAnGponOnuOptDistance) mengembalikan nilai fixed 328 untuk semua ONU (equalization delay provisioning, bukan jarak sebenarnya). Ganti ke OID `.19` (zxAnPonAniOptRtt) yang mengembalikan RTT aktual dalam nanoseconds per ONU. Konversi: `jarak_m = RTT_ns / 10` (sesuai kecepatan cahaya di fiber ≈ 2×10⁸ m/s). Contoh: ONU1=11627 ns → 1163m, ONU9=15367 ns → 1537m.
- **Uplink SMXA card status** — Setelah fix telnet paging, `show interface port-status` sekarang mengembalikan output lengkap → status active/disable port uplink SMXA ditampilkan dengan benar.
- **Stats card ONU list** — Perbaiki padding kartu (`p-4` konsisten), ukuran font lebih jelas (`text-2xl`), label uppercase tracking, dan breakdown status ONU (offline, LOS, DyingGasp) ditampilkan lebih detail.
### Files
- `internal/olt/telnet/telnet.go` — tambah `terminal length 0` setelah login berhasil untuk disable paging
- `internal/olt/vendors/zte/zte.go` — ubah `oidDistance` dari `.21` ke `.19`; konversi RTT → jarak `int(float64(dist)/10.0)`; filter batas atas dari 100000 ke 1000000
- `src/app/admin/olt/[id]/page.tsx` — stats cards: `p-4` konsisten, `text-2xl`, label uppercase, grid `grid-cols-3`, breakdown LOS/DyingGasp

---

## [2.52.67] — 2026-05-21
### Fixed
- **RxPower formula salah (lagi)** — Formula `10*log10(raw)-60` (nW asumsi) masih salah. Setelah menganalisis raw SNMP dari live device vs CSV referensi, ditemukan rumus yang benar: **`dBm = raw/500.0 - 30.0`**. Verifikasi: raw=6751 → -16.50 dBm ✓, raw=5085 → -19.83 dBm ✓, raw=5910 → -18.18 dBm ✓ (cocok dengan CSV). ZTE C320 menggunakan encoding linear: 0 raw = -30 dBm, step 0.002 dBm per unit.
- **ONU Name tidak muncul untuk FiberHome ONUs** — ONU FiberHome (prefix FHTTC/FHTT) mengembalikan `regStatus=2` dari SNMP, tapi kode hanya memproses `regStatus=1`. ONU regStatus=2 masuk ke loop fallback yang tidak membaca `description` dan masih menggunakan formula lama. Fix: terima regStatus=1 dan 2 sebagai "registered", baca description di kedua loop, gunakan formula yang benar di kedua loop.
- **Signal quality threshold** — Dikalibrasi ulang sesuai nilai aktual dari live device: rentang -10 s/d -20 dBm. Threshold baru: ≥ -14 Excellent, ≥ -18 Good, ≥ -22 Fair, < -22 Poor.
### Files
- `internal/olt/vendors/zte/zte.go` — formula rxPower & txPower → `raw/500.0 - 30.0`; filter regStatus menerima 1 dan 2; loop fallback tambah description + formula benar; hapus import `math`
- `src/app/admin/olt/[id]/page.tsx` — signal thresholds dikalibrasi

---

## [2.52.66] — 2026-05-21
### Fixed
- **RxPower formula salah** — ZTE C320 SNMP OID `.3.50.12.1.1.10` mengembalikan nilai integer dalam satuan **nanowatt (nW)**, bukan milli-dBm. Formula lama `dBm = -raw/1000` menghasilkan nilai seperti -7 dBm (salah). Formula benar: `dBm = 10 × log10(raw) - 60`. Contoh: raw=7540 nW → -21.2 dBm ✓. Nilai rxPower lama di DB di-reset agar sync berikutnya menulis nilai yang benar.
- **ONU count tidak muncul di OLT Management list** — `ListOLTs` hanya mengembalikan data OLT mentah tanpa jumlah ONU. Frontend mengakses `olt._count.olt_onu_status` dan `olt.onu_stats` yang selalu `undefined`, sehingga tampil "0 ONU". Handler sekarang menjalankan satu query GROUP BY untuk mendapatkan jumlah ONU per OLT per status, lalu menyertakan `_count` dan `onu_stats` di setiap respons OLT.
- **Signal quality threshold** — Dikalibrasi ulang untuk ONU downstream RX power (nW formula baru): ≥ -20 dBm Excellent, ≥ -24 Good, ≥ -27 Fair, < -27 Poor.
### Files
- `internal/olt/vendors/zte/zte.go` — rxPower dan txPower: formula diubah ke `10*math.Log10(raw) - 60`; import `math` ditambahkan; komentar OID diupdate
- `internal/api/handlers/olt.go` — `ListOLTs`: tambah GROUP BY query untuk `onu_stats` dan `_count`; tambah `Preload("Routers.Router")`
- `src/app/admin/olt/[id]/page.tsx` — signal quality thresholds untuk downstream ONU power

---

## [2.52.65] — 2026-05-25
### Fixed
- **Sync OLT — data tidak refresh setelah sync** — backend `SyncOLT` hanya mengembalikan `{"message":"sync triggered"}` tanpa flag `background`. Frontend langsung memanggil `fetchOLT()` sebelum sync selesai sehingga data tidak berubah. Sekarang response berisi `{"background": true, ...}` sehingga frontend menunggu 30 detik lalu refresh otomatis.
- **Port map — index 0-based vs 1-based** — port map merender port `i=0,1,2,...` tapi data di `portStats` diindeks `1,2,3,...` (1-based dari DB). Port 0 selalu kosong, semua ONU tergeser satu posisi. Fix: gunakan `i+1` pada `portColor`, `portStats`, `portTooltip`, dan `key` di render loop.
- **OID prefix bleed di BulkWalk** — `walkPONPort` tidak memvalidasi bahwa OID yang dikembalikan BulkWalk memang berasal dari subtree yang di-walk. Jika `MaxRepetitions=25` melebihi batas tabel, OID dari PON port berikutnya masuk ke data port saat ini, menyebabkan nilai `distance`/`rxPower` yang salah (misal 1328, 2328 di DB). Sekarang setiap result dicek prefix `strings.HasPrefix(oidNorm, baseOID+".")`.
- **Signal quality threshold salah** — threshold dikalibrasi untuk ONU downstream RX power (-20 s/d -27 dBm), tapi SNMP OID `.3.50.12.1.1.10` menyimpan OLT upstream received power (~-5 s/d -15 dBm setelah power leveling). Semua ONU tampil "Excellent" karena semua nilai > -20 dBm. Threshold diubah ke: ≥ -10 Excellent, ≥ -15 Good, ≥ -20 Fair, < -20 Poor.
### Performance
- **walkToMap menggunakan BulkWalk** — `GetUplink` memanggil `fetchIfMib` yang menjalankan 5 parallel walk untuk IF-MIB. Sebelumnya menggunakan `snmputil.Walk` (GetNext PDUs, lambat). Diganti ke `snmputil.BulkWalk` (GetBulk PDUs) yang 3-5× lebih cepat untuk tabel besar.
- **ONU detail & uplink Telnet pool reuse** — `ONUDetail` dan `GetUplink` sebelumnya membuat `telnet.NewPool` baru setiap request dan langsung menutupnya via `defer pool.Close()`. Ini menyebabkan Telnet login ulang (3-5 detik) setiap kali modal ONU detail atau uplink tab dibuka. Sekarang keduanya memakai pool persistent milik Poller via `h.poller.GetPool(oltID)`. Pool sementara hanya dibuat jika Poller belum mengelola OLT tersebut.
- **Poller — `GetPool` method** — tambah method `GetPool(oltID string) *telnet.Pool` pada `Poller` agar handler dapat mengakses persistent pool tanpa race condition.
- **MiscHandler — tambah Poller** — `MiscHandler.poller` ditambahkan agar `ONUDetail` bisa akses Poller pool. `NewMiscHandler` sekarang menerima `*poller.Poller`.
### Files
- `internal/api/handlers/olt.go` — `SyncOLT`: return `background:true`; `GetUplink`: reuse poller pool
- `internal/api/handlers/olt_chassis.go` — `walkToMap`: `snmputil.Walk` → `snmputil.BulkWalk`
- `internal/api/handlers/misc_handler.go` — tambah `poller` ke struct; `ONUDetail`: reuse pool, fix context bug
- `internal/api/router.go` — pass poller ke `NewMiscHandler`
- `internal/olt/poller/poller.go` — tambah `GetPool(oltID)` method
- `internal/olt/vendors/zte/zte.go` — `walkOut` tambah `baseOID`; tambah OID prefix filter per result
- `src/app/admin/olt/[id]/page.tsx` — port map `i` → `i+1`; signal quality thresholds; tambah komentar

---

## [2.52.64] — 2026-05-24
### Fixed
- **ONU detail modal crash (TypeError)** — `ONUDetailModal` rendered `detail.telnet.detail.raw` and `detail.telnet.config.raw` without optional chaining. When the API returned a mismatched shape, this caused `TypeError: Cannot read properties of undefined (reading 'detail')` crashing the OLT page. Fixed both lines to use `?.` optional chaining.
- **ONUDetail handler response shape** — replaced stub handler that returned `{ "detail": {...} }` with a real Telnet-based implementation returning `{ "telnet": { "interface", "detail": { "parsed", "summary", "raw" }, "config": { "summary", "raw" }, "optical": { "raw" } }, "onu": { "id", "customer" } }` matching what the frontend expects.
- **ONU discovery — all ONUs collapsed to onuId=1** — critical SNMP OID bug: for `zxAnGponOnuRegTable` and `zxAnGponOnuDiscoveredInfoTable`, the row index suffix is `.<onuId>.<subIdx>` (2 components), but `lastOIDComponent` was returning the trailing `subIdx` (always 1 or a column index) instead of the actual `onuId`. Added `secondToLastOIDComponent` helper and use it for all RegTable and SeenONU walk results. Serial/desc (`zxAnGponOnuCfgTable`) correctly use `lastOIDComponent` (1-component suffix). Verified against live ZTE C320 V2.1 SNMP output.
### Added
- `secondToLastOIDComponent` helper in `zte.go` — returns the second-to-last numeric OID component, used for ZTE RegTable and SeenONU tables.
- `onuParseDetailInfo`, `onuParseRunningConfig`, `onuVendorFromSN`, `onuSNPrefix`, `contains` helpers in `misc_handler.go` — parse ZTE C320 Telnet ONU detail and running-config output.
### Files
- `src/app/admin/olt/[id]/page.tsx` — lines 1567-1568: add `?.` optional chaining on `detail.telnet.detail.raw` and `detail.telnet.config.raw`
- `internal/api/handlers/misc_handler.go` — replace stub `ONUDetail` with real Telnet implementation; add `context` and `telnet` imports; add parse helpers
- `internal/olt/vendors/zte/zte.go` — add `secondToLastOIDComponent`; use it for regStatus, operState, rxPower, txPower, distance, seenONU walks; keep `lastOIDComponent` for serial/desc

---

## [2.52.63] — 2026-05-23
### Fixed
- **Port map uplink AdminStatus/LinkStatus wrong** — `parseUplinkPortStatus` was reading column 7 (Pause) and column 8 (FlowControl) instead of column 9 (AdminStatus) and column 10 (LinkStatus). All uplink ports showed `DIS`/down incorrectly on the chassis port map.
- **Minimum column count check** — updated from `len(parts) < 8` to `len(parts) < 11` so rows with fewer than 11 columns are skipped correctly.
- **Uplink description via SNMP** — when Telnet-parsed uplink states are available, SNMP `ifAlias` descriptions are now merged in, so uplink port hover tooltips show descriptions even when Telnet is the primary data source.
- **GetUplink handler** — replaced stub with real Telnet-based implementation for all four tabs: `status` (parses `show interface port-status` + `show interface` with SNMP fallback), `vlan` (parses `show running-config interface`), `config` (returns raw running-config), `optical` (parses `show interface optical-module-info` + `show ddmi interface`).
- **CreateUplink handler** — replaced stub with real Telnet config-mode implementation: `addVlan`, `removeVlan`, `enable`, `disable`, `setPvid`, `removePvid`, `setDescription`.
### Files
- `internal/api/handlers/olt_chassis.go` — fix `parseUplinkPortStatus` column indices (9, 10); merge SNMP description into Telnet-parsed uplink states
- `internal/api/handlers/olt.go` — implement real `GetUplink` + `CreateUplink` handlers with full parser helpers; add `context`, `regexp`, `strings`, snmputil imports

---

## [2.52.62] — 2026-05-23
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

---

## [2.52.61] — 2026-05-22
### Added
- **GetChassis — real Telnet + SNMP chassis data (ported from Next.js)** — `GET /api/olt/:id/chassis` now fires Telnet (`show card` + `show interface port-status`) and 5 SNMP IF-MIB walks (ifDescr, ifAdminStatus, ifOperStatus, ifHighSpeed, ifAlias) plus ZTE PON table walk **in parallel**. Real card types (GTGO/GTGH/GTGQ/SMXA/MCUD) come from Telnet `show card`; uplink port states (admin/link status, speed, description) come from `show interface port-status` with SNMP IF-MIB fallback. Falls back to SNMP board presence + DB ONU port data when Telnet is unavailable. Response includes `source: "telnet" | "snmp+db"`.
### Files
- `internal/api/handlers/olt_chassis.go` — new file: full chassis handler with `parseShowCard`, `classifyCard`, `smxaUplinkIfaces`, `parseUplinkPortStatus`, `buildUplinkStatesFromSNMP`, SNMP IF-MIB walk helpers, and `GetChassis`
- `internal/api/handlers/olt.go` — remove old inline `GetChassis` implementation (now delegated to `olt_chassis.go`)

---

## [2.52.60] — 2026-05-22
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

---

## [2.52.59] — 2026-05-21
### Fixed
- **OLT detail page crash — TypeError: Cannot read properties of undefined (reading 'length')** — GORM JSON omits empty arrays (due to `omitempty`) so `onuStatuses`, `alerts`, `routers`, `monitoringLogs`, `performanceMetrics` are `undefined` on the frontend when empty. Added `?? []` normalization after `data.olt` is received in `fetchOLT`.
### Files
- `src/app/admin/olt/[id]/page.tsx` — normalize all relation arrays to `[]` after fetch

---

## [2.52.58] — 2026-05-21
### Fixed
- **GET /api/olt/:id — 404** — Root cause: `GetOLT` handler uses `Preload("MonitoringLogs", Order("created_at DESC"))` dan `Preload("PerformanceMetrics", Order("recorded_at DESC"))`. DB columns are camelCase (`createdAt`, `recordedAt`), bukan snake_case. MySQL error "Unknown column 'created_at'" membuat seluruh query gagal → handler return 404.
- **`NetworkOLT` model — tidak ada column tags** — Semua field compound (`ipAddress`, `snmpEnabled`, `isOnline`, `totalOnu`, dst.) dipetakan GORM sebagai snake_case (`ip_address`, `snmp_enabled`, `is_online`, `total_onu`, dst.) tapi DB pakai camelCase. CREATE/UPDATE/SELECT semua salah.
- **`olt_ext.go` Monitoring SELECT — snake_case columns** — `Select("id,name,ip_address,status,is_online,total_onu,online_onu,offline_onu")` → MySQL error "Unknown column". Diganti ke `"id,name,ipAddress,status,isOnline,totalOnu,onlineOnu,offlineOnu"`.
### Files
- `internal/db/models/olt.go` — tambah `gorm:"column:..."` camelCase ke semua field `NetworkOLT`: `ipAddress`, `followRoad`, `firmwareVersion`, `snmpEnabled`, `snmpCommunity`, `snmpPort`, `telnetEnabled`, `telnetPort`, `sshEnabled`, `sshPort`, `monitoringEnabled`, `pollingInterval`, `lastPollAt`, `isOnline`, `totalOnu`, `onlineOnu`, `offlineOnu`, `createdAt`, `updatedAt`
- `internal/api/handlers/olt.go` — fix `GetOLT` preload order: `created_at` → `createdAt`, `recorded_at` → `recordedAt`
- `internal/api/handlers/olt_ext.go` — fix `Monitoring` SELECT ke camelCase column names

## [2.52.57] — 2026-05-22
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

## [2.52.56] — 2026-05-22
### Fixed
- **GET /api/olt/:id — 404 meskipun route terdaftar** — `GetOLT` handler melakukan preload `ONUStatuses`, `Alerts`, `MonitoringLogs`, dan `PerformanceMetrics` dengan GORM auto-naming. GORM mengubah `OltID` → `olt_id`, `OnuID` → `onu_id`, dst., padahal kolom DB (Prisma) adalah camelCase (`oltId`, `onuId`). MySQL melempar error "Unknown column 'olt_id'" yang diperlakukan sebagai 404. Fix: tambah explicit `gorm:"column:..."` camelCase tags ke semua field di `OLTONUStatus`, `OLTAlert`, `OLTPerformanceMetric`, `OLTMonitoringLog`.
- **ONU data selalu 0/0 di monitoring** — Poller `CreateInBatches` OLT ONU Status gagal karena INSERT menggunakan nama kolom snake_case (`olt_id`, `onu_id`, `mac_address`, dst.) yang tidak ada di DB. Dengan adanya column tags, INSERT sekarang menggunakan nama kolom yang benar (`oltId`, `onuId`, `macAddress`, dst.).
- **Poller `knownPONPorts` — WHERE clause salah** — `Where("olt_id = ?", oltID)` diganti ke `Where("oltId = ?", oltID)`.
- **Poller `checkAlerts` — WHERE clause salah** — raw SQL dengan `olt_id`, `onu_id`, `alert_type`, `is_resolved` diganti ke camelCase `oltId`, `onuId`, `alertType`, `isResolved`.
### Files
- `internal/db/models/olt.go` — `OLTONUStatus`, `OLTAlert`, `OLTPerformanceMetric`, `OLTMonitoringLog`: tambah `gorm:"column:camelCase"` tags ke semua field
- `internal/olt/poller/poller.go` — fix `knownPONPorts` dan `checkAlerts` WHERE clause ke camelCase

## [2.52.55] — 2026-05-22
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

## [2.52.54] — 2026-05-22
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

---

## [2.52.53] — 2026-05-22
### Fixed
- **OLT Management page — list selalu kosong** — `GET /api/network/olts` memanggil `ListOLTsForMap` yang mengembalikan array mentah, sedangkan frontend mengharapkan `{olts: [...]}`. Fix: ubah `ListOLTsForMap` mengembalikan `fiber.Map{"olts": olts}`.
- **OLT Detail page — crash `Cannot read properties of undefined (reading 'vendor')`** — `GET /api/olt/:id` mengembalikan model mentah, sedangkan `fetchOLT` di frontend mengharapkan `{olt: ...}` (wrapped). Akibatnya `data.olt` undefined → akses `o.vendor` crash. Fix: ubah `GetOLT` mengembalikan `fiber.Map{"olt": olt}` dan tambah preload `Alerts`.
### Files
- `internal/api/handlers/network.go` — `ListOLTsForMap`: return `{olts: olts}` bukan array mentah
- `internal/api/handlers/olt.go` — `GetOLT`: return `{olt: olt}`, tambah `Preload("Alerts")`

---

## [2.52.52] — 2026-05-21
### Fixed
- **L2TP delete VPN client — tidak membersihkan peer-routes.conf dan kernel routes** — Saat VPN client L2TP dihapus dari web, entri di `/etc/salfanet/l2tp/peer-routes.conf` tidak dihapus dan kernel routes (`ip route del`) tidak dijalankan. Fix: tambah `removeL2TPPeerRoutes(peer.PeerIP)` yang membaca peer-routes.conf, menghapus kernel routes semua subnet yang terdaftar, lalu menghapus baris dari file.
- **vpn-watchdog — L2TP route parsing format salah** — Watchdog CHECK E membaca peer-routes.conf dengan format `<net> via <ip>` (salah). Format sebenarnya adalah `<peerVpnIP> <net1> [net2]...`. Fix: parsing diubah — field pertama sebagai gateway IP, field 2+ sebagai CIDR network yang perlu ada di routing table.
### Files
- `internal/api/handlers/network_vpn_ext_handler.go` — Tambah `removeL2TPPeerRoutes()`, panggil saat delete L2TP peer
- `vpn-watchdog.sh` — Fix parsing peer-routes.conf di CHECK E (L2TP route restore)

---

## [2.52.51] — 2026-05-21
### Fixed
- **L2TP chap-secrets write — tidak lagi silent fail** — Go handler `CreateL2TPPeer` sebelumnya mengabaikan error saat menulis ke `/etc/ppp/chap-secrets` (`_, _ = ...`). Sekarang error dikembalikan sebagai HTTP 500 sehingga user tahu jika kredensial VPN gagal disimpan.
- **vpn-watchdog — PEER_IP dinamis dari ppp0** — PEER_IP sebelumnya hardcoded `10.20.30.1` (salah). Sekarang dibaca dinamis dari routing table ppp0 via `ip route show dev ppp0 | grep 'proto kernel' | awk '{print $1}'` dengan fallback `10.201.0.10`.
- **systemd ProtectSystem=strict — /etc/ppp dan /etc/salfanet read-only** — `ProtectSystem=strict` memblokir Go service dari menulis ke `/etc/ppp/chap-secrets` dan `/etc/salfanet/l2tp/peer-routes.conf`. Fix: tambah `/etc/ppp /etc/salfanet /etc/wireguard` ke `ReadWritePaths` di systemd service. Juga ditambahkan auto-patch di `updater.sh` untuk instalasi lama.
### Files
- `internal/api/handlers/network_vpn_ext_handler.go` — Chap-secrets write sekarang return error jika gagal
- `vpn-watchdog.sh` — PEER_IP dinamis dari kernel route ppp0
- `vps-install/install-go.sh` — Tambah `/etc/ppp /etc/salfanet /etc/wireguard` ke ReadWritePaths
- `vps-install/updater.sh` — Auto-patch ReadWritePaths jika belum ada `/etc/ppp`

---

## [2.52.50] — 2026-05-21
### Added
- **L2TP VPN Client — input IP lokal dan auto-routing VPS** — Form tambah VPN client sekarang menampilkan field "IP Lokal / Subnet di Balik NAS" untuk tipe L2TP (VPS L2TP server) sama seperti WireGuard. Saat VPN client L2TP dibuat dengan `localNetworks` diisi, Go handler akan: (1) menulis entri ke `/etc/salfanet/l2tp/peer-routes.conf`, (2) memasang hook `/etc/ppp/ip-up.d/99-salfanet-routes.sh` yang otomatis menambahkan route ke subnet lokal NAS setiap kali tunnel L2TP tersambung, (3) mencoba `ip route replace` langsung (best-effort jika PPP sudah aktif). Script MikroTik yang dihasilkan juga menyertakan perintah route dan firewall untuk routing balik ke subnet VPN.
- **WireGuard VPN Peer — localNetworks benar-benar digunakan** — `CreateWGPeer` sebelumnya menerima `localNetworks` tapi mengabaikannya. Sekarang subnet lokal disertakan di `AllowedIPs` peer di `wg0.conf` dan live via `wg set`, serta `ip route replace` langsung diterapkan di VPS.
### Files
- `src/app/admin/network/vpn-client/page.tsx` — Field "IP Lokal" kini tampil juga untuk L2TP VPS; label hint disesuaikan per tipe VPN
- `internal/api/handlers/network_vpn_ext_handler.go` — `CreateL2TPPeer`: handle `localNetworks` → peer-routes.conf + ip-up.d hook + immediate routes + script; `CreateWGPeer`: build `allowedIPs` dari localNetworks + `ip route replace`

### Fixed
- **Tambah OLT di admin/network/olts — 405 Method Not Allowed** — `POST /api/network/olts` tidak ada route-nya. Backend hanya punya route CRUD di `/api/olt/*` bukan `/api/network/olts`. Fix: tambah `POST`, `PUT`, `DELETE` di network group + 3 handler baru di `NetworkHandler` yang handle ID dari request body (bukan URL param).
- **OLT Test Connection — TypeError: Cannot read properties of undefined (reading 'tests')** — Dua root cause: (1) Frontend memanggil `POST /api/olt/test-connection` yang tidak ada route-nya. Fix: tambah `olt.Post("/test-connection", oltH.TestConnection)`. (2) Handler `TestOLTConnection` di AdminMiscHandler hanya stub tanpa field `results.tests`. Fix: buat handler baru di `OLTHandler.TestConnection` yang melakukan TCP check ke port SSH (dan Telnet jika enabled) lalu return `{success, results: {tests: [{method, success, message, time}]}}`. (3) Frontend tidak memproteksi akses `result.results.tests.map(...)` saat field undefined → fix dengan optional chaining + fallback.
### Files
- `internal/api/handlers/network_ext.go` — Tambah `CreateOLT`, `UpdateOLT`, `DeleteOLT` ke `NetworkHandler`; tambah import `strconv`
- `internal/api/handlers/olt.go` — Tambah `TestConnection` ke `OLTHandler`; tambah import `net`
- `internal/api/router.go` — Tambah `POST/PUT/DELETE /network/olts` + `POST /olt/test-connection`
- `src/app/admin/network/olts/page.tsx` — Fix unsafe `result.results.tests.map()` → optional chaining + fallback string

---

## [2.52.48] — 2026-05-20
### Fixed
- **Visibilitas menu per role** — 3 bug fixes:
  1. Parent menu "Payment" pakai `requiredPermission: 'settings.payment'` → diubah ke `'invoices.view'` agar role FINANCE, CUSTOMER_SERVICE, dan VIEWER bisa melihat menu Manual Payments
  2. Payroll Templates + HR Management (attendance, cash advances, commissions, payroll) pakai `settings.view` → diubah ke `keuangan.view` agar role FINANCE bisa akses menu payroll/HR
  3. Handler `POST /api/admin/users` tidak menyimpan array `permissions` dari form → diperbaiki, permissions kini disimpan ke tabel `user_permissions` saat create user
- **DB role_permissions** — Tambah permission `sessions.view` ke role FINANCE, tambah `routers.view` ke role VIEWER (via SQL INSERT IGNORE)
### Files
- `src/app/admin/AdminClientLayout.tsx` — Payment parent permission guard, Payroll Templates + HR Management permission guards
- `internal/api/handlers/admin_users.go` — Create handler: tambah field `Permissions`, loop simpan ke `user_permissions` table

---

## [2.52.47] — 2026-05-20
### Fixed
- **Tambah user admin 400 Bad Request** — Handler `POST /api/admin/users` memvalidasi field `name` wajib tidak kosong, tapi form frontend (`management/page.tsx`) tidak memiliki input `name`. Fix: jika `name` tidak dikirim (kosong), otomatis di-default ke nilai `username`.
### Files
- `internal/api/handlers/admin_users.go` — `Create`: hapus `name` dari required validation, fallback `body.Name = body.Username`

---

## [2.52.46] — 2026-05-20
### Fixed
- **ERR_CONNECTION_CLOSED pada polling admin (setelah idle)** — Tiga fungsi polling di `AdminClientLayout.tsx` (`loadPending`, `loadPendingPayments`, `pollNotifications`) menggunakan `fetch()` tanpa retry. Saat koneksi keepalive di-drop oleh browser/Cloudflare setelah beberapa menit idle, request berikutnya gagal dengan `ERR_CONNECTION_CLOSED`. Fix: tambah helper `fetchWithRetry` (1x retry setelah 1 detik) di level modul dan ganti semua `fetch()` di polling dengan `fetchWithRetry()`.
- **Nginx `proxy_next_upstream`** — Tambah `proxy_next_upstream error timeout` + `proxy_next_upstream_tries 2` + `proxy_next_upstream_timeout 5s` ke semua blok `location /api/` (catch-all) di 3 server block nginx untuk auto-retry di level proxy jika backend terputus.
### Files
- `src/app/admin/AdminClientLayout.tsx` — tambah `fetchWithRetry` helper, ganti `fetch()` di 3 polling useEffect
- `/etc/nginx/sites-enabled/salfanet-radius` — tambah `proxy_next_upstream` ke blok `/api/`

---

## [2.52.45] — 2026-05-20
### Fixed
- **Edit Admin User 500 Internal Server Error** — Handler `PUT /api/admin/users/:id` meneruskan seluruh body JSON ke GORM `Updates()` termasuk field `permissions` (array) yang bukan kolom di tabel `admin_users`, menyebabkan SQL error. Fix: whitelist hanya field valid (`username`, `email`, `phone`, `name`, `role`, `isActive`, `twoFactorEnabled`, `password`), dan proses field `permissions` secara terpisah via tabel `user_permissions`.
### Files
- `internal/api/handlers/admin_users.go` — `Update`: whitelist kolom valid, handle permissions array secara terpisah

---

## [2.52.44] — 2026-05-20
### Fixed
- **RADIUS script NAS IP kosong + RADIUS IP masih public** — Root cause: tabel `vpn_clients` kosong di DB, sehingga lookup `h.db.First(&vpnClient, "id = ?", vpnClientId)` gagal dan `nasSrcAddress` tetap kosong → `radiusServerIP` tidak pernah di-override dari public IP. Fix: tambah fallback `nasSrcAddress = router.IPAddress` saat VPN client tidak ditemukan. Untuk VPN router, `nas.ipAddress` IS the VPN IP (mis. `10.201.0.10`), sehingga `radiusServerIP` bisa di-derive dengan benar ke `10.201.0.1`.
### Files
- `internal/api/handlers/misc_handler.go` — `SetupRadiusOnRouter`: fallback ke `router.IPAddress` saat `vpn_clients` lookup gagal

---

## [2.52.43] — 2026-05-20
### Fixed
- **RADIUS setup script pakai IP publik saat router via VPN** — Sebelumnya `SetupRadiusOnRouter` selalu pakai `RADIUS_SERVER_IP` env (103.151.140.110) bahkan saat router terhubung via VPN. Seharusnya pakai IP VPN internal (mis. 10.201.0.1). Fix: saat `vpnClientId` ada, lookup VPN client `isRadiusServer=true` untuk RADIUS IP, atau derive dari VPN IP NAS (replace last octet dengan .1). Juga ditambahkan: gateway masquerade entry, `require-message-auth=no` (ROS7), PPP pool `pool-radius-default`, PPP profile `salfanetradius`, netwatch monitoring, `wireless` di service list, `interim-update=5m`.
### Files
- `internal/api/handlers/misc_handler.go` — `SetupRadiusOnRouter`: full rewrite VPN IP logic + tambah gateway masquerade, pool/profile PPP, netwatch

---

## [2.52.42] — 2026-05-20
### Fixed
- **Password & RADIUS secret kosong saat edit router** — Saat tombol edit router diklik, field password MikroTik dan RADIUS secret selalu kosong karena `models.Router` punya `json:"-"` pada kedua field tersebut sehingga tidak ikut di-return API list. Fix: `GetRouter` endpoint (`GET /api/network/routers/:id/detail`) sekarang me-return password & secret secara eksplisit via `fiber.Map`; `handleEdit` di frontend di-refactor jadi `async` dan fetch credential dari endpoint detail setelah modal terbuka, lalu patch `formData` dengan nilai yang didapat.
### Files
- `internal/api/handlers/network_ext.go` — `GetRouter`: return explicit `fiber.Map` dengan field `password` dan `secret`
- `src/app/admin/network/routers/page.tsx` — `handleEdit`: ubah ke `async`, fetch `/api/network/routers/:id/detail`, patch `password` & `secret` ke form state

---

## [2.52.41] — 2026-05-20
### Fixed
- **RADIUS script ROS6/ROS7 tidak lengkap** — Script yang di-generate tombol "RADIUS Script" hanya berisi 4 baris minimal. Banyak perintah penting hilang dibanding versi Next.js sebelumnya. Fix komprehensif:
  1. Tambah header berisi nama NAS, IP RADIUS, jenis koneksi, tanggal generate
  2. Idempotent: `/radius remove [find ... comment~"Salfanet"]` sebelum add (ROS7 pakai `where`, ROS6 tanpa `where`)
  3. Timeout benar: ROS7 = `timeout=3s`, ROS6 = `timeout=3` (tanpa 's')
  4. Tambah `/ppp aaa set use-radius=yes accounting=yes interim-update=00:10:00` (sebelumnya hilang)
  5. Tambah `radius-accounting=yes radius-interim-update=00:10:00` di hotspot profile (sebelumnya hilang)
  6. Ganti `/ppp profile set [find] use-radius=yes` (salah) → `/ppp aaa set use-radius=yes` (benar)
  7. Ganti `/ip radius incoming` (salah) → `/radius incoming` (benar)
  8. Tambah firewall rules untuk allow CoA (3799) dan Auth/Acct (1812/1813) dari RADIUS server
  9. Untuk router via VPN: lookup VPN client IP → tambah `src-address={vpnIp}` ke `/radius add`, derive RADIUS server IP dari VPN gateway (replace last octet jadi .1)
  10. Tambah blok verifikasi di akhir script
### Files
- `internal/api/handlers/misc_handler.go` — Rewrite `SetupRadiusOnRouter`: VPN client IP lookup, complete ROS6+ROS7 scripts, proper PPP AAA, hotspot accounting, firewall rules, idempotent remove, correct timeout syntax

## [2.52.40] — 2026-05-21
### Fixed
- **L2TP VPN putus setiap update (install-l2tp-server.sh overwrite ipsec.conf)** — Script installer menulis `/etc/ipsec.conf` dengan proposal IKE/ESP terlalu sempit dan flag strict (`!`), sehingga MikroTik gagal phase1 ("phase1 negotiation failed due to time up"). Dua masalah: (1) `ike=` hanya berisi `aes256-sha256-modp2048,aes256-sha1-modp1024!` — MikroTik default mengirim `aes128-sha1-modp1024` yang ditolak. (2) `esp=aes256-sha256,aes256-sha1!` tanpa modp1024, padahal MikroTik default ESP pakai PFS modp1024. Fix: perluas `ike=` dan `esp=` mencakup semua varian AES-128/192/256 + SHA1 + modp1024, hapus flag strict `!`. Juga fix `auth` → `noauth` di pppd options (Ubuntu 22.04 pppd 2.4.9 tidak support MSCHAPv2 natively; autentikasi sudah dijamin IPsec PSK phase 1).
### Files
- `vps-install/install-l2tp-server.sh` — Perluas `ike=` dan `esp=` proposals; hapus flag strict `!`; ganti `auth` → `noauth` di pppd options; hapus em dash (`—`) dari baris komentar di heredoc ipsec.conf (em dash menyebabkan strongSwan 5.9.5 diam-diam gagal parse file sehingga `Connections:` tetap kosong meski syntax benar)

## [2.52.39] — 2026-05-21
### Fixed
- **Router/NAS test connection gagal saat VPN client dipilih** — Dua bug:
  1. `TestGateway` hanya mencoba TCP probe (port 8728/22/80/443). Jika MikroTik firewall memblokir semua port TCP dari subnet VPN, ping test gagal meski VPN tunnel aktif. Fix: tambah ICMP ping fallback menggunakan `ping -c 1 -W 2` setelah TCP probe gagal.
  2. Perintah firewall MikroTik yang ditampilkan saat API test gagal menggunakan IP hardcoded salah `172.16.212.1` sebagai `src-address`. Fix: backend `TestGateway` sekarang mengembalikan `localIp` (IP VPS di tunnel, contoh `10.201.0.1`) via `ip route get`, dan frontend menggunakannya di firewall command.
### Files
- `internal/api/handlers/misc_handler.go` — `TestGateway`: tambah ICMP ping fallback; tambah helper `getLocalIPForDest()` menggunakan `ip route get`; tambah `localIp` ke semua response sukses
- `src/app/admin/network/routers/page.tsx` — Capture `pingResult.localIp` → simpan ke `vpnGatewayIp`; ganti hardcoded `172.16.212.1` dengan `${vpnGatewayIp}` di firewall command

## [2.52.38] — 2026-05-20
### Fixed
- **DELETE router 405 Method Not Allowed** — Frontend mengirim `DELETE /api/network/routers?id=xxx` (query param) tapi Go router hanya mendaftarkan `DELETE /network/routers/:id` (path param). Semua request DELETE ke URL lama diterima oleh route `GET/POST /network/routers` yang tidak punya handler DELETE → 405. Fix: ubah URL delete dari `?id=${id}` ke `/${id}`.
- **Router status selalu offline setelah simpan** — Handler `RouterStatus` (Go) mengembalikan `{ routers: [{id, name, status: "UNKNOWN"}] }` sedangkan frontend mengharapkan `{ statusMap: { [id]: { online, identity } } }`. Karena `data.statusMap` selalu `undefined`, `setStatusMap({})` dipanggil dan semua router tampil offline. Fix: rewrite handler untuk menerima `{ routerIds }`, lakukan TCP ping ke port API router secara paralel (goroutine), dan kembalikan `statusMap` dengan format yang benar.
- **Short name tidak muncul** — Form modal tidak memiliki input `shortname`, sehingga selalu tersimpan sebagai string kosong. Fix: tambah input shortname di modal dengan auto-generate dari nama router (lowercase, replace spasi/karakter ke `-`, max 20 char).
- **TestGateway/TestRouterGeneric selalu return sukses** — Kedua handler sebelumnya adalah stub yang selalu mengembalikan `success: true` tanpa melakukan koneksi nyata. Akibatnya test VPN selalu "berhasil" meski VPN tidak terhubung. Fix: implementasi TCP check nyata — `TestGateway` mencoba port 8728/22/80/443; `TestRouterGeneric` mencoba port API plain lalu SSL dengan timeout 3 detik.
### Files
- `src/app/admin/network/routers/page.tsx` — Fix DELETE URL (`?id=` → `/${id}`); tambah shortname input field dengan auto-generate; update handler `onChange` nama router
- `internal/api/handlers/network_ext.go` — Rewrite `RouterStatus`: terima `{ routerIds }`, TCP ping paralel via `tcpPing()`, kembalikan `{ statusMap: { [id]: { online, identity } } }`; tambah helper `tcpPing()`; tambah import `fmt`, `net`, `sync`
- `internal/api/handlers/misc_handler.go` — Rewrite `TestRouterGeneric` dan `TestGateway` dengan TCP connectivity check nyata; tambah import `net`

## [2.52.37] — 2026-05-20
### Fixed
- **Updater.sh tidak update kode terbaru (self-overwrite bug)** — Root cause: saat updater.sh berjalan, `rsync` menyalin file baru dari `/root/salfanet-radius/` ke `/var/www/salfanet-radius/` termasuk `vps-install/updater.sh` yang SEDANG BERJALAN. Bash membaca script dalam blok/chunk, sehingga overwrite di tengah eksekusi menyebabkan bash membaca konten campur lama+baru → step penting terlewat. Fix: tambah bootstrap pattern di awal script — copy diri sendiri ke `/tmp/` dan re-exec sekali sebelum apapun berjalan (`_UPDATER_BOOTSTRAP` env flag mencegah loop).
### Files
- `vps-install/updater.sh` — Tambah bootstrap self-copy ke `/tmp/` via `_UPDATER_BOOTSTRAP` flag di awal script

## [2.52.36] — 2026-05-20
### Fixed
- **VPN Client data masih hilang setelah updater.sh** — Root cause: `mysqldump --no-create-info` hanya meng-export INSERT statements (data), tanpa `CREATE TABLE`. Saat Prisma DROP tabel saat `prisma db push`, restore berjalan tapi INSERT gagal karena tabel tidak ada → data hilang diam-diam. Fix: hapus `--no-create-info`, ganti dengan `--add-drop-table` (default mysqldump) di semua fungsi backup (`backup_vpn_data`, `backup_vps_peers_data`, `backup_genieacs_data`). Sekarang backup menyertakan `DROP TABLE IF EXISTS` + `CREATE TABLE` + `INSERT`, jadi restore recreate tabel dari nol bahkan jika Prisma menghapusnya. Juga tambah `SET FOREIGN_KEY_CHECKS=0` di restore GenieACS untuk konsistensi.
### Files
- `vps-install/updater.sh` — `backup_vpn_data`, `backup_vps_peers_data`, `backup_genieacs_data`: hapus `--no-create-info`, tambah `--add-drop-table`; `restore_genieacs_data`: tambah `SET FOREIGN_KEY_CHECKS=0/1`

## [2.52.35] — 2026-05-20
### Fixed
- **VPN Client data selalu hilang setiap deploy (permanen fix)** — Root cause: `vpn_servers` dan `vpn_clients` masih dikelola Prisma. Saat `prisma db push --accept-data-loss` berjalan dengan schema change (misal: hapus FK di v2.52.33), Prisma DROP + RECREATE tabel sehingga data hilang. Backup/restore di updater.sh tidak cukup reliable. Fix permanen: tambah `@@ignore` ke kedua model di Prisma schema → Prisma tidak pernah menyentuh tabel ini lagi. Tambah `CREATE TABLE IF NOT EXISTS vpn_servers` dan `CREATE TABLE IF NOT EXISTS vpn_clients` ke `runMigrations` di `db.go` → Go yang create dan manage kedua tabel ini (sama seperti `vps_peers`).
### Files
- `prisma/schema.prisma` — Tambah `@@ignore` ke model `vpnServer` dan `vpnClient`
- `internal/db/db.go` — Tambah `CREATE TABLE IF NOT EXISTS vpn_servers` dan `vpn_clients` ke `runMigrations`

## [2.52.34] — 2026-05-21
### Fixed
- **TypeError: Cannot read properties of undefined (reading 'radiusServer')** — `SetupRadiusOnRouter` Go handler adalah stub kosong (hanya return `success: true` tanpa `config`, `script`, dll). Setelah router disimpan, frontend otomatis memanggil `handleSetupRadius` → `setScriptModalData({ config: result.config })` → `config` = `undefined` → modal crash saat render `scriptModalData.config.radiusServer`. Fix: implementasikan handler yang benar (lookup router, generate script RouterOS 6 + 7, return `config` object lengkap) dan tambah optional chaining `?.` di frontend sebagai safety guard.
### Files
- `internal/api/handlers/misc_handler.go` — Implementasi nyata `SetupRadiusOnRouter`: lookup router, baca `APP_BASE_URL`/`RADIUS_SERVER_IP` env untuk server IP, generate script MikroTik ROS6+ROS7, return `config` + `script` + `scriptRos6` + `scriptRos7`
- `src/app/admin/network/routers/page.tsx` — Optional chaining `?.` pada `scriptModalData.config?.radiusServer` dll agar tidak crash bila `config` undefined

## [2.52.33] — 2026-05-20
### Fixed
- **500 Error saat Tambah Router dengan VPN Client dari vps_peers** — `nas.vpnClientId` punya FK constraint ke `vpn_clients(id)`, tapi entry dari `vps_peers` ID-nya tidak ada di `vpn_clients` sehingga MySQL error 1452 FK constraint fails. Fix: hapus `@relation` dari Prisma schema (kolom tetap ada sebagai nullable string) dan tambah migration SQL untuk drop FK constraint di DB. `vpnClientId` sekarang bisa menyimpan ID dari `vpn_clients` maupun `vps_peers` tanpa FK enforcement.
### Files
- `prisma/schema.prisma` — Hapus `routers router[]` dari `vpnClient` model dan `vpnClient @relation(...)` dari `router` model
- `prisma/migrations/drop_nas_vpnclientid_fkey.sql` — `ALTER TABLE nas DROP FOREIGN KEY nas_vpnClientId_fkey`

## [2.52.32] — 2026-05-20
### Fixed
- **VPN Client data hilang setiap update** — `vpn_servers` dan `vpn_clients` adalah tabel Prisma, tapi saat `prisma db push --accept-data-loss` berjalan dengan perubahan schema, datanya bisa DROP. Fix: `updater.sh` kini mem-backup kedua tabel ini sebelum Prisma berjalan dan merestore-nya sesudah (dengan `SET FOREIGN_KEY_CHECKS=0` agar urutan restore tidak masalah).
### Files
- `vps-install/updater.sh` — Tambah `backup_vpn_data` / `restore_vpn_data`, dipanggil di Mode A dan Mode B sekitar `prisma db push`

## [2.52.31] — 2026-05-20
### Fixed
- **400 Error saat Tambah/Edit Router** — Field `port`, `apiPort`, `ports` dikirim sebagai string dari form state, tapi Go struct `Router` expect `int`. Fix: konversi ke `parseInt()` di `handleSubmit`. Juga fix: PUT URL edit sekarang `/api/network/routers/:id` (sebelumnya tidak ada `:id`). Router model ditambah field `Server`, `Community`, `VpnClientId` yang ada di tabel `nas` tapi belum ada di struct. `CreateRouter` kini menggunakan `routerBody` struct agar `password` dan `secret` (yang punya `json:"-"` di `Router` struct) tetap bisa dibaca dari request body.
### Files
- `src/app/admin/network/routers/page.tsx` — `handleSubmit`: `parseInt` untuk port fields, fix PUT URL ke `/api/network/routers/:id`
- `internal/api/handlers/network.go` — Tambah `routerBody` struct, fix `CreateRouter` untuk handle `password`/`secret`
- `internal/db/models/models.go` — `Router` struct: tambah field `Server`, `Community`, `VpnClientId`

## [2.52.30] — 2026-05-20
### Fixed
- **vps_peers terhapus saat deploy** — `updater.sh` kini mem-backup data tabel `vps_peers` sebelum `prisma db push` dan merestore-nya setelahnya (seperti GenieACS tables). Go API juga di-restart ulang setelah Prisma selesai agar `runMigrations` memastikan tabel selalu ada.
- **VPN Client dropdown tidak menampilkan VPS WireGuard/L2TP peers** — `ListRouters` sebelumnya hanya fetch dari `vpn_clients`. Kini juga fetch dari `vps_peers` (tabel Go-managed), sehingga peer WireGuard/L2TP yang didaftarkan via halaman VPN Clients ikut muncul di dropdown.
### Files
- `vps-install/updater.sh` — Tambah `backup_vps_peers_data` / `restore_vps_peers_data` + restart Go setelah Prisma (Mode A & B)
- `internal/api/handlers/network.go` — `ListRouters`: tambah fetch dari `vps_peers` dan gabungkan ke hasil `vpnClients`

## [2.52.29] — 2026-05-20
### Fixed
- **VPN Client dropdown kosong di modal "Tambah Router Baru"** — `GET /api/network/routers` sebelumnya hanya mengembalikan array router mentah. Frontend mengharapkan `{ routers: [...], vpnClients: [...] }`, sehingga `data.routers` dan `data.vpnClients` keduanya `undefined` → `[]`. Fix: handler `ListRouters` kini fetch VPN clients dari tabel `vpn_clients` (beserta NAS secret dari tabel `nas`) dan mengembalikan response dalam format yang benar.
### Files
- `internal/api/handlers/network.go` — `ListRouters`: ganti `c.JSON(routers)` dengan `c.JSON({ routers, vpnClients })`

## [2.52.28] — 2026-05-19
### Fixed / Added
- **Import: Profile Default Fallback** — Dialog import kini memiliki dropdown **Profile Default** (opsional). Jika nama profile di file tidak cocok dengan DB (atau DB profile kosong), profile default yang dipilih di UI dipakai — persis seperti sistem Next.js lama.
- **Detail error import tampil** — Backend kini mengembalikan key `errors` (bukan `failures`) dengan field `line`, `username`, `error` sehingga detail kegagalan per-baris tampil di dialog import.
- **Root cause 75 gagal** — `pppoe_profiles` tabel kosong (0 row); semua baris gagal karena profile tidak ditemukan. Kini bisa diatasi dengan memilih Profile Default di dialog import.
### Files
- `internal/api/handlers/pppoe_ext.go` — BulkImport: baca `profileId` form field sebagai fallback; ganti `failures`→`errors`, tambah field `line`/`username`
- `src/app/admin/pppoe/users/page.tsx` — Import dialog: tambah state `importProfileId`, dropdown profile, kirim `profileId` ke backend

## [2.52.26] — 2026-05-19
### Fixed / Added
- **Template kolom lengkap** — Template import kini menyertakan kolom `areaName`, `subscriptionType` (PREPAID/POSTPAID), dan `billingDay`. BulkGet export juga menambahkan kolom-kolom tersebut.
- **Password opsional** — Jika kolom `password` kosong saat import, sistem otomatis membuat password default `<username>123`. Tidak lagi menjadi required field yang memblokir seluruh baris.
- **Column alias** — Import sekarang menerima nama kolom dari output `ExportUsers` (kolom `Profile` → `profileName`, `Router` → `routerName`, `Area` → `areaName`) sehingga file export bisa langsung di-import kembali.
- **Lookup case-insensitive** — Pencarian profile, router, dan area berdasarkan nama tidak lagi case-sensitive.
- **Error message lebih jelas** — Baris yang gagal karena profile tidak ditemukan kini menampilkan nama profile yang dimaksud.
### Files
- `internal/api/handlers/pppoe_ext.go` — `BulkGet` template + export header; `BulkImport` alias, optional password, area support, subscriptionType, case-insensitive lookup

## [2.52.25] — 2026-05-19
### Fixed
- **Import xlsx fallback** — Jika file `.xlsx` yang diupload ternyata berisi konten CSV (format export lama sebelum v2.52.24), import akan otomatis fallback ke parser CSV sehingga tetap berhasil diimport tanpa error "zip: not a valid zip file".
### Files
- `internal/api/handlers/pppoe_ext.go` — `BulkImport`: tambah CSV fallback saat excelize gagal parse xlsx

## [2.52.24] — 2026-05-19
### Added
- **Export real `.xlsx`** — `GET /api/pppoe/users/export?format=excel` sekarang menghasilkan file Excel binary asli (`.xlsx`) menggunakan library `excelize v2.10.1`, bukan CSV yang di-rename. Begitu juga `GET /api/pppoe/users/bulk?type=template&format=xlsx` menghasilkan template Excel asli.
- **Import dari `.xlsx`** — `POST /api/pppoe/users/bulk` kini menerima file `.xlsx` maupun `.xls` selain `.csv`. File Excel di-parse menggunakan excelize, baris data diproses sama seperti CSV.
### Changed
- **Dependency baru** — `github.com/xuri/excelize/v2 v2.10.1` ditambahkan ke `go.mod`.
### Files
- `internal/api/handlers/pppoe_ext.go` — tambah `writeXLSX()` helper, update `ExportUsers`, `BulkGet`, `BulkImport`
- `go.mod`, `go.sum` — tambah excelize dan dependensi turunannya

## [2.52.23] — 2026-05-19
### Fixed
- **`POST /api/pppoe/users/bulk` 400 saat import** — Root cause: `c.FormFile()` di Fiber v3 beta tidak selalu berhasil parse multipart. Diganti dengan `c.MultipartForm()` yang lebih eksplisit. Ditambah juga deteksi file `.xlsx/.xls` (binary Excel) yang tidak bisa di-parse sebagai CSV, mengembalikan error yang jelas: *"Simpan file sebagai CSV terlebih dahulu"*.
- **Template download selalu `.csv`** — `GET /api/pppoe/users/bulk?type=template&format=xlsx` sebelumnya mengirim CSV dengan nama file `.xlsx`, membingungkan pengguna agar menyimpan ulang sebagai Excel lalu import gagal. Sekarang selalu menggunakan ekstensi `.csv`.
### Files
- `internal/api/handlers/pppoe_ext.go` — `BulkImport`: pakai `MultipartForm()`, deteksi xlsx, pesan error bahasa Indonesia; `BulkGet`: template selalu `.csv`

## [2.52.22] — 2026-05-20
### Fixed
- **404 pada `GET /api/pppoe/users/export`** — Di Fiber v3 beta, route parametrik `/users/:id` menangkap path statis seperti `/users/export`. Semua static sub-path `/pppoe/users/*` dipindah ke dalam `pppoe` group **sebelum** `/users/:id`, sehingga Fiber menggunakan static route yang tepat.
- **Import PPPoE user dari CSV gagal 400** — `POST /api/pppoe/users/bulk` sebelumnya ditangani `miscH.PppoeBulk` yang hanya menerima JSON. Diganti dengan `BulkImport` yang menerima multipart/form-data dengan field `file` (CSV).
- **Profiles/customers static routes juga diperbaiki** — `/profiles/sync-mikrotik`, `/profiles/sync-radius`, `/customers/export`, `/customers/bulk-create` dipindah sebelum route parametrik `:id` di masing-masing grup.
### Added
- **`GET /api/pppoe/users/bulk`** — Download template CSV (`?type=template`) atau export seluruh data user untuk re-import (`?type=export`). Support filter `?paymentStatus=paid|unpaid`.
- **`POST /api/pppoe/users/bulk`** — Import user dari file CSV (multipart form-data, field `file`). Parse header CSV secara fleksibel (case-insensitive). Mengembalikan `{success, results:{success, failed, failures[]}}`.
- **`DELETE /api/pppoe/users/bulk-delete`** — Bulk delete user PPPoE berdasarkan array `userIds`. Dipakai di halaman Stopped Users.
- **Filter di `ExportUsers`** — Support query param `profileId`, `routerId`, `status`, `format` (csv/excel). Format excel mengembalikan CSV dengan ekstensi .xlsx (kompatibel dengan Excel).
### Changed
- **Konsolidasi semua pppoe routes ke `pppoe` group** — Semua route `/pppoe/**` yang sebelumnya terdaftar via `api.Get(...)` di luar group dipindah ke dalam `pppoe := api.Group("/pppoe")` untuk memastikan ordering statis-sebelum-parametrik konsisten. Blok duplikat "PPPoE extended routes" dan pppoe-related lines di "Batch 7: Misc" dihapus.
- **`POST /users/:id/sync-radius`** — Diganti dari `pppoeH.SyncToRadius` ke `pppoeExtH.SyncUserRadius` (handler yang lebih lengkap).
### Files
- `internal/api/router.go` — Refactor pppoe group: static routes sebelum :id, hapus duplicate blocks
- `internal/api/handlers/pppoe_ext.go` — Tambah `BulkGet`, `BulkImport`, `BulkDelete`; update `ExportUsers` dengan filter

## [2.52.21] — 2026-05-19
### Changed
- **Hapus ~415 dead Next.js API routes** — Semua `src/app/api/**` kecuali `auth/[...nextauth]/route.ts` dihapus. Nginx sudah routing semua `/api/` ke Go, sehingga file-file ini tidak pernah dieksekusi.
- **Ganti Prisma di 4 layout.tsx dengan Go API** — `admin/layout.tsx`, `customer/layout.tsx`, `agent/layout.tsx`, `technician/layout.tsx` sebelumnya pakai `prisma.company.findFirst()` untuk page title. Diganti dengan `fetch('http://127.0.0.1:8080/api/public/company')`.
- **Fix nginx: `/api/auth/logout-log` route ke Go** — Sebelumnya Next.js menangani endpoint ini via `src/app/api/auth/logout-log/route.ts`. Go sudah memiliki handler (`router.go:1200`), nginx kini memiliki `location = /api/auth/logout-log` di semua 3 server block yang mengarah ke Go. File Next.js dihapus bersama dead routes.
### Files
- `src/app/api/**` — hapus semua kecuali `auth/[...nextauth]/route.ts` (415+ file dihapus)
- `src/app/admin/layout.tsx` — ganti Prisma → Go API fetch
- `src/app/customer/layout.tsx` — ganti Prisma → Go API fetch
- `src/app/agent/layout.tsx` — ganti Prisma → Go API fetch
- `src/app/technician/layout.tsx` — ganti Prisma → Go API fetch
- `vps-install/install-nginx.sh` — tambah `location = /api/auth/logout-log → Go` di 2 server block functions
- `/etc/nginx/sites-enabled/salfanet-radius` (VPS) — tambah logout-log block di 3 server block, nginx reload

## [2.52.20] — 2026-05-19
### Fixed
- **`/api/push/send` masih route ke Next.js di server block 2 & 3** — Setelah Go sudah memiliki handler `pushH.Send`, nginx di server block 2 (IP direct) dan block 3 (HTTPS default_server) masih memiliki `location = /api/push/send { proxy_pass 127.0.0.1:3000 }`. Server block 1 (Cloudflare/main) sudah benar. Block 2 & 3 kini konsisten: rule dihapus, request jatuh ke catch-all `/api/ → Go`.
### Files
- `vps-install/install-nginx.sh` — hapus 2 blok `location = /api/push/send → Next.js` dari server block 2 & 3
- `/etc/nginx/sites-enabled/salfanet-radius` (VPS) — hapus blok yang sama, nginx reload

## [2.52.19] — 2026-05-19
### Fixed
- **ERR_CONNECTION_CLOSED pada idle API connections** — Root cause: nginx `keepalive_timeout 65s` terlalu pendek, Cloudflare mempertahankan koneksi ke origin ~90s. Jika nginx tutup koneksi duluan, Cloudflare mencoba kirim request ke koneksi yang sudah tutup → browser mendapat `ERR_CONNECTION_CLOSED`. Fix: naikkan nginx `keepalive_timeout` 65→120s dan Fiber `IdleTimeout` 60→150s agar server tidak menutup koneksi sebelum Cloudflare.
- **Duplicate nginx config** — File `/etc/nginx/sites-enabled/radius.hotspotapp.net` (lama) konflik dengan `salfanet-radius`, menyebabkan warning "conflicting server name". Dihapus dari sites-enabled.
### Files
- `internal/api/router.go` — `IdleTimeout` 60s → 150s
- `vps-install/install-nginx.sh` — `keepalive_timeout` 65 → 120 (4 lokasi)

## [2.52.18] — 2026-05-19
### Fixed
- **Telegram settings tidak tampil setelah save** — `loadSettings()` melakukan `setTelegramSettings(data)` padahal API return `{ success: true, settings: {...} }`. Akibatnya seluruh field (botToken, chatId, dll) jadi `undefined` setelah reload. Data sebenarnya sudah tersimpan di DB, tapi UI tidak menampilkannya. Fix: ganti ke `setTelegramSettings(data.settings)` dengan fallback untuk tiap field.
### Files
- `src/app/admin/settings/telegram/page.tsx` — fix `loadSettings`: `setTelegramSettings(data)` → `setTelegramSettings(data.settings)` dengan field defaults

## [2.52.17] — 2026-05-19
### Fixed
- **Backup Create 500 (lanjutan 2)** — `ProtectSystem=strict` di systemd membuat `/backups` read-only karena tidak ada di `ReadWritePaths`. Meski `MkdirAll` berhasil (berjalan sebagai root), `sh -c "... > file.gz"` tetap diblokir oleh kernel sandboxing. Fix: tambah `/var/www/salfanet-radius/backups` ke `ReadWritePaths` di `/etc/systemd/system/salfanet-api.service`, daemon-reload, restart.
- **Installer future-proof** — `install-go.sh` dan `updater.sh` diperbarui agar `/backups` selalu ada di `ReadWritePaths` dan direktori `backups/` dibuat saat install/update.
### Files
- `/etc/systemd/system/salfanet-api.service` (VPS) — tambah `${APP_DIR}/backups` ke `ReadWritePaths`
- `vps-install/install-go.sh` — `ReadWritePaths` ditambah `${_APP_DIR}/backups`
- `vps-install/updater.sh` — patch `ReadWritePaths` untuk `/backups` jika belum ada; `mkdir -p backups`

## [2.52.16] — 2026-05-19
### Fixed
- **Backup Create 500 (lanjutan)** — `mysqldump` gagal dengan "Access denied; you need PROCESS privilege" saat dump tablespaces. User `salfanet_user` tidak punya privilege tersebut. Fix: tambah flag `--no-tablespaces` ke perintah mysqldump.
- **Backup Delete 404** — Route terdaftar sebagai `DELETE /api/backup/:id` tapi frontend memanggil `DELETE /api/backup/delete/{id}`. Fix: ubah route ke `/api/backup/delete/:id`.
### Files
- `internal/api/handlers/backup_handler.go` — tambah `--no-tablespaces` ke perintah mysqldump
- `internal/api/router.go` — route DELETE dari `/backup/:id` → `/backup/delete/:id`

## [2.52.15] — 2026-05-19
### Fixed
- **Backup Create HTTP 500** — handler menggunakan `DB_HOST`/`DB_USER`/`DB_PASSWORD`/`DB_NAME` yang tidak ada di VPS (hanya ada `DATABASE_URL`). Fix: `parseDBCredentials()` parse `DATABASE_URL` Prisma format dengan `net/url`. Gunakan `MYSQL_PWD` env var (bukan flag `-p`) agar password dengan karakter spesial aman dari shell escaping.
- **Backup Restore gagal** — sama, credentials parsing salah. Fixed bersama dengan fix backup create.
- **Backup history 0 items** — konsekuensi dari backup create yang selalu gagal; otomatis teratasi setelah create diperbaiki.
- **Telegram settings tidak tersimpan** — `GetTelegramSettings`/`UpdateTelegramSettings` di kedua handler adalah stub hardcoded. Fix: baca/tulis ke tabel `telegram_backup_settings` di DB. Tambah model `TelegramBackupSettings` di Go.
- **Telegram send/test-backup tidak berfungsi** — semua endpoint Telegram selalu return stub success tanpa mengirim apapun. Fix: implementasi penuh dengan Telegram Bot API (`sendMessage` untuk teks, `sendDocument` untuk file backup via multipart upload).
- **ERR_CONNECTION_CLOSED saat idle** — Fiber `IdleTimeout=270s` sedangkan nginx `keepalive_timeout` default 75s. Race condition: nginx tutup koneksi lebih dulu, browser coba reuse → `ERR_CONNECTION_CLOSED`. Fix: turunkan Fiber `IdleTimeout` ke 60s + tambah `keepalive_timeout 65;` ke semua server block di nginx.
### Files
- `internal/db/models/extra.go` — tambah model `TelegramBackupSettings` (tabel `telegram_backup_settings`)
- `internal/api/handlers/backup_handler.go` — rewrite: `parseDBCredentials()`, `doMysqlDump()`, `doMysqlRestore()`, `sendTelegramMessage()`, `sendTelegramDocument()`, semua handler implementasi penuh
- `internal/api/handlers/telegram_handler.go` — rewrite: `GetSettings`/`UpdateSettings` baca/tulis DB, `Test`/`SendBackup`/`TestBackup`/`SendHealth` panggil Telegram API sungguhan
- `internal/api/router.go` — `IdleTimeout`: 270s → 60s; fix route `/backup/telegram/test` ke `telegramH.TestBackup`
- `nginx /etc/nginx/sites-enabled/salfanet-radius` — tambah `keepalive_timeout 65;` di semua server block (VPS only)

## [2.52.14] — 2026-05-19
### Fixed
- **Email settings dark mode** — code block "App name" dan "Device" di tutorial Gmail SMTP tidak terlihat di dark mode karena typo class Tailwind `dark:bg-inputpx-1` (harusnya `dark:bg-gray-700 px-1`). Teks putih di atas background abu terang → invisible.
- **Email templates tidak tampil** — Go handler `ListEmailTemplates` mengembalikan 3 stub hardcoded (INVOICE, PAYMENT_CONFIRM, ISOLATION_NOTICE) bukan dari database. Frontend tidak menemukan type yang cocok → semua tab template menampilkan "Template belum dibuat". Fix: baca dari tabel `email_templates` di DB. Tambah model `EmailTemplate` di Go.
- **Update template tidak tersimpan** — `UpdateEmailTemplate` selalu return stub success tanpa benar-benar update DB. Fix: update ke tabel `email_templates` by `type`, return 404 bila tidak ada.
### Files
- `src/app/admin/settings/email/page.tsx` — fix class Tailwind `dark:bg-inputpx-1` → `dark:bg-gray-700 px-1`
- `internal/db/models/extra.go` — tambah model `EmailTemplate` (tabel `email_templates`)
- `internal/api/handlers/settings_ext.go` — fix `ListEmailTemplates` (baca dari DB) dan `UpdateEmailTemplate` (update ke DB)

## [2.52.13] — 2026-05-19
### Fixed
- **Crash di tab Kirim/Broadcast (`Cannot read properties of undefined (reading 'map')`)** — data list dari API (`users`, `templates`, dan `filters.*`) bisa datang `undefined` pada kondisi tertentu dan langsung dipakai di `.map(...)`. Fix: normalisasi semua payload list ke array aman sebelum disimpan ke state agar UI tidak crash.
### Files
- `src/app/admin/whatsapp/send/page.tsx` — tambah normalisasi array aman untuk `users`, `templates`, dan `filters` sebelum render `.map(...)`

## [2.52.12] — 2026-05-20
### Fixed
- **Gagal memuat history (root cause)** — `waH.ListHistory` (route terdaftar pertama, line 341) mengembalikan raw array tanpa `success: true`; Fiber v3 selalu menggunakan handler pertama yang terdaftar, sehingga fix di `waCrudH.ListHistory` (line 752) tidak pernah dipanggil. Fix: update `waH.ListHistory` langsung dengan response format `{success, data, pagination, stats}` beserta pagination, search, dan status filter.
- **WhatsApp sidebar double menu** — Hapus sub-item dropdown (Riwayat, Template, Kirim, Notifikasi, Penyedia) dari sidebar; WhatsApp kini langsung menuju `/admin/settings/whatsapp` yang sudah memiliki semua tab.
### Files
- `internal/api/handlers/whatsapp.go` — fix `ListHistory`: full response format `{success, data, pagination, stats}` + pagination + search + stats; tambah `strconv` import
- `src/app/admin/AdminClientLayout.tsx` — WhatsApp nav: hapus children dropdown, direct href ke `/admin/settings/whatsapp`

## [2.52.11] — 2026-05-20
### Fixed
- **Template WhatsApp tidak muncul** — `waH.ListTemplates` mengembalikan raw array; frontend mengecek `data.success` → templates tidak pernah dimuat. Fix: wrap response jadi `{success: true, data: [...]}`.
- **Update template gagal** — `waH.UpdateTemplate` lookup by "type" string, tapi frontend mengirim UUID → tidak ketemu, membuat record salah. Fix: lookup by UUID dulu, fallback ke type string. Response kini `{success: true}`.
- **Kirim pesan tunggal selalu error** — `waH.SendMessage` mengembalikan `{message:"sent"}` tanpa `success: true` → frontend selalu masuk blok error. Fix: return `{success: true, message:"sent", provider:"whatsapp"}`.
- **Broadcast tidak mengirim pesan** — `Broadcast` handler hanya membuat record QUEUED tanpa benar-benar mengirim ke WA service. Fix: panggil WA service untuk setiap user, track sukses/gagal, return `{total, successCount, failCount}`.
- **Hapus provider gagal (404)** — tidak ada route `DELETE /api/whatsapp/providers/:id`. Fix: tambah route.
- **REST alias untuk templates** — tambah `GET /whatsapp/templates`, `PUT /whatsapp/templates/:id`, `DELETE /whatsapp/templates/:id` dan `DELETE /whatsapp/providers/:id` sebagai override dari route lama.
- **`ListTemplates` key salah** — `waCrudH.ListTemplates` mengembalikan key `templates`, frontend mengecek `data.data`. Fix: ganti key menjadi `data`.

### Files
- `internal/api/handlers/whatsapp.go` — fix `ListTemplates`, `UpdateTemplate`, `SendMessage` response format
- `internal/api/handlers/whatsapp_crud.go` — fix `ListTemplates` key: `templates` → `data`
- `internal/api/handlers/whatsapp_ext.go` — add `httpClient`, fix `Broadcast` untuk benar-benar kirim ke WA service + response format `{total, successCount, failCount}`
- `internal/api/router.go` — tambah REST alias: `DELETE /whatsapp/providers/:id`, `GET /whatsapp/templates`, `PUT /whatsapp/templates/:id`, `DELETE /whatsapp/templates/:id`

## [2.52.10] — 2026-05-19
### Fixed
- **WhatsApp History gagal dimuat** — Go router mendaftarkan `/whatsapp/history-list` tapi frontend memanggil `/whatsapp/history`; response format juga salah (`history` vs `data`, tidak ada field `stats`). Fix: tambah route `GET /api/whatsapp/history`, ubah response menjadi `data` + `stats` (total/sent/failed/last24Hours), dan handle `search` + `status=all` dengan benar.
### Files
- `internal/api/router.go` — tambah `GET /whatsapp/history` (alias ke `waCrudH.ListHistory`)
- `internal/api/handlers/whatsapp_crud.go` — fix `ListHistory`: response format (data+stats), search param, status=all, empty array not null

## [2.52.9] — 2026-05-19
### Fixed
- **ERR_CONNECTION_CLOSED setelah idle lama** — Nginx `keepalive_timeout` global hanya 65s; browser throttle `setInterval` saat tab di background menjadi >65s sehingga koneksi terputus sebelum poll berikutnya. Fix: tambah `keepalive_timeout 300; keepalive_requests 10000;` di Nginx site config `radius.hotspotapp.net`, dan set `IdleTimeout: 270s`, `ReadTimeout: 60s`, `WriteTimeout: 60s` di Go Fiber. Ini memperbaiki error berulang pada semua polling request: `GET /api/notifications`, `GET /api/admin/registrations?status=PENDING`, `GET /api/manual-payments?status=PENDING`.
### Files
- `internal/api/router.go` — tambah `IdleTimeout`, `ReadTimeout`, `WriteTimeout` di `fiber.Config`
- `/etc/nginx/sites-enabled/radius.hotspotapp.net` (VPS) — tambah `keepalive_timeout 300; keepalive_requests 10000;`

## [2.52.8] — 2026-05-19
### Fixed
- **405 Method Not Allowed — audit & perbaiki semua HTTP method mismatch** — frontend mengirim method yang berbeda dari yang terdaftar di Go router; sekarang semua route mendukung method yang dikirim frontend
- `POST /api/telegram/settings` — tambah alias POST (was PUT only)
- `POST /api/admin/cloudflare-tunnel` — tambah alias POST (was PUT only)
- `POST /api/push/unsubscribe` — tambah alias POST (was DELETE only)
- `POST /api/push/agent-unsubscribe` — tambah alias POST (was DELETE only)
- `POST /api/push/technician-unsubscribe` — tambah alias POST (was DELETE only)
- `PUT /api/pppoe/users/status` — tambah alias PUT (was POST only)
- `PUT /api/pppoe/users/bulk-status` + `POST` alias — tambah (was GET only)
- `POST /api/admin/referrals/:id` — tambah alias POST (was PUT only)
- `POST /api/hotspot/vouchers/validate` — tambah alias POST (was GET only)
- `POST /api/network/olts/status` — tambah alias POST (was GET only)
- `POST /api/network/routers/status` — tambah alias POST (was GET only)
- `POST /api/settings/timezone` — implementasi handler baru `SetTimezone` yang simpan ke DB
- `POST /api/admin/users/:id/renewal` — tambah alias POST (was GET only)
- `POST /api/customer/wifi` — tambah alias POST (was PUT only)
### Added
- `SettingsExtHandler.SetTimezone` — handler baru untuk simpan timezone ke tabel company di DB
### Files
- `internal/api/router.go` — tambah 14 method alias untuk fix 405 errors
- `internal/api/handlers/settings_ext.go` — tambah `SetTimezone` POST handler

## [2.52.7] — 2026-05-19
### Changed
- **Audit & migrasi semua proxy ke Next.js → Go native** — 3 endpoint tersisa yang masih `proxyToNextJS` (POST/PATCH/PUT `/api/network/vpn-client`) sekarang ditangani sepenuhnya di Go. Next.js sekarang hanya melayani frontend, tidak ada lagi API call yang di-proxy.
- **POST /api/network/vpn-client (CreateVPNClient)** — Go sekarang: generate credentials, assign IP dari pool, connect ke MikroTik CHR via RouterOS API, buat PPP secret / WireGuard peer, buat Winbox NAT rule, simpan ke `vpn_clients`, auto-create NAS entry di tabel `nas`, generate `nasSetupScript` RouterOS, return credentials lengkap.
- **PATCH /api/network/vpn-client (PatchVPNClient)** — Go sekarang: validasi IP format & conflict, update PPP secret `remote-address` + NAT rules di MikroTik (best-effort non-fatal), update DB.
- **PUT /api/network/vpn-client (PutVPNClient)** — Go sekarang: toggle `isRadiusServer` flag, unset semua yang lain jika set, update DB.
### Added
- Dependency `github.com/go-routeros/routeros/v3` — RouterOS API client untuk connect ke MikroTik CHR
- `decryptVPNPassword()` — AES-256-CBC decrypt untuk password VPN server (format `ivHex:encHex`)
- `nextAvailableVPNClientIP()` — cari IP berikutnya yang tersedia dari pool VPN server
- `nextAvailableWinboxPort()` — cari Winbox port berikutnya (10000–10100)
- `generateX25519KeyPair()` — generate WireGuard X25519 key pair (base64)
- `buildNasSetupScript()` — generate RouterOS setup script untuk semua tipe VPN (L2TP, PPTP, SSTP, WireGuard)
### Removed
- `proxyToNextJS()` — dihapus, tidak ada lagi proxy ke Next.js dari Go backend
### Updated
- `prismaVpnServer` struct — tambah field `Username`, `Password`, `ApiPort`, `PoolStart`, `PoolEnd`
- `prismaVpnClient` struct — tambah field `ClientPrivateKey`
### Files
- `internal/api/handlers/network_vpn_ext_handler.go` — full migration 3 endpoints + helpers + hapus proxy
- `go.mod` / `go.sum` — tambah `github.com/go-routeros/routeros/v3 v3.0.1`

---

## [2.52.6] — 2026-05-19
### Fixed
- **DELETE VPN client 401 Unauthorized** — DELETE sebelumnya di-proxy ke Next.js. Next.js `getServerSession()` gagal karena cookie domain mismatch (cookie dari `https://radius.hotspotapp.net` tidak valid di `localhost:3000`). Solusi: DELETE sekarang ditangani langsung di Go, tidak proxy ke Next.js. Go sudah melakukan auth check sendiri untuk semua `/api/*` routes.
- **Hapus VPN client tidak hapus dari database** — Sekarang hapus dari tabel yang tepat: `vps_peers` (untuk VPS WireGuard/L2TP peer) atau `vpn_clients` (untuk MikroTik-managed client).
- **Cleanup file sistem saat hapus VPS peer** — WireGuard: hapus `[Peer]` block dari `/etc/wireguard/wg0.conf` + `wg set wg0 peer <pubkey> remove`. L2TP: hapus user line dari `/etc/ppp/chap-secrets`.
### Added
- `removeWGPeerFromConf(pubkey)` — helper hapus peer dari wg0.conf via regex
- `removeL2TPUserFromChapSecrets(username)` — helper hapus user dari chap-secrets
### Files
- `internal/api/handlers/network_vpn_ext_handler.go` — `DeleteVPNClient` rewrite + 2 helper functions

---

## [2.52.5] — 2026-05-19
### Fixed
- **SERVER VPN tampil "N/A" di list VPN client** — Go backend mengirim `vpnServerId = "__vps_wg__"` dan `"__vps_l2tp__"` untuk VPS peers, tapi frontend `resolveServer()` memeriksa `"__vps_wg_server__"` dan `"__vps_l2tp_server__"` (dengan suffix `_server`). Akibatnya server name selalu null → tampil "N/A", dan tombol "Lihat" tidak berfungsi (`if (!server) return`). Fix: ubah nilai `vpnServerId` di `ListVPNClients` sesuai yang diharapkan frontend.
### Files
- `internal/api/handlers/network_vpn_ext_handler.go` — `vpnServerId` VPS peers: `__vps_wg__` → `__vps_wg_server__`, `__vps_l2tp__` → `__vps_l2tp_server__`

---

## [2.52.4] — 2026-05-19
### Fixed
- **vps_peers table tidak pernah terbuat** — Root cause: GORM dengan `PrepareStmt: true` tidak bisa menjalankan DDL (`CREATE TABLE`) karena MySQL tidak support prepared statement untuk DDL. `runMigrations` sekarang pakai raw `sqlDB.Exec` (dari `db.DB()`) yang bypass PrepareStmt.
- **VPN Client list kosong (Total Klien: 0)** — `ListVPNClients` sekarang include semua tipe peer dari `vps_peers` (WireGuard dan L2TP), bukan hanya WireGuard.
- **L2TP VPS peer creation — username/password/vpnIp undefined** — `CreateL2TPPeer` sebelumnya stub yang hanya bind body ke struct kosong. Sekarang fully implemented: baca info server, cari IP berikutnya dari pool, generate credentials (username, password, nasSecret, apiUsername, apiPassword), tambah user ke `/etc/ppp/chap-secrets`, simpan ke `vps_peers`, return semua field yang dibutuhkan frontend.
### Added
- Helper `nextAvailableL2TPIP` — find next unused L2TP pool IP dari `vps_peers` table
### Files
- `internal/db/db.go` — `runMigrations` pakai `sqlDB.Exec` bukan `db.Exec`
- `internal/api/handlers/network_vpn_ext_handler.go` — `ListVPNClients` include semua vps_peers, `CreateL2TPPeer` implemented, tambah `nextAvailableL2TPIP`

---

## [2.52.3] — 2026-05-19
### Fixed
- **vps_peers table hilang / tidak persisten** — Migration via SQL file gagal karena BOM (byte-order mark) di file. Solusi: pindahkan pembuatan tabel ke Go code (`runMigrations` di `db.Init`) menggunakan `db.Exec("CREATE TABLE IF NOT EXISTS ...")`, sehingga tabel dibuat otomatis setiap startup — tidak perlu manual migration lagi.
### Files
- `internal/db/db.go` — Tambah fungsi `runMigrations` yang membuat tabel `vps_peers` via raw SQL saat startup

---

## [2.52.2] — 2026-05-19
### Fixed
- **VPS WireGuard peer tidak muncul di VPN Client list** — `CreateWGPeer` sebelumnya adalah stub yang tidak generate keypair, tidak assign IP, dan menyimpan ke tabel `vps_peers` yang belum ada. Rewrite penuh: generate WireGuard keypair (`wg genkey`/`wg pubkey`), alokasi IP dari pool, tambah `[Peer]` ke wg0.conf, apply live via `wg set`, generate nasSecret/apiUsername/apiPassword, simpan ke DB.
- **Script NAS IP undefined, API User api-undefined, CLIENT_PRIVATE_KEY placeholder** — `CreateWGPeer` tidak return `vpnIp`, `apiUsername`, `apiPassword`, `clientPrivateKey`, dll. Sekarang return semua field yang dibutuhkan frontend.
- **vps_peers table tidak ada** — Dibuat via migration SQL baru.
- **WG VPS peers tidak muncul di list** — `ListVPNClients` sekarang include entries dari `vps_peers` (type=wireguard), dimapping ke format VpnClient dengan `vpnServerId = "__vps_wg__"`.
### Added
- Migration `prisma/migrations/20260520_add_vps_peers_table.sql` — Create table `vps_peers` dengan kolom credentials
- Helper functions: `nextAvailableWGIP`, `wgRandomHex`, `wgRandomAlphanumeric`
- Struct `vpnClientResponse` — unified response format untuk `vpn_clients` dan `vps_peers`
### Files
- `internal/api/handlers/network_vpn_ext_handler.go` — Rewrite `CreateWGPeer`, update `ListVPNClients`, tambah struct + helpers baru
- `prisma/migrations/20260520_add_vps_peers_table.sql` — New migration

---

## [2.52.1] — 2026-05-20
### Fixed
- **VPN Client list masih kosong setelah proxy fix** — Root cause: `getServerSession` di Next.js route handler tidak bisa validasi NextAuth cookie karena perbedaan nama cookie (`__Secure-next-auth.session-token` dari HTTPS browser vs `next-auth.session-token` yang dicari di HTTP localhost). Solusi: `ListVPNClients` sekarang **baca langsung dari DB** (`vpn_clients` table via GORM + struct baru `prismaVpnClient`/`prismaVpnServer`) tanpa proxy ke Next.js, sehingga tidak perlu re-validasi session lagi.
- **Proxy POST fix** — Tambah `req.Host = "localhost"` dan strip `__Secure-` prefix dari cookie sebelum forward ke Next.js, agar `getServerSession` bisa mengenali session cookie untuk CREATE/PATCH/PUT/DELETE.
### Files
- `internal/api/handlers/network_vpn_ext_handler.go` — Tambah struct `prismaVpnClient`, `prismaVpnServer`; rewrite `ListVPNClients` baca DB langsung; fix `proxyToNextJS` Host header + cookie prefix

---

## [2.52.0] — 2026-05-19
### Fixed
- **VPN Client tidak muncul setelah ditambahkan** — Go handler `ListVPNClients`/`CreateVPNClient` menulis ke tabel `vpn_client_configs` yang tidak ada (tidak pernah di-migrate Prisma). Tabel yang benar adalah `vpn_clients`. Solusi: Go sekarang **proxy** semua request `/api/network/vpn-client` ke Next.js (port 3000) yang sudah punya logika lengkap (credential generation, MikroTik connection, Prisma).
### Added
- **Proxy handler** — `proxyToNextJS()` helper di `NetworkVPNHandler`: forward Cookie + Authorization header agar session auth tetap valid
- **Routes PATCH/PUT/DELETE** untuk `/api/network/vpn-client` yang sebelumnya tidak ada di Go router (menyebabkan 404 saat update IP / toggle RADIUS / hapus client)
### Files
- `internal/api/handlers/network_vpn_ext_handler.go` — Tambah `proxyToNextJS`, rewrite `ListVPNClients`/`CreateVPNClient`, tambah `PatchVPNClient`, `PutVPNClient`, `DeleteVPNClient`
- `internal/api/router.go` — Register `api.Patch/Put/Delete("/network/vpn-client", ...)`

---

## [2.51.9] — 2026-05-19
### Fixed
- **Sidebar admin terlalu panjang** — Semua 7 kategori menu sebelumnya selalu terbuka (`useState(true)`). Sekarang kategori collapse by default; hanya kategori yang berisi halaman aktif yang auto-expand.
### Files
- `src/app/admin/AdminClientLayout.tsx` — `CategoryItem`: `useState(true)` → `useState(hasActiveItem)`

---

## [2.51.8] — 2026-05-19
### Fixed
- **405 Method Not Allowed saat simpan konfigurasi WireGuard / L2TP** — `PATCH /api/network/vps-wg-peer` dan `PATCH /api/network/vps-l2tp-peer` tidak terdaftar di Go router (hanya ada di Next.js route handler lama yang sekarang tidak dipakai karena semua `/api/*` diproxy ke Go). `PATCH` juga tidak ada di daftar `AllowMethods` CORS sehingga preflight OPTIONS gagal.
### Added
- **Go handler `PatchWGServerConfig`** — `PATCH /api/network/vps-wg-peer`: update `poolStart`, `poolEnd`, `gatewayIp` di `wg-server-info.json`. Jika `gatewayIp` berubah, otomatis update `Address =` di `wg0.conf`, update PostUp/PostDown iptables, lalu restart WireGuard interface (`wg-quick down/up`).
- **Go handler `PatchL2TPServerConfig`** — `PATCH /api/network/vps-l2tp-peer`: update `poolStart`, `poolEnd`, `gateway` di `l2tp-server-info.json`, restart xl2tpd + reload ipsec, pastikan iptables rules untuk `ppp+`.
- **CORS PATCH** — Tambah `"PATCH"` ke `AllowMethods` di CORS middleware.
### Files
- `internal/api/handlers/network_vpn_ext_handler.go` — Tambah `PatchWGServerConfig` dan `PatchL2TPServerConfig`
- `internal/api/router.go` — Register `api.Patch("/network/vps-wg-peer", ...)`, `api.Patch("/network/vps-l2tp-peer", ...)`, dan `"PATCH"` di CORS AllowMethods

---

## [2.51.7] — 2026-05-19
### Fixed
- **updater.sh menyebabkan 502 setelah update** — Serangkaian bug yang saling terkait: (1) `local SVC_FILE` di luar function → `set -e` exit dini sebelum Go build & PM2 restart; (2) rsync `--delete` hapus `logs/` & `bin/` → salfanet-api gagal start dengan `status=226/NAMESPACE`; (3) `rm -rf .next` sebelum build → jika build gagal tidak ada fallback; (4) PM2 processes tidak pernah di-restore jika script exit di tengah jalan.
### Changed
- **updater.sh self-healing** — Tambah `_ensure_services_up()` trap (`trap ... EXIT`) yang selalu memulihkan salfanet-api dan PM2 salfanet-radius jika script exit abnormal. PM2 sekarang di-start dengan build lama segera setelah rsync (downtime minimal selama update). Setelah build selesai, pm2 reload + verifikasi online + self-heal otomatis jika masih failed.
- **updater.sh no rm .next** — Build incremental (tidak wipe .next sebelum build), sehingga jika build gagal site tetap berjalan dengan versi sebelumnya.
- **updater.sh exclude logs/ bin/ dari rsync** — `--exclude='logs/' --exclude='bin/'` ditambahkan ke rsync supaya direktori tidak dihapus oleh `--delete`.
- **updater.sh PM2 orphan process** — Tambah `fuser -k 3000/tcp` sebelum PM2 start untuk mencegah EADDRINUSE jika ada orphan node process.
### Files
- `vps-install/updater.sh` — Multiple fixes: trap, no rm .next, mkdir logs/bin, rsync exclude, PM2 self-heal & verify

---

## [2.51.6] — 2026-05-20
### Fixed
- **WireGuard/L2TP UI menampilkan "belum terinstall" meski sudah terinstall** — Root cause: nginx route semua `/api/*` ke Go backend (port 8080), bukan Next.js. Go handler `ListWGPeers` hanya return list DB peers (bukan server info), dan `GetVPSL2TPInfo` return stub `{enabled: false}`. Frontend membaca `data.installed` yang tidak ada → dianggap false. Fix: implementasi Go yang membaca `/etc/wireguard/wg-server-info.json` dan `/etc/salfanet/l2tp/l2tp-server-info.json`, dengan fallback parse `wg0.conf`/`xl2tpd.conf` + `ipsec.secrets` jika file JSON tidak ada (auto-recovery).
- **updater.sh SOURCE_DIR mismatch** — Default `/root/salfanet-radius-go` tidak cocok dengan path clone README (`/root/salfanet-radius`). Fix: auto-detect `/root/salfanet-radius/.git` terlebih dahulu sebelum fallback ke `-go` suffix.
### Added
- **WG server info fallback detection** — Jika `wg-server-info.json` tidak ada, Go handler parse `wg0.conf` untuk ListenPort/Address/subnet, baca pubkey dari file key atau `wg show wg0 public-key`, dan re-write JSON untuk akses berikutnya.
- **L2TP server info fallback detection** — Jika `l2tp-server-info.json` tidak ada, Go handler parse `xl2tpd.conf` untuk ip range/local ip, baca PSK dari `/etc/salfanet/l2tp/ipsec.psk` atau `/etc/ipsec.secrets`.
- **Live WG peers** — GET `/api/network/vps-wg-peer` kini include peers dari `wg show wg0 dump` (endpoint, allowedIps, dll).
### Files
- `internal/api/handlers/network_vpn_ext_handler.go` — Implementasi real untuk `ListWGPeers` dan `GetVPSL2TPInfo`; tambah `readWGServerInfo()` dan `readL2TPServerInfo()` helper
- `vps-install/updater.sh` — Fix SOURCE_DIR auto-detect `/root/salfanet-radius` sebelum fallback ke `/root/salfanet-radius-go`

---

## [2.51.5] — 2026-05-19
### Fixed
- **install-pm2 cleanup pkill bunuh installer sendiri** — Root cause PM2 tidak auto-start: `pkill -9 -f "/root/salfanet-radius"` di `cleanup_pm2_processes()` cocok dengan command line bash installer (`bash -c 'cd /root/salfanet-radius && ...'`), membunuh screen session sebelum `pm2 start` berjalan. Fix: hapus pkill berbasis path source dir, ganti dengan pkill spesifik Node.js (`node.*server.js`, `node.*cron-service.js`, dll). Juga: skip `sudo su - root` untuk cleanup saat APP_USER=root (redundan + berpotensi interference).
- **install-pm2 PM2 systemd supervision** — Tambah `systemctl start pm2-root` setelah `pm2 startup systemd` agar PM2 daemon berjalan di bawah systemd (bukan screen). PM2 processes kini auto-start tanpa intervensi manual.
### Files
- `vps-install/install-pm2.sh` — Hapus `pkill -f "/root/salfanet-radius"` yang bunuh installer; skip `sudo su - root` di cleanup; tambah `systemctl start pm2-root`

---

## [2.51.4] — 2026-05-19
### Fixed
- **VPS install test — sistem production ready** — Full fresh install test sukses: build OK, Go API healthy, nginx routing OK, FreeRADIUS aktif, MySQL aktif, semua PM2 processes online (salfanet-radius, salfanet-cron, salfanet-wa). Installer terbukti berjalan end-to-end tanpa error.
- **system/page.tsx duplikat deklarasi** — Build gagal karena `interface SystemInfo`, `formatUptime`, `InfoCard`, `CmdBlock` dideklraasikan dua kali. Blok duplikat dihapus.
### Files
- `src/app/admin/system/page.tsx` — Hapus blok duplikat (~52 baris) yang menyebabkan TypeScript build error

---

## [2.51.3] — 2026-05-18
### Fixed
- **install-nginx.sh missing /api/push/send** — Fresh install tidak punya `location = /api/push/send` → Next.js, sehingga Push Notifikasi tetap 405. Ditambahkan ke kedua heredoc function (`_proxy_locations` dan `_proxy_locations_https_domain`).
### Files
- `vps-install/install-nginx.sh` — Tambah location = /api/push/send → Next.js di kedua proxy helper

---

## [2.51.2] — 2026-05-19
### Changed
- **Cleanup repo — hapus folder dev-only dari GitHub** — `OLT-ZTE-C320-Provisioning-main/` (118 file proyek Laravel terpisah), `update-olt-opt/` (patch script Go), dan `update-qris/` (patch script Next.js) dihapus dari git tracking. Tidak relevan dengan production salfanet-radius.
### Files
- `.gitignore` — Tambah exclusion untuk `OLT-ZTE-C320-Provisioning-main/`, `update-olt-opt/`, `update-qris/`

---

## [2.51.1] — 2026-05-19
### Fixed
- **Push Notifikasi 405 pada GET /api/push/send** — Halaman Push Notifikasi memanggil `GET /api/push/send?action=stats` dan `GET /api/push/send?limit=30` untuk memuat stats dan riwayat broadcast, namun Go backend hanya mendaftarkan POST untuk route ini sehingga GET mengembalikan 405. Perbaikan: tambahkan lokasi nginx `= /api/push/send` yang meneruskan semua method ke Next.js (port 3000), di mana GET dan POST handler sudah ada di `route.ts`.
### Files
- `production/nginx-radius.hotspotapp.net.conf` — Tambah `location = /api/push/send` → Next.js (3000)

---

## [2.51.0] — 2026-05-19
### Changed
- **Konsolidasi manajemen teknisi — Option B** — Teknisi kini hanya dikelola via "Kelola Teknisi" menggunakan OTP WhatsApp. Role `TECHNICIAN` dihapus dari dropdown Admin & Role. Login via username/password untuk teknisi dinonaktifkan.
- **Login portal teknisi ganti ke OTP** — Halaman `/technician/login` diganti dari form username+password ke form nomor HP + kode OTP WhatsApp (2-step). Teknisi harus sudah terdaftar di sistem (tidak ada auto-create).
- **request-otp: hapus auto-create teknisi** — Sebelumnya siapa saja bisa membuat akun teknisi cukup dengan mengirim nomor HP. Kini nomor HP harus sudah terdaftar di tabel `technician`, atau API mengembalikan 404.
### Files
- `src/app/admin/management/page.tsx` — Hapus `TECHNICIAN` dari array `ROLES`
- `src/app/technician/login/page.tsx` — Ganti form login ke 2-step OTP (phone → OTP code)
- `src/app/api/technician/auth/login/route.ts` — Disabled, return 410 Gone
- `src/app/api/technician/auth/request-otp/route.ts` — Hapus auto-create technician
- `src/app/api/technician/auth/session/route.ts` — Hapus cabang `admin_user`, hanya OTP technician
- `src/app/api/technician/profile/route.ts` — Hapus cabang `admin_user`
- `src/app/api/technician/customers/route.ts` — Hapus `isAdminUser`, require routerId/areaId untuk semua teknisi
- `src/app/api/technician/customers/create/route.ts` — Hapus cabang `admin_user`
- `src/app/api/technician/work-orders/route.ts` — Hapus `isAdminUser`, semua teknisi lihat unassigned + miliknya
- `src/app/api/technician/{form-data,genieacs,genieacs/devices,genieacs/devices/[deviceId],isolated,monitor,tasks,sessions,upload,offline,tickets}/route.ts` — Hapus blok `admin_user` dari `verifyTechnician`
- `src/app/api/push/technician-subscribe/route.ts` — Hapus blok `admin_user`, hapus `upsertAdminPushSubscription`
- `src/app/api/push/technician-unsubscribe/route.ts` — Hapus blok `admin_user`, hapus `removeAdminPushSubscription`


### Fixed
- **TypeError di /admin/tickets dan /admin/tickets/categories** — Go handler `Stats` mengembalikan format flat `{open, resolved, pending}` tapi frontend mengharapkan `{byStatus: {open, inProgress, ...}, byPriority: {...}}` → crash `stats.byStatus.open`. Handler `ListCategories` mengembalikan `{success, categories:[]}` tapi frontend mengharapkan array langsung → crash `g.filter is not a function`. Fix: update kedua handler sesuai interface frontend.
### Files
- `internal/api/handlers/ticket_ext.go` — `Stats` return nested `byStatus`/`byPriority`/`unassigned`; `ListCategories` return array langsung (bukan wrapped object)

## [2.50.9] — 2026-05-18
### Fixed
- **GET /api/tickets/stats dan /api/tickets/categories 404** — Route extension (`ticketExtH`) didaftarkan di level `api.Get(...)` setelah group `tickets` yang sudah punya wildcard `/:id`. Wildcard menangkap request lebih dulu sehingga handler mencari tiket dengan id="stats"/"categories" → 404. Fix: pindah semua ticket extension routes ke dalam group `tickets`, sebelum `/:id`, agar Fiber match spesifik dulu.
### Files
- `internal/api/router.go` — Ticket extension routes (`/stats`, `/categories`, `/messages`, `/dispatch`) dipindah ke dalam `tickets` group sebelum wildcard `/:id`

## [2.50.8] — 2026-05-18
### Fixed
- **TypeError: Cannot read toLocaleString of undefined di /admin/laporan/analitik** — Root cause: Go backend mengembalikan format summary berbeda dari yang diharapkan halaman (`activeUsers` bukan `currentActiveUsers`, tidak ada `monthlyData`, `avgArpu`, dll). Fix: (1) tambah null guards di `fmtIDR`, `fmtIDRFull`, dan `currentActiveUsers` access; (2) tambah nginx `location = /api/admin/analytics` → Next.js port 3000 agar route Next.js yang detail (churn, ARPU, monthly data) digunakan, bukan Go handler yang simplified.
### Files
- `src/app/admin/laporan/analitik/page.tsx` — Null guards: `fmtIDR`/`fmtIDRFull` handle undefined/null, `AnalyticsSummary` interface semua field optional, `currentActiveUsers` fallback ke `activeUsers`
- `production/nginx-radius.hotspotapp.net.conf` — Tambah `location = /api/admin/analytics` → port 3000 (Next.js) sebelum catch-all `/api/` → Go


### Fixed
- **Error 521 Cloudflare** — Setelah deploy config HTTP-only, Cloudflare Full SSL mode tidak bisa connect ke port 443 (tidak listening) → 521. Fix: tambah `listen 443 ssl` + `listen [::]:443 ssl` dengan snakeoil cert ke nginx config agar support kedua mode (Flexible dan Full).
- **CSP violation Leaflet CSS (final fix)** — Root cause: nginx `sites-enabled/` punya dua config konflik untuk `radius.hotspotapp.net`. Config aktif tidak punya CSP, sehingga CSP dari Next.js (build lama tanpa cdnjs) yang terpakai. Fix: deploy `nginx-radius.hotspotapp.net.conf` ke `sites-available/radius.hotspotapp.net`, hapus `sites-enabled/salfanet-radius`, reload nginx. CSP sekarang dari nginx dengan `proxy_hide_header Content-Security-Policy` + `add_header` baru yang mencakup `https://cdnjs.cloudflare.com`.
### Files
- `production/nginx-radius.hotspotapp.net.conf` — Tambah `listen 443 ssl` + snakeoil cert, CSP dengan cdnjs di `location /`

## [2.50.6] — 2026-05-19
### Fixed
- **CSP violation Leaflet CSS** — nginx HTTPS server block tidak menyertakan `https://cdnjs.cloudflare.com` di `style-src`, menyebabkan Leaflet CSS diblokir. Fix: tambah `https://cdnjs.cloudflare.com` ke `style-src` dan `font-src` di nginx CSP. Deploy langsung ke VPS.
- **Light mode conflicts di halaman ODC & ODP** — Elemen cyberpunk neon (background blob, button `bg-[#00f7ff]`, icon `text-[#00f7ff]`, mobile card border `border-[#bc13fe]/20`) tampil di kedua tema tanpa fallback. Fix: neon background blobs dibungkus `hidden dark:block`, button & icon diberi fallback light mode (`bg-blue-600 dark:bg-[#00f7ff]`, `text-cyan-500 dark:text-[#00f7ff]`), card border `border-border dark:border-[#bc13fe]/20`.
- **Warna teks & background tidak konsisten di halaman Manajemen Fiber** — Fiber Cables, Fiber Cores, Fiber Joint Closures menggunakan `text-gray-500` (tanpa dark variant) → sulit dibaca di dark mode. Stat card menggunakan `bg-white dark:bg-gray-900` bukan CSS variable `bg-card`. Fix: ganti ke `text-muted-foreground` dan `bg-card border-border` di semua fiber pages.
- **Theme inconsistency di halaman Topologi** — unified-map, diagrams, trace, splice-points: `text-gray-500` dan beberapa `bg-white dark:bg-gray-900` diganti ke `text-muted-foreground` dan `bg-card`.
### Files
- `production/nginx-salfanet-radius.conf` — Tambah `https://cdnjs.cloudflare.com` ke `style-src` dan `font-src` di CSP header HTTPS block
- `/etc/nginx/sites-available/salfanet-radius` (VPS) — Sama, reload sukses
- `src/app/admin/network/odcs/page.tsx` — Fix light mode: background blobs `hidden dark:block`, button, icon, mobile card borders
- `src/app/admin/network/odps/page.tsx` — Fix light mode: sama seperti ODC
- `src/app/admin/network/fiber-cables/page.tsx` — `text-gray-500` → `text-muted-foreground`, `bg-white dark:bg-gray-900` → `bg-card`, `border dark:border-gray-800` → `border-border`
- `src/app/admin/network/fiber-cores/page.tsx` — `text-gray-500` → `text-muted-foreground`, fix borders
- `src/app/admin/network/fiber-joint-closures/page.tsx` — `text-gray-500` → `text-muted-foreground`
- `src/app/admin/network/unified-map/page.tsx` — `text-gray-500` → `text-muted-foreground`, `bg-white dark:bg-gray-900` → `bg-card`
- `src/app/admin/network/diagrams/page.tsx` — `text-gray-500` → `text-muted-foreground`
- `src/app/admin/network/trace/page.tsx` — `text-gray-500` → `text-muted-foreground`
- `src/app/admin/network/splice-points/page.tsx` — `text-gray-500` → `text-muted-foreground`, `bg-white dark:bg-gray-900` → `bg-card`

## [2.50.5] — 2026-05-18
### Fixed
- **Error Loading Map — 401 pada `/api/customers/with-location`** — Cloudflare edge node berbeda dapat memproses request `/api/customers/with-location` tanpa meneruskan cookie yang benar ke Go backend, menyebabkan `CombinedAuthMiddleware` gagal validasi. Fix: tambahkan `location /api/customers/` di nginx yang meroute langsung ke Next.js (port 3000) sebelum catch-all `/api/`. Next.js menggunakan `getServerSession` (server-side) yang tidak bergantung pada internal call Go → NextAuth, sehingga auth selalu berhasil. Response format sudah cocok (`{success: true, data: [...], count: N}`) dengan komponen `UnifiedNetworkMap`.
### Files
- `production/nginx-salfanet-radius.conf` — Tambah `location /api/customers/` → port 3000 di semua 3 server block sebelum `location /api/` catch-all
- `/etc/nginx/sites-enabled/salfanet-radius` (VPS) — Sama: tambah `location /api/customers/` → port 3000 di HTTP dan HTTPS block

## [2.50.4] — 2026-05-18
### Fixed
- **TypeError: (e.users || e).filter is not a function di halaman Network Map** — Go handler `ListUsersWithFilters` mengembalikan `"users": null` (nil slice → JSON null) saat tidak ada data, sehingga `(data.users || data).filter(...)` crash karena object tidak punya `.filter`. Fix: (1) Go: gunakan `make([]models.PppoeUser, 0)` agar nil slice menjadi `[]` bukan `null`. (2) Frontend: ganti `(data.users || data).filter(...)` → `(data.users ?? []).filter(...)`.
- **CSP violation Leaflet CSS** — `style-src` di `next.config.ts` sudah mencakup `https://cdnjs.cloudflare.com` (konfirmasi sudah benar).
### Files
- `internal/api/handlers/pppoe_ext.go` — `ListUsersWithFilters`: `var users []models.PppoeUser` → `users := make([]models.PppoeUser, 0)`
- `src/app/admin/network/map/page.tsx` — line 373: `(data.users || data).filter(...)` → `(data.users ?? []).filter(...)`

## [2.50.3] — 2026-05-18
### Fixed
- **GenieACS 400 error merah di console** — Semua handler GenieACS yang mengembalikan 400 saat GenieACS belum dikonfigurasi sekarang mengembalikan **HTTP 200** dengan `{success: false, notConfigured: true}`. Browser tidak lagi mencatat error merah di DevTools console ketika GenieACS belum di-setup.
- **Tasks page** — deteksi `notConfigured` kini menggunakan `data.notConfigured` dari response body (bukan `response.status === 400`) sesuai perubahan Go handler.
### Files
- `internal/api/handlers/genieacs.go` — tambah helper `notConfiguredErr()`, replace 4 `c.Status(400)` setelah `getCredentials()`
- `internal/api/handlers/genieacs_ext.go` — replace 21 `c.Status(400)` setelah `getCredentials()`
- `src/app/admin/genieacs/tasks/page.tsx` — cek `data.notConfigured` bukan `response.status`

## [2.50.2] — 2026-05-18
### Fixed
- **GenieACS parameter-config crash** — Go handler `ListVirtualParameters` mengembalikan key `"parameters"` tapi frontend membaca `data.data` → `TypeError: Cannot read properties of undefined (reading 'filter')`. Diperbaiki: key diubah ke `"data"` + tambah optional chaining `(data.data ?? []).filter(...)`.
- **Tasks page GenieACS not-configured** — saat `/api/genieacs/tasks` mengembalikan 400 (GenieACS belum dikonfigurasi), halaman sekarang menampilkan banner peringatan dengan link ke pengaturan, bukan diam-diam tampilkan list kosong.
### Files
- `internal/api/handlers/settings_genieacs.go` — key `"parameters"` → `"data"`
- `src/app/admin/genieacs/parameter-config/page.tsx` — optional chaining `(data.data ?? []).filter(...)`
- `src/app/admin/genieacs/tasks/page.tsx` — banner "GenieACS belum dikonfigurasi" saat 400

## [2.50.1] — 2026-05-18
### Added
- **Simulasi pembayaran QRIS (testing)** — Endpoint `POST /api/payment/qris-test` (admin-only) + UI "🧪 Simulasi Pembayaran QRIS" di tab QRIS Mandiri: masukkan Order ID → server langsung tandai invoice PAID + extend subscription tanpa perlu HP Android.
- **Android package ID baru** — APK diubah dari `id.salfanet.qrislistener` → `net.hotspotapp.qrislistener` agar bisa di-install bersamaan di satu HP dengan versi salfanet PHP lama.
- **Label app dibedakan** — Nama app di drawer HP sekarang "QRIS Listener (Radius)" vs "QRIS Listener" versi lama.
### Changed
- **Suara notifikasi Android** — Channel alert dinaikkan ke `IMPORTANCE_HIGH`, ringtone diganti `TYPE_ALARM` (lebih keras), tambah vibration pattern `0,300,200,300,200,500` + LED hijau berkedip.
- **APK versi 1.1.0** (versionCode 30) dengan package baru.
### Files
- `internal/api/handlers/payment_handler.go` — Tambah `QrisTest` handler
- `internal/api/router.go` — Route `POST /api/payment/qris-test` (admin auth)
- `src/app/admin/payment-gateway/page.tsx` — UI simulasi testing QRIS
- `public/downloads/qris-listener.apk` — APK baru package `net.hotspotapp.qrislistener` v1.1.0

## [2.50.0] — 2026-05-19
### Added
- **Android QRIS Listener — Deteksi pembayaran otomatis** — Integrasi dengan app Android `QrisListener` (NotificationListenerService) yang menangkap notifikasi e-wallet (DANA, GoPay, BRImo, ShopeePay, BCA, Mandiri) dan mengirim jumlah ke server via webhook.
- **Unique amount trick** — Setiap QRIS invoice mendapat nominal unik (base + suffix 1-999 dari MD5 invoiceId) sehingga server bisa mencocokkan pembayaran masuk ke invoice yang tepat.
- **Model `QrisPending`** — Tabel `qris_pendings` menyimpan state transaksi QRIS mandiri: `uniqueAmount`, `status` (pending/paid/expired), `expiresAt` (15 menit).
- **Field `QrisDeviceKey` di Company** — Kunci otentikasi untuk Android listener; jika diisi, QRIS berfungsi auto-konfirmasi.
- **`POST /api/payment/qris-notify`** — Endpoint publik (auth via device_key) yang menerima notifikasi dari Android, mencocokkan `uniqueAmount`, dan otomatis menandai invoice PAID + extend subscription.
- **`GET /api/payment/qris-status?orderId=`** — Polling endpoint publik untuk frontend mengetahui status pembayaran QRIS own (pending/paid/expired).
- **Payment Gateway UI** — Tambah section "Device Key — Android Listener" di tab QRIS Mandiri dengan input key + tombol Generate; tampilkan status listener (hijau = aktif, kuning = manual).
- **Pay page UI** — Tampilkan nominal unik dengan peringatan "Transfer TEPAT Rp X.XXX (jangan dibulatkan)"; auto-polling `qris-status` jika listener aktif; pesan berbeda untuk listener aktif vs manual; countdown 15 menit untuk QRIS own.
### Files
- `internal/db/models/models.go` — Tambah `QrisDeviceKey` di Company; struct `QrisPending`
- `internal/lib/qris/qris.go` — Tambah `GenerateUniqueAmount(baseAmount, invoiceId)`
- `internal/api/handlers/payment_handler.go` — Update `CreatePayment` (unique amount + QrisPending); tambah `QrisNotify`, `QrisStatus`, helper `addMonths`, `formatAmount`
- `internal/api/handlers/invoices_ext.go` — `GetByToken` return `hasListener` di qrisOwn
- `internal/api/handlers/misc_handler.go` — `CheckIsolationGlobal` return `hasListener` di qrisOwn
- `internal/api/router.go` — Route baru: `POST /api/payment/qris-notify`, `GET /api/payment/qris-status`
- `src/app/admin/payment-gateway/page.tsx` — Tambah Device Key UI + Generate button
- `src/app/pay/[token]/page.tsx` — Unique amount display + polling qris-status + UI listener vs manual

## [2.49.0] — 2026-05-18
### Added
- **QRIS Mandiri (self-hosted QRIS)** — Dukungan pembayaran QRIS tanpa pihak ketiga menggunakan QRIS statis dari bank/merchant yang dikonversi ke QRIS dinamis (EMVCo TLV + CRC-16/CCITT-FALSE).
- **Go QRIS library** — `internal/lib/qris/qris.go`: `StaticToDynamic(staticQris, amount)` mengubah QRIS statis → dinamis dengan nominal; `ValidateQris()` untuk validasi.
- **`POST /api/payment/create` → `qris_own` gateway** — Endpoint baru menerima field `gateway: "qris_own"`, query config QRIS dari company, generate `qrString` dinamis, return `{success, orderId, qrString, isQrisOwn: true}`.
- **`GET /api/invoices/by-token/:token` enriched** — Kini mengembalikan `{invoice, paymentGateways, qrisOwn, company}` sehingga halaman `/pay/[token]` bisa menampilkan gateway + QR code.
- **`GET /api/pppoe/users/check-isolation` rewrite** — Sebelumnya hanya return `{isolatedCount}` tanpa melihat query params. Sekarang mendukung `?username=` dan `?ip=`, mengembalikan data user lengkap + unpaid invoices + available gateways + qrisOwn untuk halaman `/isolated`.
- **Company model QRIS fields** — `qrisStaticCode (TEXT)`, `qrisMerchantName (VARCHAR)`, `qrisEnabled (BOOL)` di Go model + Prisma schema.
- **Frontend: halaman `/pay/[token]`** — Tambah tampilan QR code QRIS Mandiri menggunakan `qrcode.react`, polling status Duitku/Tripay, tombol pilih gateway.
- **Frontend: halaman `/isolated`** — Tambah opsi pembayaran QRIS Mandiri dengan QR code untuk user yang diisolasi.
- **Frontend: Admin → Payment Gateway** — Section khusus QRIS Mandiri (self-hosted) terpisah dari gateway pihak ketiga.
- **Frontend: Admin → Settings → Company** — Tambah field konfigurasi QRIS: textarea QRIS statis, merchant name, toggle enable/disable.
### Files
- `internal/lib/qris/qris.go` — NEW: Go QRIS library (TLV parser, CRC-16, StaticToDynamic)
- `internal/db/models/models.go` — Tambah 3 field QRIS ke struct `Company`
- `internal/api/handlers/company.go` — Default `qrisEnabled: false` di GetCompany
- `internal/api/handlers/payment_handler.go` — Tambah `qris_own` branch di `CreatePayment`
- `internal/api/handlers/invoices_ext.go` — `GetByToken` kini return gateways + qrisOwn + company
- `internal/api/handlers/misc_handler.go` — `CheckIsolationGlobal` rewrite: handle `?username=`/`?ip=`, return full user data
- `prisma/schema.prisma` — Tambah 3 field QRIS ke model `company`
- `src/lib/qris.ts` — NEW: TypeScript QRIS library (untuk referensi / SSR)
- `src/app/pay/[token]/page.tsx` — Tambah QRIS display + gateway selector
- `src/app/isolated/page.tsx` — Tambah QRIS payment option
- `src/app/admin/payment-gateway/page.tsx` — Section QRIS Mandiri
- `src/app/admin/settings/company/page.tsx` — QRIS config fields
- `package.json` — Tambah dependency `qrcode.react`


### Fixed
- **`GET /api/olt/monitoring 404`** — Go router tidak punya route `/monitoring`. Fix: tambah handler `MonitoringList` (GET) + `MonitoringPoll` (POST) di `OLTHandler`. Handler list semua OLT dengan filter `search`/`status`, tambah field `unresolvedAlerts` (count) per OLT.
- **`GET /api/olt/alerts 404`** — Go router tidak punya route global `/alerts`. Fix: tambah handler `ListAllAlerts` (GET) dengan filter `resolved`/`severity`/`type`/`limit`, serta preload data OLT dan ONU (dengan customer). Tambah `ResolveAlert` (PUT `/api/olt/alerts/:id`) untuk mark alert as resolved.
### Files
- `internal/api/handlers/olt.go` — Tambah 4 metode: `MonitoringList`, `MonitoringPoll`, `ListAllAlerts`, `ResolveAlert`
- `internal/api/router.go` — Tambah 4 routes sebelum `/:id`: GET/POST `/monitoring`, GET `/alerts`, PUT `/alerts/:id`


### Fixed
- **`admin/hotspot/voucher` crash `Cannot read properties of undefined (reading 'total')` & `(reading 'activated')`** — SSE handler kirim `{"count":N}` tapi frontend ekspek `{"stats":{total,waiting,active,expired,totalValue},"changes":{activated,expired}}`. `setStats(data.stats)` → `setStats(undefined)` → render crash. Fix: Go SSE handler kirim format lengkap dengan count per status (UNUSED→waiting, ACTIVE→active, EXPIRED/USED→expired). Fix frontend: tambah null check `data?.stats` dan `data?.changes` sebagai guard.
### Files
- `internal/api/handlers/settings_genieacs.go` — SSEVoucherUpdates kirim format stats yang benar
- `src/app/admin/hotspot/voucher/page.tsx` — handleSSEMessage pakai optional chaining untuk null safety

## [2.48.5] — 2026-05-17
### Fixed
- **`admin/pppoe/profiles` & `admin/pppoe/registrations` — `GET /api/pppoe/profiles/sync-mikrotik` 405 Method Not Allowed** — Frontend memanggil GET ke endpoint yang hanya tersedia sebagai POST (untuk sync, bukan list). Fix: ganti URL ke `GET /api/network/routers` yang sudah ada dan return raw array Router.
- **`[SSE] Connection error` loop di konsol** — Handler `SSEVoucherUpdates` langsung menutup koneksi setelah kirim 1 pesan → browser EventSource trigger `onerror` → reconnect setiap 3s → infinite error loop. Fix: ubah handler jadi long-lived SSE stream menggunakan `fasthttp.StreamWriter`, kirim `event: connected` + `event: voucher-stats` lalu heartbeat `: heartbeat` setiap 30s. Tambah header `X-Accel-Buffering: no` agar nginx tidak buffer SSE.
### Files
- `internal/api/handlers/settings_genieacs.go` — SSEVoucherUpdates jadi proper long-lived SSE stream
- `src/app/admin/pppoe/profiles/page.tsx` — loadRouterList & handleSyncMikrotik modal gunakan `/api/network/routers`
- `src/app/admin/pppoe/registrations/page.tsx` — useEffect gunakan `/api/network/routers`

## [2.48.4] — 2026-05-17
### Fixed
- **`admin/hotspot/template` crash `u.map is not a function`** — Go `GET /api/voucher-templates` return `{ success: true, templates: [...] }`. Frontend langsung `.map()` di response → crash. Fix: handler return raw array.
- **`admin/hotspot/voucher` crash `t.filter is not a function`** — Sama, handler return wrapped object bukan array. Fix sama: return raw array.
- **`admin/settings/referral` crash `Cannot read properties of undefined (reading 'enabled')`** — Go `PUT /api/admin/referrals/config` return `{ success: true, message: "..." }` tanpa field `config`. Frontend `setConfig(data.config)` → `setConfig(undefined)` → render crash. Fix: handler return `config` dalam response. GET juga fix: baca dari Company DB, return semua 5 fields dengan default.
### Files
- `internal/api/handlers/voucher_templates.go` — List return raw array (bukan wrapped object)
- `internal/api/handlers/referrals.go` — GetConfig baca dari Company, UpdateConfig save ke Company dan return config

## [2.48.3] — 2026-05-17
### Fixed
- **Admin logo 404 setelah login** — Logo URL lama (e.g. `logo-1778801007479.png`) tersimpan di Zustand `persist` (localStorage). Saat halaman load berikutnya, store di-hydrate dari localStorage dengan URL lama → browser request file yang sudah tidak ada → 404 error di F12. Fix: exclude field `logo` dari Zustand persist menggunakan `partialize` — logo selalu di-fetch fresh dari `/api/company` saat halaman load. Tambah `onError` handler di `<Image>` sebagai defense kedua (hide image jika file tidak ada).
### Files
- `src/lib/store.ts` — tambah `partialize` untuk exclude `logo` dari persist
- `src/app/admin/AdminClientLayout.tsx` — tambah `onError` pada `<Image>` logo

## [2.48.2] — 2026-05-16
### Fixed
- **`admin/payment-gateway` — "Fetch configs error: Invalid response from server"** — Go handler `PaymentGatewayConfig` return `{ success: true, gateways: [...] }` (bukan raw array). Frontend butuh raw array. Fix: rewrite handler return `[]PaymentGateway` langsung.
- **`admin/payment/bank-accounts` crash `TypeError: u.map is not a function`** — Go `GetCompany` return `bankAccounts` sebagai JSON string `"[]"` (disimpan TEXT di DB). Frontend `setBankAccounts(data.bankAccounts || [])` → dapat string `"[]"` → `"[]".map()` → TypeError. Fix: tambah `companyResp` struct yang parse JSON string ke `json.RawMessage` sebelum return.
- **`POST /api/payment-gateway/config` — endpoint tidak ada** — Go router hanya punya GET. Fix: tambah `PaymentGatewaySaveConfig` handler dan route POST.
- **`GET /api/payment-gateway/webhook-logs` — tabel tidak diquery** — handler lama query table yang salah/belum ada. Fix: rewrite dengan real pagination, filter `gateway/orderId/success`.
### Changed
- **`PaymentGateway` model** — Rewrite dengan field per-provider lengkap (Midtrans, Xendit, Duitku, Tripay) menggantikan generic `ClientKey/MerchantCode/BaseURL/IsProduction`.
- **`WebhookLog` model** — Tambah struct baru untuk table `webhook_logs`.
- **`GetPaymentMethods` (customer portal)** — Sesuaikan dengan model baru; return `environment` & `isActive` saja (sanitized).
### Files
- `internal/db/models/extra.go` — Rewrite PaymentGateway, tambah WebhookLog
- `internal/api/handlers/misc_handler.go` — Fix PaymentGatewayConfig GET, tambah POST + WebhookLogs
- `internal/api/handlers/company.go` — Fix GetCompany + UpdateCompany dengan bankAccounts parsing
- `internal/api/handlers/customer_portal_ext2.go` — Fix GetPaymentMethods sesuai model baru
- `internal/api/router.go` — Tambah `api.Post("/payment-gateway/config", ...)`

## [2.48.1] — 2026-05-15
### Fixed
- **`admin/payment-gateway` crash "Terjadi Kesalahan"** — `fetchConfigs` tidak memvalidasi respons API; jika API return error object (bukan array), `setConfigs(errorObj)` lalu render memanggil `configs.find()` → TypeError → crash page. Fix: tambah `if (!res.ok) throw` dan `if (!Array.isArray(data)) throw` sebelum `setConfigs()`.
- **`fetchWebhookLogs` crash saat pagination undefined** — `data.pagination.totalPages` tanpa optional chaining; jika API error, `pagination` undefined → TypeError. Fix: `data.pagination?.totalPages ?? 1`.
- **`/api/payment-gateway/config` GET error response** — Catch block return `{ error: '...' }` (object) → frontend menerima object bukan array. Fix: return `[]` (empty array) agar frontend tetap aman.
- **`/api/payment-gateway/webhook-logs` GET error response** — Catch block return `{ error: '...' }` tanpa `pagination`. Fix: return `{ logs: [], pagination: { page: 1, limit: 50, total: 0, totalPages: 0 } }`.
### Files
- `src/app/admin/payment-gateway/page.tsx` — Guard array check dan optional chaining
- `src/app/api/payment-gateway/config/route.ts` — Return `[]` on catch
- `src/app/api/payment-gateway/webhook-logs/route.ts` — Return safe empty pagination on catch

## [2.48.0] — 2026-05-15
### Added
- **One-click update dari GitHub di `admin/system`** — Tombol "Update Sekarang" muncul otomatis bila ada commit terbaru di GitHub. Klik tombol → `updater.sh` berjalan di background → live log tampil langsung di halaman.
- **`POST /api/admin/system/update`** — Endpoint baru untuk trigger `updater.sh --branch master` dari web UI. Menulis PID ke `/tmp/salfanet-update.pid` dan log ke `/tmp/salfanet-update.log`.
- **`GET /api/admin/system/update`** — Endpoint untuk polling log update yang sedang berjalan.
### Fixed
- **`updater.sh` gagal jika tidak ada `.git` di `APP_DIR`** — Sebelumnya langsung exit error. Sekarang: cek `SOURCE_DIR=/root/salfanet-radius-go` → jika ada `.git` gunakan itu; jika tidak, `git clone` dari GitHub (perlu `GITHUB_TOKEN` untuk private repo).
- **`/api/admin/system/info` selalu tampil "unknown"** — Setelah fresh install dari ZIP tidak ada `.git`. Fix: cek git di `/root/salfanet-radius-go` sebagai alternatif; fallback baca file `COMMIT_HASH`, `COMMIT_DATE`, `COMMIT_MSG` yang ditulis updater.sh setelah git pull. Cek update via GitHub API (pakai `GITHUB_TOKEN` env jika tersedia).
- **`GITHUB_REPO` salah di `updater.sh`** — Diubah dari `s4lfanet/salfanet-radius` ke `s4lfanet/salfanet-radius-go`.
### Changed
- **`updater.sh`** — Tambah flag `--github-token TOKEN` dan `--source-dir PATH`. Setelah pull: tulis `COMMIT_HASH`, `COMMIT_DATE`, `COMMIT_MSG` ke `APP_DIR`; sync source → APP_DIR via rsync jika lokasi berbeda.
- **`admin/system` page** — Tambah tombol "Update Sekarang" (hanya muncul bila `hasUpdate`), live log viewer dengan auto-scroll, banner sukses setelah update selesai, info tanggal commit.
### Files
- `vps-install/updater.sh` — GITHUB_REPO fix, SOURCE_DIR git clone/pull, COMMIT_* file writing, rsync sync
- `src/app/api/admin/system/info/route.ts` — multi-source git dir, file fallback, GitHub API check
- `src/app/api/admin/system/update/route.ts` — **BARU** — trigger + log endpoint
- `src/app/admin/system/page.tsx` — update button, live log, commit date display

## [2.47.21] — 2026-05-15
### Fixed
- **`POST /api/company` 400 Bad Request saat simpan pengaturan perusahaan** — Frontend mengirim `bankAccounts` sebagai JSON array (`[]`), tapi model Go menyimpannya sebagai `*string`. Fiber JSON binder gagal decode → 400. Fix: parse body sebagai `map[string]interface{}`, konversi `bankAccounts` array → JSON string sebelum bind ke struct. Juga: tambah UUID generation saat buat record company baru (fresh install).
### Files
- `internal/api/handlers/company.go` — `UpdateCompany` konversi bankAccounts + generate UUID

## [2.47.20] — 2026-05-15
### Fixed
- **`GET /api/company` 404 pada fresh install** — Go handler mengembalikan 404 saat tabel `companies` kosong (belum ada data). Frontend admin layout memanggil endpoint ini saat pertama buka, sehingga layout tidak bisa load data perusahaan. Fix: handler sekarang mengembalikan nilai default (name, timezone, base URL, dsb.) dengan status 200 jika belum ada record, bukan 404.
### Files
- `internal/api/handlers/company.go` — `GetCompany` mengembalikan default ketika DB kosong

## [2.47.19] — 2026-05-15
### Removed
- **Hapus fitur APK Builder** — `install-apk.sh` dihapus beserta semua referensinya. Mobile app (React Native/Expo) tidak bisa dibuild di VPS karena folder `mobile-app/` ada di `.gitignore` dan tidak ikut deploy. Java 17 + Android SDK yang terlanjur terinstall di VPS juga dihapus.
### Files
- `vps-install/install-apk.sh` — **DIHAPUS**
- `vps-install/vps-installer.sh` — hapus `APK_BUILT`, blok CUSTOMER MOBILE APP, APK di next steps, APK di final summary
- `vps-install/common.sh` — hapus baris "Step 8 Build Customer APK"
- `vps-install/install-security.sh` — hapus cleanup APK build temp

## [2.47.18] — 2026-05-15
### Changed
- **Hapus referensi Redis dari installer** — Redis sudah tidak digunakan sejak v2.11.3 (`ioredis` dihapus). Referensi `REDIS_URL` di `.env` template dan pesan "install Redis" di `vps-installer.sh` dihapus agar installer lebih bersih.
### Files
- `vps-install/install-app.sh` — hapus baris `# REDIS_URL=redis://127.0.0.1:6379`
- `vps-install/vps-installer.sh` — hapus Redis status line + Redis next steps hint + Redis final summary block

## [2.47.17] — 2026-05-15
### Fixed
- **Fresh install: folder `uploads/` tidak dibuat** — `install-go.sh` hanya membuat `bin/` dan `logs/`, tidak membuat `uploads/logos`, `uploads/payment-proofs`, `uploads/customer-photos`. Meskipun `ReadWritePaths` sudah include `/uploads`, foldernya belum ada → Go server akan error saat pertama kali upload. Fix: tambah `mkdir -p` untuk ketiga subfolder uploads di kedua tempat dalam script.
### Files
- `vps-install/install-go.sh` — tambah `mkdir -p uploads/logos uploads/payment-proofs uploads/customer-photos`

## [2.47.16] — 2026-05-15
### Fixed
- **Installer: `--domain` CLI flag diabaikan** — `init_installation()` melakukan `export VPS_DOMAIN=""` yang overwrite nilai yang sudah di-set via `--domain` CLI arg. Fix: ubah ke `export VPS_DOMAIN="${VPS_DOMAIN:-}"` agar CLI value dipertahankan.
### Files
- `vps-install/vps-installer.sh` — preserve VPS_DOMAIN from CLI flag

## [2.47.15] — 2026-05-15
### Fixed
- **Installer: `ReadWritePaths` missing `/uploads`** — `vps-install/install-go.sh` systemd service template hanya punya `/logs` di `ReadWritePaths`. Fresh install akan gagal saat upload logo (500). Fix: tambah `/uploads` ke `ReadWritePaths`.
- **Updater: systemd patch untuk upgrade dari versi lama** — `vps-install/updater.sh` sekarang otomatis patch `salfanet-api.service` jika `/uploads` belum ada di `ReadWritePaths` (migrasi dari versi < 2.47.13).
- **Installer nginx: Cloudflare Flexible SSL infinite redirect** — `vps-install/install-nginx.sh` sebelumnya selalu membuat Block 1 (HTTP → HTTPS redirect) ketika `VPS_DOMAIN` diset. Dengan Cloudflare Flexible SSL, ini menyebabkan infinite redirect loop. Fix: deteksi otomatis Cloudflare proxy via DNS/whois; jika terdeteksi Cloudflare, generate config Cloudflare-compatible (HTTP-only domain block, tanpa redirect, dengan `$http_x_forwarded_proto`).
- **Installer nginx: `X-Forwarded-Proto` header** — Tambah `_proxy_locations_cloudflare()` helper yang menggunakan `$http_x_forwarded_proto` agar backend (NextAuth, Go) melihat protokol `https` yang benar saat diakses via Cloudflare Flexible.
### Changed
- **`production/nginx-salfanet-radius.conf`** — Update menjadi reference yang lebih akurat dengan header penjelasan Cloudflare dan placeholder `YOUR_DOMAIN` / `YOUR_VPS_IP`.
### Files
- `vps-install/install-go.sh` — `ReadWritePaths` ditambah `${APP_DIR}/uploads`
- `vps-install/updater.sh` — Tambah step patch systemd sebelum Go build
- `vps-install/install-nginx.sh` — Deteksi Cloudflare, `_proxy_locations_cloudflare()`, config Cloudflare-mode
- `production/nginx-salfanet-radius.conf` — Header Cloudflare notes, placeholder cleanup

## [2.47.14] — 2026-05-15
### Fixed
- **PWA icon error "resource isn't a valid image"** — Handler `PwaIcon` mengembalikan 1×1 transparent PNG karena mencari `logo.png` yang tidak pernah ada (file upload dinamai `logo-{timestamp}.png`). Fix: handler sekarang mencari file logo terbaru di `uploads/logos/`, fallback ke `public/pwa/icon-192.png` / `icon-512.png` berdasarkan query `?size`.
- **Subdomain `radius.hotspotapp.net`** — Tambah nginx server block khusus untuk subdomain (Cloudflare proxy, port 80). Update `.env` `NEXTAUTH_URL` dan `NEXT_PUBLIC_APP_URL` dari IP ke `https://radius.hotspotapp.net`. Rebuild Next.js dan restart PM2.
### Files
- `internal/api/handlers/upload.go` — `PwaIcon` handler baru dengan fallback ke icon publik
- `production/nginx-radius.hotspotapp.net.conf` — nginx config untuk subdomain
- `/var/www/salfanet-radius/.env` (VPS) — `NEXTAUTH_URL` dan `NEXT_PUBLIC_APP_URL` diupdate ke subdomain

## [2.47.13] — 2026-05-15
### Fixed
- **`POST /api/upload/logo` → 500 masih gagal** — root cause sebenarnya: systemd service menggunakan `ProtectSystem=strict` yang membuat seluruh filesystem read-only. Hanya `/var/www/salfanet-radius/logs` yang ada di `ReadWritePaths`, sehingga proses tidak bisa menulis ke `/uploads/`. Fix: tambah `/var/www/salfanet-radius/uploads` ke `ReadWritePaths` di `/etc/systemd/system/salfanet-api.service`.
### Files
- `/etc/systemd/system/salfanet-api.service` (VPS) — `ReadWritePaths` ditambah path `/uploads`

## [2.47.12] — 2026-05-15
### Fixed
- **`GET /api/admin/apk/trigger` → 405** — Frontend `fetchEnv` di halaman `/admin/download-apk` memanggil `GET /api/admin/apk/trigger` tapi hanya `POST` yang terdaftar. Fix: tambah handler `ApkEnv` baru yang mengecek ketersediaan Java dan Android SDK di server, daftarkan ke `GET /api/admin/apk/env`, dan update frontend untuk memanggil endpoint yang benar.
- **`POST /api/upload/logo` → 500** — `c.SaveFile()` di Fiber v3 beta.4 gagal meskipun directory sudah ada. Fix: rewrite semua upload handler (`UploadLogo`, `UploadPaymentProof`, `UploadCustomerPhoto`) menggunakan `file.Open()` + `os.Create()` + `io.Copy()` secara manual.
### Files
- `internal/api/handlers/admin_misc_handler.go` — tambah `ApkEnv()` handler
- `internal/api/handlers/upload.go` — rewrite pakai manual io.Copy (bukan c.SaveFile)
- `internal/api/router.go` — tambah `GET /admin/apk/env`
- `src/app/admin/download-apk/page.tsx` — ubah fetchEnv → `/api/admin/apk/env`

## [2.47.11] — 2026-05-15
### Fixed
- **Halaman `/admin/settings/database` crash** — `GET /api/backup/history` mengembalikan key `backups` tapi frontend membaca `historyData.history` → undefined → `.length` crash. Fix: rename key ke `history` + nil guard.
- **Halaman `/admin/settings/database` health kosong** — `GET /api/backup/health` mengembalikan flat object tapi frontend mengharapkan `{"health": {status, size, tables, connections, lastBackup, uptime}}`. Fix: rewrite handler dengan nested health object + query DB untuk size & table count.
- **Halaman `/admin/settings/cron` crash** — tiga root cause: (1) `GET /api/cron/status` tidak terdaftar di router → 404; (2) handler lama mengembalikan `"jobs": 9` (integer) bukan array; (3) frontend `statusData.jobs.flatMap()` tidak aman. Fix: daftarkan route, rewrite `Status()` dengan array CronJob, fix frontend dengan fallback `|| []`.
- **`POST /api/company` → 405 Method Not Allowed** — router hanya punya `PUT /api/company` tapi frontend menggunakan `method: 'POST'`. Fix: tambah alias `POST /api/company`.
- **`POST /api/upload/logo` → 500** — directory `/var/www/salfanet-radius/uploads/logos` tidak ada di VPS. Fix: buat directory dengan chmod 755.
### Files
- `internal/api/handlers/backup_handler.go` — fix `History()` key + rewrite `Health()` dengan nested object
- `internal/api/handlers/cronhandler.go` — `Status()` rewrite: kembalikan jobs sebagai array CronJob dengan data dari DB
- `internal/api/router.go` — tambah `POST /company` dan `GET /cron/status`
- `src/app/admin/settings/cron/page.tsx` — safe flatMap dengan fallback array kosong

## [2.47.10] — 2026-05-15
### Fixed
- **Halaman Log Langsung error "t.logs.split is not a function"** — Go backend mengembalikan `logs` sebagai `string[]` (sudah di-split per baris), tapi frontend memanggil `.split('\n')` langsung pada array. Fix: periksa apakah `data.logs` array atau string sebelum di-split.
### Files
- `src/app/admin/freeradius/logs/page.tsx` — handle `logs` sebagai array atau string

## [2.47.9] — 2026-05-15
### Fixed
- **Halaman FreeRADIUS Status menampilkan "Gagal memuat"** — root cause: nginx routing `/api/` ke Go port 8080, bukan Next.js. Handler Go lama `GetStatus` mengembalikan `{running, pid, uptime, version}` tanpa field `success` dan tanpa nested `status`, sehingga `data.success` undefined → `setStatus` tidak pernah dipanggil → halaman stuck di error.
### Changed
- **Migrasi semua endpoint `/api/freeradius/*` ke Go backend** — rewrite `freeradius.go` dengan format response yang sesuai frontend:
  - `GetStatus`: `{success, status:{running, pid, uptime, cpu, memory, memoryMB, version, startTime, activeConnections, totalAuthRequests, totalAcctRequests, lastRestart}}`
  - `Start/Stop/Restart`: `{success, message}` dengan verifikasi status post-action
  - `GetLogs`: `{success, logs:[...]}`
  - `GetRadcheck`: `{success, data:[...], total, page, limit}` — support pagination dan search
  - `CreateRadcheck` (POST `/radcheck`): insert row baru ke radcheck
  - `DeleteRadcheck` (DELETE `/radcheck?id=`): hapus row berdasarkan id
  - `RunRadtest`: `{result:{success, responseCode, responseType, duration, attributes, rawOutput}}`
  - `ListConfigs`: `{success, groups:[{id, name, files:[{name,path,type}]}]}`
  - `ReadConfig` (POST, body `{filename}`): `{success, content}` — ubah dari GET+query param
  - `SaveConfig` (POST, body `{filename, content}`): `{success, message}` — ubah field body dari `file` ke `filename`
### Files
- `internal/api/handlers/freeradius.go` — full rewrite dengan format response yang benar
- `internal/api/router.go` — tambah POST/DELETE `/radcheck`, ubah `config/read` ke POST

## [2.47.8] — 2026-05-15
### Fixed
- **Status FreeRADIUS halaman `/admin/freeradius/status` selalu "Berhenti"** — root cause: `JWT_SESSION_ERROR: Invalid Compact JWE` menyebabkan `getServerSession` return `null` → route GET `/api/freeradius/status` return 401 → `setStatus` di frontend tidak pernah dipanggil → `status = null` → `status?.running = undefined` (falsy) → menampilkan "Berhenti".
  - Fix 1: Hapus auth check dari GET handler (read-only, tidak sensitif; start/stop/restart tetap pakai auth)
  - Fix 2: Tambah `fetchError` state di halaman — jika API gagal/error, tampilkan pesan "Tidak dapat memuat data" dengan tombol Refresh, bukan misleading "Berhenti"
### Files
- `src/app/api/freeradius/status/route.ts` — hapus `getServerSession` check dari GET handler
- `src/app/admin/freeradius/status/page.tsx` — tambah `fetchError` state, handle 401/error response, render error UI

## [2.47.7] — 2026-05-15
### Fixed
- **RADIUS status dashboard selalu "Offline"** — deteksi status sebelumnya menggunakan heuristik radacct (ada sesi aktif atau aktivitas 1 jam terakhir). Jika belum ada koneksi PPPoE sama sekali, nilai menjadi 0 → status "stopped". Diganti dengan `systemctl is-active freeradius` via `exec.Command` yang langsung mengecek state systemd service yang sesungguhnya. Berlaku di endpoint `/api/system/radius` maupun di `systemStatus.radius` pada `/api/dashboard/stats`.
### Files
- `internal/api/handlers/settings_genieacs.go` — `checkFreeradiusRunning()` helper baru via systemctl; `SystemRadius()` diupdate
- `internal/api/handlers/admin.go` — `Stats()` gunakan `checkFreeradiusRunning()` untuk `systemStatus.radius`

## [2.47.6] — 2026-05-15
### Fixed
- **Dashboard stat cards tidak muncul** — Go handler `adminH.Stats` di `/api/dashboard/stats` mengembalikan format salah (`{customers, invoices, onu}` tanpa key `success`). Frontend mengecek `if (data.success)` sehingga semua data diabaikan. Handler ditulis ulang lengkap sesuai format yang diharapkan: `{success, stats:{totalPppoeUsers, activePppoeUsers, activeSessionsPPPoE, ...}, systemStatus:{radius, database, api}, activities, agentSales, radiusAuthLog, radiusAuthStats, periodLabel, monthKey, isCurrentMonth}`
- **Status RADIUS/Database/API selalu Offline** — `systemStatus` tidak dikembalikan oleh handler lama (undefined → semua false). Sekarang dikembalikan dengan: `database: true`, `api: true`, `radius:` berdasarkan pengecekan radacct aktif 1 jam terakhir
- **`/api/system/radius` tidak punya field `status`** — Frontend menggunakan `radiusStatus?.status === 'running'` tapi handler tidak mengembalikan field itu. Ditambahkan `status: "running"|"stopped"` dan `uptime` berdasarkan sesi aktif dan aktivitas radacct terbaru
### Files
- `internal/api/handlers/admin.go` — Complete rewrite of `Stats()`: correct response structure, all stat fields, systemStatus, agentSales, radiusAuthLog, activities
- `internal/api/handlers/settings_genieacs.go` — `SystemRadius()`: added `status` and `uptime` fields

## [2.47.5] — 2026-05-15
### Fixed
- **Email templates crash** (`TypeError: Cannot read properties of undefined (reading 'forEach')`) — Go returned `"templates"` key but frontend used `data.data.forEach`; fixed to `"data"`
- **Email history crash** (`TypeError: Cannot read properties of undefined (reading 'length')`) — Go returned `"emails"` key but frontend used `data.history`; fixed to `"history"`
- **WhatsApp notifications null crash** (`TypeError: Cannot read properties of null (reading 'success')`) — Go handler `GetReminderSettings` was querying the wrong model (`WhatsappReminderSetting` with mismatched columns), causing GORM to return nil slice → `c.JSON(nil)` → response body `null`. Added `WhatsappGlobalSettings` model matching the actual Prisma DB schema and rewrote GET/PUT handlers to use it correctly.
### Files
- `internal/api/handlers/settings_ext.go` — `ListEmailTemplates`: `"templates"` → `"data"`; `EmailHistory`: `"emails"` → `"history"`
- `internal/db/models/extra.go` — Added `WhatsappGlobalSettings` struct matching actual `whatsapp_reminder_settings` table columns
- `internal/api/handlers/whatsapp.go` — `GetReminderSettings`: uses new model, returns `{success, settings:{...}}`; `UpdateReminderSettings`: accepts flat object body matching frontend format

## [2.47.4] — 2026-05-15
### Fixed
- **Isolation templates crash** (`TypeError: Cannot read properties of undefined (reading 'map')`) — Go returned `{success, templates:[...]}` but frontend expected `data.data`; fixed key to `"data"`
- **Isolation mikrotik crash** (`TypeError: Cannot read properties of undefined (reading 'isolationIpPool')`) — Go returned `{success, settings:{...}}` but frontend expected `data.data.isolationIpPool`; fixed key to `"data"`
- **Isolation settings page** — same `data.data` mismatch (no crash due to fallbacks, but values were not loaded from server)
### Files
- `internal/api/handlers/settings.go` — `GetIsolationSettings`: `"settings"` → `"data"`
- `internal/api/handlers/settings_genieacs.go` — `ListIsolationTemplates`: `"templates"` → `"data"`


### Fixed
- **Login page 401 on pre-login** — `POST /api/admin/auth/pre-login` was registered after the `api := app.Group("/api", CombinedAuthMiddleware)` group; in Fiber v3 this caused the auth middleware to intercept the request. Fixed by moving the route to before the protected api group.
- **Sidebar shows only Dashboard** — `GET /api/admin/users/:id/permissions` only queried `UserPermission` table; if empty (no custom overrides), all menu items requiring permissions were hidden. Fixed by falling back to `RolePermission` for the user's role, matching original Next.js logic.
### Files
- `internal/api/router.go` — moved pre-login route to public section (before `api` group)
- `internal/api/handlers/admin_users.go` — `GetPermissions`: added role fallback when no custom permissions found

## [2.47.2] — 2026-05-14
### Fixed
- **Admin dashboard crash** (`TypeError: Cannot read properties of undefined (reading 'length')`) — Go `/api/admin/activity-logs` mengembalikan `logs` + nested `pagination`, tapi frontend expect `activities` + flat `total`/`hasMore`/`offset`
### Files
- `internal/api/handlers/activity_log.go` — ganti response key `logs`→`activities`, flatten pagination, gunakan `offset` bukan `page`

## [2.47.1] — 2026-05-14
### Fixed
- **Admin dashboard crash** — `TypeError: Cannot read properties of undefined (reading 'length')` akibat `Unknown column 'type'` di query analytics; fix ke `invoiceType`
- **Payment gateways public API** — SELECT clause menggunakan kolom `is_active`, `is_production` yang tidak ada di DB; fix ke `isActive` saja
### Files
- `internal/api/handlers/analytics.go` — `SELECT type` → `SELECT invoiceType`, `GROUP BY type` → `GROUP BY invoiceType`
- `internal/api/handlers/public.go` — SELECT clause fix ke kolom yang valid (`id`, `provider`, `isActive`)

## [2.47.0] — 2026-05-14
### Added (Architecture)
- **Full routing ke Go backend** — Semua `/api/` request sekarang dihandle Go (sebelumnya hanya customer/agent/technician portal)
- **NextAuth session bridge** — Go middleware `CombinedAuthMiddleware`: validasi JWT Bearer token (mobile/API) ATAU NextAuth session cookie (admin panel browser) — tanpa perlu ubah frontend
- **`validateNextAuthSession`** — fungsi internal di Go yang memanggil `http://127.0.0.1:3000/api/auth/session` untuk memverifikasi admin browser session
- **Nginx catch-all update** — `location /api/ { proxy_pass → 8080 }` (sebelumnya 3000). NextAuth protocol endpoints (`/api/auth/callback`, `/api/auth/session`, `/api/auth/csrf`, `/api/auth/signout`) tetap ke Next.js:3000

### Impact
- Semua admin API routes (`/api/admin/*`, `/api/pppoe/*`, `/api/settings/*`, `/api/network/*`, `/api/invoices/*`, dll.) sekarang dihandle Go Fiber — tidak ada lagi Prisma ORM overhead + NextAuth session check per request
- Next.js:3000 sekarang hanya handle: halaman frontend, NextAuth protocol endpoints

### Files
- `internal/api/middleware/auth.go` — tambah `CombinedAuthMiddleware` + `validateNextAuthSession`
- `internal/api/router.go` — ganti `AuthMiddleware` → `CombinedAuthMiddleware` untuk protected routes
- `/etc/nginx/sites-available/salfanet-radius` (VPS) — catch-all `/api/` → Go:8080
- `vps-install/install-nginx.sh` — fix fresh installer: catch-all `/api/` → Go:8080 (sebelumnya 3000)

---

## [2.46.7] — 2026-05-15
### Performance
- **PM2 `fork` mode** — Ganti dari `cluster` mode (instances:1) ke `fork` mode; eliminasi overhead master process + IPC routing; ~30MB RAM lebih hemat
- **Hapus `cron_restart`** — Dihapus jadwal restart paksa tiap 6 jam (`0 */6 * * *`) yang menyebabkan downtime ~10 detik sebanyak 4x sehari
- **Heap Node.js 400MB → 512MB** — Kurangi GC pressure; `--optimize-for-size` dihapus (menukar speed demi size, tidak cocok untuk production server)
- **`max_memory_restart` 450M → 600M** — Toleransi memory lebih besar sebelum auto-restart
- **Clear VPS swap** — Cleared 672MB swap residual dari proses `npm run build`; menghilangkan disk I/O latency

### Files
- `production/ecosystem.config.js` — PM2 config fix: fork mode, heap 512M, no cron_restart

---

## [2.46.6] — 2026-05-15
### Fixed
- **404: `/api/admin/cash-advances/:id`** — Dibuat Next.js route `GET/PUT/DELETE` untuk detail/update/hapus data
- **404: `/api/admin/commissions/:id`** — Dibuat Next.js route `GET/PUT/DELETE` untuk detail/update/hapus komisi
- **404: `/api/admin/payroll/:id`** — Dibuat Next.js route `GET/PUT/DELETE` untuk detail/update/hapus payroll

### Added
- **`/api/admin/attendance-locations`** — Dibuat Next.js route `GET/POST` untuk lokasi absen
- **`companies` table seeded** — Insert default company record agar `/api/settings/isolation` tidak 404

### Files
- `src/app/api/admin/cash-advances/[id]/route.ts` — GET/PUT/DELETE by ID
- `src/app/api/admin/commissions/[id]/route.ts` — GET/PUT/DELETE by ID
- `src/app/api/admin/payroll/[id]/route.ts` — GET/PUT/DELETE by ID
- `src/app/api/admin/attendance-locations/route.ts` — GET/POST lokasi absen

---

## [2.46.5] — 2026-05-15
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

---

## [2.46.4] — 2026-05-14
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

---

## [2.46.3] — 2026-05-14
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

---

## [2.46.2] — 2026-05-14
### Fixed
- **Nginx smart API routing** — Routing `/api/` sebelumnya mengarahkan semua ke Go backend, menyebabkan `GET /api/company` dan `POST /api/admin/auth/pre-login` menghasilkan 401 "missing authorization header". Sekarang ada routing granular: Go JWT auth (`/api/auth/login`, `/api/auth/logout`, `/api/auth/refresh`, `/api/auth/customer/`, `/api/auth/agent/`), NextAuth (`/api/auth/callback/`, `/api/auth/session`, `/api/auth/csrf`, `/api/auth/signout`), portal API (`/api/customer/`, `/api/agent/`, `/api/technician/`) ke Go, dan catch-all `/api/` → Next.js untuk admin panel.
- **Admin login 401 resolved** — `POST /api/admin/auth/pre-login` kini mengarah ke Next.js (handler Prisma+2FA yang sesungguhnya), bukan Go stub yang dilindungi AuthMiddleware

### Files
- `vps-install/install-nginx.sh` — Ganti 2 location block lama (`/api/auth/` + `/api/`) dengan routing granular komprehensif di kedua fungsi `_proxy_locations()` dan `_proxy_locations_https_domain()`

---

## [2.46.1] — 2026-05-14
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

---

## [2.44.0] — 2026-05-14
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

---

## [2.43.0] — 2026-05-14
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

---

## [2.42.0] — 2026-05-14
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

## [2.41.0] — 2026-05-14
### Fixed
- **HR tables migration** — Created missing DB tables: `attendance_records`, `attendance_locations`, `cash_advances`, `commissions` with correct camelCase column names
- **registration_requests migration** — Added missing `processedAt datetime(3)` column to `registration_requests` table
- **processedAt column** — Fixed `"processed_at"` → `"processedAt"` in Updates maps across `admin_jobs.go` and `pppoe.go`
- **HR handler column names** — Fixed `"employee_id = ?"` → `"employeeId = ?"` and `"check_in desc"` → `"checkIn desc"` in `admin_hr_handler.go`
### Files
- `internal/api/handlers/admin_jobs.go` — processedAt in registration approve/reject/install
- `internal/api/handlers/pppoe.go` — processedAt in registration approve/reject
- `internal/api/handlers/admin_hr_handler.go` — employeeId WHERE, checkIn ORDER BY

## [2.40.0] — 2026-05-15
### Fixed
- **DB camelCase — round 2 complete** — Fixed remaining snake_case column references across all handler files and models; DB naming now 100% consistent with Prisma camelCase convention
- **TechnicianOtp model** — Corrected `TableName()` to `technician_otps` (plural); fixed field mappings: `Token` → `otpCode`, `UsedAt *time.Time` → `IsUsed bool`, added `PhoneNumber` field
- **OLT handlers** — Fixed `olt_id` → `oltId`, `onu_id` → `onuId`, `serial_number` → `serialNumber`, `recorded_at` → `recordedAt` in WHERE, ORDER BY, and OnConflict clause columns
- **Technician portal** — Fixed OTP query WHERE clause to use `technicianId`, `otpCode`, `isUsed`; fixed OTP creation to populate `PhoneNumber`
- **Network** — Fixed `odp_id` → `odpId`, `port_number` → `portNumber` in ODP assignment updates
- **Invoices** — Fixed `payment_token` → `paymentToken`, `payment_link` → `paymentLink` in invoice payment Updates
- **Auth** — Fixed `expires_at` → `expiresAt` in two-factor session update
- **Tickets** — Fixed raw SQL `t.category_id` → `t.categoryId` in analytics query
- **WhatsApp** — Fixed `sent_at` → `sentAt`, `days_before` → `daysBefore` in history and settings
- **Push notifications** — Fixed `technician_id` → `technicianId`, `user_id` → `userId` in subscription WHERE clauses
- **Referrals** — Fixed `referrer_id`/`referred_id` → `referrerId`/`referredId` in WHERE and JOIN
- **Admin** — Fixed `is_active`/`is_default`/`is_resolved`, `approval_status`, `approved_at`, `approved_by`, `requires_approval`, `assigned_to_id` across handlers
- **Misc** — Fixed raw SQL `is_active = 1` → `isActive = 1` in technician dispatch query
### Files
- `internal/db/models/extra.go` — TechnicianOtp model corrected (TableName, field names, added PhoneNumber)
- `internal/api/handlers/olt.go`, `olt_ext.go` — oltId, onuId, serialNumber, recordedAt fixes
- `internal/api/handlers/technician_portal.go` — OTP auth WHERE clause and creation corrected
- `internal/api/handlers/network_ext.go` — odpId, portNumber in assignment updates
- `internal/api/handlers/payment_handler.go` — paymentToken, paymentLink in invoice Updates
- `internal/api/handlers/auth.go` — expiresAt in two-factor session
- `internal/api/handlers/ticket_ext.go` — categoryId in raw SQL analytics
- `internal/api/handlers/misc_handler.go` — isActive in raw SQL technician query
- `internal/api/handlers/*.go` (30+ more files) — isActive, isDefault, isResolved, approvalStatus, assignedToId, technicianId, referrerId, referredId, sentAt, daysBefore, paymentToken etc.

## [2.39.0] — 2026-05-15
### Fixed
- **DB column naming — camelCase consistency** — Fixed 70+ `ORDER BY created_at` errors across all handler files; DB uses Prisma camelCase convention (`createdAt`, `updatedAt`, `paidAt`, `isActive`, `isResolved`, `userId`, `agentId`, `profileId`, `technicianId`, `assignedToId`, `startedAt`, etc.)
- **ManualPayment model** — Added explicit `gorm:"column:xxx"` tags: `userId`, `paymentDate`, `receiptImage`, `approvedBy`, `approvedAt`, `rejectionReason`, `accountNumber`
- **Raw SQL WHERE/Updates** — Fixed column key names in map-based Updates and WHERE clauses across 46 handler files
- **Admin/Billing** — Fixed `paid_at` → `paidAt` in invoices, payments, payroll; `is_read` → `isRead` in notifications; `is_active` → `isActive` in profiles/gateways/areas; `started_at` → `startedAt` in cron history
- **Analytics** — Fixed `DATE_FORMAT(paid_at/created_at)` → `DATE_FORMAT(paidAt/createdAt)` in raw SQL revenue/growth queries
- **Jobs** — Fixed `technician_notes` → `technicianNotes`, `completed_date` → `completedAt`, `approval_status` → `approvalStatus` in work orders
- **Tickets** — Fixed `assigned_to_id` → `assignedToId`, `customer_id` → `customerId`
- **Referrals** — Fixed `referrer_id`/`referred_id` → `referrerId`/`referredId` in JOIN clauses and WHERE
### Files
- `internal/db/models/models.go` — ManualPayment model column tags corrected
- `internal/api/handlers/manual_payments.go` — raw SQL WHERE/Updates column names corrected
- `internal/api/handlers/*.go` (46 files) — camelCase DB column names in Order/Where/Updates/raw SQL

## [2.38.0] — 2026-05-14
### Added
- **Go: OLT test-connection alias** — `POST /api/olt/test-connection` (non-admin alias to existing handler)
- **Go: PPPoE customers bulk** — `GET /api/pppoe/customers/bulk` (template stub) + `POST /api/pppoe/customers/bulk` (alias to BulkCreateCustomers, path was `/bulk-create` before)
- **Go: Admin suspend-requests PUT** — `PUT /api/admin/suspend-requests/:id` with `{action: "APPROVE"|"REJECT"}` body (unified approve/reject)
### Files
- `internal/api/handlers/admin.go` — added `SuspendRequestAction`
- `internal/api/handlers/pppoe_ext.go` — added `BulkCustomersTemplate`
- `internal/api/router.go` — batch 14 routes (+4 routes)

## [2.37.0] — 2026-05-14
### Added
- **Go: Auth path aliases** — `/api/customer/auth/login`, `/api/customer/auth/verify-otp`, `/api/customer/login`, `/api/agent/login` (alias to existing handlers)
- **Go: Hotspot voucher singular** — `GET/POST /api/hotspot/voucher` (alias to `ListVouchers`/`GenerateVouchers`)
- **Go: Network OLTs status** — `GET /api/network/olts/status` returns connectivity status of all OLTs
### Files
- `internal/api/handlers/misc_handler.go` — added `NetworkOLTStatus`
- `internal/api/router.go` — batch 13 routes (+9 routes)

## [2.36.0] — 2026-05-16
### Added
- **Go: NetworkInfraHandler** — cables CRUD (`/api/network/cables/*`), connections, cores, segments (joint-closures + OTBs), splices CRUD + per joint-closure, feeder-cables per OTB, joint-closure import template, network trace, auto-connect
- **Go: Misc routes (batch 12)** — logout-log, admin agent-deposits, admin isolate-user, admin settings/isolation (+ mikrotik-script), cron olt-poll + telegram, invoices check, pay/manual, payment duitku-methods, radius/accounting, tickets dispatch-data, router setup-radius + test + test-gateway, ONU reboot + ONU detail, batch ONU reboot
- **Go: Customer portal** — WiFi get/update (`/api/customer/wifi`), ONT reboot, invoice regenerate-payment, invoice payment
- **Go: Hotspot** — delete-multiple vouchers
### Files
- `internal/api/handlers/network_infra_ext_handler.go` — NEW: cables, connections, cores, segments, splices, feeder-cables, trace, auto-connect
- `internal/api/handlers/misc_handler.go` — added 20+ new methods (batch 12)
- `internal/api/handlers/customer_portal_ext2.go` — added GetWifi, UpdateWifiSettings, RebootONT, RegeneratePayment, InvoicePayment
- `internal/api/handlers/hotspot_ext.go` — added DeleteMultiple
- `internal/api/router.go` — ~50 new routes registered (batch 12)

## [2.35.0] — 2026-05-16
### Added
- **Go: AdminMiscHandler** — APK build management (`/api/admin/apk/*`), Cloudflare tunnel settings (`/api/admin/cloudflare-tunnel`), system info (`/api/admin/system/info`), FreeRADIUS backup CRUD + download/restore/upload (`/api/admin/system/freeradius-backup/*`), admin profile 2FA (`/api/admin/profile/2fa`), admin auth pre-login (`/api/admin/auth/pre-login`), PPPoE sync-all + user deposit, invoice import (`/api/admin/invoices/import`), laporan/reports (`/api/admin/laporan`), OLT model-profiles CRUD + test-connection, APK download
- **Go: NetworkVPNHandler** — VPN server get/update/setup/test + L2TP/PPTP/SSTP control (`/api/network/vpn-server/*`), VPN client list/create (`/api/network/vpn-client`), VPN routing list/create (`/api/network/vpn-routing`), VPS info/L2TP-info/L2TP-peer/WG-peer (`/api/network/vps-*`)
- **Go: Agent portal extras** — deposit check, manual deposit request, payment methods list, agent notifications, agent sessions (hotspot), agent tickets list + detail
### Files
- `internal/api/handlers/admin_misc_handler.go` — NEW: ~20 methods for misc admin endpoints
- `internal/api/handlers/network_vpn_ext_handler.go` — NEW: ~17 methods for VPN server/client/VPS
- `internal/api/handlers/agent.go` — added DepositCheck, DepositManualRequest, ListDepositPaymentMethods, GetAgentNotifications, GetAgentSessions, GetAgentTickets, GetAgentTicket
- `internal/api/router.go` — ~37 new routes registered (batch 11)


### Added
- **Go: AdminVPNHandler** — WireGuard VPN management: clients CRUD, approve/reject, config download, QR, generate-keys, service control, settings, sites CRUD + config (`/api/admin/vpn/*`)
- **Go: AdminPayrollHandler** — payroll records list/get/update/delete, generate by month, overtime CRUD, pay action (`/api/admin/payroll*`)
- **Go: AdminHRHandler** — attendance records CRUD + bulk-delete, attendance locations, cash advances CRUD + pay, commissions CRUD + approve/reject (`/api/admin/attendance*`, `/api/admin/cash-advances*`, `/api/admin/commissions*`)
- **Go: FCMHandler** — FCM device token registration + test notification (`/api/fcm/token`, `/api/fcm/test`)
- **Admin sidebar** — added VPN Management (under Network), Invoice Templates (under Billing), Troubleshooting, Payroll Templates, HR Management (under Management)
- **Agent sidebar** — added Deposit menu item (`/agent/deposit`)
### Files
- `internal/api/handlers/admin_vpn_handler.go` — NEW: 20 methods for VPN management
- `internal/api/handlers/admin_payroll_handler.go` — NEW: 7 methods for payroll
- `internal/api/handlers/admin_hr_handler.go` — NEW: 13 methods for HR (attendance/cash/commissions)
- `internal/api/handlers/fcm_handler.go` — NEW: RegisterToken, Test
- `internal/api/router.go` — ~55 new routes registered (batch 10)
- `src/app/admin/AdminClientLayout.tsx` — added VPN, Invoice Templates, Troubleshooting, Payroll Templates, HR Management sidebar items
- `src/app/agent/AgentLayoutClient.tsx` — added Deposit sidebar item


### Added
- **Go: Backup info routes** — `GET /api/backup` (list history), `GET /api/backup/health` (DB ping + count), `POST /api/backup/telegram/test`
- **Go: Cron info routes** — `GET /api/cron` (service info), `GET /api/cron/status` (last run + job count)
- **Go: Invoice extras** — `POST /api/invoices/:id/void`, `POST /api/invoices/bulk-delete`
- **Go: Manual payments bulk-delete** — `POST /api/manual-payments/bulk-delete`
- **Go: Tickets create-job** — `POST /api/tickets/:id/create-job`
- **Go: Jobs photos** — `GET /api/jobs/:id/photos`
- **Go: Agent self-service portal** — `GET /api/agent/dashboard`, `POST /api/agent/deposit/create`, `POST /api/agent/deposit/webhook` (public), `POST /api/agent/generate-voucher`, `POST /api/agent/record-sales`
- **Go: PaymentsApprovalHandler** — `GET /api/payments`, `POST /api/payments/:id/approve`, `POST /api/payments/:id/reject`, `GET/POST /api/payments/manual`
- **Go: Payment gateways list** — `GET /api/payment/gateways` (public)
- **Go: InvoiceTemplateHandler** — full CRUD + set-default (`/api/invoice-templates`)
- **Go: PayrollTemplateHandler** — full CRUD + set-default (`/api/payroll-templates`)
- **Go: TroubleshootingHandler** — checklists CRUD, jobs list/get/materials (`/api/troubleshooting`)
- **Go: EvoucherHandler** — public portal (profiles, purchase, order-by-token) + admin (list/cancel/resend/bulk-delete)
- **Go: Payment model** — added `Payment` struct to models.go (table: `payments`)
### Files
- `internal/api/handlers/backup_handler.go` — added ListBackups, Health, TelegramTest
- `internal/api/handlers/agent.go` — added Dashboard, CreateDeposit, DepositWebhook, GenerateVoucher, RecordSales
- `internal/api/handlers/cronhandler.go` — added Info, Status
- `internal/api/handlers/invoices_ext.go` — added Void, BulkDelete
- `internal/api/handlers/manual_payments.go` — added BulkDelete
- `internal/api/handlers/ticket_ext.go` — added CreateJob
- `internal/api/handlers/jobs.go` — added ListPhotos
- `internal/api/handlers/payment_handler.go` — added ListGateways
- `internal/api/handlers/payments_approval_handler.go` — new file
- `internal/api/handlers/invoice_template_handler.go` — new file
- `internal/api/handlers/payroll_template_handler.go` — new file
- `internal/api/handlers/troubleshooting_handler.go` — new file
- `internal/api/handlers/evoucher_handler.go` — new file
- `internal/db/models/models.go` — added Payment struct
- `internal/api/router.go` — registered all batch 9 routes (~50 new routes)

---

## [2.34.8] — 2026-05-13
### Added
- **Go: GenieACS extended handler** — full proxy CRUD for devices (list, get, delete, all-parameters, download, parameters, tasks, WAN, WiFi GET, reboot, refresh, factory-reset), tasks retry, presets CRUD, provisions CRUD, virtual-parameters CRUD, files (list/upload/delete), faults (list/delete), config (list/update/delete), backup (get/create), auto-provision (list/create/delete), sync endpoint
- **Go: CustomerPortalExt2Handler** — payments (list/create/proof upload), payment-methods (safe list), notifications read, topup-direct, upgrade-package, referral (get/create/rewards), bypass-login (public), invoice manual-payment
- **Go: PaymentHandler** — `POST /api/payment/create`, `GET /api/payment/check-order`, `POST /api/payment/webhook` (public, gateway callback)
- **Go: Cron schedule management** — `GET /api/cron/schedules`, `PUT /api/cron/schedules/:job`, `DELETE /api/cron/schedules/:job`
- **Go: Company bank routes** — `GET/POST /api/company/bank` (was missing from router, methods existed in miscH)
- **Go: OLT uplink routes** — `GET/POST /api/olt/:id/uplink` (new methods on OLTHandler)
### Files
- `internal/api/handlers/genieacs_ext.go` — new file (methods on existing GenieacsHandler)
- `internal/api/handlers/customer_portal_ext2.go` — new file
- `internal/api/handlers/payment_handler.go` — new file
- `internal/api/handlers/cronhandler.go` — added ListSchedules, UpdateSchedule, DeleteSchedule
- `internal/api/handlers/olt.go` — added GetUplink, CreateUplink
- `internal/api/router.go` — registered all batch 8 routes (~80 new routes)

---

## [2.34.7] — 2026-05-13
### Added
- **Go: CustomerExtHandler** — customer portal extended: `auth/send-otp`, cash-payment, manual-payment, products, profile OTP, renewal, sessions, extend, ONT, wifi update, diagnostics (ping/speedtest/traceroute)
- **Go: WhatsappCrudHandler** — full CRUD for providers, history, templates, reminder-settings, send endpoint
- **Go: NetworkHandler extensions (network_ext.go)** — router CRUD (`get/update/delete/:id`), router test-connection/detect-public-ip/interfaces/isolation-settings/ping-olt/setup-isolir/uplinks/status/import, OLT list/import/template, OLT-routers, ODC/ODP/OTB import+template, OTB stats+get, fiber paths CRUD+trace, joint closures CRUD+import, nodes CRUD, servers, paths, detect-NAS, assign customer to ODP, customers with GPS location
- **Go: AdminJobsHandler** — admin registrations workflow (list/get/approve/reject/mark-installed/request-info/tech-survey), customer-registrations CRUD, admin jobs approvals/stats/escalate/submit-approval/materials/approval-history/recurring, technician jobs (list/get/complete/customer-data/generate-credentials), team jobs
- **Go: MiscHandler** — health/db, health/radius, RADIUS protocol stubs (authorize/post-auth/coa), PPPoE search+upload-photo+traffic+bulk+check-isolation+batch-status+send-notification+sync-mikrotik, coordinator auth portal (OTP/verify/logout/session/stats/tasks), public homepage, company info, NAS list, email broadcast, notification helpers, pay-by-token, payment gateway config, inventory variance+reorder
### Files
- `internal/api/handlers/customer_ext.go` — created
- `internal/api/handlers/whatsapp_crud.go` — created
- `internal/api/handlers/network_ext.go` — created (extends network.go)
- `internal/api/handlers/admin_jobs.go` — created
- `internal/api/handlers/misc_handler.go` — created
- `internal/api/router.go` — registered ~130 new batch 7 routes

---

## [2.34.6] — 2026-05-13
### Added
- **Go: PPPoE extended handler** — `user-status`, `export-users`, `bulk-create`, `bulk-status`, `check-isolation`, `send-notification`, `sync-mikrotik`, `user-activity`, `extend`, `mark-paid`, `export-customers`, `bulk-create-customers`, `sync-profiles-mikrotik`, `sync-profiles-radius`, `sync-radius`, `list-with-filters`
- **Go: Technician portal handler** — full mobile app auth (OTP, verify, login, logout, session), profile, work-orders, tasks, customers CRUD, form-data, isolated/offline users, sessions, tickets, monitor, GenieACS proxy, file upload
- **Go: Upload handler** — `POST /api/upload/logo|payment-proof|pppoe-customer`, `GET /api/uploads/logos/:filename`, `GET /api/pwa/icon`
- **Go: WhatsApp extended handler** — broadcast, broadcast-invoice, provider status/QR/restart/test, public webhook
- **Go: Push extended handler** — agent-subscribe/unsubscribe, technician-subscribe/unsubscribe
- **Go: Settings GenieACS handler** — devices list/detail/parameters/reboot/refresh, tasks, test, parameter-display, virtual-parameters, isolation templates CRUD, restart-services, realtime-sessions, system-radius, SSE voucher-updates
- **Go: TelegramHandler.SendHealth** — `POST /api/telegram/send-health`
- **Go: 6 new models** — `TelegramBackupSettings`, `WorkOrder`, `TechnicianOtp`, `PushBroadcast`, `AgentPushSubscription`, `TechnicianPushSubscription`
### Files
- `internal/api/handlers/pppoe_ext.go` — created
- `internal/api/handlers/technician_portal.go` — created
- `internal/api/handlers/upload.go` — created
- `internal/api/handlers/whatsapp_ext.go` — created
- `internal/api/handlers/push_ext.go` — created
- `internal/api/handlers/settings_genieacs.go` — created
- `internal/api/handlers/telegram_handler.go` — added `SendHealth`
- `internal/api/router.go` — registered all new routes (~80 new routes)
- `internal/db/models/extra.go` — added 6 new GORM models

---

## [2.34.5] — 2026-05-13
### Added
- **Go: 17 new handler files** — notifications, public, freeradius, invoices_ext, referrals, admin_users, technician_admin, activity_log, hotspot_ext, voucher_templates, ticket_ext, analytics, settings_ext, backup_handler, telegram_handler, push_handler, olt_ext
- **Go: Notification routes** — `GET/PUT /api/notifications`, `DELETE /api/notifications/:id`
- **Go: Public routes (no auth)** — `GET /api/public/company|areas|profiles|stats|payment-gateways`, `POST /api/public/upload-registration`
- **Go: FreeRADIUS management** — `GET /api/freeradius/status|logs|radcheck|config/list|config/read`, `POST /api/freeradius/start|stop|restart|radtest|config/save`
- **Go: Root-level invoice routes** — `GET/POST/DELETE /api/invoices`, counts, generate, export, send-reminder, send-reminders-bulk, by-token, PDF
- **Go: Referral routes** — `GET/PUT/DELETE /api/admin/referrals`, `GET/PUT /api/admin/referrals/config`
- **Go: Admin User CRUD** — `GET/POST/PUT/DELETE /api/admin/users/:id`, `GET/PUT /api/admin/users/:id/permissions`
- **Go: Technician Admin CRUD** — `GET/POST/PUT/DELETE /api/admin/technicians/:id`
- **Go: Activity Log** — `GET /api/admin/activity-logs`
- **Go: Hotspot extensions** — bulk generate/delete, export, resync, validate, send-whatsapp, delete-expired, rekap-voucher, agent balance/history
- **Go: Voucher Templates CRUD** — `GET/POST/PUT/DELETE /api/voucher-templates/:id`
- **Go: Ticket extensions** — categories CRUD, stats, messages, dispatch
- **Go: Analytics** — `GET /api/admin/analytics`, `GET /api/dashboard/analytics|traffic`
- **Go: Settings extensions** — email templates, test email, timezone, map settings, email history
- **Go: Backup** — history, create (mysqldump+gzip), delete, download, restore, telegram settings
- **Go: Telegram & Push notification routes**
- **Go: OLT alert management** — list, get, resolve; monitoring, metrics
- **Go: Health endpoint** — `GET /api/health`
- **Go: `generateID()` helper** — shared UUID generator in handlers package
### Fixed
- **ticket_ext.go** — `Preload("Customer")` (was `Preload("User")`), `assigned_to_id` column
- **invoices_ext.go** — `user.Profile.Price` direct access (Profile is not a pointer)
### Files
- `internal/api/handlers/helpers.go` — added `generateID()` helper
- `internal/api/handlers/notifications.go` — new
- `internal/api/handlers/public.go` — new
- `internal/api/handlers/freeradius.go` — new
- `internal/api/handlers/invoices_ext.go` — new
- `internal/api/handlers/referrals.go` — new
- `internal/api/handlers/admin_users.go` — new
- `internal/api/handlers/technician_admin.go` — new
- `internal/api/handlers/activity_log.go` — new
- `internal/api/handlers/hotspot_ext.go` — new
- `internal/api/handlers/voucher_templates.go` — new
- `internal/api/handlers/ticket_ext.go` — new
- `internal/api/handlers/analytics.go` — new
- `internal/api/handlers/settings_ext.go` — new
- `internal/api/handlers/backup_handler.go` — new
- `internal/api/handlers/telegram_handler.go` — new
- `internal/api/handlers/push_handler.go` — new
- `internal/api/handlers/olt_ext.go` — new
- `internal/api/router.go` — registered all 17 new handlers (~140 new routes)

## [2.34.4] — 2026-05-13
### Added
- **Sidebar: Permintaan Top-Up & Suspend** — tambah `nav.topupRequests` (`/admin/topup-requests`) dan `nav.suspendRequests` (`/admin/suspend-requests`) sebagai child PPPoE
- **Sidebar: ODC, ODP, Peta Jaringan** — tambah 3 item ke Topology: Network Map, ODC, ODP
- **Sidebar: Fiber ODC & Fiber ODP** — tambah ke seksi Manajemen Fiber
- **Sidebar: GenieACS Files** — tambah child `nav.files` ke seksi GenieACS
- **Sidebar: Kelola Teknisi** — tambah item standalone di catManagement
- **Sidebar: Log Aktivitas** — tambah item standalone di catManagement
- **Sidebar: Pengaturan Keamanan** — tambah child `/admin/settings/security` ke settingsMenu
- **Sidebar: WhatsApp jadi submenu** — ubah dari single link ke children (Settings, Riwayat, Template, Kirim, Notifikasi, Providers)
- **i18n: tambah nav keys** — `topupRequests`, `suspendRequests`, `activityLogs`, `security`, `fiberOdcs`, `fiberOdps`
### Files
- `src/app/admin/AdminClientLayout.tsx` — tambah menu items, WhatsApp jadi submenu, import UserCog
- `src/locales/id.json` — tambah 6 nav translation keys


### Added
- **Go: GenieACS proxy handler** — `POST /api/genieacs/devices/:deviceId/wifi` (TR-069 setParameterValues), `POST /api/genieacs/devices/:deviceId/connection-request`, `GET /api/genieacs/tasks`, `DELETE /api/genieacs/tasks/:taskId`
- **Go: GenieACS settings** — `GET/POST /api/settings/genieacs` (simpan host/username/password ke DB)
- **Go: Admin Employees full CRUD** — `GET /api/admin/employees` (enhanced: stats byRole, pagination, filters), `POST /api/admin/employees`, `PUT /api/admin/employees/:id`, `DELETE /api/admin/employees/:id`
- **Go: Job Assignments routes** — `GET /api/admin/job-assignments` (alias untuk jobH.List), `DELETE /api/admin/job-assignments/:id`
- **Go: GenieacsSettings model** — `genieacs_settings` table
### Files
- `internal/api/handlers/genieacs.go` — baru
- `internal/api/handlers/employees_admin.go` — baru
- `internal/api/handlers/jobs.go` — tambah DeleteJob
- `internal/db/models/extra.go` — tambah GenieacsSettings model
- `internal/api/router.go` — tambah semua routes batch 4

---

## [2.34.2] — 2026-05-13
### Added
- **Go: Manual Payments handler** — `GET/POST /api/manual-payments`, `PUT /api/manual-payments/:id` (approve/reject dengan extend expiry + buat transaction), `DELETE /api/manual-payments/:id`
- **Go: Jobs handler** — `GET/POST /api/admin/jobs`, `GET /api/admin/jobs/stats`, `GET /api/admin/jobs/:id`, `PATCH /api/admin/jobs/:id/status`
- **Go: Employees list** — `GET /api/admin/employees` (for job assignment dropdown)
- **Go: Users list with ODP/ODC filter** — `GET /api/users/list`
- **Go: Employee, JobAssignment, OdpCustomerAssignment models**
### Changed
- **Go: ManualPayment model** — updated sesuai schema actual (bankName, accountName, transferDate, reviewedBy, dll)
- **Go: PppoeUser model** — tambah Router + ODPAssignment relations
- **Go: NetworkODP model** — tambah Status field + ODC relation
### Fixed
- **Go: billing.go** — fix ManualPayment struct creation sesuai model baru
### Files
- `internal/api/handlers/manual_payments.go` — baru
- `internal/api/handlers/jobs.go` — baru
- `internal/api/handlers/pppoe.go` — tambah ListUsersForSelect
- `internal/db/models/models.go` — ManualPayment + PppoeUser update
- `internal/db/models/extra.go` — Employee, JobAssignment, OdpCustomerAssignment, NetworkODP update
- `internal/api/router.go` — manual-payments, jobs, employees, users/list routes

---

## [2.34.1] — 2026-05-13
### Added
- **Go: Inventory handler** — `GET/POST/PUT/DELETE /api/inventory/categories`, `/suppliers`, `/items`; `GET/POST /api/inventory/movements` dengan stock transaction
- **Go: Keuangan handler** — `GET/POST/DELETE /api/keuangan/transactions` (dengan stats totalIncome/Expense/balance), `GET/POST /api/keuangan/categories`, `GET /api/keuangan/export`
- **Go: InventoryCategory, InventoryItem, InventorySupplier, InventoryMovement models**
### Changed
- **Go: Transaction model** — tambah field `Reference`, `CreatedBy`, `JournalEntryID`; `CategoryID` jadi non-nullable string
### Files
- `internal/api/handlers/inventory.go` — baru
- `internal/api/handlers/keuangan.go` — baru
- `internal/db/models/extra.go` — inventory models + Transaction update
- `internal/api/router.go` — inventory + keuangan routes

---

## [2.34.0] — 2026-05-13
### Added
- **Go: Settings handler** — `GET/POST /api/settings/email`, `GET/PUT /api/settings/isolation`, `GET/PUT /api/settings/company` alias
- **Go: Permissions handler** — `GET /api/permissions`, `GET/PUT /api/permissions/role/:role`, `GET /api/permissions/role-templates`
- **Go: Customer portal extended** — 14 new endpoints: `/me`, `/dashboard`, `/packages`, `/auto-renewal`, `/notifications`, `/payment-history`, `/usage`, `/topup-request`, `/suspend-request` (GET/POST/DELETE), `/tickets` (GET/POST)
### Changed
- **Go: Ticket model** — update schema sesuai DB (`ticketNumber`, `customerId`, `customerName`, `description`, `categoryId`, dll); fix `CloseTicket` ke status `CLOSED`
- **Go: Company model** — tambah isolation fields (`isolationIpPool`, `isolationServerIp`, `isolationRateLimit`, dll) dan `bankAccounts`
- **Go: SuspendRequest model** — tambah `startDate`, `endDate`, `adminNotes`, `approvedAt`, `approvedBy`
- **Go: Customer portal** — fix `GetInvoices` query dari `user_id/created_at` ke `userId/createdAt`
### Added (models)
- `EmailSetting`, `Permission`, `RolePermission`, `Notification`, `TicketCategory` models
### Files
- `internal/api/handlers/settings.go` — baru
- `internal/api/handlers/permissions.go` — baru
- `internal/api/handlers/customer_portal.go` — extended (14 new methods)
- `internal/api/handlers/ticket.go` — fix Preload, status casing
- `internal/db/models/models.go` — Company + Ticket struct update
- `internal/db/models/extra.go` — SuspendRequest update + 5 new models
- `internal/api/router.go` — settings + permissions + customer portal routes

---

## [2.33.2] — 2026-05-13
### Fixed
- **Go: Prisma-style NamingStrategy** — tambah custom GORM NamingStrategy yang convert PascalCase → camelCase secara global, mengatasi semua error `Unknown column` (updated_at, is_active, expired_at, job_type, dll)
- **Go: CronHistory column tags** — tambah explicit `column:` tag untuk `jobType`, `startedAt`, `completedAt`
- **Go: Cron queries** — fix semua snake_case query di scheduler.go dan pppoe_session_sync.go (`expiresAt`, `expiredAt`, `dueDate`, `subscriptionType`, `isActive`, `autoIsolationEnabled`, `groupName`)
- **Go: syncNASClients** — simplify ke count-only karena `nas` table sudah adalah tabel router app, tidak perlu INSERT ke FreeRADIUS
- **Go: OLT poller update map** — fix key names ke camelCase (`lastPollAt`, `totalOnu`, `onlineOnu`, `isOnline`)
- **Go: ONU upsert columns** — fix `serialNumber`, `rxPower`, `lastSeenAt`, `updatedAt`, `oltId`, `onuId`
- **Auth: agent login** — fix `isActive` column name (was `is_active`)
### Files
- `internal/db/db.go` — tambah `prismaStyleNamer` custom NamingStrategy
- `internal/db/models/models.go` — CronHistory column tags
- `internal/cron/scheduler.go` — fix camelCase column queries
- `internal/cron/pppoe_session_sync.go` — fix column queries + simplify syncNASClients
- `internal/olt/poller/poller.go` — fix update map keys + ONU upsert columns
- `internal/api/handlers/auth.go` — fix isActive query

---

## [2.33.1] — 2026-05-13
### Added
- **Go: Sessions Handler** — `GET /api/sessions` (list active PPPoE/hotspot sessions dengan user info, pagination, filter), `POST /api/sessions/disconnect`, `POST /api/sessions/sync` (cleanup stale), `GET /api/sessions/export` (CSV)
- **Go: Admin Isolated Users** — `GET /api/admin/isolated-users` dengan unpaid invoice summary
- **Go: Admin Topup Requests** — `GET/POST(approve/reject) /api/admin/topup-requests/:id`
- **Go: Admin Suspend Requests** — `GET/POST(approve/reject) /api/admin/suspend-requests/:id`
- **Go: Registrations CRUD** — `GET/PUT/DELETE /api/registrations/:id` + alias dari `/api/pppoe/registrations`
- **Go: Dashboard alias** — `/api/dashboard/stats` dan `/api/dashboard/revenue-chart` alias ke admin stats
### Fixed
- **Go OLT Poller** — `monitoringEnabled` (camelCase) diperbaiki dari `monitoring_enabled` yang salah menyebabkan error `Unknown column`
### Files
- `internal/api/handlers/sessions.go` — file baru; 4 endpoints + stale session cleanup
- `internal/api/handlers/admin.go` — tambah IsolatedUsers, TopupRequests, ApproveTopup, RejectTopup, SuspendRequests, ApproveSuspend, RejectSuspend
- `internal/api/handlers/pppoe.go` — tambah GetRegistration, UpdateRegistration, DeleteRegistration; ListRegistrations kini support filter by status
- `internal/api/router.go` — daftarkan semua routes baru + alias
- `internal/olt/poller/poller.go` — fix column name `monitoringEnabled`

---

## [2.33.0] — 2026-05-13
### Added
- **Go Cron: PPPoE Session Sync** — port penuh dari `pppoe-session-sync.ts`; sync radacct ↔ radcheck/radreply dengan GREATEST/LEAST safeguard untuk mencegah int overflow di MariaDB; mutex lock agar tidak overlap jika satu run lambat
- **Go Cron: FreeRADIUS Health Check** — sinkronisasi tabel `nas` otomatis dari `routers`, ganti/tambah entri NAS jika ada router baru atau secret berubah
- **Go Cron: Session Security Monitor** — tutup sesi aktif untuk user yang sedang di-isolasi agar RADIUS memutus koneksi mereka
- **Go Cron: Invoice Catch-up** — generate invoice yang hilang untuk user `isolated`/`stopped` yang belum punya invoice bulan ini
- **Go Cron: Agent Sales Recording** — catat transaksi penjualan agen setiap jam; hitung dan kredit komisi berdasarkan `agent.commission` (%)
- **Scheduler** — diperluas dari 4 menjadi 9 registered jobs; `TriggerJob()` support 9 named jobs
### Files
- `internal/cron/pppoe_session_sync.go` — file baru; 5 fungsi cron + helper `importOrphan`, `createOrphanUser`, `syncNASClients`, `isDuplicateKey`
- `internal/cron/scheduler.go` — tambah 5 job registrations + perbarui `TriggerJob()` switch

---

## [2.32.2] — 2026-05-13
### Fixed
- **System Info API: silent git errors** — Semua `execSync` git di `/api/admin/system/info` kini pakai `stdio: 'pipe'` sehingga stderr tidak bocor ke PM2 log; `getAppDir()` kini mencari `/var/www/salfanet-frontend` lebih dulu (direktori dengan `.git`) sebelum fallback ke path lain
### Files
- `src/app/api/admin/system/info/route.ts` — tambah `stdio: 'pipe'` pada `execSync`/`execFileSync`, perbarui urutan kandidat `getAppDir()`

---

## [2.32.1] — 2026-05-11
### Fixed
- **PPPoE Session Sync error 1264** — `acctsessiontime` di-clamp ke range INT MariaDB (`GREATEST(0, LEAST(..., 2147483647))`) pada semua 4 UPDATE query; sesi dengan `acctstarttime` tidak valid (`0000-00-00` atau sangat lama) tidak lagi menyebabkan cron gagal
### Files
- `src/server/jobs/pppoe-session-sync.ts` — clamp TIMESTAMPDIFF ke INT range, tambah filter `acctstarttime > '2000-01-01'` pada update aktif

---

## [2.32.0] — 2026-05-11
### Added
- **Centralized Cron Schedule Management** — jadwal semua cron job kini bisa diatur dari satu halaman Admin → Settings → Cron tab "Jadwal Cron"; perubahan disimpan ke DB `cron_schedule_config`, aktif setelah `pm2 restart salfanet-cron`
- **Schedule Editor modal** — 17 preset waktu (Every minute, Every 5 min, dll.) + custom cron expression; menampilkan default schedule sebagai referensi
- **3-tab layout cron page** — Tab: Status & Trigger, Jadwal Cron, Riwayat Eksekusi
- **API `/api/cron/schedules`** — GET/PUT/DELETE untuk manajemen schedule override per job (SUPERADMIN only)
- **DB table `cron_schedule_config`** — menyimpan override schedule per jobType
### Changed
- **`runner.ts`** — load schedule overrides dari DB saat startup; fallback ke default jika tidak ada override atau tabel belum ada; support `preload.cjs` mock untuk `server-only`
- **`jobs.config.ts`** — hapus `import 'server-only'` guard (redundant; diganti comment penjelasan)
### Fixed
- **Duplicate `CronSettingsPage` declaration** — page.tsx memiliki dua `export default function CronSettingsPage()` yang menyebabkan build error Turbopack; baris duplikat dihapus
- **`server-only` module block tsx cron runner** — `src/cron/preload.cjs` mocking module `server-only` sebelum tsx load file apapun agar standalone cron runner bisa berjalan
### Files
- `src/app/admin/settings/cron/page.tsx` — rewrite lengkap dengan 3-tab layout + ScheduleEditor modal
- `src/app/api/cron/schedules/route.ts` — NEW: CRUD API untuk schedule override
- `src/cron/runner.ts` — load schedule overrides dari DB via `initSchedules()`
- `src/cron/preload.cjs` — NEW: mock `server-only` agar tsx bisa load server files
- `src/cron/runner-wrapper.cjs` — NEW: CJS wrapper entry point (opsional)
- `src/server/jobs/jobs.config.ts` — hapus `import 'server-only'`
- `prisma/schema.prisma` — tambah model `cronScheduleConfig`

---

## [2.31.12] — 2026-05-11
### Fixed
- **MikroTik timeout empty error message** — `node-routeros` melempar empty string `""` saat timeout (bukan `Error` object); sekarang ada fallback message yang jelas jika error kosong atau `{}`
- **Library timeout conflict** — `node-routeros` internal timeout diset ke 9999s agar tidak interferensi dengan `Promise.race` timeout kita yang memberikan pesan error yang lebih informatif
### Files
- `src/server/services/mikrotik/client.ts` — set library timeout ke 9999s, tambah fallback untuk empty error message

---

## [2.31.11] — 2026-05-11
### Fixed
- **MikroTik API error diagnosis** — error message kosong setelah "Failed to connect to MikroTik:" karena `node-routeros` melempar non-Error object; diperbaiki dengan serialisasi yang robust (handle `string`, plain object, `Error`)
- **MikroTik firewall hint di UI** — saat VPN ping OK tapi API gagal timeout, UI menampilkan perintah `/ip firewall filter add` yang persis harus dijalankan di MikroTik terminal
- **Port fallback skip jika sama** — jika `port == apiPort`, tidak perlu coba SSL fallback (menghindari koneksi redundan)
### Files
- `src/server/services/mikrotik/client.ts` — fix error serialization, update timeout message
- `src/app/api/network/routers/test/route.ts` — tambah field `diagnosis`, skip SSL fallback jika port sama
- `src/app/admin/network/routers/page.tsx` — tampilkan perintah MikroTik firewall saat VPN OK tapi API blocked

---

## [2.31.10] — 2026-05-11
### Fixed
- **MikroTik API port auto-detect** — koneksi API gagal jika port berbeda dari default; sekarang test connection mencoba `port` (8728, non-SSL) terlebih dahulu, lalu fallback ke `apiPort` (8729, SSL/TLS) jika gagal; jika berhasil di port berbeda, form otomatis di-update ke port yang benar
- **MikroTik TLS/SSL support** — tambah opsi `tls: true` di `MikroTikConnection` untuk support API-SSL (port 8729) dengan self-signed certificate
### Files
- `src/server/services/mikrotik/client.ts` — tambah field `tls` di `MikroTikConfig`, pass `{ rejectUnauthorized: false }` ke RouterOSAPI untuk self-signed cert
- `src/app/api/network/routers/test/route.ts` — coba primary port dulu, fallback ke SSL port; return `usedPort` dan `usedTls` dalam response
- `src/app/admin/network/routers/page.tsx` — kirim kedua port (`port` dan `apiPort`) ke test endpoint; auto-update form jika port yang berhasil berbeda

---

## [2.31.9] — 2026-05-11
### Fixed
- **Mobile scroll semua halaman admin** — 60+ halaman admin tidak bisa di-scroll di iOS/Android; penyebab: `overflow-hidden` pada root div `bg-background relative overflow-hidden`; dihapus dari semua halaman (background blur effects sudah punya `overflow-hidden` pada `absolute inset-0` child mereka sendiri)
- **Customer offline page** — halaman offline customer portal tidak bisa di-scroll; `overflow-hidden` pada root dipindah ke background div
### Files
- `src/app/admin/**/*.tsx` — hapus `overflow-hidden` dari root div pada 60+ halaman admin
- `src/app/customer/offline/page.tsx` — hapus `overflow-hidden` dari root, pindah ke background div

---

## [2.31.8] — 2026-05-10
### Fixed
- **Mobile scroll blocked** — halaman admin/customer/technician tidak bisa di-scroll di iOS/Android; penyebab: `overflow-hidden` pada root `min-h-screen` div di semua portal layout; dipindah ke background `fixed inset-0` div agar tidak clipping konten
- **Touch overlay block scroll** — notifikasi bell dropdown admin & customer memblokir touch scroll di halaman; ditambah `touch-none` pada `fixed inset-0 z-40` overlay
- **PPPoE tambah pelanggan** — form terpotong di viewport mobile karena `h-full max-h-screen`; diganti `min-h-screen`
- **Halaman publik scroll** — `daftar/page.tsx` dan `pay/[token]/page.tsx` pakai `overflow-hidden` pada root; dihapus agar konten bisa di-scroll
### Files
- `src/app/admin/AdminClientLayout.tsx` — hapus `overflow-hidden` dari root div, pindah ke background div
- `src/app/customer/CustomerClientLayout.tsx` — sama + `touch-none` pada bell overlay
- `src/app/technician/TechnicianPortalLayout.tsx` — `touch-none` pada notif overlay
- `src/app/admin/pppoe/users/new/page.tsx` — `h-full max-h-screen` → `min-h-screen`
- `src/app/daftar/page.tsx` — hapus `overflow-hidden` dari root
- `src/app/pay/[token]/page.tsx` — hapus `overflow-hidden` dari root

---

## [2.31.7] — 2026-05-10
### Fixed
- **HTTPS domain radius.hotspotapp.net** — Install SSL certificate via Let's Encrypt certbot; nginx dikonfigurasi port 443 SSL + redirect HTTP→HTTPS; Cloudflare SSL mode Full sudah support
### Files
- `nginx-frontend.conf` — tambah SSL certificate, port 443, HTTP redirect; ganti IP ke domain `radius.hotspotapp.net`

---

## [2.31.6] — 2026-05-10
### Fixed
- **Health endpoint version** — `/api/health` selalu return `"unknown"` karena `npm_package_version` tidak tersedia di Next.js standalone; diganti dengan build-time env `APP_VERSION` dari `next.config.ts`
- **PM2 startup** — `pm2 save` + systemd `pm2-root.service` verified enabled agar services auto-restart setelah reboot VPS
### Files
- `next.config.ts` — inject `APP_VERSION = pkg.version` as build-time env
- `src/app/api/health/route.ts` — fallback `process.env.APP_VERSION`

---

## [2.31.5] — 2026-05-10
### Fixed
- **Session expiry** — NextAuth session maxAge diubah dari 2 jam menjadi 30 hari; updateAge dari 15 menit menjadi 1 jam
### Files
- `src/server/auth/config.ts` — session maxAge: 30 days, updateAge: 1 hour
- `package.json` — bump version to 2.31.5

---

## [2.31.4] — 2026-05-11
### Fixed
- **nginx routing: `/api/*` ke Next.js** — Root cause 401 errors: nginx salah routing semua `/api/*` ke Go backend (port 8080), padahal admin panel Next.js menggunakan NextAuth session cookies (bukan JWT Bearer). Semua `/api/*` sekarang diarahkan ke Next.js (port 3000). Go backend tetap berjalan di port 8080 untuk akses langsung / WebSocket OLT.
- **Cron jobs audit** — Tidak ada cron lama dari billing-radius Next.js. Semua cron (vpn-watchdog, wg-peer-watchdog, salfanet-cleanup) adalah script VPS yang valid.
### Architecture Note
- Go backend (port 8080): `GET /ws/olt/:id` WebSocket + direct API access
- Next.js (port 3000): semua `/api/*` routing via NextAuth session
- Migrasi frontend ke Go JWT adalah pekerjaan berikutnya (bukan dalam scope ini)
### Files
- `nginx-frontend.conf` — hapus `location /api/` → Go; WebSocket `/ws/` tetap ke Go, semua `/` ke Next.js

---

## [2.31.3] — 2026-05-10
### Fixed
- **Postbuild: copy `.next/static` to standalone** — Static assets (CSS, JS, fonts) were returning 404/wrong MIME type because postbuild script did not copy `.next/static` into `.next/standalone/.next/static`; all browser console errors resolved
### Files
- `package.json` — postbuild now copies `.next/static` to `.next/standalone/.next/static`
- `baileys_whatsapp_patch/package.json` — same fix

---

## [2.31.2] — 2026-05-10
### Added
- **Next.js frontend deployed on VPS** — Full stack running at `http://103.151.140.110`: Go API backend (port 8080) + Next.js frontend (port 3000) behind nginx reverse proxy
- **Database schema migrated** — `prisma db push` applied all 100+ tables to MariaDB `salfanet_radius`; custom SQL migrations confirmed already included in schema
- **PM2 process management** — `salfanet-frontend` and `wa-service` managed by PM2, auto-start on boot via `pm2-root.service` systemd unit
- **nginx proxy updated** — `/api/*` → `:8080` (Go), `/ws/*` → `:8080` (Go), `/*` → `:3000` (Next.js)
- **Company seed data** — Initial company record seeded via `npm run db:seed:company`
### Files
- `nginx-frontend.conf` — nginx config template with frontend proxy
### Notes
- Admin login: `http://103.151.140.110/admin/login`
- Customer portal: `http://103.151.140.110/customer/login`
- API health: `http://103.151.140.110/api/system/health`

---

## [2.31.1] — 2026-05-10
### Fixed
- **CustomerAuthMiddleware** — Replace placeholder with real DB-backed session validation (`customer_sessions` table, token lookup, expiry check)
- **Customer OTP send** — Plug in `notify.SendOTP` in `CustomerLogin` handler (was TODO)
### Files
- `internal/api/middleware/auth.go` — `NewCustomerAuthMiddleware(db)` factory, real session DB lookup
- `internal/api/handlers/auth.go` — Import `notify` package, call `SendOTP` on customer login
- `internal/api/router.go` — Pass DB to `NewCustomerAuthMiddleware`
- `vps-install/wa-package.json` — Valid package.json for wa-service npm install on VPS

---

## [2.31.0] — 2026-05-10
### Added
- **Go backend Phase 2+ — Full API migration** — Complete Go backend covering all major feature domains. Zero Next.js dependency for API layer.
  - `internal/db/models/extra.go` — Extra GORM models: HotspotProfile, HotspotVoucher, Agent, AgentSale, AgentDeposit, TransactionCategory, Transaction, NetworkODC/ODP/OTB, PaymentGateway, RegistrationRequest, SuspendRequest, PushSubscription, WhatsappHistory, WhatsappReminderSetting, TicketReply
  - `internal/radius/radius.go` — FreeRADIUS service: direct MySQL manipulation (radcheck/radreply/radusergroup), isolate/unisolate, rate-limit upsert, session query
  - `internal/notify/whatsapp.go` — WA sidecar HTTP client (POST to wa-service.js :3001/send), phone normalization, invoice/payment/isolation/activation templates
  - `internal/cron/scheduler.go` — robfig/cron v3 scheduler (Asia/Jakarta TZ): generate invoices (00:01), send reminders (hourly), auto-isolate unpaid (00:05), sync voucher expiry (5min); manual trigger API; CronHistory tracking
  - `internal/api/handlers/pppoe.go` — PPPoE areas, profiles, customers, users CRUD + suspend/activate/isolate/unisolate + radius sync + registrations approve/reject
  - `internal/api/handlers/billing.go` — Invoices CRUD + pay (ManualPayment + WA notify) + monthly generation + transactions + payment gateway webhooks (Midtrans/Xendit/Duitku/Tripay)
  - `internal/api/handlers/radius.go` — RADIUS user management, active sessions, stats, soft disconnect
  - `internal/api/handlers/hotspot.go` — Hotspot profiles + voucher batch generation/management
  - `internal/api/handlers/agent.go` — Agent CRUD + sales/deposits + balance topup + voucher assignment
  - `internal/api/handlers/network.go` — Network map: OLT list + ODC/ODP/OTB/Router CRUD
  - `internal/api/handlers/whatsapp.go` — WhatsApp providers + templates + manual send + history + reminder settings
  - `internal/api/handlers/ticket.go` — Support tickets: list/create/get/update/reply/close
  - `internal/api/handlers/company.go` — Company settings get/update
  - `internal/api/handlers/cronhandler.go` — Cron history + manual job trigger API
  - `internal/api/handlers/customer_portal.go` — Customer self-service: profile, invoices, pay, push-subscribe
  - `internal/api/middleware/auth.go` — Added `CustomerAuthMiddleware` for customer portal
  - `internal/api/handlers/auth.go` — Added `AgentLogin` endpoint (phone + PIN → JWT)
  - `internal/api/router.go` — Full route wiring for all domains (PPPoE, Billing, Radius, Hotspot, Agent, Network, WhatsApp, Tickets, Company, Cron, Customer)
  - `cmd/server/main.go` — Wire FreeRADIUS service + cron scheduler into startup/shutdown lifecycle
  - `vps-install/install-go.sh` — Full VPS clean install script (Go 1.23, Node.js 20, PM2, Nginx, UFW, FreeRADIUS, systemd)
### Fixed
- `internal/api/handlers/helpers.go` — `pageParams` now accepts `fiber.Ctx` directly instead of a custom interface (Fiber v3 variadic `Query` signature)
- `internal/api/handlers/billing.go` — Removed redundant `strconv` import and workaround stub
### Files
- `internal/db/models/extra.go` — new
- `internal/radius/radius.go` — new
- `internal/notify/whatsapp.go` — new
- `internal/cron/scheduler.go` — new
- `internal/api/handlers/helpers.go` — updated
- `internal/api/handlers/pppoe.go` — new
- `internal/api/handlers/billing.go` — new (fixed)
- `internal/api/handlers/radius.go` — new
- `internal/api/handlers/hotspot.go` — new
- `internal/api/handlers/agent.go` — new
- `internal/api/handlers/network.go` — new
- `internal/api/handlers/whatsapp.go` — new
- `internal/api/handlers/ticket.go` — new
- `internal/api/handlers/company.go` — new
- `internal/api/handlers/cronhandler.go` — new
- `internal/api/handlers/customer_portal.go` — new
- `internal/api/middleware/auth.go` — updated
- `internal/api/handlers/auth.go` — updated (AgentLogin)
- `internal/api/router.go` — updated (full route wiring)
- `cmd/server/main.go` — updated (wire radius + cron)
- `vps-install/install-go.sh` — new

### Added
- **Go backend Phase 1 — OLT Monitoring** — Full Go backend scaffolded alongside the existing Next.js frontend. Compiles to a single binary (`bin/server.exe`), Fiber v3 HTTP framework, GORM (MySQL, shares existing DB), zerolog structured logging.
  - `cmd/server/main.go` — entrypoint with graceful SIGTERM/SIGINT shutdown
  - `internal/config/config.go` — godotenv-based config (all env vars from `.env`)
  - `internal/db/db.go` — GORM connection pool (25 max open, 10 idle, 5min lifetime)
  - `internal/db/models/` — GORM models mirroring Prisma schema (OLT, ONU, alerts, metrics, customers, invoices, …)
  - `internal/olt/snmp/` — gosnmp walk/get thin wrapper (thread-safe, per-call session)
  - `internal/olt/telnet/` — persistent Telnet pool (max 3 sessions/OLT, 30s keepalive)
  - `internal/olt/vendors/zte/` — ZTE C320 V2.1: concurrent SNMP walk + Telnet authoritative ONU discovery, register/deregister, ONU types, T-CONT profiles
  - `internal/olt/poller/` — per-OLT polling goroutines, DB upsert, alert detection, WebSocket broadcast
  - `internal/ws/hub.go` — WebSocket broadcast hub (fasthttp upgrader, OLT-scoped subscriptions)
  - `internal/api/` — Fiber v3 router, JWT Bearer middleware, auth/admin/OLT handlers
  - `Makefile`, `Dockerfile`, `docker-compose.yml`, `.air.toml` — build/deploy tooling
### Files
- `cmd/server/main.go` — new
- `internal/**/*.go` — new (24 files)
- `Makefile`, `Dockerfile`, `docker-compose.yml`, `.air.toml` — new

## [2.29.63] — 2026-05-09
### Fixed
- **Ghost ONU "N/A" unregistered dari SNMP stale seen-table** — Root cause: `ZTE_V21_SEEN_ONU_TABLE` SNMP walk mengembalikan ONU ID yang stale/pernah tersambung sebelumnya (bukan ONU fisik aktif). Code lama menambah SNMP IDs yang tidak ada di `uncfgSerials` Telnet sebagai entri kosong → tampil di UI sebagai ONU Unregistered ke-3 dengan serial "N/A". Fix: track `hadTelnetData` — jika globalUncfgMap sudah dibangun (Telnet global berhasil), jangan tambahkan ID dari SNMP seen-table. Telnet dipercaya sebagai sumber otoritatif. SNMP fallback hanya digunakan saat Telnet benar-benar tidak tersedia.
- **Per-port Telnet extra call saat globalUncfgMap tersedia** — `else if (globalUncfgMap?.has(portKey))` salah: jika port tidak ada di map (0 ONU uncfg), jatuh ke per-port Telnet call. Fix: cek `globalUncfgMap !== null` dulu.
### Files
- `src/lib/olt/vendors/zte.ts` — `discoverPonV21`: tambah `hadTelnetData` flag, fix kondisi globalUncfgMap check

## [2.29.62] — 2026-05-09
### Fixed
- **Register ONU 422 — false positive dari MOTD login OLT** — Root cause: error detection di POST register menggunakan `errorKeywords = ['failure', ...]` dengan `lowerOutput.includes('failure')`. OLT ZTE C320 selalu menampilkan MOTD setelah login: `"0 authentication failures happened"` → keyword `failure` match → handler return 422 meski registrasi berhasil. Fix: ganti broad keyword matching dengan deteksi yang spesifik terhadap pola CLI error: (1) baris diawali `%` (ZTE/Huawei CLI error prefix), (2) `invalid input`, (3) `invalid command`, (4) `already exist`, (5) `command not found`. MOTD/banner teks tidak akan ter-trigger.
### Files
- `src/app/api/olt/[id]/onus/register/route.ts` — POST: replace broad `errorKeywords.includes()` dengan line-by-line CLI error pattern matching

## [2.29.61] — 2026-05-09
### Fixed
- **ONU Type dropdown salah isi (Telnet artifacts)** — Root cause dua masalah: (1) Command yang dipakai `show gpon onu-type` tidak valid di ZTE C320 V2.1 — command yang benar adalah `show onu-type`. (2) Parser `parseZteOnuTypes` punya catch-all regex `^([A-Za-z0-9_][A-Za-z0-9_.-]*)` yang memungut kata apa saja dari awal baris, termasuk artefak sesi Telnet seperti `Connected`, `Trying`, `Welcome`, `Last`, `spawn`, `ince`, `ZXAN#show`. Fix: (1) Ubah command ke `show onu-type`. (2) Tambah parser untuk format `ONU type name:  <name>` (output `show onu-type`). (3) Hapus catch-all regex, ganti dengan filter eksplisit untuk header/attribute lines.
### Files
- `src/app/api/olt/[id]/onus/register/route.ts` — command: `show gpon onu-type` → `show onu-type`; parser: tambah `ONU type name:` pattern, hapus catch-all

## [2.29.60] — 2026-05-09
### Fixed
- **OLT Import Template 404** — `GET /api/network/olts/template` dan `POST /api/network/olts/import` tidak ada (route files belum dibuat). Tombol "Download Template Excel" dan upload import Excel di halaman `/admin/network/olts` selalu 404/error. Fix: buat kedua route. Template menghasilkan file `.xlsx` dengan contoh 2 baris (kolom: name, ipAddress, latitude, longitude, vendor, model, snmpCommunity, snmpPort, telnetPort, username, password, pollingInterval). Import route mem-parse Excel, validasi IP + vendor, skip duplicate IP, dan insert via Prisma.
### Files
- `src/app/api/network/olts/template/route.ts` — baru: GET → download OLT_Import_Template.xlsx
- `src/app/api/network/olts/import/route.ts` — baru: POST → import OLT dari Excel

## [2.29.59] — 2026-05-09
### Fixed
- **OLT Delete 500 — FK constraint `network_otbs.oltId`** — Root cause: `network_otbs` tabel punya kolom `oltId` dengan foreign key ke `networkOLT` tanpa `onDelete: SetNull`. Saat OLT dihapus, MySQL raise FK constraint violation → handler return 500. Fix: (1) Di DELETE handler `/api/network/olts` tambah `prisma.network_otbs.updateMany({ ... data: { oltId: null } })` sebelum `networkOLT.delete`. (2) Schema Prisma diperbarui: tambah `onDelete: SetNull` ke `network_otbs.olt` relation.
- **TSC — `ZteServiceTemplate` & `RegisterMetadata` tidak terdefinisi** — Kedua tipe digunakan di `ONURegisterModal` component di `src/app/admin/olt/[id]/page.tsx` tapi tidak pernah dideklarasikan. Fix: tambah `type ZteServiceTemplate = 'basic' | 'zte_full' | 'huawei_full' | 'fiberhome_veip'` dan `interface RegisterMetadata { onuTypes, tcontProfiles, trafficProfiles, suggestedOnuId, detectedOnuType }` sebelum component. Efek domino: ini juga memperbaiki 4 error "Parameter 'type'/'profile' implicitly has an 'any' type" di map callbacks (karena sebelumnya `metadata` bertipe `any`).
- **TSC — `assign/route.ts` customer.customerId type mismatch** — `serializeOnuAssignment` function mendeclare `customer.customerId: string` tapi Prisma query include menghasilkan `customerId: string | null`. Fix: ubah ke `customerId: string | null`.
- **TSC — `detail/route.ts` type predicate state mismatch** — `.filter()` predicate mendeclare `state: string | null` tapi TypeScript meng-infer `state: string` dari `parts[3] ?? null` (tanpa `noUncheckedIndexedAccess`, `string[]` index akses menghasilkan `string`, bukan `string | undefined`). Fix: ubah predicate ke `state: string`. Ini sekaligus memperbaiki error "candidate is possibly null" di `.find()` callbacks (karena sebelumnya TS tidak bisa narrow array type akibat predicate yang broken).
### Files
- `src/app/api/network/olts/route.ts` — DELETE handler: unlink `network_otbs` sebelum delete OLT
- `prisma/schema.prisma` — `network_otbs.olt`: tambah `onDelete: SetNull`
- `src/app/admin/olt/[id]/page.tsx` — tambah `ZteServiceTemplate` type dan `RegisterMetadata` interface
- `src/app/api/olt/[id]/onus/[onuId]/assign/route.ts` — `serializeOnuAssignment`: `customerId: string | null`
- `src/app/api/olt/[id]/onus/[onuId]/detail/route.ts` — filter predicate: `state: string`

## [2.29.58] — 2026-05-09
### Fixed
- **Poller fully SNMP — hapus per-ONU Telnet serial fallback dari `discoverPonV21`** — Root cause: `discoverPonV21` menampung ONU yang gagal di-parse serial-nya dari SNMP hex ke dalam array `needsTelnetSerial`, lalu memanggil `show gpon onu detail-info gpon-onu_1/B/P:ID` via Telnet secara paralel (N `Promise.all` sessions) untuk setiap ONU dengan serial null. Jika ada banyak ONU dengan format serial non-standar (misalnya byte non-ASCII), ini bisa spawn banyak Telnet sessions secara serentak, yang saturates OLT concurrent session limit. Fix: hapus `needsTelnetSerial` array dan `Promise.all` Telnet block. Jika SNMP hex tidak bisa di-parse, `serialNumber` tetap `null` di DB — ONU masih ter-track via `onuId`. Serial bisa diisi on-demand saat user buka detail ONU (endpoint individual yang masih boleh Telnet). Polling cycle sekarang pure SNMP untuk registered ONUs.
### Files
- `src/lib/olt/vendors/zte.ts` — `discoverPonV21`: hapus `needsTelnetSerial` array + `Promise.all` Telnet serial lookup block

## [2.29.57] — 2026-05-09
### Fixed
- **Poller lambat — per-ONU Telnet optical info calls dihapus** — Root cause: `upsertONU` di poller memanggil `vendor.getOnuOpticalInfo(telnetConfig, ...)` untuk setiap ONU satu-per-satu, meski `discoverPonV21` sudah mengambil `rxPower` dan `distance` via 7 SNMP walks paralel. Dengan 400+ ONU aktif, ini berarti 400+ sesi Telnet sequential per polling cycle (~8–35s masing-masing = potensi ratusan detik). Fix: skip Telnet optical info call jika `onu.rxPower !== null` (artinya SNMP sudah menyediakan data). Telnet optical info tetap digunakan sebagai fallback hanya jika SNMP tidak menghasilkan rxPower.
- **SSH path dihapus dari optical info fallback** — ZTE C320 V2.1 hanya mendukung Telnet CLI; SSH tidak dikonfigurasi. Path `sshConfig` untuk `getOnuOpticalInfoSSH` dihapus dari `upsertONU` agar tidak terjadi double-attempt.
### Files
- `src/lib/olt/poller.ts` — `upsertONU`: skip Telnet/SSH optical info if `onu.rxPower !== null` (SNMP-sourced)

## [2.29.56] — 2026-05-09
### Fixed
- **Port Map sync lambat setelah hapus VLAN gagal** — Root cause: uplink POST menggunakan `timeout: 20` untuk sesi Telnet konfigurasi. Dengan 2 attempt (`removeVlan`) yang keduanya gagal, total waktu zombie sessions bisa mencapai 2 × 35 detik = 70 detik. ZTE C320 membatasi concurrent Telnet sessions; chassis sync yang menyusul tidak bisa langsung konek. Fix:
  1. Timeout Telnet untuk POST action dikurangi ke **8 detik** (perintah config di LAN lokal selesai <3s; 8s cukup buffer).
  2. Loop `commandAttempts` sekarang **break** jika terjadi connection-level failure (`!result.success`) — tidak ada gunanya mencoba command berbeda jika OLT tidak bisa dikoneksi. Retry (continue) hanya terjadi pada **CLI error** (command ditolak OLT, bukan koneksi gagal).
### Files
- `src/app/api/olt/[id]/uplink/route.ts` — POST action: `timeout: 8`, break on connection failure

## [2.29.55] — 2026-05-09
### Fixed
- **removeVlan di uplink tag → 500** — Root cause: satu sesi Telnet mengirim `no switchport vlan X tag`, `no switchport default vlan`, `no switchport vlan X` sekaligus; perintah fallback yang tidak berlaku di ZTE C320 mengembalikan `%Error`, `firstError` menjadi true → 500. Fix: pisahkan menjadi dua `commandAttempts` terpisah — percobaan pertama hanya `no switchport vlan ${vid} tag`, fallback percobaan kedua `no switchport default vlan`. Setiap percobaan adalah sesi Telnet independen sehingga error dari satu tidak mengontaminasi yang lain.
- **ONU Type tidak terbaca di Register ONU** — Root cause: `show run | include onu-type` pada ZTE C320 V2.1 tidak mendukung pipe filter sehingga menghasilkan seluruh running-config atau timeout, menyebabkan sesi `executeMultipleCommands` 5-command gagal dan semua data (onuTypes, tcontProfiles, trafficProfiles, suggestedOnuId, detectedOnuType) kembali kosong. Fix: ganti ke 5 panggilan `executeCommand` paralel (`Promise.allSettled`) — satu command per sesi Telnet, kegagalan satu tidak mempengaruhi lainnya. Command ONU types diganti ke `show gpon onu-type`.
- **parseZteOnuTypes** — Diperbarui untuk menangani output format tabel dari `show gpon onu-type` (`ZTEG-F670L  F670L GPON ONT`) selain format running-config lama (`onu-type ZTEG-F670L gpon ...`). Header line (`Onu-type`, `---`) dan kata kunci umum di-skip.
### Files
- `src/app/api/olt/[id]/uplink/route.ts` — removeVlan: pisah jadi 2 commandAttempts terpisah
- `src/app/api/olt/[id]/onus/register/route.ts` — ONU type GET: `Promise.allSettled(5x executeCommand)` + `show gpon onu-type` + parser update

## [2.29.54] — 2026-05-09
### Fixed
- **VLAN tab masih kosong (Mode/TLS/Tagged VLANs —)** — Root cause: `executeMultipleCommands(['show vlan port …', 'show running-config interface …'])` kadang gagal/hang karena `show vlan port xgei_1/3/2` tidak valid atau menyebabkan sesi Telnet terganggu. Fix: VLAN tab sekarang hanya menggunakan satu `executeCommand('show running-config interface …')` yang sudah terbukti bekerja. `parseRunningConfigInterface` mengekstrak `Mode`, `TLS`, `Tagged Vlan` (dari `switchport vlan 1,30,69,100,151 tag` — comma-separated), `Description`, `Speed`, `Duplex`, `Flow Control`, `Physical Type`.
### Changed
- **Chassis stats row dihapus dari Port Map** — Baris stat (UPTIME, AVG CPU, AVG MEMORY, ACTIVE CARDS, FAN STATUS) di dalam panel "ZTE C320 Rack Diagram" dihapus. AVG CPU (11%) dan AVG MEMORY (32%) adalah static placeholder yang menyesatkan; FAN STATUS dan ACTIVE CARDS belum real-time. Tampilan chassis sekarang langsung ke rack diagram.
- **Tab Metrics dihapus** — Tab "Metrics" dihapus dari halaman OLT detail (ONU List | Port Map | Alerts | Settings | Logs). State `metrics`, `metricsHours`, `metricsLoading`, callback `fetchMetrics`, dan `useEffect`-nya juga dibersihkan.
### Files
- `src/app/api/olt/[id]/uplink/route.ts` — VLAN tab: ganti multi-command ke single `executeCommand('show running-config interface')`
- `src/app/admin/olt/[id]/page.tsx` — hapus chassis stats row; hapus Metrics TabsTrigger + TabsContent + state vars

## [2.29.53] — 2026-05-10
### Fixed
- **Temperature card dihapus dari top stat** — Kartu "Temperature" di baris 4 stat card atas selalu "Not available (C320)" karena ZTE C320 tidak melaporkan suhu via SNMP. Dihapus → layout sekarang 3 kolom (Status, Uptime, ONUs). Grid berubah dari `md:grid-cols-4` → `md:grid-cols-3`.
- **CHASSIS TEMP dihapus dari chassis stats row** — Kolom "CHASSIS TEMP" di stats row chassis diagram juga dihapus (selalu "Unknown"). Grid chassis stats: dari `xl:grid-cols-6` → `xl:grid-cols-5`, border logic diperbarui.
- **VLAN tab kosong (Mode/TLS/PVID semua —)** — `parseVlanPort` hanya menangani format key:value, tapi ZTE C320 bisa mengembalikan format tabular (`VLAN Port Mode Pvid TLS`). Ditambahkan tabular fallback parser. `parseRunningConfigInterface` juga diperluas untuk menangani ZTE non-switchport style (`vlan N tag`, `pvid N`, `mode hybrid` tanpa prefix `switchport`). VLAN tab sekarang selalu return raw output untuk diagnosis bahkan jika parsing gagal.
- **CONFIG tab kosong (No configuration data)** — Config tab tidak menampilkan apa pun jika `hasCliError` triggered pada output. Diubah: raw output selalu dikembalikan ke UI, user bisa melihat output asli OLT meski ada error-string di output.
- **`parseVlanPort` "Tagged vlan(s)" tidak dikenali** — Normalisasi key sebelumnya hanya mencocokkan `'tagged vlan'` (exact). Sekarang menggunakan `startsWith('tagged vlan')` sehingga varian `tagged vlan(s)` juga ditangkap.
### Files
- `src/app/admin/olt/[id]/page.tsx` — hapus Temperature card top stats; hapus CHASSIS TEMP dari chassis stats row
- `src/app/api/olt/[id]/uplink/route.ts` — tabular fallback di `parseVlanPort`; ZTE non-switchport variants di `parseRunningConfigInterface`; VLAN tab & CONFIG tab selalu return raw

## [2.29.52] — 2026-05-10
### Fixed
- **Uplink tab lambat (status/vlan/optical)** — Setiap tab dulu membuka 2 sesi Telnet terpisah (primary + fallback) masing-masing ~5 detik → total ~10 detik per tab. Sekarang: satu `executeMultipleCommands` session mengirim primary + fallback command sekaligus. Total: ~5 detik → **2× lebih cepat**.
- **SNMP fallback getInterfaceStatusSNMP lambat** — Dulu: `snmpWalk(ifDescr)` sequential, lalu baru `Promise.all([4 snmpGet])`. Sekarang: 5 `snmpWalk` berjalan **paralel** (ifDescr, ifAdmin, ifOper, ifHighSpeed, ifAlias) + O(1) Map lookup per index.
- **removeVlan 3 sesi Telnet** — `removeVlan` POST dulu loop 3 `commandAttempts` masing-masing sesi Telnet terpisah (~15 detik worst-case). Sekarang: satu sesi tunggal dengan ketiga `no switchport` command dikirim sekaligus — OLT mengabaikan command yang tidak berlaku.
- **OLT sync lambat (`discoverPonV21`)** — Dulu: per-ONU `Promise.all([5 snmpGet])` secara sequential antar ONU = N × 5 subprocess spawns. Sekarang: **7 `snmpWalk` paralel** untuk seluruh PON port sekaligus (regStatus, operState, serial, rxPower, description, distance, seenTable), lalu build Map + lookup O(1) per ONU. Telnet serial lookup yang gagal SNMP di-batch via `Promise.all` tanpa blokir ONU lain.
### Added
- **PVID Management di VLAN tab** — Sebelumnya PVID hanya tampil read-only. Sekarang: tombol **Remove** di samping PVID aktif, dropdown berubah antara "Tagged (Trunk)" dan "Set as PVID", tombol berubah label antara "Add VLAN" dan "Set PVID". Backend: actions `setPvid` dan `removePvid` baru di POST `/api/olt/[id]/uplink`.
- **Remove VLAN button lebih jelas** — Badge VLAN tagged kini punya tombol `×` yang lebih terlihat dengan hover merah dan border animasi.

### Files
- `src/app/api/olt/[id]/uplink/route.ts` — `executeMultipleCommands` per tab; 5 parallel walks SNMP; `removeVlan` single session; `setPvid`/`removePvid` actions baru
- `src/lib/olt/vendors/zte.ts` — `discoverPonV21` rewrite: 7 parallel bulk walks → O(1) Map lookup; batch Telnet serial fallback
- `src/app/admin/olt/[id]/page.tsx` — PVID remove button; "Set as PVID" dropdown; improved VLAN remove UX

## [2.29.51] — 2026-05-09
### Fixed
- **Port Map masih lambat (chassis API)** — Root cause: dua sesi Telnet terpisah (`show card` + `show interface port-status`) masing-masing ~5 detik, ditambah SNMP fallback sequential. Sekarang: satu sesi Telnet tunggal via `executeMultipleCommands` menjalankan kedua command sekaligus, dan seluruh SNMP walk (PON table + 5 IF-MIB walk) dijalankan **paralel** bersama Telnet. Total waktu: dari ~10-15 detik → ~5 detik (50%+ lebih cepat).
- **SNMP uplink fallback N×4 GET calls** — Sebelumnya, per-interface SNMP fallback melakukan 4 `snmpGet` shell spawn terpisah per interface. Diganti dengan 5 `snmpWalk` paralel (ifDescr, ifAdminStatus, ifOperStatus, ifHighSpeed, ifAlias) + O(1) lookup dari Map.
- **Triple retry Telnet** — Pola lama: Telnet1 → (gagal) SNMP → (gagal) Telnet2 lagi. Sekarang: satu Telnet multi-cmd + SNMP paralel, tidak ada retry berlebihan.

### Files
- `src/app/api/olt/[id]/chassis/route.ts` — `executeMultipleCommands` single session; `fetchSNMPChassisData` bulk walks paralel; `buildUplinkStatesFromSNMP` O(1) lookup; hapus `loadUplinkPortStatesSNMP` + `loadUplinkPortStates` lama.

## [2.29.50] — 2026-05-09
### Changed
- **OLT Monitoring — redesign terbaik** — Monitoring page dirancang ulang dengan: ONU progress bar per OLT dengan persentase warna adaptif (hijau/kuning/merah), countdown auto-refresh 30d dengan indikator visual, tombol "Poll Semua" parallel, sort by status/name/alerts/offline-ONU, timestamp relative ("2m lalu"), animasi ping pada OLT online, badge alert mengarah ke halaman alerts, suhu color-coded (hijau < 50°C, kuning 50–65°C, merah ≥ 65°C), styling card berlapis dengan border aksen sesuai kondisi.
- **OLT Alerts — redesign terbaik** — Alerts page dirancang ulang dengan: border-left accent berwarna per severity (merah/amber/biru/abu), tombol "Selesaikan Semua" batch resolve, relative timestamp dengan tooltip tanggal penuh, back-button ke monitoring, stat card adaptif warna jika ada critical/warning, link OLT name menuju halaman detail OLT, resolved alerts tampil transparan (opacity-60).

### Files
- `src/app/admin/olt/monitoring/page.tsx` — Redesign monitoring: progress bar, countdown, poll-all, sort, relative time
- `src/app/admin/olt/alerts/page.tsx` — Redesign alerts: border accent, resolve-all, relative time, back nav

## [2.29.49] — 2026-05-09
### Fixed
- **ONU unconfigured tetap hanya 1 setelah sync** — Discovery ZTE C320 sekarang menjadikan output CLI `show gpon onu uncfg` sebagai sumber utama untuk ONU unconfigured. Data CLI global tidak lagi diproses hanya jika SNMP seen-table punya entry; port yang hanya muncul dari CLI global juga ditambahkan ke daftar PON yang dipoll.
- **Port Map lambat saat membaca uplink** — Status uplink GE/XGE di chassis sekarang memakai satu command cepat `show interface port-status`, lalu SNMP IF-MIB hanya sebagai fallback. Ini menghindari multi-command/per-interface Telnet untuk port map.
- **Komposisi warna OLT Monitoring dan Alerts** — Card, filter, dan action surface di halaman OLT Monitoring/Alerts diperhalus memakai slate surface yang konsisten untuk light/dark mode.

### Files
- `package.json` — bump versi aplikasi ke `2.29.49`.
- `src/lib/olt/vendors/zte.ts` — CLI global uncfg authoritative; merge port PON dari CLI uncfg ke discovery; unregistered IDs tidak lagi bergantung pada seen-table SNMP.
- `src/app/api/olt/[id]/chassis/route.ts` — port map uplink memakai parser `show interface port-status` satu kali, SNMP status jadi fallback.
- `src/app/admin/olt/monitoring/page.tsx` — perbaikan palette light/dark pada card/filter/control.
- `src/app/admin/olt/alerts/page.tsx` — perbaikan palette light/dark pada card/filter/control.

## [2.29.48] — 2026-05-09
### Fixed
- **ONU unconfigured kedua masih hilang meski versi 2.29.47 sudah terpasang** — Parser ZTE sebelumnya masih bisa membuang serial fallback jika satu ONU unconfigured sudah lebih dulu terpetakan ke `onuId` nyata. Sekarang serial fallback selalu digabung ke hasil akhir dengan virtual ID stabil, sehingga kombinasi `1 ONU ada ID + 1 ONU hanya serial` tidak lagi berakhir jadi satu baris.

### Files
- `package.json` — bump versi aplikasi ke `2.29.48`.
- `src/lib/olt/vendors/zte.ts` — merge fallback serial list ke `uncfgSerials` meski sebagian ONU unconfigured sudah punya ID nyata.

## [2.29.47] — 2026-05-09
### Fixed
- **ONU unconfigured masih hanya tampil 1 padahal CLI ada 2** — Pembacaan ZTE `show gpon onu uncfg` tidak lagi memetakan serial CLI secara posisi ke `onuId` dari SNMP seen-table. Sekarang parser memakai entri CLI aktual per port; jika CLI tidak memberi `onuId`, sistem membuat virtual ID yang stabil dari serial ONU agar semua ONU unconfigured tetap muncul di DB/UI.
- **Tab Uplink VLAN / Config / Optical belum sesuai output ZTE C320** — VLAN sekarang fallback ke `show running-config interface`, sehingga `switchport mode`, `switchport tls`, `description`, dan daftar `switchport vlan ... tag` terbaca langsung dari config nyata seperti output `xgei_1/3/2`. Tab Optical juga fallback ke `show interface optical-module-info` agar data SFP C320 bisa dibaca selain `show ddmi interface`.
- **Halaman OLT detail kurang responsif dan hardcoded gelap** — Modal uplink dan rack panel dirapikan untuk mobile/desktop (`max-height`, scroll, grid responsif), dan komponen yang disentuh sekarang memakai warna light/dark yang lebih konsisten.

### Files
- `package.json` — bump versi aplikasi ke `2.29.47`.
- `src/lib/olt/vendors/zte.ts` — parser ONU unconfigured ZTE sekarang memakai CLI entries aktual dan virtual ID stabil berbasis serial.
- `src/app/api/olt/[id]/uplink/route.ts` — tambah parser `show running-config interface` dan `show interface optical-module-info`; VLAN/optical fallback diperbaiki.
- `src/app/admin/olt/[id]/page.tsx` — modal uplink dan rack panel dibuat lebih responsif serta theme-aware.

## [2.29.46] — 2026-05-09
### Fixed
- **GE uplink port tidak muncul / selalu 400 Invalid port name** — Validasi regex sebelumnya hanya menerima format 3-level (`gei_1/3/1`) padahal ZTE C320 SMXA card gunakan format 2-level untuk GE port: `gei_1/3`. Regex diperbarui ke `(?:gei|xgei)_\d+\/\d+(?:\/\d+)?` yang menerima keduanya. Fix pada GET dan POST endpoint.
- **Chassis diagram tampilkan 3 port GE palsu per SMXA slot** — SMXA (plain) hanya punya 1 GE port per slot, bukan 3. Port names dikoreksi: `gei_1/{slot}` (2-level, satu GE) + `xgei_1/{slot}/1`, `xgei_1/{slot}/2` (dua XGE). Sesuai output `show interface ?` pada ZTE C320.

### Files
- `src/app/api/olt/[id]/uplink/route.ts` — Regex validasi port name diperbarui di GET dan POST handler.
- `src/app/api/olt/[id]/chassis/route.ts` — `smxaUplinkPorts`: SMXA plain → `gei_1/{slot}` + `xgei_1/{slot}/1-2`; fallback default juga diperbarui.

## [2.29.45] — 2026-05-09
### Fixed
- **Uplink STATUS tab data tidak tampil** — Command sebelumnya `show interface gei_1/x/x` menghasilkan output key-value yang tidak konsisten di ZTE C320. Diganti ke `show interface port-status gei_1/x/x` yang menghasilkan output tabular dengan kolom eksplisit: hybrid Status, Native VLAN, Negotiation, Speed (Mbps), Duplex, Flow-Ctrl, Admin Status, Link. Fallback ke `show interface` (key-value) + SNMP IF-MIB tetap dipertahankan jika tabular gagal.
- **Uplink CONFIG tab menampilkan config sintetis** — Sebelumnya config dibuat dari hasil parse `show interface` + `show vlan port` secara manual. Sekarang menggunakan `show running-config interface gei_1/x/x` yang menghasilkan config asli dari OLT (termasuk switchport mode, VLAN list, phy-attribute, dll).

### Files
- `src/app/api/olt/[id]/uplink/route.ts` — Tambah `parseInterfacePortStatus` (parser tabular); STATUS tab pakai `show interface port-status` dulu lalu fallback; CONFIG tab pakai `show running-config interface` langsung.

## [2.29.44] — 2026-05-09
### Fixed
- **Uplink port STATUS "Admin: Unknown · Port: Unknown"** — `show interface gei_1/x/x` via Telnet kadang gagal (ZTE session limit, SMXA card OFFLINE, atau output format berbeda). Sekarang API menambahkan fallback ke SNMP IF-MIB (ifAdminStatus + ifOperStatus + ifHighSpeed + ifAlias) sehingga status port tetap terbaca meski Telnet tidak responsif.
- **Uplink STATUS tidak lagi mengembalikan 503 jika Telnet tidak dikonfigurasi** — STATUS tab sekarang bisa jalan via SNMP saja; hanya VLAN/config/optical tab yang membutuhkan Telnet.

### Files
- `src/app/api/olt/[id]/uplink/route.ts` — Import `snmpWalk/snmpGet/SNMPConfig`; tambah `getSnmpConfig` + `getInterfaceStatusSNMP`; STATUS tab try Telnet dulu, fallback ke SNMP IF-MIB jika gagal.

## [2.29.43] — 2026-05-09
### Fixed
- **ONU unregistered serial tidak terbaca / hanya 1 ONU muncul** — Tiga bug sekaligus di `discoverONUsSNMP` dan `discoverPonV21`:
  1. **Command salah**: kode menggunakan `show pon onu uncfg` tapi ZTE C320 V2.1 hanya mengenali `show gpon onu uncfg`. Karena command gagal, global uncfg map selalu kosong.
  2. **Regex port tidak cocok**: regex `gpon[_-]olt[_-]` tidak mencocokkan format output asli `gpon-onu_1/1/1:1`. Akibatnya tidak ada entri yang dimasukkan ke map meskipun command benar.
  3. **Kolom serial salah**: output `show gpon onu uncfg` punya 3 kolom (`OnuIndex Sn State`), serial ada di `parts[1]`. Kode lama membaca `parts[2]` yang isinya `State` ("unknown"), bukan serial number.
- **Per-port fallback** — command `show pon onu uncfg gpon-olt_...` juga diubah ke `show gpon onu uncfg gpon-olt_...`; Format A (3-col `gpon-onu_` output) sekarang membaca serial dari `parts[1]`, format 4-col tetap dari `parts[2]`.

### Files
- `src/lib/olt/vendors/zte.ts` — Perbaikan global uncfg command, regex port, dan indeks kolom serial.

## [2.29.42] — 2026-05-09
### Fixed
- **Sync OLT tidak lagi timeout 524** — `pollOLTWithOptions` sekarang dijalankan fire-and-forget (tanpa `await`). API langsung return 202 dengan `background: true`; frontend sudah punya path untuk ini: auto-refresh setelah 30 detik. ZTE C320 SNMP+Telnet discovery bisa memakan waktu >100 detik, sebelumnya menyebabkan Cloudflare/reverse-proxy memutus koneksi dengan status 524, lalu frontend menerima HTML error page dan melempar `SyntaxError: Unexpected token '<'`.

### Files
- `src/app/api/olt/[id]/sync/route.ts` — Ganti `await pollOLTWithOptions(...)` menjadi fire-and-forget dengan `.catch()` error logging; return 202 `{ success, background: true, message }`.

## [2.29.41] — 2026-05-09
### Fixed
- **ONU yang dihapus/unregistered tidak lagi hilang setelah sync** — Akar masalah pertama: `pruneMissingOnus` menghapus baris `auth_failed` (unregistered) saat SNMP discovery tidak mengembalikannya dalam satu siklus (timing ZTE SEEN_ONU_TABLE). Sekarang baris `auth_failed` tidak pernah dipruning oleh poller; baris tersebut tetap terlihat di daftar Unregistered sampai operator mendaftar ulang ONU atau menghapus entri secara manual.
- **Delete ONU tidak lagi memicu poll langsung** — Akar masalah kedua: setelah ONU dihapus dari OLT, ZTE membutuhkan ~10–30 detik sebelum ONU muncul kembali di `SEEN_ONU_TABLE` sebagai unregistered. Poll yang dijalankan segera setelah delete tidak menemukan ONU itu, dan pruner (2.29.40) menghapus baris `auth_failed` yang baru dibuat. Sekarang delete route hanya menandai row sebagai `auth_failed` lalu langsung return — scheduled poller yang akan memperbarui statusnya pada siklus berikutnya.
- **Pengurangan tekanan koneksi Telnet** — Menghilangkan `pollOLTWithOptions` dari delete route mengurangi concurrent Telnet session. Sebelumnya delete + sync + chassis API bisa membuka 3+ sesi bersamaan dan menghabiskan session pool ZTE C320, menyebabkan `show card` gagal dan rack diagram menampilkan fallback SNMP tanpa kartu SMXA.

### Files
- `src/lib/olt/poller.ts` — `pruneMissingOnus` sekarang membaca kolom `status` dan melewati baris `auth_failed`.
- `src/app/api/olt/[id]/onus/[onuId]/delete/route.ts` — Hapus `pollOLTWithOptions` dari delete flow; hanya update DB dan return success.

## [2.29.40] — 2026-05-09
### Fixed
- **Sync OLT kembali benar-benar menjalankan poller** — Endpoint sync tidak lagi melepas `pollOLTWithOptions` secara fire-and-forget. Sekarang request menunggu poll selesai, tetapi memakai mode ringan `skipOpticalInfo` supaya tetap cukup cepat untuk manual refresh dan delete follow-up.
- **ONU yang dihapus tetap muncul di daftar register/unconfigured** — Delete ONU tidak lagi langsung menghapus row `oltOnuStatus` dari database sebelum refresh selesai. Row kini sementara diubah ke status `auth_failed` dan `customerId` dibersihkan, sehingga ONU tetap terlihat sebagai unregistered sampai hasil poll berikutnya mengonfirmasi state live dari OLT.
- **Manual sync/delete lebih ringan** — Poller manual kini bisa melewati query optical-info per ONU agar sync tidak terasa macet dan lebih andal dipakai untuk refresh setelah aksi operasional.

### Files
- `src/lib/olt/poller.ts` — Tambah opsi `skipOpticalInfo` untuk poll ringan pada manual sync/delete.
- `src/app/api/olt/[id]/sync/route.ts` — Kembalikan sync ke mode awaited dengan poll ringan, bukan background fire-and-forget.
- `src/app/api/olt/[id]/onus/[onuId]/delete/route.ts` — Ubah delete flow agar status ONU dipertahankan sebagai unregistered sampai sync selesai.

## [2.29.39] — 2026-05-09
### Fixed
- **Sync OLT tidak lagi error 524 (Cloudflare timeout)** — `pollOLTWithOptions` sebelumnya berjalan sinkron di dalam API handler; untuk OLT dengan banyak ONU, proses bisa >100 detik sehingga Cloudflare membalas HTML error (524) bukan JSON. Sekarang sync dijalankan di background (fire-and-forget), endpoint langsung merespon `202` JSON. Frontend otomatis refresh setelah 30 detik.

### Files
- `src/app/api/olt/[id]/sync/route.ts` — Background sync, return 202 langsung tanpa tunggu poll selesai.
- `src/app/admin/olt/[id]/page.tsx` — Handle response `background:true`, auto-refresh setelah 30 detik.

## [2.29.38] — 2026-05-09
### Fixed
- **Status uplink tidak lagi semua DIS** — Root cause: `smxaUplinkPorts(slot, 'SMXA')` sebelumnya menghasilkan 6 interface per slot (slot+1), sehingga dua SMXA card berbeda (slot 3 dan 4) menghasilkan interface `gei_1/5/X` yang tidak ada. Semua `show interface` gagal → semua port DIS. Sekarang tiap SMXA card menghasilkan 3 port sesuai slotnya sendiri; dua card = 6 port total dari dua row terpisah.
- **SNMP IF-MIB sebagai primary source status uplink** — Ditambahkan `loadUplinkPortStatesSNMP` yang memakai `ifDescr`/`ifAdminStatus`/`ifOperStatus`/`ifHighSpeed`/`ifAlias` dari IF-MIB standard. SNMP dipakai duluan jika OLT `snmpEnabled`; Telnet multi-command hanya sebagai fallback.
- **SMXA OFFLINE tidak lagi disembunyikan** — `isOperationalCard` kini tidak memfilter status `OFFLINE`. Pada ZTE C320, SMXA dengan status OFFLINE di `show card` tetap memiliki port fisik aktif yang harus ditampilkan di rack diagram.

### Files
- `src/app/api/olt/[id]/chassis/route.ts` — Fix `smxaUplinkPorts` SMXA→3 port/slot, tambah `loadUplinkPortStatesSNMP` (IF-MIB), SNMP primary + Telnet fallback, fix `isOperationalCard` biarkan OFFLINE.

## [2.29.37] — 2026-05-09
### Fixed
- **Command uplink ZTE dibetulkan** — Tab `Configuration` tidak lagi memakai `show running-config interface gei_1/...` yang invalid untuk uplink GE/XGE ZTE C320. Data konfigurasi sekarang dibentuk dari command yang valid: `show interface` + `show vlan port`.
- **Aksi VLAN/enable/disable uplink kini benar-benar jalan** — Endpoint POST uplink sebelumnya menjalankan `configure terminal`, `interface`, dan `switchport/shutdown` di sesi Telnet terpisah, sehingga state konfigurasi hilang di tiap langkah. Sekarang satu aksi dijalankan dalam satu sesi Telnet via `executeMultipleCommands`.
- **Parser VLAN uplink lebih konsisten** — Variasi key ZTE seperti `Tagged VLAN` dan `Tagged Vlan` sekarang dinormalisasi supaya mode, PVID, TLS, dan daftar tagged VLAN tampil stabil di UI.

### Files
- `src/app/api/olt/[id]/uplink/route.ts` — Deteksi error CLI ZTE, config uplink sintetis dari command valid, dan eksekusi action uplink dalam satu sesi Telnet.

## [2.29.36] — 2026-05-09
### Fixed
- **SMXA uplink menampilkan 6 port (bukan 3)** — `smxaUplinkPorts` untuk card type `SMXA` plain kini mengembalikan 6 interface: `gei_1/{slot}/1..3` + `gei_1/{slot+1}/1..3`, sesuai hardware ZTE C320 yang portnya tersebar di dua alamat slot.
- **Status port uplink (DIS/UP/DOWN) kini akurat** — `loadUplinkPortStates` sebelumnya memanggil `show interface` tiap port secara parallel (masing-masing buka koneksi Telnet baru), menyebabkan OLT menolak koneksi berlebihan sehingga semua port fallback ke `isEnabled: false` (DIS). Sekarang semua command dijalankan dalam satu sesi Telnet via `executeMultipleCommands`.

### Files
- `src/app/api/olt/[id]/chassis/route.ts` — `smxaUplinkPorts`: SMXA plain → 6 port; `loadUplinkPortStates`: satu sesi Telnet untuk semua `show interface`.

## [2.29.35] — 2026-05-09
### Fixed
- **Diagram ZTE C320 dibuat lebih actual** — Rack view di halaman detail OLT sekarang fokus ke slot service/uplink real, mempertahankan nomor slot actual, menampilkan gap slot kosong, dan tidak lagi mencampur layout MCU ke area card operasional.
- **Status uplink SMXA enable/down dibedakan dengan benar** — Parsing `show interface` kini menangani format real ZTE seperti `is activate, line protocol is up/down`, jadi port admin-up tapi link-down tidak lagi terlihat sebagai disable.
- **Tooltip dan warna port PON lebih informatif** — Port PON sekarang membedakan online, LOS, dying gasp, dan ONU unconfigured agar kondisi slot lebih mudah dibaca dari diagram.

### Files
- `src/app/api/olt/[id]/chassis/route.ts` — Tambah parsing state uplink actual per interface dan hanya tandai card operasional yang benar-benar aktif.
- `src/app/api/olt/[id]/uplink/route.ts` — Perbaiki parser status interface agar membaca format kalimat output ZTE C320.
- `src/app/admin/olt/[id]/page.tsx` — Redesign diagram rack ZTE C320, refresh chassis, dan tampilkan state uplink/service port yang lebih actual.

## [2.29.34] — 2026-05-09
### Added
- **Delete ONU terdaftar + sync OLT** — ONU yang sudah terdaftar sekarang bisa dihapus penuh dari ZTE OLT melalui flow clear config service lalu unregister `no onu`, lalu langsung disinkronkan kembali ke database/frontend.
- **Manual Sync OLT di halaman detail** — Halaman detail OLT sekarang punya tombol `Sync OLT` untuk memaksa refresh data dari OLT walau monitoring terjadwal tidak aktif.

### Fixed
- **Assign customer tidak lagi 500** — Response endpoint assign ONU sekarang aman untuk JSON serialization karena field `BigInt` pada status ONU disanitasi sebelum dikirim balik.
- **Reboot ONU menampilkan error OLT yang lebih nyata** — Route reboot ZTE sekarang mengekstrak output command `reboot` dari transcript Telnet, sehingga kegagalan tidak lagi selalu jatuh ke pesan generik.
- **Sync OLT membersihkan ONU yang sudah hilang di perangkat** — Poller sekarang menghapus row ONU stale yang tidak lagi ditemukan saat polling, sehingga hasil register/delete lebih konsisten antara OLT dan frontend.

### Files
- `src/app/api/olt/[id]/onus/[onuId]/assign/route.ts` — Sanitasi response assign ONU agar tidak gagal serialisasi `BigInt`.
- `src/app/api/olt/[id]/onus/[onuId]/reboot/route.ts` — Perbaiki parsing output reboot Telnet dan error reporting.
- `src/app/api/olt/[id]/onus/[onuId]/delete/route.ts` — Tambah endpoint delete/unregister ONU penuh untuk ZTE + sync setelah aksi.
- `src/app/api/olt/[id]/sync/route.ts` — Tambah endpoint manual sync OLT per perangkat.
- `src/lib/olt/poller.ts` — Tambah mode sync manual dan cleanup ONU stale saat polling.
- `src/app/admin/olt/[id]/page.tsx` — Tambah tombol Sync OLT, Delete ONU, dan refresh otomatis setelah register/reboot.

---

## [2.29.33] — 2026-05-09
### Added
- **Template config di modal register ONU** — Register ONU ZTE sekarang punya pilihan flow `Basic register`, `ZTE Full`, `Huawei Full`, dan `Fiberhome VEIP` langsung di modal, mengikuti struktur wizard referensi `oltc320_v2.1.1_linux`.
- **Traffic profile live dari OLT** — Modal register kini memuat daftar `traffic profile` dari OLT lewat `show gpon profile traffic`, jadi template full tidak lagi bergantung pada input dummy.

### Changed
- **Flow register ZTE selaras ke wizard CLI** — Endpoint register sekarang bisa menerapkan rangkaian command template untuk dual VLAN, VEIP, service-port, WAN DHCP, TR-069, dan ACS sesuai template yang dipilih saat register ONU.

### Files
- `src/app/api/olt/[id]/onus/register/route.ts` — Tambah metadata `trafficProfiles` dan eksekusi template `zte_full`, `huawei_full`, `fiberhome_veip`.
- `src/app/admin/olt/[id]/page.tsx` — Tambah pilihan template config, field template-specific, dan preview command sesuai flow register.

---

## [2.29.32] — 2026-05-09
### Fixed
- **Detail ONU unregistered salah command** — ONU yang belum terdaftar tidak lagi dipaksa memakai `show gpon onu detail-info gpon-onu_...`, karena command itu memang invalid untuk ONU unconfigured. Detail kini memakai `show pon onu uncfg gpon-olt_...` dan menampilkan type/SN/state yang valid dari OLT.
- **Register ZTE masih hardcode `type All`** — Flow register ZTE kini mengikuti wizard referensi `oltc320_v2.1.1_linux`: type ONU yang dipilih dari daftar live OLT dipakai langsung pada command `onu {id} type {onuType} sn {sn}`.

### Added
- **Register metadata live dari OLT** — Modal register sekarang mengambil `ONU Type`, `TCONT profile`, dan suggested ONU ID langsung dari OLT via Telnet, bukan dari array dummy di frontend.
- **Detected ONU type untuk unconfigured ONU** — Modal register/detail menampilkan type ONU hasil baca `show pon onu uncfg`, sehingga admin bisa lihat type aktual sebelum register.

### Files
- `src/app/api/olt/[id]/onus/[onuId]/detail/route.ts` — Branch detail khusus ONU unregistered pakai `show pon onu uncfg`.
- `src/app/api/olt/[id]/onus/register/route.ts` — Tambah GET metadata live dari OLT dan ubah register ZTE agar pakai actual ONU type.
- `src/app/admin/olt/[id]/page.tsx` — Modal register pakai data live OLT untuk ONU type/TCONT/suggested ID.

---

## [2.29.31] — 2026-05-09
### Fixed
- **ONU detail loading lebih cepat** — Endpoint detail ONU ZTE tidak lagi membuka 3 sesi Telnet terpisah. Detail dan running-config kini diambil dalam satu sesi multi-command, dan optical command hanya dipanggil bila data power/jarak belum ada di DB.
- **Pager `--More--` ZTE merusak output detail** — Script Expect sekarang otomatis menekan spasi saat output Telnet dipaginasi, sehingga modal detail tidak lagi menampilkan output terpotong/aneh seperti `ZXAN#xit`.

### Added
- **Detail vendor ONT & service summary** — Modal detail ONU kini menampilkan vendor ONT dari prefix serial, auth mode, SN bind, admin/channel state, DBA/vport/profile, VLAN service, TCONT profile, dan service-port mapping.

### Files
- `src/lib/olt/telnet.ts` — Handle pager `--More--` dan opsi multi-command tanpa `end` paksa.
- `src/app/api/olt/[id]/onus/[onuId]/detail/route.ts` — Multi-command Telnet transcript parser + summary vendor/config ONU.
- `src/app/admin/olt/[id]/page.tsx` — Tambah kartu technical detail dan service summary di modal ONU.

---

## [2.29.30] — 2026-05-09
### Fixed
- **ZTE Telnet login matcher** — Expect script tidak lagi salah menangkap teks `Last login` sebagai prompt `login:`, sehingga command Telnet (`show card`, detail ONU, reboot ONU) benar-benar jalan setelah autentikasi.
- **SMXA card tidak muncul** — Parser `show card` kini mendukung format real ZTE C320 V2.1: `Rack Shelf Slot CfgType RealType Port HardVer SoftVer Status`, termasuk card `SMXA` dan `GTGHG`.
- **ONU serial number registered/unregistered** — Mapping port Telnet ZTE C320 diperbaiki: CLI memakai PON 1-based (`gpon-olt_1/1/1`), sementara DB/UI tetap 0-based. Registered ONU yang SNMP-nya kosong kini fallback ke `show gpon onu detail-info` untuk mengambil `Serial number`.
- **Reboot ONU failed** — Reboot ZTE kini pakai workflow Telnet `configure terminal → pon-onu-mng gpon-onu_... → reboot`, bukan SSH-only command lama.
- **404 `/admin/network/onus`** — Route redirect ditambahkan agar link statistik ONU dari OLT Management tidak lagi 404.

### Added
- **ONU Detail Modal** — Tombol Detail pada ONU List menampilkan detail Telnet (`show gpon onu detail-info`, optical power, running-config) dan data customer/ODP terkait.
- **Assign Customer ONU** — Tombol Assign pada ONU registered untuk menghubungkan ONU ke PPPoE customer (`olt_onu_status.customerId`).

### Files
- `src/lib/olt/telnet.ts` — Fix matcher login dan multi-command Telnet.
- `src/lib/olt/vendors/zte.ts` — Fix mapping CLI port, serial fallback dari detail-info, optical command parser.
- `src/lib/olt/poller.ts` — Simpan serial dari Telnet optical/detail fallback.
- `src/app/api/olt/[id]/chassis/route.ts` — Parser `show card` format Rack/Shelf/Slot.
- `src/app/api/olt/[id]/onus/[onuId]/reboot/route.ts` — Reboot ZTE via Telnet `pon-onu-mng`.
- `src/app/api/olt/[id]/onus/[onuId]/detail/route.ts` — **NEW** detail ONU API.
- `src/app/api/olt/[id]/onus/[onuId]/assign/route.ts` — **NEW** assign customer API.
- `src/app/admin/network/onus/page.tsx` — **NEW** redirect route untuk link lama.
- `src/app/admin/olt/[id]/page.tsx` — Detail/Assign modal dan filter dari query string.

---

## [2.29.29] — 2026-05-13
### Fixed
- **Unregistered ONU serial N/A** — Parser kini gunakan global `show pon onu uncfg` (satu Telnet call) yang hasilkan format `gpon_olt-1/1/0  N/A  ZTEGDA5918AC  unknown`; fallback per-port juga handle format `gpon-onu_1/1/0:N` (prefix berbeda + ONU ID setelah titik dua)
- **Chassis diagram card type** — Sebelumnya hardcoded GTGQ; sekarang baca `show card` via Telnet → card type aktual (GTGHG, SMXA-B, MCUD1)
- **Uplink slot posisi** — Sebelumnya hardcoded slot 15/16 (GICF); sekarang baca slot aktual SMXA dari `show card` (slot 3 & 4 di ZTE C320 ini)

### Added
- **Uplink Port Modal** — Klik port dot SMXA di chassis diagram untuk lihat detail dengan 4 tab: Status, VLAN, Config (running-config), Optical (DDM)
- **Uplink Configuration** — Tambah/hapus tagged VLAN dan enable/disable port lewat modal
- **`/api/olt/[id]/uplink` endpoint** — GET (4 tab) + POST (addVlan, removeVlan, enable, disable, setDescription) dengan validasi input port dan VLAN ID

### Changed
- **Chassis API** — `/api/olt/[id]/chassis` kini: Telnet `show card` sebagai sumber utama, SNMP+DB sebagai fallback. Response tambah field `uplinkIfaces`, `hardVer`, `softVer`, `cardStatus`, `source`

### Files
- `src/lib/olt/vendors/zte.ts` — Fix ONU serial: global uncfg pre-fetch + dual-format parser
- `src/app/api/olt/[id]/chassis/route.ts` — Rewrite: Telnet show card + SNMP fallback
- `src/app/api/olt/[id]/uplink/route.ts` — **NEW**: Uplink port detail & config API
- `src/app/admin/olt/[id]/page.tsx` — UplinkPortModal, ZTEChassisView gunakan data dari chassis API

---

## [2.29.28] — 2026-05-09
### Fixed
- **Unregistered ONU serial number** — Serial number ONU yang belum terdaftar (status `auth_failed`) kini tampil di UI. Fix 2 bug di `zte.ts`:
  1. Port 0-based vs 1-based: command Telnet `show pon onu uncfg gpon-olt_1/{board}/{pon-1}` — sebelumnya salah kirim `pon` (SNMP 1-based), sekarang benar kirim `pon-1` (CLI 0-based)
  2. Regex parser salah format: sebelumnya cari `gpon-onu_` (format ONU terdaftar), kini parse format aktual ZTE C320 — `gpon_olt-1/1/0  N/A  ZTEGDA5918AC  unknown` (field[2] = serial)
- **CPU/Memory/Temp display** — Panel ZTE Chassis kini menampilkan `—` (dash) alih-alih `N/A` untuk metrik yang tidak didukung hardware ZTE C320 V2.1, dengan tooltip penjelasan. Status card Temperature juga menampilkan `—` dan sub-label "Not available (C320)"

### Changed
- **OLT Detail page redesign** — Halaman `/admin/olt/[id]` diperbarui:
  - Status cards (4 kartu): tambah `border-l-4` dengan warna aksen per tipe (hijau/merah untuk status, amber untuk temp, biru untuk uptime, teal untuk ONU). Setiap kartu kini punya sub-label informatif (waktu polling terakhir, vendor, model, jumlah offline)
  - Header: tampilkan vendor badge, model, firmware version di subtitle IP address
  - Tabel ONU: status kini ditampilkan sebagai **pill badge** berwarna (hijau/kuning/merah/abu). Row hover fix dark mode (`dark:hover:bg-gray-800/50`). Header tabel punya `bg-gray-50 dark:bg-gray-900/60`. Padding semua cell konsisten (`py-2.5`). Kolom Actions rapi dengan `rounded-md` dan `transition-colors`
  - Cancel button di confirm-reboot kini support dark mode: `dark:bg-gray-700 dark:text-gray-300`
  - Tabel dibungkus `rounded-lg border border-gray-200 dark:border-gray-800`
- **Command preview block** — Terminal preview di modal Register ONU kini: background `bg-gray-950 dark:bg-black`, border `border-gray-800`, label uppercase tracking, fake blinking block cursor di akhir
- **Input caret color** — Field ONU ID dan VLAN di modal Register ONU kini explicit `caret-gray-900 dark:caret-white` agar cursor terlihat di semua tema

### Files
- `src/lib/olt/vendors/zte.ts` — Fix unregistered ONU serial: port 0-based + regex parser format ZTE C320
- `src/app/admin/olt/[id]/page.tsx` — Redesign status cards, header, ONU table, command preview, cursor color
- `package.json` — Bump ke 2.29.28

---

## [2.29.27] — 2026-05-08
### Added
- **Vendor-aware ONU Registration Modal** — Modal Register ONU di halaman OLT Detail kini otomatis menyesuaikan field dan preview command berdasarkan vendor OLT:
  - **ZTE C320** — ONU Type (All/ZTE-F6xx) + TCONT Profile (1G/100M/…) + Telnet CLI `configure terminal → interface gpon-olt → onu N type All sn SN → tcont/gemport/service-port → end`
  - **Huawei MA5608T/MA5800** — Line Profile ID + Service Profile ID + Telnet CLI `enable → config → interface gpon → ont add → service-port → quit`
  - **FiberHome AN5516/AN6010** — ONU Type (AN5506-04-FA/…) + Service Profile Name + Telnet CLI `enable → config → interface gpon-olt → onu add → onu profile → onu vlan → commit → exit`
- **Vendor-aware Register API** — `POST /api/olt/[id]/onus/register` kini membangun urutan command yang berbeda per vendor berdasarkan referensi `zte_command.py → register_onu_stepbystep()` dari oltc320_v2.1.1_linux
- **ZTE Telnet System Metrics (best-effort)** — Tambah `getSystemMetricsTelnet()` di `zte.ts` yang mencoba `show card` dan `show environment` via Telnet untuk parse CPU/Memory/Temp. Pada ZTE C320 V2.1 akan selalu return null (hardware tidak support), tapi tersedia untuk model ZTE lain (C600/C300)

### Notes
- ZTE C320 V2.1 CPU/Memory/Temp via Telnet tetap tidak tersedia — dikonfirmasi oleh oltc320_v2.1.1_linux CHANGELOG: "Removed unsupported CPU/memory/temperature monitoring". UI menampilkan N/A, perilaku ini sudah benar.

### Files
- `src/app/admin/olt/[id]/page.tsx` — ONURegisterModal rewritten: vendor-aware fields + preview; prop `vendor` ditambahkan ke render call
- `src/app/api/olt/[id]/onus/register/route.ts` — Full rewrite: vendor detection + per-vendor CLI command sequence
- `src/lib/olt/vendors/zte.ts` — Tambah `getSystemMetricsTelnet()` best-effort function

---

## [2.29.26] — 2026-05-08
### Fixed
- **OLT Management ONU count** — Halaman OLT Management (network/olts) sebelumnya selalu menampilkan "0 ONUs" karena API `GET /api/network/olts` tidak menyertakan `_count.onuStatuses`. Kini field `olt_onu_status` (jumlah ONU) dan `onu_stats` (online/offline) disertakan dari field `totalOnu` & `onlineOnu` yang sudah tersimpan di DB setelah polling
- **Password OLT tidak muncul** — Halaman Settings di OLT Detail (`/admin/olt/[id]`) selalu me-reset field password ke kosong saat halaman di-refresh. Kini password diambil dari API response dan ditampilkan. PUT handler juga diubah agar tidak menghapus password yang tersimpan jika field dikirim kosong (hanya update jika ada isi)
- **updater.sh clean build** — Tambah `rm -rf .next` sebelum `npm run build` di updater agar tidak ada artifact build lama yang menyebabkan update tidak efektif / lock file conflict
- **package.json version sync** — Versi di `package.json` kini diselaraskan dengan versi CHANGELOG (sebelumnya masih `2.29.20`)

### Files
- `vps-install/updater.sh` — Tambah `rm -rf .next` sebelum build
- `package.json` — Bump version ke `2.29.26`
- `src/app/api/network/olts/route.ts` — Tambah `_count.onuStatuses` + mapping `olt_onu_status` + `onu_stats`
- `src/app/admin/olt/[id]/page.tsx` — Load `password` dari API response di `fetchOLT`
- `src/app/api/olt/[id]/route.ts` — PUT: skip update password jika kosong

---

## [2.29.25] — 2026-05-08
### Fixed
- **ZTE C320 unregistered ONU discovery** — ONU yang belum diregister (tampak di seen-ONU table SNMP tapi tidak di reg table) kini berhasil di-discover dan disimpan ke DB dengan status `auth_failed`. Serial number diambil via **Telnet** (`show pon onu uncfg gpon-olt_1/{board}/{pon}`) karena SNMP cfg table tidak memiliki entry untuk ONU yang belum register. Parsing mendukung dua format output ZTE C320: `gpon-onu_1/1/1:2  ZTEGDA5918AC` dan `  2  ZTEGDA5918AC`
- **upsertONU serial update** — Kolom `serialNumber` kini ikut di-update ketika polling berikutnya berhasil mendapat serial (sebelumnya hanya disimpan saat create, tidak di-update)
- **discoverONUsSNMP telnet passthrough** — Fungsi `discoverONUsSNMP` kini menerima parameter `telnetConfig` opsional dan meneruskannya ke `discoverPonV21`, memungkinkan fetch serial via Telnet ketika OLT memiliki Telnet enabled
- **Poller telnet passthrough** — `pollOLT` kini meneruskan `telnetConfig` ke `discoverONUsSNMP` agar unregistered ONU dapat memiliki serial

### Files
- `src/lib/olt/vendors/zte.ts` — `discoverPonV21` + Telnet serial fetch untuk unregistered ONU; `discoverONUsSNMP` signature + telnetConfig passthrough
- `src/lib/olt/poller.ts` — Pass `telnetConfig` ke `discoverONUsSNMP`; update `serialNumber` di block update upsert

---

## [2.29.24] — 2026-05-07
### Changed
- **ZTE C320 chassis diagram redesign** — Ganti tampilan horizontal strip (slot chip berjejer) ke layout **vertical rack blade** ala NMS profesional: setiap slot ditampilkan sebagai baris horizontal (card label | port grid | slot number), FAN column di kiri dengan animasi, 6 stats card di header (Uptime, Chassis Temp, Avg CPU, Avg Memory, Active Cards, Fan Status), legend di bawah (Online/Disabled/Admin UP Port DOWN/LOS ONU/Unregistered), dan indikator LED PWR/SYS/ALM di header. Port squares 6×24px berwarna dengan dot di dalam (hijau=online, merah=LOS, oranye=partial, biru=uplink, kosong=slate). Badge kuning kecil muncul di atas port yang punya unregistered ONU.

### Files
- `src/app/admin/olt/[id]/page.tsx` — Rewrite `ZTEChassisView` dari horizontal chip → vertical rack blade NMS-style

---

## [2.29.23] — 2026-05-10
### Added
- **Realistic ZTE C320 chassis diagram** — Halaman detail OLT kini menampilkan diagram front-panel chassis ZTE C320 dengan semua 18 slot: MCU-A (slot 0), 14 service card slots (1–14), 2 uplink slots (15–16, GICF), MCU-B (slot 17), plus FAN dan PWR di kiri/kanan. Setiap slot menampilkan card type label (GTGQ/GTGH/GTGO/GICF/MCUD1), grid port berwarna (hijau=online, oranye=partial, merah=offline, hitam=kosong), dan slot kosong ditampilkan gelap. Chassis disertai panel detail per-port (persentase online, avg RX power) di bawahnya
- **ONU registration modal** — Tombol "Register" muncul di kolom aksi tabel ONU untuk ONU yang berstatus `auth_failed` (unregistered). Tombol membuka modal yang menampilkan: form ONU ID, ONU type (ZTE-F609, F660, F673, F600W, CZTE, All, dll), VLAN, TCONT profile (1G/100M/50M/20M/10M), deskripsi, serta preview command Telnet yang akan dikirim ke OLT
- **ONU register API** (`POST /api/olt/[id]/onus/register`) — Endpoint baru yang membangun dan mengirim command registrasi ZTE via Telnet (`configure terminal → interface gpon-olt → onu … type All sn … → tcont → gemport → service-port → exit/end`) menggunakan `executeMultipleCommands()`
- **Telnet multi-command** (`executeMultipleCommands()` di `telnet.ts`) — Fungsi baru yang membuat expect script untuk mengirim banyak command sekaligus ke OLT via sesi Telnet, menunggu prompt `[>#]` setelah setiap command
- **Chassis API** (`GET /api/olt/[id]/chassis`) — Endpoint baru yang mengembalikan layout slot chassis ZTE C320 beserta data per-port dari DB dan SNMP

### Changed
- **ONU action column** — Untuk ONU unregistered (`auth_failed`), kolom aksi menampilkan tombol "Register" (hijau) alih-alih tombol "Reboot"
- **Port diagram tab** — Diganti dari layout grup horizontal lama (`OLTPortDiagram`) ke komponen `ZTEChassisView` baru yang realistis

### Files
- `src/app/admin/olt/[id]/page.tsx` — Ganti `OLTPortDiagram`/`getOLTTemplate` dengan `ZTEChassisView`; tambah `ONURegisterModal`; tambah state `registeringOnu`; tombol Register di tabel ONU
- `src/app/api/olt/[id]/onus/register/route.ts` — **NEW** POST endpoint registrasi ONU via Telnet
- `src/app/api/olt/[id]/chassis/route.ts` — **NEW** GET endpoint layout chassis
- `src/lib/olt/telnet.ts` — Tambah `executeMultipleCommands()`

---

## [2.29.22] — 2026-05-09
### Added
- **ONU description/name (ZTE V2.1)** — `discoverPonV21()` kini fetch nama ONU dari `zxAnGponOnuCfgTable` col 2 (`.3.28.1.1.2.{ponIndex}.{onuId}`) secara paralel, disimpan ke kolom `description` di DB, dan ditampilkan sebagai kolom "Name" di tabel ONU pada halaman detail OLT
- **ONU distance (ZTE V2.1)** — Jarak ONU ke OLT diambil dari `zxAnGponOnuRegTable` col 21 (`.3.50.12.1.1.21.{ponIndex}.{slot}.{onuId}`) dalam satuan meter, disimpan ke DB, dan ditampilkan sebagai kolom "Distance" di tabel ONU
- **Unregistered ONU discovery (ZTE V2.1)** — Setelah menemukan ONU terdaftar, `discoverPonV21()` kini juga walk tabel `zxAnGponOnuDiscoveredInfoTable` (`.3.27.4.1.1.{ponIndex}`) untuk menemukan semua ONU yang terdeteksi OLT tetapi belum diregistrasi, ditambahkan dengan status `unregistered` (→ DB: `auth_failed`)
- **Parallel SNMP fetches (ZTE V2.1)** — Pengambilan oper-state, serial, RX power, description, dan distance dilakukan secara paralel menggunakan `Promise.all()` per ONU untuk mempercepat polling

### Changed
- **ONU table columns** — Halaman detail OLT kini menampilkan kolom "Name" (deskripsi ONU) dan "Distance" di tabel ONU list; status `auth_failed` kini ditampilkan sebagai "Unregistered" (bukan "Auth failed")
- **`poller.ts` upsertONU** — Kini menyimpan field `description` dari SNMP; `distance` dan `txPower` menggunakan `onu.distance`/`onu.txPower` sebagai fallback jika data optik tidak tersedia

### Files
- `src/lib/olt/vendors/zte.ts` — Update V21 constants (tambah `onuDescription`, `onuDistance`, `ZTE_V21_SEEN_ONU_TABLE`); rewrite `discoverPonV21()` dengan parallel fetch + unregistered ONU discovery
- `src/lib/olt/poller.ts` — `upsertONU()` simpan `description`, gunakan `onu.distance`/`onu.txPower` sebagai fallback
- `src/app/admin/olt/[id]/page.tsx` — Tambah field `description` di interface ONU; tambah kolom Name + Distance di tabel; fix label "Unregistered" untuk status `auth_failed`

---

## [2.29.21] — 2026-05-09
### Fixed
- **ONU discovery ZTE C320 V2.1.0 (CRITICAL)** — `discoverPonV21()` sebelumnya walk OID `.3.50.11.2.1.1` yang tidak ada di firmware V2.1.0, menyebabkan 0 ONU terdiskover. Kini menggunakan tabel registrasi yang telah diverifikasi live via SNMP: walk `zxAnGponOnuRegTable` col 1 (`.3.50.12.1.1.1.{ponIndex}`) untuk menemukan ONU terdaftar, lalu GET serial dari `zxAnGponOnuCfgTable` col 5 (`.3.28.1.1.5`), oper-state dari col 6 registrasi tabel (nilai 5=online), dan RX power dari col 10 (formula: `-(raw/1000)` dBm)
- **ONU RX power ZTE V2.1** — Kini membaca dari kolom 10 tabel registrasi (`3.50.12.1.1.10.{ponIndex}.{slot}.{onuId}`), disimpan sebagai integer positif dalam satuan 0.001 dBm, dikonversi dengan `-(raw/1000)`. Contoh: nilai 9501 → −9.501 dBm (valid GPON)
- **ZTE C320 template port count** — Template diagram ZTE C320 slot 1 diubah dari 8 menjadi 16 port agar sesuai dengan data SNMP aktual (16 PON port pada board 1). `getEffectivePortCount` tetap mengexpand lebih jauh jika ada ONU di port >15
- **Temperature/CPU/memory OIDs V2.1** — OID `C320_TEMP_V21/CPU/MEM` yang menunjuk ke tabel ONU yang salah kini diganti dengan alamat yang lebih tepat; jika tidak accessible, semua metrik return null dan UI menampilkan N/A dengan benar

### Files
- `src/lib/olt/vendors/zte.ts` — Update V21 OID constants; rewrite `discoverPonV21()` dengan OIDs terverifikasi dari SNMP live
- `src/app/admin/olt/[id]/page.tsx` — ZTE C320 template slot 1 portCount: 8 → 16

---

## [2.29.20] — 2026-05-09
### Fixed
- **VPN route persistence (WireGuard)** — `addPeerToConf()` now writes `PostUp`/`PostDown` lines to `wg.conf [Interface]` so local-network routes (e.g. OLT IPs) survive WG interface restarts and VPS reboots
- **VPN route persistence (watchdog WG)** — `vpn-watchdog.sh` (CHECK D) now parses `wg0.conf` every 2 min and re-adds any missing kernel routes for WG peer local networks
- **VPN route persistence (watchdog L2TP)** — `vpn-watchdog.sh` (CHECK E) reads `/etc/salfanet/l2tp/peer-routes.conf` and restores missing L2TP peer local-network routes when ppp0 is up
- **L2TP localNetworks persistence** — `vps-l2tp-peer` API now accepts `localNetworks`, appends idempotent `ip route replace` lines to `/etc/ppp/ip-up.d/99-vpn-routes`, and saves routes to `/etc/salfanet/l2tp/peer-routes.conf`
- **L2TP UI localNetworks** — VPN client page now sends `localNetworks` field when adding an L2TP VPS peer

### Files
- `vpn-watchdog.sh` — Added CHECK D (WireGuard route restoration) and CHECK E (L2TP route restoration)
- `src/app/api/network/vps-wg-peer/route.ts` — PostUp/PostDown persistence in wg.conf
- `src/app/api/network/vps-l2tp-peer/route.ts` — Handle localNetworks: ip-up.d append + peer-routes.conf
- `src/app/admin/network/vpn-client/page.tsx` — Pass localNetworks for L2TP VPS peer creation

---

## [2.29.19] — 2026-05-08

### Fixed
- **Port OLT terbaca hanya 8 dari 16** — `discoverONUsSNMP` sebelumnya hardcode 2 boards × 8 pon. Kini untuk V2.1, fungsi `discoverPONPortsV21()` walk PON port table (`1.3.6.1.4.1.3902.1012.3.11.3.1.1`) secara dinamis, ekstrak semua ponIndex, konversi ke pasangan (board, pon). ZTE C320 dengan 16 port di board 1 kini terdiskover semuanya. Fallback ke 2×8 jika walk gagal.
- **Diagram OLT menampilkan jumlah port yang tidak sesuai** — Port diagram sebelumnya pakai `portCount` dari template hardcode. Kini `getEffectivePortCount()` mengambil `max(templatePortCount, maxPortDariData + 1)` berdasarkan data `onuStatuses` aktual, sehingga diagram otomatis scale ke 16 port jika SNMP menemukan ONU di port 8–15.

### Files
- `src/lib/olt/vendors/zte.ts` — Tambah `ZTE_V21_PON_TABLE` OID, fungsi `discoverPONPortsV21()`, update `discoverONUsSNMP` ke dynamic loop
- `src/app/admin/olt/[id]/page.tsx` — Tambah `maxPortPerSlot` tracking, `getEffectivePortCount()`, gunakan di render port diagram

---

## [2.29.18] — 2026-05-07

### Fixed
- **HTTP 500 pada `/api/olt/[id]`** — BigInt fields (`bandwidthUp`, `bandwidthDown` di `onuStatuses`, serta `uptime`, `rxBytes`, `txBytes`, `rxErrors`, `txErrors` di `performanceMetrics`) tidak dikonversi sebelum JSON serialization. Diperbaiki dengan map eksplisit `Number()` di response.
- **HTTP 500 pada `/api/olt/metrics`** — Sama, BigInt fields di `oltPerformanceMetric` tidak dikonversi. Diperbaiki.
- **Telnet tidak tampilkan username/password** — Saat hanya Telnet enabled (SSH disabled), field username/password tidak muncul di Settings tab OLT detail. Kini username/password tampil di bagian Telnet jika SSH dinonaktifkan.

### Files
- `src/app/api/olt/[id]/route.ts` — Konversi BigInt di `performanceMetrics` dan `onuStatuses` sebelum JSON response
- `src/app/api/olt/metrics/route.ts` — Konversi BigInt di metrics response
- `src/app/admin/olt/[id]/page.tsx` — Tampilkan username/password di Telnet section saat SSH disabled

---

## [2.29.17] — 2026-05-08

### Fixed
- **ONU List selalu kosong (0/0)** — Root cause 3 bug sekaligus:
  1. **SNMP parser gagal parse OID** — NET-SNMP mengembalikan format `iso.3.6.1...` (dengan prefix `iso.`) tapi regex hanya cocok `^[\d.]+`. Diperbaiki: tambah flag `-On` ke `snmpget`/`snmpwalk` agar output selalu numeric (`1.3.6.1...`), dan update regex untuk handle leading dot.
  2. **OID status salah (V2.1)** — OID `.3.31.4.1.100` mengembalikan INTEGER: 1 untuk SEMUA 8 slot per PON (bukan status ONU individual). Diperbaiki ke tabel `3.50.11.2` yang terbukti via live SNMP test.
  3. **Hex-STRING serial gagal parse** — Type prefix `Hex-STRING:` (dengan tanda hubung) tidak cocok dengan regex `\w+:`. Diperbaiki regex ke `[\w-]+:`.
- **Status ONU terbalik** — OID `3.50.11.2.1.6`: nilai `2=online` (bukan `1=online`). Terbukti dari ONU dengan uptime 83 hari yang return status=2.
- **Port numbering salah** — SNMP V2.1 `pon=1` harus disimpan sebagai `port=0` (ZTE CLI pakai 0-based port). Diperbaiki dengan offset `pon - 1`.
- **Serial number salah format** — ZTE GPON serial 8 bytes: 4 bytes ASCII vendor prefix + 4 bytes hex suffix. Misal `5A 54 45 47 DA 59 18 AC` → `ZTEGDA5918AC`. Konversi sebelumnya tidak benar.
- **Temperature menampilkan nilai tidak valid** — OID `3.50.12.1.1.4` mengembalikan `1` (bukan suhu). Ditambah validasi range 10–85°C agar nilai tidak masuk akal ditolak.
- **Poll Now tidak ada feedback** — Tambah loading state (`polling` state + spinner) dan alert jika gagal.

### Changed
- **OID profile ZTE C320 V2.1** diupdate ke tabel `3.50.11.2` yang sudah diverifikasi live:
  - `onuName`: `.3.50.11.2.1.1` (vendor prefix string, dipakai untuk walk discovery)
  - `onuSerial`: `.3.50.11.2.1.3` (Hex-STRING 8 bytes)
  - `onuStatus`: `.3.50.11.2.1.6` (INTEGER: 2=online, 1=init, 3=fault)
  - `onuModel`: `.3.50.11.2.1.9` (STRING model name)

### Files
- `src/lib/olt/snmp.ts` — Tambah `-On` flag, fix regex parser untuk `iso.` prefix dan `Hex-STRING:` type
- `src/lib/olt/vendors/zte.ts` — Update V21 OID profile, fix serial conversion, fix status mapping, fix port offset, fix temperature validation
- `src/app/admin/olt/[id]/page.tsx` — Tambah `polling` state + spinner + error feedback pada Poll Now

---


### Fixed
- **WireGuard peer hilang setelah reboot/re-install** — Root cause: `install-wg-server.sh` membuat ulang `wg0.conf` tanpa peer saat dijalankan ulang. Dibuat `wg-peer-watchdog.sh` yang berjalan tiap 5 menit via cron — otomatis restore peer dari database jika hilang dari `wg0.conf`.
- **LocalNetworks tidak tersimpan di DB** — API `POST /api/network/vps-wg-peer` kini menyimpan `localNetworks` ke kolom `description` (`localNets=x.x.x.x/yy,...`) agar watchdog bisa restore AllowedIPs dengan benar.

### Added
- **`/usr/local/bin/wg-peer-watchdog.sh`** — Script watchdog WireGuard di VPS: cek semua peer aktif dari DB, restore ke `wg0.conf` + `wg syncconf` jika hilang. Crontab: `*/5 * * * *`.
- **`scripts/wg-peer-watchdog.sh`** — Source script tersimpan di repo untuk referensi.

### Files
- `src/app/api/network/vps-wg-peer/route.ts` — Simpan localNetworks ke `description` saat create/update vpnClient
- `scripts/wg-peer-watchdog.sh` — Script watchdog WireGuard peer

---

## [2.29.15] — 2026-05-07

### Fixed
- **Router selalu "No router"** — Field name mismatch: API mengembalikan `routers[].router` tapi UI membaca `network_olt_routers[].nas`. Diperbaiki agar konsisten menggunakan `routers[].router`.
- **Router tidak ter-load saat Edit OLT** — `handleEdit` menggunakan `olt.network_olt_routers?.map(r => r.nas?.id)` yang selalu undefined. Diperbaiki ke `olt.routers?.map(r => r.router?.id)`.
- **Kolom "Model Profile" selalu "No profile"** — Model profile API adalah stub (selalu return `[]`). Diganti tampilkan `vendor + model` langsung dari data OLT.

### Files
- `src/app/admin/network/olts/page.tsx` — Fix OLT interface, handleEdit, display table, mobile card

---

## [2.29.14] — 2026-05-07

### Added
- **Firmware Version di modal Add/Edit OLT** — Field Firmware Version (e.g. `V2.1.0`, `V2.2.0`) ditambahkan ke form Add OLT dan Edit OLT di halaman `/admin/network/olts`. Kritis untuk ZTE C320 agar OID yang digunakan sesuai versi firmware.

### Files
- `src/app/admin/network/olts/page.tsx` — Tambah `firmwareVersion` ke formData, OLT interface, handleEdit, dan form UI
- `src/app/api/network/olts/route.ts` — Tambah `firmwareVersion` ke POST (create) dan PUT (update) handler

---

## [2.29.13] — 2026-05-07

### Fixed
- **ZTE C320 suhu/CPU/memori selalu N/A** — OID `1.3.6.1.4.1.3902.1015.1015.*` adalah untuk C300/C600, bukan C320. Diganti dengan walk-based approach yang mencoba C320 V2.1 OIDs (`1.3.6.1.4.1.3902.1012.3.50.12.*`), V2.2 OIDs (`1.3.6.1.4.1.3902.1082.500.20.2.1.2.*`), lalu fallback ke C300/C600.
- **ONU unregistered tidak terdeteksi (ZTE)** — Ditambahkan discovery `show gpon onu uncfg gpon-olt_1/{slot}/{port}` via Telnet/SSH. ONU belum terdaftar akan muncul dengan status `auth_failed` (unregistered).
- **ZTE port numbering** — Telnet/SSH discovery diperbaiki dari port 1–8 menjadi 0–7 sesuai notasi ZTE (`gpon-olt_1/1/0` bukan `gpon-olt_1/1/1`).
- **Router/NAS tidak bisa disimpan** — Halaman Settings OLT tidak memiliki field router sama sekali. Ditambahkan router selector (multi-checkbox) + simpan ke `networkOLTRouter` junction table via API PUT.
- **Firmware version tidak ada di Settings** — Ditambahkan field Firmware Version di Settings OLT. Ini kritis untuk memilih OID yang benar (V2.1 vs V2.2).

### Changed
- **Port diagram** — Redesign visual menjadi front-panel style (seperti NetMument): dark metallic chassis, SFP slot hole, fiber dot indicator, badge ONU count per port, CON/MGT port dummy, LED PWR/SYS/ALM glow effect.
- **API GET `/api/olt/[id]`** — Sekarang include `routers` (dengan data router name + IP) di response.
- **API PUT `/api/olt/[id]`** — Support `routerIds[]` untuk update router assignments.
- **package.json** — Version synced ke `2.29.12`

### Files
- `src/lib/olt/vendors/zte.ts` — Fix temperature/CPU/memory OIDs, add unregistered ONU discovery, fix port 0-based numbering
- `src/lib/olt/poller.ts` — Map `unregistered` status ke `auth_failed`
- `src/app/api/olt/[id]/route.ts` — Add routers include in GET, add routerIds handling in PUT
- `src/app/admin/olt/[id]/page.tsx` — Add firmware field, router selector, unregistered filter, redesign port diagram



### Fixed
- **ZTE C320 ONU list menampilkan 0 ONU** — Root cause: OID yang digunakan salah (dari C300/C600 MIB bukan C320). Rewrite `zte.ts` dengan OID yang benar dari referensi go-api-c320:
  - V2.1 firmware: base `1.3.6.1.4.1.3902.1012`, PON index = `board_base + pon × 256`
  - V2.2 firmware: base `1.3.6.1.4.1.3902.1082`, ID suffix & type suffix per board/PON
- **ONU discovery via SNMP untuk ZTE C320** — `poller.ts` sekarang mencoba SNMP dulu (`discoverONUsSNMP`) sebelum fallback ke SSH/Telnet. Ini lebih reliable untuk C320.
- **RxPower dari SNMP V2.2** — RX power diambil langsung saat SNMP discovery (V2.2 supports it × 0.01 dBm). Digunakan sebagai fallback jika opticalInfo Telnet/SSH tidak tersedia.
- **Port diagram slot count** — Telnet/SSH discovery di `zte.ts` sebelumnya iterasi slot 1–4; C320 hanya punya 2 GCOB cards, diperbaiki menjadi slot 1–2.
- **Port diagram index mismatch** — Port Map tab menampilkan semua port kosong karena lookup menggunakan 0-based index (`port 0`) sementara DB menyimpan port 1-based (`port 1–8`). Fix: `getPortStyle` dan `getPortTitle` sekarang menggunakan `portIndex + 1` untuk key lookup `portStats`.

### Changed
- **ZTE C320 Port Diagram — LED indicators** — Header chassis sekarang menampilkan LED PWR/SYS/ALM sesuai kondisi OLT dan alerts, memberi tampilan lebih mirip hardware asli.

### Files
- `src/lib/olt/vendors/zte.ts` — Complete rewrite dengan V2.1 + V2.2 OID profiles, `discoverONUsSNMP()`, fix slot count 1-2
- `src/lib/olt/poller.ts` — SNMP discovery diprioritaskan, `rxPower` dari SNMP digunakan sebagai fallback
- `src/app/admin/olt/[id]/page.tsx` — Fix port index bug (0-based→1-based), LED indicators di chassis header

---

## [2.29.11] — 2026-05-07

### Fixed
- **Telnet Test Connection selalu timeout (30s)** — Script expect sebelumnya menunggu prompt `#`/`>` lalu menjalankan `display version` yang bisa hang. Diganti dengan script yang lebih robust: TCP port check 3s dulu, kemudian expect yang menunggu pola prompt manapun (`#`, `>`, `$`, atau username/password prompt) dan langsung exit. Total timeout 15s, tidak perlu menjalankan perintah CLI apapun.

### Added
- **Dropdown Model OLT per Vendor** — Field "Model" di form tambah/edit OLT sekarang menjadi dropdown dinamis yang berubah sesuai vendor yang dipilih:
  - **ZTE**: C320, C300, C350, C600, C610, C650
  - **Huawei**: MA5608T, MA5680T, MA5683T, MA5800-X15, MA5800-X7, MA5800-X2
  - **FiberHome**: AN5516-01, AN5516-06, AN5516-04, AN5506-04-B, AN5516-06B
  - **Hioso / C-Data**: HA7304V, HA7304VX, HA7304C, HA8080G, HA8040G (dengan profil SNMP yang sesuai)
  - **BDCOM**: P3310C, P3310D, GP3600, GP3000, P3320C
  - **Raisecom**: ISCOM5508, ISCOM5504, ISCOM5516
  - Masing-masing model menampilkan tipe PON (GPON/EPON/XGS-PON)
- **Vendor Hioso ditambahkan** di dropdown vendor OLT (form OLT Management dan Settings detail OLT)
- **Port Diagram untuk Hioso, BDCOM, Raisecom** — Diagram fisik port OLT di tab "Port Map" sekarang mendukung semua vendor tersebut

### Files
- `src/lib/olt/telnet.ts` — `testTelnet()` diganti dengan robust expect script
- `src/app/admin/network/olts/page.tsx` — `VENDOR_MODELS` constant + dropdown model dinamis + tambah Hioso di vendor list
- `src/app/admin/olt/[id]/page.tsx` — tambah Hioso di vendor select settings, tambah template port diagram Hioso/BDCOM/Raisecom

---

## [2.29.10] — 2026-05-07

### Fixed
- **SSH gagal dengan "Unsupported algorithm: blowfish-cbc"** — OpenSSL 3.x (Ubuntu 22+) menghapus cipher `blowfish-cbc`. Dihapus dari semua algorithm lists di `ssh.ts` (`executeCommand`, `executeCommandsInShell`, `testSSH`).
- **OLT status tetap "Offline" setelah Test Connection berhasil** — `test-connection` API tidak mengupdate kolom `isOnline` di DB. Fix: setelah semua test selesai dan `oltId` diketahui, `prisma.networkOLT.update({ isOnline: anySuccess })` dijalankan.

### Added
- **OLT Port Diagram** — Tab baru "Port Map" di halaman detail OLT (`/admin/olt/[id]`). Menampilkan diagram visual front-panel OLT sesuai merk dan model:
  - ZTE C320: 1U chassis, 2x 10GE uplink, 2x GPON card masing-masing 8 port
  - ZTE C300: 7U chassis, 4x 10GE uplink, 4x GPON slot masing-masing 16 port
  - ZTE C350: 14U chassis, 8x 100GE uplink, 8x GPON slot masing-masing 16 port
  - Huawei MA5608T: 2U compact, 2x GE/10GE uplink, 8x GPON
  - Huawei MA5683T/MA5680T: 7U chassis, 4x 10GE uplink, 4x GPON slot
  - FiberHome AN5516-series: chassis, 4x uplink, 4x GPON slot
  - Generic: tampilan fallback 2 uplink + 8 PON
- Setiap port PON diwarnai sesuai status ONU: hijau (semua online), oranye (sebagian offline), merah (semua offline), abu-abu (kosong)
- Hover tooltip per port menampilkan ID port, jumlah ONU, avg RX power
- Tabel detail per port PON: progress bar online/total, avg RX power

### Files
- `src/lib/olt/ssh.ts` — hapus `blowfish-cbc` dari 3 algorithm list
- `src/app/api/olt/test-connection/route.ts` — update `isOnline` di DB saat test selesai
- `src/app/admin/olt/[id]/page.tsx` — tambah tab "Port Map" + komponen `OLTPortDiagram` + helper `getOLTTemplate`

---

## [2.29.9] — 2026-05-07

### Fixed
- **OLT Test Connection selalu gagal (SNMP/SSH/Telnet)** — Root cause dua masalah:
  1. `snmpget` dan `expect` tidak terinstall di VPS → diinstall manual via `apt-get install snmp expect`
  2. `ssh.ts` tidak ada legacy algorithms → ZTE/Huawei OLT lama hanya support `aes128-cbc`, `3des-cbc`, `diffie-hellman-group1-sha1` yang tidak ada di default ssh2
- `testSSH()` diubah menjadi test handshake only (bukan jalankan `display version`) agar lebih reliable lintas vendor
- `testTelnet()` sekarang cek TCP port open dulu sebelum jalankan full expect auth

### Files
- `src/lib/olt/ssh.ts` — tambah legacy cipher/kex/hmac algorithms, fix `testSSH()` to handshake-only
- `src/lib/olt/telnet.ts` — `testTelnet()` cek port open sebelum full auth

---

## [2.29.8] — 2026-05-07

### Fixed
- **POST /api/network/olts/status 404** — Endpoint untuk mengecek status konektivitas OLT belum ada. Halaman `/admin/network/olts` melakukan polling status setiap 30 detik ke endpoint ini. Fix: buat route baru yang membaca `isOnline`, `sshEnabled`, `telnetEnabled` dari DB dan kembalikan `statusMap`.
- **GET /admin/network/olt/[id] 404** — Link "View detail" di halaman daftar OLT mengarah ke `/admin/network/olt/[id]` tapi halaman detail sebenarnya ada di `/admin/olt/[id]`. Fix: perbaiki dua link di halaman daftar OLT (tabel desktop + kartu mobile).

### Files
- `src/app/api/network/olts/status/route.ts` — **BARU**: POST handler batch OLT status check
- `src/app/admin/network/olts/page.tsx` — Perbaiki 2 link ke `/admin/olt/${olt.id}`

---

## [2.29.7] — 2026-05-07

### Fixed
- **GET /api/network/olts 500 — BigInt tidak bisa di-serialize** — `networkOLT.uptime` bertipe `BigInt` di Prisma schema (MySQL BIGINT). `JSON.stringify` tidak bisa handle BigInt secara native. Fix: convert `uptime` ke `Number()` di semua OLT endpoint responses.

### Files
- `src/app/api/network/olts/route.ts` — GET/POST/PUT: `uptime: Number(olt.uptime)`
- `src/app/api/olt/[id]/route.ts` — GET/PUT: convert uptime + ONU statuses uptime
- `src/app/api/olt/monitoring/route.ts` — GET: convert uptime in map

---

## [2.29.6] — 2026-05-07

### Fixed
- **Update terhenti setelah GenieACS restore** — `apply_sql_migrations()` memanggil `mysql --force` yang tetap exit code non-zero meski ada SQL error (1060 duplicate column, 1061 duplicate index). Karena `updater.sh` pakai `set -e` + `set -o pipefail`, script langsung abort tepat setelah GenieACS restore, sebelum sempat build. Fix: tambah `|| true` setelah pemanggilan `mysql --force` (error dari mysql diabaikan — kita selalu mark file sebagai applied dan cek error real secara manual), dan `|| true` di kedua call site `apply_sql_migrations` sebagai defense-in-depth.

### Files
- `vps-install/updater.sh` — `mysql --force ... || true` + `apply_sql_migrations || true` di kedua call site (incremental + fresh install path)

---

## [2.29.5] — 2026-05-07

### Fixed
- **Telegram backup dikirim 2x ke bot** — `autoBackupToTelegram()` bisa dipanggil dari dua proses/trigger berbeda (misal `runner.ts` + proses lain) pada waktu yang sama. Ditambahkan deduplication guard: sebelum mulai, cek apakah ada `cronHistory` dengan `jobType=telegram_backup` dan status `running` atau `success` dalam 5 menit terakhir. Jika ada, langsung skip (return success tanpa kirim backup). Ini menghilangkan double-send tanpa perlu koordinasi antar proses.

### Files
- `src/server/jobs/telegram-cron.ts` — `autoBackupToTelegram()`: tambah deduplication guard (cek recent run dalam 5 menit terakhir sebelum proceed)

---

## [2.29.4] — 2026-05-08

### Fixed
- **SQL migration: `ADD COLUMN IF NOT EXISTS` tidak didukung MySQL (hanya MariaDB)** — Semua file migration yang pakai syntax MariaDB-only ini sekarang dipecah menjadi satu `ALTER TABLE ADD COLUMN` per statement dan syntax `IF NOT EXISTS` dihapus. `CREATE TABLE IF NOT EXISTS` tetap digunakan (MySQL sudah support). File yang diperbaiki: `20251223_add_billing_fields.sql`, `20260228_add_registration_fields.sql`, `20260320_add_pppoe_profile_hpp_ppn.sql`, `20260421_add_vpn_pool_config.sql`, `20260506_add_olt_monitoring_tables.sql`, `add_wireguard_fields.sql`.
- **`apply_sql_migrations()` di `updater.sh`: migration gagal tidak dicatat, muncul ulang tiap update** — Sebelumnya file hanya dicatat ke APPLIED_LOG jika exit code 0; jika ada error (termasuk yang benign seperti ERROR 1060 duplicate column), file akan dijalankan ulang di setiap update. Sekarang: (1) `mysql --force` digunakan agar error di satu statement tidak menghentikan statement lainnya; (2) semua file selalu dicatat sebagai applied setelah dijalankan (prisma db push adalah source of truth untuk schema); (3) hanya error real (bukan 1060 duplicate column / 1061 duplicate index) yang ditampilkan ke user.
- **`CREATE INDEX IF NOT EXISTS` di migration billing** — Diganti dengan `CREATE INDEX` biasa (MySQL tidak support IF NOT EXISTS untuk INDEX; --force handle duplikasi).

### Files
- `vps-install/updater.sh` — `apply_sql_migrations()`: `mysql --force`, selalu mark applied, filter error 1060/1061
- `prisma/migrations/20251223_add_billing_fields.sql` — split ADD COLUMN, hapus IF NOT EXISTS
- `prisma/migrations/20260228_add_registration_fields.sql` — split ADD COLUMN, hapus IF NOT EXISTS
- `prisma/migrations/20260320_add_pppoe_profile_hpp_ppn.sql` — split ADD COLUMN, hapus IF NOT EXISTS
- `prisma/migrations/20260421_add_vpn_pool_config.sql` — hapus IF NOT EXISTS dari ADD COLUMN
- `prisma/migrations/20260506_add_olt_monitoring_tables.sql` — split 21 ADD COLUMN, hapus IF NOT EXISTS
- `prisma/migrations/add_wireguard_fields.sql` — hapus IF NOT EXISTS dari ADD COLUMN

---

## [2.29.3] — 2026-05-06

### Fixed
- **Invoice number: format seragam `INV-YYYYMMDD-XXXXXX` di semua tempat** — Sebelumnya ada 3 format berbeda yang dipakai secara tidak konsisten: `INV-YYYYMM-0001` (sequential counter — di cron generate & extend API), `INV-YYYYMM-A3F9B2C1` (8 char UUID prefix — di registrasi user baru & manual generate UI), dan `INV-YYYYMM-0001` (sequential counter dengan DB count — di import CSV). Sekarang semua tempat menggunakan format tunggal: **`INV-YYYYMMDD-XXXXXX`** (tanggal 8 digit + 6 karakter random uppercase hex). Tidak ada lagi DB query untuk hitung urutan; tidak ada lagi race condition pada concurrent invoice generation.

### Changed
- `generateInvoiceNumber()` di `invoice.service.ts` sekarang fungsi **sync** (tidak async), tidak lagi butuh Prisma DB count.

### Files
- `src/server/services/billing/invoice.service.ts` — `generateInvoiceNumber()`: format baru, sync, tidak perlu DB
- `src/server/services/pppoe.service.ts` — pakai `generateInvoiceNumber()` dari billing service
- `src/server/jobs/voucher-sync.ts` — pakai `generateInvoiceNumber()`; hapus `invoiceCount` DB query
- `src/app/api/invoices/generate/route.ts` — pakai `generateInvoiceNumber()`; hapus inline format
- `src/app/api/admin/invoices/import/route.ts` — hapus local `generateInvoiceNumber()`, pakai dari billing service
- `src/app/api/pppoe/users/[id]/extend/route.ts` — hapus `await` (fungsi sudah sync)

---

## [2.29.2] — 2026-05-06

### Fixed
- **Invoice PREPAID: window mulai dari hari ini (H+0), bukan H+invoiceGenerateDays** — Sebelumnya `prepaidStartDate = today + 7`, sehingga user yang jatuh tempo besok (misal May 7 saat today=May 6) tidak termasuk dalam query dan invoice tidak di-generate — bahkan saat manual trigger (force=true). Sekarang window dimulai dari H+0 sehingga semua user yang expire hari ini hingga 30 hari ke depan tercakup. Duplikasi dicegah oleh check `existingInvoice` yang sudah ada.
- **Invoice PREPAID force mode: window diperlebar 90 hari ke belakang** — Saat admin trigger manual (force=true), query PREPAID sekarang mencakup `H-90` hingga `H+30` sehingga semua user yang missed bisa di-catch-up sekaligus.
- **Invoice first-period check: gunakan validitas paket, bukan hardcode 31 hari** — Check `firstPeriodEnd = createdAt + 31 hari` memblokir semua user paket 30-hari (karena `expiredAt ≈ createdAt + 30` selalu ≤ `createdAt + 31`). Sekarang `firstPeriodEnd = createdAt + validityDays + invoiceGenerateDays`, di mana `validityDays` diambil dari profil user. Invoice baru di-skip jika user belum pernah renew (masih periode pertama); setelah renew, `expiredAt > firstPeriodEnd` dan invoice di-generate normal.
- **Invoice catch-up: juga include user ACTIVE yang expiredAt sudah lewat** — Sebelumnya catch-up hanya untuk status `isolated/blocked/suspended`. User ACTIVE yang statusnya belum terupdate tapi `expiredAt` sudah lewat tidak tercakup. Sekarang menggunakan `eligibleStatuses` (termasuk `active`) untuk catch-up query.

### Files
- `src/server/jobs/voucher-sync.ts` — `generateInvoices()`: fix PREPAID window start, force mode wide window, first-period check berbasis validitas paket, catch-up include active users

---

## [2.29.1] — 2026-05-06

### Fixed
- **GenieACS WiFi: task pending/fault tidak lagi menumpuk** — Sebelumnya `POST /api/genieacs/devices/[id]/wifi` mengirim 3 task terpisah (SSID, security mode, password). Hanya task pertama yang manfaatkan `connection_request`; task berikutnya masuk antrean dan bisa fault jika device offline di antara task. Sekarang semua parameter (SSID, mode, password) digabung dalam **1 task `setParameterValues`** → 1 connection request → device menerapkan semua sekaligus.
- **GenieACS WAN: vendor VLAN params tidak lagi memblokir koneksi** — Parameter vendor-specific (`X_HW_VLAN`, `X_ZTE-COM_VLANIDMark`, `X_CMCC_VLANIDMark`) berada dalam task yang sama dengan PPPoE username/password. Jika device tidak support salah satu path, seluruh task fault termasuk koneksi. Sekarang dipisah jadi task tersendiri (best-effort) — koneksi PPPoE tetap diterapkan meski VLAN vendor gagal.
- **GenieACS: stale task accumulation** — Setiap kali user ubah setting, task baru ditumpuk di atas task pending lama. Ditambah helper `clearPendingTasks()` yang membersihkan semua pending/fault task milik device sebelum task baru dikirim.
- **GenieACS: 202 response ditangani benar** — Status 200 = task langsung dieksekusi di device; 202 = task diantrekan (device akan terapkan pada sesi TR-069 berikutnya). Keduanya dianggap sukses dengan pesan berbeda. Tidak ada lagi error palsu saat device lambat merespons.
- **GenieACS WiFi: hapus `refreshObject` task yang redundan** — Setelah update, sebelumnya ada task `refreshObject` tambahan yang kirim connection request lagi tanpa manfaat nyata.

### Files
- `src/app/api/genieacs/devices/[deviceId]/wifi/route.ts` — POST: gabung 3 task → 1 task; tambah `clearPendingTasks()`; hapus `refreshObject`; handle 202
- `src/app/api/genieacs/devices/[deviceId]/wan/route.ts` — POST/PUT/DELETE: tambah `clearPendingTasks()`; pisah vendor VLAN ke task best-effort; handle 202; tambah field `executed` di response

---

## [2.29.0] — 2026-05-10

### Fixed
- **TSC errors in `poller.ts`** — 3 TypeScript errors (TS2352/TS2322) saat Prisma `JsonValue` di-cast ke custom types. Fixed dengan double-cast `as unknown as Type` dan `as unknown as Prisma.InputJsonValue`.
- **OLT Add/Edit form missing SSH/Telnet port fields** — Form OLT hanya punya checkbox `sshEnabled`/`telnetEnabled` tanpa input port. Ditambah input field "SSH Port" (22) dan "Telnet Port" (23) yang muncul kondisional saat enabled. Juga tambah SNMP Port (161).
- **OLT API tidak menyimpan field credentials** — `POST` dan `PUT` `/api/network/olts` hanya menyimpan `name, ipAddress, latitude, longitude, status, followRoad`. Sekarang juga menyimpan: `vendor, model, username, password, snmpCommunity, sshEnabled, telnetEnabled, sshPort, telnetPort, snmpPort`.
- **OLT Test Connection gagal untuk OLT baru** — Backend hanya menerima `oltId` (DB lookup). Sekarang juga menerima direct params (`ipAddress, username, password, snmpCommunity, sshPort, telnetPort, snmpPort`) sebagai fallback saat `oltId` tidak ada. Semua protocol (SNMP/SSH/Telnet) bisa ditest sekaligus tanpa harus simpan OLT dulu.
- **Telegram Health double-send race condition** — `startHealthCron()` dan `startBackupCron()` bisa dipanggil dua kali bersamaan (concurrent requests) karena `healthCronJob === null` diperiksa sebelum async DB await selesai. Fixed dengan mutex flag `healthCronStarting` / `backupCronStarting`.
- **GenieACS task timeout terlalu singkat** — WiFi route menggunakan `timeout=3000ms`, WAN route `timeout=5000ms`. Kedua dinaikan ke `timeout=30000ms` agar device yang lambat merespons tidak langsung fault.

### Files
- `src/lib/olt/poller.ts` — Fix TSC2352: `as unknown as RuleCondition[]`, `as unknown as RuleAction[]`, `as unknown as Prisma.InputJsonValue`
- `src/app/api/network/olts/route.ts` — POST/PUT: simpan semua field OLT termasuk credentials dan port
- `src/app/admin/network/olts/page.tsx` — Tambah `sshPort`, `telnetPort`, `snmpPort` ke state; tambah input fields SSH/Telnet port kondisional; pass port ke test-connection
- `src/app/api/olt/test-connection/route.ts` — Rewrite: terima direct params ATAU oltId; test semua protocol jika tidak ada protocol tertentu
- `src/server/jobs/telegram-cron.ts` — Tambah mutex flag `healthCronStarting`/`backupCronStarting` untuk cegah race condition double-send
- `src/app/api/genieacs/devices/[deviceId]/wifi/route.ts` — Timeout: 3000ms/5000ms → 30000ms
- `src/app/api/genieacs/devices/[deviceId]/wan/route.ts` — Timeout: 5000ms → 30000ms (semua 3 handler: POST/PUT/DELETE)

---

## [2.28.0] — 2026-05-06

### Added
- **OLT Detail: tab Metrics dengan recharts** — Halaman `/admin/olt/[id]` kini memiliki tab "Metrics" berisi 4 chart interaktif: CPU & Memory (LineChart), Temperature (AreaChart), ONU Status online/offline (AreaChart), Network Traffic TX/RX (AreaChart). Range waktu bisa dipilih: 6h, 12h, 24h, 48h.
- **OLT Detail: batch reboot ONU** — Di tab ONU List, setiap baris kini memiliki checkbox. Pilih beberapa ONU lalu klik "Reboot N ONUs" untuk reboot massal (maks 50 sekaligus). Progress bar real-time menampilkan status per-ONU.
- **OLT Detail: single ONU reboot** — Tombol "Reboot" per baris ONU dengan confirm step sebelum eksekusi.
- **OLT Detail: CSV export** — Tombol "Export CSV" di header halaman untuk mengunduh daftar ONU lengkap dengan data customer.
- **OLT Detail: kolom Signal Quality** — Kolom baru "Signal" di tabel ONU menampilkan kualitas sinyal (Excellent/Good/Fair/Poor) berdasarkan nilai RX Power.
- **API: POST `/api/olt/[id]/onus/[onuId]/reboot`** — Endpoint baru untuk reboot satu ONU via SSH ke OLT. Mendukung command vendor spesifik: Huawei, ZTE, FiberHome, BDCOM, Raisecom.
- **API: POST `/api/olt/[id]/onus/batch-reboot`** — Endpoint baru untuk batch reboot ONUs, mengembalikan hasil per-ONU.

### Fixed
- **OLT Test Connection 404** — Halaman `/admin/network/olts` memanggil `/api/admin/olt/test-connection` yang tidak ada. URL dikoreksi ke `/api/olt/test-connection`.

### Files
- `src/app/admin/olt/[id]/page.tsx` — Tambah tab Metrics (recharts), batch/single ONU reboot, CSV export, Signal Quality column, layout kompak
- `src/app/api/olt/[id]/onus/[onuId]/reboot/route.ts` — **BARU** — Single ONU reboot via SSH
- `src/app/api/olt/[id]/onus/batch-reboot/route.ts` — **BARU** — Batch ONU reboot via SSH
- `src/app/admin/network/olts/page.tsx` — Fix URL test-connection `/api/admin/olt/...` → `/api/olt/...`

---

## [2.27.0] — 2026-05-06

### Added
- **OLT Monitoring: field "IP Lokal / Subnet di Balik NAS" saat tambah VPN WireGuard peer** — Form tambah VPN client (WireGuard VPS) kini memiliki input opsional untuk memasukkan IP/subnet lokal di balik NAS Mikrotik (contoh: `192.168.75.0/24,136.1.1.100/32`). Network lokal yang diisikan otomatis:
  - Ditambahkan ke `AllowedIPs` peer block di `wg.conf` VPS sehingga WireGuard tahu harus meneruskan traffic ke peer tersebut.
  - Ditambahkan route kernel di VPS (`ip route add`) sehingga VPS bisa menjangkau jaringan lokal dan IP OLT di balik Mikrotik tanpa konfigurasi manual.
- **OLT Monitoring UI: tampilan halaman `/admin/olt/monitoring` diperbarui** — Seluruh halaman diubah mengikuti gaya admin kompak (bukan `container mx-auto p-6`): heading kecil `text-lg font-semibold` + ikon teal, stat card native Tailwind tanpa shadcn, filter menggunakan `<select>/<input>` native, card OLT grid ringkas dengan dark mode support.
- **OLT Alerts UI: tampilan halaman `/admin/olt/alerts` diperbarui** — Konsisten dengan gaya admin: stat summary 4 kolom, filter native select, alert card compact dengan badge severity inline.

### Fixed
- **OLT Monitoring: link mati ke `/admin/olt/model-profiles-new/new`** — Tiga link yang mengarah ke halaman yang tidak ada dihapus dari halaman daftar OLT.
- **OLT Monitoring: dropdown vendor/model kosong** — Vendor diganti ke static dropdown (Huawei, ZTE, FiberHome, BDCOM, Raisecom, Other) dan model diubah ke input teks bebas, karena tabel `oltProfiles` belum ada di database.
- **Build error `ssh2` bundling** — Paket `ssh2`, `cpu-features`, dan `sshcrypto` ditambahkan ke `serverExternalPackages` di `next.config.ts` untuk mencegah Next.js mencoba bundling modul native crypto.

### Files
- `src/app/admin/network/vpn-client/page.tsx` — Tambah field `localNetworks` di form + UI input subnet lokal
- `src/app/api/network/vps-wg-peer/route.ts` — Terima `localNetworks`, tambahkan ke `AllowedIPs` wg.conf + `ip route` di VPS
- `src/app/admin/olt/monitoring/page.tsx` — Rewrite UI ke gaya admin kompak
- `src/app/admin/olt/alerts/page.tsx` — Rewrite UI ke gaya admin kompak
- `src/app/admin/network/olts/page.tsx` — Hapus link mati; vendor static dropdown; model free-text input
- `next.config.ts` — Tambah `ssh2`, `cpu-features`, `sshcrypto` ke `serverExternalPackages`

---

## [2.25.17] — 2026-05-03

### Fixed
- **Generate tagihan manual hanya untuk POSTPAID** — Endpoint `POST /api/invoices/generate` sebelumnya memfilter `subscriptionType: 'POSTPAID'` sehingga pelanggan PREPAID tidak pernah mendapat tagihan dari fitur generate manual. Filter dihapus sehingga semua pelanggan aktif (POSTPAID dan PREPAID) diproses.
- **Generate tagihan menggunakan tanggal jatuh tempo yang salah** — Due date sebelumnya selalu diset ke hari terakhir `targetMonth` untuk semua user. Diperbaiki dengan logika per-user:
  - **POSTPAID**: `dueDate = billingDay user di targetMonth` (diclamped ke hari terakhir bulan jika billingDay > jumlah hari di bulan tersebut)
  - **PREPAID**: `dueDate = user.expiredAt` (tanggal kedaluwarsa aktual yang sudah tersimpan di profil user)
- **PREPAID tanpa expiredAt dilewati** — PREPAID user yang belum memiliki `expiredAt` tidak akan di-generate invoice (di-skip) karena tidak ada tanggal jatuh tempo yang bisa dipakai.
- **Cek duplikat invoice sekarang mencakup tipe RENEWAL** — Batch check existing invoices sebelumnya hanya mengecek `invoiceType: 'MONTHLY'`. Sekarang mengecek keduanya (`MONTHLY` dan `RENEWAL`) agar PREPAID tidak ter-generate ulang.
- **Invoice PREPAID menggunakan `invoiceType: 'RENEWAL'`** — Sebelumnya semua invoice di-create dengan `invoiceType: 'MONTHLY'`. Invoice untuk PREPAID sekarang menggunakan `RENEWAL` sesuai konvensi sistem.
- **UI: deskripsi dialog generate tagihan diperbarui** — Teks "Buat tagihan bulanan untuk pelanggan POSTPAID" diganti menjadi "Buat tagihan untuk pelanggan POSTPAID dan PREPAID". Info teks scope "all" juga diperbarui.

### Files
- `src/app/api/invoices/generate/route.ts` — Hapus filter POSTPAID-only; per-user dueDate (billingDay / expiredAt); invoiceType MONTHLY/RENEWAL; cek duplikat RENEWAL
- `src/app/admin/invoices/page.tsx` — Update deskripsi dialog generate tagihan

---

## [2.25.16] — 2026-05-02

### Added
- **Notifikasi WhatsApp ke admin saat pembayaran manual baru masuk** — Saat pelanggan submit bukti pembayaran manual, sistem sekarang mengirim notifikasi WA instan ke semua admin. Pesan berisi nama pelanggan, username, nomor invoice, jumlah bayar, info bank pengirim, dan link langsung ke halaman approval. Sebelumnya hanya membuat record notifikasi di database.
- **Notifikasi WhatsApp ke semua SUPER_ADMIN** — Notifikasi WA untuk pendaftaran baru dan pembayaran manual kini dikirim ke semua admin (bukan hanya `company.adminPhone`). Nomor dikumpulkan dari dua sumber: `companies.adminPhone` + semua `admin_users` dengan role SUPER_ADMIN yang aktif dan punya nomor HP. Nomor duplikat di-deduplikasi otomatis. Pengiriman paralel (fire-and-forget) sehingga tidak memperlambat response API.
- **Helper `getAdminPhones()`** — Fungsi di `whatsapp-templates.service.ts` yang mengumpulkan dan mendeduplikasi semua nomor HP admin dari database. Memfilter nomor invalid (< 10 digit).
- **Helper `notifyAdminsViaWhatsApp(message)`** — Fungsi reusable untuk mengirim pesan WA ke semua admin. Dapat digunakan di endpoint lain yang butuh notifikasi admin.

### Files
- `src/server/services/notifications/whatsapp-templates.service.ts` — Tambah `getAdminPhones()` + `notifyAdminsViaWhatsApp()`
- `src/app/api/manual-payments/route.ts` — Tambah notifikasi WA ke semua admin saat POST (pembayaran manual baru)
- `src/app/api/registrations/route.ts` — Ubah notifikasi dari `adminPhone` saja ke semua admin via `notifyAdminsViaWhatsApp()`

---

## [2.25.15] — 2026-05-01

### Fixed
- **Import pelanggan PPPoE: username muncul sebagai `[object Object]`** — ExcelJS mem-parse cell yang berisi `@` (seperti `user@domain.id`) sebagai `CellHyperlinkValue` (`{ text, hyperlink }`). `String(cell.value)` menghasilkan `"[object Object]"` sehingga username salah terbaca. Diperbaiki dengan menangani semua tipe ExcelJS complex cell: hyperlink (ekstrak `.text`), richText (gabungkan `.richText[].text`), formula (ambil `.result`).
- **Import pelanggan PPPoE: semua baris gagal "Username already exists"** — Import sebelumnya hanya mendukung CREATE baru. File hasil Export berisi user yang sudah ada, sehingga semua baris gagal. Diperbaiki dengan logika **upsert**: jika username sudah ada di DB maka data diperbarui (password, nama, profile, IP, dll) + sync ulang ke RADIUS. Hasil import sekarang menampilkan `X Dibuat · Y Diperbarui`.
- **Template isolasi gagal disimpan ("data gagal disimpan")** — Endpoint `PUT /api/settings/isolation/templates/[id]` menggunakan pola params lama (`params: { id: string }`) tanpa `await`. Di Next.js 15+ `params` adalah Promise, sehingga `params.id` menjadi `undefined` dan Prisma gagal update. Diperbaiki dengan mengubah semua handler (GET/PUT/DELETE) ke `params: Promise<{ id: string }>` + `const { id } = await params`.

### Files
- `src/app/api/pppoe/users/bulk/route.ts` — Fix ExcelJS cell parsing + upsert logic untuk existing users
- `src/app/admin/pppoe/users/page.tsx` — Tampilkan counter "Diperbarui" di hasil import
- `src/app/api/settings/isolation/templates/[id]/route.ts` — Fix async params Next.js 15

---

## [2.25.14] — 2026-05-01

### Fixed
- **FreeRADIUS log error "Server returned no data"** — `rlm_rest` mencatat error ini setiap kali API radius mengembalikan `{}` (JSON kosong tanpa attribute RADIUS). Diperbaiki dengan mengubah semua response pass-through menjadi HTTP 204 No Content. `rlm_rest` mengenali 204 sebagai "tidak ada atribut yang di-set" dan tidak mencatat error.
- **FreeRADIUS error "Connection failed: 7 / Opening connection failed"** — REST module tidak punya timeout, sehingga saat app di-restart (npm build + pm2 restart) FreeRADIUS menunggu indefinitely dan menumpuk duplicate packets. Diperbaiki dengan menambahkan `connect_timeout = 4` detik dan `timeout = 4-5` detik per-seksi di konfigurasi REST module.
- **FreeRADIUS "Ignoring duplicate packet ... unfinished request in component authorize module rest"** — Akibat tidak adanya timeout di REST module. Setelah timeout ditambahkan, FreeRADIUS cepat fail-over ke SQL module (karena `-rest` non-fatal) tanpa menunggu.
- **Post-auth: voucher expired mengembalikan HTTP 403 dengan JSON non-RADIUS** — Response `{success: false, error: "Voucher expired"}` tidak dipahami rlm_rest. Diperbaiki menjadi RADIUS attribute format: `{"control:Auth-Type": "Reject", "reply:Reply-Message": "Voucher Kadaluarsa"}`.
- **FreeRADIUS REST `retry_delay` dikurangi** — Dari 30 detik menjadi 10 detik agar koneksi ke app pulih lebih cepat setelah restart.

### Added
- **Export PPPoE: filter status pembayaran** — Dropdown filter "Bayar" di halaman Pelanggan PPPoE dengan opsi: Semua, Sudah Bayar, Belum Bayar, Isolir. Filter berlaku untuk export Excel, PDF, dan CSV.
- **Export PPPoE: kolom Password di Excel dan PDF** — Password PPPoE sekarang disertakan di ekspor Excel dan PDF untuk keperluan backup/recovery (sebelumnya hanya tersedia di ekspor CSV).
- **Export PPPoE: filter paymentStatus di API** — Endpoint `/api/pppoe/users/export` dan `/api/pppoe/users/bulk?type=export` mendukung query param `paymentStatus=paid|unpaid|isolated` menggunakan join tabel Invoice.

### Files
- `freeradius-config/mods-available/rest` — Tambah `connect_timeout`, `timeout` per-seksi, kurangi `retry_delay`
- `src/app/api/radius/authorize/route.ts` — Pass-through responses → HTTP 204
- `src/app/api/radius/post-auth/route.ts` — Pass-through responses → HTTP 204, fix expired reject format
- `src/app/api/radius/accounting/route.ts` — Response → HTTP 204
- `src/app/api/pppoe/users/export/route.ts` — Tambah paymentStatus filter + kolom password
- `src/app/api/pppoe/users/bulk/route.ts` — Tambah paymentStatus filter pada type=export
- `src/app/admin/pppoe/users/page.tsx` — Filter UI "Bayar" + pass paymentStatus ke semua export handler

---

## [2.25.13] — 2026-05-01

### Fixed
- **Password PPPoE tidak berubah saat approval pembayaran manual** — Ditambahkan diagnostic logging di approval handler untuk membuktikan bahwa `pppoe_users.password` tidak berubah saat pembayaran disetujui. Perubahan yang terlihat di `radcheck.value` adalah perilaku yang disengaja (sinkronisasi RADIUS). Ditambahkan `autoComplete="new-password"` di modal edit user untuk mencegah browser autofill mengisi field password secara diam-diam.
- **Gambar bukti pembayaran manual tidak tampil** — URL gambar yang tersimpan di DB adalah path relatif (`/uploads/...`) sehingga komponen `Image` Next.js tidak bisa merendernya. Diperbaiki dengan membangun URL absolut menggunakan `NEXT_PUBLIC_BASE_URL` sebelum dikirim ke client.
- **Error approval pembayaran manual (500)** — Prisma update `manualPayment.status` gagal karena field `updatedAt` tidak ada di schema. Diperbaiki dengan menghapus field `updatedAt` dari data update.
- **Logo APK mobile tidak tampil** — Aset icon APK tidak ter-resolve dengan benar. Diperbaiki path resolusi icon.

### Changed
- **Diagnostic logging approval manual payment** — Log password sebelum dan sesudah transaksi approval agar dapat diverifikasi via `pm2 logs`.

### Files
- `src/app/api/manual-payments/[id]/route.ts` — Diagnostic logging + fix `updatedAt` field
- `src/components/UserDetailModal.tsx` — `autoComplete="new-password"` pada field password

---

## [2.25.12] — 2026-04-30

### Added
- **Backup & Restore GenieACS Config** — Tombol Backup dan Restore di halaman VP Scripts, Provisions, dan Presets. Format JSON, mendukung export per-tipe maupun backup semua sekaligus via `GET /api/genieacs/backup?type=all|vp|provisions|presets`. Restore via `POST /api/genieacs/backup`.

### Changed
- **Cache device list GenieACS 5 menit** — TTL cache device list ditingkatkan dari 60 detik ke 5 menit (stale-while-revalidate). Mengurangi load ke GenieACS NBI ~5x, response tetap instan.

### Files
- `src/app/admin/genieacs/vp-scripts/page.tsx` — Tombol Backup + Restore ditambahkan
- `src/app/admin/genieacs/provisions/page.tsx` — Tombol Backup + Restore ditambahkan
- `src/app/admin/genieacs/presets/page.tsx` — Tombol Backup + Restore ditambahkan
- `src/app/api/genieacs/backup/route.ts` — API endpoint baru (GET + POST)
- `src/app/api/settings/genieacs/devices/route.ts` — Cache TTL 60s → 300s

---

## [2.25.11] — 2026-05-02

### Added
- **Generate Tagihan Manual di Halaman Tagihan** — Tombol "Generate Tagihan" baru di header halaman `/admin/invoices`. Membuka dialog dengan opsi:
  - **Target**: Semua Pelanggan POSTPAID aktif, atau Satu Pelanggan (dengan pencarian nama/username/HP)
  - **Bulan Tagihan**: Picker bulan (`YYYY-MM`), default bulan berjalan
  - **Opsi**: Lewati jika tagihan bulan tersebut sudah ada (default aktif), Kirim notifikasi WhatsApp setelah generate
  - Setelah generate: tampilkan ringkasan (dibuat / dilewati / gagal) + detail error jika ada
- **API POST `/api/invoices/generate`** — Endpoint baru untuk generate tagihan manual. Mendukung `scope: 'all' | 'single'`, `targetMonth (YYYY-MM)`, `userId`, `skipExisting`, `sendWa`. Menghitung PPN otomatis sesuai profil. Due date = hari terakhir bulan target.

### Files
- `src/app/admin/invoices/page.tsx` — Dialog + tombol Generate Tagihan ditambahkan
- `src/app/api/invoices/generate/route.ts` — API endpoint baru

---

## [2.25.10] — 2026-05-01

### Changed
- **Redesign Form Tambah Pelanggan — 4 Tab Layout** — Form dibagi menjadi 4 tab: 📡 Akun RADIUS, 👤 Data Pelanggan, 🔧 Instalasi, ⚙️ Pengaturan. Navigasi via tombol Sebelumnya/Berikutnya + dot indicator. Tidak perlu scroll panjang. Tab menampilkan tanda hijau jika field wajib sudah terisi.
- **Support Pelanggan Tanpa Akun PPPoE** — Toggle "Punya Akun PPPoE / Tanpa Akun PPPoE" di tab Akun RADIUS. Jika dimatikan, username & password tidak wajib diisi — sistem auto-generate username `STATIC-{customerId}`. Cocok untuk pelanggan IP statis atau MAC-based. RADIUS sync dilewati kecuali `Framed-IP-Address` jika IP statis diisi.

### Files
- `src/app/admin/pppoe/users/new/page.tsx` — Rewritten with 4-tab layout
- `src/app/api/pppoe/users/route.ts` — Validation updated for optional PPPoE credentials
- `src/server/services/pppoe.service.ts` — Auto-generate username + skip RADIUS sync for static customers

---

## [2.25.9] — 2026-04-30

### Added
- **Subdomain Routing Frontend UI** — Admin → Settings → Subdomain Routing: halaman panduan interaktif untuk mengatur subdomain per portal (`customer.domain.com`, `agent.domain.com`, `teknisi.domain.com`, `admin.domain.com`). Input domain dinamis (auto-detect dari Base URL), tampilkan DNS records yang perlu ditambahkan, Nginx config siap pakai (bisa di-download .conf), panduan Certbot SSL, dan perintah test curl. Semua script bisa disalin dengan satu klik.
- **Subdomain Routing di Middleware (`proxy.ts`)** — Next.js middleware membaca header `Host`, parse subdomain, lalu `NextResponse.rewrite()` ke path portal yang sesuai tanpa redirect (URL tetap). Map: `customer`/`pelanggan` → `/customer`, `agent`/`agen` → `/agent`, `teknisi`/`technician` → `/technician`, `admin` → `/admin`.
- **Prorate Billing di Form Tambah Pelanggan PPPoE** — Untuk tipe POSTPAID: estimasi tagihan prorate dihitung otomatis (live) berdasarkan profil, tanggal jatuh tempo, dan tanggal daftar. Ditampilkan dalam kotak hijau "Estimasi Tagihan Pertama (Prorate)".
- **Info Alur Pembayaran di Form Tambah Pelanggan** — Kotak biru (POSTPAID) dan ungu (PREPAID) menjelaskan alur pembayaran 4-langkah, muncul otomatis sesuai pilihan tipe langganan.
- **Field Aksi Jatuh Tempo di Form Tambah Pelanggan** — Dropdown "⚡ Aksi Jatuh Tempo" di section Informasi Tambahan: pilih antara `ISOLIR INTERNET (Suspend)` atau `TETAP TERHUBUNG (No Action)`. Default: ISOLIR. Field ini sebelumnya tidak ada di form tambah pelanggan baru.
- **Entri nav sidebar: Subdomain Routing** — Menu Settings admin memiliki sub-menu baru "Subdomain Routing" di bawah Cloudflare Tunnel.

### Fixed
- **Isolasi PPPoE — user tetap online setelah expired** — Sebelumnya, user expired hanya diubah grup RADIUS ke `isolir` tapi session PPP lama tetap jalan. Fix 3-layer:
  1. **Langsung** (sebelum disconnect): API MikroTik tambahkan IP aktif ke address-list `isolir` → firewall `src-address-list=isolir action=drop` blokir internet saat itu juga.
  2. **CoA/disconnect**: disconnect PPP paksa re-auth.
  3. **Reconnect**: RADIUS kirim atribut `Mikrotik-Address-List=isolir` → MikroTik auto-add IP baru ke address-list.
- **Script MikroTik Setup Page — gunakan address-list bukan subnet** — Firewall filter dan NAT rules di halaman Setup MikroTik diubah dari `src-address=192.168.200.0/24` (subnet) ke `src-address-list=isolir` (address-list dinamis). Lebih presisi dan langsung efektif tanpa menunggu reconnect. PPP profile ditambah `use-mpls=no use-compression=no use-encryption=no`.
- **Export CSV PPPoE — kolom area, subscriptionType, billingDay hilang** — Export CSV kini menyertakan kolom `area`, `subscriptionType`, dan `billingDay`.
- **Form Tambah Pelanggan — field area, billingDay, registeredAt tidak ada** — Form tambah pelanggan baru kini menyertakan semua field yang diperlukan API.

### Files
- `src/proxy.ts` — subdomain routing middleware
- `src/app/admin/settings/subdomain/page.tsx` *(baru)* — UI panduan subdomain routing
- `src/app/admin/pppoe/users/new/page.tsx` — prorate billing, payment flow info, Aksi Jatuh Tempo field
- `src/app/api/pppoe/users/bulk/route.ts` — export CSV + kolom area/subscriptionType/billingDay
- `src/server/jobs/auto-isolation.ts` — isolasi langsung via address-list sebelum disconnect
- `src/server/services/radius/coa-handler.service.ts` — fungsi baru `addToMikrotikAddressList()`
- `src/app/api/settings/isolation/route.ts` — tambah `Mikrotik-Address-List` ke radgroupreply isolir
- `src/app/admin/settings/isolation/mikrotik/page.tsx` — script firewall/NAT pakai `src-address-list=isolir`
- `src/app/admin/AdminClientLayout.tsx` — nav entry Subdomain Routing
- `src/locales/id.json` — translation key `subdomainRouting`

---

## [2.25.8] — 2026-05-02

### Added
- **WAN Management di GenieACS Device Detail** — Halaman detail perangkat GenieACS kini mendukung manajemen koneksi WAN lengkap:
  - **Add WAN**: Tombol "Add WAN" di Quick Actions dan di header seksi WAN. Modal add menampilkan pemilihan Connection Type (PPPoE/IP), Nama koneksi, WANDevice index (port binding) 1–2, dan WANConnectionDevice index 1–8 untuk binding ke LAN port spesifik. Implementasi via GenieACS `addObject` diikuti `setParameterValues` pada instance baru.
  - **Edit WAN**: Edit username/password PPPoE, VLAN ID (0–4094), VLAN Priority (0–7), Service Type, dan toggle Enable/Disable per koneksi WAN. Implementasi via `setParameterValues` multi-parameter.
  - **Delete WAN**: Tombol hapus per kartu WAN, implementasi via GenieACS `deleteObject`.
  - **VLAN Configuration**: Set `X_HW_VLAN` (Huawei), `X_ZTE-COM_VLANIDMark`, `X_CMCC_VLANIDMark`, dan `X_HW_VLANPriority` dalam satu request.
  - **Service Type**: Pilihan INTERNET, TR069, VOIP, IPTV, INTERNET_TR069, OTHER — dikirim ke `X_HW_ServiceList` dan `X_ZTE-COM_ServiceList`.
  - **Port Binding**: WANDevice.{N} dan WANConnectionDevice.{N} bisa dipilih saat add WAN.
- **WAN Connection Display dengan Badge** — Kartu WAN menampilkan badge: service type (oranye), VLAN ID (cyan), connection type (abu), status connected/disconnected. Path TR-069 ditampilkan dalam teks monospace kecil.
- **In-Memory Cache untuk Device List GenieACS** — `GET /api/settings/genieacs/devices` kini menggunakan cache di level modul (PM2 process-persistent):
  - TTL 60 detik; response langsung dari cache saat masih fresh.
  - **Stale-while-revalidate**: Jika cache sudah kedaluwarsa, data lama langsung dikembalikan ke client (tanpa blocking) sambil refresh dilakukan di background secara async.
  - Cache key menggunakan hash `host:username` — otomatis invalid jika kredensial GenieACS berubah.
  - Response menyertakan field `fromCache: boolean` dan `cacheAge: number` (ms).
  - Strategi ini membuat halaman Perangkat GenieACS terasa instan setelah load pertama.

### API Files
- `src/app/api/settings/genieacs/devices/route.ts` — cache ditambahkan (module-level stale-while-revalidate)
- `src/app/api/genieacs/devices/[deviceId]/wan/route.ts` — API WAN baru (POST update, PUT add, DELETE)

### UI Files
- `src/app/admin/genieacs/devices/page.tsx` — WAN modal lengkap (add/edit/delete + VLAN/service/port binding)

---

## [2.25.7] — 2026-04-29

### Added
- **Halaman Cloudflare Tunnel Setup** — Admin → Settings → Cloudflare Tunnel: panduan langkah-langkah interaktif (6 step) install cloudflared di VPS, login, buat tunnel, simpan domain ke database, konfigurasi Nginx, dan verifikasi. Domain tunnel tersimpan ke `company.baseUrl` via API `POST /api/admin/cloudflare-tunnel`. Auto-compress backup >50MB sebelum kirim ke Telegram.
- **API `GET/POST /api/admin/cloudflare-tunnel`** — Endpoint baru untuk membaca status konfigurasi tunnel (`baseUrl`, `NEXTAUTH_URL`, `NEXT_PUBLIC_APP_URL`) dan menyimpan domain tunnel ke database.
- **Entry nav sidebar admin: Cloudflare Tunnel** — Menu Settings admin kini memiliki sub-menu "Cloudflare Tunnel" di antara Update Sistem dan Download APK.

### Fixed
- **Admin sidebar — semua halaman tema terang (light mode) perbaikan massal** — 62 halaman admin masih menggunakan warna neon cyberpunk (`#00f7ff`, `#bc13fe`, `#ff44cc`, dll.) tanpa prefix `dark:` sehingga teks/border/card tidak terbaca di light mode. Seluruh kelas diganti ke pola TailAdmin: title `text-foreground dark:text-transparent dark:bg-clip-text`, spinner `text-brand-500 dark:text-[#00f7ff]`, border `border-border dark:border-[#bc13fe]/30`, card `bg-card dark:bg-[#1a1525]/80`.

### Affected
- `src/app/admin/AdminClientLayout.tsx`
- `src/app/admin/settings/cloudflare-tunnel/page.tsx` *(baru)*
- `src/app/api/admin/cloudflare-tunnel/route.ts` *(baru)*
- `src/locales/id.json`
- 62 halaman admin (network, settings, tickets, notifications, dll.)

---

## [2.25.6] — 2026-04-28

### Fixed
- **Tema terang agent portal — seluruh teks/border neon tidak terbaca** — Halaman `vouchers`, `sessions`, dan `tickets` portal agen masih menggunakan warna hex neon (`#00f7ff`, `#bc13fe`, `#ff44cc`, `#00ff88`, dll.) yang di theme terang menjadi tidak terbaca karena di-override oleh `globals.css`. Seluruh warna tersebut diganti dengan pasangan class Tailwind standar yang aman untuk light dan dark mode.

### Added
- **Input lokasi GPS di form tiket agen** — Form "Buat Tiket" portal agen kini memiliki field tag lokasi (teks manual) dan tombol GPS yang mengambil koordinat dari browser (`navigator.geolocation`). Lokasi dan link Google Maps otomatis disisipkan ke deskripsi tiket agar teknisi lebih mudah menemukan lokasi pelanggan.

### Changed
- **Redesign UI agent/vouchers — pure Tailwind dark/light** — Loading spinner, container utama, filter controls, mobile cards, desktop table, pagination, dan dialog WhatsApp semuanya diperbarui ke class Tailwind standar (`bg-white dark:bg-slate-800/60`, `border-slate-200 dark:border-slate-700`, status badge `bg-emerald-100 text-emerald-700`, dll.).
- **Redesign UI agent/sessions — pure Tailwind dark/light** — Header, tombol refresh, stats cards (cyan/emerald/pink), search bar, daftar sesi (mobile card + desktop table) diperbarui; upload `text-emerald-600 dark:text-emerald-400`, download `text-pink-600 dark:text-pink-400`.
- **Redesign UI agent/tickets — pure Tailwind dark/light** — Header, tombol "Buat Tiket", form tiket, filter status, daftar tiket, chat bubble, dan reply box diperbarui dari neon gradient ke `from-violet-600 to-cyan-600`; active filter `bg-violet-100 dark:bg-violet-500/20`.

### Affected
- `src/app/agent/vouchers/page.tsx`
- `src/app/agent/sessions/page.tsx`
- `src/app/agent/tickets/page.tsx`

---

## [2.25.5] — 2026-04-28

### Added
- **APK Android: notifikasi native dengan suara, getaran & floating** — APK WebView kini menyertakan `NotificationChannel` (Android 8+), JavaScript bridge (`Android.showNotificationWithTag`) yang terhubung ke service worker push event (`PUSH_RECEIVED`), serta `NotificationWorker` berbasis WorkManager yang polling `/api/notifications` setiap 15 menit di background. Notifikasi tampil dengan prioritas HIGH, suara default, getaran, dan heads-up notification bahkan saat aplikasi ditutup.
- **Logo square 1:1 di semua halaman** — Semua container logo (login Admin/Customer/Technician/Agent, sidebar admin, settings company, download APK, halaman isolated) kini menggunakan rasio persegi (1:1) dengan `object-contain` sehingga logo 512×200 ditampilkan dalam kanvas 512×512 dengan letterbox — tidak distretch.

---

## [2.25.4] — 2026-04-28

### Added
- **Input lokasi GPS di form tiket pelanggan** — Halaman pembuatan tiket pelanggan kini mendukung tag lokasi, pengambilan koordinat GPS dari browser, dan penyisipan link Google Maps otomatis ke deskripsi tiket agar teknisi lebih mudah menemukan rumah pelanggan.
- **Tanggal Register editable untuk user PPPoE** — Form tambah dan edit user PPPoE kini menyediakan field `Tanggal Register` yang tersimpan ke `createdAt`, sehingga data historis pelanggan bisa dikoreksi tanpa manipulasi database manual.
- **Logo perusahaan di sidebar admin** — Sidebar admin kini menampilkan logo perusahaan secara langsung dengan fallback ke inisial jika logo belum tersedia.

### Fixed
- **CSV import/export PPPoE belum mendukung `registeredAt`** — Template CSV/XLSX, normalization map import, dan parsing data bulk kini mendukung `Tanggal Register` / `registeredAt` sehingga tanggal registrasi historis tidak lagi hilang saat impor massal.
- **Penyimpanan rate limit isolir ke tabel RADIUS tidak pernah update** — Endpoint pengaturan isolasi sebelumnya memakai `ON DUPLICATE KEY UPDATE` pada tabel `radgroupreply` yang tidak memiliki UNIQUE constraint. Diperbaiki ke pola `DELETE + INSERT` untuk atribut `Mikrotik-Rate-Limit`, `Mikrotik-Group`, dan `Framed-Pool`.
- **Label tanggal isolasi PPPoE masih memakai istilah kedaluwarsa** — Teks UI terkait `expiredAt` di form PPPoE dan detail user kini diseragamkan menjadi `Tanggal Isolir`.
- **Preview/logo branding belum konsisten di semua halaman** — Login Admin, Customer, Technician, Agent, halaman Isolated, Settings Company, dan Download APK kini memakai pola logo dinamis dengan `object-contain` dan batas layout ideal agar logo horizontal maupun vertikal tetap proporsional.

### Changed
- **Upload logo kini mendukung lebih banyak format** — Upload logo perusahaan sekarang menerima PNG, JPG, SVG, WebP, AVIF, dan GIF, dengan mapping ekstensi berbasis MIME type agar nama file hasil upload lebih konsisten.
- **Download APK memakai preview logo full-area** — Kartu logo pada halaman download APK kini memakai container preview lebih besar agar admin bisa melihat hasil branding secara proporsional sebelum build APK.
- **Versi aplikasi disinkronkan dengan changelog** — Metadata versi project dinaikkan ke `2.25.4` agar badge versi, package metadata, dan changelog tetap selaras.

### Affected
- `src/app/admin/AdminClientLayout.tsx`
- `src/app/admin/download-apk/page.tsx`
- `src/app/admin/login/page.tsx`
- `src/app/admin/pppoe/users/page.tsx`
- `src/app/admin/settings/company/page.tsx`
- `src/app/agent/page.tsx`
- `src/app/api/pppoe/users/bulk/route.ts`
- `src/app/api/settings/isolation/route.ts`
- `src/app/api/upload/logo/route.ts`
- `src/app/customer/login/page.tsx`
- `src/app/customer/tickets/create/page.tsx`
- `src/app/isolated/page.tsx`
- `src/app/technician/login/page.tsx`
- `src/components/UserDetailModal.tsx`
- `src/locales/id.json`
- `src/server/services/pppoe.service.ts`

## [2.25.3] — 2026-04-27

### Fixed
- **Nama perusahaan tidak terlihat di tema terang pada semua portal login role** — Beberapa halaman login menampilkan heading brand dengan gaya yang bisa kehilangan kontras di light mode (teks putih/gradient terhadap latar terang), sehingga nama perusahaan nyaris tidak terbaca. Diperbaiki dengan pola heading kontras yang konsisten (`text-slate-900` untuk light mode, `text-white` untuk dark mode) dan fallback nama perusahaan yang aman.

### Changed
- **Redesign UI login lintas role (Admin, Customer, Agent, Technician)** — Semua halaman login portal diseragamkan tata letaknya agar konsisten antar-role dan tetap responsif desktop/mobile.
  - Panel form login diseragamkan (`lg:w-[430px]`, background `bg-card`, batas `border-border`) untuk ritme visual yang sama.
  - Ditambahkan blok branding **"Nama Perusahaan"** di sisi form agar identitas tetap terbaca jelas pada tema terang maupun gelap.
  - Area hero kanan diperbarui dengan heading tunggal yang tegas + accent bar gradient per role untuk visual yang lebih clean dan kontras.
  - Gradien latar hero desktop dirapikan ke palet yang lebih lembut di light mode agar elemen teks tidak tenggelam.

### Affected
- `src/app/admin/login/page.tsx`
- `src/app/customer/login/page.tsx`
- `src/app/agent/page.tsx`
- `src/app/technician/login/page.tsx`

---

## [2.25.2] — 2026-04-26

### Added
- **WhatsApp Baileys — Native WhatsApp gateway built-in di VPS** — Provider baru `baileys` menggunakan library `@whiskeysockets/baileys` yang berjalan sebagai proses PM2 terpisah (`salfanet-wa`) di `127.0.0.1:4000`. Tidak perlu layanan pihak ketiga (Fonnte, WAHA, MPWA, dll).
  - `GET /api/whatsapp/providers/:id/qr` — Ambil QR code untuk scan WhatsApp Web
  - `GET /api/whatsapp/providers/:id/status` — Cek status koneksi (connected/disconnected)
  - `POST /api/whatsapp/providers/:id/restart` — Logout session & generate QR baru
  - `wa-service.js` — Express server standalone yang mengelola koneksi Baileys + generate QR (base64 PNG)
  - PM2 process `salfanet-wa` ditambahkan ke `production/ecosystem.config.js`
  - Auth session tersimpan di `/var/data/salfanet/baileys_auth` (persist across restart)
  - `vps-install/updater.sh` otomatis setup direktori auth + start `salfanet-wa`
- **QR Modal: success state + auto-refresh** — Setelah scan berhasil, modal WhatsApp QR menampilkan animasi centang hijau "WhatsApp Berhasil Terhubung!" beserta tombol tutup. Status provider card di-refresh otomatis tanpa reload halaman.

### Fixed
- **HTTP 400 saat QR belum siap (WAITING state)** — Saat Baileys masih inisialisasi (belum generate QR), `/qr` endpoint sebelumnya mengembalikan 400 → frontend tampil error dan tutup modal. Sekarang server balas 202 dengan `{ waiting: true }`, dan frontend otomatis retry setiap 2,5 detik dengan spinner loading tetap tampil.
- **Spinner menghilang saat WAITING** — Bug `finally { setQrLoading(false) }` selalu dieksekusi meskipun ada `return` di `try` block. Diperbaiki dengan flag `retrying` yang dideklarasi di luar `try` — `finally` hanya stop spinner jika `!retrying`.
- **Status tetap "terhubung" setelah device disconnect** — Saat perangkat melepas Linked Device dari HP, Baileys set status `logged_out` tapi tidak ada auto-reconnect. Klik tombol QR hanya mengembalikan WAITING tanpa pernah generate QR baru. Diperbaiki: endpoint `/qr` kini otomatis memanggil `connectToWhatsApp()` jika status `logged_out` atau `error`, sehingga QR baru muncul otomatis.
- **"Tidak dapat menautkan" saat scan QR** — WhatsApp menolak koneksi karena fingerprint browser `macOS Desktop` memicu deteksi bot. Diperbaiki dengan mengubah ke `Browsers.ubuntu('Chrome')` + `markOnlineOnConnect: false` + `connectTimeoutMs: 60000`.
- **`wa-service.js` crash: MODULE_NOT_FOUND `express`** — Modul `express` tidak ada di `node_modules` karena bukan dependency sebelumnya. Diperbaiki dengan menambahkan `"express": "^4.21.2"` ke `package.json` root.

### Changed
- **`whatsapp.service.ts`** — Menambahkan `'baileys'` ke union type provider dan method `sendViaBaileys()` yang memanggil `http://127.0.0.1:${WA_SERVICE_PORT}/send`
- **Dependencies tambahan di `package.json`** — `@whiskeysockets/baileys ^7.0.0-rc.9`, `pino ^10.3.1`, `express ^4.21.2`

---

## [2.25.1] — 2026-04-26

### Added
- **`vps-install/install-security.sh` — Modul keamanan server otomatis** — Script baru yang dipanggil di Step 8 installer dan setiap `updater.sh`. Memasang tiga lapisan perlindungan secara otomatis:
  - **fail2ban**: ban IP brute-force SSH setelah 5x gagal dalam 10 menit (ban 2 jam). Jail aktif: `sshd`, `nginx-http-auth`, `nginx-limit-req`. IP jaringan lokal (`192.168.x.x`, `10.x.x.x`) tidak pernah di-ban.
  - **UFW Firewall**: default deny semua incoming, allow hanya port yang dibutuhkan: 22/TCP (SSH), 80/TCP (HTTP), 443/TCP (HTTPS), 1812-1813/UDP (RADIUS), 3799/UDP (RADIUS CoA). Di-skip otomatis untuk LXC container (pakai Proxmox host firewall).
  - **Disk cleanup cronjob**: script `/usr/local/bin/salfanet-cleanup.sh` berjalan otomatis setiap hari jam 02:00. Membersihkan: journal systemd (max 200MB/7 hari), syslog lama, btmp (truncate jika >50MB), APT cache, tmp files, PM2 logs besar, Gradle cache >30 hari, APK build temp.
  - Bisa dijalankan manual: `bash vps-install/install-security.sh`
  - Log cleanup: `/var/log/salfanet-cleanup.log` (auto-trim jika >5MB)

### Fixed
- **Disk penuh 100% menyebabkan MySQL deadlock & API 500** — Disk VPS publik penuh akibat log systemd journal (~2.9GB) dan syslog (~2.2GB) menumpuk. MySQL tidak bisa commit karena disk penuh → semua query FreeRADIUS (`radpostauth`, `radacct`) stuck "waiting for handler commit" → Prisma connection pool exhausted (P2024) → semua API endpoint 500. Diatasi dengan cleanup log + install cronjob harian.
- **Build APK customer/technician/agent: connection pool exhausted saat 3 build serentak** — Menjalankan Gradle build untuk 3 role sekaligus menyebabkan VPS overload. Prisma connection pool (limit 10) habis karena server tidak bisa melayani request DB selama build berjalan. Build sebenarnya tetap berjalan di background; yang "berhenti" hanya tampilan UI karena polling API gagal 500.

### Changed
- **`vps-install/vps-installer.sh`: tambah Step 8 (Security)** — Installer utama kini memanggil `install-security.sh` secara otomatis setelah Step 7 (PM2 & Build). Instalasi baru langsung terlindungi fail2ban + UFW + cleanup cron tanpa langkah manual.
- **`vps-install/updater.sh`: security check saat setiap update** — Setiap kali `bash updater.sh` dijalankan, script memastikan: (1) cleanup cronjob terpasang, (2) fail2ban dalam keadaan running. Idempotent — aman dijalankan berulang kali.

### Fixed
- **Self-heal login pasca update GitHub (legacy install)** — `updater.sh` kini menjalankan `vps-install/fix-auth-after-update.sh` setelah `prisma db push` untuk mencegah kasus gagal login setelah update pada instalasi lama. Perbaikan otomatis meliputi:
  - Migrasi akun dari tabel legacy `admin_user` ke `admin_users` jika `admin_users` kosong
  - Menjamin minimal ada 1 akun `SUPER_ADMIN` aktif
  - Membuat fallback `superadmin` hanya jika database benar-benar kosong
- **Self-heal PM2 app mode** — `updater.sh` kini mendeteksi proses PM2 legacy yang masih jalan via `next start`/`npm start`, lalu migrasi otomatis ke `.next/standalone/server.js` dari `ecosystem.config.js`.

---

## [2.25.0] — 2026-04-26

### Added
- **Build APK Android langsung di server VPS** ([`91a45d5`]) — Fitur baru di halaman `/admin/download-apk`: build APK Android Kotlin (WebView wrapper) langsung di server menggunakan Gradle, tanpa perlu upload ke GitHub atau install Android Studio. APK tersimpan di server dan bisa didownload kapan saja.
  - `GET /api/admin/apk/trigger` — cek ketersediaan Java JDK dan Android SDK di server
  - `POST /api/admin/apk/trigger?role=admin|customer|technician|agent` — mulai build di background (detached process, tidak timeout)
  - `GET /api/admin/apk/status?role=...` — polling status build: `idle` / `building` / `done` / `failed` / `stale`
  - `GET /api/admin/apk/file?role=...` — download APK hasil build
  - UI polling otomatis setiap 3 detik selama build berjalan
  - Deteksi stale build: jika status masih `building` setelah 15 menit, otomatis ditandai `stale`
  - Panduan install Android SDK ditampilkan di UI jika environment belum siap (copy-able bash command)
  - Fallback ZIP download tetap tersedia via collapsible section
- **Setup Android SDK di VPS** (manual, satu kali) — Jalankan command berikut via SSH sebelum menggunakan fitur build:
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
  Build pertama ±3–5 menit (download Gradle dependencies). Build berikutnya ±1 menit (Gradle cache di `/var/data/salfanet/gradle-cache`).

---

## [2.24.0] — 2026-04-26

### Removed
- **Update otomatis via web panel dihapus** ([`4692059`]) — Fitur update via browser (SSE live log, tombol Apply Update / Force Rebuild) dihapus karena tidak reliable: bash script `update.sh` selalu mati saat `pm2 stop` dipanggil dari dalam Next.js process group, menyebabkan `.next` terhapus dan server 502 yang harus dipulihkan manual. File yang dihapus:
  - `scripts/update.sh` — script update yang dipanggil via API
  - `src/app/api/admin/system/update/route.ts` — SSE API endpoint (GET stream + POST trigger)
  - Halaman `/admin/system` diganti menjadi halaman **Informasi Sistem** statis: versi, commit, Node.js, uptime, banner update tersedia, dan panduan SSH siap-copy untuk update manual

### Fixed
- **`vps-install/updater.sh`: default ke `--branch master`** ([`5aa05b7`]) — Menjalankan `bash updater.sh` tanpa flag sebelumnya masuk ke Mode B (GitHub Releases) yang langsung error 404 karena repo tidak menggunakan GitHub Releases. Sekarang jika tidak ada `--branch` maupun `--version`, script otomatis pakai `--branch master`.

---

## [2.23.0] — 2026-04-26

### Removed
- **Coordinator role dihapus sepenuhnya** ([`e0cd701`]) — Role coordinator adalah fitur yang tidak pernah selesai diimplementasi. Semua endpoint API tidak pernah dibuat, sehingga halaman-halamannya selalu error. File yang dihapus:
  - `src/app/coordinator/` — seluruh direktori portal coordinator (dashboard, tasks)
  - `src/app/admin/coordinators/` — halaman manajemen coordinator di admin panel
  - `src/locales/id.json` — key `coordinator`, `coordinatorLogin`, `manageCoordinators`, namespace `"coordinator"` (~40 key), dan `"senderType_COORDINATOR"` dihapus
  - `src/app/admin/tickets/[id]/page.tsx` — `COORDINATOR` dihapus dari `SenderType` union type dan dari objek styling `getSenderBadgeColor()`
- **Firebase Admin SDK & FCM dihapus** ([`fdc730b`]) — Seluruh integrasi Firebase Cloud Messaging dihapus. Push notification kini menggunakan VAPID Web Push murni (tidak ada dependency firebase-admin). File yang dihapus: `src/server/push.service.ts`, `firebase-service-account.json`. Stub `firebase-admin` di `src/lib/` digantikan dengan implementasi VAPID native.

### Added
- **`src/cron/runner.ts` — Cron runner baru berbasis tsx** ([`fdc730b`]) — Menggantikan `cron-service.js` (Node.js CJS) dengan TypeScript runner yang dijalankan via `npx tsx`. 16 cron jobs diload dari satu entry point, distributed locking tetap aktif. FreeRADIUS Health Check berjalan 5 detik setelah startup.
- **`production/ecosystem.config.js` — Template konfigurasi PM2** ([`fdc730b`]) — File baru sebagai source of truth untuk konfigurasi PM2. `salfanet-cron` kini berjalan sebagai proses fork (`npx tsx src/cron/runner.ts`) dengan `NODE_OPTIONS: '--conditions=react-server'` (wajib agar `server-only` package tidak throw di luar Next.js).
- **`vps-install/cleanup-refactor.sh` — Script cleanup instalasi lama** ([`f71256c`], [`c41f44f`]) — Script idempotent untuk membersihkan file-file stale dari instalasi sebelum refactor. Fitur:
  - Support `--dry-run` (preview tanpa hapus)
  - Phase 1: cleanup Firebase/FCM push service, firebase-service-account.json
  - Phase 3: sync `ecosystem.config.js` dari `production/` (migrasi cron-service.js → tsx runner)
  - Phase 8: hapus `src/app/coordinator/`, `src/app/admin/coordinators/`
  - Auto-deteksi jika `salfanet-cron` masih pakai `cron-service.js` → migrate ke tsx runner otomatis
  - Usage: `bash vps-install/cleanup-refactor.sh [--dry-run] [--app-dir=/path]`

### Changed
- **`scripts/update.sh`: refactor-aware** ([`f71256c`]) — Update script (dipanggil via admin panel → `/api/admin/system/update`) ditingkatkan:
  - Setelah `git reset --hard`, otomatis copy `production/ecosystem.config.js` → root (file ini untracked, tidak tereset oleh git)
  - Cleanup stale files dari Phase 1-8 refactor (push.service.ts, coordinator, firebase, dll.) di setiap update
  - PM2 cron restart: jika `ecosystem.config.js` berubah → `pm2 delete` + `pm2 start` ulang (bukan sekedar `pm2 restart`)
  - `pm2 save` otomatis setelah restart
- **`vps-install/updater.sh`: refactor-aware** ([`f71256c`]) — CLI update script ditingkatkan:
  - `npm ci` dengan fallback ke `npm install --production=false` jika lock file tidak sinkron (umum terjadi setelah refactor)
  - Copy `production/ecosystem.config.js` setelah `git clean -fd`
  - Cleanup stale files refactor (list sama dengan update.sh)
  - Copy static assets ke `.next/standalone` setelah build
  - PM2 cron: deteksi perubahan script → `pm2 delete` + `pm2 start` jika perlu

### Fixed
- **`cleanup-refactor.sh`: `set -e` safe** ([`c41f44f`]) — Fungsi `remove_path()` sebelumnya `return 1` saat file tidak ditemukan → script keluar prematur karena `set -e`. Diperbaiki ke `return 0`. Kondisi `diff` juga diperbaiki (inversi `!` yang salah menyebabkan ecosystem.config.js tidak pernah disync).

---

## [2.22.0] — 2026-04-26

### Added
- **Script `scripts/backup-freeradius-local.sh`** ([`8652ea4`]) — Script bash untuk membuat arsip `.tar.gz` seluruh direktori `/etc/freeradius/3.0/` ke `backups/freeradius/` dengan nama file bertimestamp (`freeradius-config-YYYYMMDD-HHMMSS.tar.gz`). Otomatis cleanup backup lama (simpan 10 terbaru). Output baris `BACKUP_FILE: <nama>` di akhir agar UI polling bisa deteksi selesai. Script sebelumnya tidak ada sehingga tombol "Buat Backup" selalu gagal dengan error `Script not found`.

### Fixed
- **Restore FreeRADIUS: error "same file" saat restore `mods-enabled/`** ([`c268123`]) — File `mods-enabled/sql` dan `mods-enabled/rest` di FreeRADIUS adalah **symlink** ke `../mods-available/sql`. Saat tar mengekstrak backup, symlink tetap sebagai symlink. Perintah `cp symlink dest` gagal karena keduanya resolve ke file fisik yang sama (`cp: ... are the same file`). Fix: cek tipe file via `stat -c '%F'` sebelum restore — jika `symbolic link`, gunakan `ln -sf <target> <dest>` alih-alih `cp`.
- **Build VPS: OOM (Out of Memory) saat fase TypeScript check** ([`0aee02f`]) — Build `npm run build` menjalankan TypeScript type-checker (`tsc`) setelah compile selesai. Pada VPS 4GB dengan PM2 berjalan, proses `tsc` membutuhkan heap hingga 1.6GB dan di-kill oleh OOM killer (`FATAL ERROR: Ineffective mark-compacts near heap limit`). Fix: set `typescript.ignoreBuildErrors: true` di `next.config.ts` untuk skip fase `tsc` saat build produksi (type error tetap terdeteksi di development/editor).
- **Build VPS: OOM saat build karena PM2 mengonsumsi RAM** ([`08eba82`]) — PM2 process salfanet-radius mengonsumsi ~500MB RAM saat berjalan. Dengan heap build 1536MB (bawaan `npm run build`), total RAM yang dibutuhkan melebihi 4GB. Fix: `update.sh` kini stop PM2 sebelum build dan gunakan `npm run build:low-mem` (heap 1024MB). PM2 distart kembali setelah build selesai (atau gagal).
- **Build VPS: script baru tidak executable setelah `git reset --hard`** ([`8ce6421`]) — Script yang ditambahkan via commit baru tidak otomatis dapat izin execute di VPS setelah `git reset --hard`. Fix: tambah `chmod +x scripts/*.sh` di `update.sh` setelah git reset.
- **VPN Client: list tidak refresh setelah tambah client** ([`b55d3e6`]) — Setelah berhasil tambah WireGuard atau L2TP client, list VPN tidak diperbarui otomatis. Fix: panggil `loadClients()` di success path WireGuard dan L2TP.
- **VPN Client: modal tidak menutup / formData tidak ter-reset setelah submit** ([`b55d3e6`]) — Form WireGuard menggunakan `formData.name` setelah `formData` di-clear sehingga nama yang dikirim ke credentials dialog kosong. Fix: simpan nama ke variabel lokal `peerName` sebelum clear, gunakan `peerName` di credentials dialog.
- **VPN Client: IP pool tidak bisa dipakai ulang (orphan WG peers)** ([`288a094`]) — Peer WireGuard yang dihapus dari DB tetap tersisa di `wg.conf`. Saat tambah client baru, `nextAvailableIp` membaca `wg.conf` dan skip IP yang sebenarnya sudah bebas. Fix: tambah langkah cleanup orphan peers di `wg.conf` (compare dengan DB) sebelum alokasi IP baru.
- **VPN Client delete: peer tidak dihapus dari `wg.conf` di VPS** ([`db9ae7a`]) — Handler DELETE untuk `vpnServerId === '__vps_wg_server__'` hanya menghapus record DB tanpa menghapus `[Peer]` di `wg0.conf`. Fix: tambah call ke `POST /api/network/vps-wg-peer` dengan `action: 'remove'` sebelum delete DB.
- **Auto-create NAS/router saat tambah VPN client WireGuard** ([`701bfb7`]) — Endpoint `vps-wg-peer` secara otomatis membuat NAS record dan router saat tambah peer. Fix: hapus blok auto-create — NAS dikelola terpisah.
- **Auto-create NAS/router saat tambah VPN client L2TP** ([`8303308`]) — Sama seperti WireGuard, endpoint `vps-l2tp-peer` juga membuat NAS otomatis. Fix: hapus blok auto-create.
- **Panel redundansi di halaman VPN Client & VPN Server masih tampil** ([`8303308`], [`096d446`]) — Panel "Setup RADIUS Redundancy" yang sudah diputuskan untuk dihapus masih ter-render karena ada sisa JSX dan komponen stub. Fix: komponen `VpnServerRedundancyPanel` dijadikan stub `return null`, semua JSX orphan dibersihkan.

### Changed
- **`update.sh`: safe zero-downtime update** ([`08eba82`], [`8ce6421`], sesi ini) — Perbaikan menyeluruh pada script update:
  - `.env` di-backup ke `/tmp/salfanet-env-backup-<timestamp>` sebelum `git reset --hard` (extra safety meski `.env` ada di `.gitignore`)
  - Jika `.env` hilang setelah git reset, otomatis restore dari backup terakhir
  - Cleanup direktori orphan dari deployment lama (`srcappadmin`, `srclocales`, dll.) otomatis tiap update
  - PM2 `reload` (rolling zero-downtime) tetap digunakan saat restart — sesi PPPoE/Hotspot aktif tidak terputus oleh update kode
  - PM2 direstart (safety net) bahkan jika build gagal — server tidak dibiarkan mati
  - Tmp env backup lama (>7 hari) dibersihkan otomatis
  - Komentar safety guarantee ditambahkan di header script

---

## [2.21.0] — 2026-04-22

### Added
- **Panel "Konfigurasi VPS Built-in VPN" di halaman VPN Client** ([`1903085`]) — Panel baru (collapsible) untuk mengatur pool IP & gateway WireGuard dan L2TP yang terinstall langsung di VPS (bukan MikroTik CHR). Menampilkan IP Mulai, IP Akhir, Gateway VPS beserta tombol Edit inline. Konfigurasi ini terpisah sepenuhnya dari menu VPN Server yang khusus untuk MikroTik CHR.
- **PATCH endpoint `vps-wg-peer`** ([`1903085`]) — Endpoint baru `PATCH /api/network/vps-wg-peer` untuk update `poolStart`, `poolEnd`, `gatewayIp` di `wg-server-info.json`. Saat `gatewayIp` disimpan, endpoint juga otomatis memperbarui baris `Address =` di `wg0.conf` dan field `subnet` di info file, lalu reload WireGuard interface via `wg syncconf` (zero-downtime).
- **PATCH endpoint `vps-l2tp-peer`** ([`1903085`]) — Endpoint baru `PATCH /api/network/vps-l2tp-peer` untuk update `poolStart`, `poolEnd`, `gateway` di `l2tp-server-info.json`.
- **Pool IP menerima full IP address** ([`17d83da`]) — Input poolStart/poolEnd sebelumnya hanya menerima angka oktet terakhir (mis. `2`, `254`). Sekarang menerima full IP lengkap (mis. `172.16.212.2`) sehingga pool bisa dikonfigurasi ke subnet manapun, tidak terbatas pada subnet WireGuard interface default.

### Fixed
- **VPN Client page: redirect paksa jika tidak ada MikroTik CHR** ([`1903085`]) — Halaman VPN Client sebelumnya memaksa redirect ke menu VPN Server jika belum ada VPN Server (CHR) terdaftar, sehingga user tidak bisa pakai VPS built-in VPN (WG/L2TP) tanpa setup CHR dulu. Redirect dihapus sepenuhnya.
- **`loadWgServerInfo`: semua field undefined** ([`17d83da`]) — Fungsi membaca `data.info?.publicIp`, `data.info?.publicKey`, dst., padahal API `GET /api/network/vps-wg-peer` mengembalikan fields di top level (`data.publicIp`, bukan `data.info.publicIp`). Mapping diperbaiki ke `data.X` langsung.
- **`nextAvailableIp` (WG) & `getNextAvailableIp` (L2TP): selalu gunakan prefix `info.subnet`** ([`8636800`]) — Meskipun poolStart dikonfigurasi ke subnet lain (mis. `172.16.212.2`), IP yang dialokasikan tetap menggunakan prefix interface WireGuard default (`10.200.0.x`). Sekarang jika poolStart adalah full IP string, prefixnya digunakan sebagai base. Scan "IP yang sudah terpakai" juga dibatasi ke prefix yang sama untuk menghindari false conflict lintas subnet.
- **WG ADD response: `vpnSubnet` dan `gatewayIp` tidak mencerminkan pool prefix** ([`8636800`]) — Response POST add peer sekarang menghitung `effectiveVpnSubnet` dan `effectiveGatewayIp` dari prefix poolStart, bukan dari `info.subnet`. Script MikroTik yang di-generate (allowed-address, route, RADIUS address) otomatis menggunakan subnet yang benar.
- **Display subnet footer: selalu tampilkan subnet interface WG, bukan pool subnet** ([`6a8bd04`]) — Footer di panel kini menampilkan "Pool subnet" yang diturunkan dari prefix poolStart. Edit button prefill juga diperbaiki untuk menggunakan prefix dari poolStart yang sudah tersimpan (bukan selalu prefix `info.subnet`).
- **`wg0.conf Address` dan `info.subnet` tidak diupdate saat gatewayIp berubah** ([`62b0c88`]) — PATCH endpoint sekarang juga memperbarui baris `Address =` di `wg0.conf` dan `info.subnet` di JSON sehingga subnet yang ditampilkan di UI dan digunakan untuk alokasi IP selalu konsisten dengan konfigurasi pool.
- **Pool config dipindah ke halaman yang salah (VPN Server)** ([`1903085`]) — Konfigurasi pool IP built-in VPS sebelumnya salah ditempatkan di halaman VPN Server (khusus MikroTik CHR). Sekarang ada di halaman VPN Client yang tepat.

### Changed
- **VPN Server page dibersihkan dari state/handler/UI pool VPS** ([`1903085`]) — Semua state `wgPoolEdit`, `wgPoolForm`, `l2tpPoolEdit`, `l2tpPoolForm` beserta handler dan UI-nya dihapus dari `vpn-server/page.tsx`. Halaman VPN Server sekarang murni untuk manajemen MikroTik CHR.

---

## [2.20.0] — 2026-04-20

### Fixed
- **Script RADIUS: hapus perintah `rate-limit=""` di hotspot user profile** ([`9d5688d`]) — Command `/ip hotspot user profile add ... rate-limit=""` menyebabkan error `expected end of command` di RouterOS karena `rate-limit` bukan parameter valid di context tersebut. Block tersebut dihapus; RADIUS yang mengatur bandwidth via `Mikrotik-Rate-Limit` reply attribute.
- **Script RADIUS: `keepalive-timeout` dan `lcp-echo` tidak valid di `/ppp profile`** ([`fd3a1a0`], [`d0a9d82`]) — RouterOS tidak mengenal `keepalive-timeout` maupun `lcp-echo-interval`/`lcp-echo-failure` pada `/ppp profile set`. Kedua perintah dihapus dari generated script.
- **Script RADIUS: `address` selalu `127.0.0.1` saat `RADIUS_SERVER_IP` tidak di-set** ([`b511b88`]) — Fallback chain diperbarui: `RADIUS_SERVER_IP` → `VPS_IP` → **hostname dari `NEXTAUTH_URL`** → `127.0.0.1`. Instalasi tanpa env var eksplisit (VPS lokal/LXC) kini otomatis menggunakan IP yang benar dari `NEXTAUTH_URL`.
- **Script RADIUS: router non-VPN tidak menyertakan `src-address`** ([`34f953e`]) — Tanpa `src-address`, MikroTik memilih source IP dari routing table yang bisa berbeda dari `nasname` terdaftar di FreeRADIUS → request ditolak sebagai "unknown client". Sekarang `src-address` selalu di-set untuk semua router (VPN maupun direct/public IP).

### Added
- **Script RADIUS: Netwatch monitor RADIUS server** ([`9d5688d`]) — Generated script kini menyertakan `/tool netwatch add host=<RADIUS_IP> interval=30s` dengan `down-script` log warning dan `up-script` log info. MikroTik otomatis mencatat jika RADIUS tidak reachable.
- **`vpn-watchdog.sh`: RADIUS health check** ([`c2aa096`]) — Watchdog kini memeriksa apakah service `freeradius` sedang berjalan (Check A) dan apakah port UDP 1812 listening (Check B), serta auto-restart jika service mati. Ditambahkan log rotation otomatis (max 5000 baris).

### Changed
- **`Acct-Interim-Interval` FreeRADIUS: 60 → 300 detik** ([`c2aa096`]) — Interval akuntansi diperpanjang dari 1 menit ke 5 menit untuk mengurangi beban DB dan selaras dengan setting PPP interim-update MikroTik (`interim-update=5m`).
- **Stale session threshold `pppoe-session-sync.ts`: 1 HOUR → 30 MINUTE** ([`c2aa096`]) — Sesi tanpa `Accounting-Interim` lebih dari 30 menit (= 6× interval 5 menit) dianggap stale dan ditutup. Memberi window cukup bagi VPN untuk reconnect tanpa menutup sesi aktif secara prematur.

---

## [2.19.0] — 2026-04-11

### Added
- **Tab "📷 Foto" di `UserDetailModal`** ([`817887a`]) — Tab baru di sebelah kanan Invoice untuk melihat foto KTP dan foto instalasi pelanggan secara read-only. Fitur:
  - Foto KTP ditampilkan full-width dengan NIK di pojok kanan
  - Foto Instalasi ditampilkan grid 2×kolom dengan label "Foto 1/2/3…"
  - **Lightbox**: klik foto manapun → full screen overlay, klik luar atau tombol × untuk tutup
  - Placeholder kosong dengan ikon jika belum ada foto

### Improved
- **Kompresi foto otomatis sebelum upload** ([`8ff86c1`]) — Semua foto yang diambil via kamera maupun dipilih dari galeri dikompresi otomatis sebelum dikirim ke server:
  - Util baru `compressImage()` di `src/lib/utils.ts`: resize max **1280×1280px** + JPEG quality **78%**
  - Estimasi ukuran: foto ~5MB dari HP 50MP → **200–400KB** tersimpan di database
  - Berlaku di `CameraPhotoInput` (galeri, native capture, getUserMedia takePhoto) dan `CameraViewfinder` (takePhoto + native capture)
- **Tampilan viewfinder kamera diperbaiki** ([`8ff86c1`]) — Viewfinder live camera dari fixed `h-48` (192px) → `aspect-[4/3]` (proporsional). Ditambahkan **corner guide overlay** (4 sudut biru) di viewfinder dan `CameraViewfinder`.
- **Preview foto hasil diperbaiki** ([`8ff86c1`]) — Border berubah jadi hijau, badge "✓ Foto tersimpan" di pojok kiri atas, action bar (Galeri | Kamera) di bagian bawah foto.

### Fixed
- **`getUserMedia` error tidak fallback ke native camera** ([`382dbb3`]) — Sebelumnya jika `getUserMedia` melempar error apapun (`NotAllowedError`, Permissions Policy violation, tidak ada kamera, dll), komponen menampilkan pesan error merah "Izin kamera ditolak..." alih-alih fallback otomatis. Sekarang setiap error dari `getUserMedia` langsung memicu `captureRef.current?.click()` / `setUseNativeCapture(true)` sehingga native camera OS terbuka tanpa error.
  - `CameraPhotoInput.tsx` — `catch` block `startCamera()`: hapus seluruh `setCameraError(msg)` logic, ganti dengan `captureRef.current?.click()`
  - `CameraViewfinder.tsx` — `catch` block `startStream()`: hapus `setError(msg)`, ganti dengan `setUseNativeCapture(true)`
  - State `cameraError` dan render block error merah dihapus sepenuhnya dari `CameraPhotoInput`

---

## [2.18.0] — 2026-04-11

### Fixed
- **CRITICAL: Tombol "Kamera HP" tidak membuka kamera di iOS Safari / Android** — root cause: `<input type="file" capture="environment">` dengan `className="hidden"` (display:none) yang di-trigger via `ref.current?.click()` kehilangan "trusted user gesture" context, sehingga iOS Safari mengabaikan atribut `capture` dan membuka galeri biasa sebagai fallback.
  - `CameraPhotoInput.tsx` — Ganti `useRef` + `button` + `.click()` dengan `useId()` + `<label htmlFor>`. Label trigger input secara native tanpa JavaScript, iOS Safari menghormati `capture="environment"` dengan benar. Ganti `className="hidden"` (display:none) ke `className="sr-only"` (off-screen, elemen tetap aktif di DOM).
  - `admin/pppoe/users` Add form — `className="hidden"` → `"sr-only"`; tambah `pointer-events-none` pada label saat uploading.
  - `UserDetailModal` Edit form — sama seperti di atas.
- **Foto Instalasi hilang di form Teknisi (`/technician/register`)** — section Foto Instalasi sama sekali tidak ada di form tambah pelanggan teknisi. Ditambahkan `CameraPhotoInput` multi-foto dengan state `installationPhotos`, `uploadingInstallation`, dan dikirim ke API saat submit.

### Changed
- **Script RADIUS, Isolir, dan VPN Client dipisah tanggung jawabnya** *(commit d649bee)*:
  - `setup-radius` — Hapus profile duplikat `radius-default`, konsolidasi ke satu profile `salfanetradius`. Semua rule isolasi (SALFANET-ISOLIR) dipindahkan ke Setup Isolir.
  - `setup-isolir` — Diubah dari eksekusi API langsung (RouterOSAPI) ke **script generator** (paste-able ke terminal MikroTik). Script mencakup: `pool-isolir`, PPP profile `isolir`, firewall filter + NAT (SALFANET-ISOLIR), catatan route VPS.
  - `routers/page.tsx` — Ditambah tombol **Setup Isolir** (ikon gembok oranye) di samping tombol RADIUS, dengan handler `handleSetupIsolir()` yang menampilkan script modal.
  - `vpn-client/page.tsx` — Hapus `radiusSection` dan `wgRadiusSection` dari semua script VPN (L2TP/SSTP/PPTP/WireGuard). Script hanya berisi setup tunnel + API user + catatan langkah berikutnya.

---

## [2.17.0] — 2026-04-10

### Added
- **`CameraPhotoInput` component** — Komponen reusable baru `src/components/CameraPhotoInput.tsx`. Menampilkan dua tombol **[🖼 Galeri] [📷 Kamera HP]** side-by-side saat belum ada foto. Tombol *Kamera HP* menggunakan `capture="environment"` sehingga langsung membuka kamera belakang di HP tanpa melalui file picker. Setelah upload berhasil, komponen otomatis meminta izin GPS via `navigator.geolocation.getCurrentPosition` dan menampilkan badge **📍 lat, lng · Lihat di Maps ↗** yang dapat diklik. Theme `dark` (cyberpunk) untuk halaman publik, theme `light` untuk modal admin/teknisi.
- **Kamera HP langsung di form tambah pelanggan (`/daftar`)** — Upload foto KTP diganti dengan `CameraPhotoInput` (dark theme). GPS yang tertangkap otomatis mengisi `formData.latitude` dan `formData.longitude` — bisa melengkapi atau menggantikan input MapPicker manual.
- **Kamera HP + GPS di `AddPppoeUserModal` (`/admin/pppoe/users`)** — Foto KTP menggunakan `CameraPhotoInput`. Foto instalasi mendapat dua tombol [Galeri] [Kamera HP]; memilih via kamera HP otomatis menangkap GPS ke field latitude/longitude.
- **Kamera HP di form registrasi teknisi (`/technician/register`)** — Foto KTP diganti dengan `CameraPhotoInput`. Menampilkan badge GPS setelah foto diambil dari kamera.
- **Kamera HP + GPS di `UserDetailModal`** — Foto KTP menggunakan `CameraPhotoInput`. Foto instalasi mendapat [Galeri] [Kamera HP]; kamera otomatis mengisi GPS ke `formData.latitude/longitude`.

### Changed
- **Unified photo upload UX** — Semua 4 titik entry pelanggan (daftar publik, modal tambah admin, form teknisi, edit user) sekarang konsisten: dua aksi foto (galeri vs kamera HP), preview langsung, GPS otomatis setelah foto, badge koordinat clickable ke Google Maps.

---

## [2.16.0] — 2026-04-10

### Added
- **PWA Web Push — Sistem notifikasi push penuh (VAPID)** — notifikasi push browser bekerja di semua portal (customer, teknisi, admin). Teknisi dan admin kini dapat menerima notifikasi push Android/PWA untuk tiket, gangguan, dan broadcast.
- **`adminPushSubscription` model** — tabel baru `admin_push_subscriptions` untuk menyimpan push subscription admin/operator yang login melalui portal teknisi (`admin_user` type). Sebelumnya diabaikan dengan `{skipped:true}`.
- **Toggle notif push permanen di sidebar teknisi** — `SidebarPushToggle` selalu tampil di sidebar portal teknisi dengan state ON/OFF yang jelas. ([`d0a97ec`])
- **Dispatch tiket ke semua teknisi via WA + push** — saat tiket dibuat/di-assign, broadcast WhatsApp + push notification dikirim ke semua teknisi aktif. ([`1eb9358`])
- **GitHub Actions auto-deploy** — workflow `.github/workflows/deploy.yml` untuk auto-deploy ke VPS saat ada push ke branch `master`. ([`e195e4f`])
- **`update.sh` auto-rebuild jika standalone hilang** — jika `.next/standalone/server.js` tidak ada, build dipaksa meski kode tidak berubah. API `/api/admin/system/check` mengembalikan `needsBuild: true` dan UI menampilkan tombol rebuild. ([`8ee6c03`])
- **Bell push + badge di portal teknisi** — SW menangani `push` event, menampilkan notifikasi, badge, dan toast dari service worker. ([`72665f0`])
- **Silent sync push subscription** — saat portal teknisi/customer dimuat di browser, jika browser masih punya push subscription aktif, langsung di-sync ulang ke DB tanpa user perlu re-toggle.

### Fixed
- **CRITICAL: Push subscription tidak tersimpan ke DB (semua tabel 0 row)** — root cause: `fetch('/api/push/technician-subscribe', ...)` tidak mengirim cookie `technician-token` karena tidak ada `credentials: 'same-origin'`. Tanpa cookie, `admin_user` tidak terdeteksi → API mencari ID di tabel `technician` → 404 "Technician not found" → subscription tidak tersimpan. Fix: tambah `credentials: 'same-origin'` ke semua 3 fetch call (silent sync, subscribe, unsubscribe). ([`57f6169`])
- **CRITICAL: `admin_user` push subscription diabaikan** — route `POST /api/push/technician-subscribe` mengembalikan `{skipped:true}` untuk `admin_user` tanpa menyimpan data. Sekarang menyimpan ke `adminPushSubscription`. ([`7df3a8f`])
- **Push 404 untuk `admin_user`** — route `GET /api/push/vapid-public-key` dan subscribe/unsubscribe mengembalikan 404 saat user adalah `admin_user`. Diperbaiki dengan early return yang benar. ([`1ef8edc`])
- **`PushManager` in `window` vs `navigator`** — `SidebarPushToggle` menggunakan `PushManager in window` (sesuai spec) bukan `PushManager in navigator`, konsisten dengan `usePushNotification` hook. ([`c31a316`])
- **Dashboard teknisi: tiket selesai tidak muncul** — dashboard masih menggunakan model `work_orders` yang sudah dihapus. Diperbarui ke model `ticket`. ([`1602b7e`], [`ed3619b`])
- **PPPoE username GenieACS** — username untuk lookup GenieACS dinormalisasi dengan benar. ([`72665f0`])
- **WA notif teknisi melalui `WhatsAppService`** — notifikasi WhatsApp ke teknisi sekarang melalui service standar. ([`72665f0`])

### Changed
- **`push-notification.service.ts`** — `getPushDashboardStats()` mengembalikan `adminSubscribers` + `fcmUserCount`. `sendWebPushBroadcast()` juga mengirim ke admin saat target `technician` atau `all`. `sendToStoredSubscriptions()` mendukung role `'admin'`.
- **Admin push notifications page** — menampilkan breakdown terpisah: Teknisi X teknisi, Admin X admin, dan total penerima yang benar.
- **Cleanup: hapus file patch sementara** — `scripts/patch-push-fix.mjs`, `scripts/patch-push-toggle.mjs`, `scripts/patch-push-toggle2.mjs`, `tmp-check.sh` dihapus dari repo.

### Migration
- Tabel `admin_push_subscriptions` dibuat otomatis via `prisma db push` (field di schema.prisma sudah ditambahkan).

---

## [2.15.0] — 2026-01-15

### Fixed — Cron Job & Backup System Audit
- **CRITICAL: `backupTopicId` non-nullable** — field di schema `telegramBackupSettings` sebelumnya `String` (wajib), menyebabkan Prisma error saat simpan settings tanpa Topic ID → settings tidak tersimpan → backup Telegram selalu di-skip. Diubah ke `String?` (nullable)
- **CRITICAL: `MYSQL_PWD` shell syntax** — sebelumnya menggunakan `MYSQL_PWD="${password}" mysqldump ...` yang gagal jika password DB mengandung karakter khusus (`"`, `$`, `` ` ``, `\`). Sekarang menggunakan `env` option dari `execAsync` yang lebih aman
- **CRITICAL: `/api/cron/telegram` GET undefined `status`** — variabel `status` tidak pernah di-declare, `getTelegramCronStatus()` diimport tapi tidak dipanggil → runtime error saat cek status. Fixed
- **CRITICAL: `/api/cron` POST tanpa auth** — endpoint bisa dipanggil siapa saja dari internet. Ditambahkan auth check: `CRON_SECRET` header, User-Agent `SALFANET-CRON-SERVICE`, atau session SUPER_ADMIN
- **Double cron execution** — `initCronJobs()` di `instrumentation.ts` DAN `cron-service.js` menjalankan job yang sama (voucher sync, agent sales, invoice, dll). Sekarang `initCronJobs()` hanya menginisialisasi Telegram cron (yang memang tidak ada di cron-service.js)
- **Placeholder `/api/backup/telegram/settings`** — endpoint mengembalikan data hardcoded `{ enabled: false }` dan tidak baca/tulis DB. Sekarang baca/tulis ke database `telegramBackupSettings`

### Improved
- **Health report Telegram** — sekarang menampilkan informasi lengkap: active sessions, total users, active users, overdue invoices, issues (sebelumnya hanya status, size, tables, connections, uptime)
- **Telegram file size check** — tambah validasi 50MB limit sebelum kirim backup ke Telegram, mencegah silent failure dari Telegram API

### Migration
- `prisma/migrations/20260615_fix_telegram_backup_topic_nullable.sql` — `ALTER TABLE telegram_backup_settings MODIFY COLUMN backupTopicId VARCHAR(191) NULL`

---

## [2.14.0] — 2026-01-15

### Added
- **ID Pelanggan (`customerId`) di semua template notifikasi WA** — template yang diperbarui:
  - `registration-approval` — menampilkan ID pelanggan sebelum username
  - `admin-create-user` — menampilkan ID pelanggan + area
  - `invoice-reminder` — menampilkan ID pelanggan di detail invoice
  - `payment-success` — menampilkan ID pelanggan, paket, dan area
  - `auto-renewal-success` — menampilkan ID pelanggan + area
  - `manual-payment-approval` — menampilkan ID pelanggan, paket, dan area
  - `manual-payment-rejection` — menampilkan ID pelanggan dan username
  - `account-info` — menampilkan ID pelanggan
- **ID Pelanggan di template email** — ditambahkan ke:
  - `registration-approval` — baris ID Pelanggan sebelum Username
  - `manual-payment-approval` — baris ID Pelanggan di tabel detail
  - `manual-payment-rejection` — baris ID Pelanggan + Username di tabel detail
- **Field `customerId` di service interfaces** — `sendRegistrationApproval`, `sendPaymentSuccess`, `sendAutoRenewalSuccess`, `sendInvoiceReminder` (WA + Email) sekarang menerima `customerId?: string`
- **Field `area` di notifikasi payment-success dan auto-renewal-success** — service interfaces + variabel template diperbarui

### Fixed
- **Seed template tidak update `message`/`htmlBody`** — bug di `whatsapp-templates.ts` dan `email-templates.ts`: branch `update` tanpa flag `--force-templates` hanya meng-update `name` dan `isActive`, BUKAN konten pesan. Sekarang `message`/`htmlBody` selalu diupdate pada setiap seed.
- **`update.sh` tidak menjalankan seed** — seed hanya berjalan jika file di `prisma/seeds/` berubah. Sekarang seed selalu berjalan di setiap update.

### Changed
- **`update.sh` menggunakan `stdbuf`** — `npm run db:seed` dibungkus dengan `stdbuf -oL` agar output log muncul secara real-time di SSH / admin live log panel

---

## [2.13.2] — 2026-04-05

### Changed
- **Redesign UI: Modern Clean Blue/Indigo theme** — seluruh halaman login (admin, technician, customer, agent) didesain ulang dari cyberpunk/neon ke tampilan modern bersih dengan palette biru/indigo. Sidebar dan komponen global mengikuti skema warna baru. ([`6ec9783`])
- **`CyberButton` — warna diperbarui** — semua warna neon (cyan/pink/yellow/green) diganti ke blue/indigo/emerald palette yang konsisten dengan tema baru. ([`6ec9783`])
- **`globals.css` — CSS variables diperbarui** — dark mode: navy background + blue primary; light mode: blue-600 primary; dark mode neon remap dihapus; custom scrollbar diperbarui. ([`6ec9783`])

### Fixed
- **VPN Client: VPS IP field hanya manual** — auto-fill VPS IP sekarang skip domain name (Cloudflare-proxied, dsb). Field VPS IP di halaman VPN Client menjadi input manual penuh — tidak lagi menarik domain dari API. ([`910cddd`], [`5049e02`])
- **`scripts/update.sh` — abort jika copy static gagal** — sebelumnya menggunakan `|| true` sehingga kegagalan copy aset statis diabaikan dan `pm2 reload` tetap dipanggil dengan build stale. Sekarang menggunakan `|| err "..."` untuk abort. ([`7c85dd3`])
- **`scripts/update.sh` — nesting bug `cp -r`** — `cp -r .next/static .next/standalone/.next/static` bisa membuat nested directory jika target sudah ada. Diperbaiki ke `mkdir -p` + `cp -r src/. dst/`. ([`7c85dd3`])

---

## [2.13.1] — 2026-04-05

### Fixed
- **Wablas send gagal** — ganti dari `POST /api/v2/send-message` (JSON body) ke `GET /api/send-message?token=...` (v1 simple endpoint). V2 endpoint tidak tersedia di semua server Wablas (`wa`, `deu`, `jakarta`, dll). Format token tetap `token.secret_key`. ([`e8bdf6b`])
- **Hint form Wablas** diperjelas: sebelumnya hanya "Opsional: token.secret_key", sekarang "Format: token.secret_key (dari Device → Settings di dashboard Wablas)".

---

## [2.13.0] — 2026-04-05

### Added
- **WhatsApp webhook endpoint** (`/api/whatsapp/webhook`) — terima pesan masuk dari Kirimi.id, Wablas, Fonnte, WAHA. Pesan dicatat ke `whatsapp_history` dengan `status: incoming`. Mendukung GET untuk challenge verification. ([`d2ff368`])
- **Webhook URL display** di halaman providers — panel info dengan URL webhook dan tombol copy. ([`48a213d`])
- **Kirimi.id native broadcast** — `sendBroadcastViaKirimi()` menggunakan endpoint `/v1/broadcast-message` untuk kirim ke banyak nomor sekaligus. Pesan dikelompokkan per konten unik untuk efisiensi. 1 penerima otomatis pakai `/v1/send-message`. ([`fa136f1`], [`f4b3d4c`])
- **Per-provider error detail** — saat semua provider gagal, response API menyertakan detail error per provider (nama, tipe, pesan error) agar mudah diagnosa. ([`b7e0544`])

### Fixed
- **Kirimi.id endpoint salah** — `/send-message` → `/v1/send-message` (sesuai docs resmi Kirimi.id v2.0). ([`11bc666`])
- **Kirimi.id field penerima salah** — `number` → `receiver` (sesuai docs resmi). ([`11bc666`])
- **Kirimi.id trailing slash** — `provider.apiUrl` sekarang di-strip trailing slash seperti provider lain. ([`b7e0544`])
- **Broadcast response mismatch** — route broadcast sekarang return `successCount` / `failCount` di top-level agar frontend toast menampilkan angka yang benar. ([`f4b3d4c`])
- **HTTP status 502 diubah ke 500** — 502 secara semantik berarti upstream proxy error; 500 lebih tepat untuk kegagalan provider. ([`b7e0544`])

### Changed
- **Broadcast delay Kirimi.id** diubah dari 5 detik → **30 detik** (rekomendasi resmi Kirimi.id untuk menghindari blokir WhatsApp). ([`2af263c`])

---

## [2.12.0] — 2026-04-02

### Fixed
- **Isolasi PPPoE manual: radusergroup dioverwrite saat edit user** — `updatePppoeUser` selalu menulis ulang `radusergroup = profile.groupName` tanpa memeriksa status user. Sekarang menghormati `effectiveStatus`: `isolated` → group `isolir`, `blocked`/`stop` → RADIUS kosong, `active` → sync penuh. ([`958fc3a`])
- **`radclient disconnect` tidak memuat MikroTik vendor dictionary** — tambahkan flag `-d /usr/share/freeradius` ke `coa-handler.service.ts` agar `Disconnect-Request` dikirim dengan format yang benar ke MikroTik. ([`958fc3a`])
- **CoA "Bad Requests=133, Acks=0"** — `coa.service.ts` tidak memuat MikroTik vendor dict, membuat `Mikrotik-Rate-Limit` dikirim tanpa vendor ID. Tambahkan `-d /usr/share/freeradius` ke `executeRadclient()`. ([`b2fe4fa`])
- **setup-isolir hardcode IP pool dan rate limit** — `setup-isolir/route.ts` tidak lagi hardcode `10.255.255.2-254 @ 64k/64k`. Sekarang baca `isolationIpPool` + `isolationRateLimit` dari DB company. ([`cb91699`])
- **9739 duplicate rows di `radgroupreply`** — `freeradius-health.ts` menggunakan `INSERT IGNORE` pada tabel tanpa UNIQUE constraint. Diganti pola `DELETE + INSERT` untuk semua 3 atribut isolir. ([`cb91699`])
- **footerAgent tidak tersimpan ke database** — field `footerAgent` ada di CREATE query tapi tidak di UPDATE. ([`2adef92`])
- **Footer login agent hardcoded** — hapus fallback `"Powered by ${poweredBy}"` yang dihardcode di `agent/page.tsx`. ([`f70967f`])

### Added
- **`production/99-vpn-routes`** — script PPP ip-up untuk otomatis menambahkan route `10.20.30.0/24` via ppp0 ke VPS saat VPN tunnel connect. Diperlukan agar CoA/disconnect packet bisa reach MikroTik.

### Changed
- Nginx config (`production/nginx-salfanet-radius.conf`) disinkronkan dengan VPS aktual: tambah blok `/api/` dengan no-cache headers, CSP header Cloudflare, `Referrer-Policy`, hide upstream security headers.

---

## [2.11.8] — 2026-03-31

### Fixed
- **billingDay reset ke 1 saat edit user** — `UserDetailModal.tsx` menggunakan `user.subscriptionType || 'PREPAID'` (wrong default). User POSTPAID tampil di view PREPAID, billingDay selalu reset ke 1. Fix: `subscriptionType: user.subscriptionType ?? 'POSTPAID'` dan `billingDay: user.billingDay ?? new Date(user.expiredAt).getDate()`.
- **MikroTik local-address verification** — setelah sync local-address ke RouterOS PPP profile, sekarang membaca kembali untuk verifikasi.
- **NAS IP di kolom tabel PPPoE** — menampilkan IP NAS/router, bukan IP statis user.
- **updatePppoeUser POSTPAID billingDay** — saat billingDay berubah, `expiredAt` di-recalculate ke tanggal tagihan berikutnya.
- **Ghost sessions** — `sessions/route.ts` skip session yang tidak ada di `pppoeUser` maupun `hotspotVoucher`. `authorize/route.ts` kirim REJECT untuk user tidak terdaftar.
- **Dashboard hotspot count selalu 0** — hapus pengecekan Service-Type yang keliru, ganti ke lookup `pppoeUser` vs `hotspotVoucher`.
- **Next.js prerender crash pada `/_global-error`** — buat `src/app/global-error.tsx` sebagai `'use client'` component.
- **MapPicker z-index di balik modal** — tambah `createPortal(jsx, document.body)` ke `MapPicker.tsx`.
- **Nginx manifest 404** — ganti `alias + try_files` (broken dengan regex location) ke `root /var/www/salfanet-radius/public`.

### Added
- Area badge (kuning, ikon MapPin) di kolom Data Pelanggan PPPoE.
- Form Tambah Pelanggan: select Area (opsional).
- 5 action button baru: Eye, Pencil, RefreshCw, Shield, Trash.
- Agent manual top-up: pilih rekening admin tujuan, upload bukti transfer.

---

## [2.11.6] — 2026-03-28

### Fixed
- **expiredAt reset otomatis saat save user** — dihapus kalkulasi otomatis `expiredAt` dari `billingDay` di setiap `updatePppoeUser`. `expiredAt` hanya diupdate jika eksplisit dikirim dari form.
- **Redis crash-loop setelah install** — hardening konfigurasi Redis installer.
- **Ubuntu UFW tidak auto-enabled** — installer sekarang auto-detect SSH port dan enable UFW.

### Added
- `scripts/run-deploy.js` — cross-platform deploy wrapper.
- `npm run clean:local` dan `clean:all`.
- GenieACS TR-069 device management (`/admin/network/olt`).
- WiFi configuration dari customer portal.

---

## [2.10.27] — 2026-03-15

### Added
- Technician portal (11 pages + 19 API routes).
- Restructuring complete (5 phases).

---

## [2.6.x] — 2025-12

### Added
- PPPoE isolation system dengan template WhatsApp/Email/HTML.
- `radgroupreply` untuk group `isolir`: `Mikrotik-Rate-Limit`, `Mikrotik-Group`, `Framed-Pool`.

---

## [2.4.x] — 2025-10

### Added
- CoA service (real-time disconnect via radclient + MikroTik API).
- Auto-disconnect cronjob.

