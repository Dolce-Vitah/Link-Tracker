package poller

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/botclient"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

type OutboxDispatcher struct {
	store   repository.OutboxService
	sender  botclient.MessageSender
	batch   int
	retryIn time.Duration
}

func NewOutboxDispatcher(store repository.OutboxService, sender botclient.MessageSender, batch int, retryIn time.Duration) *OutboxDispatcher {
	if batch <= 0 {
		batch = 100
	}
	if retryIn <= 0 {
		retryIn = 2 * time.Second
	}
	return &OutboxDispatcher{store: store, sender: sender, batch: batch, retryIn: retryIn}
}

func (d *OutboxDispatcher) DispatchOnce(ctx context.Context) {
	pending, err := d.store.FetchPendingOutbox(d.batch, time.Now().UTC())
	if err != nil {
		slog.Warn("Failed to fetch outbox batch", slog.String("error", err.Error()))
		return
	}
	sentIDs := make([]int64, 0, len(pending))
	for _, msg := range pending {
		if msg.EventType != repository.EventTypeLinkUpdate {
			sentIDs = append(sentIDs, msg.ID)
			continue
		}
		var update trackerapi.LinkUpdate
		if decodeErr := json.Unmarshal(msg.Payload, &update); decodeErr != nil {
			_ = d.store.MarkOutboxRetry(msg.ID, time.Now().UTC().Add(d.retryIn), decodeErr.Error())
			continue
		}
		if sendErr := d.sender.SendLinkUpdate(ctx, update); sendErr != nil {
			_ = d.store.MarkOutboxRetry(msg.ID, time.Now().UTC().Add(d.retryIn), sendErr.Error())
			continue
		}
		sentIDs = append(sentIDs, msg.ID)
	}
	if err := d.store.MarkOutboxSent(sentIDs); err != nil {
		slog.Warn("Failed to mark outbox sent", slog.String("error", err.Error()))
	}
}
