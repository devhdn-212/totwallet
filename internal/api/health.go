package api

import (
	"github.com/devhdn-212/totwallet/dto"

	"github.com/gofiber/fiber/v3"
)

// NewHealthCheckApi mendaftarkan endpoint health check publik (tanpa JWT). Dipanggil
// frontend pas pertama kali load buat menangkap IP asli pengguna sebelum login — IP itu
// ikut dikirim di body request login (/api/auth, field `ipaddress`) buat di-simpan ke
// tbl_admin.ipaddress.
func NewHealthCheckApi(app *fiber.App) {
	app.Get("/api/health", HealthCheck)
}

// HealthCheck balikin IP asli client. Di belakang proxy/Cloud Run, c.IP() bisa berubah jadi
// IP internal proxy — makanya dicek dulu c.IPs() (header X-Forwarded-For), fallback ke c.IP().
func HealthCheck(c fiber.Ctx) error {
	realip := "0.0.0.0"
	if ips := c.IPs(); len(ips) > 0 {
		realip = ips[0]
	}
	if realip == "0.0.0.0" {
		realip = c.IP()
	}

	// Anti-cache biar IP yang didapat selalu fresh (request berikutnya gak balikin yang lama).
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
	c.Set("Surrogate-Control", "no-store")

	return c.Status(fiber.StatusOK).JSON(dto.CreateResponseSuccess(dto.HealthCheckData{
		RealIP:      realip,
		ContainerIP: c.IP(),
		IPList:      c.IPs(),
	}))
}
