package telegram

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/handlerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/linkdto"
)

func (b *Bot) StartHTTPServer(address string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /updates", b.handleLinkUpdateHTTP)
	mux.HandleFunc("POST /updates/failures", b.handleProcessingFailuresHTTP)

	ln, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("start bot http server: %w", err)
	}

	slog.Info("Starting bot HTTP server", slog.String("address", ln.Addr().String()))

	go func() {
		if serveErr := http.Serve(ln, mux); serveErr != nil {
			slog.Error("Bot HTTP server stopped", slog.String("error", serveErr.Error()))
		}
	}()

	return nil
}

func (b *Bot) handleLinkUpdateHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	var update linkdto.LinkUpdate

	decodeErr := json.NewDecoder(r.Body).Decode(&update)
	if decodeErr != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request schema")

		return
	}

	validateErr := update.Validate()

	if validateErr != nil {
		writeAPIError(w, http.StatusBadRequest, validateErr.Error())

		return
	}

	if err := b.ProcessLinkUpdate(update); err != nil {
		slog.Error("Failed to process update message in HTTP handler", slog.String("error", err.Error()))
	}

	w.WriteHeader(http.StatusOK)
}

func (b *Bot) handleProcessingFailuresHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	var report linkdto.ProcessingFailureReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request schema")
		return
	}
	if err := report.Validate(); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	text := formatFailureReportMessage(report.URLs, report.Detail)
	msg := tgbotapi.NewMessage(report.TgChatID, text)
	if _, sendErr := b.sendMessage(msg); sendErr != nil {
		slog.Error("Failed to send failure report",
			slog.String("error", sendErr.Error()),
			slog.Int64("chat_id", report.TgChatID),
		)
	}

	w.WriteHeader(http.StatusOK)
}

func (b *Bot) ProcessLinkUpdate(update linkdto.LinkUpdate) error {
	if err := update.Validate(); err != nil {
		return fmt.Errorf("validate link update: %w", err)
	}

	text := formatLinkUpdateMessage(update)
	for _, chatID := range update.TgChatIDs {
		msg := tgbotapi.NewMessage(chatID, text)
		if _, sendErr := b.sendMessage(msg); sendErr != nil {
			return fmt.Errorf("send update message to chat %d: %w", chatID, sendErr)
		}
	}

	return nil
}

func writeAPIError(w http.ResponseWriter, status int, description string) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(handlerapi.ErrorResponse{
		Description: description,
		Code:        strconv.Itoa(status),
	})
}
