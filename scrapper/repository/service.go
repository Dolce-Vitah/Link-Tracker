package repository

import (
	"errors"
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
	OutboxService
	TagService
}
