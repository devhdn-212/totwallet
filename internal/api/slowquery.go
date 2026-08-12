package api

import (
	"context"
	"net/http"
	"time"

	"github.com/devhdn-212/totwallet/domain"
	"github.com/devhdn-212/totwallet/dto"
	"github.com/devhdn-212/totwallet/internal/connection"

	"github.com/gofiber/fiber/v3"
)

type slowQueryApi struct {
	service domain.SlowQueryService
}

func NewSlowQueryApi(app *fiber.App, service domain.SlowQueryService, authmidle fiber.Handler) {
	sq := slowQueryApi{service: service}
	app.Post("/api/slowquery", authmidle, sq.Index)
}

func (sq *slowQueryApi) Index(ctx fiber.Ctx) error {
	c, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
	defer cancel()

	var req dto.SlowQueryRequest
	_ = ctx.Bind().Body(&req)

	res, total, err := sq.service.List(c, req.Limit, req.Offset)
	if err != nil {
		go connection.NotifyServerError("SlowQuery.Index", err, "")
		return ctx.Status(http.StatusInternalServerError).
			JSON(dto.CreateResponseError(http.StatusInternalServerError, "internal server error"))
	}

	return ctx.JSON(fiber.Map{
		"status":  fiber.StatusOK,
		"message": "success",
		"record":  res,
		"total":   total,
	})
}
