package external

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPClient_GitHubNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewHTTPClient(time.Second, "", "")
	client.githubClient.baseURL = srv.URL
	_, err := client.GetLastUpdated(context.Background(), "https://github.com/user/repo")
	if err == nil {
		t.Fatalf("expected error for non-2xx github response")
	}
}

func TestHTTPClient_StackOverflowInvalidSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[{"unexpected":1}]}`))
	}))
	defer srv.Close()

	client := NewHTTPClient(time.Second, "", "")
	client.stackOverflowClient.baseURL = srv.URL
	_, err := client.GetLastUpdated(context.Background(), "https://stackoverflow.com/questions/123")
	if err == nil {
		t.Fatalf("expected error for invalid stackoverflow schema")
	}
}

func TestStackOverflowClient_RejectsNonStackOverflowHost(t *testing.T) {
	client := NewStackOverflowClient(time.Second, "")
	_, err := client.GetLastUpdated(context.Background(), "https://github.com/user/repo")
	if err == nil {
		t.Fatalf("expected unsupported host error")
	}
	if !strings.Contains(err.Error(), "unsupported host") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPClient_GitHubInvalidPath(t *testing.T) {
	client := NewHTTPClient(time.Second, "", "")
	_, err := client.GetLastUpdated(context.Background(), "https://github.com/user/repo/issues/1")
	if err == nil {
		t.Fatalf("expected error for non-repository github path")
	}
	if !strings.Contains(err.Error(), "invalid github repository url") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPClient_RejectsHostContainingGitHub(t *testing.T) {
	client := NewHTTPClient(time.Second, "", "")
	_, err := client.GetLastUpdated(context.Background(), "https://evilgithub.com/user/repo")
	if err == nil {
		t.Fatalf("expected unsupported host error")
	}
	if !strings.Contains(err.Error(), "unsupported host") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGitHubRepoPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		path      string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{name: "valid", path: "/owner/repo", wantOwner: "owner", wantRepo: "repo"},
		{name: "valid trailing slash", path: "/owner/repo/", wantOwner: "owner", wantRepo: "repo"},
		{name: "too many segments", path: "/owner/repo/issues/1", wantErr: true},
		{name: "too few segments", path: "/owner", wantErr: true},
		{name: "git suffix", path: "/owner/repo.git", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			owner, repo, err := parseGitHubRepoPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for path %q", tt.path)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for path %q: %v", tt.path, err)
			}
			if owner != tt.wantOwner || repo != tt.wantRepo {
				t.Fatalf("unexpected parsed values: owner=%q repo=%q", owner, repo)
			}
		})
	}
}

func TestHTTPClient_StackOverflowInvalidPath(t *testing.T) {
	client := NewHTTPClient(time.Second, "", "")
	_, err := client.GetLastUpdated(context.Background(), "https://stackoverflow.com/questions/not-a-number")
	if err == nil {
		t.Fatalf("expected error for non-numeric stackoverflow question id")
	}
	if !strings.Contains(err.Error(), "invalid stackoverflow question url") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseStackOverflowQuestionPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		wantID  int64
		wantErr bool
	}{
		{name: "valid", path: "/questions/123", wantID: 123},
		{name: "valid trailing slash", path: "/questions/123/", wantID: 123},
		{name: "with title slug", path: "/questions/123/title", wantID: 123},
		{name: "with long slug", path: "/questions/123/title/more/segments", wantID: 123},
		{name: "missing id", path: "/questions", wantErr: true},
		{name: "wrong prefix", path: "/question/123", wantErr: true},
		{name: "non numeric id", path: "/questions/abc", wantErr: true},
		{name: "zero id", path: "/questions/0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotID, err := parseStackOverflowQuestionPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for path %q", tt.path)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for path %q: %v", tt.path, err)
			}
			if gotID != tt.wantID {
				t.Fatalf("unexpected parsed id: %d", gotID)
			}
		})
	}
}
