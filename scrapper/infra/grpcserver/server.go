package grpcserver

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"google.golang.org/grpc"
)

const serviceName = "scrapper.ScrapperService"

type ScrapperServiceServer interface {
	registerChatHandler(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error)
	deleteChatHandler(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error)
	addLinkHandler(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error)
	removeLinkHandler(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error)
	listLinksHandler(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error)
}

type Server struct {
	handlers *Handlers
}

func NewServer(service *repository.Service) *Server {
	return &Server{
		handlers: NewHandlers(service),
	}
}

func (s *Server) Register(server *grpc.Server) {
	RegisterJSONCodec()
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: serviceName,
		HandlerType: (*ScrapperServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{MethodName: "RegisterChat", Handler: s.handlers.registerChatHandler},
			{MethodName: "DeleteChat", Handler: s.handlers.deleteChatHandler},
			{MethodName: "AddLink", Handler: s.handlers.addLinkHandler},
			{MethodName: "RemoveLink", Handler: s.handlers.removeLinkHandler},
			{MethodName: "ListLinks", Handler: s.handlers.listLinksHandler},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "scrapper-json",
	}, s.handlers)
}

func Register(server *grpc.Server, service *repository.Service) {
	NewServer(service).Register(server)
}
