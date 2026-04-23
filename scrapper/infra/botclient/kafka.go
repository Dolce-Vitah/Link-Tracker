package botclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
)

type KafkaUpdatesClient struct {
	writer *kafka.Writer
}

func NewKafkaUpdatesClient(brokers []string, topic string, writeTimeout time.Duration) (*KafkaUpdatesClient, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	if topic == "" {
		return nil, fmt.Errorf("kafka topic is required")
	}
	if writeTimeout <= 0 {
		writeTimeout = 5 * time.Second
	}

	return &KafkaUpdatesClient{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
			Async:        false,
			WriteTimeout: writeTimeout,
		},
	}, nil
}

func (c *KafkaUpdatesClient) SendLinkUpdate(ctx context.Context, update trackerapi.LinkUpdate) error {
	if err := update.Validate(); err != nil {
		return fmt.Errorf("validate link update: %w", err)
	}

	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal link update: %w", err)
	}

	if writeErr := c.writer.WriteMessages(ctx, kafka.Message{Value: body}); writeErr != nil {
		return fmt.Errorf("write kafka message: %w", writeErr)
	}

	return nil
}

func (c *KafkaUpdatesClient) SendProcessingFailureReport(_ context.Context, _ trackerapi.ProcessingFailureReport) error {
	return nil
}

func (c *KafkaUpdatesClient) Close() error {
	return c.writer.Close()
}
