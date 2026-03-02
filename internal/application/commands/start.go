package command

import (
	"fmt"
	"log/slog"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type StartCommand struct {}

func (c *StartCommand) Name() string { return "start" }
func (c *StartCommand) Description() string { return "Начать работу с ботом" }

func (c *StartCommand) Handle(update *tgbotapi.Update, bot Sender) error {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Добро пожаловать! Используйте /help, чтобы узнать о доступных командах.")
	_, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send start command response: %w", err)
	}

	slog.Info("Start command processed", slog.Int64("chat_id", update.Message.Chat.ID))
	return nil
}