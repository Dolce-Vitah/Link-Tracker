//go:build integration

package integration

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestBotAndScrapperContainersStart(t *testing.T) {
	ctx := context.Background()

	scrapperReq := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "../../",
			Dockerfile: "Dockerfile.scrapper",
		},
		ExposedPorts: []string{"8080/tcp", "8090/tcp"},
		WaitingFor:   wait.ForListeningPort("8080/tcp").WithStartupTimeout(40 * time.Second),
	}
	scrapperC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: scrapperReq,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start scrapper container: %v", err)
	}
	defer scrapperC.Terminate(ctx)

	botReq := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "../../",
			Dockerfile: "Dockerfile.bot",
		},
		ExposedPorts: []string{"8081/tcp"},
		WaitingFor:   wait.ForListeningPort("8081/tcp").WithStartupTimeout(40 * time.Second),
	}
	botC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: botReq,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start bot container: %v", err)
	}
	defer botC.Terminate(ctx)

	host, err := botC.Host(ctx)
	if err != nil {
		t.Fatalf("bot host: %v", err)
	}
	port, err := botC.MappedPort(ctx, "8081/tcp")
	if err != nil {
		t.Fatalf("bot mapped port: %v", err)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post("http://"+host+":"+port.Port()+"/updates", "application/json", strings.NewReader(`{"id":1,"url":"https://github.com/user/repo","description":"ping","tgChatIds":[1]}`))
	if err != nil {
		t.Fatalf("post /updates to bot: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from bot /updates, got %d", resp.StatusCode)
	}
}
