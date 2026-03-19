package handler_test

import (
	"context"
	"errors"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/handler"
	handlermock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/handler/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dialog"
	trackermock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
)

func TestTrackDialogHandler_Handle(t *testing.T) {
	addErr := errors.New("add error")
	sendErr := errors.New("send error")

	tests := []struct {
		name       string
		request    dto.DialogRequest
		setupMocks func(tracker *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender)
		assertErr  func(t *testing.T, err error)
	}{
		{
			name:    "invalid request ignored",
			request: dto.DialogRequest{},
			setupMocks: func(tracker *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {},
			assertErr:  func(t *testing.T, err error) { assert.NoError(t, err) },
		},
		{
			name: "no session found",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 100}, Text: "hello"},
			}},
			setupMocks: func(tracker *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				sessions.On("Get", int64(100)).Return(dialog.Session{}, false).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.NoError(t, err) },
		},
		{
			name: "awaiting url invalid",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 102}, Text: "bad-url"},
			}},
			setupMocks: func(tracker *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				sessions.On("Get", int64(102)).Return(dialog.Session{State: dialog.StateAwaitingURL}, true).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.NoError(t, err) },
		},
		{
			name: "awaiting url success",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 100}, Text: "https://github.com/user/repo"},
			}},
			setupMocks: func(tracker *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				sessions.On("Get", int64(100)).Return(dialog.Session{State: dialog.StateAwaitingURL}, true).Once()
				sessions.On("Set", int64(100), dialog.Session{
					State:      dialog.StateAwaitingTags,
					PendingURL: "https://github.com/user/repo",
				}).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.NoError(t, err) },
		},
		{
			name: "awaiting tags success",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 100}, Text: "work,docs"},
			}},
			setupMocks: func(tracker *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				sessions.On("Get", int64(100)).Return(dialog.Session{
					State:      dialog.StateAwaitingTags,
					PendingURL: "https://github.com/user/repo",
				}, true).Once()
				tracker.On("AddLink", testifymock.Anything, int64(100), api.AddLinkRequest{
					Link: "https://github.com/user/repo",
					Tags: []string{"work", "docs"},
				}).Return(api.LinkResponse{}, nil).Once()
				sessions.On("Clear", int64(100)).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.NoError(t, err) },
		},
		{
			name: "awaiting tags duplicate link",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 103}, Text: "work"},
			}},
			setupMocks: func(tracker *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				sessions.On("Get", int64(103)).Return(dialog.Session{
					State:      dialog.StateAwaitingTags,
					PendingURL: "https://github.com/user/repo",
				}, true).Once()
				tracker.On("AddLink", testifymock.Anything, int64(103), api.AddLinkRequest{
					Link: "https://github.com/user/repo",
					Tags: []string{"work"},
				}).Return(api.LinkResponse{}, errors.New("already exists")).Once()
				sessions.On("Clear", int64(103)).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.NoError(t, err) },
		},
		{
			name: "awaiting tags add error",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 104}, Text: "work"},
			}},
			setupMocks: func(tracker *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				sessions.On("Get", int64(104)).Return(dialog.Session{
					State:      dialog.StateAwaitingTags,
					PendingURL: "https://github.com/user/repo",
				}, true).Once()
				tracker.On("AddLink", testifymock.Anything, int64(104), api.AddLinkRequest{
					Link: "https://github.com/user/repo",
					Tags: []string{"work"},
				}).Return(api.LinkResponse{}, addErr).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.ErrorIs(t, err, addErr) },
		},
		{
			name: "command during dialog cancels",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 101}, Text: "/help",
					Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
				},
			}},
			setupMocks: func(tracker *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				sessions.On("Get", int64(101)).Return(dialog.Session{State: dialog.StateAwaitingURL}, true).Once()
				sessions.On("Clear", int64(101)).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.NoError(t, err) },
		},
		{
			name: "command cancel keeps dialog",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 101}, Text: "/cancel",
					Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 7}},
				},
			}},
			setupMocks: func(tracker *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				sessions.On("Get", int64(101)).Return(dialog.Session{State: dialog.StateAwaitingURL}, true).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.NoError(t, err) },
		},
		{
			name: "send error while cancelling by command",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 101}, Text: "/help",
					Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
				},
			}},
			setupMocks: func(tracker *trackermock.MockService, sessions *handlermock.MockSessionStore, sender *mock.Sender) {
				sessions.On("Get", int64(101)).Return(dialog.Session{State: dialog.StateAwaitingURL}, true).Once()
				sessions.On("Clear", int64(101)).Once()
				sender.EXPECT().Send(testifymock.Anything).Return(tgbotapi.Message{}, sendErr).Once()
			},
			assertErr: func(t *testing.T, err error) { assert.ErrorIs(t, err, sendErr) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tracker := trackermock.NewMockService(t)
			sessions := handlermock.NewMockSessionStore(t)
			sender := mock.NewSender(t)
			tt.setupMocks(tracker, sessions, sender)

			dialogHandler := handler.NewTrackDialogHandler(tracker, sessions, sender, nil)
			err := dialogHandler.Handle(context.Background(), tt.request)
			tt.assertErr(t, err)
		})
	}
}
