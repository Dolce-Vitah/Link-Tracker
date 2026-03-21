package trackerclient

import (
	"context"
	"fmt"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/grpcserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type GRPCClient struct {
	conn   *grpc.ClientConn
	target string
}

func NewGRPCClient(target string, _ time.Duration) (*GRPCClient, error) {
	grpcserver.RegisterJSONCodec()
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(grpcserver.JSONCodec{})),
	)
	if err != nil {
		return nil, fmt.Errorf("dial scrapper grpc target: %w", err)
	}
	conn.Connect()
	return &GRPCClient{conn: conn, target: target}, nil
}

func (c *GRPCClient) RegisterChat(ctx context.Context, chatID int64) error {
	var response grpcserver.RegisterChatResponse
	err := c.conn.Invoke(ctx, "/scrapper.ScrapperService/RegisterChat", grpcserver.RegisterChatRequest{ChatID: chatID}, &response)
	return mapGRPCError(err)
}

func (c *GRPCClient) DeleteChat(ctx context.Context, chatID int64) error {
	var response grpcserver.DeleteChatResponse
	err := c.conn.Invoke(ctx, "/scrapper.ScrapperService/DeleteChat", grpcserver.DeleteChatRequest{ChatID: chatID}, &response)
	return mapGRPCError(err)
}

func (c *GRPCClient) AddLink(ctx context.Context, chatID int64, request api.AddLinkRequest) (api.LinkResponse, error) {
	var response grpcserver.AddLinkGRPCResponse
	err := c.conn.Invoke(ctx, "/scrapper.ScrapperService/AddLink", grpcserver.AddLinkGRPCRequest{ChatID: chatID, Body: request}, &response)
	if err != nil {
		return api.LinkResponse{}, mapGRPCError(err)
	}
	return response.Link, nil
}

func (c *GRPCClient) RemoveLink(ctx context.Context, chatID int64, request api.RemoveLinkRequest) (api.LinkResponse, error) {
	var response grpcserver.RemoveLinkGRPCResponse
	err := c.conn.Invoke(ctx, "/scrapper.ScrapperService/RemoveLink", grpcserver.RemoveLinkGRPCRequest{ChatID: chatID, Body: request}, &response)
	if err != nil {
		return api.LinkResponse{}, mapGRPCError(err)
	}
	return response.Link, nil
}

func (c *GRPCClient) ListLinks(ctx context.Context, chatID int64) (api.ListLinksResponse, error) {
	var response grpcserver.ListLinksResponse
	err := c.conn.Invoke(ctx, "/scrapper.ScrapperService/ListLinks", grpcserver.ListLinksRequest{ChatID: chatID}, &response)
	if err != nil {
		return api.ListLinksResponse{}, mapGRPCError(err)
	}
	return response.Body, nil
}

func (c *GRPCClient) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("close grpc client connection: %w", err)
	}
	return nil
}

func mapGRPCError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() {
	case codes.NotFound:
		return fmt.Errorf("%w: %s", tracker.ErrNotFound, st.Message())
	case codes.AlreadyExists:
		return fmt.Errorf("%w: %s", tracker.ErrAlreadyExists, st.Message())
	case codes.InvalidArgument:
		return fmt.Errorf("%w: %s", tracker.ErrBadRequest, st.Message())
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
