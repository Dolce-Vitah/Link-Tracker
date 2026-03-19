package handler_test

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/handler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dialog"
)

func TestTrackCommand_StartsDialog(t *testing.T) {
	sessions := dialog.NewStore()
	sender := mock.NewSender(t)
	cmd := handler.NewTrackCommandHandler(sessions, &fakeTracker{}, nil, sender)

	sender.EXPECT().
		Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
			return msg.ChatID == 1
		})).
		Return(tgbotapi.Message{}, nil).
		Once()

	err := cmd.Handle(context.Background(), "/track", 1)
	assert.NoError(t, err)

	session, ok := sessions.Get(1)
	assert.True(t, ok)
	assert.Equal(t, dialog.StateAwaitingURL, session.State)
}
