package handler_test

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type fakeTracker struct {
	links        []api.LinkResponse
	addLinkFn    func(ctx context.Context, chatID int64, request api.AddLinkRequest) (api.LinkResponse, error)
	removeLinkFn func(ctx context.Context, chatID int64, request api.RemoveLinkRequest) (api.LinkResponse, error)
}

func (f *fakeTracker) RegisterChat(_ context.Context, _ int64) error { return nil }
func (f *fakeTracker) DeleteChat(_ context.Context, _ int64) error   { return nil }
func (f *fakeTracker) AddLink(ctx context.Context, chatID int64, request api.AddLinkRequest) (api.LinkResponse, error) {
	if f.addLinkFn != nil {
		return f.addLinkFn(ctx, chatID, request)
	}
	return api.LinkResponse{}, nil
}
func (f *fakeTracker) RemoveLink(ctx context.Context, chatID int64, request api.RemoveLinkRequest) (api.LinkResponse, error) {
	if f.removeLinkFn != nil {
		return f.removeLinkFn(ctx, chatID, request)
	}
	return api.LinkResponse{}, nil
}
func (f *fakeTracker) ListLinks(_ context.Context, _ int64) (api.ListLinksResponse, error) {
	return api.ListLinksResponse{Links: f.links, Size: int32(len(f.links))}, nil
}

var _ tracker.Service = (*fakeTracker)(nil)
