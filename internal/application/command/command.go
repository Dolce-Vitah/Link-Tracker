package command

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type (
	Sender interface {
		Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	}

	Command interface {
		Name() string
		Description() string
		Handle(ctx context.Context, update *tgbotapi.Update, sender Sender) error
	}
)
