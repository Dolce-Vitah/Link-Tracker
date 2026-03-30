package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

type requestSpec struct {
	method string
	path   string
	body   string
	chatID string
}

func newTestHandler() http.Handler {
	return New(repository.NewService()).Handler()
}

func performRequest(t *testing.T, h http.Handler, req requestSpec) *httptest.ResponseRecorder {
	t.Helper()

	var body *bytes.Buffer
	if req.body != "" {
		body = bytes.NewBufferString(req.body)
	} else {
		body = bytes.NewBuffer(nil)
	}

	httpReq := httptest.NewRequest(req.method, req.path, body)
	if req.chatID != "" {
		httpReq.Header.Set("Tg-Chat-Id", req.chatID)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httpReq)
	return rec
}

func mustStatus(t *testing.T, h http.Handler, req requestSpec, want int) {
	t.Helper()

	rec := performRequest(t, h, req)
	if rec.Code != want {
		t.Fatalf("%s %s status=%d want=%d", req.method, req.path, rec.Code, want)
	}
}

func listLinksSize(t *testing.T, h http.Handler, chatID string) int32 {
	t.Helper()

	rec := performRequest(t, h, requestSpec{
		method: http.MethodGet,
		path:   "/links",
		chatID: chatID,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /links status=%d want=200", rec.Code)
	}

	var payload trackerapi.ListLinksResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	return payload.Size
}

func TestHTTPServer_StatusScenarios_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		prepare    func(t *testing.T, h http.Handler)
		request    requestSpec
		wantStatus int
	}{
		{
			name: "add link for unknown chat returns not found",
			request: requestSpec{
				method: http.MethodPost,
				path:   "/links",
				body:   `{"link":"https://github.com/user/repo"}`,
				chatID: "2",
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "add link after chat deletion returns not found",
			prepare: func(t *testing.T, h http.Handler) {
				t.Helper()

				mustStatus(t, h, requestSpec{method: http.MethodPost, path: "/tg-chat/1"}, http.StatusOK)

				mustStatus(t, h, requestSpec{method: http.MethodDelete, path: "/tg-chat/1"}, http.StatusOK)
			},
			request: requestSpec{
				method: http.MethodPost,
				path:   "/links",
				body:   `{"link":"https://github.com/user/repo"}`,
				chatID: "1",
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "delete non-existent chat returns not found",
			request: requestSpec{
				method: http.MethodDelete,
				path:   "/tg-chat/1",
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "register existing chat returns conflict",
			prepare: func(t *testing.T, h http.Handler) {
				t.Helper()

				mustStatus(t, h, requestSpec{method: http.MethodPost, path: "/tg-chat/1"}, http.StatusOK)
			},
			request: requestSpec{
				method: http.MethodPost,
				path:   "/tg-chat/1",
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "add duplicate link returns conflict",
			prepare: func(t *testing.T, h http.Handler) {
				t.Helper()

				mustStatus(t, h, requestSpec{method: http.MethodPost, path: "/tg-chat/1"}, http.StatusOK)

				mustStatus(t, h, requestSpec{
					method: http.MethodPost,
					path:   "/links",
					body:   `{"link":"https://github.com/user/repo"}`,
					chatID: "1",
				}, http.StatusOK)
			},
			request: requestSpec{
				method: http.MethodPost,
				path:   "/links",
				body:   `{"link":"https://github.com/user/repo"}`,
				chatID: "1",
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newTestHandler()
			if tt.prepare != nil {
				tt.prepare(t, h)
			}

			mustStatus(t, h, tt.request, tt.wantStatus)
		})
	}
}

func TestHTTPServer_LinkStateScenarios_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		exercise     func(t *testing.T, h http.Handler)
		verifyChatID string
		wantSize     int32
	}{
		{
			name: "chat and links lifecycle ends with empty list",
			exercise: func(t *testing.T, h http.Handler) {
				t.Helper()

				mustStatus(t, h, requestSpec{method: http.MethodPost, path: "/tg-chat/1"}, http.StatusOK)

				mustStatus(t, h, requestSpec{
					method: http.MethodPost,
					path:   "/links",
					body:   `{"link":"https://github.com/user/repo","tags":["work"]}`,
					chatID: "1",
				}, http.StatusOK)

				mustStatus(t, h, requestSpec{
					method: http.MethodDelete,
					path:   "/links",
					body:   `{"link":"https://github.com/user/repo"}`,
					chatID: "1",
				}, http.StatusOK)
			},
			verifyChatID: "1",
			wantSize:     0,
		},
		{
			name: "delete from unknown chat does not affect existing links",
			exercise: func(t *testing.T, h http.Handler) {
				t.Helper()

				mustStatus(t, h, requestSpec{method: http.MethodPost, path: "/tg-chat/1"}, http.StatusOK)

				mustStatus(t, h, requestSpec{
					method: http.MethodPost,
					path:   "/links",
					body:   `{"link":"https://github.com/user/repo"}`,
					chatID: "1",
				}, http.StatusOK)

				mustStatus(t, h, requestSpec{
					method: http.MethodDelete,
					path:   "/links",
					body:   `{"link":"https://github.com/user/repo"}`,
					chatID: "999",
				}, http.StatusNotFound)
			},
			verifyChatID: "1",
			wantSize:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newTestHandler()
			tt.exercise(t, h)

			gotSize := listLinksSize(t, h, tt.verifyChatID)
			if gotSize != tt.wantSize {
				t.Fatalf("list size=%d want=%d", gotSize, tt.wantSize)
			}
		})
	}
}
