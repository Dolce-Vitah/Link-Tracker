package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

const (
	TransportModeHTTP = "http"
	TransportModeGRPC = "grpc"
)

type Config struct {
	TelegramToken       string `json:"telegram_token"`
	TelegramAPIURL      string `json:"telegram_api_url,omitempty"`
	BotHTTPAddress      string `json:"bot_http_address"`
	ScrapperBaseURL     string `json:"scrapper_base_url"`
	ScrapperGRPCTarget  string `json:"scrapper_grpc_target"`
	TransportMode       string `json:"transport_mode"`
	ExternalHTTPTimeout string `json:"external_http_timeout"`
}

func Load(path string) (*Config, error) {
	var cfg Config
	if err := decode(path, "config.json", &cfg); err != nil {

		return nil, err
	}
	applyDefaults(&cfg)
	if err := normalizeAndValidate(&cfg); err != nil {

		return nil, err
	}

	return &cfg, nil
}

func decode(primaryPath string, fallbackPath string, out any) error {
	err := decodeSingle(primaryPath, out)
	if err == nil {

		return nil
	}

	if errors.Is(err, fs.ErrNotExist) && fallbackPath != "" {
		fallbackErr := decodeSingle(fallbackPath, out)
		if fallbackErr == nil {

			return nil
		}

		return fmt.Errorf("load fallback config file %q: %w", fallbackPath, fallbackErr)
	}

	return err
}

func decodeSingle(path string, out any) (_ error) {
	file, err := os.Open(path)
	if err != nil {

		return fmt.Errorf("open config file %q: %w", path, err)
	}

	defer func() {
		_ = file.Close()
	}()

	decodeErr := json.NewDecoder(file).Decode(out)
	if decodeErr != nil {

		return fmt.Errorf("decode config file %q: %w", path, decodeErr)
	}

	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.BotHTTPAddress == "" {
		cfg.BotHTTPAddress = ":8081"
	}
	if cfg.ScrapperBaseURL == "" {
		cfg.ScrapperBaseURL = "http://localhost:8080"
	}
	if cfg.ScrapperGRPCTarget == "" {
		cfg.ScrapperGRPCTarget = "localhost:8090"
	}
	if cfg.TransportMode == "" {
		cfg.TransportMode = TransportModeHTTP
	}
	if cfg.ExternalHTTPTimeout == "" {
		cfg.ExternalHTTPTimeout = "5s"
	}
}

func normalizeAndValidate(cfg *Config) error {
	mode := strings.ToLower(strings.TrimSpace(cfg.TransportMode))
	switch mode {
	case TransportModeHTTP, TransportModeGRPC:
		cfg.TransportMode = mode
		return nil
	default:
		return fmt.Errorf("invalid transport_mode %q: expected %q or %q", cfg.TransportMode, TransportModeHTTP, TransportModeGRPC)
	}
}
