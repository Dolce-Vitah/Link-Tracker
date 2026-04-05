package linkdto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLinkUpdate_Validate_rejectsLongPreview(t *testing.T) {
	t.Parallel()
	long := string(make([]rune, PreviewMaxRunes+1))
	u := LinkUpdate{
		URL:       "https://a",
		TgChatIDs: []int64{1},
		EventKind: EventKindGitHubIssue,
		Title:     "t",
		Author:    "a",
		CreatedAt: time.Now().UTC(),
		Preview:   long,
	}
	require.Error(t, u.Validate())
}

func TestNewLinkUpdate_truncatesPreviewRunes(t *testing.T) {
	t.Parallel()
	longBody := string(make([]rune, 300))
	u := NewLinkUpdate(1, "https://a", []int64{1}, EventKindGitHubIssue, "t", "a", time.Now().UTC(), longBody)
	require.NoError(t, u.Validate())
	require.Len(t, []rune(u.Preview), PreviewMaxRunes)
}
