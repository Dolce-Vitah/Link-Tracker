package handler_test

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	repositorymock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/gateway/repository/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/link"
	commandmock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/command/mock"
)

func TestCancelCommandHandler_Handle(t *testing.T) {
	t.Parallel()

	sendErr := errors.New("send error")

	tests := []struct {
		name       string
		request    dto.CommandRequest
		setupMocks func(sessions *repositorymock.MockSessionRepository, sender *commandmock.Sender)
		assertErr  func(t *testing.T, err error)
	}{
		{
			name:    "success",
			request: dto.CommandRequest{Text: "/cancel", ChatID: 1},
			setupMocks: func(sessions *repositorymock.MockSessionRepository, sender *commandmock.Sender) {
				sessions.EXPECT().Clear(int64(1)).Once()

				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 1 && msg.Text != ""
				})).Return(tgbotapi.Message{}, nil).Once()
			},
			assertErr: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:    "send error",
			request: dto.CommandRequest{Text: "/cancel", ChatID: 1},
			setupMocks: func(sessions *repositorymock.MockSessionRepository, sender *commandmock.Sender) {
				sessions.EXPECT().Clear(int64(1)).Once()

				sender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == 1 && msg.Text != ""
				})).Return(tgbotapi.Message{}, sendErr).Once()
			},
			assertErr: func(t *testing.T, err error) { require.ErrorIs(t, err, sendErr) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sessions := repositorymock.NewMockSessionRepository(t)
			sender := commandmock.NewSender(t)

			tt.setupMocks(sessions, sender)

			linkHandler := link.NewLinkHandler(sessions, nil, sender, nil)
			cmd := link.NewCancelCommand(linkHandler)

			err := cmd.Handle(context.Background(), tt.request)

			tt.assertErr(t, err)
		})
	}
}
