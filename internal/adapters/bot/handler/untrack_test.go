package handler_test

import (
	"context"
	"errors"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/handler"
	trackermock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker/mock"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

func TestUntrackCommand_Handle(t *testing.T) {
	registerErr := errors.New("register error")
	removeErr := errors.New("remove error")
	sendErr := errors.New("send error")

	tests := []struct {
		name       string
		request    dto.CommandRequest
		setupMocks func(trackerMock *trackermock.MockService, sender *mock.Sender)
		assertErr  func(t *testing.T, err error)
	}{
		{
			name:    "invalid url",
			request: dto.CommandRequest{Text: "/untrack xxx", ChatID: 9},
			setupMocks: func(trackerMock *trackermock.MockService, sender *mock.Sender) {
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:    "register error",
			request: dto.CommandRequest{Text: "/untrack https://a.b", ChatID: 9},
			setupMocks: func(trackerMock *trackermock.MockService, sender *mock.Sender) {
				trackerMock.On("RegisterChat", testifymock.Anything, int64(9)).Return(registerErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name:    "not found",
			request: dto.CommandRequest{Text: "/untrack https://a.b", ChatID: 9},
			setupMocks: func(trackerMock *trackermock.MockService, sender *mock.Sender) {
				trackerMock.On("RegisterChat", testifymock.Anything, int64(9)).Return(nil).Once()
				trackerMock.On("RemoveLink", testifymock.Anything, int64(9), api.RemoveLinkRequest{Link: "https://a.b"}).
					Return(api.LinkResponse{}, errors.New(tracker.ErrNotFound.Error())).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:    "remove error",
			request: dto.CommandRequest{Text: "/untrack https://a.b", ChatID: 9},
			setupMocks: func(trackerMock *trackermock.MockService, sender *mock.Sender) {
				trackerMock.On("RegisterChat", testifymock.Anything, int64(9)).Return(nil).Once()
				trackerMock.On("RemoveLink", testifymock.Anything, int64(9), api.RemoveLinkRequest{Link: "https://a.b"}).
					Return(api.LinkResponse{}, removeErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, removeErr) },
		},
		{
			name:    "success",
			request: dto.CommandRequest{Text: "/untrack https://a.b", ChatID: 9},
			setupMocks: func(trackerMock *trackermock.MockService, sender *mock.Sender) {
				trackerMock.On("RegisterChat", testifymock.Anything, int64(9)).Return(nil).Once()
				trackerMock.On("RemoveLink", testifymock.Anything, int64(9), api.RemoveLinkRequest{Link: "https://a.b"}).
					Return(api.LinkResponse{}, nil).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:    "send error after remove",
			request: dto.CommandRequest{Text: "/untrack https://a.b", ChatID: 9},
			setupMocks: func(trackerMock *trackermock.MockService, sender *mock.Sender) {
				trackerMock.On("RegisterChat", testifymock.Anything, int64(9)).Return(nil).Once()
				trackerMock.On("RemoveLink", testifymock.Anything, int64(9), api.RemoveLinkRequest{Link: "https://a.b"}).
					Return(api.LinkResponse{}, nil).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, sendErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, sendErr) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			trackerMock := trackermock.NewMockService(t)
			sender := mock.NewSender(t)
			tt.setupMocks(trackerMock, sender)

			cmd := handler.NewUntrackCommandHandler(trackerMock, nil, sender)
			err := cmd.Handle(context.Background(), tt.request)
			tt.assertErr(t, err)
		})
	}
}
