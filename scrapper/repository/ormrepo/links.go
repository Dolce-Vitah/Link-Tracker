package ormrepo

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"gorm.io/gorm"
)

//nolint:gocognit // Transaction flow is explicit to preserve repository invariants.
func (r *Repository) AddLink(chatID int64, request trackerapi.AddLinkRequest) (trackerapi.LinkResponse, error) {
	link, linkErr := trimAndValidateLink(request.Link)
	if linkErr != nil {
		return trackerapi.LinkResponse{}, linkErr
	}
	tags := deduplicate(request.Tags)
	filters := deduplicate(request.Filters)
	filtersRaw, marshalErr := json.Marshal(filters)
	if marshalErr != nil {
		return trackerapi.LinkResponse{}, fmt.Errorf("marshal filters orm: %w", marshalErr)
	}

	var output trackerapi.LinkResponse
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var chat ChatModel
		if chatErr := tx.Where("chat_id = ?", chatID).Take(&chat).Error; chatErr != nil {
			if errors.Is(chatErr, gorm.ErrRecordNotFound) {
				return repository.ErrChatNotFound
			}
			return fmt.Errorf("load chat orm: %w", chatErr)
		}

		var linkModel LinkModel
		loadErr := tx.Where("url = ?", link).Take(&linkModel).Error
		if loadErr != nil {
			if errors.Is(loadErr, gorm.ErrRecordNotFound) {
				linkModel = LinkModel{URL: link}
				if createErr := tx.Create(&linkModel).Error; createErr != nil {
					return fmt.Errorf("create link orm: %w", createErr)
				}
			} else {
				return fmt.Errorf("load link orm: %w", loadErr)
			}
		}

		chatLink := ChatLinkModel{ChatID: chatID, LinkID: linkModel.ID, Filters: filtersRaw}
		if createChatLinkErr := tx.Create(&chatLink).Error; createChatLinkErr != nil {
			if isUniqueViolation(createChatLinkErr) {
				return repository.ErrLinkExists
			}
			return fmt.Errorf("create chat_link orm: %w", createChatLinkErr)
		}

		if clearErr := tx.Where("chat_link_id = ?", chatLink.ID).Delete(&ChatLinkTagModel{}).Error; clearErr != nil {
			return fmt.Errorf("clear chat_link_tags orm: %w", clearErr)
		}
		for _, rawTag := range tags {
			tagName := normalizeTag(rawTag)
			if tagName == "" {
				continue
			}
			var tag TagModel
			tagErr := tx.Where("chat_id = ? AND name = ?", chatID, tagName).Take(&tag).Error
			if tagErr != nil {
				if errors.Is(tagErr, gorm.ErrRecordNotFound) {
					tag = TagModel{ChatID: chatID, Name: tagName}
					if createTagErr := tx.Create(&tag).Error; createTagErr != nil {
						return fmt.Errorf("create tag orm: %w", createTagErr)
					}
				} else {
					return fmt.Errorf("load tag orm: %w", tagErr)
				}
			}
			if bindErr := tx.Create(&ChatLinkTagModel{ChatLinkID: chatLink.ID, TagID: tag.ID, ChatID: chatID}).Error; bindErr != nil && !isUniqueViolation(bindErr) {
				return fmt.Errorf("bind chat_link_tag orm: %w", bindErr)
			}
		}

		output = trackerapi.LinkResponse{
			ID:      linkModel.ID,
			URL:     linkModel.URL,
			Tags:    tags,
			Filters: filters,
		}
		return nil
	})
	if err != nil {
		return trackerapi.LinkResponse{}, fmt.Errorf("add link orm transaction: %w", err)
	}
	return output, nil
}

func (r *Repository) RemoveLink(chatID int64, request trackerapi.RemoveLinkRequest) (trackerapi.LinkResponse, error) {
	linkURL := strings.TrimSpace(request.Link)
	if linkURL == "" {
		return trackerapi.LinkResponse{}, repository.ErrInvalidLink
	}
	var output trackerapi.LinkResponse
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var chat ChatModel
		if chatErr := tx.Where("chat_id = ?", chatID).Take(&chat).Error; chatErr != nil {
			if errors.Is(chatErr, gorm.ErrRecordNotFound) {
				return repository.ErrChatNotFound
			}
			return fmt.Errorf("load chat before remove orm: %w", chatErr)
		}

		var link LinkModel
		if linkErr := tx.Where("url = ?", linkURL).Take(&link).Error; linkErr != nil {
			if errors.Is(linkErr, gorm.ErrRecordNotFound) {
				return repository.ErrLinkNotFound
			}
			return fmt.Errorf("load link before remove orm: %w", linkErr)
		}

		var chatLink ChatLinkModel
		if chatLinkErr := tx.Where("chat_id = ? AND link_id = ?", chatID, link.ID).Take(&chatLink).Error; chatLinkErr != nil {
			if errors.Is(chatLinkErr, gorm.ErrRecordNotFound) {
				return repository.ErrLinkNotFound
			}
			return fmt.Errorf("load chat_link before remove orm: %w", chatLinkErr)
		}

		tags, tagsErr := selectTagNamesORM(tx, chatLink.ID)
		if tagsErr != nil {
			return tagsErr
		}

		filters := make([]string, 0)
		if len(chatLink.Filters) > 0 {
			if decodeErr := json.Unmarshal(chatLink.Filters, &filters); decodeErr != nil {
				return fmt.Errorf("decode filters before remove orm: %w", decodeErr)
			}
		}

		if delErr := tx.Delete(&chatLink).Error; delErr != nil {
			return fmt.Errorf("delete chat_link orm: %w", delErr)
		}
		if cleanupErr := tx.Exec(`DELETE FROM links WHERE id = ? AND NOT EXISTS (SELECT 1 FROM chat_links WHERE link_id = ?)`, link.ID, link.ID).Error; cleanupErr != nil {
			return fmt.Errorf("cleanup link after remove orm: %w", cleanupErr)
		}
		if cleanupTagsErr := tx.Exec(`
			DELETE FROM tags t
			WHERE t.chat_id = ?
			  AND NOT EXISTS (
			    SELECT 1
			    FROM chat_link_tags clt
			    JOIN chat_links cl ON cl.id = clt.chat_link_id
			    WHERE cl.chat_id = t.chat_id AND clt.tag_id = t.id
			  )
		`, chatID).Error; cleanupTagsErr != nil {
			return fmt.Errorf("cleanup tags after remove orm: %w", cleanupTagsErr)
		}

		output = trackerapi.LinkResponse{
			ID:      link.ID,
			URL:     linkURL,
			Tags:    tags,
			Filters: deduplicate(filters),
		}
		return nil
	})
	if err != nil {
		return trackerapi.LinkResponse{}, fmt.Errorf("remove link orm transaction: %w", err)
	}
	return output, nil
}

func (r *Repository) ListLinks(chatID int64) (trackerapi.ListLinksResponse, error) {
	const pageSize = 100
	offset := 0
	out := trackerapi.ListLinksResponse{Links: make([]trackerapi.LinkResponse, 0)}
	for {
		chunk, err := r.ListLinksPage(chatID, pageSize, offset)
		if err != nil {
			return trackerapi.ListLinksResponse{}, err
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

func (r *Repository) ListLinksPage(chatID int64, limit int, offset int) ([]trackerapi.LinkResponse, error) {
	var chatExists int64
	if err := r.db.Model(&ChatModel{}).Where("chat_id = ?", chatID).Count(&chatExists).Error; err != nil {
		return nil, fmt.Errorf("check chat exists orm: %w", err)
	}
	if chatExists == 0 {
		return nil, repository.ErrChatNotFound
	}
	if limit <= 0 {
		limit = 100
	}

	var chatLinks []ChatLinkModel
	if err := r.db.Where("chat_id = ?", chatID).Order("id ASC").Limit(limit).Offset(offset).Find(&chatLinks).Error; err != nil {
		return nil, fmt.Errorf("list chat_links orm: %w", err)
	}

	out := make([]trackerapi.LinkResponse, 0, len(chatLinks))
	for _, chatLink := range chatLinks {
		var link LinkModel
		if err := r.db.Where("id = ?", chatLink.LinkID).Take(&link).Error; err != nil {
			return nil, fmt.Errorf("load link in list orm: %w", err)
		}
		filters := make([]string, 0)
		if len(chatLink.Filters) > 0 {
			if decodeErr := json.Unmarshal(chatLink.Filters, &filters); decodeErr != nil {
				return nil, fmt.Errorf("decode filters in list orm: %w", decodeErr)
			}
		}
		tags, tagsErr := selectTagNamesORM(r.db, chatLink.ID)
		if tagsErr != nil {
			return nil, tagsErr
		}
		out = append(out, trackerapi.LinkResponse{
			ID:      link.ID,
			URL:     link.URL,
			Tags:    tags,
			Filters: deduplicate(filters),
		})
	}
	return out, nil
}
