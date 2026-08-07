package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/devhdn-212/totwallet/dto"
	"github.com/devhdn-212/totwallet/internal/api"
	"github.com/devhdn-212/totwallet/internal/config"
	"github.com/devhdn-212/totwallet/internal/connection"
	"github.com/devhdn-212/totwallet/internal/repository"
	"github.com/devhdn-212/totwallet/internal/service"

	jwtMid "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	storageRedis "github.com/gofiber/storage/redis/v3"
	"github.com/golang-jwt/jwt/v5"
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
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)
		rid, _ := c.Locals("requestid").(string)
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
	// sendiri yang lebih ketat (lihat internal/api/auth.go).
	app.Use("/api", limiter.New(limiter.Config{
		Max:        cnf.Limiter.Max,
		Expiration: time.Duration(cnf.Limiter.Exp) * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				JSON(dto.CreateResponseError(fiber.StatusTooManyRequests, "too many requests"))
		},
		Storage: storageRedis.New(storageRedis.Config{
			Host:     cnf.Redis.Host,
			Port:     redisPort(cnf.Redis.Port),
			Password: cnf.Redis.Pass,
			Database: redisDB(cnf.Redis.Name),
		}),
	}))

	app.Static("/", "./web2026/dist", fiber.Static{
		Compress:  true,
		ByteRange: true,
		Browse:    false,
		Index:     "index.html",
	})

	jwtMidd := jwtMid.New(jwtMid.Config{
		SigningKey: jwtMid.SigningKey{Key: []byte(cnf.Jwt.Key), JWTAlg: "HS256"},
		SuccessHandler: func(c *fiber.Ctx) error {
			token, ok := c.Locals("user").(*jwt.Token)
			if !ok || token == nil {
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
		ErrorHandler: func(c *fiber.Ctx, err error) error {
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

func redisPort(port string) int {
	p, err := strconv.Atoi(port)
	if err != nil || p <= 0 {
		return 6379
	}
	return p
}

func redisDB(db string) int {
	d, err := strconv.Atoi(db)
	if err != nil || d < 0 {
		return 0
	}
	return d
}
