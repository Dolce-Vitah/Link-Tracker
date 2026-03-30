package botclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

func TestHTTPUpdatesClient_SendUpdate_StatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		status      int
		expectedErr error
	}{
		{name: "ok status", status: http.StatusOK, expectedErr: nil},
		{name: "bad request", status: http.StatusBadRequest, expectedErr: ErrBadRequest},
		{name: "unauthorized", status: http.StatusUnauthorized, expectedErr: ErrUnauthorized},
		{name: "forbidden", status: http.StatusForbidden, expectedErr: ErrForbidden},
		{name: "not found", status: http.StatusNotFound, expectedErr: ErrNotFound},
		{name: "conflict", status: http.StatusConflict, expectedErr: ErrConflict},
		{name: "too many requests", status: http.StatusTooManyRequests, expectedErr: ErrTooManyRequests},
		{name: "other 4xx", status: http.StatusTeapot, expectedErr: ErrClient},
		{name: "5xx", status: http.StatusInternalServerError, expectedErr: ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			t.Cleanup(server.Close)

			client := NewHTTPUpdatesClient(server.URL, time.Second)
			err := client.SendUpdate(context.Background(), api.LinkUpdate{
				URL:       "https://example.com",
				TgChatIDs: []int64{1},
			})

			if tt.expectedErr == nil {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)

			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestHTTPUpdatesClient_SendUpdate_RejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	client := NewHTTPUpdatesClient("http://unused.example", time.Second)
	err := client.SendUpdate(context.Background(), api.LinkUpdate{URL: "https://example.com"})
	require.Error(t, err)
	require.ErrorIs(t, err, api.ErrInvalidLinkUpdate)
}
