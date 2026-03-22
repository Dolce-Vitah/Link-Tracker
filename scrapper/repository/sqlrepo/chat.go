//nolint:noctx // Repository contract does not accept context; data-layer calls use internal defaults.
package sqlrepo

import (
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (r *Repository) RegisterChat(chatID int64) error {
	_, err := r.db.Exec(`INSERT INTO chats(chat_id) VALUES ($1)`, chatID)
	if err == nil {
		return nil
	}
	if isUniqueViolation(err) {
		return repository.ErrChatExists
	}
	return fmt.Errorf("register chat: %w", err)
}

func (r *Repository) DeleteChat(chatID int64) error {
	res, err := r.db.Exec(`DELETE FROM chats WHERE chat_id = $1`, chatID)
	if err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}
	affected, affErr := res.RowsAffected()
	if affErr != nil {
		return fmt.Errorf("delete chat rows affected: %w", affErr)
	}
	if affected == 0 {
		return repository.ErrChatNotFound
	}
	_, cleanupErr := r.db.Exec(`
		DELETE FROM links
		WHERE NOT EXISTS (SELECT 1 FROM chat_links cl WHERE cl.link_id = links.id)
	`)
	if cleanupErr != nil {
		return fmt.Errorf("cleanup orphan links after delete chat: %w", cleanupErr)
	}
	return nil
}
