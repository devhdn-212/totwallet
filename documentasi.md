# Wallet API — Dokumentasi

Aplikasi wallet: admin panel (Svelte 5, `web2026/`) buat kelola akun admin, member, dan
transaksi (deposit/withdraw), plus API publik yang bisa dipanggil website game eksternal
buat lapor menang/kalah dan cek saldo member.

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

## 2. Arsitektur Backend (Go / Fiber v2)

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

### Kenapa Fiber v2, bukan v3?

Seluruh app (`main.go`, semua `internal/api/*.go`) sudah dibangun di atas Fiber **v2**.
Fiber v3 sempat ditambahkan (`go get github.com/gofiber/fiber/v3`) sesuai permintaan, tapi
**belum dipakai untuk wiring apapun** — mencampur `*fiber.App` v2 dan v3 dalam satu proses
tidak bisa (beda tipe, beda middleware API). v3 ada di `go.mod` siap dipakai kalau nanti
mau migrasi endpoint baru atau seluruh app.

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
4. Generate `notrx` dari `tbl_counter` (lihat `util.GetNextCounterManualTx`).
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
| POST | `/api/transaksi` | Riwayat transaksi (`username` kosong = semua member) |
| POST | `/api/transaksi/deposit` | Deposit saldo member (CREDIT/DEPOSIT) |
| POST | `/api/transaksi/withdraw` | Withdraw saldo member (DEBIT/WITHDRAW) |

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

## 4. Environment Variables

Lihat [`.env.example`](.env.example). Tambahan buat fitur ini:

| Variable | Fungsi |
|---|---|
| `DB_REDIS_NAME` | Index Redis DB (project ini pakai **DB 2**, bukan default 0) |
| `PUBLIC_API_KEY` | Secret yang harus dikirim website game lewat header `X-API-KEY` ke `/api/public/*` |

## 5. Frontend (`web2026/`, Svelte 5 + Vite + Tailwind)

| Halaman | File | Fungsi |
|---|---|---|
| Login | `src/Login.svelte` | Login admin (username + password) |
| Admin | `src/admin/` | List + create/edit akun admin |
| Member | `src/member/` | List + create/edit akun member (saldo, status, lihat token) |
| Transaksi | `src/transaksi/` | Deposit/withdraw + riwayat transaksi semua member |

Setiap fitur punya 2 file: `<Fitur>.svelte` (ambil data lewat hook `src/lib/use<Fitur>.ts`,
cek token) dan `Home.svelte` (tampilan tabel + modal form, submit langsung ke API).

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
  ngikutin pola CREDIT (Deposit/Win) & DEBIT (Withdraw/Payout) sesuai schema. `fiber/v3` di-`go get`
  tapi belum dipasang (app tetap Fiber v2 — lihat §2).
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
