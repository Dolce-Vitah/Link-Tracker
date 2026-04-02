package scheduler

import (
	"context"
	"log/slog"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/botclient"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/external"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

type Scheduler struct {
	store      repository.TrackedLinkService
	external   external.LastUpdatedClient
	botUpdates botclient.UpdatesSender
	interval   time.Duration
}

func New(
	store repository.TrackedLinkService,
	externalClient external.LastUpdatedClient,
	updatesClient botclient.UpdatesSender,
	interval time.Duration,
) *Scheduler {
	return &Scheduler{
		store:      store,
		external:   externalClient,
		botUpdates: updatesClient,
		interval:   interval,
	}
}

//nolint:gocognit // Scheduler flow intentionally handles all update branches explicitly.
func (s *Scheduler) ProcessOnce(ctx context.Context) {
	const pageSize = 100
	var afterID int64
	checkedBefore := time.Now().UTC().Add(-s.interval)
	for {
		links, err := s.store.ListTrackedLinksPage(pageSize, checkedBefore, afterID)
		if err != nil {
			slog.Error("Failed to load tracked links page", slog.String("error", err.Error()))
			return
		}
		if len(links) == 0 {
			return
		}
		for _, link := range links {
			logger := slog.With("url", link.Response.URL)
			lastUpdated, fetchErr := s.external.GetLastUpdated(ctx, link.Response.URL)
			if fetchErr != nil {
				logger.Warn("Failed to fetch last updated from external source", slog.String("error", fetchErr.Error()))
				if touchErr := s.store.TouchLastChecked(link.Response.URL, time.Now().UTC()); touchErr != nil {
					logger.Warn("Failed to mark last_checked_at after external error", slog.String("error", touchErr.Error()))
				}
				continue
			}
			if !link.LastUpdated.IsZero() && !lastUpdated.After(link.LastUpdated) {
				if touchErr := s.store.TouchLastChecked(link.Response.URL, time.Now().UTC()); touchErr != nil {
					logger.Warn("Failed to mark last_checked_at when no updates", slog.String("error", touchErr.Error()))
				}
				continue
			}

			chatIDs := make([]int64, 0, len(link.ChatIDs))
			for chatID := range link.ChatIDs {
				chatIDs = append(chatIDs, chatID)
			}

			update := trackerapi.LinkUpdate{
				ID:          link.Response.ID,
				URL:         link.Response.URL,
				Description: "Обнаружено новое обновление",
				TgChatIDs:   chatIDs,
			}
			sendErr := s.botUpdates.SendUpdate(ctx, update)
			if sendErr != nil {
				logger.Warn("Failed to send update to bot", slog.String("error", sendErr.Error()))
				if touchErr := s.store.TouchLastChecked(link.Response.URL, time.Now().UTC()); touchErr != nil {
					logger.Warn("Failed to mark last_checked_at after send error", slog.String("error", touchErr.Error()))
				}
				continue
			}
			if updErr := s.store.UpdateLastUpdated(link.Response.URL, lastUpdated); updErr != nil {
				logger.Warn("Failed to update last_updated", slog.String("error", updErr.Error()))
			}
			if touchErr := s.store.TouchLastChecked(link.Response.URL, time.Now().UTC()); touchErr != nil {
				logger.Warn("Failed to mark last_checked_at after successful send", slog.String("error", touchErr.Error()))
			}
		}
		afterID = links[len(links)-1].Response.ID
	}
}
