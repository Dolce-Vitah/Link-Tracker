package chat_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/chat"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
	commandmock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
	trackermock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker/mock"
)

func TestStartCommandHandler_Handle(t *testing.T) {
	t.Parallel()

	sendErr := errors.New("send error")
	registerErr := errors.New("register error")

	tests := []struct {
		name          string
		request       dto.CommandRequest
		withTracker   bool
		setupTracker  func(m *trackermock.MockClient)
		setupSender   func(m *commandmock.Sender)
		assertErrFunc func(t *testing.T, err error)
	}{
		{
			name:        "success",
			request:     dto.CommandRequest{Text: "/start", ChatID: 12345},
			withTracker: true,
			setupTracker: func(m *trackermock.MockClient) {
				m.EXPECT().RegisterChat(testifymock.Anything, int64(12345)).Return(nil).Once()
			},
			setupSender: func(m *commandmock.Sender) {
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
			setupTracker: func(m *trackermock.MockClient) {
				m.EXPECT().RegisterChat(testifymock.Anything, int64(12345)).
					Return(fmt.Errorf("wrapped: %w", tracker.ErrAlreadyExists)).Once()
			},
			setupSender: func(m *commandmock.Sender) {
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
			name:        "register error",
			request:     dto.CommandRequest{Text: "/start", ChatID: 12345},
			withTracker: true,
			setupTracker: func(m *trackermock.MockClient) {
				m.EXPECT().RegisterChat(testifymock.Anything, int64(12345)).Return(registerErr).Once()
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
			setupTracker: func(m *trackermock.MockClient) {
				m.EXPECT().RegisterChat(testifymock.Anything, int64(12345)).Return(nil).Once()
			},
			setupSender: func(m *commandmock.Sender) {
				m.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 12345 && msg.Text != ""
				})).Return(tgbotapi.Message{}, sendErr).Once()
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

			mockSender := commandmock.NewSender(t)
			if tt.setupSender != nil {
				tt.setupSender(mockSender)
			}

			var trackerSvc *trackermock.MockClient
			if tt.withTracker {
				trackerSvc = trackermock.NewMockClient(t)
				if tt.setupTracker != nil {
					tt.setupTracker(trackerSvc)
				}
			}

			logger := slog.Default()
			chatHandler := chat.NewChatHandler(trackerSvc, mockSender, logger)
			cmd := chat.NewStartCommand(chatHandler)

			err := cmd.Handle(context.Background(), tt.request)
			tt.assertErrFunc(t, err)
		})
	}
}

func TestHelpCommandHandler_Handle(t *testing.T) {
	t.Parallel()

	type mockBehavior func(sender *commandmock.Sender)
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
			mockBehavior: func(sender *commandmock.Sender) {
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
			mockBehavior: func(sender *commandmock.Sender) {
				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 12345 && msg.Text != ""
				})).Return(tgbotapi.Message{}, sendErr)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, sendErr)
			},
		},
		{
			name:    "invalid request",
			request: dto.CommandRequest{},
			mockBehavior: func(_ *commandmock.Sender) {
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

			mockSender := commandmock.NewSender(t)
			tt.mockBehavior(mockSender)

			logger := slog.Default()
			chatHandler := chat.NewChatHandler(nil, mockSender, logger)
			cmd := chat.NewHelpCommand(chatHandler)

			err := cmd.Handle(context.Background(), tt.request)
			tt.checkError(t, err)
		})
	}
}
