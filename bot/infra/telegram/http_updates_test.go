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
			name:       "valid payload returns 200",
			body:       `{"id":1,"url":"https://github.com/org/repo","description":"u","tgChatIds":[1,2]}`,
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
