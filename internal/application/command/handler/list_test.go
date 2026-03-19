package handler_test

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/handler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

func TestListCommand_EmptyListMessage(t *testing.T) {
	sender := mock.NewSender(t)
	cmd := handler.NewListCommandHandler(&fakeTracker{}, nil, sender)

	sender.EXPECT().
		Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
			return msg.Text == "Список отслеживаемых ссылок пуст."
		})).
		Return(tgbotapi.Message{}, nil).
		Once()

	err := cmd.Handle(context.Background(), "/list", 7)
	assert.NoError(t, err)
}

func TestListCommand_FilterByTag(t *testing.T) {
	sender := mock.NewSender(t)
	cmd := handler.NewListCommandHandler(&fakeTracker{
		links: []api.LinkResponse{
			{ID: 1, URL: "https://github.com/a/b", Tags: []string{"work"}},
			{ID: 2, URL: "https://stackoverflow.com/questions/1/x", Tags: []string{"misc"}},
		},
	}, nil, sender)

	sender.EXPECT().
		Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
			return msg.ChatID == 10 && msg.Text != "" && msg.Text != "Список отслеживаемых ссылок пуст."
		})).
		Return(tgbotapi.Message{}, nil).
		Once()

	err := cmd.Handle(context.Background(), "/list work", 10)
	assert.NoError(t, err)
}
