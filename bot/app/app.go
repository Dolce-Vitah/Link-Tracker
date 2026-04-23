package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	trackergateway "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/gateway/client/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/chat"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/infra/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/infra/notifications"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/infra/telegram"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/infra/trackerclient"
)

type App struct {
	bot        *telegram.Bot
	config     *config.Config
	trackerSvc tracker.Client
	consumer   *notifications.KafkaConsumer
}

func (a *App) New() error {
	cfg, err := config.Load("config.bot.json")
	if err != nil {

		return fmt.Errorf("load app config: %w", err)
	}

	timeout, err := time.ParseDuration(cfg.ExternalHTTPTimeout)
	if err != nil {

		return fmt.Errorf("parse external http timeout: %w", err)
	}

	var trackerService tracker.Client
	if cfg.TransportMode == config.TransportModeGRPC {
		grpcClient, grpcErr := trackerclient.NewGRPCClient(cfg.ScrapperGRPCTarget, timeout)
		if grpcErr != nil {

			return fmt.Errorf("create scrapper grpc client: %w", grpcErr)
		}
		trackerService = grpcClient
	} else {
		trackerService = trackergateway.NewHTTPClient(cfg.ScrapperBaseURL, timeout)
	}

	bot, err := telegram.NewBot(cfg.TelegramToken, cfg.TelegramAPIURL, slog.Default(), trackerService)
	if err != nil {

		return fmt.Errorf("create telegram bot: %w", err)
	}

	sessions := bot.Sessions()

	chatHandler := chat.NewChatHandler(trackerService, bot.Client(), slog.Default())

	linkHandler := link.NewLinkHandler(sessions, trackerService, bot.Client(), slog.Default())

	bot.RegisterCommand(chat.NewStartCommand(chatHandler))

	bot.RegisterCommand(chat.NewHelpCommand(chatHandler))

	bot.RegisterCommand(link.NewTrackCommand(linkHandler))

	bot.RegisterCommand(link.NewUntrackCommand(linkHandler))

	bot.RegisterCommand(link.NewListCommand(linkHandler))

	bot.RegisterCommand(link.NewCancelCommand(linkHandler))

	setCommandsErr := bot.SetMyCommands()
	if setCommandsErr != nil {
		slog.Error("Failed to set bot commands menu", slog.String("error", setCommandsErr.Error()))
	} else {
		slog.Info("Bot commands menu updated successfully in UI")
	}

	a.bot = bot
	a.config = cfg
	a.trackerSvc = trackerService

	if cfg.NotificationMode == config.NotificationModeKafka {
		consumer, consumerErr := notifications.NewKafkaConsumer(notifications.KafkaConfig{
			Brokers:          cfg.KafkaBrokers,
			Topic:            cfg.KafkaTopic,
			DLQTopic:         cfg.KafkaDLQTopic,
			ConsumerGroup:    cfg.KafkaConsumerGroup,
			ReaderMinBytes:   cfg.KafkaReaderMinBytes,
			ReaderMaxBytes:   cfg.KafkaReaderMaxBytes,
			MaxRetryAttempts: cfg.KafkaMaxRetryAttempts,
		}, bot)
		if consumerErr != nil {
			return fmt.Errorf("create kafka consumer: %w", consumerErr)
		}
		a.consumer = consumer
	}

	return nil
}

func (a *App) Run() {
	slog.Info("Bot is up and running...")
	if a.config.NotificationMode == config.NotificationModeHTTP {
		go func() {
			if err := a.bot.StartHTTPServer(a.config.BotHTTPAddress); err != nil {
				slog.Error("Bot HTTP server failed", slog.String("error", err.Error()))
			}
		}()
	} else if a.consumer != nil {
		go a.consumer.Run(context.Background())
	}

	a.bot.Start()
}
