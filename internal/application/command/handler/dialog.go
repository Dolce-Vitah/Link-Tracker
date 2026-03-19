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

type TrackDialogHandler struct {
	trackerService tracker.Service
	sessions       *dialog.Store
	bot            command.Sender
	logger         *slog.Logger
}

func NewTrackDialogHandler(
	trackerService tracker.Service,
	sessions *dialog.Store,
	bot command.Sender,
	logger *slog.Logger,
) *TrackDialogHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &TrackDialogHandler{
		trackerService: trackerService,
		sessions:       sessions,
		bot:            bot,
		logger:         logger,
	}
}

func HandleTrackDialogStep(
	ctx context.Context,
	trackerService tracker.Service,
	sessions *dialog.Store,
	update *tgbotapi.Update,
	bot command.Sender,
	logger *slog.Logger,
) error {
	return NewTrackDialogHandler(trackerService, sessions, bot, logger).Handle(ctx, update)
}

func (h *TrackDialogHandler) Handle(ctx context.Context, update *tgbotapi.Update) error {
	if update == nil || update.Message == nil {
		return nil
	}

	chatID := update.Message.Chat.ID
	session, ok := h.sessions.Get(chatID)
	if !ok {
		return nil
	}

	text := strings.TrimSpace(update.Message.Text)
	if update.Message.IsCommand() {
		return h.handleCommandDuringDialog(chatID, update.Message.Command())
	}

	if text == "" {
		return h.sendPlainMessage(chatID, "Сообщение пустое. Введите корректные данные или /cancel.")
	}

	switch session.State {
	case dialog.StateAwaitingURL:
		return h.handleAwaitingURL(chatID, text)
	case dialog.StateAwaitingTags:
		return h.handleAwaitingTags(ctx, chatID, session.PendingURL, text)
	default:
		return nil
	}
}

func (h *TrackDialogHandler) handleCommandDuringDialog(chatID int64, cmdName string) error {
	if cmdName == "cancel" {
		return nil
	}

	h.sessions.Clear(chatID)
	if err := h.sendPlainMessage(chatID, "Процесс отслеживания отменен из-за новой команды."); err != nil {
		return fmt.Errorf("send cancellation due to command: %w", err)
	}

	h.logger.Info("Track dialog cancelled by another command",
		slog.Int64("chat_id", chatID),
		slog.String("command", cmdName),
	)
	return nil
}

func (h *TrackDialogHandler) handleAwaitingURL(chatID int64, text string) error {
	if !isValidURL(text) {
		return h.sendPlainMessage(chatID, "Ссылка некорректна. Введите ссылку в формате https://example.com/path")
	}

	h.sessions.Set(chatID, dialog.Session{
		State:      dialog.StateAwaitingTags,
		PendingURL: text,
	})

	return h.sendPlainMessage(chatID, "Введите теги через запятую (например: работа,документация) или '-' если без тегов.")
}

func (h *TrackDialogHandler) handleAwaitingTags(ctx context.Context, chatID int64, pendingURL string, rawTags string) error {
	tags := parseTags(rawTags)
	cleanURL := textOrPendingURL(pendingURL)

	_, err := h.trackerService.AddLink(ctx, chatID, api.AddLinkRequest{
		Link: cleanURL,
		Tags: tags,
	})
	if err != nil {
		if strings.Contains(err.Error(), tracker.ErrAlreadyExists.Error()) {
			h.sessions.Clear(chatID)
			if sendErr := h.sendPlainMessage(chatID, "Ссылка уже отслеживается"); sendErr != nil {
				return fmt.Errorf("send duplicate track message: %w", sendErr)
			}
			return nil
		}
		return fmt.Errorf("add tracked link: %w", err)
	}

	h.sessions.Clear(chatID)
	if sendErr := h.sendPlainMessage(chatID, "Ссылка успешно добавлена в отслеживание."); sendErr != nil {
		return fmt.Errorf("send tracking success message: %w", sendErr)
	}

	h.logger.Info("Track dialog completed",
		slog.Int64("chat_id", chatID),
		slog.String("url", cleanURL),
	)
	return nil
}

func (h *TrackDialogHandler) sendPlainMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := h.bot.Send(msg)
	return err
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
