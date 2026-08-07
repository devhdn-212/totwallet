package domain

import (
	"context"
	"database/sql"
	"time"

	"github.com/devhdn-212/totwallet/dto"

	"github.com/shopspring/decimal"
)

const (
	TrxTipeCredit = "CREDIT"
	TrxTipeDebit  = "DEBIT"

	// Nilai source mengikuti CHECK constraint tbl_trx_tipe_source_ck di sql/schema.sql:
	// CREDIT hanya boleh DEPOSIT/WIN, DEBIT hanya boleh WITHDRAW/BET.
	TrxSourceDeposit  = "DEPOSIT" // CREDIT
	TrxSourceWin      = "WIN"     // CREDIT (menang game)
	TrxSourceWithdraw = "WITHDRAW"
	TrxSourceBet      = "BET" // DEBIT, dipakai juga untuk payout/potongan game karena schema tidak punya source lain untuk DEBIT

	TrxStatusPending = "0"
	TrxStatusSuccess = "1"
	TrxStatusFailed  = "2"
)

// WalletTransaction merepresentasikan tbl_trx_transaksi: ledger mutasi saldo wallet.
type WalletTransaction struct {
	IDTrx       string          `db:"idtrx"`
	NoTrx       string          `db:"notrx"`
	Username    string          `db:"username"`
	Tipe        string          `db:"tipe"`
	Source      string          `db:"source"`
	Amount      decimal.Decimal `db:"amount"`
	SaldoBefore decimal.Decimal `db:"saldo_before"`
	SaldoAfter  decimal.Decimal `db:"saldo_after"`
	Refno       sql.NullString  `db:"refno"`
	Status      sql.NullString  `db:"status"`
	Keterangan  sql.NullString  `db:"keterangan"`
	CreateBy    sql.NullString  `db:"create_by"`
	CreateAt    sql.NullTime    `db:"create_at"`
	UpdateBy    sql.NullString  `db:"update_by"`
	UpdateAt    sql.NullTime    `db:"update_at"`
}

type WalletTransactionRepository interface {
	FindByID(ctx context.Context, idtrx string) (WalletTransaction, error)
	FindByNoTrx(ctx context.Context, notrx string) (WalletTransaction, error)
	FindByUsername(ctx context.Context, username string, limit, offset int) ([]WalletTransaction, error)
	// FindByUsernameRefnoAndSource dipakai buat idempotency: cegah refno (mis. playerinvoice
	// dari website game) diproses dobel kalau request-nya kekirim ulang (retry/duplicate).
	// Di-scope juga per source (WIN/BET/DEPOSIT/WITHDRAW) — playerinvoice yang sama BOLEH
	// dipakai buat satu BET (taruhan) dan satu WIN (menang) terpisah untuk bet-slip yang sama,
	// keduanya kejadian finansial yang beda dan harus tetap tercatat masing-masing.
	FindByUsernameRefnoAndSource(ctx context.Context, username, refno, source string) (WalletTransaction, error)
	// FindAll dipakai halaman admin buat menampilkan transaksi semua member.
	FindAll(ctx context.Context, limit, offset int) ([]WalletTransaction, error)
	// CountAll / CountByUsername dipakai buat hitung total halaman di halaman Transaksi
	// (paging pakai select, 1 halaman = 500 baris — lihat internal/service/wallet_transaction.go).
	CountAll(ctx context.Context) (int, error)
	CountByUsername(ctx context.Context, username string) (int, error)
	Save(ctx context.Context, t *WalletTransaction) error
	// Summary dipakai dashboard admin: agregat total deposit & withdraw dalam rentang
	// [start, end) plus total keseluruhan transaksi.
	Summary(ctx context.Context, start, end time.Time) (TrxSummary, error)
}

// TrxSummary adalah hasil agregat tbl_trx_transaksi buat kebutuhan dashboard admin.
type TrxSummary struct {
	DepositToday  decimal.Decimal
	WithdrawToday decimal.Decimal
	TotalTrx      int
}

type WalletTransactionService interface {
	// Deposit & WinGame -> CREDIT (nambah saldo).
	Deposit(ctx context.Context, req dto.DepositRequest, createBy string) (dto.TrxData, error)
	WinGame(ctx context.Context, req dto.WinGameRequest, createBy string) (dto.TrxData, error)
	// Withdraw & PayoutGame -> DEBIT (kurangi saldo).
	Withdraw(ctx context.Context, req dto.WithdrawRequest, createBy string) (dto.TrxData, error)
	PayoutGame(ctx context.Context, req dto.PayoutGameRequest, createBy string) (dto.TrxData, error)

	Show(ctx context.Context, idtrx string) (dto.TrxData, error)
	History(ctx context.Context, req dto.TrxHistoryRequest) ([]dto.TrxData, error)
	// CountHistory dipakai buat hitung total halaman (username kosong = hitung semua member).
	CountHistory(ctx context.Context, username string) (int, error)
}
