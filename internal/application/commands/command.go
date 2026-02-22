package commands

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Sender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

type Command interface {
	Name() string
	Description() string
	Handle(update *tgbotapi.Update, bot Sender) error
}