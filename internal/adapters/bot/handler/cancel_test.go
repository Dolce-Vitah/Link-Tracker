package handler_test

import (
	"context"
	"errors"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/handler"
	handlermock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/handler/mock"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
)

func TestCancelCommandHandler_Handle(t *testing.T) {
	sendErr := errors.New("send error")

	tests := []struct {
		name       string
		request    dto.CommandRequest
		setupMocks func(sessions *handlermock.MockSessionStore, sender *mock.Sender)
		assertErr  func(t *testing.T, err error)
	}{
		{
			name:    "success",
			request: dto.CommandRequest{Text: "/cancel", ChatID: 1},
			setupMocks: func(sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				sessions.On("Clear", int64(1)).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:    "send error",
			request: dto.CommandRequest{Text: "/cancel", ChatID: 1},
			setupMocks: func(sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				sessions.On("Clear", int64(1)).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, sendErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, sendErr) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sessions := handlermock.NewMockSessionStore(t)
			sender := mock.NewSender(t)
			tt.setupMocks(sessions, sender)

			cmd := handler.NewCancelCommandHandler(sessions, nil, sender)
			err := cmd.Handle(context.Background(), tt.request)
			tt.assertErr(t, err)
		})
	}
}
