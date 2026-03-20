package telegram

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestBot_HandleLinkUpdateHTTP(t *testing.T) {
	t.Run("valid payload returns 200", func(t *testing.T) {
		bot := &Bot{
			sendMessage: func(_ tgbotapi.Chattable) (tgbotapi.Message, error) {
				return tgbotapi.Message{}, nil
			},
		}
		body := []byte(`{"id":1,"url":"https://github.com/org/repo","description":"u","tgChatIds":[1,2]}`)
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		bot.handleLinkUpdateHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("invalid payload returns 400", func(t *testing.T) {
		bot := &Bot{
			sendMessage: func(_ tgbotapi.Chattable) (tgbotapi.Message, error) {
				return tgbotapi.Message{}, nil
			},
		}
		body := []byte(`{"id":`)
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		bot.handleLinkUpdateHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		bot := &Bot{
			sendMessage: func(_ tgbotapi.Chattable) (tgbotapi.Message, error) {
				return tgbotapi.Message{}, nil
			},
		}
		body := []byte(`{"id":1,"description":"u","tgChatIds":[]}`)
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		bot.handleLinkUpdateHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}
