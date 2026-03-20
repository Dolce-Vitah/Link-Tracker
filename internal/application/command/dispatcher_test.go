package command_test

import (
	"context"
	"errors"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/dto"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
)

func TestDispatcher_Dispatch(t *testing.T) {
	type mockBehavior func(cmd *mock.Command)
	handleErr := errors.New("handle error")

	tests := []struct {
		name         string
		commandName  string
		text         string
		chatID       int64
		inputCmdName string
		mockBehavior mockBehavior
		checkError   func(t *testing.T, err error)
	}{
		{
			name:         "known command success",
			commandName:  "start",
			text:         "/start",
			chatID:       12345,
			inputCmdName: "start",
			mockBehavior: func(cmd *mock.Command) {
				cmd.EXPECT().Name().Return("start")
				cmd.EXPECT().Handle(testifymock.Anything, dto.CommandRequest{
					Text:   "/start",
					ChatID: int64(12345),
				}).Return(nil)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:         "known command error",
			commandName:  "start",
			text:         "/start",
			chatID:       12345,
			inputCmdName: "start",
			mockBehavior: func(cmd *mock.Command) {
				cmd.EXPECT().Name().Return("start")
				cmd.EXPECT().Handle(testifymock.Anything, dto.CommandRequest{
					Text:   "/start",
					ChatID: int64(12345),
				}).Return(handleErr)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, handleErr)
			},
		},
		{
			name:         "unknown command",
			commandName:  "",
			text:         "/unknown",
			chatID:       12345,
			inputCmdName: "unknown",
			mockBehavior: func(cmd *mock.Command) {
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
				tt.mockBehavior(mockCmd)
				dispatcher.Register(mockCmd)
			}

			err := dispatcher.Dispatch(context.Background(), dto.CommandRequest{
				Text:   tt.text,
				ChatID: tt.chatID,
			}, tt.inputCmdName)
			tt.checkError(t, err)
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
		require.Len(t, commands, 2)
		require.Equal(t, "a_command", commands[0].Name())
		require.Equal(t, "b_command", commands[1].Name())
	})
}
