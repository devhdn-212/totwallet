package domain

import (
	"context"
	"database/sql"

	"github.com/devhdn-212/totwallet/dto"

	"github.com/shopspring/decimal"
)

const (
	WalletStatusActive   = "Y"
	WalletStatusInactive = "N"
)

// Wallet merepresentasikan tbl_user: akun wallet milik member (username, saldo, dsb).
type Wallet struct {
	Username string          `db:"username"`
	Password sql.NullString  `db:"password"`
	Token    sql.NullString  `db:"token"`
	Nama     sql.NullString  `db:"nama"`
	Saldo    decimal.Decimal `db:"saldo"`
	Status   sql.NullString  `db:"status"`
	CreateBy sql.NullString  `db:"create_by"`
	CreateAt sql.NullTime    `db:"create_at"`
	UpdateBy sql.NullString  `db:"update_by"`
	UpdateAt sql.NullTime    `db:"update_at"`
}

type WalletRepository interface {
	FindAll(ctx context.Context) ([]Wallet, error)
	FindByUsername(ctx context.Context, username string) (Wallet, error)
	// FindByToken dipakai endpoint publik /api/public/balance: website game kirim token
	// milik member (kolom tbl_user.token) buat lookup username + saldo.
	FindByToken(ctx context.Context, token string) (Wallet, error)
	// FindByUsernameForUpdate mengunci baris (SELECT ... FOR UPDATE) supaya mutasi saldo
	// aman dari race condition saat ada beberapa transaksi konkuren untuk username yang sama.
	// Hanya efektif kalau dipanggil di dalam DB transaction (pgx.Tx).
	FindByUsernameForUpdate(ctx context.Context, username string) (Wallet, error)
	Save(ctx context.Context, w *Wallet) error
	Update(ctx context.Context, w *Wallet) error
	// UpdateSaldo menimpa saldo dengan nilai akhir yang sudah dihitung di service layer.
	UpdateSaldo(ctx context.Context, username string, saldo decimal.Decimal, updateBy string) error
	// CountAll dipakai dashboard admin buat total member.
	CountAll(ctx context.Context) (int, error)
}

type WalletService interface {
	Index(ctx context.Context) ([]dto.WalletData, error)
	Show(ctx context.Context, username string) (dto.WalletData, error)
	Create(ctx context.Context, req dto.CreateWalletRequest, createBy string) error
	Update(ctx context.Context, req dto.UpdateWalletRequest, updateBy string) error
	// ShowByToken dipakai API publik /api/public/balance.
	ShowByToken(ctx context.Context, token string) (dto.WalletBalanceData, error)
}
