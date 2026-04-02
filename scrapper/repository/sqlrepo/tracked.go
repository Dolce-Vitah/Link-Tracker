//nolint:noctx // Repository contract does not accept context; data-layer calls use internal defaults.
package sqlrepo

import (
	"database/sql"
	"fmt"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (r *Repository) ListTrackedLinksPage(limit int, checkedBefore time.Time, afterID int64) ([]repository.TrackedLink, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.Query(`
		SELECT l.id, l.url, l.last_updated, l.last_checked_at
		FROM links l
		WHERE (l.last_checked_at IS NULL OR l.last_checked_at <= $1)
		  AND l.id > $2
		ORDER BY l.id ASC
		LIMIT $3
	`, checkedBefore, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("list tracked links page: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]repository.TrackedLink, 0, limit)
	for rows.Next() {
		var (
			linkID        int64
			urlValue      string
			lastUpdated   sql.NullTime
			lastCheckedAt sql.NullTime
		)
		if scanErr := rows.Scan(&linkID, &urlValue, &lastUpdated, &lastCheckedAt); scanErr != nil {
			return nil, fmt.Errorf("scan tracked link row: %w", scanErr)
		}
		chatIDs, chatErr := r.selectChatIDs(linkID)
		if chatErr != nil {
			return nil, chatErr
		}
		item := repository.TrackedLink{
			Response: trackerapi.LinkResponse{
				ID:  linkID,
				URL: urlValue,
			},
			ChatIDs: make(map[int64]struct{}, len(chatIDs)),
		}
		if lastUpdated.Valid {
			item.LastUpdated = lastUpdated.Time
		}
		if lastCheckedAt.Valid {
			item.LastCheckedAt = lastCheckedAt.Time
		}
		for _, chatID := range chatIDs {
			item.ChatIDs[chatID] = struct{}{}
		}
		out = append(out, item)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate tracked links page: %w", rowsErr)
	}
	return out, nil
}

func (r *Repository) UpdateLastUpdated(linkURL string, value time.Time) error {
	res, err := r.db.Exec(`UPDATE links SET last_updated = $2, updated_at = NOW() WHERE url = $1`, linkURL, value)
	if err != nil {
		return fmt.Errorf("update last_updated: %w", err)
	}
	affected, affErr := res.RowsAffected()
	if affErr != nil {
		return fmt.Errorf("update last_updated rows affected: %w", affErr)
	}
	if affected == 0 {
		return repository.ErrLinkNotFound
	}
	return nil
}

func (r *Repository) TouchLastChecked(linkURL string, value time.Time) error {
	res, err := r.db.Exec(`UPDATE links SET last_checked_at = $2, updated_at = NOW() WHERE url = $1`, linkURL, value)
	if err != nil {
		return fmt.Errorf("touch last_checked_at: %w", err)
	}
	affected, affErr := res.RowsAffected()
	if affErr != nil {
		return fmt.Errorf("touch last_checked_at rows affected: %w", affErr)
	}
	if affected == 0 {
		return repository.ErrLinkNotFound
	}
	return nil
}
