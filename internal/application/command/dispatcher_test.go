package command_test

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
)

func TestDispatcher_Dispatch(t *testing.T) {
	t.Parallel()

	type mockBehavior func(cmd *mock.Command, update *tgbotapi.Update)
	handleErr := errors.New("handle error")

	tests := []struct {
		name         string
		commandName  string
		update       *tgbotapi.Update
		mockBehavior mockBehavior
		checkError   func(t *testing.T, err error)
	}{
		{
			name:        "known command success",
			commandName: "start",
			update:      newCommandUpdate("start"),
			mockBehavior: func(cmd *mock.Command, update *tgbotapi.Update) {
				cmd.EXPECT().Name().Return("start")
				cmd.EXPECT().Handle(context.Background(), update).Return(nil)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:        "known command error",
			commandName: "start",
			update:      newCommandUpdate("start"),
			mockBehavior: func(cmd *mock.Command, update *tgbotapi.Update) {
				cmd.EXPECT().Name().Return("start")
				cmd.EXPECT().Handle(context.Background(), update).Return(handleErr)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, handleErr)
			},
		},
		{
			name:        "unknown command",
			commandName: "",
			update:      newCommandUpdate("unknown"),
			mockBehavior: func(_ *mock.Command, _ *tgbotapi.Update) {
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, command.ErrUnknownCommand)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dispatcher := command.NewDispatcher()

			if tt.commandName != "" {
				mockCmd := mock.NewCommand(t)
				tt.mockBehavior(mockCmd, tt.update)
				dispatcher.Register(mockCmd)
			}

			err := dispatcher.Dispatch(context.Background(), tt.update)
			tt.checkError(t, err)
		})
	}
}

func newCommandUpdate(commandName string) *tgbotapi.Update {
	commandText := "/" + commandName

	return &tgbotapi.Update{
		Message: &tgbotapi.Message{
			Text: commandText,
			Chat: &tgbotapi.Chat{ID: 12345},
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: len(commandText)},
			},
		},
	}
}

func TestDispatcher_Commands(t *testing.T) {
	t.Parallel()

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
		require.Len(t, commands, 2)
		assert.Equal(t, "a_command", commands[0].Name())
		assert.Equal(t, "b_command", commands[1].Name())
	})
}
