//nolint:noctx // Repository contract does not accept context; data-layer calls use internal defaults.
package sqlrepo

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (r *Repository) AddLink(chatID int64, request api.AddLinkRequest) (api.LinkResponse, error) {
	link, linkErr := trimAndValidateLink(request.Link)
	if linkErr != nil {
		return api.LinkResponse{}, linkErr
	}
	filters := deduplicate(request.Filters)
	tags := deduplicate(request.Tags)

	tx, err := r.db.Begin()
	if err != nil {
		return api.LinkResponse{}, fmt.Errorf("begin tx add link: %w", err)
	}
	defer rollback(tx)

	var chatExists bool
	if existsErr := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM chats WHERE chat_id = $1)`, chatID).Scan(&chatExists); existsErr != nil {
		return api.LinkResponse{}, fmt.Errorf("check chat exists: %w", existsErr)
	}
	if !chatExists {
		return api.LinkResponse{}, repository.ErrChatNotFound
	}

	var linkID int64
	linkErr = tx.QueryRow(`
		INSERT INTO links(url) VALUES ($1)
		ON CONFLICT (url) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, link).Scan(&linkID)
	if linkErr != nil {
		return api.LinkResponse{}, fmt.Errorf("upsert link: %w", linkErr)
	}

	filtersJSON, marshalErr := json.Marshal(filters)
	if marshalErr != nil {
		return api.LinkResponse{}, fmt.Errorf("marshal filters: %w", marshalErr)
	}

	var chatLinkID int64
	chatLinkErr := tx.QueryRow(`
		INSERT INTO chat_links(chat_id, link_id, filters)
		VALUES ($1, $2, $3::jsonb)
		RETURNING id
	`, chatID, linkID, string(filtersJSON)).Scan(&chatLinkID)
	if chatLinkErr != nil {
		if isUniqueViolation(chatLinkErr) {
			return api.LinkResponse{}, repository.ErrLinkExists
		}
		return api.LinkResponse{}, fmt.Errorf("create chat link: %w", chatLinkErr)
	}

	if _, delErr := tx.Exec(`DELETE FROM chat_link_tags WHERE chat_link_id = $1`, chatLinkID); delErr != nil {
		return api.LinkResponse{}, fmt.Errorf("clear link tags for chat link: %w", delErr)
	}

	for _, tag := range tags {
		tagID, upsertErr := upsertTagTx(tx, chatID, tag)
		if upsertErr != nil {
			return api.LinkResponse{}, upsertErr
		}
		if _, insertErr := tx.Exec(`INSERT INTO chat_link_tags(chat_link_id, tag_id, chat_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, chatLinkID, tagID, chatID); insertErr != nil {
			return api.LinkResponse{}, fmt.Errorf("bind link tag: %w", insertErr)
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return api.LinkResponse{}, fmt.Errorf("commit add link tx: %w", commitErr)
	}

	return api.LinkResponse{
		ID:      linkID,
		URL:     link,
		Tags:    tags,
		Filters: filters,
	}, nil
}

func (r *Repository) RemoveLink(chatID int64, request api.RemoveLinkRequest) (api.LinkResponse, error) {
	link := strings.TrimSpace(request.Link)
	if link == "" {
		return api.LinkResponse{}, repository.ErrInvalidLink
	}

	tx, err := r.db.Begin()
	if err != nil {
		return api.LinkResponse{}, fmt.Errorf("begin tx remove link: %w", err)
	}
	defer rollback(tx)

	var chatExists bool
	if existsErr := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM chats WHERE chat_id = $1)`, chatID).Scan(&chatExists); existsErr != nil {
		return api.LinkResponse{}, fmt.Errorf("check chat exists: %w", existsErr)
	}
	if !chatExists {
		return api.LinkResponse{}, repository.ErrChatNotFound
	}

	type row struct {
		linkID     int64
		chatLinkID int64
		filtersRaw []byte
	}
	var data row
	queryErr := tx.QueryRow(`
		SELECT l.id, cl.id, cl.filters
		FROM links l
		JOIN chat_links cl ON cl.link_id = l.id
		WHERE cl.chat_id = $1 AND l.url = $2
	`, chatID, link).Scan(&data.linkID, &data.chatLinkID, &data.filtersRaw)
	if queryErr != nil {
		if errors.Is(queryErr, sql.ErrNoRows) {
			return api.LinkResponse{}, repository.ErrLinkNotFound
		}
		return api.LinkResponse{}, fmt.Errorf("get link relation for delete: %w", queryErr)
	}

	tags, tagsErr := selectTagNamesTx(tx, data.chatLinkID)
	if tagsErr != nil {
		return api.LinkResponse{}, tagsErr
	}

	filters := make([]string, 0)
	if len(data.filtersRaw) > 0 {
		if unmarshalErr := json.Unmarshal(data.filtersRaw, &filters); unmarshalErr != nil {
			return api.LinkResponse{}, fmt.Errorf("unmarshal filters: %w", unmarshalErr)
		}
	}

	if _, delErr := tx.Exec(`DELETE FROM chat_links WHERE id = $1`, data.chatLinkID); delErr != nil {
		return api.LinkResponse{}, fmt.Errorf("delete chat link: %w", delErr)
	}

	if _, cleanupErr := tx.Exec(`DELETE FROM links WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM chat_links WHERE link_id = $1)`, data.linkID); cleanupErr != nil {
		return api.LinkResponse{}, fmt.Errorf("cleanup orphan link: %w", cleanupErr)
	}
	if _, cleanupTagsErr := tx.Exec(`
		DELETE FROM tags t
		WHERE t.chat_id = $1
		  AND NOT EXISTS (
			SELECT 1
			FROM chat_link_tags clt
			JOIN chat_links cl ON cl.id = clt.chat_link_id
			WHERE cl.chat_id = t.chat_id AND clt.tag_id = t.id
		  )
	`, chatID); cleanupTagsErr != nil {
		return api.LinkResponse{}, fmt.Errorf("cleanup orphan tags: %w", cleanupTagsErr)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return api.LinkResponse{}, fmt.Errorf("commit remove link tx: %w", commitErr)
	}

	return api.LinkResponse{
		ID:      data.linkID,
		URL:     link,
		Tags:    tags,
		Filters: deduplicate(filters),
	}, nil
}

func (r *Repository) ListLinks(chatID int64) (api.ListLinksResponse, error) {
	const pageSize = 100
	offset := 0
	out := api.ListLinksResponse{Links: make([]api.LinkResponse, 0)}
	for {
		chunk, err := r.ListLinksPage(chatID, pageSize, offset)
		if err != nil {
			return api.ListLinksResponse{}, err
		}
		if len(chunk) == 0 {
			break
		}
		out.Links = append(out.Links, chunk...)
		offset += len(chunk)
	}
	out.Size = int32(len(out.Links))
	return out, nil
}

func (r *Repository) ListLinksPage(chatID int64, limit int, offset int) ([]api.LinkResponse, error) {
	var chatExists bool
	if err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM chats WHERE chat_id = $1)`, chatID).Scan(&chatExists); err != nil {
		return nil, fmt.Errorf("check chat exists: %w", err)
	}
	if !chatExists {
		return nil, repository.ErrChatNotFound
	}
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.Query(`
		SELECT l.id, l.url, cl.id, cl.filters
		FROM chat_links cl
		JOIN links l ON l.id = cl.link_id
		WHERE cl.chat_id = $1
		ORDER BY l.id ASC
		LIMIT $2 OFFSET $3
	`, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]api.LinkResponse, 0, limit)
	for rows.Next() {
		var (
			linkID     int64
			urlValue   string
			chatLinkID int64
			filtersRaw []byte
		)
		if scanErr := rows.Scan(&linkID, &urlValue, &chatLinkID, &filtersRaw); scanErr != nil {
			return nil, fmt.Errorf("scan listed link: %w", scanErr)
		}
		filters := make([]string, 0)
		if len(filtersRaw) > 0 {
			if decodeErr := json.Unmarshal(filtersRaw, &filters); decodeErr != nil {
				return nil, fmt.Errorf("decode listed filters: %w", decodeErr)
			}
		}
		tags, tagErr := r.selectTagNames(chatLinkID)
		if tagErr != nil {
			return nil, tagErr
		}
		out = append(out, api.LinkResponse{
			ID:      linkID,
			URL:     urlValue,
			Tags:    tags,
			Filters: deduplicate(filters),
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate listed links: %w", rowsErr)
	}
	return out, nil
}
