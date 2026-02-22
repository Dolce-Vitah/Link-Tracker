package commands

import (
	"log/slog"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HelpCommand struct {}

func (c *HelpCommand) Name() string { return "help" }
func (c *HelpCommand) Description() string { return "Вывести список доступных команд" }

func (c *HelpCommand) Handle(update *tgbotapi.Update, bot Sender) error {
	helpText := "Доступные команды:\n/start - Начало работы с ботом\n/help - Показать этот список команд"

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, helpText)
	_, err := bot.Send(msg)

	slog.Info("Help command processed", slog.Int64("chat_id", update.Message.Chat.ID))
	return err
}