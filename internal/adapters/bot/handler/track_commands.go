package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository"
)

type TrackCommandHandler struct {
	sessions repository.SessionRepository
	tracker  tracker.Service
	logger   *slog.Logger
	bot      command.Sender
}

func NewTrackCommandHandler(
	sessions repository.SessionRepository,
	trackerService tracker.Service,
	logger *slog.Logger,
	bot command.Sender,
) *TrackCommandHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &TrackCommandHandler{
		sessions: sessions,
		tracker:  trackerService,
		logger:   logger,
		bot:      bot,
	}
}

func (c *TrackCommandHandler) Name() string { return "track" }
func (c *TrackCommandHandler) Description() string {
	return "Начать отслеживание ссылки"
}

func (c *TrackCommandHandler) Handle(ctx context.Context, request dto.CommandRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate track request: %w", err)
	}

	if err := c.tracker.RegisterChat(ctx, request.ChatID); err != nil && !strings.Contains(err.Error(), tracker.ErrAlreadyExists.Error()) {
		return fmt.Errorf("register chat before track: %w", err)
	}

	c.sessions.Set(request.ChatID, repository.Session{State: repository.StateAwaitingURL})
	msg := tgbotapi.NewMessage(request.ChatID, "Отправьте ссылку, которую нужно отслеживать. Для отмены используйте /cancel.")
	if _, err := c.bot.Send(msg); err != nil {
		return fmt.Errorf("send track prompt: %w", err)
	}

	c.logger.Info("Track command processed", slog.Int64("chat_id", request.ChatID))
	return nil
}

type CancelCommandHandler struct {
	sessions repository.SessionRepository
	logger   *slog.Logger
	bot      command.Sender
}

func NewCancelCommandHandler(sessions repository.SessionRepository, logger *slog.Logger, bot command.Sender) *CancelCommandHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &CancelCommandHandler{sessions: sessions, logger: logger, bot: bot}
}

func (c *CancelCommandHandler) Name() string { return "cancel" }
func (c *CancelCommandHandler) Description() string {
	return "Отменить текущий диалог"
}

func (c *CancelCommandHandler) Handle(_ context.Context, request dto.CommandRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate cancel request: %w", err)
	}

	c.sessions.Clear(request.ChatID)
	msg := tgbotapi.NewMessage(request.ChatID, "Диалог отменен.")
	if _, err := c.bot.Send(msg); err != nil {
		return fmt.Errorf("send cancel confirmation: %w", err)
	}

	c.logger.Info("Cancel command processed", slog.Int64("chat_id", request.ChatID))
	return nil
}
