package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type TrackDialogHandler struct {
	trackerService tracker.Service
	sessions       repository.SessionRepository
	bot            command.Sender
	logger         *slog.Logger
}

func NewTrackDialogHandler(
	trackerService tracker.Service,
	sessions repository.SessionRepository,
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

func (h *TrackDialogHandler) Handle(ctx context.Context, request dto.DialogRequest) error {
	if err := request.Validate(); err != nil {
		return nil
	}

	chatID := request.ChatID()
	session, ok := h.sessions.Get(chatID)
	if !ok {
		return nil
	}

	text := request.Text()
	if request.IsCommand() {
		return h.handleCommandDuringDialog(chatID, request.Command())
	}

	if text == "" {
		return h.sendPlainMessage(chatID, "Сообщение пустое. Введите корректные данные или /cancel.")
	}

	switch session.State {
	case repository.StateAwaitingURL:
		return h.handleAwaitingURL(chatID, text)
	case repository.StateAwaitingTags:
		return h.handleAwaitingTags(ctx, chatID, session.PendingURL, text)
	case repository.StateIdle:
		return nil
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

	h.sessions.Set(chatID, repository.Session{
		State:      repository.StateAwaitingTags,
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
	if err != nil {
		return fmt.Errorf("send dialog message: %w", err)
	}
	return nil
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
