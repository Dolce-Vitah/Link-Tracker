package ormrepo

import (
	"fmt"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (r *Repository) ListTrackedLinksPage(limit int, checkedBefore time.Time, afterID int64) ([]repository.TrackedLink, error) {
	if limit <= 0 {
		limit = 100
	}
	var links []LinkModel
	if err := r.db.
		Where("last_checked_at IS NULL OR last_checked_at <= ?", checkedBefore).
		Where("id > ?", afterID).
		Order("id ASC").
		Limit(limit).
		Find(&links).Error; err != nil {
		return nil, fmt.Errorf("list tracked links orm: %w", err)
	}

	out := make([]repository.TrackedLink, 0, len(links))
	for _, link := range links {
		var chatLinks []ChatLinkModel
		if err := r.db.Where("link_id = ?", link.ID).Find(&chatLinks).Error; err != nil {
			return nil, fmt.Errorf("load chat links for tracked link orm: %w", err)
		}
		item := repository.TrackedLink{
			Response: trackerapi.LinkResponse{
				ID:  link.ID,
				URL: link.URL,
			},
			ChatIDs: make(map[int64]struct{}, len(chatLinks)),
		}
		if link.LastUpdated != nil {
			item.LastUpdated = *link.LastUpdated
		}
		if link.LastCheckedAt != nil {
			item.LastCheckedAt = *link.LastCheckedAt
		}
		for _, chatLink := range chatLinks {
			item.ChatIDs[chatLink.ChatID] = struct{}{}
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *Repository) UpdateLastUpdated(linkURL string, value time.Time) error {
	res := r.db.Model(&LinkModel{}).Where("url = ?", linkURL).Updates(map[string]any{
		"last_updated": value,
		"updated_at":   time.Now().UTC(),
	})
	if res.Error != nil {
		return fmt.Errorf("update last_updated orm: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrLinkNotFound
	}
	return nil
}

func (r *Repository) TouchLastChecked(linkURL string, value time.Time) error {
	res := r.db.Model(&LinkModel{}).Where("url = ?", linkURL).Updates(map[string]any{
		"last_checked_at": value,
		"updated_at":      time.Now().UTC(),
	})
	if res.Error != nil {
		return fmt.Errorf("touch last_checked_at orm: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrLinkNotFound
	}
	return nil
}
