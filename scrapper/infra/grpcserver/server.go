package grpcserver

import (
	"context"
	"errors"
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/adapters/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	service *repository.Service
}

func Register(server *grpc.Server, service *repository.Service) {
	RegisterJSONCodec()
	s := &Server{service: service}
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: serviceName,
		HandlerType: (*ScrapperServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{MethodName: "RegisterChat", Handler: s.registerChatHandler},
			{MethodName: "DeleteChat", Handler: s.deleteChatHandler},
			{MethodName: "AddLink", Handler: s.addLinkHandler},
			{MethodName: "RemoveLink", Handler: s.removeLinkHandler},
			{MethodName: "ListLinks", Handler: s.listLinksHandler},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "scrapper-json",
	}, s)
}

//nolint:revive // gRPC handler signature follows grpc.MethodDesc handler contract.
func (s *Server) registerChatHandler(_ any, _ context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	var req dto.RegisterChatRequest
	if err := dec(&req); err != nil {
		return nil, fmt.Errorf("decode register chat request: %w", status.Error(codes.InvalidArgument, err.Error()))
	}
	if err := s.service.RegisterChat(req.ChatID); err != nil {
		return nil, mapError(err)
	}
	return dto.RegisterChatResponse{}, nil
}

//nolint:revive // gRPC handler signature follows grpc.MethodDesc handler contract.
func (s *Server) deleteChatHandler(_ any, _ context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	var req dto.DeleteChatRequest
	if err := dec(&req); err != nil {
		return nil, fmt.Errorf("decode delete chat request: %w", status.Error(codes.InvalidArgument, err.Error()))
	}
	if err := s.service.DeleteChat(req.ChatID); err != nil {
		return nil, mapError(err)
	}
	return dto.DeleteChatResponse{}, nil
}

//nolint:revive // gRPC handler signature follows grpc.MethodDesc handler contract.
func (s *Server) addLinkHandler(_ any, _ context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	var req dto.AddLinkGRPCRequest
	if err := dec(&req); err != nil {
		return nil, fmt.Errorf("decode add link request: %w", status.Error(codes.InvalidArgument, err.Error()))
	}
	link, err := s.service.AddLink(req.ChatID, req.Body)
	if err != nil {
		return nil, mapError(err)
	}
	return dto.AddLinkGRPCResponse{Link: link}, nil
}

//nolint:revive // gRPC handler signature follows grpc.MethodDesc handler contract.
func (s *Server) removeLinkHandler(_ any, _ context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	var req dto.RemoveLinkGRPCRequest
	if err := dec(&req); err != nil {
		return nil, fmt.Errorf("decode remove link request: %w", status.Error(codes.InvalidArgument, err.Error()))
	}
	link, err := s.service.RemoveLink(req.ChatID, req.Body)
	if err != nil {
		return nil, mapError(err)
	}
	return dto.RemoveLinkGRPCResponse{Link: link}, nil
}

//nolint:revive // gRPC handler signature follows grpc.MethodDesc handler contract.
func (s *Server) listLinksHandler(_ any, _ context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	var req dto.ListLinksRequest
	if err := dec(&req); err != nil {
		return nil, fmt.Errorf("decode list links request: %w", status.Error(codes.InvalidArgument, err.Error()))
	}
	resp, err := s.service.ListLinks(req.ChatID)
	if err != nil {
		return nil, mapError(err)
	}
	return dto.ListLinksResponse{Body: resp}, nil
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
