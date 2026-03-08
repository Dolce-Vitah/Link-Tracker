package external

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPClient_GitHubNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewHTTPClient(time.Second)
	client.githubBaseURL = srv.URL
	_, err := client.GetLastUpdated(context.Background(), "https://github.com/user/repo")
	if err == nil {
		t.Fatalf("expected error for non-2xx github response")
	}
}

func TestHTTPClient_StackOverflowInvalidSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[{"unexpected":1}]}`))
	}))
	defer srv.Close()

	client := NewHTTPClient(time.Second)
	client.stackBaseURL = srv.URL
	_, err := client.GetLastUpdated(context.Background(), "https://stackoverflow.com/questions/123/title")
	if err == nil {
		t.Fatalf("expected error for invalid stackoverflow schema")
	}
}
