package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
)

const defaultStartText = "Добро пожаловать! Используйте /help, чтобы узнать о доступных командах."

type StartCommandHandler struct {
	logger    *slog.Logger
	startText string
	tracker   tracker.Service
}

func NewStartCommandHandler(trackerService tracker.Service, logger *slog.Logger) *StartCommandHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &StartCommandHandler{
		logger:    logger,
		startText: defaultStartText,
		tracker:   trackerService,
	}
}

func (c *StartCommandHandler) Name() string        { return "start" }
func (c *StartCommandHandler) Description() string { return "Начать работу с ботом" }

func (c *StartCommandHandler) Handle(ctx context.Context, update *tgbotapi.Update, bot command.Sender) error {
	if c.tracker != nil {
		err := c.tracker.RegisterChat(ctx, update.Message.Chat.ID)
		if err != nil && !strings.Contains(err.Error(), tracker.ErrAlreadyExists.Error()) {
			return fmt.Errorf("register chat: %w", err)
		}
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, c.startText)
	_, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send start command response: %w", err)
	}

	c.logger.Info("Start command processed", slog.Int64("chat_id", update.Message.Chat.ID))
	return nil
}
