//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/domain/poller"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/db/migrations"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/external"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository/sqlrepo"
)

type captureMessageSender struct {
	mu       sync.Mutex
	updates  []trackerapi.LinkUpdate
	reports  []trackerapi.ProcessingFailureReport
	sendErr  error
	reportOK bool
}

func (c *captureMessageSender) SendLinkUpdate(_ context.Context, u trackerapi.LinkUpdate) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sendErr != nil {
		return c.sendErr
	}
	c.updates = append(c.updates, u)
	return nil
}

func (c *captureMessageSender) SendProcessingFailureReport(_ context.Context, r trackerapi.ProcessingFailureReport) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reports = append(c.reports, r)
	c.reportOK = true
	return nil
}

func TestPoller_GitHubIssue_NotificationViaStubAPI(t *testing.T) {
	t.Parallel()
	requireTestcontainers(t)

	sqlDB, _, cleanup := startPostgresDB(t)
	defer cleanup()
	require.NoError(t, migrations.Run(sqlDB, "../../migrations"))
	repo := sqlrepo.New(sqlDB)

	const (
		chatID = int64(501)
		ghURL  = "https://github.com/acme/demo"
	)
	require.NoError(t, repo.RegisterChat(chatID))
	_, err := repo.AddLink(chatID, trackerapi.AddLinkRequest{Link: ghURL})
	require.NoError(t, err)
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.UpdateLastUpdated(ghURL, old))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		path := r.URL.Path
		if strings.HasPrefix(path, "/repos/acme/demo/issues") {
			body := `[{"title":"New bug","body":"Hello **world** long text","created_at":"2026-04-02T15:00:00Z","user":{"login":"dev1"},"html_url":"https://github.com/acme/demo/issues/9"}]`
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	ext := external.NewHTTPClient(5*time.Second, srv.URL, srv.URL)
	cap := &captureMessageSender{}
	p := poller.New(repo, ext, cap, poller.Config{
		DBPageSize:      100,
		SuperBatchSize:  1000,
		WorkerCount:     4,
		CheckInterval:   time.Hour,
	})
	p.ProcessOnce(context.Background())

	cap.mu.Lock()
	defer cap.mu.Unlock()
	require.Len(t, cap.updates, 1)
	u := cap.updates[0]
	require.Equal(t, ghURL, u.URL)
	require.Equal(t, trackerapi.EventKindGitHubIssue, u.EventKind)
	require.Equal(t, "New bug", u.Title)
	require.Equal(t, "dev1", u.Author)
	require.Contains(t, u.Preview, "Hello")
	require.LessOrEqual(t, len([]rune(u.Preview)), trackerapi.PreviewMaxRunes)
}

func TestPoller_StackOverflowAnswer_NotificationViaStubAPI(t *testing.T) {
	t.Parallel()
	requireTestcontainers(t)

	sqlDB, _, cleanup := startPostgresDB(t)
	defer cleanup()
	require.NoError(t, migrations.Run(sqlDB, "../../migrations"))
	repo := sqlrepo.New(sqlDB)

	const (
		chatID = int64(502)
		soURL  = "https://stackoverflow.com/questions/4242"
	)
	require.NoError(t, repo.RegisterChat(chatID))
	_, err := repo.AddLink(chatID, trackerapi.AddLinkRequest{Link: soURL})
	require.NoError(t, err)
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.UpdateLastUpdated(soURL, old))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		path := r.URL.Path
		switch {
		case path == "/questions/4242" && r.URL.Query().Get("site") == "stackoverflow":
			_, _ = w.Write([]byte(`{"items":[{"question_id":4242,"title":"How to Go?","last_activity_date":1712000000}]}`))
		case strings.HasPrefix(path, "/questions/4242/timeline"):
			_, _ = w.Write([]byte(`{"items":[{"timeline_type":"answer","creation_date":1712001000,"post_id":9001,"user_id":7,"detail":""}]}`))
		case strings.HasPrefix(path, "/users/7"):
			_, _ = w.Write([]byte(`{"items":[{"user_id":7,"display_name":"Gopher"}]}`))
		case strings.HasPrefix(path, "/answers/9001"):
			_, _ = w.Write([]byte(`{"items":[{"answer_id":9001,"body":"<p>Use modules.</p>"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	ext := external.NewHTTPClient(5*time.Second, srv.URL, srv.URL)
	cap := &captureMessageSender{}
	p := poller.New(repo, ext, cap, poller.Config{
		DBPageSize:      100,
		SuperBatchSize:  500,
		WorkerCount:     2,
		CheckInterval:   time.Hour,
	})
	p.ProcessOnce(context.Background())

	cap.mu.Lock()
	defer cap.mu.Unlock()
	require.Len(t, cap.updates, 1)
	u := cap.updates[0]
	require.Equal(t, trackerapi.EventKindStackAnswer, u.EventKind)
	require.Equal(t, "How to Go?", u.Title)
	require.Equal(t, "Gopher", u.Author)
	require.Contains(t, u.Preview, "modules")
}

func TestPoller_ExternalError_FailureReport(t *testing.T) {
	t.Parallel()
	requireTestcontainers(t)

	sqlDB, _, cleanup := startPostgresDB(t)
	defer cleanup()
	require.NoError(t, migrations.Run(sqlDB, "../../migrations"))
	repo := sqlrepo.New(sqlDB)

	const (
		chatID = int64(503)
		ghURL  = "https://github.com/acme/broken"
	)
	require.NoError(t, repo.RegisterChat(chatID))
	_, err := repo.AddLink(chatID, trackerapi.AddLinkRequest{Link: ghURL})
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	ext := external.NewHTTPClient(5*time.Second, srv.URL, srv.URL)
	cap := &captureMessageSender{}
	p := poller.New(repo, ext, cap, poller.Config{
		DBPageSize:     50,
		SuperBatchSize: 1000,
		WorkerCount:    4,
		CheckInterval:  time.Hour,
	})
	p.ProcessOnce(context.Background())

	cap.mu.Lock()
	defer cap.mu.Unlock()
	require.True(t, cap.reportOK)
	require.Len(t, cap.reports, 1)
	require.Equal(t, chatID, cap.reports[0].TgChatID)
	require.Contains(t, cap.reports[0].URLs, ghURL)
}

func TestPoller_MultipleLinks_PartialFailure_Isolated(t *testing.T) {
	t.Parallel()
	requireTestcontainers(t)

	sqlDB, _, cleanup := startPostgresDB(t)
	defer cleanup()
	require.NoError(t, migrations.Run(sqlDB, "../../migrations"))
	repo := sqlrepo.New(sqlDB)

	const chatID = int64(504)
	require.NoError(t, repo.RegisterChat(chatID))
	good := "https://github.com/good/ok"
	bad := "https://github.com/bad/fail"
	_, err := repo.AddLink(chatID, trackerapi.AddLinkRequest{Link: good})
	require.NoError(t, err)
	_, err = repo.AddLink(chatID, trackerapi.AddLinkRequest{Link: bad})
	require.NoError(t, err)
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.UpdateLastUpdated(good, old))
	require.NoError(t, repo.UpdateLastUpdated(bad, old))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/repos/good/ok/issues") {
			body := `[{"title":"OK","body":"x","created_at":"2026-04-02T16:00:00Z","user":{"login":"u"},"html_url":"https://github.com/good/ok/issues/1"}]`
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
			return
		}
		if strings.HasPrefix(path, "/repos/bad/fail/issues") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	ext := external.NewHTTPClient(5*time.Second, srv.URL, srv.URL)
	cap := &captureMessageSender{}
	p := poller.New(repo, ext, cap, poller.Config{
		DBPageSize:      50,
		SuperBatchSize:  1000,
		WorkerCount:     4,
		CheckInterval:   time.Hour,
	})
	p.ProcessOnce(context.Background())

	cap.mu.Lock()
	defer cap.mu.Unlock()
	require.Len(t, cap.updates, 1, "good link should still produce an update")
	require.Len(t, cap.reports, 1)
	require.Contains(t, cap.reports[0].URLs, bad)
}
