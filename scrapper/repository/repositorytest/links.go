package repositorytest

import (
	"sort"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

func (s *Store) AddLink(chatID int64, request api.AddLinkRequest) (api.LinkResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.chats[chatID]; !ok {
		return api.LinkResponse{}, repository.ErrChatNotFound
	}
	link, linkErr := trimAndValidateLink(request.Link)
	if linkErr != nil {
		return api.LinkResponse{}, linkErr
	}

	if _, ok := s.linksByChat[chatID]; !ok {
		s.linksByChat[chatID] = make(map[string]struct{})
	}
	if _, exists := s.linksByChat[chatID][link]; exists {
		return api.LinkResponse{}, repository.ErrLinkExists
	}

	tracked, exists := s.linksByURL[link]
	if !exists {
		tracked = &memoryLink{
			tracked: repository.TrackedLink{
				Response: api.LinkResponse{
					ID:      s.nextID,
					URL:     link,
					Tags:    nil,
					Filters: deduplicate(request.Filters),
				},
				ChatIDs: make(map[int64]struct{}),
			},
		}
		s.linksByURL[link] = tracked
		s.nextID++
	}
	tracked.tracked.ChatIDs[chatID] = struct{}{}
	s.linksByChat[chatID][link] = struct{}{}

	if _, ok := s.tagsByChatLink[chatID]; !ok {
		s.tagsByChatLink[chatID] = make(map[string]map[string]struct{})
	}
	if _, ok := s.tagsByChatLink[chatID][link]; !ok {
		s.tagsByChatLink[chatID][link] = make(map[string]struct{})
	}
	for _, rawTag := range deduplicate(request.Tags) {
		normalized := normalizeTag(rawTag)
		if normalized == "" {
			continue
		}
		if _, ok := s.tagsByChat[chatID]; !ok {
			s.tagsByChat[chatID] = make(map[string]repository.Tag)
		}
		if _, existsTag := s.tagsByChat[chatID][normalized]; !existsTag {
			s.tagsByChat[chatID][normalized] = repository.Tag{
				ID:     s.nextTagID,
				ChatID: chatID,
				Name:   normalized,
			}
			s.nextTagID++
		}
		s.tagsByChatLink[chatID][link][normalized] = struct{}{}
	}

	resp := tracked.tracked.Response
	resp.Tags = s.getLinkTagsLocked(chatID, link)
	return resp, nil
}

func (s *Store) RemoveLink(chatID int64, request api.RemoveLinkRequest) (api.LinkResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.chats[chatID]; !ok {
		return api.LinkResponse{}, repository.ErrChatNotFound
	}
	link := strings.TrimSpace(request.Link)
	if link == "" {
		return api.LinkResponse{}, repository.ErrInvalidLink
	}

	chatLinks, ok := s.linksByChat[chatID]
	if !ok {
		return api.LinkResponse{}, repository.ErrLinkNotFound
	}
	if _, exists := chatLinks[link]; !exists {
		return api.LinkResponse{}, repository.ErrLinkNotFound
	}

	resp := s.linksByURL[link].tracked.Response
	resp.Tags = s.getLinkTagsLocked(chatID, link)

	delete(chatLinks, link)
	if byLink, okByLink := s.tagsByChatLink[chatID]; okByLink {
		delete(byLink, link)
	}

	tracked := s.linksByURL[link]
	delete(tracked.tracked.ChatIDs, chatID)
	if len(tracked.tracked.ChatIDs) == 0 {
		delete(s.linksByURL, link)
	}

	return resp, nil
}

func (s *Store) ListLinks(chatID int64) (api.ListLinksResponse, error) {
	const pageSize = 100
	offset := 0
	out := api.ListLinksResponse{Links: make([]api.LinkResponse, 0)}
	for {
		chunk, err := s.ListLinksPage(chatID, pageSize, offset)
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

func (s *Store) ListLinksPage(chatID int64, limit int, offset int) ([]api.LinkResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.chats[chatID]; !ok {
		return nil, repository.ErrChatNotFound
	}
	if limit <= 0 {
		limit = len(s.linksByChat[chatID])
	}

	all := make([]api.LinkResponse, 0, len(s.linksByChat[chatID]))
	for linkURL := range s.linksByChat[chatID] {
		resp := s.linksByURL[linkURL].tracked.Response
		resp.Tags = s.getLinkTagsLocked(chatID, linkURL)
		all = append(all, resp)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})
	if offset >= len(all) {
		return []api.LinkResponse{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}
