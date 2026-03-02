package main

import (
	"fmt"
	"log/slog"

	command "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/telegram"
)

type App struct {
	bot *telegram.Bot
}

func (a *App) New() error {
	cfg, err := config.Load("config.json")
	if err != nil {
		return fmt.Errorf("load app config: %w", err)
	}

	bot, err := telegram.NewBot(cfg.TelegramToken)
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", err)
	}

	bot.RegisterCommand(&command.StartCommand{})
	bot.RegisterCommand(&command.HelpCommand{})

	if err := bot.SetMyCommands(); err != nil {
		slog.Error("Failed to set bot commands menu", slog.String("error", err.Error()))
	} else {
		slog.Info("Bot commands menu updated successfully in UI")
	}

	a.bot = bot
	return nil
}

func (a *App) Run() {
	slog.Info("Bot is up and running...")
	a.bot.Start()
}
