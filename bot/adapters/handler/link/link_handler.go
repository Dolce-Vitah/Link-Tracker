package link

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/common"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type Handler struct {
	sessions repository.SessionRepository
	tracker  tracker.Service
	logger   *slog.Logger
	bot      command.Sender
}

func NewLinkHandler(
	sessions repository.SessionRepository,
	trackerService tracker.Service,
	bot command.Sender,
	logger *slog.Logger,
) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		sessions: sessions,
		tracker:  trackerService,
		logger:   logger,
		bot:      bot,
	}
}

func (h *Handler) Create(ctx context.Context, request dto.CommandRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate track request: %w", err)
	}

	if err := h.tracker.RegisterChat(ctx, request.ChatID); err != nil && !strings.Contains(err.Error(), tracker.ErrAlreadyExists.Error()) {
		return fmt.Errorf("register chat before track: %w", err)
	}

	h.sessions.Set(request.ChatID, repository.Session{State: repository.StateAwaitingURL})
	msg := tgbotapi.NewMessage(request.ChatID, "Отправьте ссылку, которую нужно отслеживать. Для отмены используйте /cancel.")
	if _, err := h.bot.Send(msg); err != nil {
		return fmt.Errorf("send track prompt: %w", err)
	}

	h.logger.Info("Track command processed", slog.Int64("chat_id", request.ChatID))
	return nil
}

func (h *Handler) Read(ctx context.Context, request dto.CommandRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate list request: %w", err)
	}

	if err := h.tracker.RegisterChat(ctx, request.ChatID); err != nil && !strings.Contains(err.Error(), tracker.ErrAlreadyExists.Error()) {
		return fmt.Errorf("register chat before list: %w", err)
	}

	resp, err := h.tracker.ListLinks(ctx, request.ChatID)
	if err != nil {
		return fmt.Errorf("list tracked links: %w", err)
	}

	filterTag := extractCommandArgs(request.Text)
	filtered := make([]string, 0, len(resp.Links))
	for _, link := range resp.Links {
		if filterTag == "" || hasTag(link.Tags, filterTag) {
			line := fmt.Sprintf("- %s", link.URL)
			if len(link.Tags) > 0 {
				line += fmt.Sprintf(" (теги: %s)", strings.Join(link.Tags, ", "))
			}
			filtered = append(filtered, line)
		}
	}

	if len(filtered) == 0 {
		msg := tgbotapi.NewMessage(request.ChatID, "Список отслеживаемых ссылок пуст.")
		if _, sendErr := h.bot.Send(msg); sendErr != nil {
			return fmt.Errorf("send empty list response: %w", sendErr)
		}
		return nil
	}

	listText := "Ваши отслеживаемые ссылки:\n" + strings.Join(filtered, "\n")
	msg := tgbotapi.NewMessage(request.ChatID, listText)
	if _, sendErr := h.bot.Send(msg); sendErr != nil {
		return fmt.Errorf("send list response: %w", sendErr)
	}

	h.logger.Info("List command processed", slog.Int64("chat_id", request.ChatID), slog.Int("count", len(filtered)))
	return nil
}

func (h *Handler) Update(_ context.Context, request dto.CommandRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate cancel request: %w", err)
	}

	h.sessions.Clear(request.ChatID)
	msg := tgbotapi.NewMessage(request.ChatID, "Диалог отменен.")
	if _, err := h.bot.Send(msg); err != nil {
		return fmt.Errorf("send cancel confirmation: %w", err)
	}

	h.logger.Info("Cancel command processed", slog.Int64("chat_id", request.ChatID))
	return nil
}

func (h *Handler) Delete(ctx context.Context, request dto.CommandRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate untrack request: %w", err)
	}

	args := extractCommandArgs(request.Text)
	if !isValidUntrackURL(args) {
		msg := tgbotapi.NewMessage(request.ChatID, "Укажите ссылку после команды: /untrack https://example.com")
		if _, sendErr := h.bot.Send(msg); sendErr != nil {
			return fmt.Errorf("send untrack usage hint: %w", sendErr)
		}
		return nil
	}

	if err := h.tracker.RegisterChat(ctx, request.ChatID); err != nil && !strings.Contains(err.Error(), tracker.ErrAlreadyExists.Error()) {
		return fmt.Errorf("register chat before untrack: %w", err)
	}

	_, err := h.tracker.RemoveLink(ctx, request.ChatID, api.RemoveLinkRequest{Link: args})
	if err != nil {
		if strings.Contains(err.Error(), tracker.ErrNotFound.Error()) {
			msg := tgbotapi.NewMessage(request.ChatID, "Ссылка не найдена в отслеживаемых.")
			if _, sendErr := h.bot.Send(msg); sendErr != nil {
				return fmt.Errorf("send not-tracked response: %w", sendErr)
			}
			return nil
		}
		return fmt.Errorf("remove link from tracking: %w", err)
	}

	msg := tgbotapi.NewMessage(request.ChatID, "Ссылка удалена из отслеживания.")
	if _, sendErr := h.bot.Send(msg); sendErr != nil {
		return fmt.Errorf("send untrack success response: %w", sendErr)
	}

	h.logger.Info("Untrack command processed", slog.Int64("chat_id", request.ChatID), slog.String("url", args))
	return nil
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(strings.TrimSpace(t), tag) {
			return true
		}
	}
	return false
}

func isValidUntrackURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func NewTrackCommand(handler *Handler) command.Command {
	return common.NewCommand("track", "Начать отслеживание ссылки", handler.Create)
}

func NewListCommand(handler *Handler) command.Command {
	return common.NewCommand("list", "Показать отслеживаемые ссылки", handler.Read)
}

func NewCancelCommand(handler *Handler) command.Command {
	return common.NewCommand("cancel", "Отменить текущий диалог", handler.Update)
}

func NewUntrackCommand(handler *Handler) command.Command {
	return common.NewCommand("untrack", "Прекратить отслеживание ссылки", handler.Delete)
}
