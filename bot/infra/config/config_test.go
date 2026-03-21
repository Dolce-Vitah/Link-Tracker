package config

import (
	"os"
	"path/filepath"
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
