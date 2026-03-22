//nolint:noctx // Repository contract does not accept context; data-layer calls use internal defaults.
package sqlrepo

import (
	"database/sql"
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (r *Repository) selectTagNames(chatLinkID int64) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT t.name
		FROM chat_link_tags clt
		JOIN tags t ON t.id = clt.tag_id
		WHERE clt.chat_link_id = $1
		ORDER BY t.id ASC
	`, chatLinkID)
	if err != nil {
		return nil, fmt.Errorf("select tags for chat link: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	out := make([]string, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scan tag name: %w", scanErr)
		}
		out = append(out, name)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate tag names: %w", rowsErr)
	}
	return out, nil
}

func selectTagNamesTx(tx *sql.Tx, chatLinkID int64) ([]string, error) {
	rows, err := tx.Query(`
		SELECT t.name
		FROM chat_link_tags clt
		JOIN tags t ON t.id = clt.tag_id
		WHERE clt.chat_link_id = $1
		ORDER BY t.id ASC
	`, chatLinkID)
	if err != nil {
		return nil, fmt.Errorf("select tags in tx: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	out := make([]string, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scan tx tag: %w", scanErr)
		}
		out = append(out, name)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate tx tags: %w", rowsErr)
	}
	return out, nil
}

func upsertTagTx(tx *sql.Tx, chatID int64, rawTag string) (int64, error) {
	tag := normalizeTag(rawTag)
	if tag == "" {
		return 0, repository.ErrInvalidTag
	}
	var id int64
	err := tx.QueryRow(`
		INSERT INTO tags(chat_id, name)
		VALUES ($1, $2)
		ON CONFLICT (chat_id, name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, chatID, tag).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("upsert tag: %w", err)
	}
	return id, nil
}

func (r *Repository) selectChatIDs(linkID int64) ([]int64, error) {
	rows, err := r.db.Query(`SELECT chat_id FROM chat_links WHERE link_id = $1`, linkID)
	if err != nil {
		return nil, fmt.Errorf("select chat ids by link: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	out := make([]int64, 0)
	for rows.Next() {
		var chatID int64
		if scanErr := rows.Scan(&chatID); scanErr != nil {
			return nil, fmt.Errorf("scan chat id: %w", scanErr)
		}
		out = append(out, chatID)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate chat ids: %w", rowsErr)
	}
	return out, nil
}

func (r *Repository) chatExists(chatID int64) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM chats WHERE chat_id = $1)`, chatID).Scan(&exists); err != nil {
		return false, fmt.Errorf("scan chat exists: %w", err)
	}
	return exists, nil
}
