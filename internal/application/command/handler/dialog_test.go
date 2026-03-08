package handler_test

import (
	"context"
	"fmt"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/handler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/command/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dialog"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

func TestHandleTrackDialogStep_FullFlowWithTags(t *testing.T) {
	sessions := dialog.NewStore()
	sessions.Set(100, dialog.Session{State: dialog.StateAwaitingURL})

	var captured api.AddLinkRequest
	tracker := &fakeTracker{
		addLinkFn: func(ctx context.Context, chatID int64, request api.AddLinkRequest) (api.LinkResponse, error) {
			captured = request
			return api.LinkResponse{ID: 1, URL: request.Link, Tags: request.Tags, Filters: request.Filters}, nil
		},
	}
	sender := mock.NewSender(t)

	sender.EXPECT().
		Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
			return msg.ChatID == 100 && msg.Text != ""
		})).
		Return(tgbotapi.Message{}, nil).
		Times(2)

	// Step 1: URL
	err := handler.HandleTrackDialogStep(context.Background(), tracker, sessions, &tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 100},
			Text: "https://github.com/user/repo",
		},
	}, sender, nil)
	assert.NoError(t, err)

	session, ok := sessions.Get(100)
	assert.True(t, ok)
	assert.Equal(t, dialog.StateAwaitingTags, session.State)

	// Step 2: tags
	err = handler.HandleTrackDialogStep(context.Background(), tracker, sessions, &tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 100},
			Text: "work,docs",
		},
	}, sender, nil)
	assert.NoError(t, err)

	_, ok = sessions.Get(100)
	assert.False(t, ok, "dialog should be cleared after successful completion")
	assert.Equal(t, "https://github.com/user/repo", captured.Link)
	assert.Equal(t, []string{"work", "docs"}, captured.Tags)
	assert.Nil(t, captured.Filters)
}

func TestHandleTrackDialogStep_CancelOnAnotherCommand(t *testing.T) {
	sessions := dialog.NewStore()
	sessions.Set(101, dialog.Session{State: dialog.StateAwaitingURL})
	sender := mock.NewSender(t)
	tracker := &fakeTracker{}

	sender.EXPECT().
		Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
			return msg.ChatID == 101 && msg.Text == "Процесс отслеживания отменен из-за новой команды."
		})).
		Return(tgbotapi.Message{}, nil).
		Once()

	err := handler.HandleTrackDialogStep(context.Background(), tracker, sessions, &tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 101},
			Text: "/help",
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: 5},
			},
		},
	}, sender, nil)
	assert.NoError(t, err)

	_, ok := sessions.Get(101)
	assert.False(t, ok, "dialog should be cleared when another command is received")
}

func TestHandleTrackDialogStep_InvalidURL(t *testing.T) {
	sessions := dialog.NewStore()
	sessions.Set(102, dialog.Session{State: dialog.StateAwaitingURL})
	sender := mock.NewSender(t)

	sender.EXPECT().
		Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
			return msg.ChatID == 102 && msg.Text == "Ссылка некорректна. Введите ссылку в формате https://example.com/path"
		})).
		Return(tgbotapi.Message{}, nil).
		Once()

	err := handler.HandleTrackDialogStep(context.Background(), &fakeTracker{}, sessions, &tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 102},
			Text: "tbank://github.com/user/repo",
		},
	}, sender, nil)
	assert.NoError(t, err)
}

func TestHandleTrackDialogStep_DuplicateLink(t *testing.T) {
	sessions := dialog.NewStore()
	sessions.Set(103, dialog.Session{
		State:      dialog.StateAwaitingTags,
		PendingURL: "https://github.com/user/repo",
	})
	sender := mock.NewSender(t)
	trackerSvc := &fakeTracker{
		addLinkFn: func(ctx context.Context, chatID int64, request api.AddLinkRequest) (api.LinkResponse, error) {
			return api.LinkResponse{}, fmt.Errorf("%w: duplicated", tracker.ErrAlreadyExists)
		},
	}

	sender.EXPECT().
		Send(testifymock.MatchedBy(func(msg tgbotapi.MessageConfig) bool {
			return msg.ChatID == 103 && msg.Text == "Ссылка уже отслеживается"
		})).
		Return(tgbotapi.Message{}, nil).
		Once()

	err := handler.HandleTrackDialogStep(context.Background(), trackerSvc, sessions, &tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 103},
			Text: "work",
		},
	}, sender, nil)
	assert.NoError(t, err)
}
