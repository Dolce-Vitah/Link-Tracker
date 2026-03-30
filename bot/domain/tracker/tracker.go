package tracker

import (
	"context"
	"errors"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrBadRequest    = errors.New("bad request")
)

type Client interface {
	RegisterChat(ctx context.Context, chatID int64) error
	DeleteChat(ctx context.Context, chatID int64) error
	AddLink(ctx context.Context, chatID int64, request trackerapi.AddLinkRequest) (trackerapi.LinkResponse, error)
	RemoveLink(ctx context.Context, chatID int64, request trackerapi.RemoveLinkRequest) (trackerapi.LinkResponse, error)
	ListLinks(ctx context.Context, chatID int64) (trackerapi.ListLinksResponse, error)
}
