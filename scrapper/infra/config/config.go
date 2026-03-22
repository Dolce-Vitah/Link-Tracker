package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

type Config struct {
	BotBaseURL          string `json:"bot_base_url"`
	ScrapperHTTPAddress string `json:"scrapper_http_address"`
	ScrapperGRPCAddress string `json:"scrapper_grpc_address"`
	SchedulerInterval   string `json:"scheduler_interval"`
	ExternalHTTPTimeout string `json:"external_http_timeout"`
	AccessType          string `json:"access_type"`
	DBDsn               string `json:"db_dsn"`
	DBMaxOpenConns      int    `json:"db_max_open_conns"`
	DBMaxIdleConns      int    `json:"db_max_idle_conns"`
	DBConnMaxLifetime   string `json:"db_conn_max_lifetime"`
	AutoMigrate         bool   `json:"auto_migrate"`
}

func Load(path string) (*Config, error) {
	var cfg Config
	if err := decode(path, "config.json", &cfg); err != nil {
		return nil, err
	}
	applyDefaults(&cfg)
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
	if cfg.BotBaseURL == "" {
		cfg.BotBaseURL = "http://localhost:8081"
	}
	if cfg.ScrapperHTTPAddress == "" {
		cfg.ScrapperHTTPAddress = ":8080"
	}
	if cfg.ScrapperGRPCAddress == "" {
		cfg.ScrapperGRPCAddress = ":8090"
	}
	if cfg.SchedulerInterval == "" {
		cfg.SchedulerInterval = "30s"
	}
	if cfg.ExternalHTTPTimeout == "" {
		cfg.ExternalHTTPTimeout = "5s"
	}
	if cfg.AccessType == "" {
		cfg.AccessType = "SQL"
	}
	if cfg.DBDsn == "" {
		cfg.DBDsn = "postgres://postgres:postgres@localhost:5432/link_tracker?sslmode=disable"
	}
	if cfg.DBMaxOpenConns == 0 {
		cfg.DBMaxOpenConns = 10
	}
	if cfg.DBMaxIdleConns == 0 {
		cfg.DBMaxIdleConns = 5
	}
	if cfg.DBConnMaxLifetime == "" {
		cfg.DBConnMaxLifetime = "30m"
	}
}
