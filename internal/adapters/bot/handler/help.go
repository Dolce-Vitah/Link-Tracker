package handler

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
)

const defaultHelpText = "Доступные команды:\n" +
	"/start - Начало работы с ботом\n" +
	"/help - Показать этот список команд\n" +
	"/track - Добавить ссылку на отслеживание\n" +
	"/untrack <url> - Убрать ссылку из отслеживания\n" +
	"/list [tag] - Показать отслеживаемые ссылки\n" +
	"/cancel - Отменить текущий диалог"

type HelpCommandHandler struct {
	logger   *slog.Logger
	bot      command.Sender
	helpText string
}

func NewHelpCommandHandler(logger *slog.Logger, bot command.Sender) *HelpCommandHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &HelpCommandHandler{
		logger:   logger,
		bot:      bot,
		helpText: defaultHelpText,
	}
}

func (c *HelpCommandHandler) Name() string { return "help" }
func (c *HelpCommandHandler) Description() string {
	return "Вывести список доступных команд"
}

func (c *HelpCommandHandler) Handle(ctx context.Context, request dto.CommandRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate help request: %w", err)
	}

	msg := tgbotapi.NewMessage(request.ChatID, c.helpText)
	_, err := c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send help command response: %w", err)
	}

	c.logger.Info("Help command processed", slog.Int64("chat_id", request.ChatID))
	return nil
}
