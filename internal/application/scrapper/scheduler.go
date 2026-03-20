package scrapper

import (
	"context"
	"log/slog"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/botclient"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/external"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type Scheduler struct {
	store      *repository.Service
	external   external.LastUpdatedClient
	botUpdates botclient.UpdatesSender
	interval   time.Duration
}

func NewScheduler(store *repository.Service, externalClient external.LastUpdatedClient, updatesClient botclient.UpdatesSender, interval time.Duration) *Scheduler {
	return &Scheduler{
		store:      store,
		external:   externalClient,
		botUpdates: updatesClient,
		interval:   interval,
	}
}

func (s *Scheduler) ProcessOnce(ctx context.Context) {
	links := s.store.SnapshotLinks()
	for _, link := range links {
		logger := slog.With("url", link.Response.URL)
		lastUpdated, err := s.external.GetLastUpdated(ctx, link.Response.URL)
		if err != nil {
			logger.Warn("Failed to fetch last updated from external source", slog.String("error", err.Error()))
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
		sendErr := s.botUpdates.SendUpdate(ctx, update)
		if sendErr != nil {
			logger.Warn("Failed to send update to bot", slog.String("error", sendErr.Error()))
			continue
		}
		s.store.UpdateLastUpdated(link.Response.URL, lastUpdated)
	}
}
