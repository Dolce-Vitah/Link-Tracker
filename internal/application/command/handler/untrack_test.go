package handler_test

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/handler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
)

func TestUntrackCommand_InvalidURL(t *testing.T) {
	cmd := handler.NewUntrackCommandHandler(&fakeTracker{}, nil)
	sender := mock.NewSender(t)
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 9},
			Text: "/untrack xxx",
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: 8},
			},
		},
	}

	sender.EXPECT().
		Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
			return msg.ChatID == 9
		})).
		Return(tgbotapi.Message{}, nil).
		Once()

	err := cmd.Handle(context.Background(), update, sender)
	assert.NoError(t, err)
}
