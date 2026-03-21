package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	botclientmock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/botclient/mock"
	externalmock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/external/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func TestScheduler_ProcessOnce_Flow(t *testing.T) {
	t.Parallel()

	const (
		urlMain = "https://github.com/user/repo"
		urlA    = "https://github.com/user/a"
		urlB    = "https://github.com/user/b"
	)

	baseTime := time.Date(2026, time.March, 19, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name                 string
		setup                func(t *testing.T, service *repository.Service, external *externalmock.MockLastUpdatedClient, updates *botclientmock.MockUpdatesSender, sent *[]api.LinkUpdate, now time.Time)
		expectedUpdatesCount int
		expectedRecipients   map[string][]int64
		expectedLastUpdated  map[string]time.Time
	}{
		{
			name: "send update only to subscribed chats",
			setup: func(t *testing.T, service *repository.Service, external *externalmock.MockLastUpdatedClient, updates *botclientmock.MockUpdatesSender, sent *[]api.LinkUpdate, now time.Time) {
				registerChats(t, service, 1, 2, 999)
				addLink(t, service, 1, urlMain)
				addLink(t, service, 2, urlMain)
				external.On("GetLastUpdated", testifymock.Anything, urlMain).Return(now.Add(10*time.Minute), nil).Once()
				expectSendUpdate(updates, sent, urlMain, nil)
			},
			expectedUpdatesCount: 1,
			expectedRecipients: map[string][]int64{
				urlMain: {1, 2},
			},
			expectedLastUpdated: map[string]time.Time{
				urlMain: baseTime.Add(10 * time.Minute),
			},
		},
		{
			name: "skip when external source returns error",
			setup: func(t *testing.T, service *repository.Service, external *externalmock.MockLastUpdatedClient, _ *botclientmock.MockUpdatesSender, _ *[]api.LinkUpdate, _ time.Time) {
				registerChats(t, service, 10)
				addLink(t, service, 10, urlMain)
				external.On("GetLastUpdated", testifymock.Anything, urlMain).Return(time.Time{}, errors.New("external failure")).Once()
			},
			expectedUpdatesCount: 0,
			expectedRecipients:   map[string][]int64{},
			expectedLastUpdated: map[string]time.Time{
				urlMain: {},
			},
		},
		{
			name: "skip when last updated is not newer",
			setup: func(t *testing.T, service *repository.Service, external *externalmock.MockLastUpdatedClient, _ *botclientmock.MockUpdatesSender, _ *[]api.LinkUpdate, now time.Time) {
				registerChats(t, service, 20)
				addLink(t, service, 20, urlMain)
				service.UpdateLastUpdated(urlMain, now.Add(30*time.Minute))
				external.On("GetLastUpdated", testifymock.Anything, urlMain).Return(now.Add(5*time.Minute), nil).Once()
			},
			expectedUpdatesCount: 0,
			expectedRecipients:   map[string][]int64{},
			expectedLastUpdated: map[string]time.Time{
				urlMain: baseTime.Add(30 * time.Minute),
			},
		},
		{
			name: "do not update timestamp when sending update fails",
			setup: func(t *testing.T, service *repository.Service, external *externalmock.MockLastUpdatedClient, updates *botclientmock.MockUpdatesSender, sent *[]api.LinkUpdate, now time.Time) {
				registerChats(t, service, 30)
				addLink(t, service, 30, urlMain)
				external.On("GetLastUpdated", testifymock.Anything, urlMain).Return(now.Add(15*time.Minute), nil).Once()
				expectSendUpdate(updates, sent, urlMain, errors.New("send failure"))
			},
			expectedUpdatesCount: 1,
			expectedRecipients: map[string][]int64{
				urlMain: {30},
			},
			expectedLastUpdated: map[string]time.Time{
				urlMain: {},
			},
		},
		{
			name: "process multiple links independently",
			setup: func(t *testing.T, service *repository.Service, external *externalmock.MockLastUpdatedClient, updates *botclientmock.MockUpdatesSender, sent *[]api.LinkUpdate, now time.Time) {
				registerChats(t, service, 1, 2)
				addLink(t, service, 1, urlA)
				addLink(t, service, 2, urlB)
				external.On("GetLastUpdated", testifymock.Anything, urlA).Return(now.Add(1*time.Minute), nil).Once()
				external.On("GetLastUpdated", testifymock.Anything, urlB).Return(time.Time{}, errors.New("external failure")).Once()
				expectSendUpdate(updates, sent, urlA, nil)
			},
			expectedUpdatesCount: 1,
			expectedRecipients: map[string][]int64{
				urlA: {1},
			},
			expectedLastUpdated: map[string]time.Time{
				urlA: baseTime.Add(1 * time.Minute),
				urlB: {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := repository.NewService()
			external := externalmock.NewMockLastUpdatedClient(t)
			updates := botclientmock.NewMockUpdatesSender(t)
			sentUpdates := make([]api.LinkUpdate, 0)
			tt.setup(t, service, external, updates, &sentUpdates, baseTime)

			scheduler := New(service, external, updates, time.Second)
			scheduler.ProcessOnce(context.Background())

			require.Len(t, sentUpdates, tt.expectedUpdatesCount)

			updatesByURL := make(map[string]api.LinkUpdate, len(sentUpdates))
			for _, update := range sentUpdates {
				updatesByURL[update.URL] = update
			}

			for expectedURL, expectedChatIDs := range tt.expectedRecipients {
				update, ok := updatesByURL[expectedURL]
				require.True(t, ok, "expected update for URL %s", expectedURL)
				require.ElementsMatch(t, expectedChatIDs, update.TgChatIDs)
			}

			for expectedURL, expectedTime := range tt.expectedLastUpdated {
				actualTime := getLastUpdatedFromSnapshot(t, service, expectedURL)
				require.Equal(t, expectedTime, actualTime)
			}
		})
	}
}

func expectSendUpdate(updates *botclientmock.MockUpdatesSender, sent *[]api.LinkUpdate, rawURL string, returnErr error) {
	updates.
		On("SendUpdate", testifymock.Anything, testifymock.MatchedBy(func(update api.LinkUpdate) bool {
			return update.URL == rawURL
		})).
		Run(func(args testifymock.Arguments) {
			*sent = append(*sent, args[1].(api.LinkUpdate))
		}).
		Return(returnErr).
		Once()
}

func registerChats(t *testing.T, service *repository.Service, chatIDs ...int64) {
	t.Helper()
	for _, chatID := range chatIDs {
		require.NoError(t, service.RegisterChat(chatID))
	}
}

func addLink(t *testing.T, service *repository.Service, chatID int64, rawURL string) {
	t.Helper()
	_, err := service.AddLink(chatID, api.AddLinkRequest{Link: rawURL})
	require.NoError(t, err)
}

func getLastUpdatedFromSnapshot(t *testing.T, service *repository.Service, rawURL string) time.Time {
	t.Helper()
	for _, tracked := range service.SnapshotLinks() {
		if tracked.Response.URL == rawURL {
			return tracked.LastUpdated
		}
	}
	return time.Time{}
}
