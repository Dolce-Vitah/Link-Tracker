package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	var app App
	if err := app.New(); err != nil {
		slog.Error("Failed to initialize app", slog.String("error", err.Error()))
		os.Exit(1)
	}

	app.Run()
<<<<<<< HEAD
}
=======
}
>>>>>>> 88c4bf20fc093c0a75b5b0152b79b20569a3635e
