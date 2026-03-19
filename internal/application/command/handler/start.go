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
	bot       command.Sender
	startText string
	tracker   tracker.Service
}

func NewStartCommandHandler(trackerService tracker.Service, logger *slog.Logger, bot command.Sender) *StartCommandHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &StartCommandHandler{
		logger:    logger,
		bot:       bot,
		startText: defaultStartText,
		tracker:   trackerService,
	}
}

func (c *StartCommandHandler) Name() string        { return "start" }
func (c *StartCommandHandler) Description() string { return "Начать работу с ботом" }

func (c *StartCommandHandler) Handle(ctx context.Context, text string, chatID int64) error {
	if c.tracker != nil {
		err := c.tracker.RegisterChat(ctx, chatID)
		if err != nil && !strings.Contains(err.Error(), tracker.ErrAlreadyExists.Error()) {
			return fmt.Errorf("register chat: %w", err)
		}
	}

	msg := tgbotapi.NewMessage(chatID, c.startText)
	_, err := c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send start command response: %w", err)
	}

	c.logger.Info("Start command processed", slog.Int64("chat_id", chatID))
	return nil
}
