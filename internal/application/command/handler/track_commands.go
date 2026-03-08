package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dialog"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
)

type TrackCommandHandler struct {
	sessions *dialog.Store
	tracker  tracker.Service
	logger   *slog.Logger
}

func NewTrackCommandHandler(sessions *dialog.Store, trackerService tracker.Service, logger *slog.Logger) *TrackCommandHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &TrackCommandHandler{
		sessions: sessions,
		tracker:  trackerService,
		logger:   logger,
	}
}

func (c *TrackCommandHandler) Name() string { return "track" }
func (c *TrackCommandHandler) Description() string {
	return "Начать отслеживание ссылки"
}

func (c *TrackCommandHandler) Handle(ctx context.Context, update *tgbotapi.Update, bot command.Sender) error {
	chatID := update.Message.Chat.ID
	if err := c.tracker.RegisterChat(ctx, chatID); err != nil && !strings.Contains(err.Error(), tracker.ErrAlreadyExists.Error()) {
		return fmt.Errorf("register chat before track: %w", err)
	}

	c.sessions.Set(chatID, dialog.Session{State: dialog.StateAwaitingURL})
	msg := tgbotapi.NewMessage(chatID, "Отправьте ссылку, которую нужно отслеживать. Для отмены используйте /cancel.")
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("send track prompt: %w", err)
	}

	c.logger.Info("Track command processed", slog.Int64("chat_id", chatID))
	return nil
}

type CancelCommandHandler struct {
	sessions *dialog.Store
	logger   *slog.Logger
}

func NewCancelCommandHandler(sessions *dialog.Store, logger *slog.Logger) *CancelCommandHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &CancelCommandHandler{sessions: sessions, logger: logger}
}

func (c *CancelCommandHandler) Name() string { return "cancel" }
func (c *CancelCommandHandler) Description() string {
	return "Отменить текущий диалог"
}

func (c *CancelCommandHandler) Handle(_ context.Context, update *tgbotapi.Update, bot command.Sender) error {
	chatID := update.Message.Chat.ID
	c.sessions.Clear(chatID)
	msg := tgbotapi.NewMessage(chatID, "Диалог отменен.")
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("send cancel confirmation: %w", err)
	}

	c.logger.Info("Cancel command processed", slog.Int64("chat_id", chatID))
	return nil
}
