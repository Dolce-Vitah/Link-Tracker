package scrapper

import (
	"errors"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

var (
	ErrChatExists   = errors.New("chat already exists")
	ErrChatNotFound = errors.New("chat not found")
	ErrLinkExists   = errors.New("link already tracked")
	ErrLinkNotFound = errors.New("link not tracked")
	ErrInvalidLink  = errors.New("invalid link")
)

type TrackedLink struct {
	Response    api.LinkResponse
	ChatIDs     map[int64]struct{}
	LastUpdated time.Time
}

type Service struct {
	mu          sync.RWMutex
	nextID      int64
	chats       map[int64]struct{}
	linksByURL  map[string]*TrackedLink
	linksByChat map[int64]map[string]struct{}
}

func NewService() *Service {
	return &Service{
		nextID:      1,
		chats:       make(map[int64]struct{}),
		linksByURL:  make(map[string]*TrackedLink),
		linksByChat: make(map[int64]map[string]struct{}),
	}
}

func (s *Service) RegisterChat(chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; ok {
		return ErrChatExists
	}
	s.chats[chatID] = struct{}{}
	return nil
}

func (s *Service) DeleteChat(chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; !ok {
		return ErrChatNotFound
	}
	delete(s.chats, chatID)
	if linkURLs, ok := s.linksByChat[chatID]; ok {
		for linkURL := range linkURLs {
			tracked := s.linksByURL[linkURL]
			delete(tracked.ChatIDs, chatID)
			if len(tracked.ChatIDs) == 0 {
				delete(s.linksByURL, linkURL)
			}
		}
		delete(s.linksByChat, chatID)
	}
	return nil
}

func (s *Service) AddLink(chatID int64, request api.AddLinkRequest) (api.LinkResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.chats[chatID]; !ok {
		return api.LinkResponse{}, ErrChatNotFound
	}
	link := strings.TrimSpace(request.Link)
	if !isValidLink(link) {
		return api.LinkResponse{}, ErrInvalidLink
	}

	if _, ok := s.linksByChat[chatID]; !ok {
		s.linksByChat[chatID] = make(map[string]struct{})
	}
	if _, exists := s.linksByChat[chatID][link]; exists {
		return api.LinkResponse{}, ErrLinkExists
	}

	tracked, exists := s.linksByURL[link]
	if !exists {
		tracked = &TrackedLink{
			Response: api.LinkResponse{
				ID:      s.nextID,
				URL:     link,
				Tags:    deduplicate(request.Tags),
				Filters: deduplicate(request.Filters),
			},
			ChatIDs: make(map[int64]struct{}),
		}
		s.linksByURL[link] = tracked
		s.nextID++
	}
	tracked.ChatIDs[chatID] = struct{}{}
	s.linksByChat[chatID][link] = struct{}{}

	return tracked.Response, nil
}

func (s *Service) RemoveLink(chatID int64, request api.RemoveLinkRequest) (api.LinkResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.chats[chatID]; !ok {
		return api.LinkResponse{}, ErrChatNotFound
	}
	link := strings.TrimSpace(request.Link)
	if link == "" {
		return api.LinkResponse{}, ErrInvalidLink
	}

	chatLinks, ok := s.linksByChat[chatID]
	if !ok {
		return api.LinkResponse{}, ErrLinkNotFound
	}
	if _, exists := chatLinks[link]; !exists {
		return api.LinkResponse{}, ErrLinkNotFound
	}

	delete(chatLinks, link)
	tracked := s.linksByURL[link]
	delete(tracked.ChatIDs, chatID)
	if len(tracked.ChatIDs) == 0 {
		delete(s.linksByURL, link)
	}

	return tracked.Response, nil
}

func (s *Service) ListLinks(chatID int64) (api.ListLinksResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.chats[chatID]; !ok {
		return api.ListLinksResponse{}, ErrChatNotFound
	}

	out := api.ListLinksResponse{
		Links: make([]api.LinkResponse, 0),
	}
	for linkURL := range s.linksByChat[chatID] {
		out.Links = append(out.Links, s.linksByURL[linkURL].Response)
	}
	out.Size = int32(len(out.Links))
	return out, nil
}

func (s *Service) SnapshotLinks() []TrackedLink {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TrackedLink, 0, len(s.linksByURL))
	for _, link := range s.linksByURL {
		clone := TrackedLink{
			Response:    link.Response,
			ChatIDs:     make(map[int64]struct{}, len(link.ChatIDs)),
			LastUpdated: link.LastUpdated,
		}
		for chatID := range link.ChatIDs {
			clone.ChatIDs[chatID] = struct{}{}
		}
		out = append(out, clone)
	}
	return out
}

func (s *Service) UpdateLastUpdated(linkURL string, value time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tracked, ok := s.linksByURL[linkURL]
	if !ok {
		return
	}
	tracked.LastUpdated = value
}

func deduplicate(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || slices.Contains(out, trimmed) {
			continue
		}
		out = append(out, trimmed)
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
