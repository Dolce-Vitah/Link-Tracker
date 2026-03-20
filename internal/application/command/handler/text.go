package handler

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
)

type textCommandHandler struct {
	logger      *slog.Logger
	bot         command.Sender
	name        string
	description string
	response    string
	logMessage  string
	errorPrefix string
}

func newTextCommandHandler(
	logger *slog.Logger,
	bot command.Sender,
	name string,
	description string,
	response string,
	logMessage string,
	errorPrefix string,
) *textCommandHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &textCommandHandler{
		logger:      logger,
		bot:         bot,
		name:        name,
		description: description,
		response:    response,
		logMessage:  logMessage,
		errorPrefix: errorPrefix,
	}
}

func (c *textCommandHandler) Name() string {
	return c.name
}

func (c *textCommandHandler) Description() string {
	return c.description
}

func (c *textCommandHandler) Handle(_ context.Context, _ string, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, c.response)
	_, err := c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("%s: %w", c.errorPrefix, err)
	}

	c.logger.Info(c.logMessage, slog.Int64("chat_id", chatID))
	return nil
}
