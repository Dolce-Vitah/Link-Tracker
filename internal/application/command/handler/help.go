package handler

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
)

const defaultHelpText = "Доступные команды:\n/start - Начало работы с ботом\n/help - Показать этот список команд"

type HelpCommand struct {
	logger   *slog.Logger
	helpText string
}

func NewHelpCommand(logger *slog.Logger) *HelpCommand {
	if logger == nil {
		logger = slog.Default()
	}

	return &HelpCommand{
		logger:   logger,
		helpText: defaultHelpText,
	}
}

func (c *HelpCommand) Name() string { return "help" }
func (c *HelpCommand) Description() string {
	return "Вывести список доступных команд"
}

func (c *HelpCommand) Handle(ctx context.Context, update *tgbotapi.Update, bot command.Sender) error {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, c.helpText)
	_, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send help command response: %w", err)
	}

	c.logger.Info("Help command processed", slog.Int64("chat_id", update.Message.Chat.ID))
	return nil
}
