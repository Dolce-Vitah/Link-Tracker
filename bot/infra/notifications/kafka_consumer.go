package notifications

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/linkdto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/avrokafka"
)

type LinkUpdateHandler interface {
	ProcessLinkUpdate(update linkdto.LinkUpdate) error
}

type KafkaConfig struct {
	SchemaRegistryURL string
	Brokers           []string
	Topic             string
	DLQTopic          string
	ConsumerGroup     string
	ReaderMinBytes    int
	ReaderMaxBytes    int
	MaxRetryAttempts  int
}

type DLQMessage struct {
	SourceTopic string    `json:"source_topic"`
	Reason      string    `json:"reason"`
	Payload     string    `json:"payload_base64"`
	CreatedAt   time.Time `json:"created_at"`
}

type KafkaConsumer struct {
	reader      *kafka.Reader
	dlqWriter   *kafka.Writer
	handler     LinkUpdateHandler
	sourceTopic string
	maxRetry    int
	codec       *avrokafka.LinkUpdateCodec
}

func NewKafkaConsumer(cfg KafkaConfig, handler LinkUpdateHandler) (*KafkaConsumer, error) {
	if handler == nil {
		return nil, fmt.Errorf("link update handler is required")
	}
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka topic is required")
	}
	if cfg.DLQTopic == "" {
		return nil, fmt.Errorf("kafka dlq topic is required")
	}
	if cfg.ConsumerGroup == "" {
		return nil, fmt.Errorf("kafka consumer group is required")
	}
	if cfg.ReaderMinBytes <= 0 {
		cfg.ReaderMinBytes = 10000
	}
	if cfg.ReaderMaxBytes <= 0 {
		cfg.ReaderMaxBytes = 10000000
	}
	if cfg.MaxRetryAttempts <= 0 {
		cfg.MaxRetryAttempts = 3
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		GroupID:     cfg.ConsumerGroup,
		Topic:       cfg.Topic,
		MinBytes:    cfg.ReaderMinBytes,
		MaxBytes:    cfg.ReaderMaxBytes,
		StartOffset: kafka.FirstOffset,
	})

	codec, codecErr := avrokafka.NewLinkUpdateCodec(cfg.SchemaRegistryURL, cfg.Topic+"-value")
	if codecErr != nil {
		return nil, fmt.Errorf("init avro codec: %w", codecErr)
	}
	dlqWriter := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.DLQTopic,
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		Balancer:     &kafka.LeastBytes{},
	}

	return &KafkaConsumer{
		reader:      reader,
		dlqWriter:   dlqWriter,
		handler:     handler,
		sourceTopic: cfg.Topic,
		maxRetry:    cfg.MaxRetryAttempts,
		codec:       codec,
	}, nil
}

func (c *KafkaConsumer) Close() error {
	var closeErr error
	if err := c.reader.Close(); err != nil {
		closeErr = err
	}
	if err := c.dlqWriter.Close(); err != nil && closeErr == nil {
		closeErr = err
	}
	return closeErr
}

func (c *KafkaConsumer) Run(ctx context.Context) {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Kafka fetch message failed", slog.String("error", err.Error()))
			continue
		}

		if handleErr := c.handleMessage(ctx, msg); handleErr != nil {
			slog.Error("Kafka message handling failed", slog.String("error", handleErr.Error()))
		}

		if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
			slog.Error("Kafka commit failed", slog.String("error", commitErr.Error()))
		}
	}
}

func (c *KafkaConsumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	const retryBackoff = 200 * time.Millisecond

	var update linkdto.LinkUpdate
	decoded, err := c.codec.Decode(msg.Value)
	if err != nil {
		return c.sendToDLQ(ctx, msg.Value, fmt.Sprintf("deserialization_error: %v", err))
	}
	update = decoded
	if err := update.Validate(); err != nil {
		return c.sendToDLQ(ctx, msg.Value, fmt.Sprintf("validation_error: %v", err))
	}

	var processingErr error
	for attempt := 1; attempt <= c.maxRetry; attempt++ {
		processingErr = c.handler.ProcessLinkUpdate(update)
		if processingErr == nil {
			return nil
		}
		if attempt < c.maxRetry {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryBackoff):
			}
		}
	}

	return c.sendToDLQ(ctx, msg.Value, fmt.Sprintf("processing_error_after_retries: %v", processingErr))
}

func (c *KafkaConsumer) sendToDLQ(ctx context.Context, payload []byte, reason string) error {
	dlqBody, err := json.Marshal(DLQMessage{
		SourceTopic: c.sourceTopic,
		Reason:      reason,
		Payload:     base64.StdEncoding.EncodeToString(payload),
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("marshal dlq message: %w", err)
	}

	if writeErr := c.dlqWriter.WriteMessages(ctx, kafka.Message{Value: dlqBody}); writeErr != nil {
		return fmt.Errorf("write dlq message: %w", writeErr)
	}

	return nil
}
