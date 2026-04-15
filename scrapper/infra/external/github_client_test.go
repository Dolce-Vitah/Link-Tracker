package external

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGitHubClient_GetLastUpdated_validationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		rawURL        string
		wantErrSubstr string
	}{
		{
			name:          "unsupported_host",
			rawURL:        "https://stackoverflow.com/questions/123",
			wantErrSubstr: "unsupported host",
		},
		{
			name:          "invalid_repo_path",
			rawURL:        "https://github.com/user/repo/issues/1",
			wantErrSubstr: "invalid github repository url",
		},
		{
			name:          "invalid_url",
			rawURL:        "http://a b",
			wantErrSubstr: "parse url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := NewGitHubClient(time.Second)
			_, err := client.GetLastUpdated(context.Background(), tt.rawURL)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErrSubstr)
		})
	}
}

func TestGitHubClient_GetLastUpdated_http(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		rawURL        string
		status        int
		body          string
		wantPath      string
		wantTime      time.Time
		wantErrSubstr string
	}{
		{
			name:     "success",
			rawURL:   "https://github.com/acme/widgets",
			status:   http.StatusOK,
			body:     `{"updated_at":"2024-06-15T12:30:00Z"}`,
			wantPath: "/repos/acme/widgets",
			wantTime: time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC),
		},
		{
			name:     "success_www_host",
			rawURL:   "https://www.github.com/org/service",
			status:   http.StatusOK,
			body:     `{"updated_at":"2024-01-01T00:00:00Z"}`,
			wantPath: "/repos/org/service",
			wantTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "non_2xx",
			rawURL:        "https://github.com/user/repo",
			status:        http.StatusNotFound,
			wantErrSubstr: "github non-2xx",
		},
		{
			name:          "decode_error",
			rawURL:        "https://github.com/user/repo",
			status:        http.StatusOK,
			body:          `not-json`,
			wantErrSubstr: "decode github response",
		},
		{
			name:          "missing_updated_at",
			rawURL:        "https://github.com/user/repo",
			status:        http.StatusOK,
			body:          `{}`,
			wantErrSubstr: "missing updated_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				if tt.body != "" {
					_, _ = io.WriteString(w, tt.body)
				}
			}))
			defer srv.Close()

			client := NewGitHubClient(5 * time.Second)
			client.baseURL = srv.URL

			got, err := client.GetLastUpdated(context.Background(), tt.rawURL)
			if tt.wantErrSubstr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErrSubstr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantTime, got)
			require.Equal(t, tt.wantPath, gotPath)
		})
	}
}
