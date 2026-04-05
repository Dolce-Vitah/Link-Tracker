package textutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTruncateRunes(t *testing.T) {
	t.Parallel()

	require.Equal(t, "", TruncateRunes("hello", 0))
	require.Equal(t, "hello", TruncateRunes("hello", 10))
	require.Equal(t, "при", TruncateRunes("привет", 3))
	require.Equal(t, "a界", TruncateRunes("a界b", 2))
}
