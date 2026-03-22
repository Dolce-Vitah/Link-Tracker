package ormrepo

import (
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (r *Repository) RegisterChat(chatID int64) error {
	err := r.db.Create(&ChatModel{ChatID: chatID}).Error
	if err == nil {
		return nil
	}
	if isUniqueViolation(err) {
		return repository.ErrChatExists
	}
	return fmt.Errorf("register chat orm: %w", err)
}

func (r *Repository) DeleteChat(chatID int64) error {
	res := r.db.Where("chat_id = ?", chatID).Delete(&ChatModel{})
	if res.Error != nil {
		return fmt.Errorf("delete chat orm: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrChatNotFound
	}
	cleanupErr := r.db.Exec(`
		DELETE FROM links
		WHERE NOT EXISTS (SELECT 1 FROM chat_links cl WHERE cl.link_id = links.id)
	`).Error
	if cleanupErr != nil {
		return fmt.Errorf("cleanup links after chat delete orm: %w", cleanupErr)
	}
	return nil
}
