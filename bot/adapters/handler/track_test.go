package handler_test

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command/mock"
	trackermock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/repository"
	repositorymock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/repository/mock"
)

func TestTrackCommand_Handle(t *testing.T) {
	t.Parallel()

	registerErr := errors.New("register error")
	sendErr := errors.New("send error")

	tests := []struct {
		name       string
		request    dto.CommandRequest
		setupMocks func(trackerMock *trackermock.MockService, sessions *repositorymock.MockSessionRepository, sender *mock.Sender)
		assertErr  func(t *testing.T, err error)
	}{
		{
			name:    "success",
			request: dto.CommandRequest{Text: "/track", ChatID: 1},
			setupMocks: func(trackerMock *trackermock.MockService, sessions *repositorymock.MockSessionRepository, sender *mock.Sender) {
				trackerMock.On("RegisterChat", testifymock.Anything, int64(1)).Return(nil).Once()
				sessions.On("Set", int64(1), repository.Session{State: repository.StateAwaitingURL}).Once()
				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 1
				})).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:    "register error",
			request: dto.CommandRequest{Text: "/track", ChatID: 1},
			setupMocks: func(trackerMock *trackermock.MockService, _ *repositorymock.MockSessionRepository, _ *mock.Sender) {
				trackerMock.On("RegisterChat", testifymock.Anything, int64(1)).Return(registerErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name:    "send error",
			request: dto.CommandRequest{Text: "/track", ChatID: 1},
			setupMocks: func(trackerMock *trackermock.MockService, sessions *repositorymock.MockSessionRepository, sender *mock.Sender) {
				trackerMock.On("RegisterChat", testifymock.Anything, int64(1)).Return(nil).Once()
				sessions.On("Set", int64(1), repository.Session{State: repository.StateAwaitingURL}).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, sendErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, sendErr) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			trackerMock := trackermock.NewMockService(t)
			sessions := repositorymock.NewMockSessionRepository(t)
			sender := mock.NewSender(t)
			tt.setupMocks(trackerMock, sessions, sender)

			cmd := link.NewTrackCommandHandler(sessions, trackerMock, nil, sender)
			err := cmd.Handle(context.Background(), tt.request)
			tt.assertErr(t, err)
		})
	}
}
