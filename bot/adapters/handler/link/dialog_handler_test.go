package link_test

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/gateway/repository"
	repositorymock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/gateway/repository/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/link"
	commandmock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command/mock"
	trackermock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
)

func TestTrackDialogHandler_Handle(t *testing.T) {
	t.Parallel()

	addErr := errors.New("add error")
	sendErr := errors.New("send error")

	tests := []struct {
		name       string
		request    dto.DialogRequest
		setupMocks func(tracker *trackermock.MockClient, sessions *repositorymock.MockSessionRepository, sender *commandmock.Sender)
		assertErr  func(t *testing.T, err error)
	}{
		{
			name:    "invalid request ignored",
			request: dto.DialogRequest{},
			setupMocks: func(_ *trackermock.MockClient, _ *repositorymock.MockSessionRepository, _ *commandmock.Sender) {
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "no session found",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 100}, Text: "hello"},
			}},
			setupMocks: func(_ *trackermock.MockClient, sessions *repositorymock.MockSessionRepository, _ *commandmock.Sender) {
				sessions.EXPECT().Get(int64(100)).Return(repository.Session{}, false).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "awaiting url invalid",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 102}, Text: "bad-url"},
			}},
			setupMocks: func(_ *trackermock.MockClient, sessions *repositorymock.MockSessionRepository, sender *commandmock.Sender) {
				sessions.EXPECT().Get(int64(102)).Return(repository.Session{State: repository.StateAwaitingURL}, true).Once()
				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 102 && msg.Text != ""
				})).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "awaiting url success",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 100}, Text: "https://github.com/user/repo"},
			}},
			setupMocks: func(_ *trackermock.MockClient, sessions *repositorymock.MockSessionRepository, sender *commandmock.Sender) {
				sessions.EXPECT().Get(int64(100)).Return(repository.Session{State: repository.StateAwaitingURL}, true).Once()
				sessions.EXPECT().Set(int64(100), repository.Session{
					State:      repository.StateAwaitingTags,
					PendingURL: "https://github.com/user/repo",
				}).Once()
				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 100 && msg.Text != ""
				})).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "awaiting tags success",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 100}, Text: "work,docs"},
			}},
			setupMocks: func(tracker *trackermock.MockClient, sessions *repositorymock.MockSessionRepository, sender *commandmock.Sender) {
				sessions.EXPECT().Get(int64(100)).Return(repository.Session{
					State:      repository.StateAwaitingTags,
					PendingURL: "https://github.com/user/repo",
				}, true).Once()
				tracker.EXPECT().AddLink(testifymock.Anything, int64(100), trackerapi.AddLinkRequest{
					Link: "https://github.com/user/repo",
					Tags: []string{"work", "docs"},
				}).Return(trackerapi.LinkResponse{}, nil).Once()
				sessions.EXPECT().Clear(int64(100)).Once()
				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 100 && msg.Text != ""
				})).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "awaiting tags duplicate link",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 103}, Text: "work"},
			}},
			setupMocks: func(tracker *trackermock.MockClient, sessions *repositorymock.MockSessionRepository, sender *commandmock.Sender) {
				sessions.EXPECT().Get(int64(103)).Return(repository.Session{
					State:      repository.StateAwaitingTags,
					PendingURL: "https://github.com/user/repo",
				}, true).Once()
				tracker.EXPECT().AddLink(testifymock.Anything, int64(103), trackerapi.AddLinkRequest{
					Link: "https://github.com/user/repo",
					Tags: []string{"work"},
				}).Return(trackerapi.LinkResponse{}, errors.New("already exists")).Once()
				sessions.EXPECT().Clear(int64(103)).Once()
				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 103 && msg.Text != ""
				})).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "awaiting tags add error",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 104}, Text: "work"},
			}},
			setupMocks: func(tracker *trackermock.MockClient, sessions *repositorymock.MockSessionRepository, _ *commandmock.Sender) {
				sessions.EXPECT().Get(int64(104)).Return(repository.Session{
					State:      repository.StateAwaitingTags,
					PendingURL: "https://github.com/user/repo",
				}, true).Once()
				tracker.EXPECT().AddLink(testifymock.Anything, int64(104), trackerapi.AddLinkRequest{
					Link: "https://github.com/user/repo",
					Tags: []string{"work"},
				}).Return(trackerapi.LinkResponse{}, addErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, addErr) },
		},
		{
			name: "command during dialog cancels",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 101}, Text: "/help",
					Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
				},
			}},
			setupMocks: func(_ *trackermock.MockClient, sessions *repositorymock.MockSessionRepository, sender *commandmock.Sender) {
				sessions.EXPECT().Get(int64(101)).Return(repository.Session{State: repository.StateAwaitingURL}, true).Once()
				sessions.EXPECT().Clear(int64(101)).Once()
				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 101 && msg.Text != ""
				})).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "command cancel keeps dialog",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 101}, Text: "/cancel",
					Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 7}},
				},
			}},
			setupMocks: func(_ *trackermock.MockClient, sessions *repositorymock.MockSessionRepository, _ *commandmock.Sender) {
				sessions.EXPECT().Get(int64(101)).Return(repository.Session{State: repository.StateAwaitingURL}, true).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "send error while cancelling by command",
			request: dto.DialogRequest{Update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 101}, Text: "/help",
					Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
				},
			}},
			setupMocks: func(_ *trackermock.MockClient, sessions *repositorymock.MockSessionRepository, sender *commandmock.Sender) {
				sessions.EXPECT().Get(int64(101)).Return(repository.Session{State: repository.StateAwaitingURL}, true).Once()
				sessions.EXPECT().Clear(int64(101)).Once()
				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 101 && msg.Text != ""
				})).Return(tgbotapi.Message{}, sendErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, sendErr) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tracker := trackermock.NewMockClient(t)
			sessions := repositorymock.NewMockSessionRepository(t)
			sender := commandmock.NewSender(t)
			tt.setupMocks(tracker, sessions, sender)

			dialogHandler := link.NewDialogHandler(tracker, sessions, sender, nil)
			err := dialogHandler.Handle(context.Background(), tt.request)

			tt.assertErr(t, err)
		})
	}
}
