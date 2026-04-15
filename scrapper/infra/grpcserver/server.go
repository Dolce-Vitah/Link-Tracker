package grpcserver

import (
	scrapperv1 "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/api/proto/scrapper/v1"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"google.golang.org/grpc"
)

type Server struct {
	handlers *Handlers
}

func NewServer(service *repository.Service) *Server {
	return &Server{
		handlers: NewHandlers(service),
	}
}

func (s *Server) Register(server *grpc.Server) {

	scrapperv1.RegisterScrapperServiceServer(server, s.handlers)
}

func Register(server *grpc.Server, service *repository.Service) {

	NewServer(service).Register(server)
}
