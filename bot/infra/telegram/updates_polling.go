package telegram

import (
	"context"
	"errors"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command"
)

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		b.handleUpdate(context.Background(), &update, b.api)
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update *tgbotapi.Update, sender command.Sender) {
	if update == nil || update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	cmdName := update.Message.Command()
	text := update.Message.Text

	if b.tracker != nil {
		dialogHandler := handler.NewTrackDialogHandler(b.tracker, b.sessions, sender, b.logger)
		if err := dialogHandler.Handle(ctx, dto.DialogRequest{Update: update}); err != nil {
			slog.Error("Failed to process track dialog step", slog.String("error", err.Error()))
			return
		}
	}

	if !update.Message.IsCommand() {
		return
	}

	err := b.dispatcher.Dispatch(ctx, dto.CommandRequest{
		Text:   text,
		ChatID: chatID,
	}, cmdName)
	if err == nil {
		return
	}

	if errors.Is(err, command.ErrUnknownCommand) {
		var username string
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

	b.logger.Error("Failed to handle command",
		slog.String("command", cmdName),
		slog.String("error", err.Error()),
		slog.Int64("chat_id", chatID),
	)
}
