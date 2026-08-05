package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/devhdn-212/totwallet/domain"
	"github.com/devhdn-212/totwallet/dto"
	"github.com/devhdn-212/totwallet/internal/connection"
	"github.com/devhdn-212/totwallet/internal/util"
)

const (
	RedisDashboardKey = "wallet:dashboard"
	// RedisDashboardDB pakai DB Redis terpisah (3) dari cache lain supaya gampang di-flush sendiri.
	RedisDashboardDB  = 3
	RedisDashboardTTL = 24 * time.Hour
)

type dashboardService struct {
	walletRepo domain.WalletRepository
	trxRepo    domain.WalletTransactionRepository
}

func NewDashboardService(walletRepo domain.WalletRepository, trxRepo domain.WalletTransactionRepository) domain.DashboardService {
	return &dashboardService{
		walletRepo: walletRepo,
		trxRepo:    trxRepo,
	}
}

func (s dashboardService) Summary(ctx context.Context) (dto.DashboardData, error) {
	cached, found, err := connection.GetRedis(RedisDashboardKey, RedisDashboardDB)
	if err == nil && found {
		var data dto.DashboardData
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			connection.Log.Info("Returning data from Redis - Dashboard")
			return data, nil
		}
	}

	now := util.GetNowJakarta()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, util.LocJakarta)
	end := start.Add(24 * time.Hour)

	trxSummary, err := s.trxRepo.Summary(ctx, start, end)
	if err != nil {
		return dto.DashboardData{}, err
	}

	totalMember, err := s.walletRepo.CountAll(ctx)
	if err != nil {
		return dto.DashboardData{}, err
	}

	result := dto.DashboardData{
		TotalDepositToday:  trxSummary.DepositToday.StringFixed(2),
		TotalWithdrawToday: trxSummary.WithdrawToday.StringFixed(2),
		TotalMember:        totalMember,
		TotalTransaksi:     trxSummary.TotalTrx,
	}

	go connection.SetRedis(RedisDashboardKey, result, RedisDashboardTTL, RedisDashboardDB)
	connection.Log.Info("Returning data Database - Dashboard")
	return result, nil
}
