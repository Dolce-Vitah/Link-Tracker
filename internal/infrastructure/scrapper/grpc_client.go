package scrapper

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/tracker"
	grpcscrapper "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/grpcscrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/status"
)

type GRPCClient struct {
	conn   *grpc.ClientConn
	target string
}

func NewGRPCClient(target string, timeout time.Duration) (*GRPCClient, error) {
	encoding.RegisterCodec(grpcscrapperJSONCodec{})
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.ForceCodec(grpcscrapperJSONCodec{})))
	if err != nil {
		return nil, fmt.Errorf("dial scrapper grpc target: %w", err)
	}
	return &GRPCClient{
		conn:   conn,
		target: target,
	}, nil
}

func (c *GRPCClient) RegisterChat(ctx context.Context, chatID int64) error {
	var response grpcscrapper.RegisterChatResponse
	err := c.conn.Invoke(ctx, "/scrapper.ScrapperService/RegisterChat", grpcscrapper.RegisterChatRequest{ChatID: chatID}, &response)
	return mapGRPCError(err)
}

func (c *GRPCClient) DeleteChat(ctx context.Context, chatID int64) error {
	var response grpcscrapper.DeleteChatResponse
	err := c.conn.Invoke(ctx, "/scrapper.ScrapperService/DeleteChat", grpcscrapper.DeleteChatRequest{ChatID: chatID}, &response)
	return mapGRPCError(err)
}

func (c *GRPCClient) AddLink(ctx context.Context, chatID int64, request api.AddLinkRequest) (api.LinkResponse, error) {
	var response grpcscrapper.AddLinkGRPCResponse
	err := c.conn.Invoke(ctx, "/scrapper.ScrapperService/AddLink", grpcscrapper.AddLinkGRPCRequest{
		ChatID: chatID,
		Body:   request,
	}, &response)
	if err != nil {
		return api.LinkResponse{}, mapGRPCError(err)
	}
	return response.Link, nil
}

func (c *GRPCClient) RemoveLink(ctx context.Context, chatID int64, request api.RemoveLinkRequest) (api.LinkResponse, error) {
	var response grpcscrapper.RemoveLinkGRPCResponse
	err := c.conn.Invoke(ctx, "/scrapper.ScrapperService/RemoveLink", grpcscrapper.RemoveLinkGRPCRequest{
		ChatID: chatID,
		Body:   request,
	}, &response)
	if err != nil {
		return api.LinkResponse{}, mapGRPCError(err)
	}
	return response.Link, nil
}

func (c *GRPCClient) ListLinks(ctx context.Context, chatID int64) (api.ListLinksResponse, error) {
	var response grpcscrapper.ListLinksResponse
	err := c.conn.Invoke(ctx, "/scrapper.ScrapperService/ListLinks", grpcscrapper.ListLinksRequest{ChatID: chatID}, &response)
	if err != nil {
		return api.ListLinksResponse{}, mapGRPCError(err)
	}
	return response.Body, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
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
	default:
		return err
	}
}

type grpcscrapperJSONCodec struct{}

func (grpcscrapperJSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (grpcscrapperJSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (grpcscrapperJSONCodec) Name() string {
	return "json"
}
