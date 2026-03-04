package command

import (
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
	handlers map[string]Handler
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		commands: make(map[string]Command),
		handlers: make(map[string]Handler),
	}
}

func (d *Dispatcher) Register(cmd Command, handler Handler) {
	name := cmd.Name()
	d.commands[name] = cmd
	d.handlers[name] = handler
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

func (d *Dispatcher) Dispatch(update *tgbotapi.Update, sender Sender) error {
	if update == nil || update.Message == nil || !update.Message.IsCommand() {
		return nil
	}

	cmdName := update.Message.Command()
	handler, exists := d.handlers[cmdName]
	if !exists {
		return &UnknownCommandError{Command: cmdName}
	}

	return handler.Handle(update, sender)
}
