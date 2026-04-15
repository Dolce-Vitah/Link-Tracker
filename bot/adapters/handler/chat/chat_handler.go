package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/handlerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
)

const (
	defaultStartText = "Добро пожаловать! Используйте /help, чтобы узнать о доступных командах."
	defaultHelpText  = "Доступные команды:\n" +
		"/start - Начало работы с ботом\n" +
		"/help - Показать этот список команд\n" +
		"/track - Добавить ссылку на отслеживание\n" +
		"/untrack <url> - Убрать ссылку из отслеживания\n" +
		"/list [tag] - Показать отслеживаемые ссылки\n" +
		"/cancel - Отменить текущий диалог"
)

type Handler struct {
	logger    *slog.Logger
	bot       command.Sender
	startText string
	helpText  string
	tracker   tracker.Client
}

func NewChatHandler(trackerService tracker.Client, bot command.Sender, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		logger:    logger,
		bot:       bot,
		startText: defaultStartText,
		helpText:  defaultHelpText,
		tracker:   trackerService,
	}
}

func (h *Handler) Create(ctx context.Context, request dto.CommandRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate start request: %w", err)
	}

	if h.tracker != nil {
		err := h.tracker.RegisterChat(ctx, request.ChatID)
		if err != nil && !errors.Is(err, tracker.ErrAlreadyExists) {
			return fmt.Errorf("register chat: %w", err)
		}
	}

	msg := tgbotapi.NewMessage(request.ChatID, h.startText)
	if _, err := h.bot.Send(msg); err != nil {
		return fmt.Errorf("send start command response: %w", err)
	}

	h.logger.Info("Start command processed", slog.Int64("chat_id", request.ChatID))
	return nil
}

func (h *Handler) Read(_ context.Context, request dto.CommandRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate help request: %w", err)
	}

	msg := tgbotapi.NewMessage(request.ChatID, h.helpText)
	if _, err := h.bot.Send(msg); err != nil {
		return fmt.Errorf("send help command response: %w", err)
	}

	h.logger.Info("Help command processed", slog.Int64("chat_id", request.ChatID))
	return nil
}

func NewStartCommand(handler *Handler) command.Command {
	return handlerapi.NewCommand("start", "Начать работу с ботом", handler.Create)
}

func NewHelpCommand(handler *Handler) command.Command {
	return handlerapi.NewCommand("help", "Вывести список доступных команд", handler.Read)
}
