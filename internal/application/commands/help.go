package command

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log/slog"
)

type HelpCommand struct{}

func (c *HelpCommand) Name() string { return "help" }
func (c *HelpCommand) Description() string {
	return "Вывести список доступных команд"
}

type HelpHandler struct{}

func (h *HelpHandler) Handle(update *tgbotapi.Update, bot Sender) error {
	helpText := "Доступные команды:\n/start - Начало работы с ботом\n/help - Показать этот список команд"

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, helpText)
	_, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send help command response: %w", err)
	}

	slog.Info("Help command processed", slog.Int64("chat_id", update.Message.Chat.ID))
	return nil
}
