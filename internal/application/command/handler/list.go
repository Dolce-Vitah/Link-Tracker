package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
)

type ListCommandHandler struct {
	tracker tracker.Service
	logger  *slog.Logger
	bot     command.Sender
}

func NewListCommandHandler(trackerService tracker.Service, logger *slog.Logger, bot command.Sender) *ListCommandHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &ListCommandHandler{tracker: trackerService, logger: logger, bot: bot}
}

func (c *ListCommandHandler) Name() string { return "list" }
func (c *ListCommandHandler) Description() string {
	return "Показать отслеживаемые ссылки"
}

func (c *ListCommandHandler) Handle(ctx context.Context, text string, chatID int64) error {
	if err := c.tracker.RegisterChat(ctx, chatID); err != nil && !strings.Contains(err.Error(), tracker.ErrAlreadyExists.Error()) {
		return fmt.Errorf("register chat before list: %w", err)
	}

	resp, err := c.tracker.ListLinks(ctx, chatID)
	if err != nil {
		return fmt.Errorf("list tracked links: %w", err)
	}

	filterTag := extractCommandArgs(text)
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
		msg := tgbotapi.NewMessage(chatID, "Список отслеживаемых ссылок пуст.")
		_, sendErr := c.bot.Send(msg)
		return sendErr
	}

	listText := "Ваши отслеживаемые ссылки:\n" + strings.Join(filtered, "\n")
	msg := tgbotapi.NewMessage(chatID, listText)
	_, sendErr := c.bot.Send(msg)
	if sendErr != nil {
		return sendErr
	}

	c.logger.Info("List command processed", slog.Int64("chat_id", chatID), slog.Int("count", len(filtered)))
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
