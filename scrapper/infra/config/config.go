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
	NotificationModeHTTP  = "http"
	NotificationModeKafka = "kafka"
	defaultKafkaTopic     = "link-updates"
)

type Config struct {
	BotBaseURL              string   `json:"bot_base_url"`
	ScrapperHTTPAddress     string   `json:"scrapper_http_address"`
	ScrapperGRPCAddress     string   `json:"scrapper_grpc_address"`
	NotificationMode        string   `json:"notification_mode"`
	KafkaBrokers            []string `json:"kafka_brokers"`
	KafkaTopic              string   `json:"kafka_topic"`
	KafkaWriteTimeout       string   `json:"kafka_write_timeout"`
	SchedulerInterval       string   `json:"scheduler_interval"`
	SchedulerDBPageSize     int      `json:"scheduler_db_page_size"`
	SchedulerSuperBatchSize int      `json:"scheduler_super_batch_size"`
	SchedulerWorkerCount    int      `json:"scheduler_worker_count"`
	ExternalHTTPTimeout     string   `json:"external_http_timeout"`
	GitHubAPIBaseURL        string   `json:"github_api_base_url"`
	StackExchangeAPIBaseURL string   `json:"stackexchange_api_base_url"`
	AccessType              string   `json:"access_type"`
	DBDsn                   string   `json:"db_dsn"`
	DBMaxOpenConns          int      `json:"db_max_open_conns"`
	DBMaxIdleConns          int      `json:"db_max_idle_conns"`
	DBConnMaxLifetime       string   `json:"db_conn_max_lifetime"`
	AutoMigrate             bool     `json:"auto_migrate"`
}

func Load(path string) (*Config, error) {
	var cfg Config
	if err := decode(path, "config.json", &cfg); err != nil {
		return nil, err
	}
	applyDefaults(&cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (cfg *Config) Validate() error {
	mode := strings.ToLower(strings.TrimSpace(cfg.NotificationMode))
	switch mode {
	case NotificationModeHTTP, NotificationModeKafka:
		cfg.NotificationMode = mode
	default:
		return fmt.Errorf("invalid notification_mode %q: expected %q or %q", cfg.NotificationMode, NotificationModeHTTP, NotificationModeKafka)
	}

	if cfg.NotificationMode == NotificationModeKafka {
		if len(cfg.KafkaBrokers) == 0 {
			return errors.New("kafka_brokers must not be empty when notification_mode is kafka")
		}
		if strings.TrimSpace(cfg.KafkaTopic) == "" {
			return errors.New("kafka_topic must not be empty when notification_mode is kafka")
		}
	}

	if cfg.SchedulerDBPageSize < 50 || cfg.SchedulerDBPageSize > 500 {
		return fmt.Errorf("scheduler_db_page_size must be between 50 and 500, got %d", cfg.SchedulerDBPageSize)
	}
	if cfg.SchedulerSuperBatchSize < 1 {
		return fmt.Errorf("scheduler_super_batch_size must be >= 1, got %d", cfg.SchedulerSuperBatchSize)
	}
	if cfg.SchedulerWorkerCount < 1 {
		return fmt.Errorf("scheduler_worker_count must be >= 1, got %d", cfg.SchedulerWorkerCount)
	}
	return nil
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
		cfg.BotBaseURL = "http://127.0.0.1:8081"
	}
	if cfg.ScrapperHTTPAddress == "" {
		cfg.ScrapperHTTPAddress = ":8080"
	}
	if cfg.ScrapperGRPCAddress == "" {
		cfg.ScrapperGRPCAddress = ":8090"
	}
	if cfg.NotificationMode == "" {
		cfg.NotificationMode = NotificationModeKafka
	}
	if len(cfg.KafkaBrokers) == 0 {
		cfg.KafkaBrokers = []string{"localhost:9092", "localhost:9093", "localhost:9094"}
	}
	if cfg.KafkaTopic == "" {
		cfg.KafkaTopic = defaultKafkaTopic
	}
	if cfg.KafkaWriteTimeout == "" {
		cfg.KafkaWriteTimeout = "5s"
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
	if cfg.SchedulerDBPageSize == 0 {
		cfg.SchedulerDBPageSize = 100
	}
	if cfg.SchedulerSuperBatchSize == 0 {
		cfg.SchedulerSuperBatchSize = 1000
	}
	if cfg.SchedulerWorkerCount == 0 {
		cfg.SchedulerWorkerCount = 4
	}
}
