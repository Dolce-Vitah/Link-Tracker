package command

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate go run github.com/vektra/mockery/v2@latest --config ../../../.mockery.yaml

type (
	Sender interface {
		Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	}

	Command interface {
		Name() string
		Description() string
	}

	Handler interface {
		Handle(update *tgbotapi.Update, bot Sender) error
	}
)
