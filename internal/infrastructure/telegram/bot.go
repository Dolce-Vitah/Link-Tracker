package telegram

import (
	"fmt"
	"log/slog"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/commands"
)

type Bot struct {
	api		*tgbotapi.BotAPI
	commands map[string]commands.Command
}

func NewBot(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("new telegram bot api: %w", err)
	}

	return &Bot{
		api: 		api,
		commands:	make(map[string]commands.Command),
	}, nil
}

func (b *Bot) RegisterCommand(cmd commands.Command) {
	b.commands[cmd.Name()] = cmd
}

func (b *Bot) SetMyCommands() error {
	var tgCommands []tgbotapi.BotCommand
	for _, cmd := range b.commands {
		tgCommands = append(tgCommands, tgbotapi.BotCommand{
			Command: cmd.Name(),
			Description: cmd.Description(),
		})
	}

	config := tgbotapi.NewSetMyCommands(tgCommands...)
	_, err := b.api.Request(config)
	if err != nil {
		return fmt.Errorf("set telegram commands menu: %w", err)
	}

	return nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		b.handleUpdate(&update, b.api)
	}
}

func (b *Bot) handleUpdate(update *tgbotapi.Update, sender commands.Sender) {
	if update.Message == nil || !update.Message.IsCommand() {
		return
	}

	cmdName := update.Message.Command()
	chatID := update.Message.Chat.ID
	username := update.Message.From.UserName

	cmd, exists := b.commands[cmdName]
	if !exists {
		msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.")
		_, err := sender.Send(msg)
		if err != nil {
			slog.Error("Failed to send unknown command warning", slog.String("error", err.Error()))
		}

		slog.Warn("Unknown command received",
			slog.String("command", cmdName),
			slog.Int64("chat_id", chatID),
			slog.String("username", username),
		)
		return
	}

	err := cmd.Handle(update, sender)
	if err != nil {
		slog.Error("Failed to handle command",
			slog.String("command", cmdName),
			slog.String("error", err.Error()),
			slog.Int64("chat_id", chatID),
		)
	}
}