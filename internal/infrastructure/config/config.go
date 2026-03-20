package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	TelegramToken       string `json:"telegram_token"`
	TelegramAPIURL      string `json:"telegram_api_url,omitempty"`
	BotHTTPAddress      string `json:"bot_http_address"`
	BotBaseURL          string `json:"bot_base_url"`
	ScrapperHTTPAddress string `json:"scrapper_http_address"`
	ScrapperBaseURL     string `json:"scrapper_base_url"`
	ScrapperGRPCAddress string `json:"scrapper_grpc_address"`
	ScrapperGRPCTarget  string `json:"scrapper_grpc_target"`
	TransportMode       string `json:"transport_mode"`
	SchedulerInterval   string `json:"scheduler_interval"`
	ExternalHTTPTimeout string `json:"external_http_timeout"`
}

func Load(path string) (_ *Config, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file %q: %w", path, err)
	}

	defer func() {
		closeErr := file.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close config file %q: %w", path, closeErr)
		}
	}()

	var cfg Config
	decodeErr := json.NewDecoder(file).Decode(&cfg)
	if decodeErr != nil {
		return nil, fmt.Errorf("decode config file %q: %w", path, decodeErr)
	}

	if cfg.BotHTTPAddress == "" {
		cfg.BotHTTPAddress = ":8081"
	}
	if cfg.BotBaseURL == "" {
		cfg.BotBaseURL = "http://localhost:8081"
	}
	if cfg.ScrapperHTTPAddress == "" {
		cfg.ScrapperHTTPAddress = ":8080"
	}
	if cfg.ScrapperBaseURL == "" {
		cfg.ScrapperBaseURL = "http://localhost:8080"
	}
	if cfg.ScrapperGRPCAddress == "" {
		cfg.ScrapperGRPCAddress = ":8090"
	}
	if cfg.ScrapperGRPCTarget == "" {
		cfg.ScrapperGRPCTarget = "localhost:8090"
	}
	if cfg.TransportMode == "" {
		cfg.TransportMode = "http"
	}
	if cfg.SchedulerInterval == "" {
		cfg.SchedulerInterval = "30s"
	}
	if cfg.ExternalHTTPTimeout == "" {
		cfg.ExternalHTTPTimeout = "5s"
	}

	return &cfg, nil
}
