package telegram

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestBot_HandleLinkUpdateHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name: "valid payload returns 200",
			body: `{"id":1,"url":"https://github.com/org/repo","tgChatIds":[1,2],` +
				`"eventKind":"github_issue","title":"T","author":"a","createdAt":"2026-01-01T00:00:00Z","preview":"p"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid payload returns 400",
			body:       `{"id":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing required fields returns 400",
			body:       `{"id":1,"description":"u","tgChatIds":[]}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bot := &Bot{
				sendMessage: func(_ tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				},
			}
			req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader([]byte(tt.body)))
			rec := httptest.NewRecorder()

			bot.handleLinkUpdateHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestBot_HandleProcessingFailuresHTTP_valid(t *testing.T) {
	t.Parallel()

	bot := &Bot{
		sendMessage: func(_ tgbotapi.Chattable) (tgbotapi.Message, error) {
			return tgbotapi.Message{}, nil
		},
	}
	body := `{"tgChatId":99,"urls":["https://a.com"],"detail":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/updates/failures", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()
	bot.handleProcessingFailuresHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestBot_HandleLinkUpdateHTTP_MethodIsRejectedByRoute(t *testing.T) {
	t.Parallel()

	bot := &Bot{
		sendMessage: func(_ tgbotapi.Chattable) (tgbotapi.Message, error) {
			return tgbotapi.Message{}, nil
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /updates", bot.handleLinkUpdateHTTP)

	req := httptest.NewRequest(http.MethodGet, "/updates", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}
