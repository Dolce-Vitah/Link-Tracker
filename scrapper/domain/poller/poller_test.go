package poller

import (
	"context"
	"errors"
	"testing"
	"time"

	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/domain/linkchange"
	botclientmock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/botclient/mock"
	externalmock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/external/mock"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository/repositorytest"
)

func TestSplitIntoChunks_distribution(t *testing.T) {
	t.Parallel()
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	chunks := splitIntoChunks(items, 4)
	require.Len(t, chunks, 4)
	require.Equal(t, []int{1, 2, 3}, chunks[0])
	require.Equal(t, []int{4, 5, 6}, chunks[1])
	require.Equal(t, []int{7, 8}, chunks[2])
	require.Equal(t, []int{9, 10}, chunks[3])
}

func TestSplitIntoChunks_moreWorkersThanItems(t *testing.T) {
	t.Parallel()
	chunks := splitIntoChunks([]int{1, 2}, 4)
	require.Len(t, chunks, 4)
	require.Equal(t, []int{1}, chunks[0])
	require.Equal(t, []int{2}, chunks[1])
	require.Nil(t, chunks[2])
	require.Nil(t, chunks[3])
}

func TestPoller_ProcessOnce_Flow(t *testing.T) {
	t.Parallel()

	const (
		urlMain = "https://github.com/user/repo"
		urlA    = "https://github.com/user/a"
		urlB    = "https://github.com/user/b"
	)

	baseTime := time.Date(2026, time.March, 19, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name                 string
		setup                func(t *testing.T, service repository.Service, ext *externalmock.MockLinkChangesClient, sender *botclientmock.MockMessageSender, sent *[]trackerapi.LinkUpdate, now time.Time)
		expectedUpdatesCount int
		expectedRecipients   map[string][]int64
		expectedLastUpdated  map[string]time.Time
	}{
		{
			name: "send update only to subscribed chats",
			setup: func(t *testing.T, service repository.Service, ext *externalmock.MockLinkChangesClient, sender *botclientmock.MockMessageSender, sent *[]trackerapi.LinkUpdate, now time.Time) {
				registerChats(t, service, 1, 2, 999)
				addLink(t, service, 1, urlMain)
				addLink(t, service, 2, urlMain)
				wm := now.Add(10 * time.Minute)
				ext.On("FetchChangesSince", testifymock.Anything, urlMain, testifymock.Anything).
					Return([]linkchange.Change{{
						Kind:      linkchange.KindGitHubIssue,
						Title:     "Issue 1",
						Author:    "alice",
						CreatedAt: wm,
						Preview:   "body",
					}}, wm, nil).
					Once()
				expectSendLinkUpdate(sender, sent, urlMain, nil)
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
			name: "failure report when external source returns error",
			setup: func(t *testing.T, service repository.Service, ext *externalmock.MockLinkChangesClient, sender *botclientmock.MockMessageSender, _ *[]trackerapi.LinkUpdate, _ time.Time) {
				registerChats(t, service, 10)
				addLink(t, service, 10, urlMain)
				ext.On("FetchChangesSince", testifymock.Anything, urlMain, testifymock.Anything).
					Return([]linkchange.Change(nil), time.Time{}, errors.New("external failure")).
					Once()
				sender.On("SendProcessingFailureReport", testifymock.Anything, testifymock.MatchedBy(func(r trackerapi.ProcessingFailureReport) bool {
					return r.TgChatID == 10 && len(r.URLs) == 1 && r.URLs[0] == urlMain
				})).
					Return(nil).
					Once()
			},
			expectedUpdatesCount: 0,
			expectedRecipients:   map[string][]int64{},
			expectedLastUpdated:  map[string]time.Time{urlMain: {}},
		},
		{
			name: "skip when no new changes after last_updated",
			setup: func(t *testing.T, service repository.Service, ext *externalmock.MockLinkChangesClient, _ *botclientmock.MockMessageSender, _ *[]trackerapi.LinkUpdate, now time.Time) {
				registerChats(t, service, 20)
				addLink(t, service, 20, urlMain)
				require.NoError(t, service.UpdateLastUpdated(urlMain, now.Add(30*time.Minute)))
				ext.On("FetchChangesSince", testifymock.Anything, urlMain, now.Add(30*time.Minute)).
					Return([]linkchange.Change(nil), now.Add(5*time.Minute), nil).
					Once()
			},
			expectedUpdatesCount: 0,
			expectedRecipients:   map[string][]int64{},
			expectedLastUpdated: map[string]time.Time{
				urlMain: baseTime.Add(30 * time.Minute),
			},
		},
		{
			name: "do not update timestamp when sending update fails",
			setup: func(t *testing.T, service repository.Service, ext *externalmock.MockLinkChangesClient, sender *botclientmock.MockMessageSender, sent *[]trackerapi.LinkUpdate, now time.Time) {
				registerChats(t, service, 30)
				addLink(t, service, 30, urlMain)
				wm := now.Add(15 * time.Minute)
				ext.On("FetchChangesSince", testifymock.Anything, urlMain, testifymock.Anything).
					Return([]linkchange.Change{{
						Kind:      linkchange.KindGitHubIssue,
						Title:     "T",
						Author:    "u",
						CreatedAt: wm,
						Preview:   "p",
					}}, wm, nil).
					Once()
				expectSendLinkUpdate(sender, sent, urlMain, errors.New("send failure"))
				sender.On("SendProcessingFailureReport", testifymock.Anything, testifymock.MatchedBy(func(r trackerapi.ProcessingFailureReport) bool {
					return r.TgChatID == 30 && len(r.URLs) == 1 && r.URLs[0] == urlMain
				})).
					Return(nil).
					Once()
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
			setup: func(t *testing.T, service repository.Service, ext *externalmock.MockLinkChangesClient, sender *botclientmock.MockMessageSender, sent *[]trackerapi.LinkUpdate, now time.Time) {
				registerChats(t, service, 1, 2)
				addLink(t, service, 1, urlA)
				addLink(t, service, 2, urlB)
				wmA := now.Add(1 * time.Minute)
				ext.On("FetchChangesSince", testifymock.Anything, urlA, testifymock.Anything).
					Return([]linkchange.Change{{
						Kind: linkchange.KindGitHubIssue, Title: "A", Author: "a", CreatedAt: wmA, Preview: "",
					}}, wmA, nil).
					Once()
				ext.On("FetchChangesSince", testifymock.Anything, urlB, testifymock.Anything).
					Return([]linkchange.Change(nil), time.Time{}, errors.New("external failure")).
					Once()
				expectSendLinkUpdate(sender, sent, urlA, nil)
				sender.On("SendProcessingFailureReport", testifymock.Anything, testifymock.MatchedBy(func(r trackerapi.ProcessingFailureReport) bool {
					return r.TgChatID == 2 && len(r.URLs) == 1 && r.URLs[0] == urlB
				})).
					Return(nil).
					Once()
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

			service := repositorytest.NewStore()
			ext := externalmock.NewMockLinkChangesClient(t)
			sender := botclientmock.NewMockMessageSender(t)
			sentUpdates := make([]trackerapi.LinkUpdate, 0)

			tt.setup(t, service, ext, sender, &sentUpdates, baseTime)

			p := New(service, ext, sender, Config{
				DBPageSize:     100,
				SuperBatchSize: 1000,
				WorkerCount:    1,
				CheckInterval:  time.Second,
			})
			p.ProcessOnce(context.Background())

			require.Len(t, sentUpdates, tt.expectedUpdatesCount)

			updatesByURL := make(map[string]trackerapi.LinkUpdate, len(sentUpdates))
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

func expectSendLinkUpdate(sender *botclientmock.MockMessageSender, sent *[]trackerapi.LinkUpdate, rawURL string, returnErr error) {
	sender.On("SendLinkUpdate", testifymock.Anything, testifymock.MatchedBy(func(update trackerapi.LinkUpdate) bool {
		return update.URL == rawURL
	})).
		Run(func(args testifymock.Arguments) {
			*sent = append(*sent, args.Get(1).(trackerapi.LinkUpdate))
		}).
		Return(returnErr).
		Once()
}

func registerChats(t *testing.T, service repository.Service, chatIDs ...int64) {
	t.Helper()
	for _, chatID := range chatIDs {
		require.NoError(t, service.RegisterChat(chatID))
	}
}

func addLink(t *testing.T, service repository.Service, chatID int64, rawURL string) {
	t.Helper()
	_, err := service.AddLink(chatID, trackerapi.AddLinkRequest{Link: rawURL})
	require.NoError(t, err)
}

func getLastUpdatedFromSnapshot(t *testing.T, service repository.Service, rawURL string) time.Time {
	t.Helper()
	items, err := service.ListTrackedLinksPage(1000, time.Now().UTC(), 0)
	require.NoError(t, err)
	for _, tracked := range items {
		if tracked.Response.URL == rawURL {
			return tracked.LastUpdated
		}
	}
	return time.Time{}
}
