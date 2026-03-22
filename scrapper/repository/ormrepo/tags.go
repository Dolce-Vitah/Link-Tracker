package ormrepo

import (
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (r *Repository) CreateTag(chatID int64, rawTag string) (repository.Tag, error) {
	tagName := normalizeTag(rawTag)
	if tagName == "" {
		return repository.Tag{}, repository.ErrInvalidTag
	}
	chatExists, err := r.chatExists(chatID)
	if err != nil {
		return repository.Tag{}, fmt.Errorf("check chat exists before create tag orm: %w", err)
	}
	if !chatExists {
		return repository.Tag{}, repository.ErrChatNotFound
	}
	model := TagModel{ChatID: chatID, Name: tagName}
	if err = r.db.Create(&model).Error; err != nil {
		if isUniqueViolation(err) {
			return repository.Tag{}, repository.ErrTagExists
		}
		if isForeignKeyViolation(err) {
			return repository.Tag{}, repository.ErrChatNotFound
		}
		return repository.Tag{}, fmt.Errorf("create tag orm: %w", err)
	}
	return repository.Tag{ID: model.ID, ChatID: chatID, Name: tagName}, nil
}

func (r *Repository) ListTags(chatID int64, limit int, offset int) ([]repository.Tag, error) {
	chatExists, err := r.chatExists(chatID)
	if err != nil {
		return nil, fmt.Errorf("check chat exists before list tags orm: %w", err)
	}
	if !chatExists {
		return nil, repository.ErrChatNotFound
	}
	if limit <= 0 {
		limit = 100
	}
	var models []TagModel
	if err = r.db.Where("chat_id = ?", chatID).Order("id ASC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list tags orm: %w", err)
	}
	out := make([]repository.Tag, 0, len(models))
	for _, model := range models {
		out = append(out, repository.Tag{ID: model.ID, ChatID: model.ChatID, Name: model.Name})
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
		return repository.Tag{}, fmt.Errorf("check chat exists before update tag orm: %w", err)
	}
	if !chatExists {
		return repository.Tag{}, repository.ErrChatNotFound
	}
	res := r.db.Model(&TagModel{}).Where("chat_id = ? AND name = ?", chatID, oldName).Update("name", newName)
	if res.Error != nil {
		if isUniqueViolation(res.Error) {
			return repository.Tag{}, repository.ErrTagExists
		}
		return repository.Tag{}, fmt.Errorf("update tag orm: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.Tag{}, repository.ErrTagNotFound
	}
	var model TagModel
	if queryErr := r.db.Where("chat_id = ? AND name = ?", chatID, newName).Take(&model).Error; queryErr != nil {
		return repository.Tag{}, fmt.Errorf("load updated tag orm: %w", queryErr)
	}
	return repository.Tag{ID: model.ID, ChatID: model.ChatID, Name: model.Name}, nil
}

func (r *Repository) DeleteTag(chatID int64, rawTag string) error {
	tagName := normalizeTag(rawTag)
	if tagName == "" {
		return repository.ErrTagNotFound
	}
	chatExists, err := r.chatExists(chatID)
	if err != nil {
		return fmt.Errorf("check chat exists before delete tag orm: %w", err)
	}
	if !chatExists {
		return repository.ErrChatNotFound
	}
	res := r.db.Where("chat_id = ? AND name = ?", chatID, tagName).Delete(&TagModel{})
	if res.Error != nil {
		return fmt.Errorf("delete tag orm: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrTagNotFound
	}
	return nil
}
