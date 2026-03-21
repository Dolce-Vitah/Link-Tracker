package main

import (
	"log/slog"
	"os"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/app"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	var app app.App
	if err := app.New(); err != nil {
		slog.Error("Failed to initialize app", slog.String("error", err.Error()))
		os.Exit(1)
	}

	app.Run()
}
