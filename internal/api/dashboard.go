package api

import (
	"context"
	"net/http"
	"time"

	"github.com/devhdn-212/totwallet/domain"
	"github.com/devhdn-212/totwallet/dto"

	"github.com/gofiber/fiber/v3"
)

type dashboardApi struct {
	service domain.DashboardService
}

func NewDashboardApi(app *fiber.App, service domain.DashboardService, authmidle fiber.Handler) {
	da := dashboardApi{service: service}
	app.Post("/api/dashboard", authmidle, da.Summary)
}

func (da *dashboardApi) Summary(ctx fiber.Ctx) error {
	c, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
	defer cancel()

	res, err := da.service.Summary(c)
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
