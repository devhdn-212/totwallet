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

type walletApi struct {
	walletService domain.WalletService
}

// NewWalletApi mendaftarkan endpoint pengelolaan member (tbl_user) buat admin panel.
func NewWalletApi(app *fiber.App, walletService domain.WalletService, authmidle fiber.Handler) {
	wa := walletApi{walletService: walletService}
	member := app.Group("/api/member", authmidle)
	member.Post("", wa.Index)
	member.Post("/save", wa.Save)
}

func (wa *walletApi) Index(ctx *fiber.Ctx) error {
	c, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
	defer cancel()

	res, err := wa.walletService.Index(c)
	if err != nil {
		go connection.NotifyServerError("Member.Index", err, "")
		return ctx.Status(http.StatusInternalServerError).
			JSON(dto.CreateResponseError(http.StatusInternalServerError, "internal server error"))
	}
	return ctx.JSON(fiber.Map{
		"status":  fiber.StatusOK,
		"message": "success",
		"record":  res,
	})
}

func (wa *walletApi) Save(ctx *fiber.Ctx) error {
	c, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
	defer cancel()

	var req dto.MemberSaveRequest
	if err := ctx.BodyParser(&req); err != nil {
		connection.Log.Error("Failed to parse request body",
			zap.String("endpoint", "Save Member"),
			zap.String("body", string(ctx.Body())),
			zap.String("error", err.Error()),
		)
		return ctx.SendStatus(http.StatusUnprocessableEntity)
	}
	fails := util.Validate(req)
	if len(fails) > 0 {
		connection.Log.Warn("Validation failed for Save Member",
			zap.Any("validation_errors", fails),
			zap.Any("body", req),
		)
		return ctx.Status(http.StatusBadRequest).
			JSON(dto.CreateResponseErrorData(http.StatusBadRequest, "validation failed", fails))
	}

	datatoken := ctx.Locals("client_username").(string)
	client_username := util.Parsing_final(datatoken)

	var err error
	if req.Type == "New" {
		if req.Password == "" || len(req.Password) < 6 {
			return ctx.Status(http.StatusBadRequest).
				JSON(dto.CreateResponseError(http.StatusBadRequest, "password minimal 6 karakter"))
		}
		err = wa.walletService.Create(c, dto.CreateWalletRequest{
			Username: req.Username,
			Password: req.Password,
			Nama:     req.Nama,
		}, client_username)
	} else {
		err = wa.walletService.Update(c, dto.UpdateWalletRequest{
			Username: req.Username,
			Nama:     req.Nama,
			Status:   req.Status,
		}, client_username)
	}
	if err != nil {
		recordJson, _ := json.Marshal(req)
		connection.Log.Error("Failed to create / update member",
			zap.String("username", req.Username),
			zap.String("error", err.Error()),
			zap.String("record", string(recordJson)),
		)
		if err == util.ErrDuplicate {
			return ctx.Status(http.StatusConflict).
				JSON(dto.CreateResponseError(http.StatusConflict, "Duplicate Entry"))
		}
		if err == util.ErrNotFound {
			return ctx.Status(http.StatusNotFound).
				JSON(dto.CreateResponseError(http.StatusNotFound, "member not found"))
		}
		// Jangan sertakan req.Password mentah di notifikasi — cuma username.
		go connection.NotifyServerError("Member.Save", err, "username="+req.Username)
		return ctx.Status(http.StatusInternalServerError).
			JSON(dto.CreateResponseError(http.StatusInternalServerError, "internal server error"))
	}
	connection.Log.Info("Member create / update successfully",
		zap.String("username", req.Username),
	)
	return ctx.Status(http.StatusOK).
		JSON(dto.CreateResponseSuccess(""))
}
