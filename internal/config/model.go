package config

type Config struct {
	Server   Server
	Database Database
	Jwt      Jwt
	Redis    Redis
	Public   Public
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
