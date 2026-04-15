package handlerapi

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command"
)

type commandAdapter struct {
	name        string
	description string
	handle      func(context.Context, dto.CommandRequest) error
}

func (a *commandAdapter) Name() string { return a.name }

func (a *commandAdapter) Description() string { return a.description }

func (a *commandAdapter) Handle(ctx context.Context, request dto.CommandRequest) error {
	return a.handle(ctx, request)
}

func NewCommand(name string, description string, handle func(context.Context, dto.CommandRequest) error) command.Command {
	return &commandAdapter{
		name:        name,
		description: description,
		handle:      handle,
	}
}
