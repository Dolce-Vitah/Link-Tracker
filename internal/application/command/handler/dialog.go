package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dialog"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

func HandleTrackDialogStep(
	ctx context.Context,
	trackerService tracker.Service,
	sessions *dialog.Store,
	update *tgbotapi.Update,
	bot command.Sender,
	logger *slog.Logger,
) error {
	if logger == nil {
		logger = slog.Default()
	}
	if update == nil || update.Message == nil {
		return nil
	}
	chatID := update.Message.Chat.ID
	session, ok := sessions.Get(chatID)
	if !ok {
		return nil
	}

	text := strings.TrimSpace(update.Message.Text)
	if update.Message.IsCommand() {
		if update.Message.Command() != "cancel" {
			sessions.Clear(chatID)
			msg := tgbotapi.NewMessage(chatID, "Процесс отслеживания отменен из-за новой команды.")
			if _, err := bot.Send(msg); err != nil {
				return fmt.Errorf("send cancellation due to command: %w", err)
			}
			logger.Info("Track dialog cancelled by another command", slog.Int64("chat_id", chatID), slog.String("command", update.Message.Command()))
		}
		return nil
	}

	if text == "" {
		msg := tgbotapi.NewMessage(chatID, "Сообщение пустое. Введите корректные данные или /cancel.")
		_, err := bot.Send(msg)
		return err
	}

	switch session.State {
	case dialog.StateAwaitingURL:
		if !isValidURL(text) {
			msg := tgbotapi.NewMessage(chatID, "Ссылка некорректна. Введите ссылку в формате https://example.com/path")
			_, err := bot.Send(msg)
			return err
		}
		sessions.Set(chatID, dialog.Session{
			State:      dialog.StateAwaitingTags,
			PendingURL: text,
		})
		msg := tgbotapi.NewMessage(chatID, "Введите теги через запятую (например: работа,документация) или '-' если без тегов.")
		_, err := bot.Send(msg)
		return err
	case dialog.StateAwaitingTags:
		tags := parseTags(text)
		_, err := trackerService.AddLink(ctx, chatID, api.AddLinkRequest{
			Link: textOrPendingURL(session.PendingURL),
			Tags: tags,
		})
		if err != nil {
			if strings.Contains(err.Error(), tracker.ErrAlreadyExists.Error()) {
				sessions.Clear(chatID)
				msg := tgbotapi.NewMessage(chatID, "Ссылка уже отслеживается")
				_, sendErr := bot.Send(msg)
				if sendErr != nil {
					return fmt.Errorf("send duplicate track message: %w", sendErr)
				}
				return nil
			}
			return fmt.Errorf("add tracked link: %w", err)
		}
		sessions.Clear(chatID)
		msg := tgbotapi.NewMessage(chatID, "Ссылка успешно добавлена в отслеживание.")
		_, sendErr := bot.Send(msg)
		if sendErr != nil {
			return fmt.Errorf("send tracking success message: %w", sendErr)
		}
		logger.Info("Track dialog completed", slog.Int64("chat_id", chatID), slog.String("url", textOrPendingURL(session.PendingURL)))
		return nil
	default:
		return nil
	}
}

func parseTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "-" {
		return nil
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func isValidURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func textOrPendingURL(value string) string {
	return strings.TrimSpace(value)
}
