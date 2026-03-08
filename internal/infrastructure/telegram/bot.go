package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/handler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dialog"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	dispatcher  *command.Dispatcher
	logger      *slog.Logger
	sessions    *dialog.Store
	tracker     tracker.Service
	sendMessage func(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

func NewBot(token string, logger *slog.Logger) (*Bot, error) {
	if logger == nil {
		logger = slog.Default()
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("new telegram bot api: %w", err)
	}

	return &Bot{
		api:         api,
		dispatcher:  command.NewDispatcher(),
		logger:      logger,
		sessions:    dialog.NewStore(),
		sendMessage: api.Send,
	}, nil
}

func (b *Bot) RegisterCommand(cmd command.Command) {
	b.dispatcher.Register(cmd)
}

func (b *Bot) Sessions() *dialog.Store {
	return b.sessions
}

func (b *Bot) SetTrackerService(service tracker.Service) {
	b.tracker = service
}

func (b *Bot) SetMyCommands() error {
	commands := b.dispatcher.Commands()
	tgCommands := make([]tgbotapi.BotCommand, 0, len(commands))
	for _, cmd := range commands {
		tgCommands = append(tgCommands, tgbotapi.BotCommand{
			Command:     cmd.Name(),
			Description: cmd.Description(),
		})
	}

	config := tgbotapi.NewSetMyCommands(tgCommands...)
	_, err := b.api.Request(config)
	if err != nil {
		return fmt.Errorf("set telegram commands menu: %w", err)
	}

	return nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		b.handleUpdate(context.Background(), &update, b.api)
	}
}

func (b *Bot) StartHTTPServer(address string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/updates", b.handleLinkUpdateHTTP)

	slog.Info("Starting bot HTTP server", slog.String("address", address))
	return http.ListenAndServe(address, mux)
}

func (b *Bot) handleUpdate(ctx context.Context, update *tgbotapi.Update, sender command.Sender) {
	if b.tracker != nil {
		if err := handler.HandleTrackDialogStep(ctx, b.tracker, b.sessions, update, sender, b.logger); err != nil {
			slog.Error("Failed to process track dialog step", slog.String("error", err.Error()))
			return
		}
	}

	err := b.dispatcher.Dispatch(ctx, update, sender)
	if err == nil {
		return
	}

	if errors.Is(err, command.ErrUnknownCommand) && update.Message != nil && update.Message.IsCommand() {
		var (
			cmdName  = update.Message.Command()
			chatID   = update.Message.Chat.ID
			username string
		)
		if update.Message.From != nil {
			username = update.Message.From.UserName
		}
		msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.")
		_, sendErr := sender.Send(msg)
		if sendErr != nil {
			b.logger.Error("Failed to send unknown command warning", slog.String("error", sendErr.Error()))
		}

		b.logger.Warn("Unknown command received",
			slog.String("command", cmdName),
			slog.Int64("chat_id", chatID),
			slog.String("username", username),
		)
		return
	}

	var (
		cmdName string
		chatID  int64
	)
	if update.Message != nil {
		chatID = update.Message.Chat.ID
		if update.Message.IsCommand() {
			cmdName = update.Message.Command()
		}
	}

	b.logger.Error("Failed to handle command",
		slog.String("command", cmdName),
		slog.String("error", err.Error()),
		slog.Int64("chat_id", chatID),
	)
}

func (b *Bot) handleLinkUpdateHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var update api.LinkUpdate
	if err := json.Unmarshal(body, &update); err != nil {
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
		if _, err := b.sendMessage(msg); err != nil {
			slog.Error("Failed to send update message",
				slog.String("error", err.Error()),
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
	_ = json.NewEncoder(w).Encode(api.ApiErrorResponse{
		Description: description,
		Code:        fmt.Sprintf("%d", status),
	})
}
