package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FallsBackToConfigJSONWhenPrimaryMissing(t *testing.T) {
	tmp := t.TempDir()
	primary := filepath.Join(tmp, "config.scrapper.json")
	fallback := filepath.Join(tmp, "config.json")

	content := `{
		"bot_base_url": "http://localhost:18081",
		"scrapper_http_address": ":18080",
		"scrapper_grpc_address": ":18090",
		"scheduler_interval": "15s",
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
	if cfg.BotBaseURL != "http://localhost:18081" {
		t.Fatalf("expected bot base url from fallback, got %q", cfg.BotBaseURL)
	}
}
