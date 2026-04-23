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
	TransportModeHTTP         = "http"
	TransportModeGRPC         = "grpc"
	NotificationModeHTTP      = "http"
	NotificationModeKafka     = "kafka"
	defaultKafkaTopic         = "link-updates"
	defaultKafkaDLQTopic      = "link-updates-dlq"
	defaultKafkaConsumerGroup = "bot-notifications"
)

type Config struct {
	TelegramToken         string   `json:"telegram_token"`
	TelegramAPIURL        string   `json:"telegram_api_url,omitempty"`
	BotHTTPAddress        string   `json:"bot_http_address"`
	ScrapperBaseURL       string   `json:"scrapper_base_url"`
	ScrapperGRPCTarget    string   `json:"scrapper_grpc_target"`
	TransportMode         string   `json:"transport_mode"`
	NotificationMode      string   `json:"notification_mode"`
	KafkaBrokers          []string `json:"kafka_brokers"`
	KafkaTopic            string   `json:"kafka_topic"`
	KafkaDLQTopic         string   `json:"kafka_dlq_topic"`
	KafkaConsumerGroup    string   `json:"kafka_consumer_group"`
	KafkaReaderMinBytes   int      `json:"kafka_reader_min_bytes"`
	KafkaReaderMaxBytes   int      `json:"kafka_reader_max_bytes"`
	KafkaMaxRetryAttempts int      `json:"kafka_max_retry_attempts"`
	ExternalHTTPTimeout   string   `json:"external_http_timeout"`
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
	if cfg.NotificationMode == "" {
		cfg.NotificationMode = NotificationModeKafka
	}
	if len(cfg.KafkaBrokers) == 0 {
		cfg.KafkaBrokers = []string{"localhost:9092", "localhost:9093", "localhost:9094"}
	}
	if cfg.KafkaTopic == "" {
		cfg.KafkaTopic = defaultKafkaTopic
	}
	if cfg.KafkaDLQTopic == "" {
		cfg.KafkaDLQTopic = defaultKafkaDLQTopic
	}
	if cfg.KafkaConsumerGroup == "" {
		cfg.KafkaConsumerGroup = defaultKafkaConsumerGroup
	}
	if cfg.KafkaReaderMinBytes <= 0 {
		cfg.KafkaReaderMinBytes = 10000
	}
	if cfg.KafkaReaderMaxBytes <= 0 {
		cfg.KafkaReaderMaxBytes = 10000000
	}
	if cfg.KafkaMaxRetryAttempts <= 0 {
		cfg.KafkaMaxRetryAttempts = 3
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
	default:
		return fmt.Errorf("invalid transport_mode %q: expected %q or %q", cfg.TransportMode, TransportModeHTTP, TransportModeGRPC)
	}

	notificationMode := strings.ToLower(strings.TrimSpace(cfg.NotificationMode))
	switch notificationMode {
	case NotificationModeHTTP, NotificationModeKafka:
		cfg.NotificationMode = notificationMode
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
		if strings.TrimSpace(cfg.KafkaDLQTopic) == "" {
			return errors.New("kafka_dlq_topic must not be empty when notification_mode is kafka")
		}
		if strings.TrimSpace(cfg.KafkaConsumerGroup) == "" {
			return errors.New("kafka_consumer_group must not be empty when notification_mode is kafka")
		}
	}

	return nil
}
