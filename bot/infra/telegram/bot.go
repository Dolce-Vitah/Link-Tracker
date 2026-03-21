package telegram

import (
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/repository"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	dispatcher  *command.Dispatcher
	logger      *slog.Logger
	sessions    repository.SessionRepository
	tracker     tracker.Service
	sendMessage func(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

func NewBot(token string, apiURL string, logger *slog.Logger) (*Bot, error) {
	if logger == nil {
		logger = slog.Default()
	}

	var api *tgbotapi.BotAPI
	var err error

	if apiURL != "" {
		api, err = tgbotapi.NewBotAPIWithAPIEndpoint(token, apiURL)
	} else {
		api, err = tgbotapi.NewBotAPI(token)
	}

	if err != nil {
		return nil, fmt.Errorf("new telegram bot api: %w", err)
	}

	return &Bot{
		api:         api,
		dispatcher:  command.NewDispatcher(),
		logger:      logger,
		sessions:    repository.NewInMemorySessionRepository(),
		sendMessage: api.Send,
	}, nil
}

func (b *Bot) Client() command.Sender {
	return b.api
}

func (b *Bot) RegisterCommand(cmd command.Command) {
	b.dispatcher.Register(cmd)
}

func (b *Bot) Sessions() repository.SessionRepository {
	return b.sessions
}

func (b *Bot) SetTrackerService(service tracker.Service) {
	b.tracker = service
}
