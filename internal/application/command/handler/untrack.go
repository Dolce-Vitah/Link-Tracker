package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type UntrackCommandHandler struct {
	tracker tracker.Service
	logger  *slog.Logger
}

func NewUntrackCommandHandler(trackerService tracker.Service, logger *slog.Logger) *UntrackCommandHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &UntrackCommandHandler{tracker: trackerService, logger: logger}
}

func (c *UntrackCommandHandler) Name() string { return "untrack" }
func (c *UntrackCommandHandler) Description() string {
	return "Прекратить отслеживание ссылки"
}

func (c *UntrackCommandHandler) Handle(ctx context.Context, update *tgbotapi.Update, bot command.Sender) error {
	chatID := update.Message.Chat.ID
	args := strings.TrimSpace(update.Message.CommandArguments())
	if !isValidUntrackURL(args) {
		msg := tgbotapi.NewMessage(chatID, "Укажите ссылку после команды: /untrack https://example.com")
		_, err := bot.Send(msg)
		return err
	}

	if err := c.tracker.RegisterChat(ctx, chatID); err != nil && !strings.Contains(err.Error(), tracker.ErrAlreadyExists.Error()) {
		return fmt.Errorf("register chat before untrack: %w", err)
	}

	_, err := c.tracker.RemoveLink(ctx, chatID, api.RemoveLinkRequest{Link: args})
	if err != nil {
		if strings.Contains(err.Error(), tracker.ErrNotFound.Error()) {
			msg := tgbotapi.NewMessage(chatID, "Ссылка не найдена в отслеживаемых.")
			_, sendErr := bot.Send(msg)
			return sendErr
		}
		return fmt.Errorf("remove link from tracking: %w", err)
	}

	msg := tgbotapi.NewMessage(chatID, "Ссылка удалена из отслеживания.")
	_, sendErr := bot.Send(msg)
	if sendErr != nil {
		return sendErr
	}

	c.logger.Info("Untrack command processed", slog.Int64("chat_id", chatID), slog.String("url", args))
	return nil
}

func isValidUntrackURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
