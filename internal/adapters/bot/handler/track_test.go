package handler_test

import (
	"context"
	"errors"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/handler"
	handlermock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/handler/mock"
	trackermock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker/mock"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
)

func TestTrackCommand_Handle(t *testing.T) {
	registerErr := errors.New("register error")
	sendErr := errors.New("send error")

	tests := []struct {
		name       string
		request    dto.CommandRequest
		setupMocks func(trackerMock *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender)
		assertErr  func(t *testing.T, err error)
	}{
		{
			name:    "success",
			request: dto.CommandRequest{Text: "/track", ChatID: 1},
			setupMocks: func(trackerMock *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				trackerMock.On("RegisterChat", testifymock.Anything, int64(1)).Return(nil).Once()
				sessions.On("Set", int64(1), testifymock.Anything).Once()
				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 1
				})).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.NoError(t, err) },
		},
		{
			name:    "register error",
			request: dto.CommandRequest{Text: "/track", ChatID: 1},
			setupMocks: func(trackerMock *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				trackerMock.On("RegisterChat", testifymock.Anything, int64(1)).Return(registerErr).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.Error(t, err) },
		},
		{
			name:    "send error",
			request: dto.CommandRequest{Text: "/track", ChatID: 1},
			setupMocks: func(trackerMock *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				trackerMock.On("RegisterChat", testifymock.Anything, int64(1)).Return(nil).Once()
				sessions.On("Set", int64(1), testifymock.Anything).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, sendErr).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.ErrorIs(t, err, sendErr) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			trackerMock := trackermock.NewMockService(t)
			sessions := handlermock.NewMockSessionStore(t)
			sender := mock.NewSender(t)
			tt.setupMocks(trackerMock, sessions, sender)

			cmd := handler.NewTrackCommandHandler(sessions, trackerMock, nil, sender)
			err := cmd.Handle(context.Background(), tt.request)
			tt.assertErr(t, err)
		})
	}
}
