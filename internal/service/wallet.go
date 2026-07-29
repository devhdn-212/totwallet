package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/devhdn-212/totwallet/domain"
	"github.com/devhdn-212/totwallet/dto"
	"github.com/devhdn-212/totwallet/internal/connection"
	"github.com/devhdn-212/totwallet/internal/repository"
	"github.com/devhdn-212/totwallet/internal/util"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

const (
	RedisWalletAllKey = "wallet:all"
	RedisWalletDetail = "wallet:detail:"
)

type walletService struct {
	db   *pgxpool.Pool
	repo domain.WalletRepository
}

func NewWalletService(db *pgxpool.Pool, repo domain.WalletRepository) domain.WalletService {
	return &walletService{
		db:   db,
		repo: repo,
	}
}

func toWalletData(w domain.Wallet) dto.WalletData {
	var createdAt, updatedAt string
	if w.CreateAt.Valid {
		createdAt = w.CreateAt.Time.In(util.LocJakarta).Format("2006-01-02 15:04:05")
	}
	if w.UpdateAt.Valid {
		updatedAt = w.UpdateAt.Time.In(util.LocJakarta).Format("2006-01-02 15:04:05")
	}
	return dto.WalletData{
		Username:  w.Username,
		Token:     util.NsToStr(w.Token),
		Nama:      util.NsToStr(w.Nama),
		Saldo:     w.Saldo.StringFixed(2),
		Status:    util.NsToStr(w.Status),
		CreatedAt: createdAt,
		UpdateAt:  updatedAt,
	}
}

func (s walletService) Index(ctx context.Context) ([]dto.WalletData, error) {
	cached, found, err := connection.GetRedis(RedisWalletAllKey)
	if err != nil {
		return nil, err
	}
	if found {
		var data []dto.WalletData
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			connection.Log.Info("Returning data from Redis - Wallet")
			return data, nil
		}
	}

	wallets, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	var result []dto.WalletData
	for _, w := range wallets {
		result = append(result, toWalletData(w))
	}

	go connection.SetRedis(RedisWalletAllKey, result, 5*time.Minute)
	connection.Log.Info("Returning data Database - Wallet")
	return result, nil
}

func (s walletService) Show(ctx context.Context, username string) (dto.WalletData, error) {
	redisDetail := RedisWalletDetail + username

	cached, found, err := connection.GetRedis(redisDetail)
	if err == nil && found {
		var data dto.WalletData
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			connection.Log.Info("Returning data from Redis - Wallet/" + username)
			return data, nil
		}
	}

	w, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return dto.WalletData{}, err
	}
	if w.Username == "" {
		return dto.WalletData{}, util.ErrNotFound
	}

	result := toWalletData(w)
	_ = connection.SetRedis(redisDetail, result, 5*time.Minute)
	connection.Log.Info("Returning data Database - Wallet/" + username)
	return result, nil
}

func (s walletService) ShowByToken(ctx context.Context, token string) (dto.WalletBalanceData, error) {
	w, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return dto.WalletBalanceData{}, err
	}
	if w.Username == "" {
		return dto.WalletBalanceData{}, util.ErrNotFound
	}
	return dto.WalletBalanceData{
		Username: w.Username,
		Balance:  w.Saldo.StringFixed(2),
	}, nil
}

func (s walletService) Create(ctx context.Context, req dto.CreateWalletRequest, createBy string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	txExec := repository.NewPGXTxExecutor(tx)
	txRepo := repository.NewWalletRepository(txExec)

	flag, err := txRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return err
	}
	if flag.Username != "" {
		return util.ErrDuplicate
	}

	hashed, err := util.HashPassword(req.Password)
	if err != nil {
		return err
	}

	now := util.GetNowJakarta()
	wallet := domain.Wallet{
		Username: req.Username,
		Password: sql.NullString{Valid: true, String: hashed},
		// Token adalah kredensial lookup buat API publik /api/public/balance
		// (website game kirim token ini buat cek username + saldo).
		Token:    sql.NullString{Valid: true, String: uuid.NewString()},
		Nama:     sql.NullString{Valid: true, String: req.Nama},
		Saldo:    decimal.Zero,
		Status:   sql.NullString{Valid: true, String: domain.WalletStatusActive},
		CreateBy: sql.NullString{Valid: createBy != "", String: createBy},
		CreateAt: sql.NullTime{Valid: true, Time: now},
	}

	if err = txRepo.Save(ctx, &wallet); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return util.ErrDuplicate
		}
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	go connection.DeleteRedis(RedisWalletAllKey)
	return nil
}

func (s walletService) Update(ctx context.Context, req dto.UpdateWalletRequest, updateBy string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	txExec := repository.NewPGXTxExecutor(tx)
	txRepo := repository.NewWalletRepository(txExec)

	flag, err := txRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return err
	}
	if flag.Username == "" {
		return util.ErrNotFound
	}

	flag.Nama = sql.NullString{Valid: true, String: req.Nama}
	flag.Status = sql.NullString{Valid: true, String: req.Status}
	flag.UpdateBy = sql.NullString{Valid: true, String: updateBy}
	flag.UpdateAt = sql.NullTime{Valid: true, Time: util.GetNowJakarta()}

	if err = txRepo.Update(ctx, &flag); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	go connection.DeleteRedis(RedisWalletAllKey)
	go connection.DeleteRedis(RedisWalletDetail + req.Username)
	return nil
}
