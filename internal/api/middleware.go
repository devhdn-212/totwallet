package api

import (
	"github.com/devhdn-212/totwallet/dto"

	"github.com/gofiber/fiber/v3"
)

// ApiKeyMiddleware menjaga endpoint publik (/api/public/*) yang dipanggil website game
// eksternal secara server-to-server. Wajib kirim header X-API-KEY sesuai PUBLIC_API_KEY di .env.
func ApiKeyMiddleware(apiKey string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if apiKey == "" || c.Get("X-API-KEY") != apiKey {
			return c.Status(fiber.StatusUnauthorized).
				JSON(dto.CreateResponseError(fiber.StatusUnauthorized, "invalid api key"))
		}
		return c.Next()
	}
}
