package trackerclient

import (
	"context"
	"net"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/grpcserver"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
	"google.golang.org/grpc"
)

func TestGRPCClient_AddAndListLinks(t *testing.T) {
	service := repository.NewService()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	grpcserver.Register(server, service)
	go server.Serve(listener)
	defer server.Stop()

	client, err := NewGRPCClient(listener.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatalf("new grpc client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	registerErr := client.RegisterChat(ctx, 1)
	if registerErr != nil {
		t.Fatalf("register chat: %v", registerErr)
	}
	_, addErr := client.AddLink(ctx, 1, api.AddLinkRequest{Link: "https://github.com/user/repo"})
	if addErr != nil {
		t.Fatalf("add link: %v", addErr)
	}
	resp, err := client.ListLinks(ctx, 1)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if resp.Size != 1 {
		t.Fatalf("expected 1 link, got %d", resp.Size)
	}
}
