//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	kafkaio "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/linkdto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/infra/notifications"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/infra/telegram"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/botclient"
)

type noopTrackerClient struct{}

func (noopTrackerClient) RegisterChat(context.Context, int64) error {
	return nil
}
func (noopTrackerClient) DeleteChat(context.Context, int64) error {
	return nil
}
func (noopTrackerClient) AddLink(context.Context, int64, trackerapi.AddLinkRequest) (trackerapi.LinkResponse, error) {
	return trackerapi.LinkResponse{}, nil
}
func (noopTrackerClient) RemoveLink(context.Context, int64, trackerapi.RemoveLinkRequest) (trackerapi.LinkResponse, error) {
	return trackerapi.LinkResponse{}, nil
}
func (noopTrackerClient) ListLinks(context.Context, int64) (trackerapi.ListLinksResponse, error) {
	return trackerapi.ListLinksResponse{}, nil
}

var _ tracker.Client = noopTrackerClient{}

func TestIntegration_ScrapperKafkaBotToTelegram(t *testing.T) {
	t.Parallel()

	brokers := startSharedKafka(t)
	topic := fmt.Sprintf("link-updates-e2e-%d", time.Now().UnixNano())
	dlqTopic := fmt.Sprintf("link-updates-dlq-e2e-%d", time.Now().UnixNano())
	createKafkaTopic(t, brokers, topic, 3)
	createKafkaTopic(t, brokers, dlqTopic, 1)

	telegramMessages := make(chan string, 1)
	telegramStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok": true, "result": {"id": 1, "is_bot": true, "first_name": "TestBot", "username": "test_bot"}}`))
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			body, _ := io.ReadAll(r.Body)
			telegramMessages <- string(body)
			_, _ = w.Write([]byte(`{"ok": true, "result": {"message_id": 1, "date": 1700000000, "chat": {"id": 777, "type": "private"}, "text": "ok"}}`))
		default:
			_, _ = w.Write([]byte(`{"ok": true, "result": true}`))
		}
	}))
	t.Cleanup(telegramStub.Close)

	botInstance, err := telegram.NewBot("fake-token", telegramStub.URL+"/bot%s/%s", slog.Default(), noopTrackerClient{})
	require.NoError(t, err)

	consumer, err := notifications.NewKafkaConsumer(notifications.KafkaConfig{
		Brokers:          brokers,
		Topic:            topic,
		DLQTopic:         dlqTopic,
		ConsumerGroup:    fmt.Sprintf("bot-group-%d", time.Now().UnixNano()),
		ReaderMinBytes:   1,
		ReaderMaxBytes:   10e6,
		MaxRetryAttempts: 3,
	}, botInstance)
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go consumer.Run(ctx)

	producer, err := botclient.NewKafkaUpdatesClient(brokers, topic, 5*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	sendErr := producer.SendLinkUpdate(context.Background(), trackerapi.NewLinkUpdate(
		1,
		"https://github.com/acme/repo",
		[]int64{777},
		trackerapi.EventKindGitHubIssue,
		"New issue",
		"dev",
		time.Now().UTC(),
		"preview text",
	))
	require.NoError(t, sendErr)

	select {
	case got := <-telegramMessages:
		require.Contains(t, got, "New+issue")
		require.Contains(t, got, "github.com%2Facme%2Frepo")
	case <-time.After(20 * time.Second):
		t.Fatal("timeout waiting for telegram sendMessage call")
	}
}

type flakyHandler struct {
	failFor int32
	calls   atomic.Int32
}

func (h *flakyHandler) ProcessLinkUpdate(linkdto.LinkUpdate) error {
	call := h.calls.Add(1)
	if call <= h.failFor {
		return errors.New("temporary processing error")
	}
	return nil
}

func TestIntegration_KafkaConsumer_InvalidMessageToDLQ(t *testing.T) {
	t.Parallel()

	brokers := startSharedKafka(t)
	topic := fmt.Sprintf("link-updates-invalid-%d", time.Now().UnixNano())
	dlqTopic := fmt.Sprintf("link-updates-dlq-invalid-%d", time.Now().UnixNano())
	createKafkaTopic(t, brokers, topic, 1)
	createKafkaTopic(t, brokers, dlqTopic, 1)

	handler := &flakyHandler{}
	consumer, err := notifications.NewKafkaConsumer(notifications.KafkaConfig{
		Brokers:          brokers,
		Topic:            topic,
		DLQTopic:         dlqTopic,
		ConsumerGroup:    fmt.Sprintf("bot-invalid-%d", time.Now().UnixNano()),
		ReaderMinBytes:   1,
		ReaderMaxBytes:   10e6,
		MaxRetryAttempts: 2,
	}, handler)
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go consumer.Run(ctx)

	writer := &kafkaio.Writer{
		Addr:  kafkaio.TCP(brokers...),
		Topic: topic,
	}
	t.Cleanup(func() { _ = writer.Close() })
	require.NoError(t, writer.WriteMessages(context.Background(), kafkaio.Message{Value: []byte("{invalid-json")}))

	dlq := readKafkaMessage(t, brokers, dlqTopic, 20*time.Second)
	var msg notifications.DLQMessage
	require.NoError(t, json.Unmarshal(dlq, &msg))
	require.Contains(t, msg.Reason, "deserialization_error")
	require.EqualValues(t, 0, handler.calls.Load())
}

func TestIntegration_KafkaConsumer_ProcessingRetryThenDLQ(t *testing.T) {
	t.Parallel()

	brokers := startSharedKafka(t)
	topic := fmt.Sprintf("link-updates-retry-%d", time.Now().UnixNano())
	dlqTopic := fmt.Sprintf("link-updates-dlq-retry-%d", time.Now().UnixNano())
	createKafkaTopic(t, brokers, topic, 1)
	createKafkaTopic(t, brokers, dlqTopic, 1)

	handler := &flakyHandler{failFor: 3}
	consumer, err := notifications.NewKafkaConsumer(notifications.KafkaConfig{
		Brokers:          brokers,
		Topic:            topic,
		DLQTopic:         dlqTopic,
		ConsumerGroup:    fmt.Sprintf("bot-retry-%d", time.Now().UnixNano()),
		ReaderMinBytes:   1,
		ReaderMaxBytes:   10e6,
		MaxRetryAttempts: 3,
	}, handler)
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go consumer.Run(ctx)

	producer, err := botclient.NewKafkaUpdatesClient(brokers, topic, 5*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	sendErr := producer.SendLinkUpdate(context.Background(), trackerapi.NewLinkUpdate(
		2,
		"https://github.com/acme/repo2",
		[]int64{123},
		trackerapi.EventKindGitHubIssue,
		"Retry me",
		"dev",
		time.Now().UTC(),
		"preview",
	))
	require.NoError(t, sendErr)

	dlq := readKafkaMessage(t, brokers, dlqTopic, 20*time.Second)
	var msg notifications.DLQMessage
	require.NoError(t, json.Unmarshal(dlq, &msg))
	require.Contains(t, msg.Reason, "processing_error_after_retries")
	require.EqualValues(t, 3, handler.calls.Load())
}
