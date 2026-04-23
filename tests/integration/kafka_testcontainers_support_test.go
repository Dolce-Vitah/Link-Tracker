//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	kafkaio "github.com/segmentio/kafka-go"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

var (
	sharedKafkaOnce      sync.Once
	sharedKafkaContainer *tckafka.KafkaContainer
	sharedKafkaBrokers   []string
	sharedKafkaErr       error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedKafkaContainer != nil {
		_ = sharedKafkaContainer.Terminate(context.Background())
	}
	os.Exit(code)
}

func startSharedKafka(t *testing.T) []string {
	t.Helper()
	requireTestcontainers(t)

	sharedKafkaOnce.Do(func() {
		ctx := context.Background()
		container, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.7.1")
		if err != nil {
			sharedKafkaErr = err
			return
		}
		brokers, brokersErr := container.Brokers(ctx)
		if brokersErr != nil {
			sharedKafkaErr = brokersErr
			_ = container.Terminate(ctx)
			return
		}
		sharedKafkaContainer = container
		sharedKafkaBrokers = brokers
	})

	if sharedKafkaErr != nil {
		t.Skipf("kafka testcontainer unavailable: %v", sharedKafkaErr)
	}

	return sharedKafkaBrokers
}

func createKafkaTopic(t *testing.T, brokers []string, topic string, partitions int) {
	t.Helper()
	if len(brokers) == 0 {
		t.Fatalf("kafka brokers are empty")
	}

	conn, err := kafkaio.Dial("tcp", brokers[0])
	if err != nil {
		t.Fatalf("dial kafka bootstrap: %v", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		t.Fatalf("get kafka controller: %v", err)
	}

	controllerConn, err := kafkaio.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		t.Fatalf("dial kafka controller: %v", err)
	}
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafkaio.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: 1,
	})
	if err != nil {
		t.Fatalf("create kafka topic %q: %v", topic, err)
	}
}

func readKafkaMessage(t *testing.T, brokers []string, topic string, timeout time.Duration) []byte {
	t.Helper()
	reader := kafkaio.NewReader(kafkaio.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		StartOffset: kafkaio.FirstOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
	})
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("read kafka message from %s: %v", topic, err)
	}
	return msg.Value
}
