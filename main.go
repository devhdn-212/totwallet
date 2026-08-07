package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devhdn-212/totwallet/dto"
	"github.com/devhdn-212/totwallet/internal/api"
	"github.com/devhdn-212/totwallet/internal/config"
	"github.com/devhdn-212/totwallet/internal/connection"
	"github.com/devhdn-212/totwallet/internal/repository"
	"github.com/devhdn-212/totwallet/internal/service"

	jwtMid "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	logger := NewGCPLogger()

	connection.SetLogger(logger)

	cnf := config.Get()
	connection.SetTelegramConfig(cnf.Telegram)
	// 3. Koneksi Database (pgxpool)
	dbPool := connection.GetDatabase(cnf.Database)
	defer dbPool.Close()

	// 4. Inisialisasi & Health Check Redis
	if err := connection.InitRedis(cnf.Redis); err != nil {
		logger.Fatal("Failed to init Redis", zap.Error(err))
	}
	defer connection.RDB.Close()

	if !connection.RedisHealth() {
		logger.Fatal("Redis is not healthy")
	}

	app := fiber.New()
	app.Use(requestid.New())
	app.Use(etag.New())
	app.Use(func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)
		rid := requestid.FromContext(c)
		fields := []zap.Field{
			zap.String("request_id", rid),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Int64("latency_ms", latency.Microseconds()),
			zap.String("ip", c.IP()),
		}
		// Optional: POST JSON (HATI-HATI DI PROD)
		if c.Method() == fiber.MethodPost && c.Is("json") {
			if len(c.Body()) < 2048 { // guard
				var parsed map[string]interface{}
				if err := json.Unmarshal(c.Body(), &parsed); err == nil {
					delete(parsed, "password")
					delete(parsed, "token")
					//capture user input
					//fields = append(fields, zap.Any("json_body", parsed))
				}
			}
		}

		// log
		logger.Info("http_request", fields...)
		return err
	})

	// 5. Rate limit global endpoint /api — per IP, counter di Redis (DB yang sama dengan
	// cache) supaya limit konsisten antar restart/multi instance. `/api/auth` punya limiter
	// sendiri yang lebih ketat (lihat internal/api/auth.go). Storage pakai koneksi Redis
	// yang sama dengan cache (connection.RDB) via adapter go-redis, bukan storage gofiber.
	app.Use("/api", limiter.New(limiter.Config{
		Max:        cnf.Limiter.Max,
		Expiration: time.Duration(cnf.Limiter.Exp) * time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				JSON(dto.CreateResponseError(fiber.StatusTooManyRequests, "too many requests"))
		},
		Storage: NewRedisStorage(connection.RDB),
	}))

	// Serve static frontend build (Svelte). Di Fiber v3 app.Static dipindah jadi middleware static.
	app.Use("/", static.New("./web2026/dist", static.Config{
		Compress:   true,
		ByteRange:  true,
		Browse:     false,
		IndexNames: []string{"index.html"},
	}))

	jwtMidd := jwtMid.New(jwtMid.Config{
		SigningKey: jwtMid.SigningKey{Key: []byte(cnf.Jwt.Key), JWTAlg: "HS256"},
		SuccessHandler: func(c fiber.Ctx) error {
			token := jwtMid.FromContext(c)
			if token == nil {
				return c.Status(fiber.StatusUnauthorized).
					JSON(dto.CreateResponseError(fiber.StatusUnauthorized, "invalid token"))
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.Status(fiber.StatusUnauthorized).
					JSON(dto.CreateResponseError(fiber.StatusUnauthorized, "invalid token"))
			}
			username, ok := claims["clien_admin"].(string)
			c.Locals("client_username", username)
			if !ok || username == "" {
				return c.Status(fiber.StatusUnauthorized).
					JSON(dto.CreateResponseError(fiber.StatusUnauthorized, "invalid token - Username"))
			}
			jti, ok := claims["jti"].(string)
			if !ok || jti == "" {
				return c.Status(fiber.StatusUnauthorized).
					JSON(dto.CreateResponseError(fiber.StatusUnauthorized, "invalid token"))
			}
			isBlacklisted, err := connection.IsJWTBlacklisted(jti)
			if err != nil || isBlacklisted {
				return c.Status(fiber.StatusUnauthorized).
					JSON(dto.CreateResponseError(fiber.StatusUnauthorized, "invalid token"))
			}
			if !validateJwtClaims(claims, cnf.Jwt.Issuer, cnf.Jwt.Audience) {
				return c.Status(fiber.StatusUnauthorized).
					JSON(dto.CreateResponseError(fiber.StatusUnauthorized, "invalid token"))
			}
			return c.Next()
		},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).
				JSON(dto.CreateResponseError(fiber.StatusUnauthorized, "missing token, please login"))
		},
	})
	pgxExec := repository.NewPGXExecutor(dbPool)
	adminRepository := repository.NewAdminRepository(pgxExec)
	walletRepository := repository.NewWalletRepository(pgxExec)
	walletTrxRepository := repository.NewWalletTransactionRepository(pgxExec)

	adminService := service.NewAdminService(dbPool, adminRepository)
	authService := service.NewAuth(dbPool, cnf, adminRepository)
	walletService := service.NewWalletService(dbPool, walletRepository)
	walletTrxService := service.NewWalletTransactionService(dbPool, walletTrxRepository)
	dashboardService := service.NewDashboardService(walletRepository, walletTrxRepository)

	api.NewAdminApi(app, adminService, jwtMidd)
	api.NewAuth(app, authService, jwtMidd)
	api.NewWalletApi(app, walletService, jwtMidd)
	api.NewWalletTransactionApi(app, walletTrxService, jwtMidd)
	api.NewWalletPublicApi(app, walletService, walletTrxService, cnf.Public.ApiKey)
	api.NewDashboardApi(app, dashboardService, jwtMidd)

	go func() {
		appsPort := cnf.Server.Port
		if err := app.Listen(":" + appsPort); err != nil {
			logger.Fatal("Failed to start app", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	logger.Info("Gracefully shutting down...")
	_ = app.Shutdown()

	logger.Info("Running cleanup tasks...")

	// Your cleanup tasks go here
	dbPool.Close()
	// Berikan timeout untuk shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Shutdown complete")
}

func validateJwtClaims(claims jwt.MapClaims, issuer, audience string) bool {
	if issuer != "" {
		iss, ok := claims["iss"].(string)
		if !ok || iss != issuer {
			return false
		}
	}
	if audience != "" {
		switch aud := claims["aud"].(type) {
		case string:
			if aud != audience {
				return false
			}
		case []interface{}:
			found := false
			for _, v := range aud {
				if s, ok := v.(string); ok && s == audience {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		case []string:
			found := false
			for _, s := range aud {
				if s == audience {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func NewGCPLogger() *zap.Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		loc, _ := time.LoadLocation("Asia/Jakarta")
		enc.AppendString(t.In(loc).Format("2006-01-02 15:04:05"))
	}
	encoderConfig.LevelKey = "severity"
	encoderConfig.MessageKey = "message"

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.Lock(zapcore.AddSync(os.Stdout)),
		zap.InfoLevel,
	)

	return zap.New(core, zap.AddCaller())
}

// redisStorage mengadaptasi *redis.Client (github.com/redis/go-redis/v9) yang sudah
// dipakai app lewat connection.RDB ke interface fiber.Storage milik middleware limiter.
// Dipakai supaya rate limit & cache berbagi koneksi Redis yang sama, tanpa harus
// tergantung package storage dari gofiber.
type redisStorage struct {
	client *redis.Client
}

// NewRedisStorage membungkus client redis milik app sebagai fiber.Storage.
// Client TIDAK akan ditutup oleh storage ini — umurnya dikelola connection.RDB.
func NewRedisStorage(client *redis.Client) *redisStorage {
	return &redisStorage{client: client}
}

func (s *redisStorage) Get(key string) ([]byte, error) {
	return s.GetWithContext(context.Background(), key)
}

func (s *redisStorage) GetWithContext(ctx context.Context, key string) ([]byte, error) {
	if len(key) == 0 {
		return nil, nil
	}
	val, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

func (s *redisStorage) Set(key string, val []byte, exp time.Duration) error {
	return s.SetWithContext(context.Background(), key, val, exp)
}

func (s *redisStorage) SetWithContext(ctx context.Context, key string, val []byte, exp time.Duration) error {
	if len(key) == 0 || len(val) == 0 {
		return nil
	}
	return s.client.Set(ctx, key, val, exp).Err()
}

func (s *redisStorage) Delete(key string) error {
	return s.DeleteWithContext(context.Background(), key)
}

func (s *redisStorage) DeleteWithContext(ctx context.Context, key string) error {
	if len(key) == 0 {
		return nil
	}
	return s.client.Del(ctx, key).Err()
}

func (s *redisStorage) Reset() error {
	return s.ResetWithContext(context.Background())
}

func (s *redisStorage) ResetWithContext(ctx context.Context) error {
	return s.client.FlushDB(ctx).Err()
}

// Close no-op: client global (connection.RDB) dikelola di tempat lain, jangan ditutup di sini.
func (s *redisStorage) Close() error {
	return nil
}
