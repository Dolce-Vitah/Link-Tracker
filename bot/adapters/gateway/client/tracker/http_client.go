package trackerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/domain/tracker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *HTTPClient) RegisterChat(ctx context.Context, chatID int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/tg-chat/%d", c.baseURL, chatID), nil)
	if err != nil {

		return fmt.Errorf("build register chat request: %w", err)
	}

	return c.handleNoBody(req)
}

func (c *HTTPClient) DeleteChat(ctx context.Context, chatID int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%s/tg-chat/%d", c.baseURL, chatID), nil)
	if err != nil {

		return fmt.Errorf("build delete chat request: %w", err)
	}

	return c.handleNoBody(req)
}

func (c *HTTPClient) AddLink(ctx context.Context, chatID int64, request trackerapi.AddLinkRequest) (trackerapi.LinkResponse, error) {
	req, err := c.newJSONRequest(ctx, http.MethodPost, "/links", request)
	if err != nil {

		return trackerapi.LinkResponse{}, err
	}
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	return c.handleLinkResponse(req)
}

func (c *HTTPClient) RemoveLink(ctx context.Context, chatID int64, request trackerapi.RemoveLinkRequest) (trackerapi.LinkResponse, error) {
	req, err := c.newJSONRequest(ctx, http.MethodDelete, "/links", request)
	if err != nil {

		return trackerapi.LinkResponse{}, err
	}
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	return c.handleLinkResponse(req)
}

func (c *HTTPClient) ListLinks(ctx context.Context, chatID int64) (trackerapi.ListLinksResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/links", nil)
	if err != nil {

		return trackerapi.ListLinksResponse{}, fmt.Errorf("build list links request: %w", err)
	}
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := c.client.Do(req)
	if err != nil {

		return trackerapi.ListLinksResponse{}, fmt.Errorf("do list links request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	mapErr := mapHTTPError(resp)
	if mapErr != nil {

		return trackerapi.ListLinksResponse{}, mapErr
	}

	var out trackerapi.ListLinksResponse
	decodeErr := json.NewDecoder(resp.Body).Decode(&out)
	if decodeErr != nil {

		return trackerapi.ListLinksResponse{}, fmt.Errorf("decode list links response: %w", decodeErr)
	}

	return out, nil
}

func (c *HTTPClient) newJSONRequest(ctx context.Context, method string, path string, payload any) (*http.Request, error) {
	body, err := json.Marshal(payload)
	if err != nil {

		return nil, fmt.Errorf("marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {

		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *HTTPClient) handleNoBody(req *http.Request) error {
	resp, err := c.client.Do(req)
	if err != nil {

		return fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return mapHTTPError(resp)
}

func (c *HTTPClient) handleLinkResponse(req *http.Request) (trackerapi.LinkResponse, error) {
	resp, err := c.client.Do(req)
	if err != nil {

		return trackerapi.LinkResponse{}, fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	mapErr := mapHTTPError(resp)
	if mapErr != nil {

		return trackerapi.LinkResponse{}, mapErr
	}
	var out trackerapi.LinkResponse
	decodeErr := json.NewDecoder(resp.Body).Decode(&out)
	if decodeErr != nil {

		return trackerapi.LinkResponse{}, fmt.Errorf("decode link response: %w", decodeErr)
	}

	return out, nil
}

func mapHTTPError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {

		return nil
	}
	var apiErr trackerapi.ErrorResponse
	_ = json.NewDecoder(resp.Body).Decode(&apiErr)
	switch resp.StatusCode {
	case http.StatusBadRequest:

		return fmt.Errorf("%w: %s", tracker.ErrBadRequest, apiErr.Description)
	case http.StatusNotFound:

		return fmt.Errorf("%w: %s", tracker.ErrNotFound, apiErr.Description)
	case http.StatusConflict:

		return fmt.Errorf("%w: %s", tracker.ErrAlreadyExists, apiErr.Description)
	default:
		if apiErr.Description != "" {

			return errors.New(apiErr.Description)
		}

		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
