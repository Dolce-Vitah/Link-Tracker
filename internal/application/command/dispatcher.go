package command

import (
	"context"
	"errors"
	"fmt"
	"sort"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var ErrUnknownCommand = errors.New("unknown command")

type UnknownCommandError struct {
	Command string
}

func (e *UnknownCommandError) Error() string {
	return fmt.Sprintf("%s: %s", ErrUnknownCommand.Error(), e.Command)
}

func (e *UnknownCommandError) Unwrap() error {
	return ErrUnknownCommand
}

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

func (d *Dispatcher) Dispatch(ctx context.Context, update *tgbotapi.Update) error {
	if update == nil || update.Message == nil {
		return errors.New("dispatch command: invalid update")
	}

	cmdName := update.Message.Command()
	cmd, exists := d.commands[cmdName]
	if !exists {
		return &UnknownCommandError{Command: cmdName}
	}

	handleErr := cmd.Handle(ctx, update)
	if handleErr != nil {
		return fmt.Errorf("handle command %q: %w", cmdName, handleErr)
	}

	return nil
}
