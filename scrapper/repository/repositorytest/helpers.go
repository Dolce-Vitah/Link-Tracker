package repositorytest

import (
	"net/url"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (s *Store) getLinkTagsLocked(chatID int64, link string) []string {
	byLink, ok := s.tagsByChatLink[chatID]
	if !ok {
		return nil
	}
	linkTags, ok := byLink[link]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(linkTags))
	for name := range linkTags {
		out = append(out, name)
	}
	return deduplicate(out)
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

func normalizeTag(value string) string {
	return strings.TrimSpace(value)
}

func trimAndValidateLink(link string) (string, error) {
	trimmed := strings.TrimSpace(link)
	if !isValidLink(trimmed) {
		return "", repository.ErrInvalidLink
	}
	return trimmed, nil
}

func isValidLink(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
