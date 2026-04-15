package trackerclient

import (
	"context"

	scrapperv1 "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/api/proto/scrapper/v1"
	"google.golang.org/grpc"
)

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
