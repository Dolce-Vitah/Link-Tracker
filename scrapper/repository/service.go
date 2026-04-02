package repository

import (
	"errors"
	"net/url"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
)

var (
	ErrChatExists   = errors.New("chat already exists")
	ErrChatNotFound = errors.New("chat not found")
	ErrLinkExists   = errors.New("link already tracked")
	ErrLinkNotFound = errors.New("link not tracked")
	ErrInvalidLink  = errors.New("invalid link")
	ErrInvalidTag   = errors.New("invalid tag")
	ErrTagExists    = errors.New("tag already exists")
	ErrTagNotFound  = errors.New("tag not found")
)

type TrackedLink struct {
	Response      trackerapi.LinkResponse
	ChatIDs       map[int64]struct{}
	LastUpdated   time.Time
	LastCheckedAt time.Time
}

type Tag struct {
	ID     int64
	ChatID int64
	Name   string
}

type ChatService interface {
	RegisterChat(chatID int64) error
	DeleteChat(chatID int64) error
}

type LinkService interface {
	AddLink(chatID int64, request trackerapi.AddLinkRequest) (trackerapi.LinkResponse, error)
	RemoveLink(chatID int64, request trackerapi.RemoveLinkRequest) (trackerapi.LinkResponse, error)
	ListLinks(chatID int64) (trackerapi.ListLinksResponse, error)
	ListLinksPage(chatID int64, limit int, offset int) ([]trackerapi.LinkResponse, error)
}

type TrackedLinkService interface {
	ListTrackedLinksPage(limit int, checkedBefore time.Time, afterID int64) ([]TrackedLink, error)
	UpdateLastUpdated(linkURL string, value time.Time) error
	TouchLastChecked(linkURL string, value time.Time) error
}

type TagService interface {
	CreateTag(chatID int64, tag string) (Tag, error)
	ListTags(chatID int64, limit int, offset int) ([]Tag, error)
	UpdateTag(chatID int64, oldTag string, newTag string) (Tag, error)
	DeleteTag(chatID int64, tag string) error
}

type TrackingService interface {
	ChatService
	LinkService
}

type Service interface {
	TrackingService
	TrackedLinkService
	TagService
}

type MemoryService struct {
	mu             sync.RWMutex
	nextID         int64
	nextTagID      int64
	chats          map[int64]struct{}
	linksByURL     map[string]*TrackedLink
	linksByChat    map[int64]map[string]struct{}
	tagsByChat     map[int64]map[string]Tag
	tagsByChatLink map[int64]map[string]map[string]struct{}
}

func NewService() *MemoryService {
	return &MemoryService{
		nextID:         1,
		nextTagID:      1,
		chats:          make(map[int64]struct{}),
		linksByURL:     make(map[string]*TrackedLink),
		linksByChat:    make(map[int64]map[string]struct{}),
		tagsByChat:     make(map[int64]map[string]Tag),
		tagsByChatLink: make(map[int64]map[string]map[string]struct{}),
	}
}

func (s *MemoryService) RegisterChat(chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.chats[chatID]; ok {
		return ErrChatExists
	}
	s.chats[chatID] = struct{}{}

	return nil
}

func (s *MemoryService) DeleteChat(chatID int64) error {
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
	delete(s.tagsByChat, chatID)
	delete(s.tagsByChatLink, chatID)

	return nil
}

func (s *MemoryService) AddLink(chatID int64, request trackerapi.AddLinkRequest) (trackerapi.LinkResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.chats[chatID]; !ok {
		return trackerapi.LinkResponse{}, ErrChatNotFound
	}
	link, linkErr := trimAndValidateLink(request.Link)
	if linkErr != nil {
		return trackerapi.LinkResponse{}, linkErr
	}

	if _, ok := s.linksByChat[chatID]; !ok {
		s.linksByChat[chatID] = make(map[string]struct{})
	}
	if _, exists := s.linksByChat[chatID][link]; exists {
		return trackerapi.LinkResponse{}, ErrLinkExists
	}

	tracked, exists := s.linksByURL[link]
	if !exists {
		tracked = &TrackedLink{
			Response: trackerapi.LinkResponse{
				ID:      s.nextID,
				URL:     link,
				Tags:    nil,
				Filters: deduplicate(request.Filters),
			},
			ChatIDs: make(map[int64]struct{}),
		}
		s.linksByURL[link] = tracked
		s.nextID++
	}
	tracked.ChatIDs[chatID] = struct{}{}
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
			s.tagsByChat[chatID] = make(map[string]Tag)
		}
		if _, existsTag := s.tagsByChat[chatID][normalized]; !existsTag {
			s.tagsByChat[chatID][normalized] = Tag{
				ID:     s.nextTagID,
				ChatID: chatID,
				Name:   normalized,
			}
			s.nextTagID++
		}
		s.tagsByChatLink[chatID][link][normalized] = struct{}{}
	}

	resp := tracked.Response
	resp.Tags = s.getLinkTagsLocked(chatID, link)
	return resp, nil
}

func (s *MemoryService) RemoveLink(chatID int64, request trackerapi.RemoveLinkRequest) (trackerapi.LinkResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.chats[chatID]; !ok {
		return trackerapi.LinkResponse{}, ErrChatNotFound
	}
	link := strings.TrimSpace(request.Link)
	if link == "" {
		return trackerapi.LinkResponse{}, ErrInvalidLink
	}

	chatLinks, ok := s.linksByChat[chatID]
	if !ok {
		return trackerapi.LinkResponse{}, ErrLinkNotFound
	}
	if _, exists := chatLinks[link]; !exists {
		return trackerapi.LinkResponse{}, ErrLinkNotFound
	}

	resp := s.linksByURL[link].Response
	resp.Tags = s.getLinkTagsLocked(chatID, link)

	delete(chatLinks, link)
	if byLink, okByLink := s.tagsByChatLink[chatID]; okByLink {
		delete(byLink, link)
	}

	tracked := s.linksByURL[link]
	delete(tracked.ChatIDs, chatID)
	if len(tracked.ChatIDs) == 0 {
		delete(s.linksByURL, link)
	}

	return resp, nil
}

func (s *MemoryService) ListLinks(chatID int64) (trackerapi.ListLinksResponse, error) {
	const pageSize = 100
	offset := 0
	out := trackerapi.ListLinksResponse{Links: make([]trackerapi.LinkResponse, 0)}
	for {
		chunk, err := s.ListLinksPage(chatID, pageSize, offset)
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

func (s *MemoryService) ListLinksPage(chatID int64, limit int, offset int) ([]trackerapi.LinkResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.chats[chatID]; !ok {
		return nil, ErrChatNotFound
	}
	if limit <= 0 {
		limit = len(s.linksByChat[chatID])
	}

	all := make([]trackerapi.LinkResponse, 0, len(s.linksByChat[chatID]))
	for linkURL := range s.linksByChat[chatID] {
		resp := s.linksByURL[linkURL].Response
		resp.Tags = s.getLinkTagsLocked(chatID, linkURL)
		all = append(all, resp)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})
	if offset >= len(all) {
		return []trackerapi.LinkResponse{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (s *MemoryService) ListTrackedLinksPage(limit int, checkedBefore time.Time, afterID int64) ([]TrackedLink, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = len(s.linksByURL)
	}
	all := make([]TrackedLink, 0, len(s.linksByURL))
	for _, link := range s.linksByURL {
		if link.Response.ID <= afterID {
			continue
		}
		if !checkedBefore.IsZero() && !link.LastCheckedAt.IsZero() && link.LastCheckedAt.After(checkedBefore) {
			continue
		}
		clone := TrackedLink{
			Response:      link.Response,
			ChatIDs:       make(map[int64]struct{}, len(link.ChatIDs)),
			LastUpdated:   link.LastUpdated,
			LastCheckedAt: link.LastCheckedAt,
		}
		for chatID := range link.ChatIDs {
			clone.ChatIDs[chatID] = struct{}{}
		}
		all = append(all, clone)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Response.ID < all[j].Response.ID
	})

	end := limit
	if end > len(all) {
		end = len(all)
	}
	return all[:end], nil
}

func (s *MemoryService) UpdateLastUpdated(linkURL string, value time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tracked, ok := s.linksByURL[linkURL]
	if !ok {
		return ErrLinkNotFound
	}
	tracked.LastUpdated = value
	return nil
}

func (s *MemoryService) TouchLastChecked(linkURL string, value time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tracked, ok := s.linksByURL[linkURL]
	if !ok {
		return ErrLinkNotFound
	}
	tracked.LastCheckedAt = value
	return nil
}

func (s *MemoryService) CreateTag(chatID int64, tag string) (Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; !ok {
		return Tag{}, ErrChatNotFound
	}
	normalized := normalizeTag(tag)
	if normalized == "" {
		return Tag{}, ErrInvalidTag
	}
	if _, ok := s.tagsByChat[chatID]; !ok {
		s.tagsByChat[chatID] = make(map[string]Tag)
	}
	if existing, exists := s.tagsByChat[chatID][normalized]; exists {
		return existing, ErrTagExists
	}
	created := Tag{ID: s.nextTagID, ChatID: chatID, Name: normalized}
	s.nextTagID++
	s.tagsByChat[chatID][normalized] = created
	return created, nil
}

func (s *MemoryService) ListTags(chatID int64, limit int, offset int) ([]Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.chats[chatID]; !ok {
		return nil, ErrChatNotFound
	}
	chatTags := s.tagsByChat[chatID]
	out := make([]Tag, 0, len(chatTags))
	for _, tag := range chatTags {
		out = append(out, tag)
	}
	if limit <= 0 {
		limit = len(out)
	}
	if offset >= len(out) {
		return []Tag{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (s *MemoryService) UpdateTag(chatID int64, oldTag string, newTag string) (Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; !ok {
		return Tag{}, ErrChatNotFound
	}
	oldName := normalizeTag(oldTag)
	newName := normalizeTag(newTag)
	if oldName == "" || newName == "" {
		return Tag{}, ErrInvalidTag
	}
	chatTags := s.tagsByChat[chatID]
	existing, ok := chatTags[oldName]
	if !ok {
		return Tag{}, ErrTagNotFound
	}
	if _, exists := chatTags[newName]; exists && oldName != newName {
		return Tag{}, ErrTagExists
	}
	delete(chatTags, oldName)
	existing.Name = newName
	chatTags[newName] = existing

	if byLink, okByLink := s.tagsByChatLink[chatID]; okByLink {
		for _, linkTags := range byLink {
			if _, has := linkTags[oldName]; has {
				delete(linkTags, oldName)
				linkTags[newName] = struct{}{}
			}
		}
	}
	return existing, nil
}

func (s *MemoryService) DeleteTag(chatID int64, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; !ok {
		return ErrChatNotFound
	}
	name := normalizeTag(tag)
	if name == "" {
		return ErrTagNotFound
	}
	chatTags := s.tagsByChat[chatID]
	if _, ok := chatTags[name]; !ok {
		return ErrTagNotFound
	}
	delete(chatTags, name)
	if byLink, okByLink := s.tagsByChatLink[chatID]; okByLink {
		for _, linkTags := range byLink {
			delete(linkTags, name)
		}
	}
	return nil
}

func (s *MemoryService) getLinkTagsLocked(chatID int64, link string) []string {
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
	return deduplicateStringSlice(out)
}

func deduplicateStringSlice(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if slices.Contains(out, trimmed) {
			continue
		}
		out = append(out, trimmed)
	}
	return out
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

func normalizeTag(value string) string {
	return strings.TrimSpace(value)
}

func trimAndValidateLink(link string) (string, error) {
	trimmed := strings.TrimSpace(link)
	if !isValidLink(trimmed) {
		return "", ErrInvalidLink
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

var _ Service = (*MemoryService)(nil)
