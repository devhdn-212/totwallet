package repository

import (
	"context"
	"errors"

	"github.com/devhdn-212/totwallet/domain"
	"github.com/devhdn-212/totwallet/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

type walletRepository struct {
	db DBExecutor
}

func NewWalletRepository(db DBExecutor) domain.WalletRepository {
	return &walletRepository{
		db: db,
	}
}

const walletColumns = `username, password, token, nama, saldo, status, create_by, create_at, update_by, update_at`

func (r walletRepository) FindAll(ctx context.Context) ([]domain.Wallet, error) {
	query := `SELECT ` + walletColumns + ` FROM ` + config.DB_tbl_user + ` ORDER BY create_at DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Wallet])
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r walletRepository) FindByUsername(ctx context.Context, username string) (domain.Wallet, error) {
	query := `SELECT ` + walletColumns + ` FROM ` + config.DB_tbl_user + ` WHERE username = $1 LIMIT 1`

	rows, err := r.db.Query(ctx, query, username)
	if err != nil {
		return domain.Wallet{}, err
	}

	w, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.Wallet])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Wallet{}, nil
		}
		return domain.Wallet{}, err
	}
	return w, nil
}

func (r walletRepository) FindByToken(ctx context.Context, token string) (domain.Wallet, error) {
	query := `SELECT ` + walletColumns + ` FROM ` + config.DB_tbl_user + ` WHERE token = $1 LIMIT 1`

	rows, err := r.db.Query(ctx, query, token)
	if err != nil {
		return domain.Wallet{}, err
	}

	w, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.Wallet])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Wallet{}, nil
		}
		return domain.Wallet{}, err
	}
	return w, nil
}

func (r walletRepository) FindByUsernameForUpdate(ctx context.Context, username string) (domain.Wallet, error) {
	query := `SELECT ` + walletColumns + ` FROM ` + config.DB_tbl_user + ` WHERE username = $1 LIMIT 1 FOR UPDATE`

	rows, err := r.db.Query(ctx, query, username)
	if err != nil {
		return domain.Wallet{}, err
	}

	w, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.Wallet])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Wallet{}, nil
		}
		return domain.Wallet{}, err
	}
	return w, nil
}

func (r walletRepository) Save(ctx context.Context, w *domain.Wallet) error {
	query := `INSERT INTO ` + config.DB_tbl_user + `
	          (username, password, token, nama, saldo, status, create_by, create_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.Exec(ctx, query,
		w.Username,
		w.Password,
		w.Token,
		w.Nama,
		w.Saldo,
		w.Status,
		w.CreateBy,
		w.CreateAt,
	)
	return err
}

func (r walletRepository) Update(ctx context.Context, w *domain.Wallet) error {
	query := `UPDATE ` + config.DB_tbl_user + ` SET
	          nama = $1, status = $2, update_by = $3, update_at = $4
	          WHERE username = $5`

	res, err := r.db.Exec(ctx, query,
		w.Nama,
		w.Status,
		w.UpdateBy,
		w.UpdateAt,
		w.Username,
	)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r walletRepository) UpdateSaldo(ctx context.Context, username string, saldo decimal.Decimal, updateBy string) error {
	query := `UPDATE ` + config.DB_tbl_user + ` SET saldo = $1, update_by = $2, update_at = now() WHERE username = $3`

	res, err := r.db.Exec(ctx, query, saldo, updateBy, username)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
