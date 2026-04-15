package command

import (
	"context"
	"fmt"
	"sort"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
)

type Dispatcher struct {
	commands map[string]Command
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		commands: make(map[string]Command),
	}
}

func (d *Dispatcher) Register(cmd Command) {
	d.commands[cmd.Name()] = cmd
}

func (d *Dispatcher) Commands() []Command {
	result := make([]Command, 0, len(d.commands))
	for _, cmd := range d.commands {
		result = append(result, cmd)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

func (d *Dispatcher) Dispatch(ctx context.Context, request dto.CommandRequest, cmdName string) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate command request: %w", err)
	}

	cmd, exists := d.commands[cmdName]
	if !exists {
		return &UnknownCommandError{Command: cmdName}
	}

	handleErr := cmd.Handle(ctx, request)
	if handleErr != nil {
		return fmt.Errorf("handle command %q: %w", cmdName, handleErr)
	}
	return nil
}
