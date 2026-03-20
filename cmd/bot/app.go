package main

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/handler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/telegram"
)

type App struct {
	bot        *telegram.Bot
	config     *config.Config
	trackerSvc tracker.Service
}

func (a *App) New() error {
	cfg, err := config.Load("config.json")
	if err != nil {
		return fmt.Errorf("load app config: %w", err)
	}

	bot, err := telegram.NewBot(cfg.TelegramToken, cfg.TelegramAPIURL, slog.Default())
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", err)
	}

	timeout, err := time.ParseDuration(cfg.ExternalHTTPTimeout)
	if err != nil {
		return fmt.Errorf("parse external http timeout: %w", err)
	}

	var trackerService tracker.Service
	if strings.EqualFold(cfg.TransportMode, "grpc") {
		grpcClient, grpcErr := scrapper.NewGRPCClient(cfg.ScrapperGRPCTarget, timeout)
		if grpcErr != nil {
			return fmt.Errorf("create scrapper grpc client: %w", grpcErr)
		}
		trackerService = grpcClient
	} else {
		trackerService = scrapper.NewHTTPClient(cfg.ScrapperBaseURL, timeout)
	}
	bot.SetTrackerService(trackerService)
	sessions := bot.Sessions()

	bot.RegisterCommand(handler.NewStartCommandHandler(trackerService, slog.Default(), bot.Client()))
	bot.RegisterCommand(handler.NewHelpCommandHandler(slog.Default(), bot.Client()))
	bot.RegisterCommand(handler.NewTrackCommandHandler(sessions, trackerService, slog.Default(), bot.Client()))
	bot.RegisterCommand(handler.NewUntrackCommandHandler(trackerService, slog.Default(), bot.Client()))
	bot.RegisterCommand(handler.NewListCommandHandler(trackerService, slog.Default(), bot.Client()))
	bot.RegisterCommand(handler.NewCancelCommandHandler(sessions, slog.Default(), bot.Client()))

	setCommandsErr := bot.SetMyCommands()
	if setCommandsErr != nil {
		slog.Error("Failed to set bot commands menu", slog.String("error", setCommandsErr.Error()))
	} else {
		slog.Info("Bot commands menu updated successfully in UI")
	}

	a.bot = bot
	a.config = cfg
	a.trackerSvc = trackerService
	return nil
}

func (a *App) Run() {
	slog.Info("Bot is up and running...")
	go func() {
		if err := a.bot.StartHTTPServer(a.config.BotHTTPAddress); err != nil {
			slog.Error("Bot HTTP server failed", slog.String("error", err.Error()))
		}
	}()
	a.bot.Start()
}
