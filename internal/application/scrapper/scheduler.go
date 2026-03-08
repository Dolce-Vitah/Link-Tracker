package scrapper

import (
	"context"
	"log/slog"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/botclient"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/external"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type Scheduler struct {
	store      *Service
	external   external.LastUpdatedClient
	botUpdates botclient.UpdatesSender
	interval   time.Duration
}

func NewScheduler(store *Service, externalClient external.LastUpdatedClient, updatesClient botclient.UpdatesSender, interval time.Duration) *Scheduler {
	return &Scheduler{
		store:      store,
		external:   externalClient,
		botUpdates: updatesClient,
		interval:   interval,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.ProcessOnce(ctx)
		}
	}
}

func (s *Scheduler) ProcessOnce(ctx context.Context) {
	links := s.store.SnapshotLinks()
	for _, link := range links {
		lastUpdated, err := s.external.GetLastUpdated(ctx, link.Response.URL)
		if err != nil {
			slog.Warn("Failed to fetch last updated from external source",
				slog.String("url", link.Response.URL),
				slog.String("error", err.Error()),
			)
			continue
		}
		if !link.LastUpdated.IsZero() && !lastUpdated.After(link.LastUpdated) {
			continue
		}

		chatIDs := make([]int64, 0, len(link.ChatIDs))
		for chatID := range link.ChatIDs {
			chatIDs = append(chatIDs, chatID)
		}

		update := api.LinkUpdate{
			ID:          link.Response.ID,
			URL:         link.Response.URL,
			Description: "Обнаружено новое обновление",
			TgChatIDs:   chatIDs,
		}
		if err := s.botUpdates.SendUpdate(ctx, update); err != nil {
			slog.Warn("Failed to send update to bot",
				slog.String("url", link.Response.URL),
				slog.String("error", err.Error()),
			)
			continue
		}
		s.store.UpdateLastUpdated(link.Response.URL, lastUpdated)
	}
}
