package ormrepo

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"gorm.io/gorm"
)

func (r *Repository) StoreLinkUpdates(linkURL string, updates []trackerapi.LinkUpdate, newLastUpdated time.Time, checkedAt time.Time) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			payload, marshalErr := json.Marshal(update)
			if marshalErr != nil {
				return fmt.Errorf("marshal outbox payload orm: %w", marshalErr)
			}
			if err := tx.Create(&OutboxMessageModel{EventType: repository.EventTypeLinkUpdate, Payload: payload}).Error; err != nil {
				return fmt.Errorf("insert outbox message orm: %w", err)
			}
		}
		state := map[string]any{"last_checked_at": checkedAt, "updated_at": time.Now().UTC()}
		if !newLastUpdated.IsZero() {
			state["last_updated"] = newLastUpdated
		}
		res := tx.Model(&LinkModel{}).Where("url = ?", linkURL).Updates(state)
		if res.Error != nil {
			return fmt.Errorf("update link state with outbox orm: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return repository.ErrLinkNotFound
		}
		return nil
	})
}

func (r *Repository) FetchPendingOutbox(limit int, now time.Time) ([]repository.OutboxMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []OutboxMessageModel
	if err := r.db.Where("sent_at IS NULL AND next_attempt_at <= ?", now).Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("fetch pending outbox orm: %w", err)
	}
	out := make([]repository.OutboxMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, repository.OutboxMessage{ID: row.ID, EventType: row.EventType, Payload: row.Payload, Attempts: row.Attempts, NextAttempt: row.NextAttemptAt, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (r *Repository) MarkOutboxSent(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if err := r.db.Model(&OutboxMessageModel{}).Where("id IN ?", ids).Update("sent_at", time.Now().UTC()).Error; err != nil {
		return fmt.Errorf("mark outbox sent orm: %w", err)
	}
	return nil
}

func (r *Repository) MarkOutboxRetry(id int64, nextAttempt time.Time, reason string) error {
	res := r.db.Model(&OutboxMessageModel{}).Where("id = ?", id).Updates(map[string]any{
		"attempts":        gorm.Expr("attempts + 1"),
		"next_attempt_at": nextAttempt,
		"last_error":      reason,
	})
	if res.Error != nil {
		return fmt.Errorf("mark outbox retry orm: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
