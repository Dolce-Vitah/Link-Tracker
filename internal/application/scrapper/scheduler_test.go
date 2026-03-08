package scrapper

import (
	"context"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type fakeExternalClient struct {
	lastUpdated time.Time
	err         error
}

func (f *fakeExternalClient) GetLastUpdated(ctx context.Context, rawURL string) (time.Time, error) {
	return f.lastUpdated, f.err
}

type fakeUpdatesSender struct {
	updates []api.LinkUpdate
	err     error
}

func (f *fakeUpdatesSender) SendUpdate(ctx context.Context, update api.LinkUpdate) error {
	f.updates = append(f.updates, update)
	return f.err
}

func TestScheduler_SendsUpdateOnlyToSubscribedChats(t *testing.T) {
	service := NewService()
	if err := service.RegisterChat(1); err != nil {
		t.Fatalf("register chat 1: %v", err)
	}
	if err := service.RegisterChat(2); err != nil {
		t.Fatalf("register chat 2: %v", err)
	}
	if err := service.RegisterChat(999); err != nil {
		t.Fatalf("register chat 999: %v", err)
	}

	if _, err := service.AddLink(1, api.AddLinkRequest{Link: "https://github.com/user/repo"}); err != nil {
		t.Fatalf("add link for chat 1: %v", err)
	}
	if _, err := service.AddLink(2, api.AddLinkRequest{Link: "https://github.com/user/repo"}); err != nil {
		t.Fatalf("add link for chat 2: %v", err)
	}

	external := &fakeExternalClient{lastUpdated: time.Now().Add(10 * time.Minute)}
	updates := &fakeUpdatesSender{}
	scheduler := NewScheduler(service, external, updates, time.Second)

	scheduler.ProcessOnce(context.Background())

	if len(updates.updates) != 1 {
		t.Fatalf("expected exactly one update, got %d", len(updates.updates))
	}
	got := updates.updates[0]
	if len(got.TgChatIDs) != 2 {
		t.Fatalf("expected update for 2 subscribed chats, got %d", len(got.TgChatIDs))
	}

	hasChat := func(id int64) bool {
		for _, chatID := range got.TgChatIDs {
			if chatID == id {
				return true
			}
		}
		return false
	}
	if !hasChat(1) || !hasChat(2) {
		t.Fatalf("expected chat IDs 1 and 2 in update, got %+v", got.TgChatIDs)
	}
	if hasChat(999) {
		t.Fatalf("unexpected chat ID 999 in update recipients")
	}
}
