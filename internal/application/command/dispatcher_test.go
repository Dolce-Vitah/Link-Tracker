package command_test

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
)

func TestDispatcher_Dispatch(t *testing.T) {
	type mockBehavior func(cmd *mock.Command, sender *mock.Sender)

	tests := []struct {
		name          string
		commandName   string
		update        *tgbotapi.Update
		mockBehavior  mockBehavior
		expectedError error
	}{
		{
			name:        "known command success",
			commandName: "start",
			update:      newCommandUpdate("start"),
			mockBehavior: func(cmd *mock.Command, sender *mock.Sender) {
				cmd.EXPECT().Name().Return("start")
				cmd.EXPECT().Handle(testifymock.Anything, testifymock.AnythingOfType("*tgbotapi.Update"), sender).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:        "known command error",
			commandName: "start",
			update:      newCommandUpdate("start"),
			mockBehavior: func(cmd *mock.Command, sender *mock.Sender) {
				cmd.EXPECT().Name().Return("start")
				cmd.EXPECT().Handle(testifymock.Anything, testifymock.AnythingOfType("*tgbotapi.Update"), sender).Return(errors.New("handle error"))
			},
			expectedError: errors.New("handle error"),
		},
		{
			name:        "unknown command",
			commandName: "",
			update:      newCommandUpdate("unknown"),
			mockBehavior: func(cmd *mock.Command, sender *mock.Sender) {
			},
			expectedError: command.ErrUnknownCommand,
		},
		{
			name:        "non command update",
			commandName: "",
			update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					Text: "just text",
					Chat: &tgbotapi.Chat{ID: 1},
				},
			},
			mockBehavior: func(cmd *mock.Command, sender *mock.Sender) {
			},
			expectedError: nil,
		},
		{
			name:        "nil update",
			commandName: "",
			update:      nil,
			mockBehavior: func(cmd *mock.Command, sender *mock.Sender) {
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				dispatcher = command.NewDispatcher()
				mockSender = mock.NewSender(t)
			)

			if tt.commandName != "" {
				mockCmd := mock.NewCommand(t)
				tt.mockBehavior(mockCmd, mockSender)
				dispatcher.Register(mockCmd)
			}

			err := dispatcher.Dispatch(context.Background(), tt.update, mockSender)

			if tt.expectedError != nil {
				if assert.Error(t, err) {
					if err.Error() != tt.expectedError.Error() && !errors.Is(err, tt.expectedError) {
						t.Errorf("expected error %v, got %v", tt.expectedError, err)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDispatcher_Commands(t *testing.T) {
	t.Run("commands are returned in stable order", func(t *testing.T) {
		t.Parallel()

		dispatcher := command.NewDispatcher()

		cmd1 := mock.NewCommand(t)
		cmd1.EXPECT().Name().Return("b_command")

		cmd2 := mock.NewCommand(t)
		cmd2.EXPECT().Name().Return("a_command")

		dispatcher.Register(cmd1)
		dispatcher.Register(cmd2)

		commands := dispatcher.Commands()
		assert.Equal(t, 2, len(commands))
		assert.Equal(t, "a_command", commands[0].Name())
		assert.Equal(t, "b_command", commands[1].Name())
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
