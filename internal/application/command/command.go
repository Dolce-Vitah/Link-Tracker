package command

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/dto"
)

type (
	Sender interface {
		Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	}

	Command interface {
		Name() string
		Description() string
		Handle(ctx context.Context, request dto.CommandRequest) error
	}
)
