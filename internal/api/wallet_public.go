package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/devhdn-212/totwallet/domain"
	"github.com/devhdn-212/totwallet/dto"
	"github.com/devhdn-212/totwallet/internal/connection"
	"github.com/devhdn-212/totwallet/internal/util"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const publicApiCreateBy = "public-api"

type walletPublicApi struct {
	walletService domain.WalletService
	trxService    domain.WalletTransactionService
}

// NewWalletPublicApi mendaftarkan endpoint publik yang dipanggil server-to-server oleh
// website game eksternal (bukan browser admin), makanya diamankan pakai ApiKeyMiddleware
// (header X-API-KEY) alih-alih JWT admin.
func NewWalletPublicApi(app *fiber.App, walletService domain.WalletService, trxService domain.WalletTransactionService, apiKey string) {
	wp := walletPublicApi{walletService: walletService, trxService: trxService}
	public := app.Group("/api/public", ApiKeyMiddleware(apiKey))
	public.Post("/transaction", wp.Transaction)
	public.Post("/balance", wp.Balance)
}

// Transaction menerima laporan hasil game dari website eksternal. Field wajib:
// invoice, pasaran, playerinvoice, username, credit, debit — persis satu dari
// credit/debit yang harus > 0 (credit = member menang/WIN, debit = taruhan/BET).
func (wp *walletPublicApi) Transaction(ctx *fiber.Ctx) error {
	c, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
	defer cancel()

	var req dto.PublicTransactionRequest
	if err := ctx.BodyParser(&req); err != nil {
		connection.Log.Error("Failed to parse request body",
			zap.String("endpoint", "PublicTransaction"),
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

	creditPositive := req.Credit.GreaterThan(decimal.Zero)
	debitPositive := req.Debit.GreaterThan(decimal.Zero)
	if creditPositive == debitPositive {
		return ctx.Status(http.StatusBadRequest).
			JSON(dto.CreateResponseError(http.StatusBadRequest, "exactly one of credit or debit must be greater than zero"))
	}

	keterangan := fmt.Sprintf("pasaran=%s;playerinvoice=%s", req.Pasaran, req.PlayerInvoice)

	var res dto.TrxData
	var err error
	if creditPositive {
		res, err = wp.trxService.WinGame(c, dto.WinGameRequest{
			Username:   req.Username,
			Amount:     req.Credit,
			Refno:      req.Invoice,
			Keterangan: keterangan,
		}, publicApiCreateBy)
	} else {
		res, err = wp.trxService.PayoutGame(c, dto.PayoutGameRequest{
			Username:   req.Username,
			Amount:     req.Debit,
			Refno:      req.Invoice,
			Keterangan: keterangan,
		}, publicApiCreateBy)
	}
	if err != nil {
		return handleTrxError(ctx, err, "PublicTransaction", req)
	}
	return ctx.Status(http.StatusOK).JSON(dto.CreateResponseSuccess(dto.PublicTransactionData{
		Invoice:  req.Invoice,
		Username: res.Username,
		Balance:  res.SaldoAfter,
		Status:   dto.PublicTrxStatusComplete,
	}))
}

// Balance mengembalikan username + saldo terkini milik member berdasarkan token
// (tbl_user.token) yang dikirim website game.
func (wp *walletPublicApi) Balance(ctx *fiber.Ctx) error {
	c, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
	defer cancel()

	var req dto.PublicBalanceRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.SendStatus(http.StatusUnprocessableEntity)
	}
	fails := util.Validate(req)
	if len(fails) > 0 {
		return ctx.Status(http.StatusBadRequest).
			JSON(dto.CreateResponseErrorData(http.StatusBadRequest, "validation failed", fails))
	}

	res, err := wp.walletService.ShowByToken(c, req.Token)
	if err != nil {
		if err == util.ErrNotFound {
			return ctx.Status(http.StatusNotFound).
				JSON(dto.CreateResponseError(http.StatusNotFound, "member not found"))
		}
		return ctx.Status(http.StatusInternalServerError).
			JSON(dto.CreateResponseError(http.StatusInternalServerError, "internal server error"))
	}
	return ctx.Status(http.StatusOK).JSON(dto.CreateResponseSuccess(res))
}
