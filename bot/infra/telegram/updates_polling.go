package telegram

import (
	"context"
	"errors"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command"
)

func (b *Bot) Start() {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updates := b.api.GetUpdatesChan(updateConfig)

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

	log := b.logger.With(
		slog.Int64("chat_id", chatID),
		slog.String("command", cmdName),
	)

	dialogHandler := link.NewDialogHandler(b.tracker, b.sessions, sender, log)
	if err := dialogHandler.Handle(ctx, dto.DialogRequest{Update: update}); err != nil {
		log.Error("Failed to process track dialog step", slog.String("error", err.Error()))

		return
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
			log.Error("Failed to send unknown command warning", slog.String("error", sendErr.Error()))
		}

		log.Warn("Unknown command received", slog.String("username", username))

		return
	}

	log.Error("Failed to handle command", slog.String("error", err.Error()))
}
