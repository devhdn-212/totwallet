package connection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/devhdn-212/totwallet/internal/config"
)

var telegramConf config.Telegram

// SetTelegramConfig dipanggil dari main.go sekali di awal (isi TELEGRAM_BOT_TOKEN /
// TELEGRAM_CHAT_ID di .env). Kalau kosong, NotifyServerError jadi no-op (gak nge-block
// startup app kalau notifikasi belum di-setup).
func SetTelegramConfig(conf config.Telegram) {
	telegramConf = conf
}

// NotifyServerError kirim notifikasi ke Telegram khusus buat server error (500) — bukan
// buat error bisnis biasa (validasi/saldo kurang/dsb) biar gak spam. SELALU dipanggil lewat
// goroutine (go connection.NotifyServerError(...)) oleh caller supaya gak nge-block response
// ke client, dan sengaja gak pernah bikin app crash walau gagal kirim (cuma di-log).
func NotifyServerError(endpoint string, err error, detail string) {
	if telegramConf.BotToken == "" || telegramConf.ChatID == "" {
		return
	}

	text := fmt.Sprintf("🚨 Wallet API — Server Error\nEndpoint: %s\nError: %s", endpoint, err.Error())
	if detail != "" {
		text += "\n" + detail
	}

	payload, marshalErr := json.Marshal(map[string]any{
		"chat_id": telegramConf.ChatID,
		"text":    text,
	})
	if marshalErr != nil {
		Log.Error("Failed to build Telegram payload", zap.Error(marshalErr))
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramConf.BotToken)
	client := http.Client{Timeout: 5 * time.Second}

	resp, reqErr := client.Post(url, "application/json", bytes.NewReader(payload))
	if reqErr != nil {
		Log.Error("Failed to send Telegram notification", zap.Error(reqErr))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		Log.Error("Telegram notification rejected", zap.Int("status", resp.StatusCode))
	}
}
