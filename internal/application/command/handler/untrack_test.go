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
	sender := mock.NewSender(t)
	cmd := handler.NewUntrackCommandHandler(&fakeTracker{}, nil, sender)

	sender.EXPECT().
		Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
			return msg.ChatID == 9
		})).
		Return(tgbotapi.Message{}, nil).
		Once()

	err := cmd.Handle(context.Background(), "/untrack xxx", 9)
	assert.NoError(t, err)
}
