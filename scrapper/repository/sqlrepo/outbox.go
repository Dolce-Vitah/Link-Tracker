//nolint:noctx // Repository contract does not accept context; data-layer calls use internal defaults.
package sqlrepo

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (r *Repository) StoreLinkUpdates(linkURL string, updates []trackerapi.LinkUpdate, newLastUpdated time.Time, checkedAt time.Time) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin outbox tx: %w", err)
	}
	defer rollback(tx)

	for _, update := range updates {
		payload, marshalErr := json.Marshal(update)
		if marshalErr != nil {
			return fmt.Errorf("marshal outbox payload: %w", marshalErr)
		}
		if _, execErr := tx.Exec(`INSERT INTO outbox_messages(event_type, payload) VALUES ($1, $2)`, repository.EventTypeLinkUpdate, payload); execErr != nil {
			return fmt.Errorf("insert outbox message: %w", execErr)
		}
	}

	if !newLastUpdated.IsZero() {
		if _, updErr := tx.Exec(`UPDATE links SET last_updated = $2, last_checked_at = $3, updated_at = NOW() WHERE url = $1`, linkURL, newLastUpdated, checkedAt); updErr != nil {
			return fmt.Errorf("update link state with outbox: %w", updErr)
		}
	} else {
		if _, touchErr := tx.Exec(`UPDATE links SET last_checked_at = $2, updated_at = NOW() WHERE url = $1`, linkURL, checkedAt); touchErr != nil {
			return fmt.Errorf("touch link state with outbox: %w", touchErr)
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("commit outbox tx: %w", commitErr)
	}
	return nil
}

func (r *Repository) FetchPendingOutbox(limit int, now time.Time) ([]repository.OutboxMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(`
		SELECT id, event_type, payload, attempts, next_attempt_at, created_at
		FROM outbox_messages
		WHERE sent_at IS NULL AND next_attempt_at <= $1
		ORDER BY id ASC
		LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending outbox: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]repository.OutboxMessage, 0, limit)
	for rows.Next() {
		var m repository.OutboxMessage
		if scanErr := rows.Scan(&m.ID, &m.EventType, &m.Payload, &m.Attempts, &m.NextAttempt, &m.CreatedAt); scanErr != nil {
			return nil, fmt.Errorf("scan outbox row: %w", scanErr)
		}
		out = append(out, m)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate outbox rows: %w", rowsErr)
	}
	return out, nil
}

func (r *Repository) MarkOutboxSent(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin mark sent tx: %w", err)
	}
	defer rollback(tx)
	for _, id := range ids {
		if _, execErr := tx.Exec(`UPDATE outbox_messages SET sent_at = NOW() WHERE id = $1`, id); execErr != nil {
			return fmt.Errorf("mark outbox sent %d: %w", id, execErr)
		}
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("commit mark sent tx: %w", commitErr)
	}
	return nil
}

func (r *Repository) MarkOutboxRetry(id int64, nextAttempt time.Time, reason string) error {
	res, err := r.db.Exec(`
		UPDATE outbox_messages
		SET attempts = attempts + 1,
		    next_attempt_at = $2,
		    last_error = $3
		WHERE id = $1
	`, id, nextAttempt, reason)
	if err != nil {
		return fmt.Errorf("mark outbox retry: %w", err)
	}
	affected, affErr := res.RowsAffected()
	if affErr != nil {
		return fmt.Errorf("mark outbox retry affected: %w", affErr)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
