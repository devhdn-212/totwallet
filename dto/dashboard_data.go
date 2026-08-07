package dto

// DashboardData adalah ringkasan statistik yang ditampilkan di halaman dashboard admin.
type DashboardData struct {
	TotalDepositToday  string            `json:"total_deposit_today"`
	TotalWithdrawToday string            `json:"total_withdraw_today"`
	TotalDebitToday    string            `json:"total_debit_today"`
	TotalMember        int               `json:"total_member"`
	TotalTransaksi     int               `json:"total_transaksi"`
	Chart              []DashboardMonthly `json:"chart"`
}

// DashboardMonthly adalah agregat debit & credit per bulan (bulan format "YYYY-MM")
// buat chart per bulan di halaman dashboard.
type DashboardMonthly struct {
	Bulan  string `json:"bulan"`
	Debit  string `json:"debit"`
	Credit string `json:"credit"`
}
