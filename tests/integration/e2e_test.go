//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestBotAndScrapperContainersStart(t *testing.T) {
	requireTestcontainers(t)

	ctx := context.Background()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/getMe") {
			w.Write([]byte(`{"ok": true, "result": {"id": 1, "is_bot": true, "first_name": "TestBot", "username": "test_bot"}}`))
			return
		}
		if strings.Contains(r.URL.Path, "/setMyCommands") {
			w.Write([]byte(`{"ok": true, "result": true}`))
			return
		}
		w.Write([]byte(`{"ok": true, "result": true}`))
	})

	mockTg := httptest.NewUnstartedServer(handler)

	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("failed to listen on 0.0.0.0: %v", err)
	}
	mockTg.Listener = l
	mockTg.Start()
	defer mockTg.Close()

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	tgAPIEndpoint := fmt.Sprintf("http://host.docker.internal:%s/bot%%s/%%s", port)

	postgresReq := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "link_tracker",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}
	postgresC, pgErr := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: postgresReq,
		Started:          true,
	})
	if pgErr != nil {
		t.Fatalf("start postgres container: %v", pgErr)
	}
	defer postgresC.Terminate(ctx)

	pgPort, mappedErr := postgresC.MappedPort(ctx, "5432/tcp")
	if mappedErr != nil {
		t.Fatalf("map postgres port: %v", mappedErr)
	}
	dbDSN := fmt.Sprintf("postgres://postgres:postgres@host.docker.internal:%s/link_tracker?sslmode=disable", pgPort.Port())

	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "config.json")
	testConfigContent := fmt.Sprintf(`{
		"telegram_token": "fake-test-token",
		"telegram_api_url": "%s",
		"bot_http_address": ":8081",
		"bot_base_url": "http://localhost:8081",
		"scrapper_http_address": ":8080",
		"scrapper_base_url": "http://localhost:8080",
		"scrapper_grpc_address": ":8090",
		"scrapper_grpc_target": "localhost:8090",
		"transport_mode": "http",
		"scheduler_interval": "30s",
		"external_http_timeout": "5s",
		"access_type": "SQL",
		"db_dsn": "%s",
		"db_max_open_conns": 10,
		"db_max_idle_conns": 5,
		"db_conn_max_lifetime": "30m",
		"auto_migrate": true
	}`, tgAPIEndpoint, dbDSN)

	if err := os.WriteFile(testConfigPath, []byte(testConfigContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	scrapperReq := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "../../",
			Dockerfile: "Dockerfile.scrapper",
		},
		ExposedPorts: []string{"8080/tcp", "8090/tcp"},
		ExtraHosts:   []string{"host.docker.internal:host-gateway"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      testConfigPath,
				ContainerFilePath: "/app/config.json",
				FileMode:          0644,
			},
		},
		WaitingFor: wait.ForListeningPort("8080/tcp").WithStartupTimeout(40 * time.Second),
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
		ExtraHosts:   []string{"host.docker.internal:host-gateway"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      testConfigPath,
				ContainerFilePath: "/app/config.json",
				FileMode:          0644,
			},
		},
		WaitingFor: wait.ForHTTP("/updates").
			WithPort("8081/tcp").
			WithStatusCodeMatcher(func(status int) bool {
				return status == http.StatusMethodNotAllowed || status == http.StatusOK
			}).
			WithStartupTimeout(40 * time.Second),
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
	botPort, err := botC.MappedPort(ctx, "8081/tcp")
	if err != nil {
		t.Fatalf("bot mapped port: %v", err)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post("http://"+host+":"+botPort.Port()+"/updates", "application/json", strings.NewReader(`{"id":1,"url":"https://github.com/user/repo","description":"ping","tgChatIds":[1]}`))
	if err != nil {
		t.Fatalf("post /updates to bot: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from bot /updates, got %d", resp.StatusCode)
	}
}
