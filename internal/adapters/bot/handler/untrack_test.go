package handler_test

import (
	"context"
	"testing"

	botdto "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/dto"
	bothandler "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapters/bot/handler"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
)

func TestUntrackCommand_InvalidURL(t *testing.T) {
	sender := mock.NewSender(t)
	cmd := bothandler.NewUntrackCommandHandler(&fakeTracker{}, nil, sender)

	sender.EXPECT().
		Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
			return msg.ChatID == 9
		})).
		Return(tgbotapi.Message{}, nil).
		Once()

	err := cmd.Handle(context.Background(), botdto.CommandRequest{
		Text:   "/untrack xxx",
		ChatID: 9,
	})
	assert.NoError(t, err)
}
