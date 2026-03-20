package handler

import (
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
)

const defaultStartText = "Добро пожаловать! Используйте /help, чтобы узнать о доступных командах."

type StartCommandHandler struct {
	*textCommandHandler
}

func NewStartCommandHandler(logger *slog.Logger, bot command.Sender) *StartCommandHandler {
	return &StartCommandHandler{
		textCommandHandler: newTextCommandHandler(
			logger,
			bot,
			"start",
			"Начать работу с ботом",
			defaultStartText,
			"Start command processed",
			"send start command response",
		),
	}
}
