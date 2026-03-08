package scrapper

import (
	"context"
	"net"
	"testing"
	"time"

	appscrapper "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/grpcscrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"google.golang.org/grpc"
)

func TestGRPCClient_AddAndListLinks(t *testing.T) {
	service := appscrapper.NewService()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	grpcscrapper.Register(server, service)
	go server.Serve(listener)
	defer server.Stop()

	client, err := NewGRPCClient(listener.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatalf("new grpc client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	if err := client.RegisterChat(ctx, 1); err != nil {
		t.Fatalf("register chat: %v", err)
	}
	if _, err := client.AddLink(ctx, 1, api.AddLinkRequest{Link: "https://github.com/user/repo"}); err != nil {
		t.Fatalf("add link: %v", err)
	}
	resp, err := client.ListLinks(ctx, 1)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if resp.Size != 1 {
		t.Fatalf("expected 1 link, got %d", resp.Size)
	}
}
