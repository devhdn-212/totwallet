package dto

// DashboardData adalah ringkasan statistik yang ditampilkan di halaman dashboard admin.
type DashboardData struct {
	TotalDepositToday  string `json:"total_deposit_today"`
	TotalWithdrawToday string `json:"total_withdraw_today"`
	TotalMember        int    `json:"total_member"`
	TotalTransaksi     int    `json:"total_transaksi"`
}
