package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/linkdto"
)

func TestFormatLinkUpdateMessage_containsFields(t *testing.T) {
	t.Parallel()
	u := linkdto.LinkUpdate{
		URL:       "https://github.com/o/r",
		EventKind: linkdto.EventKindGitHubIssue,
		Title:     "My issue",
		Author:    "alice",
		CreatedAt: time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC),
		Preview:   "short",
	}
	text := formatLinkUpdateMessage(u)
	require.Contains(t, text, "My issue")
	require.Contains(t, text, "alice")
	require.Contains(t, text, "short")
	require.Contains(t, text, "https://github.com/o/r")
}

func TestFormatFailureReportMessage_listsURLs(t *testing.T) {
	t.Parallel()
	text := formatFailureReportMessage([]string{"https://a", "https://b"}, "detail")
	require.Contains(t, text, "https://a")
	require.Contains(t, text, "https://b")
	require.True(t, strings.Contains(text, "detail"))
}
