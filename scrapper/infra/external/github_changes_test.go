package external

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/domain/linkchange"
)

func TestGitHubClient_FetchRepoChanges_baseline(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/o/r/issues", r.URL.Path)
		_, _ = io.WriteString(w, `[{"title":"x","body":"y","created_at":"2026-03-01T10:00:00Z","user":{"login":"a"},"html_url":"http://x"}]`)
	}))
	t.Cleanup(srv.Close)

	c := NewGitHubClient(time.Second, srv.URL)
	ch, wm, err := c.FetchRepoChanges(context.Background(), "o", "r", time.Time{})
	require.NoError(t, err)
	require.Empty(t, ch)
	require.Equal(t, time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), wm.UTC())
}

func TestGitHubClient_FetchRepoChanges_deltaNewIssue(t *testing.T) {
	t.Parallel()

	since := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"title":"PR title","body":"description text","created_at":"2026-04-01T12:00:00Z","user":{"login":"bot"},"pull_request":{"url":"http://pr"},"html_url":"http://pr"}]`)
	}))
	t.Cleanup(srv.Close)

	c := NewGitHubClient(time.Second, srv.URL)
	ch, wm, err := c.FetchRepoChanges(context.Background(), "o", "r", since)
	require.NoError(t, err)
	require.Len(t, ch, 1)
	require.Equal(t, linkchange.KindGitHubPullRequest, ch[0].Kind)
	require.Equal(t, "PR title", ch[0].Title)
	require.Equal(t, "bot", ch[0].Author)
	require.LessOrEqual(t, len([]rune(ch[0].Preview)), 200)
	require.Equal(t, time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), wm.UTC())
}

func TestHTTPClient_FetchChangesSince_routing(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/x/y/issues" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(srv.Close)

	h := NewHTTPClient(time.Second, srv.URL, srv.URL)
	_, _, err := h.FetchChangesSince(context.Background(), "https://github.com/x/y", time.Time{})
	require.NoError(t, err)
}
