package ormrepo

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"gorm.io/gorm"
)

func selectTagNamesORM(db *gorm.DB, chatLinkID int64) ([]string, error) {
	type nameRow struct {
		Name string `gorm:"column:name"`
	}
	rows := make([]nameRow, 0)
	if err := db.Table("chat_link_tags clt").
		Select("t.name").
		Joins("JOIN tags t ON t.id = clt.tag_id").
		Where("clt.chat_link_id = ?", chatLinkID).
		Order("t.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("select tag names orm: %w", err)
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Name)
	}
	return out, nil
}

func trimAndValidateLink(link string) (string, error) {
	trimmed := strings.TrimSpace(link)
	if !isValidLink(trimmed) {
		return "", repository.ErrInvalidLink
	}
	return trimmed, nil
}

func normalizeTag(value string) string {
	return strings.TrimSpace(value)
}

func deduplicate(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		already := false
		for _, existing := range out {
			if existing == trimmed {
				already = true
				break
			}
		}
		if !already {
			out = append(out, trimmed)
		}
	}
	return out
}

func isValidLink(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}

func (r *Repository) chatExists(chatID int64) (bool, error) {
	var count int64
	if err := r.db.Model(&ChatModel{}).Where("chat_id = ?", chatID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
