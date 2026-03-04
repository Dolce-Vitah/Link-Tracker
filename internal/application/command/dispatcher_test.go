package command_test

import (
	"errors"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	testifymock "github.com/stretchr/testify/mock"
)

func TestDispatcher_Dispatch(t *testing.T) {
	t.Run("known command", func(t *testing.T) {
		t.Parallel()

		var (
			dispatcher = command.NewDispatcher()
			mockSender = mock.NewSender(t)
		)
		dispatcher.Register(&command.StartCommand{}, &command.StartHandler{})

		mockSender.On("Send", testifymock.AnythingOfType("tgbotapi.MessageConfig")).
			Return(tgbotapi.Message{}, nil).
			Once()

		err := dispatcher.Dispatch(newCommandUpdate("start"), mockSender)
		if err != nil {
			t.Fatalf("dispatch should succeed: %v", err)
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		t.Parallel()

		var (
			dispatcher = command.NewDispatcher()
			mockSender = mock.NewSender(t)
		)

		err := dispatcher.Dispatch(newCommandUpdate("abracadabra"), mockSender)
		if !errors.Is(err, command.ErrUnknownCommand) {
			t.Fatalf("expected ErrUnknownCommand, got: %v", err)
		}
	})

	t.Run("non command update", func(t *testing.T) {
		t.Parallel()

		var (
			dispatcher = command.NewDispatcher()
			mockSender = mock.NewSender(t)
		)

		update := &tgbotapi.Update{
			Message: &tgbotapi.Message{
				Text: "hello",
				Chat: &tgbotapi.Chat{ID: 12345},
				From: &tgbotapi.User{UserName: "test_user"},
			},
		}

		err := dispatcher.Dispatch(update, mockSender)
		if err != nil {
			t.Fatalf("dispatch should ignore non-command update: %v", err)
		}
	})

	t.Run("nil update", func(t *testing.T) {
		t.Parallel()

		var (
			dispatcher = command.NewDispatcher()
			mockSender = mock.NewSender(t)
		)

		err := dispatcher.Dispatch(nil, mockSender)
		if err != nil {
			t.Fatalf("dispatch should ignore nil update: %v", err)
		}
	})

	t.Run("commands are returned in stable order", func(t *testing.T) {
		t.Parallel()

		var dispatcher = command.NewDispatcher()
		dispatcher.Register(&command.StartCommand{}, &command.StartHandler{})
		dispatcher.Register(&command.HelpCommand{}, &command.HelpHandler{})

		commands := dispatcher.Commands()
		if len(commands) != 2 {
			t.Fatalf("expected 2 commands, got: %d", len(commands))
		}
		if commands[0].Name() != "help" || commands[1].Name() != "start" {
			t.Fatalf("unexpected command order: %s, %s", commands[0].Name(), commands[1].Name())
		}
	})
}

func newCommandUpdate(cmd string) *tgbotapi.Update {
	return &tgbotapi.Update{
		Message: &tgbotapi.Message{
			Text: "/" + cmd,
			Chat: &tgbotapi.Chat{ID: 12345},
			From: &tgbotapi.User{
				ID:       98765,
				UserName: "test_user",
			},
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: len("/" + cmd)},
			},
		},
	}
}
