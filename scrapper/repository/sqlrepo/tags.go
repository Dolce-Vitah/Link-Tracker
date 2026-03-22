//nolint:noctx // Repository contract does not accept context; data-layer calls use internal defaults.
package sqlrepo

import (
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (r *Repository) CreateTag(chatID int64, rawTag string) (repository.Tag, error) {
	tag := normalizeTag(rawTag)
	if tag == "" {
		return repository.Tag{}, repository.ErrInvalidTag
	}
	chatExists, err := r.chatExists(chatID)
	if err != nil {
		return repository.Tag{}, fmt.Errorf("check chat exists before create tag: %w", err)
	}
	if !chatExists {
		return repository.Tag{}, repository.ErrChatNotFound
	}

	var id int64
	err = r.db.QueryRow(`INSERT INTO tags(chat_id, name) VALUES ($1, $2) RETURNING id`, chatID, tag).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return repository.Tag{}, repository.ErrTagExists
		}
		if isForeignKeyViolation(err) {
			return repository.Tag{}, repository.ErrChatNotFound
		}
		return repository.Tag{}, fmt.Errorf("create tag: %w", err)
	}
	return repository.Tag{ID: id, ChatID: chatID, Name: tag}, nil
}

func (r *Repository) ListTags(chatID int64, limit int, offset int) ([]repository.Tag, error) {
	chatExists, err := r.chatExists(chatID)
	if err != nil {
		return nil, fmt.Errorf("check chat exists before list tags: %w", err)
	}
	if !chatExists {
		return nil, repository.ErrChatNotFound
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(`
		SELECT id, name
		FROM tags
		WHERE chat_id = $1
		ORDER BY id ASC
		LIMIT $2 OFFSET $3
	`, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]repository.Tag, 0, limit)
	for rows.Next() {
		var (
			id   int64
			name string
		)
		if scanErr := rows.Scan(&id, &name); scanErr != nil {
			return nil, fmt.Errorf("scan listed tag: %w", scanErr)
		}
		out = append(out, repository.Tag{ID: id, ChatID: chatID, Name: name})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate listed tags: %w", rowsErr)
	}
	return out, nil
}

func (r *Repository) UpdateTag(chatID int64, oldTag string, newTag string) (repository.Tag, error) {
	oldName := normalizeTag(oldTag)
	newName := normalizeTag(newTag)
	if oldName == "" || newName == "" {
		return repository.Tag{}, repository.ErrInvalidTag
	}
	chatExists, err := r.chatExists(chatID)
	if err != nil {
		return repository.Tag{}, fmt.Errorf("check chat exists before update tag: %w", err)
	}
	if !chatExists {
		return repository.Tag{}, repository.ErrChatNotFound
	}
	res, err := r.db.Exec(`UPDATE tags SET name = $3 WHERE chat_id = $1 AND name = $2`, chatID, oldName, newName)
	if err != nil {
		if isUniqueViolation(err) {
			return repository.Tag{}, repository.ErrTagExists
		}
		return repository.Tag{}, fmt.Errorf("update tag: %w", err)
	}
	affected, affErr := res.RowsAffected()
	if affErr != nil {
		return repository.Tag{}, fmt.Errorf("update tag rows affected: %w", affErr)
	}
	if affected == 0 {
		return repository.Tag{}, repository.ErrTagNotFound
	}
	var id int64
	if err = r.db.QueryRow(`SELECT id FROM tags WHERE chat_id = $1 AND name = $2`, chatID, newName).Scan(&id); err != nil {
		return repository.Tag{}, fmt.Errorf("load updated tag: %w", err)
	}
	return repository.Tag{ID: id, ChatID: chatID, Name: newName}, nil
}

func (r *Repository) DeleteTag(chatID int64, rawTag string) error {
	tag := normalizeTag(rawTag)
	if tag == "" {
		return repository.ErrTagNotFound
	}
	chatExists, err := r.chatExists(chatID)
	if err != nil {
		return fmt.Errorf("check chat exists before delete tag: %w", err)
	}
	if !chatExists {
		return repository.ErrChatNotFound
	}
	res, err := r.db.Exec(`DELETE FROM tags WHERE chat_id = $1 AND name = $2`, chatID, tag)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	affected, affErr := res.RowsAffected()
	if affErr != nil {
		return fmt.Errorf("delete tag rows affected: %w", affErr)
	}
	if affected == 0 {
		return repository.ErrTagNotFound
	}
	return nil
}
