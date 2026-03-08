package handler_test

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/handler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
)

func TestStartCommand_Handle(t *testing.T) {
	type args struct {
		chatId int64
	}
	tests := []struct {
		name          string
		args          args
		expectedText  string
		mockSendError error
		wantErr       bool
	}{
		{
			name: "success",
			args: args{
				chatId: 12345,
			},
			expectedText:  "Добро пожаловать! Используйте /help, чтобы узнать о доступных командах.",
			mockSendError: nil,
			wantErr:       false,
		},
		{
			name: "send error",
			args: args{
				chatId: 67890,
			},
			expectedText:  "Добро пожаловать! Используйте /help, чтобы узнать о доступных командах.",
			mockSendError: errors.New("network error"),
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				cmd        = handler.NewStartCommandHandler(nil)
				mockSender = mock.NewSender(t)
				ctx        = context.Background()
				update     = &tgbotapi.Update{
					Message: &tgbotapi.Message{
						Chat: &tgbotapi.Chat{ID: tt.args.chatId},
					},
				}
			)

			mockSender.EXPECT().
				Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
					return msg.ChatID == tt.args.chatId && msg.Text == tt.expectedText
				})).
				Return(tgbotapi.Message{}, tt.mockSendError).
				Once()

			err := cmd.Handle(ctx, update, mockSender)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStartCommand_Metadata(t *testing.T) {
	cmd := handler.NewStartCommandHandler(nil)
	require.Equal(t, "start", cmd.Name())
	require.NotEmpty(t, cmd.Description())
}
