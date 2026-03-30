package trackerclient

import (
	"context"
	"fmt"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/grpcapi/scrapperv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type GRPCClient struct {
	client scrapperGRPCClient
	closer grpcCloser
}

type grpcCloser interface {
	Close() error
}

type scrapperGRPCClient interface {
	RegisterChat(ctx context.Context, in *scrapperv1.RegisterChatRequest, opts ...grpc.CallOption) (*scrapperv1.RegisterChatResponse, error)
	DeleteChat(ctx context.Context, in *scrapperv1.DeleteChatRequest, opts ...grpc.CallOption) (*scrapperv1.DeleteChatResponse, error)
	AddLink(ctx context.Context, in *scrapperv1.AddLinkRequest, opts ...grpc.CallOption) (*scrapperv1.AddLinkResponse, error)
	RemoveLink(ctx context.Context, in *scrapperv1.RemoveLinkRequest, opts ...grpc.CallOption) (*scrapperv1.RemoveLinkResponse, error)
	ListLinks(ctx context.Context, in *scrapperv1.ListLinksRequest, opts ...grpc.CallOption) (*scrapperv1.ListLinksResponse, error)
}

func NewGRPCClient(target string, _ time.Duration) (*GRPCClient, error) {
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {

		return nil, fmt.Errorf("dial scrapper grpc target: %w", err)
	}

	conn.Connect()

	return &GRPCClient{
		client: scrapperv1.NewScrapperServiceClient(conn),
		closer: conn,
	}, nil
}

func (c *GRPCClient) RegisterChat(ctx context.Context, chatID int64) error {
	_, err := c.client.RegisterChat(ctx, &scrapperv1.RegisterChatRequest{ChatId: chatID})

	return mapGRPCError(err)
}

func (c *GRPCClient) DeleteChat(ctx context.Context, chatID int64) error {
	_, err := c.client.DeleteChat(ctx, &scrapperv1.DeleteChatRequest{ChatId: chatID})

	return mapGRPCError(err)
}

func (c *GRPCClient) AddLink(ctx context.Context, chatID int64, request api.AddLinkRequest) (api.LinkResponse, error) {
	response, err := c.client.AddLink(ctx, &scrapperv1.AddLinkRequest{
		ChatId: chatID,
		Body: &scrapperv1.AddLinkPayload{
			Link:    request.Link,
			Tags:    request.Tags,
			Filters: request.Filters,
		},
	})
	if err != nil {

		return api.LinkResponse{}, mapGRPCError(err)
	}

	return fromProtoLink(response.GetLink()), nil
}

func (c *GRPCClient) RemoveLink(ctx context.Context, chatID int64, request api.RemoveLinkRequest) (api.LinkResponse, error) {
	response, err := c.client.RemoveLink(ctx, &scrapperv1.RemoveLinkRequest{
		ChatId: chatID,
		Body: &scrapperv1.RemoveLinkPayload{
			Link: request.Link,
		},
	})
	if err != nil {

		return api.LinkResponse{}, mapGRPCError(err)
	}

	return fromProtoLink(response.GetLink()), nil
}

func (c *GRPCClient) ListLinks(ctx context.Context, chatID int64) (api.ListLinksResponse, error) {
	response, err := c.client.ListLinks(ctx, &scrapperv1.ListLinksRequest{ChatId: chatID})
	if err != nil {

		return api.ListLinksResponse{}, mapGRPCError(err)
	}

	return fromProtoListLinks(response), nil
}

func (c *GRPCClient) Close() error {
	if c.closer == nil {

		return nil
	}

	if err := c.closer.Close(); err != nil {

		return fmt.Errorf("close grpc client connection: %w", err)
	}

	return nil
}

func mapGRPCError(err error) error {
	if err == nil {

		return nil
	}

	grpcStatus, isGRPCStatus := status.FromError(err)
	if !isGRPCStatus {

		return err
	}

	switch grpcStatus.Code() {
	case codes.NotFound:
		return fmt.Errorf("%w: %s", tracker.ErrNotFound, grpcStatus.Message())
	case codes.AlreadyExists:
		return fmt.Errorf("%w: %s", tracker.ErrAlreadyExists, grpcStatus.Message())
	case codes.InvalidArgument:
		return fmt.Errorf("%w: %s", tracker.ErrBadRequest, grpcStatus.Message())
	case codes.OK,
		codes.Canceled,
		codes.Unknown,
		codes.DeadlineExceeded,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unimplemented,
		codes.Internal,
		codes.Unavailable,
		codes.DataLoss,
		codes.Unauthenticated:
		return err
	default:
		return err
	}
}

func fromProtoLink(link *scrapperv1.LinkResponse) api.LinkResponse {
	if link == nil {

		return api.LinkResponse{}
	}

	return api.LinkResponse{
		ID:      link.GetId(),
		URL:     link.GetUrl(),
		Tags:    link.GetTags(),
		Filters: link.GetFilters(),
	}
}

func fromProtoListLinks(response *scrapperv1.ListLinksResponse) api.ListLinksResponse {
	if response == nil {

		return api.ListLinksResponse{}
	}

	links := make([]api.LinkResponse, 0, len(response.GetLinks()))
	for _, link := range response.GetLinks() {
		links = append(links, fromProtoLink(link))
	}

	return api.ListLinksResponse{
		Links: links,
		Size:  response.GetSize(),
	}
}
