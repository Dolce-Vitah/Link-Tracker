package handler

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
)

const defaultStartText = "Добро пожаловать! Используйте /help, чтобы узнать о доступных командах."

type StartCommand struct {
	logger    *slog.Logger
	startText string
}

func NewStartCommand(logger *slog.Logger) *StartCommand {
	if logger == nil {
		logger = slog.Default()
	}

	return &StartCommand{
		logger:    logger,
		startText: defaultStartText,
	}
}

func (c *StartCommand) Name() string        { return "start" }
func (c *StartCommand) Description() string { return "Начать работу с ботом" }

func (c *StartCommand) Handle(ctx context.Context, update *tgbotapi.Update, bot command.Sender) error {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, c.startText)
	_, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send start command response: %w", err)
	}

	c.logger.Info("Start command processed", slog.Int64("chat_id", update.Message.Chat.ID))
	return nil
}
