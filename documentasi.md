# Wallet API — Dokumentasi

Aplikasi wallet: admin panel (Svelte 5, `web2026/`) buat kelola akun admin, member, dan
transaksi (deposit/withdraw), plus API publik yang bisa dipanggil website game eksternal
buat lapor menang/kalah dan cek saldo member.

> **Arsitektur & business flow lengkap (diagram):** lihat [`ARCHITECTURE.md`](ARCHITECTURE.md).

## 1. Skema Database

Lihat [`sql/schema.sql`](sql/schema.sql). Tabel yang dipakai:

| Tabel | Fungsi |
|---|---|
| `tbl_user` | Akun member/wallet: `username`, `password`, `token` (kredensial buat API publik), `nama`, `saldo`, `status` |
| `tbl_trx_transaksi` | Ledger mutasi saldo (lihat aturan CREDIT/DEBIT di bawah) |
| `tbl_admin` | Akun admin panel |
| `tbl_counter` | Counter buat generate `notrx` (nomor transaksi berurutan, mis. `TRX-000001`) |

### Aturan CREDIT / DEBIT

Ditentukan oleh CHECK constraint `tbl_trx_tipe_source_ck` di schema — **tidak bisa diubah
dari kode**, cuma dua kombinasi valid:

| Tipe | Source | Kejadian |
|---|---|---|
| `CREDIT` | `DEPOSIT` | Admin deposit saldo ke member |
| `CREDIT` | `WIN` | Member menang game (dilapor website game via API publik) |
| `DEBIT` | `WITHDRAW` | Admin withdraw saldo member |
| `DEBIT` | `BET` | Member kalah / bayar taruhan (dilapor website game via API publik) — **field "payout game" dipetakan ke source `BET`** karena schema cuma punya 4 source itu, tidak ada source `PAYOUT` terpisah |

Saldo tidak boleh minus (`tbl_user_saldo_ck`) — kalau DEBIT bikin saldo minus, request ditolak
dengan `insufficient balance` sebelum sampai ke database.

## 2. Arsitektur Backend (Go / Fiber v3)

Layer per fitur, semua terhubung lewat interface di `domain/`:

```
domain/        -> struct entity + interface repository & service (kontrak)
dto/           -> request/response API (JSON)
internal/repository/  -> implementasi query SQL (pgx v5)
internal/service/      -> business logic, transaksi DB, cache Redis
internal/api/          -> handler Fiber (HTTP)
```

File wallet: `domain/wallet.go`, `domain/wallet_transaction.go`, dan pasangannya di
`dto/`, `internal/repository/`, `internal/service/`, `internal/api/`.

### Catatan migrasi Fiber v2 → v3

App sudah **dimigrasi penuh ke Fiber v3** (`github.com/gofiber/fiber/v3 v3.4.0`,
butuh Go 1.25+). Perubahan sintaks yang diterapkan:

- **Import**: semua `github.com/gofiber/fiber/v2` → `fiber/v3`; middleware
  (`etag`, `requestid`, `limiter`) ikut versi `fiber/v3`; JWT contrib pindah ke
  `github.com/gofiber/contrib/v3/jwt` (package-nya `jwtware`).
- **`fiber.Ctx` jadi interface** — handler berubah dari `func(c *fiber.Ctx) error`
  menjadi `func(c fiber.Ctx) error` (sama berlaku untuk `KeyGenerator`,
  `LimitReached`, `SuccessHandler`, `ErrorHandler`).
- **`ctx.BodyParser(&req)` dihapus** → diganti `ctx.Bind().Body(&req)` (binding baru
  Fiber v3, manual mode — error tetap dihandle manual oleh handler).
- **`app.Static` dihapus** → diganti middleware `static.New(root, cfg)` yang
  didaftarkan via `app.Use("/", ...)` (config `Compress`, `ByteRange`, `Browse`, dan
  `Index` diganti `IndexNames []string`).
- **Token JWT tidak lagi di `c.Locals("user")`** → contrib v3 menyimpan token via
  `fiber.StoreInContext` dengan key internal; ambil pakai `jwtware.FromContext(c)`
  yang mengembalikan `*jwt.Token` (golang-jwt/v5). Dipakai di `main.go` (SuccessHandler)
  dan `internal/api/auth.go` (Logout).
- **Rate limiter global `/api`**: storage Redis tidak lagi pakai
  `github.com/gofiber/storage/redis/v3` (package storage gofiber) — diganti adapter
  kecil `limiter_storage.go` (package main) yang membungkus **`*redis.Client`
  (`github.com/redis/go-redis/v9`, client `connection.RDB`)** langsung ke interface
  `fiber.Storage`. Jadinya rate limit & cache berbagi koneksi Redis yang sama.
- `app.Listen`/`app.Shutdown`/`app.ShutdownWithContext` tetap sama di v3.

### Konkurensi & Idempotency (mutasi saldo)

Setiap mutasi saldo (`internal/service/wallet_transaction.go`, fungsi `process`) jalan
dalam satu DB transaction:

1. `SELECT ... FOR UPDATE` pada baris `tbl_user` — mengunci baris member itu, jadi request
   konkuren buat username yang sama otomatis antre (mencegah race condition dua transaksi
   baca saldo lama yang sama).
2. Kalau `refno` sudah pernah dipakai buat username itu (mis. `invoice` yang sama dikirim
   ulang oleh website game karena retry), hasil transaksi lama langsung dikembalikan **tanpa**
   mutasi saldo lagi — mencegah double credit/debit.
3. Hitung `saldo_after`, tolak kalau DEBIT bikin saldo minus.
4. Generate `notrx` dari sequence Postgres `seq_notrx_transaksi` (lihat `util.NextSequenceValue`) —
   pakai `nextval()` yang atomic & non-blocking, bukan lagi row lock manual di `tbl_counter`
   (row lock lama serialize SEMUA transaksi wallet lewat satu baris, jadi bottleneck pas ada
   burst transaksi konkuren dan bisa timeout, lihat riwayat error "error select counter").
5. Insert baris `tbl_trx_transaksi`, commit.

## 3. Endpoint API

### 3.1 Admin panel (JWT, header `Authorization: Bearer <token>`)

| Method | Path | Fungsi |
|---|---|---|
| POST | `/api/auth` | Login admin → `{token}` |
| POST | `/api/auth/page` | Cek halaman (saat ini selalu sukses — lihat catatan di bawah) |
| POST | `/api/auth/logout` | Logout, blacklist token |
| POST | `/api/admin` | List admin |
| POST | `/api/admin/save` | Create/update admin (`type: "New"` atau `"Edit"`) |
| POST | `/api/member` | List member/wallet |
| POST | `/api/member/save` | Create/update member (`type: "New"` atau `"Edit"`) — saat create, `token` member digenerate otomatis (UUID) |
| POST | `/api/transaksi` | Riwayat transaksi, paginated (`username` kosong = semua member) |
| POST | `/api/transaksi/deposit` | Deposit saldo member (CREDIT/DEPOSIT) |
| POST | `/api/transaksi/withdraw` | Withdraw saldo member (DEBIT/WITHDRAW) |
| POST | `/api/dashboard` | Ringkasan dashboard admin: total deposit/withdraw/debit hari ini + total member & transaksi + chart debit/credit per bulan (12 bulan terakhir) |

**Response `/api/dashboard`** — selain 5 kartu statistik (deposit/withdraw/debit hari ini,
member, transaksi), `record` ikut bawa `chart` (array 12 entri, urut dari bulan 11 bulan lalu
sampai bulan berjalan, format `YYYY-MM`). Bulan tanpa transaksi tetap diisi `"0.00"` supaya
sumbu chart selalu penuh 12 bulan:
```json
{
  "status": 200,
  "message": "success",
  "record": {
    "total_deposit_today": "5000000.00",
    "total_withdraw_today": "1000000.00",
    "total_debit_today": "2500000.00",
    "total_member": 120,
    "total_transaksi": 4800,
    "chart": [
      { "bulan": "2025-09", "debit": "0.00", "credit": "0.00" },
      { "bulan": "2026-08", "debit": "3200000.00", "credit": "6800000.00" }
    ]
  }
}
```
`chart[].debit` = total `tipe = 'DEBIT'` (semua source) bulan itu, `chart[].credit` = total
`tipe = 'CREDIT'` (semua source). Endpoint di-cache Redis **TTL 1 jam**
(`RedisDashboardTTL` di `internal/service/dashboard.go`).

**Paging `/api/transaksi`** — body `{ username?, tipe?, limit?, offset? }`. `limit` default &
maksimal **500** (dipaksa ke 500 kalau kosong/negatif/lebih dari 500 — lihat `trxPageSize` di
`internal/service/wallet_transaction.go`). Response ikut nambahin `total` (jumlah keseluruhan
baris yang match filter, dipakai frontend buat hitung total halaman):
```json
{ "status": 200, "message": "success", "record": [ /* max 500 baris */ ], "total": 1234 }
```
Halaman berikutnya: kirim ulang dengan `offset = (halaman-1) * 500`.

> **Catatan:** fitur role/permission per-halaman (`tbl_adminrole`) dihapus karena tabelnya
> sudah tidak ada di schema baru — semua admin yang berhasil login sekarang **full access**
> ke seluruh endpoint admin panel.

### 3.2 API Publik (buat website game eksternal)

Diamankan pakai **API key**, bukan JWT — karena dipanggil server-to-server, bukan browser.
Kirim header:

```
X-API-KEY: <PUBLIC_API_KEY dari .env>
```

**POST `/api/public/transaction`** — lapor hasil game (menang/kalah):

```json
{
  "invoice": "12345",
  "pasaran": "Hongkong",
  "playerinvoice": "123451",
  "username": "budi",
  "credit": 0,
  "debit": 5000
}
```

- Wajib persis **satu** dari `credit`/`debit` yang > 0. `credit` > 0 → dicatat CREDIT/WIN.
  `debit` > 0 → dicatat DEBIT/BET (payout/kalah game).
- `invoice` = ID game/draw, **boleh sama** di banyak request (satu draw wajar punya banyak
  playerinvoice, bahkan bisa lebih dari satu buat username yang sama).
- `playerinvoice` dipakai sebagai `refno` (**kunci idempotency**) dan **harus unik per bet-slip
  per jenis kejadian (BET atau WIN)** — kalau `playerinvoice` yang sama dikirim ulang dengan
  jenis yang SAMA (mis. BET dua kali) buat username yang sama, request ke-2 dst dianggap
  duplikat (lihat respon `409` di bawah). **Jangan pernah pakai `invoice` sebagai idempotency
  key di sisi kamu** — karena memang sengaja diulang per draw, kalau dipakai buat dedup malah
  bikin bet-slip lain di draw yang sama ke-skip.
  > Idempotency di-scope per (username, playerinvoice, source) — jadi `playerinvoice` yang SAMA
  > boleh dipakai buat satu `BET` (pas taruhan dipasang) **dan** satu `WIN` (pas menang) buat
  > bet-slip yang sama; dua-duanya tetap tercatat & mutasi saldo masing-masing karena beda
  > `source`. Yang di-block cuma pengiriman ulang jenis yang sama persis.
- `invoice` dan `pasaran` tidak punya kolom khusus di schema, disimpan gabungan di kolom
  `keterangan` (format `invoice=...;pasaran=...`).

Response sukses (`status:200`) — sengaja minimal, cuma `invoice`+`playerinvoice` (buat website
game cocokkan ke request mereka), `username`, `balance` (saldo terbaru setelah transaksi), dan
`status: "COMPLETE"`, bukan detail ledger internal:

```json
{
  "status": 200,
  "message": "success",
  "record": {
    "invoice": "12345",
    "playerinvoice": "123451",
    "username": "budi",
    "balance": "105000.00",
    "status": "COMPLETE"
  }
}
```

Endpoint ini **synchronous** — begitu response `200` dengan `record.status: "COMPLETE"` balik,
saldo sudah ter-update & tersimpan di database saat itu juga (bukan diproses di background).
Website pemanggil **tidak perlu polling/nunggu status lain** — kalau responnya sudah balik
dengan `record.status: "COMPLETE"`, transaksinya sudah final.

**Kalau `playerinvoice` yang sama (buat username yang sama) dikirim ulang** — HTTP **409**,
BUKAN 200. Sengaja dibedain biar jelas kelihatan ini bukan transaksi baru yang berhasil diproses,
tapi request yang di-skip karena udah pernah:

```json
{
  "status": 409,
  "message": "duplicate transaction: playerinvoice already processed",
  "record": {
    "invoice": "12345",
    "playerinvoice": "123451",
    "username": "budi",
    "balance": "105000.00",
    "status": "DUPLICATE"
  }
}
```

`record.balance` tetap dikirim (saldo terkini) biar website game tetap bisa sinkron tanpa perlu
panggil `/balance` terpisah, tapi jangan salah artikan `409` ini sebagai "gagal total" — cuma
berarti "gak ada state baru yang berubah dari request ini", saldo di `record.balance` tetap valid.

Error yang mungkin dibalikin:

| Status | Kondisi |
|---|---|
| 400 | field wajib kosong / format salah (`record` berisi map error per-field) |
| 400 | `credit` dan `debit` dua-duanya 0 atau dua-duanya diisi |
| 400 | `insufficient balance` — debit bikin saldo minus |
| 401 | `X-API-KEY` salah/kosong |
| 404 | `username` tidak ditemukan |
| 409 | `playerinvoice` udah pernah diproses buat username ini (lihat contoh di atas) |

**POST `/api/public/balance`** — cek username + saldo terkini:

```json
{ "token": "<token member, lihat kolom tbl_user.token>" }
```

Response:
```json
{ "status": 200, "message": "success", "record": { "username": "budi", "balance": "150000.00" } }
```

Token per member bisa dilihat admin di halaman **Member** (tombol Edit → field Token, ada
tombol copy).

Error yang mungkin dibalikin:

| Status | Kondisi |
|---|---|
| 400 | `token` kosong |
| 401 | `X-API-KEY` salah/kosong |
| 404 | tidak ada member dengan `token` tersebut |

### 3.3 Notifikasi Error (Telegram)

Kalau ada **server error (500)** di endpoint manapun (admin, member, transaksi, public API),
sistem otomatis kirim pesan ke Telegram — tujuannya biar ketauan cepet tanpa harus buka log
Railway satu-satu. Implementasi: `internal/connection/telegram.go`.

**Setup:**
1. Bikin bot baru lewat [@BotFather](https://t.me/BotFather) di Telegram (`/newbot`) → dapet
   `TELEGRAM_BOT_TOKEN`.
2. Kirim 1 pesan apa aja ke bot itu (atau add ke grup), lalu buka
   `https://api.telegram.org/bot<TOKEN>/getUpdates` di browser → cari `"chat":{"id": ...}` →
   itu `TELEGRAM_CHAT_ID`.
3. Isi kedua env var itu di `.env` (lokal) atau dashboard Railway (production).

Kosongin salah satu/dua-duanya = fitur ini otomatis nonaktif (gak nge-block startup, gak error).

**Kapan kekirim (sengaja dibatasi cuma server error, BUKAN error bisnis biasa — biar gak spam):**

| Dikirim notif? | Contoh |
|---|---|
| ✅ Ya | Query DB gagal, koneksi database putus, bug internal lain (500) |
| ❌ Tidak | Validasi field kosong (400), saldo kurang (400), duplikat transaksi (409), username/password salah (401), member/username gak ketemu (404) |

**Format pesan:**
```
🚨 Wallet API — Server Error
Endpoint: PublicTransaction
Error: <pesan error asli>
username=budi
```

Dikirim dari titik-titik ini: `Admin.Index`, `Admin.Save`, `Member.Index`, `Member.Save`,
`Login`, dan `handleTrxError` (dipakai deposit/withdraw admin + `/api/public/transaction`).
Data sensitif (password, token) sengaja **tidak pernah** disertakan di pesan notifikasi.

## 4. Environment Variables

Lihat [`.env.example`](.env.example). Tambahan buat fitur ini:

| Variable | Fungsi |
|---|---|
| `DB_REDIS_NAME` | Index Redis DB (project ini pakai **DB 2**, bukan default 0) |
| `PUBLIC_API_KEY` | Secret yang harus dikirim website game lewat header `X-API-KEY` ke `/api/public/*` |
| `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID` | Notifikasi Telegram kalau ada server error (500) — lihat §3.3 |
| `RATE_LIMIT_MAX` | Maks request per window (default `300`) — rate limit global `/api` per IP, lihat §2 |
| `RATE_LIMIT_EXP` | Panjang window rate limit dalam menit (default `1`) |

## 5. Frontend (`web2026/`, Svelte 5 + Vite + Tailwind)

| Halaman | File | Fungsi |
|---|---|---|
| Login | `src/Login.svelte` | Login admin (username + password) |
| Dashboard | `src/dashboard/` | Kartu statistik: total deposit/withdraw/debit hari ini, total member, total transaksi + bar chart debit vs credit per bulan (12 bulan terakhir, pakai komponen Chart shadcn-svelte / LayerChart v2) |
| Admin | `src/admin/` | List + create/edit akun admin |
| Member | `src/member/` | List + create/edit akun member (saldo, status, lihat token) |
| Transaksi | `src/transaksi/` | Deposit/withdraw + riwayat transaksi semua member |

Setiap fitur punya 2 file: `<Fitur>.svelte` (ambil data lewat hook `src/lib/use<Fitur>.ts`,
cek token) dan `Home.svelte` (tampilan tabel + modal form, submit langsung ke API).

Semua request authenticated lewat satu wrapper `src/lib/api.ts` (`apiFetch`) — otomatis
nyetel header `Authorization: Bearer <token>`, dan **kalau backend balas 401 (token
invalid/expired) langsung auto-logout**: token dihapus (`setToken(null)`), `Root.svelte`
langsung render halaman Login. Jadi user yang token-nya expire tidak bisa tetap diam di
halaman admin.

**Komponen bersama:** `src/components/DepositWithdrawModal.svelte` — modal deposit/withdraw
yang dipakai di 2 tempat:
- Halaman **Transaksi**: tombol Deposit/Withdraw di header, username diisi manual.
- Halaman **Member**: 2 ikon (↓ biru = deposit, ↑ oranye = withdraw) di kolom SALDO tiap baris
  — modal kebuka dengan username baris itu udah otomatis keisi & terkunci (`usernameLocked`),
  jadi gak perlu bolak-balik ke halaman Transaksi buat top-up/potong saldo member tertentu.

Kedua tempat manggil endpoint yang sama (`/api/transaksi/deposit` / `/withdraw`, lihat §3.1)
dan refresh listnya masing-masing lewat prop `onSuccess`.

Build: `cd web2026 && npm run build` → hasil di `web2026/dist`, di-serve langsung oleh Go
lewat `app.Static("/", "./web2026/dist", ...)` di `main.go`.

## 6. Docker

```
docker build -t wallet-api .
docker run -p 6167:6167 --env-file .env wallet-api
```

`Dockerfile` sekarang 3 stage: build frontend (Node 22 → `web2026/dist`), build backend
(Go → binary `app`), lalu image final (`alpine`) yang cuma isi binary + `.env` + hasil
build frontend — jadi satu image serve API + admin panel sekaligus.

> **Catatan platform:** stage frontend pakai base image `node:22-bookworm-slim` (glibc), bukan
> `node:22-alpine` (musl), dan install dependency dari `package.json` langsung (bukan
> `npm ci` pakai `package-lock.json` yang di-commit) — supaya native binding Vite/rolldown
> ke-resolve buat platform Linux target build, bukan ketinggalan hasil resolve dari mesin dev
> (Windows). Kalau `package-lock.json` di repo sudah pernah di-regenerate di Linux/CI dan stabil,
> boleh balik pakai `npm ci` buat build yang lebih reproducible & cepat.

## 7. Known Issues / Follow-up

- `internal/repository/admin.go` fungsi `Update`: kolom `update_by` ke-isi `admin.Username`
  (username admin yang diedit), bukan `admin.Update` (username admin yang melakukan edit).
  Bug lama, bukan blocker, tapi audit trail "siapa yang update" jadi tidak akurat.
- Endpoint `/api/auth/page` sekarang selalu balas sukses buat request yang tervalidasi
  (tidak ada lagi pengecekan permission per-halaman) — dipertahankan cuma buat kompatibilitas,
  bisa dihapus kalau frontend tidak butuh lagi.
- Layer `internal/api/*` buat wallet (member/transaksi/public) baru ditambahkan di iterasi
  ini; kalau butuh endpoint tambahan (mis. nonaktifkan member, ganti/regenerate token,
  filter riwayat transaksi per tanggal), tinggal tambah method di service yang sudah ada.

## 8. Riwayat Update

- **Setup awal**: layer `domain/dto/repository/service` buat wallet (`Wallet` + `WalletTransaction`),
  ngikutin pola CREDIT (Deposit/Win) & DEBIT (Withdraw/Payout) sesuai schema.
- **Beres-beres modul lama**: hapus 60+ file domain/dto/repository/service/api fitur yang gak
  relevan sama wallet app (company, currency, pasarantoto, groupcompany, dst — sisa dari
  `totmaster_api`), termasuk fitur `adminrule` (permission per-role) karena tabelnya sudah
  gak ada di schema baru → admin sekarang full-access. Sinkronin ulang module path
  `totmaster_api` → `totwallet` di semua file (ngikutin perubahan `go.mod`). Redis diarahkan ke
  **DB 2** (`DB_REDIS_NAME=2`).
- **API layer lengkap**: `internal/api/wallet.go` (CRUD member), `internal/api/wallet_transaction.go`
  (deposit/withdraw/riwayat, buat admin panel), `internal/api/wallet_public.go` (API publik buat
  website game — `POST /api/public/transaction` & `POST /api/public/balance`), diamankan
  `X-API-KEY` (`internal/api/middleware.go`). Idempotency ditambahkan di `process()`
  (`internal/service/wallet_transaction.go`) berbasis `refno`, plus token per-member (UUID,
  digenerate otomatis saat create) buat lookup di `/api/public/balance`.
- **Frontend `web2026/`**: halaman Login, Admin (create/edit), Member (create/edit + lihat
  token), Transaksi (deposit/withdraw + riwayat). Endpoint yang tadinya mismatch sama backend
  asli (`/api/login`, `/api/adminall`, dst — peninggalan scaffold lama) diluruskan ke kontrak
  yang benar (`/api/auth`, `/api/admin`, dst).
- **Docker**: `Dockerfile` jadi 3 stage (build frontend Node → build Go → image final gabungan),
  tambah `.dockerignore`.
- **Bug fix — login admin error `cannot scan NULL into *string`**: kolom `create_by`/`update_by`
  di `tbl_admin` nullable, tapi `domain.Admin.Created`/`.Update` masih `string` biasa. Diganti ke
  `sql.NullString` (`domain/admin.go`), plus penyesuaian di `internal/service/admin.go`. Sekalian
  kefix bug kolom lama: `internal/repository/admin.go` masih pakai nama kolom
  `createadmin/createdateadmin/updateadmin/updatedateadmin` (skema lama) padahal schema baru
  pakai `create_by/create_at/update_by/update_at` — udah diselarasin.
- **Response `/api/public/transaction` disederhanakan**: awalnya balikin seluruh `TrxData`
  (detail ledger internal: `idtrx`, `notrx`, `tipe`, `source`, dst). Sekarang cuma balikin
  `invoice` (echo dari request), `username`, `balance` (saldo terbaru), dan `status: "COMPLETE"`
  (`dto.PublicTransactionData` di `dto/wallet_transaction_data.go`) — biar website game gak perlu
  paham vocab ledger internal, dan eksplisit tau transaksinya sudah final & tersimpan (bukan
  proses async yang perlu di-polling).
- **Bug fix — idempotency key salah field, bikin bet-slip kedua+ di draw yang sama silent-skip**:
  ketauan waktu debugging production (client kirim debit, respon `200 COMPLETE`, tapi saldo &
  jumlah baris di `tbl_trx_transaksi` gak berubah). Root cause: `refno` (kunci idempotency)
  awalnya diisi dari `invoice`, padahal `invoice` itu ID game/draw yang **memang sengaja sama**
  buat banyak `playerinvoice` (bet-slip) dalam draw yang sama — jadi bet-slip ke-2/3/dst buat
  username yang sama ke-anggep "udah pernah diproses" dan di-skip diam-diam (tetep balas `200`).
  Fix: `refno` sekarang diisi dari `playerinvoice` (unik per bet-slip), `invoice` cuma disimpen
  di `keterangan` sebagai metadata. Response `/api/public/transaction` juga nambah field
  `playerinvoice` (`internal/api/wallet_public.go`, `dto/wallet_transaction_data.go`).
  Diagnosa langsung dari database production (query manual pgx: histori `tbl_trx_transaksi` +
  `tbl_counter` cocok 100% sama saldo `tbl_user`, jadi kepastian bug-nya di logic idempotency,
  bukan di penyimpanan/commit).
- **Bug fix — response duplikat disamarkan jadi "sukses"**: sebelumnya kalau kena idempotency
  (lihat poin di atas), API tetap balas `200 COMPLETE` kayak transaksi baru berhasil — bikin
  susah dibedain "beneran baru diproses" vs "di-skip karena duplikat" dari sisi client.
  Sekarang kasus duplikat balas **`409`** dengan `record.status: "DUPLICATE"` (tetap bawa
  `balance` terkini). Ditambah `util.ErrDuplicateTransaction` (`internal/util/dberror.go`) dan
  helper `dto.CreateResponse` (`dto/response.go`) buat bikin body response dengan `status` yang
  konsisten sama HTTP status code aslinya (sebelumnya kalau asal pakai `CreateResponseSuccess`,
  field `status` di body ke-hardcode 200 walau HTTP code-nya bukan 200).
- **Bug fix — idempotency nge-block WIN gara-gara BET sebelumnya (playerinvoice sama)**:
  ketauan sebelum sempat kejadian di production, dari pertanyaan "kalau abis BET nanti dikirim
  WIN gimana, kan playerinvoice-nya sama?". Sebelumnya cek idempotency cuma di-scope per
  (username, refno) — jadi WIN yang pakai `playerinvoice` yang sama kayak BET sebelumnya bakal
  ke-anggep duplikat dan **saldo menangnya gak pernah masuk**. Fix: scope ditambah `source`
  (`FindByUsernameRefnoAndSource`, `domain/wallet_transaction.go` +
  `internal/repository/wallet_transaction.go`) — jadi BET dan WIN buat `playerinvoice` yang
  sama sekarang dua-duanya tetap diproses & tercatat terpisah, cuma pengulangan source yang
  sama persis yang di-anggap duplikat.
- **Bug fix — halaman Admin error 500 (`/api/admin`)**: ketauan dari testing production —
  `internal/repository/admin.go` fungsi `FindAll` pakai `SELECT *`, dan tabel `tbl_admin` di
  database production ternyata **masih punya kolom legacy** peninggalan schema lama
  (`createadmin`, `createdateadmin`, `updateadmin`, `updatedateadmin` — 4 kolom ekstra di luar
  13 kolom yang didefinisikan `sql/schema.sql`). `pgx.RowToStructByName` gagal karena kolom
  legacy itu gak punya field yang cocok di `domain.Admin`, jadi tiap buka halaman Admin selalu
  500. Fix: `FindAll` diganti pakai daftar kolom eksplisit (bukan `SELECT *`) — konsisten sama
  `FindByUsername` yang emang udah dari awal begitu. Kolom legacy di database production
  **belum dihapus** (gak masalah buat app, cuma dead weight) — kalau mau beres-beres, bisa
  `ALTER TABLE tbl_admin DROP COLUMN createadmin, createdateadmin, updateadmin, updatedateadmin;`
  kapan-kapan.
- **Fitur baru — notifikasi Telegram buat server error (500)**: lihat §3.3. Ditambah
  `internal/connection/telegram.go` (`NotifyServerError`, non-blocking via goroutine, no-op
  kalau env var kosong), config `Telegram` (`internal/config/model.go` + `loader.go`), dan
  dipasang di semua titik yang balikin 500: `Admin.Index/Save`, `Member.Index/Save`, `Login`,
  `handleTrxError` (dipakai deposit/withdraw admin + `/api/public/transaction`),
  `PublicBalance`. Sekalian kefix bug terkait: sebelumnya `Login` balikin HTTP **500** juga
  buat username/password salah (bukan cuma error server beneran) — kalau notifikasi Telegram
  langsung dipasang tanpa fix ini, tiap orang salah ketik password bakal spam notif. Ditambah
  `util.ErrInvalidCredentials` (`internal/util/dberror.go`) supaya salah login sekarang balas
  **401** (gak notif Telegram), error server beneran tetap **500** (notif). Frontend
  `Login.svelte` disesuaikan (sebelumnya cuma cek `status==500` buat nampilin pesan error).
- **Frontend — loading state pas fetch data**: halaman Admin/Member/Transaksi udah dari awal
  punya store `isLoading` (`src/lib/use*.ts`) dan di-passing ke `Home.svelte`, tapi propnya
  gak pernah dipakai (nyasar ada `let loading = $state(false)` lokal yang gak pernah di-toggle,
  cuma numpang di tombol Refresh). Sekarang `isLoading` beneran dipakai: nampilin baris
  "Memuat data..." + ikon spinner di tabel selama fetch, dan tombol Refresh ke-disable +
  ikonnya muter selama itu juga.
- **Frontend — shortcut deposit/withdraw langsung dari halaman Member**: modal deposit/withdraw
  yang tadinya cuma ada di halaman Transaksi (`transaksi/Home.svelte`) di-extract jadi komponen
  bersama `src/components/DepositWithdrawModal.svelte`. Ditambah 2 ikon (deposit/withdraw) di
  kolom SALDO halaman Member — klik langsung buka modal dengan username baris itu udah keisi &
  terkunci, jadi admin gak perlu bolak-balik ke halaman Transaksi buat top-up/potong saldo
  member tertentu. Lihat §5.
- **Fitur baru — paging halaman Transaksi (1 halaman = 500 baris, pakai dropdown select)**:
  sebelumnya `/api/transaksi` limit-nya kepatok maks 200 dan gak pernah balikin total data, jadi
  gak mungkin dibikin paging beneran. Ditambah `CountAll`/`CountByUsername`
  (`internal/repository/wallet_transaction.go`) + `CountHistory` di service, limit maks
  dinaikin ke `trxPageSize = 500`, dan response `/api/transaksi` sekarang ikut nambahin field
  `total` (lihat §3.1). Frontend: `useTransaksi.ts` nerima parameter `page`, `transaksi/Home.svelte`
  nampilin dropdown "Halaman X / Y" di bawah tabel (pakai komponen `Select` yang sama kayak di
  form Status) buat lompat ke halaman manapun tanpa reload.
- **Fitur baru — cache Redis buat riwayat transaksi**: `/api/transaksi` sebelumnya selalu query
  langsung ke Postgres (gak ke-cache). Sekarang di-cache pakai TTL **1 menit** (`trxCacheTTL`,
  `internal/service/wallet_transaction.go`), key-nya kombinasi `username:tipe:limit:offset`
  (`trxHistoryCacheKey`) buat data + `username` doang buat total count (`trxCountCacheKey`) —
  sengaja TTL-only (bukan invalidate manual tiap ada transaksi), soalnya kombinasi
  limit/offset/tipe per user bisa banyak jadi ribet kalau mesti dihapus satu-satu.
  **Kecuali** abis **Deposit/Withdraw** (aksi admin) — itu di-invalidate manual seketika
  (`invalidateTrxHistoryCache`, pakai `connection.DeleteRedisPattern` yang SCAN key
  `wallet:trx:history:<username>:*` dan `wallet:trx:history:all:*`), biar admin langsung liat
  hasilnya tanpa nunggu TTL. WinGame/PayoutGame (dari API publik, bisa high-frequency) sengaja
  TIDAK di-invalidate manual, cukup andalin TTL 1 menit biar gak sering-sering SCAN Redis.
- **TTL cache Admin & Member ikut diseragamin ke 1 menit** (`RedisAdminAllKey` tadinya 60 menit,
  `RedisWalletAllKey`/`RedisWalletDetail` tadinya 5 menit) — biar konsisten & data gak ketinggalan
  kelamaan di panel admin.
  > **Sengaja dikecualikan:** `ShowByToken` (dipanggil `/api/public/balance`) — **TIDAK PERNAH**
  > di-cache, selalu langsung query DB. Balance itu data yang paling kritis buat website game
  > eksternal (dipakai buat keputusan finansial real-time), jadi risiko kebaca stale walau cuma
  > semenit gak sepadan sama gain performa-nya. Jangan tambahin caching di sini tanpa
  > pertimbangan matang.
- **Bug fix — `Log.Fatal` tersisa di `GetRedis`/`DeleteRedis`**: (`internal/connection/redis.go`)
  masih ada 2 titik yang bisa bikin app crash total (`os.Exit`) kalau Redis error sesaat —
  sisa dari fix serupa yang udah dibenerin di `SetRedis` sebelumnya. Diganti ke `Log.Error`.
- **Fitur baru — Total Debit hari ini di halaman Dashboard**: kartu statistik "Total Debit Hari
  Ini" ditambahkan di `src/dashboard/Dashboard.svelte` (grid diubah ke `xl:grid-cols-5`).
  Datanya ambil dari `tbl_trx_transaksi` dengan `tipe = 'DEBIT'` dan `source = 'BET'` di
  rentang hari ini (Jam Jakarta) — beda sama "Total Withdraw" yang juga DEBIT tapi source
  `WITHDRAW`. Backend: query `Summary` di `internal/repository/wallet_transaction.go`
  ditambah kolom `DebitBetToday`, terus di-propagation ke `domain.TrxSummary`,
  `dto.DashboardData.TotalDebitToday` (`json: "total_debit_today"`),
  `internal/service/dashboard.go`, dan `web2026/src/lib/useDashboard.ts`. Cache Redis dashboard
  (TTL 1 jam, lihat `RedisDashboardTTL`) tetap dipakai seperti field lain.
- **Fitur baru — chart transaksi per bulan (1 tahun) di halaman Dashboard**: ditambahkan section
  "Transaksi Per Bulan" di `src/dashboard/Dashboard.svelte` — bar chart 12 bulan terakhir dengan
  2 bar per bulan: **Debit** (rose, total `tipe = 'DEBIT'` semua source) dan **Credit** (emerald,
  total `tipe = 'CREDIT'` semua source).
  Backend: repository `MonthlySummary` (`internal/repository/wallet_transaction.go`, GROUP BY
  `to_char(create_at, 'YYYY-MM')`, di-scope `create_at >= monthStart && < monthEnd`), hasil
  `[]domain.TrxMonthly`, dimapping jadi `dto.DashboardData.Chart` (`dto/dashboard_data.go`,
  field `json: "chart"`). Service `Summary` isi otomatis 12 bulan penuh — bulan tanpa transaksi
  diisi `"0.00"` biar sumbu chart selalu konsisten (lihat §3.1). Frontend: `useDashboard.ts`
  nambah tipe `DashboardMonthly` + `chart`.
- **Perubahan — chart pindah ke komponen Chart shadcn-svelte (LayerChart v2 BarChart)**:
  bar chart yang tadinya CSS/Tailwind manual diganti pakai komponen Chart bawaan shadcn-svelte
  (`npx shadcn-svelte add chart` → `src/lib/components/ui/chart/`), yang dibangun di atas
  **LayerChart v2** (`npm i -D layerchart@next`, dependency `layerchart` ditambahkan ke
  `package.json`). Di `src/dashboard/Dashboard.svelte`: `Chart.Container` + `BarChart`
  (`seriesLayout="group"`, `bandPadding`, `legend`, xAxis format `YYYY-MM` → nama bulan, tooltip
  `Chart.Tooltip`). Warna Debit/Credit didefinisikan langsung di `chartConfig` (`#f43f5e` /
  `#10b981`), bukan lewat CSS variable `--chart-*`. Data di-convert string → number dulu
  (`chartData`) karena LayerChart butuh numerik.
- **Perubahan — TTL cache dashboard diturunkan dari 24 jam ke 1 jam**: `RedisDashboardTTL` di
  `internal/service/dashboard.go` diubah dari `24 * time.Hour` ke `time.Hour` — biar kartu
  statistik dan chart per bulan gak ketinggalan data terlalu lama.
- **Perubahan — cache key dashboard di-versioning (`wallet:dashboard` → `wallet:dashboard:v2`)**:
  pas fitur chart ditambah, entry cache lama yang masih kepake (isi JSON tanpa field `chart`,
  TTL lama 24 jam) bikin `chart` kebaca `null` walau backend sudah pakai kode baru. Karena
  bentuk payload cache berubah, key-nya dinaikin versi (`RedisDashboardKey`) — entry cache
  versi lama otomatis diabaikan, jadi gak perlu flush Redis manual. Kalau suatu saat payload
  cache dashboard berubah bentuk lagi, naikin lagi versi key-nya.
- **Perubahan — cache chart dipisah ke key sendiri `wallet:dashboard:chart`**:
  cache kartu statistik (`wallet:dashboard:v2`) dan chart per bulan (`wallet:dashboard:chart`)
  sekarang terpisah (DB Redis 3, TTL 1 jam dua-duanya). Di `internal/service/dashboard.go`,
  `Summary` baca tiap bagian dari cache sendiri-sendiri — kalau salah satu kena cache dan yang
  lain miss, cuma bagian yang miss yang di-query ulang ke DB, lalu dua-duanya di-write ulang
  (lewat goroutine `SetRedis`). Pas pakai `redis-cli` buat bersihin cache dashboard, hapus
  dua-dua key-nya: `DEL wallet:dashboard:v2 wallet:dashboard:chart` (di DB 3).
- **Fitur keamanan — auto-logout kalau token expire (frontend)**: semua request
  authenticated sekarang lewat `src/lib/api.ts` (`apiFetch`), dipakai di semua hook
  (`useAdmin/useMember/useTransaksi/useDashboard`), save admin/member, dan
  `DepositWithdrawModal`. Kalau backend balas `401` (token invalid/expired, dicek JWT
  middleware di `main.go`), `apiFetch` langsung `setToken(null)` → `Root.svelte` render
  halaman Login. `$effect` logout yang tadinya duplikat di tiap halaman (`Admin/Member/
  Transaksi/Dashboard.svelte`) dan prop `HandleLogout` di 4 halaman itu dihapus (tombol
  Logout di sidebar tetap jalan). Signature hook diubah terima `token: string | null`.
- **Fitur keamanan — rate limiter global `/api` (Redis-backed)**: middleware Fiber `limiter`
  dipasang di `main.go` via `app.Use("/api", ...)` — per IP, counter disimpan di Redis
  (DB `DB_REDIS_NAME`) lewat koneksi Redis yang sama dengan cache (`connection.RDB`), jadi
  konsisten antar restart / multi instance. Default `300` request/menit, bisa diatur lewat
  `RATE_LIMIT_MAX` / `RATE_LIMIT_EXP` (lihat §4). Limit dicapai → `429 too many requests`.
  `/api/auth` tetap punya limiter lebih ketat (`20`/menit per IP, `internal/api/auth.go`).
- **Migrasi Fiber v2 → v3**: seluruh app dimigrasi ke `github.com/gofiber/fiber/v3`
  (detail sintaks di §2). JWT contrib pindah ke `github.com/gofiber/contrib/v3/jwt`
  (token diambil via `jwtware.FromContext`), `BodyParser` → `ctx.Bind().Body()`,
  `app.Static` → middleware `static.New` di `app.Use("/", ...)`, handler `*fiber.Ctx`
  → `fiber.Ctx`. Rate limiter tidak lagi pakai package storage gofiber: `limiter_storage.go`
  (package main) mengadaptasi `*redis.Client` (go-redis v9 / `connection.RDB`) langsung ke
  interface `fiber.Storage`. `go build ./...` & `go vet ./...` bersih.
- **Bug fix — form Member gagal save, error "validation failed"**: `member/Home.svelte`
  fungsi `HandleSave` ngirim field `name` di body `POST /api/member/save`, padahal
  `dto.MemberSaveRequest` di backend nunggu **`nama`** (`validate:"required"`) — jadi field
  `Nama` selalu kebaca kosong di server dan gagal validasi, walau form-nya udah diisi lengkap
  di sisi user. Diganti ke `nama: form.field_name`. (`admin/Home.svelte` aman, `dto.AdminSave`
  emang expect `name`, beda kontrak sama member.)
