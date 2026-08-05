package repository

import (
	"context"
	"errors"
	"time"

	"github.com/devhdn-212/totwallet/domain"
	"github.com/devhdn-212/totwallet/internal/config"

	"github.com/jackc/pgx/v5"
)

type walletTransactionRepository struct {
	db DBExecutor
}

func NewWalletTransactionRepository(db DBExecutor) domain.WalletTransactionRepository {
	return &walletTransactionRepository{
		db: db,
	}
}

const walletTrxColumns = `idtrx, notrx, username, tipe, source, amount, saldo_before, saldo_after, refno, status, keterangan, create_by, create_at, update_by, update_at`

func (r walletTransactionRepository) FindByID(ctx context.Context, idtrx string) (domain.WalletTransaction, error) {
	query := `SELECT ` + walletTrxColumns + ` FROM ` + config.DB_tbl_trx_transaksi + ` WHERE idtrx = $1 LIMIT 1`

	rows, err := r.db.Query(ctx, query, idtrx)
	if err != nil {
		return domain.WalletTransaction{}, err
	}

	t, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.WalletTransaction])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.WalletTransaction{}, nil
		}
		return domain.WalletTransaction{}, err
	}
	return t, nil
}

func (r walletTransactionRepository) FindByNoTrx(ctx context.Context, notrx string) (domain.WalletTransaction, error) {
	query := `SELECT ` + walletTrxColumns + ` FROM ` + config.DB_tbl_trx_transaksi + ` WHERE notrx = $1 LIMIT 1`

	rows, err := r.db.Query(ctx, query, notrx)
	if err != nil {
		return domain.WalletTransaction{}, err
	}

	t, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.WalletTransaction])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.WalletTransaction{}, nil
		}
		return domain.WalletTransaction{}, err
	}
	return t, nil
}

func (r walletTransactionRepository) FindByUsername(ctx context.Context, username string, limit, offset int) ([]domain.WalletTransaction, error) {
	query := `SELECT ` + walletTrxColumns + ` FROM ` + config.DB_tbl_trx_transaksi + `
	          WHERE username = $1 ORDER BY create_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, username, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.WalletTransaction])
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r walletTransactionRepository) FindByUsernameRefnoAndSource(ctx context.Context, username, refno, source string) (domain.WalletTransaction, error) {
	query := `SELECT ` + walletTrxColumns + ` FROM ` + config.DB_tbl_trx_transaksi + `
	          WHERE username = $1 AND refno = $2 AND source = $3 LIMIT 1`

	rows, err := r.db.Query(ctx, query, username, refno, source)
	if err != nil {
		return domain.WalletTransaction{}, err
	}

	t, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.WalletTransaction])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.WalletTransaction{}, nil
		}
		return domain.WalletTransaction{}, err
	}
	return t, nil
}

func (r walletTransactionRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.WalletTransaction, error) {
	query := `SELECT ` + walletTrxColumns + ` FROM ` + config.DB_tbl_trx_transaksi + `
	          ORDER BY create_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.WalletTransaction])
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Summary menghitung agregat deposit/withdraw dalam rentang [start, end) plus total
// keseluruhan transaksi, dipakai buat dashboard admin.
func (r walletTransactionRepository) Summary(ctx context.Context, start, end time.Time) (domain.TrxSummary, error) {
	query := `SELECT
	            COALESCE(SUM(amount) FILTER (WHERE tipe = 'CREDIT' AND source = 'DEPOSIT' AND create_at >= $1 AND create_at < $2), 0),
	            COALESCE(SUM(amount) FILTER (WHERE tipe = 'DEBIT' AND source = 'WITHDRAW' AND create_at >= $1 AND create_at < $2), 0),
	            COUNT(*)
	          FROM ` + config.DB_tbl_trx_transaksi

	var s domain.TrxSummary
	err := r.db.QueryRow(ctx, query, start, end).Scan(&s.DepositToday, &s.WithdrawToday, &s.TotalTrx)
	if err != nil {
		return domain.TrxSummary{}, err
	}
	return s, nil
}

func (r walletTransactionRepository) Save(ctx context.Context, t *domain.WalletTransaction) error {
	query := `INSERT INTO ` + config.DB_tbl_trx_transaksi + `
	          (idtrx, notrx, username, tipe, source, amount, saldo_before, saldo_after, refno, status, keterangan, create_by, create_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.db.Exec(ctx, query,
		t.IDTrx,
		t.NoTrx,
		t.Username,
		t.Tipe,
		t.Source,
		t.Amount,
		t.SaldoBefore,
		t.SaldoAfter,
		t.Refno,
		t.Status,
		t.Keterangan,
		t.CreateBy,
		t.CreateAt,
	)
	return err
}
