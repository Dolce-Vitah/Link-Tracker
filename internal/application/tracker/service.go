package tracker

import (
	"context"
	"errors"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrBadRequest    = errors.New("bad request")
)

type Service interface {
	RegisterChat(ctx context.Context, chatID int64) error
	DeleteChat(ctx context.Context, chatID int64) error
	AddLink(ctx context.Context, chatID int64, request api.AddLinkRequest) (api.LinkResponse, error)
	RemoveLink(ctx context.Context, chatID int64, request api.RemoveLinkRequest) (api.LinkResponse, error)
	ListLinks(ctx context.Context, chatID int64) (api.ListLinksResponse, error)
}
