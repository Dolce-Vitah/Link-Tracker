package telegram

import (
	"errors"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	command "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/commands"
)

type Bot struct {
	api        *tgbotapi.BotAPI
	dispatcher *command.Dispatcher
}

func NewBot(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("new telegram bot api: %w", err)
	}

	return &Bot{
		api:        api,
		dispatcher: command.NewDispatcher(),
	}, nil
}

func (b *Bot) RegisterCommand(cmd command.Command, handler command.Handler) {
	b.dispatcher.Register(cmd, handler)
}

func (b *Bot) SetMyCommands() error {
	var tgCommands []tgbotapi.BotCommand
	for _, cmd := range b.dispatcher.Commands() {
		tgCommands = append(tgCommands, tgbotapi.BotCommand{
			Command:     cmd.Name(),
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

func (b *Bot) handleUpdate(update *tgbotapi.Update, sender command.Sender) {
	err := b.dispatcher.Dispatch(update, sender)
	if err == nil {
		return
	}

	if errors.Is(err, command.ErrUnknownCommand) && update.Message != nil && update.Message.IsCommand() {
		cmdName := update.Message.Command()
		chatID := update.Message.Chat.ID
		username := ""
		if update.Message.From != nil {
			username = update.Message.From.UserName
		}
		msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.")
		_, sendErr := sender.Send(msg)
		if sendErr != nil {
			slog.Error("Failed to send unknown command warning", slog.String("error", sendErr.Error()))
		}

		slog.Warn("Unknown command received",
			slog.String("command", cmdName),
			slog.Int64("chat_id", chatID),
			slog.String("username", username),
		)
		return
	}

	cmdName := ""
	chatID := int64(0)
	if update.Message != nil {
		chatID = update.Message.Chat.ID
		if update.Message.IsCommand() {
			cmdName = update.Message.Command()
		}
	}

	slog.Error("Failed to handle command",
		slog.String("command", cmdName),
		slog.String("error", err.Error()),
		slog.Int64("chat_id", chatID),
	)
}
