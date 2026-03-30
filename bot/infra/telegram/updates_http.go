package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

func (b *Bot) StartHTTPServer(address string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /updates", b.handleLinkUpdateHTTP)

	slog.Info("Starting bot HTTP server", slog.String("address", address))

	err := http.ListenAndServe(address, mux)
	if err != nil {

		return fmt.Errorf("start bot http server: %w", err)
	}

	return nil
}

func (b *Bot) handleLinkUpdateHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "failed to read body")

		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	var update api.LinkUpdate
	unmarshalErr := json.Unmarshal(body, &update)
	if unmarshalErr != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request schema")

		return
	}

	if update.URL == "" || len(update.TgChatIDs) == 0 {
		writeAPIError(w, http.StatusBadRequest, "url and tgChatIds are required")

		return
	}

	for _, chatID := range update.TgChatIDs {
		text := fmt.Sprintf("Обнаружено обновление по ссылке %s", update.URL)
		if strings.TrimSpace(update.Description) != "" {
			text = update.Description
		}
		msg := tgbotapi.NewMessage(chatID, text)
		if _, sendErr := b.sendMessage(msg); sendErr != nil {
			slog.Error("Failed to send update message",
				slog.String("error", sendErr.Error()),
				slog.Int64("chat_id", chatID),
				slog.String("url", update.URL),
			)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func writeAPIError(w http.ResponseWriter, status int, description string) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(api.ErrorResponse{
		Description: description,
		Code:        strconv.Itoa(status),
	})
}
