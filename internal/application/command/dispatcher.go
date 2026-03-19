package command

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

func (d *Dispatcher) Dispatch(ctx context.Context, text string, chatID int64, cmdName string) error {
	cmd, exists := d.commands[cmdName]
	if !exists {
		return &UnknownCommandError{Command: cmdName}
	}

	return cmd.Handle(ctx, text, chatID)
}
