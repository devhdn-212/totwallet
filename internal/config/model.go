package config

type Config struct {
	Server   Server
	Database Database
	Jwt      Jwt
	Redis    Redis
	Public   Public
	Telegram Telegram
	Limiter  Limiter
}

type Server struct {
	Host string
	Port string
}
type Jwt struct {
	Key      string
	Exp      int
	Issuer   string
	Audience string
}
type Database struct {
	Host   string
	Port   string
	Name   string
	Schema string
	User   string
	Pass   string
	Tz     string
}

type Redis struct {
	Host string
	Port string
	Pass string
	Name string
}

// Public menampung kredensial buat API publik yang diakses website game eksternal
// (endpoint credit/debit saldo & cek balance), lihat internal/api/wallet_public.go
type Public struct {
	ApiKey string
}

// Telegram dipakai buat kirim notifikasi kalau ada server error (500) — lihat
// internal/connection/telegram.go
type Telegram struct {
	BotToken string
	ChatID   string
}

// Limiter mengatur rate limit global endpoint /api (per IP, disimpan di Redis supaya
// konsisten antar restart/multi instance). Stricter limit di /api/auth tetap di luar ini.
type Limiter struct {
	Max  int // maks request per window
	Exp  int // window dalam menit
}
