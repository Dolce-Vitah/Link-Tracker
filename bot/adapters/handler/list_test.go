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
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

func TestListCommand_Handle(t *testing.T) {
	t.Parallel()

	listErr := errors.New("list error")
	registerErr := errors.New("register error")
	sendErr := errors.New("send error")

	tests := []struct {
		name       string
		request    dto.CommandRequest
		setupMocks func(tracker *trackermock.MockService, sender *mock.Sender)
		assertErr  func(t *testing.T, err error)
	}{
		{
			name:    "empty list",
			request: dto.CommandRequest{Text: "/list", ChatID: 7},
			setupMocks: func(tracker *trackermock.MockService, sender *mock.Sender) {
				tracker.On("RegisterChat", testifymock.Anything, int64(7)).Return(nil).Once()
				tracker.On("ListLinks", testifymock.Anything, int64(7)).
					Return(api.ListLinksResponse{Links: nil, Size: 0}, nil).Once()
				sender.EXPECT().
					Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
						return msg.Text == "Список отслеживаемых ссылок пуст."
					})).
					Return(tgbotapi.Message{}, nil).
					Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:    "filter by tag",
			request: dto.CommandRequest{Text: "/list work", ChatID: 10},
			setupMocks: func(tracker *trackermock.MockService, sender *mock.Sender) {
				tracker.On("RegisterChat", testifymock.Anything, int64(10)).Return(nil).Once()
				tracker.On("ListLinks", testifymock.Anything, int64(10)).
					Return(api.ListLinksResponse{Links: []api.LinkResponse{
						{ID: 1, URL: "https://github.com/a/b", Tags: []string{"work"}},
						{ID: 2, URL: "https://stackoverflow.com/questions/1/x", Tags: []string{"misc"}},
					}, Size: 2}, nil).Once()
				sender.EXPECT().
					Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
						return msg.ChatID == 10 && msg.Text != "" && msg.Text != "Список отслеживаемых ссылок пуст."
					})).
					Return(tgbotapi.Message{}, nil).
					Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:    "register error",
			request: dto.CommandRequest{Text: "/list", ChatID: 10},
			setupMocks: func(tracker *trackermock.MockService, _ *mock.Sender) {
				tracker.On("RegisterChat", testifymock.Anything, int64(10)).Return(registerErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name:    "list error",
			request: dto.CommandRequest{Text: "/list", ChatID: 10},
			setupMocks: func(tracker *trackermock.MockService, _ *mock.Sender) {
				tracker.On("RegisterChat", testifymock.Anything, int64(10)).Return(nil).Once()
				tracker.On("ListLinks", testifymock.Anything, int64(10)).Return(api.ListLinksResponse{}, listErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, listErr) },
		},
		{
			name:    "send error",
			request: dto.CommandRequest{Text: "/list", ChatID: 10},
			setupMocks: func(tracker *trackermock.MockService, sender *mock.Sender) {
				tracker.On("RegisterChat", testifymock.Anything, int64(10)).Return(nil).Once()
				tracker.On("ListLinks", testifymock.Anything, int64(10)).
					Return(api.ListLinksResponse{Links: nil, Size: 0}, nil).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, sendErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, sendErr) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tracker := trackermock.NewMockService(t)
			sender := mock.NewSender(t)
			tt.setupMocks(tracker, sender)

			cmd := link.NewListCommandHandler(tracker, nil, sender)
			err := cmd.Handle(context.Background(), tt.request)
			tt.assertErr(t, err)
		})
	}
}
