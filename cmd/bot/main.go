package main

import (
	"log/slog"
	"os"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/telegram"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load("config.json")
	if err != nil {
		slog.Error("Failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	bot, err := telegram.NewBot(cfg.TelegramToken)
	if err != nil {
		slog.Error("Failed to create bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	bot.RegisterCommand(&commands.StartCommand{})
	bot.RegisterCommand(&commands.HelpCommand{})

	if err := bot.SetMyCommands(); err != nil {
		slog.Error("Failed to set bot commands menu", slog.String("error", err.Error()))
	} else {
		slog.Info("Bot commands menu updated successfully in UI")
	}

	slog.Info("Bot is up and running...")
	bot.Start()
}