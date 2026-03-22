package sqlrepo

import (
	"database/sql"
	"errors"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
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

func isValidLink(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
