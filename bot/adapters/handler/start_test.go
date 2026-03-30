package handler_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command/mock"
	trackermock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker/mock"
)

func TestStartCommandHandler_NameAndDescription(t *testing.T) {
	t.Parallel()

	cmd := handler.NewStartCommandHandler(nil, nil, nil)

	require.Equal(t, "start", cmd.Name())
	require.NotEmpty(t, cmd.Description())
}

func TestStartCommandHandler_Handle(t *testing.T) {
	t.Parallel()

	sendErr := errors.New("send error")
	registerErr := errors.New("register error")

	tests := []struct {
		name          string
		request       dto.CommandRequest
		withTracker   bool
		setupTracker  func(m *trackermock.MockService)
		setupSender   func(m *mock.Sender)
		assertErrFunc func(t *testing.T, err error)
	}{
		{
			name:        "success",
			request:     dto.CommandRequest{Text: "/start", ChatID: 12345},
			withTracker: true,
			setupTracker: func(m *trackermock.MockService) {
				m.On("RegisterChat", testifymock.Anything, int64(12345)).Return(nil).Once()
			},
			setupSender: func(m *mock.Sender) {
				m.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 12345 && msg.Text != ""
				})).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErrFunc: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:        "register already exists is ignored",
			request:     dto.CommandRequest{Text: "/start", ChatID: 12345},
			withTracker: true,
			setupTracker: func(m *trackermock.MockService) {
				m.On("RegisterChat", testifymock.Anything, int64(12345)).
					Return(errors.New("already exists")).Once()
			},
			setupSender: func(m *mock.Sender) {
				m.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErrFunc: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:        "register error",
			request:     dto.CommandRequest{Text: "/start", ChatID: 12345},
			withTracker: true,
			setupTracker: func(m *trackermock.MockService) {
				m.On("RegisterChat", testifymock.Anything, int64(12345)).Return(registerErr).Once()
			},
			setupSender: nil,
			assertErrFunc: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorContains(t, err, "register chat")
			},
		},
		{
			name:        "send error",
			request:     dto.CommandRequest{Text: "/start", ChatID: 12345},
			withTracker: true,
			setupTracker: func(m *trackermock.MockService) {
				m.On("RegisterChat", testifymock.Anything, int64(12345)).Return(nil).Once()
			},
			setupSender: func(m *mock.Sender) {
				m.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, sendErr).Once()
			},
			assertErrFunc: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, sendErr)
			},
		},
		{
			name:        "invalid request",
			request:     dto.CommandRequest{Text: "", ChatID: 0},
			setupSender: nil,
			assertErrFunc: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSender := mock.NewSender(t)
			if tt.setupSender != nil {
				tt.setupSender(mockSender)
			}
			var trackerSvc *trackermock.MockService
			if tt.withTracker {
				trackerSvc = trackermock.NewMockService(t)
				if tt.setupTracker != nil {
					tt.setupTracker(trackerSvc)
				}
			}

			logger := slog.Default()
			cmd := handler.NewStartCommandHandler(trackerSvc, logger, mockSender)
			err := cmd.Handle(context.Background(), tt.request)
			tt.assertErrFunc(t, err)
		})
	}
}
