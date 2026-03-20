package grpcscrapper

import (
	"context"
	"errors"
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/scrapper"
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
	service *scrapper.Service
}

func Register(server *grpc.Server, service *scrapper.Service) {
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

func (s *Server) registerChatHandler(_ any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	var req RegisterChatRequest
	if err := dec(&req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.service.RegisterChat(req.ChatID); err != nil {
		return nil, mapError(err)
	}
	return RegisterChatResponse{}, nil
}

func (s *Server) deleteChatHandler(_ any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	var req DeleteChatRequest
	if err := dec(&req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.service.DeleteChat(req.ChatID); err != nil {
		return nil, mapError(err)
	}
	return DeleteChatResponse{}, nil
}

func (s *Server) addLinkHandler(_ any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	var req AddLinkGRPCRequest
	if err := dec(&req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	link, err := s.service.AddLink(req.ChatID, req.Body)
	if err != nil {
		return nil, mapError(err)
	}
	return AddLinkGRPCResponse{Link: link}, nil
}

func (s *Server) removeLinkHandler(_ any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	var req RemoveLinkGRPCRequest
	if err := dec(&req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	link, err := s.service.RemoveLink(req.ChatID, req.Body)
	if err != nil {
		return nil, mapError(err)
	}
	return RemoveLinkGRPCResponse{Link: link}, nil
}

func (s *Server) listLinksHandler(_ any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	var req ListLinksRequest
	if err := dec(&req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	resp, err := s.service.ListLinks(req.ChatID)
	if err != nil {
		return nil, mapError(err)
	}
	return ListLinksResponse{Body: resp}, nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, scrapper.ErrChatExists), errors.Is(err, scrapper.ErrLinkExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, scrapper.ErrChatNotFound), errors.Is(err, scrapper.ErrLinkNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, scrapper.ErrInvalidLink):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, fmt.Sprintf("internal server error: %s", err.Error()))
	}
}
