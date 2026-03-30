package botclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
)

var (
	ErrBadRequest      = errors.New("bad request")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrTooManyRequests = errors.New("too many requests")
	ErrClient          = errors.New("client error")
	ErrInternal        = errors.New("internal error")
)

type UpdatesSender interface {
	SendUpdate(ctx context.Context, update trackerapi.LinkUpdate) error
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

func (c *HTTPUpdatesClient) SendUpdate(ctx context.Context, update trackerapi.LinkUpdate) error {
	if err := update.Validate(); err != nil {
		return fmt.Errorf("validate link update: %w", err)
	}

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
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapStatusError(resp.StatusCode)
	}
	return nil
}

func mapStatusError(status int) error {
	const serverErrorMinStatus = 500

	switch status {
	case http.StatusBadRequest:
		return ErrBadRequest
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusConflict:
		return ErrConflict
	case http.StatusTooManyRequests:
		return ErrTooManyRequests
	default:
		if status >= 400 && status < 500 {
			return ErrClient
		}
		if status >= serverErrorMinStatus {
			return ErrInternal
		}
		return fmt.Errorf("unexpected status code: %d", status)
	}
}
