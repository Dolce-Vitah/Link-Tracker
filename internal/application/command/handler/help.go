package handler

import (
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
)

const defaultHelpText = "Доступные команды:\n/start - Начало работы с ботом\n/help - Показать этот список команд"

type HelpCommandHandler struct {
	*textCommandHandler
}

func NewHelpCommandHandler(logger *slog.Logger, bot command.Sender) *HelpCommandHandler {
	return &HelpCommandHandler{
		textCommandHandler: newTextCommandHandler(
			logger,
			bot,
			"help",
			"Вывести список доступных команд",
			defaultHelpText,
			"Help command processed",
			"send help command response",
		),
	}
}
