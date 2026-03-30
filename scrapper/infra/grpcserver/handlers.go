package grpcserver

import (
	"context"
	"errors"
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/grpcapi/scrapperv1"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handlers struct {
	service *repository.Service
	scrapperv1.UnimplementedScrapperServiceServer
}

func NewHandlers(service *repository.Service) *Handlers {

	return &Handlers{service: service}
}

func (h *Handlers) RegisterChat(_ context.Context, req *scrapperv1.RegisterChatRequest) (*scrapperv1.RegisterChatResponse, error) {
	if req == nil {

		return nil, fmt.Errorf("register chat request is nil: %w", status.Error(codes.InvalidArgument, "request must not be nil"))
	}

	if err := h.service.RegisterChat(req.GetChatId()); err != nil {

		return nil, mapError(err)
	}

	return &scrapperv1.RegisterChatResponse{}, nil
}

func (h *Handlers) DeleteChat(_ context.Context, req *scrapperv1.DeleteChatRequest) (*scrapperv1.DeleteChatResponse, error) {
	if req == nil {

		return nil, fmt.Errorf("delete chat request is nil: %w", status.Error(codes.InvalidArgument, "request must not be nil"))
	}

	if err := h.service.DeleteChat(req.GetChatId()); err != nil {

		return nil, mapError(err)
	}

	return &scrapperv1.DeleteChatResponse{}, nil
}

func (h *Handlers) AddLink(_ context.Context, req *scrapperv1.AddLinkRequest) (*scrapperv1.AddLinkResponse, error) {
	if req == nil {

		return nil, fmt.Errorf("add link request is nil: %w", status.Error(codes.InvalidArgument, "request must not be nil"))
	}

	if req.GetBody() == nil {

		return nil, fmt.Errorf("add link request body is nil: %w", status.Error(codes.InvalidArgument, "request body must not be nil"))
	}

	link, err := h.service.AddLink(req.GetChatId(), trackerapi.AddLinkRequest{
		Link:    req.GetBody().GetLink(),
		Tags:    req.GetBody().GetTags(),
		Filters: req.GetBody().GetFilters(),
	})
	if err != nil {

		return nil, mapError(err)
	}

	return &scrapperv1.AddLinkResponse{Link: toProtoLinkResponse(link)}, nil
}

func (h *Handlers) RemoveLink(_ context.Context, req *scrapperv1.RemoveLinkRequest) (*scrapperv1.RemoveLinkResponse, error) {
	if req == nil {

		return nil, fmt.Errorf("remove link request is nil: %w", status.Error(codes.InvalidArgument, "request must not be nil"))
	}

	if req.GetBody() == nil {

		return nil, fmt.Errorf("remove link request body is nil: %w", status.Error(codes.InvalidArgument, "request body must not be nil"))
	}

	link, err := h.service.RemoveLink(req.GetChatId(), trackerapi.RemoveLinkRequest{Link: req.GetBody().GetLink()})
	if err != nil {

		return nil, mapError(err)
	}

	return &scrapperv1.RemoveLinkResponse{Link: toProtoLinkResponse(link)}, nil
}

func (h *Handlers) ListLinks(_ context.Context, req *scrapperv1.ListLinksRequest) (*scrapperv1.ListLinksResponse, error) {
	if req == nil {

		return nil, fmt.Errorf("list links request is nil: %w", status.Error(codes.InvalidArgument, "request must not be nil"))
	}

	resp, err := h.service.ListLinks(req.GetChatId())
	if err != nil {

		return nil, mapError(err)
	}

	return toProtoListLinksResponse(resp), nil
}

func toProtoLinkResponse(link trackerapi.LinkResponse) *scrapperv1.LinkResponse {
	return &scrapperv1.LinkResponse{
		Id:      link.ID,
		Url:     link.URL,
		Tags:    link.Tags,
		Filters: link.Filters,
	}
}

func toProtoListLinksResponse(response trackerapi.ListLinksResponse) *scrapperv1.ListLinksResponse {
	links := make([]*scrapperv1.LinkResponse, 0, len(response.Links))

	for _, link := range response.Links {
		links = append(links, toProtoLinkResponse(link))
	}

	return &scrapperv1.ListLinksResponse{
		Links: links,
		Size:  response.Size,
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, repository.ErrChatExists), errors.Is(err, repository.ErrLinkExists):

		return fmt.Errorf("map grpc conflict error: %w", status.Error(codes.AlreadyExists, err.Error()))
	case errors.Is(err, repository.ErrChatNotFound), errors.Is(err, repository.ErrLinkNotFound):

		return fmt.Errorf("map grpc not found error: %w", status.Error(codes.NotFound, err.Error()))
	case errors.Is(err, repository.ErrInvalidLink):

		return fmt.Errorf("map grpc invalid argument error: %w", status.Error(codes.InvalidArgument, err.Error()))
	default:

		return fmt.Errorf("map grpc internal error: %w", status.Error(codes.Internal, fmt.Sprintf("internal server error: %s", err.Error())))
	}
}
