package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
)

type Bot struct {
	api        *tgbotapi.BotAPI
	dispatcher *command.Dispatcher
	logger     *slog.Logger
}

func NewBot(token string, logger *slog.Logger) (*Bot, error) {
	if logger == nil {
		logger = slog.Default()
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("new telegram bot api: %w", err)
	}

	return &Bot{
		api:        api,
		dispatcher: command.NewDispatcher(),
		logger:     logger,
	}, nil
}

func (b *Bot) RegisterCommand(cmd command.Command) {
	b.dispatcher.Register(cmd)
}

func (b *Bot) SetMyCommands() error {
	commands := b.dispatcher.Commands()
	tgCommands := make([]tgbotapi.BotCommand, 0, len(commands))
	for _, cmd := range commands {
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
		b.handleUpdate(context.Background(), &update, b.api)
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update *tgbotapi.Update, sender command.Sender) {
	err := b.dispatcher.Dispatch(ctx, update, sender)
	if err == nil {
		return
	}

	if errors.Is(err, command.ErrUnknownCommand) && update.Message != nil && update.Message.IsCommand() {
		var (
			cmdName  = update.Message.Command()
			chatID   = update.Message.Chat.ID
			username string
		)
		if update.Message.From != nil {
			username = update.Message.From.UserName
		}
		msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.")
		_, sendErr := sender.Send(msg)
		if sendErr != nil {
			b.logger.Error("Failed to send unknown command warning", slog.String("error", sendErr.Error()))
		}

		b.logger.Warn("Unknown command received",
			slog.String("command", cmdName),
			slog.Int64("chat_id", chatID),
			slog.String("username", username),
		)
		return
	}

	var (
		cmdName string
		chatID  int64
	)
	if update.Message != nil {
		chatID = update.Message.Chat.ID
		if update.Message.IsCommand() {
			cmdName = update.Message.Command()
		}
	}

	b.logger.Error("Failed to handle command",
		slog.String("command", cmdName),
		slog.String("error", err.Error()),
		slog.Int64("chat_id", chatID),
	)
}
