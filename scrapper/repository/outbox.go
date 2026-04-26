package repository

import (
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
)

const EventTypeLinkUpdate = "link.update"

type OutboxMessage struct {
	ID          int64
	EventType   string
	Payload     []byte
	Attempts    int
	NextAttempt time.Time
	CreatedAt   time.Time
}

type OutboxService interface {
	StoreLinkUpdates(linkURL string, updates []trackerapi.LinkUpdate, newLastUpdated time.Time, checkedAt time.Time) error
	FetchPendingOutbox(limit int, now time.Time) ([]OutboxMessage, error)
	MarkOutboxSent(ids []int64) error
	MarkOutboxRetry(id int64, nextAttempt time.Time, reason string) error
}
