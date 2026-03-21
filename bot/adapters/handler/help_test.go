package handler_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command/mock"
)

func TestHelpCommandHandler_NameAndDescription(t *testing.T) {
	t.Parallel()

	cmd := handler.NewHelpCommandHandler(nil, nil)

	require.Equal(t, "help", cmd.Name())
	require.NotEmpty(t, cmd.Description())
}

func TestHelpCommandHandler_Handle(t *testing.T) {
	t.Parallel()

	type mockBehavior func(sender *mock.Sender)
	sendErr := errors.New("send error")

	tests := []struct {
		name         string
		request      dto.CommandRequest
		mockBehavior mockBehavior
		checkError   func(t *testing.T, err error)
	}{
		{
			name:    "success",
			request: dto.CommandRequest{Text: "/help", ChatID: 12345},
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
			name:    "send error",
			request: dto.CommandRequest{Text: "/help", ChatID: 12345},
			mockBehavior: func(sender *mock.Sender) {
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, sendErr)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, sendErr)
			},
		},
		{
			name:    "invalid request",
			request: dto.CommandRequest{},
			mockBehavior: func(_ *mock.Sender) {
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
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

			err := cmd.Handle(context.Background(), tt.request)
			tt.checkError(t, err)
		})
	}
}
