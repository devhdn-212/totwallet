package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/devhdn-212/totwallet/domain"
	"github.com/devhdn-212/totwallet/dto"
	"github.com/devhdn-212/totwallet/internal/connection"
	"github.com/devhdn-212/totwallet/internal/repository"
	"github.com/devhdn-212/totwallet/internal/util"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

const (
	RedisWalletTrxHistory = "wallet:trx:history:"
	RedisWalletTrxCount   = "wallet:trx:count:"
	// TTL pendek buat cache riwayat transaksi — sengaja gak di-invalidate manual pas ada
	// transaksi baru (key-nya kombinasi username+tipe+limit+offset, kebanyakan variasi kalau
	// mesti dihapus satu-satu). Paling lambat data ketinggalan segini lama sebelum refresh.
	trxCacheTTL = 1 * time.Minute
	// nama sequence Postgres untuk generate notrx, lihat sql/schema.sql
	NoTrxSequenceName = "seq_notrx_transaksi"
)

type walletTransactionService struct {
	db   *pgxpool.Pool
	repo domain.WalletTransactionRepository
}

func NewWalletTransactionService(db *pgxpool.Pool, repo domain.WalletTransactionRepository) domain.WalletTransactionService {
	return &walletTransactionService{
		db:   db,
		repo: repo,
	}
}

func toTrxData(t domain.WalletTransaction) dto.TrxData {
	var createdAt string
	if t.CreateAt.Valid {
		createdAt = t.CreateAt.Time.In(util.LocJakarta).Format("2006-01-02 15:04:05")
	}
	return dto.TrxData{
		IDTrx:       t.IDTrx,
		NoTrx:       t.NoTrx,
		Username:    t.Username,
		Tipe:        t.Tipe,
		Source:      t.Source,
		Amount:      t.Amount.StringFixed(2),
		SaldoBefore: t.SaldoBefore.StringFixed(2),
		SaldoAfter:  t.SaldoAfter.StringFixed(2),
		Refno:       util.NsToStr(t.Refno),
		Status:      util.NsToStr(t.Status),
		Keterangan:  util.NsToStr(t.Keterangan),
		CreatedAt:   createdAt,
	}
}

// process adalah inti mutasi saldo yang dipakai oleh Deposit/Withdraw/WinGame/PayoutGame:
//  1. Kunci baris wallet (SELECT ... FOR UPDATE) supaya aman dari race condition kalau ada
//     beberapa transaksi konkuren untuk username yang sama (mis. WIN & BET datang bersamaan).
//  2. Hitung saldo_after sesuai tipe (CREDIT nambah, DEBIT ngurang), tolak kalau bakal minus
//     (mirror dari CHECK constraint tbl_user_saldo_ck di schema).
//  3. Generate notrx dari tbl_counter dan catat baris ledger di tbl_trx_transaksi.
//     Semua dalam satu DB transaction supaya saldo & ledger selalu konsisten.
func (s walletTransactionService) process(ctx context.Context, username, tipe, source string, amount decimal.Decimal, refno, keterangan, createBy string) (dto.TrxData, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return dto.TrxData{}, util.ErrInvalidAmount
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return dto.TrxData{}, err
	}
	defer tx.Rollback(ctx)

	txExec := repository.NewPGXTxExecutor(tx)
	txWalletRepo := repository.NewWalletRepository(txExec)
	txTrxRepo := repository.NewWalletTransactionRepository(txExec)

	wallet, err := txWalletRepo.FindByUsernameForUpdate(ctx, username)
	if err != nil {
		return dto.TrxData{}, err
	}
	if wallet.Username == "" {
		return dto.TrxData{}, util.ErrNotFound
	}

	// Idempotency: kalau refno (mis. playerinvoice dari website game) udah pernah diproses
	// buat username + source yang SAMA (mis. BET dua kali, atau WIN dua kali), JANGAN mutasi
	// saldo lagi. Di-scope per source (bukan cuma username+refno) supaya BET dan WIN yang
	// pakai playerinvoice yang sama (satu bet-slip: kena taruhan dulu, menang belakangan)
	// tetap DUA-DUANYA tercatat & mutasi saldo masing-masing — bukan yang kedua ke-skip.
	// Balikin util.ErrDuplicateTransaction (plus data transaksi lama, buat referensi) biar
	// caller (API handler) bisa kasih response yang jelas beda dari sukses murni.
	// Aman dari race condition karena baris wallet di atas sudah dikunci (FOR UPDATE) duluan.
	if refno != "" {
		existing, err := txTrxRepo.FindByUsernameRefnoAndSource(ctx, username, refno, source)
		if err != nil {
			return dto.TrxData{}, err
		}
		if existing.IDTrx != "" {
			return toTrxData(existing), util.ErrDuplicateTransaction
		}
	}

	saldoBefore := wallet.Saldo
	var saldoAfter decimal.Decimal
	if tipe == domain.TrxTipeCredit {
		saldoAfter = saldoBefore.Add(amount)
	} else {
		saldoAfter = saldoBefore.Sub(amount)
		if saldoAfter.IsNegative() {
			return dto.TrxData{}, util.ErrInsufficientBalance
		}
	}

	if err = txWalletRepo.UpdateSaldo(ctx, username, saldoAfter, createBy); err != nil {
		return dto.TrxData{}, err
	}

	counter, err := util.NextSequenceValue(ctx, tx, NoTrxSequenceName)
	if err != nil {
		return dto.TrxData{}, err
	}
	notrx := fmt.Sprintf("TRX-%06d", counter)

	now := util.GetNowJakarta()
	trx := domain.WalletTransaction{
		IDTrx:       time.Now().Format("060102") + "-" + uuid.NewString(),
		NoTrx:       notrx,
		Username:    username,
		Tipe:        tipe,
		Source:      source,
		Amount:      amount,
		SaldoBefore: saldoBefore,
		SaldoAfter:  saldoAfter,
		Refno:       sql.NullString{Valid: refno != "", String: refno},
		Status:      sql.NullString{Valid: true, String: domain.TrxStatusSuccess},
		Keterangan:  sql.NullString{Valid: keterangan != "", String: keterangan},
		CreateBy:    sql.NullString{Valid: createBy != "", String: createBy},
		CreateAt:    sql.NullTime{Valid: true, Time: now},
	}

	if err = txTrxRepo.Save(ctx, &trx); err != nil {
		return dto.TrxData{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return dto.TrxData{}, err
	}

	go connection.DeleteRedis(RedisWalletAllKey)
	go connection.DeleteRedis(RedisWalletDetail + username)
	// Cache riwayat transaksi (RedisWalletTrxHistory/RedisWalletTrxCount) sengaja TIDAK
	// di-invalidate di sini — TTL-nya pendek (trxCacheTTL), lihat History()/CountHistory().
	// Dashboard nampilin total deposit/withdraw hari ini -> cache-nya jadi stale begitu
	// ada uang masuk (deposit/win) atau keluar (withdraw/bet), jadi harus di-invalidate juga.
	go connection.DeleteRedis(RedisDashboardKey, RedisDashboardDB)

	return toTrxData(trx), nil
}

func (s walletTransactionService) Deposit(ctx context.Context, req dto.DepositRequest, createBy string) (dto.TrxData, error) {
	res, err := s.process(ctx, req.Username, domain.TrxTipeCredit, domain.TrxSourceDeposit, req.Amount, req.Refno, req.Keterangan, createBy)
	if err == nil {
		invalidateTrxHistoryCache(req.Username)
	}
	return res, err
}

func (s walletTransactionService) Withdraw(ctx context.Context, req dto.WithdrawRequest, createBy string) (dto.TrxData, error) {
	res, err := s.process(ctx, req.Username, domain.TrxTipeDebit, domain.TrxSourceWithdraw, req.Amount, req.Refno, req.Keterangan, createBy)
	if err == nil {
		invalidateTrxHistoryCache(req.Username)
	}
	return res, err
}

// invalidateTrxHistoryCache dipanggil khusus abis Deposit/Withdraw (aksi admin, jarang/gak
// high-frequency) — biar admin langsung liat transaksinya di halaman Transaksi tanpa nunggu
// TTL. WinGame/PayoutGame (dari API publik game, bisa rame) sengaja TIDAK invalidate manual,
// cukup andalin trxCacheTTL biar gak sering-sering SCAN Redis.
func invalidateTrxHistoryCache(username string) {
	go connection.DeleteRedisPattern(RedisWalletTrxHistory + username + ":*")
	go connection.DeleteRedisPattern(RedisWalletTrxHistory + "all:*")
	go connection.DeleteRedis(trxCountCacheKey(username))
	go connection.DeleteRedis(trxCountCacheKey(""))
}

func (s walletTransactionService) WinGame(ctx context.Context, req dto.WinGameRequest, createBy string) (dto.TrxData, error) {
	return s.process(ctx, req.Username, domain.TrxTipeCredit, domain.TrxSourceWin, req.Amount, req.Refno, req.Keterangan, createBy)
}

func (s walletTransactionService) PayoutGame(ctx context.Context, req dto.PayoutGameRequest, createBy string) (dto.TrxData, error) {
	// Disimpan dengan source BET karena CHECK constraint tbl_trx_tipe_source_ck di schema
	// cuma mengizinkan pasangan DEBIT+BET (tidak ada source "PAYOUT" tersendiri).
	return s.process(ctx, req.Username, domain.TrxTipeDebit, domain.TrxSourceBet, req.Amount, req.Refno, req.Keterangan, createBy)
}

func (s walletTransactionService) Show(ctx context.Context, idtrx string) (dto.TrxData, error) {
	t, err := s.repo.FindByID(ctx, idtrx)
	if err != nil {
		return dto.TrxData{}, err
	}
	if t.IDTrx == "" {
		return dto.TrxData{}, util.ErrNotFound
	}
	return toTrxData(t), nil
}

// trxPageSize adalah ukuran 1 halaman di halaman Transaksi (paging pakai select di frontend).
const trxPageSize = 500

// trxHistoryCacheKey gabungin semua parameter yang mempengaruhi hasil query jadi satu key —
// kombinasi username+tipe+limit+offset beda hasil, jadi harus beda key juga.
func trxHistoryCacheKey(req dto.TrxHistoryRequest, limit, offset int) string {
	username := req.Username
	if username == "" {
		username = "all"
	}
	tipe := req.Tipe
	if tipe == "" {
		tipe = "all"
	}
	return fmt.Sprintf("%s%s:%s:%d:%d", RedisWalletTrxHistory, username, tipe, limit, offset)
}

func trxCountCacheKey(username string) string {
	if username == "" {
		username = "all"
	}
	return RedisWalletTrxCount + username
}

func (s walletTransactionService) History(ctx context.Context, req dto.TrxHistoryRequest) ([]dto.TrxData, error) {
	limit := req.Limit
	if limit <= 0 || limit > trxPageSize {
		limit = trxPageSize
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	cacheKey := trxHistoryCacheKey(req, limit, offset)
	if cached, found, err := connection.GetRedis(cacheKey); err == nil && found {
		var data []dto.TrxData
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
	}

	var trxs []domain.WalletTransaction
	var err error
	if req.Username == "" {
		trxs, err = s.repo.FindAll(ctx, limit, offset)
	} else {
		trxs, err = s.repo.FindByUsername(ctx, req.Username, limit, offset)
	}
	if err != nil {
		return nil, err
	}

	var result []dto.TrxData
	for _, t := range trxs {
		if req.Tipe != "" && t.Tipe != req.Tipe {
			continue
		}
		result = append(result, toTrxData(t))
	}

	go connection.SetRedis(cacheKey, result, trxCacheTTL)
	return result, nil
}

func (s walletTransactionService) CountHistory(ctx context.Context, username string) (int, error) {
	cacheKey := trxCountCacheKey(username)
	if cached, found, err := connection.GetRedis(cacheKey); err == nil && found {
		var count int
		if err := json.Unmarshal([]byte(cached), &count); err == nil {
			return count, nil
		}
	}

	count, err := s.countHistoryFromDB(ctx, username)
	if err != nil {
		return 0, err
	}

	go connection.SetRedis(cacheKey, count, trxCacheTTL)
	return count, nil
}

func (s walletTransactionService) countHistoryFromDB(ctx context.Context, username string) (int, error) {
	if username == "" {
		return s.repo.CountAll(ctx)
	}
	return s.repo.CountByUsername(ctx, username)
}
