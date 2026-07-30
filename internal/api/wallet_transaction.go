package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/devhdn-212/totwallet/domain"
	"github.com/devhdn-212/totwallet/dto"
	"github.com/devhdn-212/totwallet/internal/connection"
	"github.com/devhdn-212/totwallet/internal/util"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type walletTransactionApi struct {
	trxService domain.WalletTransactionService
}

// NewWalletTransactionApi mendaftarkan endpoint transaksi (deposit/withdraw/riwayat)
// buat admin panel. Ini beda dengan /api/public/transaction yang dipanggil website game.
func NewWalletTransactionApi(app *fiber.App, trxService domain.WalletTransactionService, authmidle fiber.Handler) {
	wa := walletTransactionApi{trxService: trxService}
	trx := app.Group("/api/transaksi", authmidle)
	trx.Post("", wa.Index)
	trx.Post("/deposit", wa.Deposit)
	trx.Post("/withdraw", wa.Withdraw)
}

func (wa *walletTransactionApi) Index(ctx *fiber.Ctx) error {
	c, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
	defer cancel()

	var req dto.TrxHistoryRequest
	_ = ctx.BodyParser(&req)

	res, err := wa.trxService.History(c, req)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).
			JSON(dto.CreateResponseError(http.StatusInternalServerError, "internal server error"))
	}
	return ctx.JSON(fiber.Map{
		"status":  fiber.StatusOK,
		"message": "success",
		"record":  res,
	})
}

func (wa *walletTransactionApi) Deposit(ctx *fiber.Ctx) error {
	c, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
	defer cancel()

	var req dto.DepositRequest
	if err := ctx.BodyParser(&req); err != nil {
		connection.Log.Error("Failed to parse request body",
			zap.String("endpoint", "Deposit"),
			zap.String("body", string(ctx.Body())),
			zap.String("error", err.Error()),
		)
		return ctx.SendStatus(http.StatusUnprocessableEntity)
	}
	fails := util.Validate(req)
	if len(fails) > 0 {
		return ctx.Status(http.StatusBadRequest).
			JSON(dto.CreateResponseErrorData(http.StatusBadRequest, "validation failed", fails))
	}

	datatoken := ctx.Locals("client_username").(string)
	client_username := util.Parsing_final(datatoken)

	res, err := wa.trxService.Deposit(c, req, client_username)
	if err != nil {
		return handleTrxError(ctx, err, "Deposit", req)
	}
	return ctx.Status(http.StatusOK).JSON(dto.CreateResponseSuccess(res))
}

func (wa *walletTransactionApi) Withdraw(ctx *fiber.Ctx) error {
	c, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
	defer cancel()

	var req dto.WithdrawRequest
	if err := ctx.BodyParser(&req); err != nil {
		connection.Log.Error("Failed to parse request body",
			zap.String("endpoint", "Withdraw"),
			zap.String("body", string(ctx.Body())),
			zap.String("error", err.Error()),
		)
		return ctx.SendStatus(http.StatusUnprocessableEntity)
	}
	fails := util.Validate(req)
	if len(fails) > 0 {
		return ctx.Status(http.StatusBadRequest).
			JSON(dto.CreateResponseErrorData(http.StatusBadRequest, "validation failed", fails))
	}

	datatoken := ctx.Locals("client_username").(string)
	client_username := util.Parsing_final(datatoken)

	res, err := wa.trxService.Withdraw(c, req, client_username)
	if err != nil {
		return handleTrxError(ctx, err, "Withdraw", req)
	}
	return ctx.Status(http.StatusOK).JSON(dto.CreateResponseSuccess(res))
}

// handleTrxError menyeragamkan mapping error service transaksi -> HTTP status,
// dipakai baik oleh endpoint admin (deposit/withdraw) maupun API publik.
func handleTrxError(ctx *fiber.Ctx, err error, endpoint string, req any) error {
	recordJson, _ := json.Marshal(req)
	connection.Log.Error("Failed to process transaction",
		zap.String("endpoint", endpoint),
		zap.String("error", err.Error()),
		zap.String("record", string(recordJson)),
	)
	switch err {
	case util.ErrNotFound:
		return ctx.Status(http.StatusNotFound).
			JSON(dto.CreateResponseError(http.StatusNotFound, "member not found"))
	case util.ErrInsufficientBalance:
		return ctx.Status(http.StatusBadRequest).
			JSON(dto.CreateResponseError(http.StatusBadRequest, "insufficient balance"))
	case util.ErrInvalidAmount:
		return ctx.Status(http.StatusBadRequest).
			JSON(dto.CreateResponseError(http.StatusBadRequest, "amount must be greater than zero"))
	case util.ErrDuplicateTransaction:
		return ctx.Status(http.StatusConflict).
			JSON(dto.CreateResponseError(http.StatusConflict, "duplicate transaction: refno already processed"))
	default:
		go connection.NotifyServerError(endpoint, err, string(recordJson))
		return ctx.Status(http.StatusInternalServerError).
			JSON(dto.CreateResponseError(http.StatusInternalServerError, "internal server error"))
	}
}
