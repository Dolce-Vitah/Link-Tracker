package command

import (
	"context"
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
)

var ErrUnknownCommand = errors.New("unknown command")

type (
	Sender interface {
		Send(message tgbotapi.Chattable) (tgbotapi.Message, error)
	}

	Command interface {
		Name() string
		Description() string
		Handle(ctx context.Context, request dto.CommandRequest) error
	}
)

type DialogHandler interface {
	Handle(context.Context, dto.DialogRequest) error
}

type BotClient interface {
	Send(tgbotapi.Chattable) (tgbotapi.Message, error)
}

type UnknownCommandError struct {
	Command string
}

func (e *UnknownCommandError) Error() string {

	return fmt.Sprintf("%s: %s", ErrUnknownCommand.Error(), e.Command)
}

func (e *UnknownCommandError) Unwrap() error {

	return ErrUnknownCommand
}
