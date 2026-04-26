package repositorytest

import (
	"encoding/json"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (s *Store) StoreLinkUpdates(linkURL string, updates []trackerapi.LinkUpdate, newLastUpdated time.Time, checkedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tracked, ok := s.linksByURL[linkURL]
	if !ok {
		return repository.ErrLinkNotFound
	}
	for _, update := range updates {
		s.nextOutboxID++
		s.outbox = append(s.outbox, repository.OutboxMessage{ID: s.nextOutboxID, EventType: repository.EventTypeLinkUpdate, CreatedAt: time.Now().UTC(), NextAttempt: time.Now().UTC(), Payload: mustJSON(update)})
	}
	if !newLastUpdated.IsZero() {
		tracked.tracked.LastUpdated = newLastUpdated
	}
	tracked.tracked.LastCheckedAt = checkedAt
	return nil
}

func (s *Store) FetchPendingOutbox(limit int, now time.Time) ([]repository.OutboxMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = len(s.outbox)
	}
	out := make([]repository.OutboxMessage, 0, limit)
	for _, msg := range s.outbox {
		if msg.NextAttempt.After(now) {
			continue
		}
		out = append(out, msg)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (s *Store) MarkOutboxSent(ids []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		for i := range s.outbox {
			if s.outbox[i].ID == id {
				s.outbox = append(s.outbox[:i], s.outbox[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (s *Store) MarkOutboxRetry(id int64, nextAttempt time.Time, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.outbox {
		if s.outbox[i].ID == id {
			s.outbox[i].Attempts++
			s.outbox[i].NextAttempt = nextAttempt
			break
		}
	}
	return nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
