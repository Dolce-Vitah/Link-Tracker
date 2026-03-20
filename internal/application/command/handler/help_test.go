package handler_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/handler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
)

func TestHelpCommandHandler_NameAndDescription(t *testing.T) {
	t.Parallel()

	cmd := handler.NewHelpCommandHandler(nil, nil)

	assert.Equal(t, "help", cmd.Name())
	assert.NotEmpty(t, cmd.Description())
}

func TestHelpCommandHandler_Handle(t *testing.T) {
	t.Parallel()

	type mockBehavior func(sender *mock.Sender)
	sendErr := errors.New("send error")

	tests := []struct {
		name         string
		update       *tgbotapi.Update
		mockBehavior mockBehavior
		checkError   func(t *testing.T, err error)
	}{
		{
			name:   "success",
			update: newCommandUpdate("help"),
			mockBehavior: func(sender *mock.Sender) {
				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 12345 && msg.Text != ""
				})).Return(tgbotapi.Message{}, nil)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:   "send error",
			update: newCommandUpdate("help"),
			mockBehavior: func(sender *mock.Sender) {
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, sendErr)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, sendErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSender := mock.NewSender(t)
			tt.mockBehavior(mockSender)

			logger := slog.Default()
			cmd := handler.NewHelpCommandHandler(logger, mockSender)

			err := cmd.Handle(context.Background(), tt.update)
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
