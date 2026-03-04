package command_test

import (
	"context"
	"errors"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
)

func TestHelpCommand_Handle(t *testing.T) {
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
			expectedText:  "Доступные команды:\n/start - Начало работы с ботом\n/help - Показать этот список команд",
			mockSendError: nil,
			wantErr:       false,
		},
		{
			name: "send error",
			args: args{
				chatId: 67890,
			},
			expectedText:  "Доступные команды:\n/start - Начало работы с ботом\n/help - Показать этот список команд",
			mockSendError: errors.New("network error"),
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				cmd        = &command.HelpCommand{}
				mockSender = mock.NewSender(t)
				ctx        = context.Background()
				update     = &tgbotapi.Update{
					Message: &tgbotapi.Message{
						Chat: &tgbotapi.Chat{ID: tt.args.chatId},
					},
				}
			)

			mockSender.EXPECT().Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
				return msg.ChatID == tt.args.chatId && msg.Text == tt.expectedText
			})).Return(tgbotapi.Message{}, tt.mockSendError).Once()

			err := cmd.Handle(ctx, update, mockSender)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHelpCommand_Metadata(t *testing.T) {
	cmd := &command.HelpCommand{}
	assert.Equal(t, "help", cmd.Name())
	assert.NotEmpty(t, cmd.Description())
}
