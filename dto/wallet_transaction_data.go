package dto

import "github.com/shopspring/decimal"

type TrxData struct {
	IDTrx       string `json:"idtrx"`
	NoTrx       string `json:"notrx"`
	Username    string `json:"username"`
	Tipe        string `json:"tipe"`
	Source      string `json:"source"`
	Amount      string `json:"amount"`
	SaldoBefore string `json:"saldo_before"`
	SaldoAfter  string `json:"saldo_after"`
	Refno       string `json:"refno"`
	Status      string `json:"status"`
	Keterangan  string `json:"keterangan"`
	CreatedAt   string `json:"created_datetime"`
}

// DepositRequest -> CREDIT / DEPOSIT
type DepositRequest struct {
	Username   string          `json:"username" validate:"required"`
	Amount     decimal.Decimal `json:"amount" validate:"required"`
	Refno      string          `json:"refno"`
	Keterangan string          `json:"keterangan"`
}

// WithdrawRequest -> DEBIT / WITHDRAW
type WithdrawRequest struct {
	Username   string          `json:"username" validate:"required"`
	Amount     decimal.Decimal `json:"amount" validate:"required"`
	Refno      string          `json:"refno"`
	Keterangan string          `json:"keterangan"`
}

// WinGameRequest -> CREDIT / WIN
type WinGameRequest struct {
	Username   string          `json:"username" validate:"required"`
	Amount     decimal.Decimal `json:"amount" validate:"required"`
	Refno      string          `json:"refno" validate:"required"` // referensi round/game
	Keterangan string          `json:"keterangan"`
}

// PayoutGameRequest -> DEBIT / BET (potongan taruhan / payout game)
type PayoutGameRequest struct {
	Username   string          `json:"username" validate:"required"`
	Amount     decimal.Decimal `json:"amount" validate:"required"`
	Refno      string          `json:"refno" validate:"required"` // referensi round/game
	Keterangan string          `json:"keterangan"`
}

type TrxHistoryRequest struct {
	// Username kosong = tampilkan transaksi semua member (dipakai halaman admin).
	Username string `query:"username"`
	Tipe     string `query:"tipe"`
	Limit    int    `query:"limit"`
	Offset   int    `query:"offset"`
}

// PublicTransactionRequest adalah kontrak API publik POST /api/public/transaction
// yang dipanggil website game eksternal buat lapor hasil game (menang/kalah).
// Persis satu dari Credit/Debit yang harus > 0:
//   - Credit > 0 -> dicatat sebagai CREDIT/WIN (member menang)
//   - Debit  > 0 -> dicatat sebagai DEBIT/BET  (member kalah / bayar taruhan)
type PublicTransactionRequest struct {
	Invoice       string          `json:"invoice" validate:"required"`
	Pasaran       string          `json:"pasaran" validate:"required"`
	PlayerInvoice string          `json:"playerinvoice" validate:"required"`
	Username      string          `json:"username" validate:"required"`
	Credit        decimal.Decimal `json:"credit"`
	Debit         decimal.Decimal `json:"debit"`
}

// PublicBalanceRequest adalah kontrak API publik POST /api/public/balance yang
// dipanggil website game buat cek username + saldo terkini lewat token member.
type PublicBalanceRequest struct {
	Token string `json:"token" validate:"required"`
}

// PublicTransactionData adalah response API publik POST /api/public/transaction.
// Sengaja cuma balikin invoice+playerinvoice (buat website game cocokkan ke request
// mereka), username, dan balance terbaru — bukan detail ledger internal
// (idtrx/notrx/tipe/source).
//
// Endpoint ini synchronous: begitu response ini balik dengan Status "COMPLETE",
// transaksi sudah tersimpan & saldo sudah ter-update di database (bukan proses
// background/async) — website pemanggil tidak perlu polling/nunggu status lagi.
type PublicTransactionData struct {
	Invoice       string `json:"invoice"`
	PlayerInvoice string `json:"playerinvoice"`
	Username      string `json:"username"`
	Balance       string `json:"balance"`
	Status        string `json:"status"`
}

const (
	PublicTrxStatusComplete = "COMPLETE"
	// PublicTrxStatusDuplicate dipakai kalau playerinvoice (refno) udah pernah diproses
	// sebelumnya buat username yang sama — request ini TIDAK memproses apapun yang baru.
	PublicTrxStatusDuplicate = "DUPLICATE"
)
