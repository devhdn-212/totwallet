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
	// RedisDashboardKey :v2 — versi key dinaikin pas payload cache berubah (nambah field chart),
	// biar entry cache lama (tanpa chart) otomatis di-skip tanpa harus flush manual.
	RedisDashboardKey = "wallet:dashboard:v2"
	// RedisDashboardChartKey cache chart per bulan dipisah dari kartu statistik — biar pas
	// salah satu berubah (mis. chart ke-invalidate), yang satunya gak ikut kena.
	RedisDashboardChartKey = "wallet:dashboard:chart"
	// RedisDashboardDB pakai DB Redis terpisah (3) dari cache lain supaya gampang di-flush sendiri.
	RedisDashboardDB  = 3
	RedisDashboardTTL = time.Hour
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
	now := util.GetNowJakarta()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, util.LocJakarta)
	end := start.Add(24 * time.Hour)

	// Chart per bulan: 12 bulan terakhir (termasuk bulan berjalan), tiap bulan total
	// DEBIT (semua source) vs CREDIT (semua source). Bulan tanpa transaksi tetap diisi 0.
	monthStart := time.Date(now.Year(), now.Month()-11, 1, 0, 0, 0, 0, util.LocJakarta)
	monthEnd := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, util.LocJakarta)

	// Kartu statistik & chart di-cache terpisah (key beda, TTL sama 1 jam) — tiap bagian
	// dibaca dari cache sendiri-sendiri, kalau satu kena cache yang lain miss, yang miss
	// doang yang di-query ulang ke DB.
	result := dto.DashboardData{Chart: []dto.DashboardMonthly{}}
	summaryCached := false
	chartCached := false

	if cached, found, err := connection.GetRedis(RedisDashboardKey, RedisDashboardDB); err == nil && found {
		var data dto.DashboardData
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			result.TotalDepositToday = data.TotalDepositToday
			result.TotalWithdrawToday = data.TotalWithdrawToday
			result.TotalDebitToday = data.TotalDebitToday
			result.TotalMember = data.TotalMember
			result.TotalTransaksi = data.TotalTransaksi
			summaryCached = true
		}
	}

	if cached, found, err := connection.GetRedis(RedisDashboardChartKey, RedisDashboardDB); err == nil && found {
		var chart []dto.DashboardMonthly
		if err := json.Unmarshal([]byte(cached), &chart); err == nil {
			result.Chart = chart
			chartCached = true
		}
	}

	if summaryCached && chartCached {
		connection.Log.Info("Returning data from Redis - Dashboard")
		return result, nil
	}

	if !summaryCached {
		trxSummary, err := s.trxRepo.Summary(ctx, start, end)
		if err != nil {
			return dto.DashboardData{}, err
		}

		totalMember, err := s.walletRepo.CountAll(ctx)
		if err != nil {
			return dto.DashboardData{}, err
		}

		result.TotalDepositToday = trxSummary.DepositToday.StringFixed(2)
		result.TotalWithdrawToday = trxSummary.WithdrawToday.StringFixed(2)
		result.TotalDebitToday = trxSummary.DebitBetToday.StringFixed(2)
		result.TotalMember = totalMember
		result.TotalTransaksi = trxSummary.TotalTrx
	}

	if !chartCached {
		monthly, err := s.trxRepo.MonthlySummary(ctx, monthStart, monthEnd)
		if err != nil {
			return dto.DashboardData{}, err
		}
		result.Chart = buildDashboardChart(monthly, monthStart)
	}

	go connection.SetRedis(RedisDashboardKey, result, RedisDashboardTTL, RedisDashboardDB)
	go connection.SetRedis(RedisDashboardChartKey, result.Chart, RedisDashboardTTL, RedisDashboardDB)
	connection.Log.Info("Returning data Database - Dashboard")
	return result, nil
}

// buildDashboardChart menyusun array 12 bulan (dari monthStart) dan isi bulan tanpa
// transaksi dengan 0 supaya sumbu chart selalu konsisten.
func buildDashboardChart(monthly []domain.TrxMonthly, monthStart time.Time) []dto.DashboardMonthly {
	byMonth := make(map[string]dto.DashboardMonthly, len(monthly))
	for _, m := range monthly {
		byMonth[m.Month] = dto.DashboardMonthly{
			Bulan:  m.Month,
			Debit:  m.Debit.StringFixed(2),
			Credit: m.Credit.StringFixed(2),
		}
	}

	chart := make([]dto.DashboardMonthly, 0, 12)
	for i := 0; i < 12; i++ {
		key := monthStart.AddDate(0, i, 0).Format("2006-01")
		if v, ok := byMonth[key]; ok {
			chart = append(chart, v)
		} else {
			chart = append(chart, dto.DashboardMonthly{Bulan: key, Debit: "0.00", Credit: "0.00"})
		}
	}
	return chart
}
