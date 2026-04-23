package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_FallsBackToConfigJSONWhenPrimaryMissing(t *testing.T) {
	tmp := t.TempDir()
	primary := filepath.Join(tmp, "config.bot.json")
	fallback := filepath.Join(tmp, "config.json")

	content := `{
		"telegram_token": "token",
		"bot_http_address": ":18081",
		"scrapper_base_url": "http://localhost:18080",
		"scrapper_grpc_target": "localhost:18090",
		"transport_mode": "http",
		"external_http_timeout": "3s"
	}`
	if err := os.WriteFile(fallback, []byte(content), 0o644); err != nil {
		t.Fatalf("write fallback config: %v", err)
	}

	t.Chdir(tmp)

	cfg, loadErr := Load(primary)
	if loadErr != nil {
		t.Fatalf("load with fallback: %v", loadErr)
	}
	if cfg.TelegramToken != "token" {
		t.Fatalf("expected telegram token from fallback, got %q", cfg.TelegramToken)
	}
}

func TestLoad_InvalidTransportMode_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.bot.json")
	content := `{
		"telegram_token": "token",
		"transport_mode": "mq"
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatalf("expected invalid transport mode error")
	}
	if !strings.Contains(err.Error(), "invalid transport_mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_TransportMode_Normalized(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.bot.json")
	content := `{
		"telegram_token": "token",
		"transport_mode": " GRPC "
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.TransportMode != TransportModeGRPC {
		t.Fatalf("expected normalized mode %q, got %q", TransportModeGRPC, cfg.TransportMode)
	}
}

func TestLoad_NotificationMode_DefaultsToKafka(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.bot.json")
	content := `{
		"telegram_token": "token"
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.NotificationMode != NotificationModeKafka {
		t.Fatalf("expected notification mode %q, got %q", NotificationModeKafka, cfg.NotificationMode)
	}
}
