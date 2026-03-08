package botclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type UpdatesSender interface {
	SendUpdate(ctx context.Context, update api.LinkUpdate) error
}

type HTTPUpdatesClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPUpdatesClient(baseURL string, timeout time.Duration) *HTTPUpdatesClient {
	return &HTTPUpdatesClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *HTTPUpdatesClient) SendUpdate(ctx context.Context, update api.LinkUpdate) error {
	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal link update: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/updates", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build update request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send update request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("bot update endpoint returned status %d", resp.StatusCode)
	}
	return nil
}
