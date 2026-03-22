package repositorytest

import (
	"sort"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (s *Store) ListTrackedLinksPage(limit int, checkedBefore time.Time, afterID int64) ([]repository.TrackedLink, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = len(s.linksByURL)
	}
	all := make([]repository.TrackedLink, 0, len(s.linksByURL))
	for _, link := range s.linksByURL {
		if link.tracked.Response.ID <= afterID {
			continue
		}
		if !checkedBefore.IsZero() && !link.tracked.LastCheckedAt.IsZero() && link.tracked.LastCheckedAt.After(checkedBefore) {
			continue
		}
		clone := repository.TrackedLink{
			Response:      link.tracked.Response,
			ChatIDs:       make(map[int64]struct{}, len(link.tracked.ChatIDs)),
			LastUpdated:   link.tracked.LastUpdated,
			LastCheckedAt: link.tracked.LastCheckedAt,
		}
		for chatID := range link.tracked.ChatIDs {
			clone.ChatIDs[chatID] = struct{}{}
		}
		all = append(all, clone)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Response.ID < all[j].Response.ID
	})

	end := limit
	if end > len(all) {
		end = len(all)
	}
	return all[:end], nil
}

func (s *Store) UpdateLastUpdated(linkURL string, value time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tracked, ok := s.linksByURL[linkURL]
	if !ok {
		return repository.ErrLinkNotFound
	}
	tracked.tracked.LastUpdated = value
	return nil
}

func (s *Store) TouchLastChecked(linkURL string, value time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tracked, ok := s.linksByURL[linkURL]
	if !ok {
		return repository.ErrLinkNotFound
	}
	tracked.tracked.LastCheckedAt = value
	return nil
}
