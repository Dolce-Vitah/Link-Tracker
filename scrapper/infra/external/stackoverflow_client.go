package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/adapters/dto"
)

type StackOverflowClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewStackOverflowClient(timeout time.Duration, baseURL string) *StackOverflowClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.stackexchange.com/2.3"
	}
	return &StackOverflowClient{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(baseURL, "/"),
	}
}

func (c *StackOverflowClient) GetLastUpdated(ctx context.Context, rawURL string) (time.Time, error) {
	u, err := url.Parse(rawURL)
	if err != nil {

		return time.Time{}, fmt.Errorf("parse url: %w", err)
	}

	if !isStackOverflowHost(u.Host) {

		return time.Time{}, fmt.Errorf("unsupported host: %s", u.Host)
	}

	return c.getLastUpdatedFromURL(ctx, u)
}

func (c *StackOverflowClient) getLastUpdatedFromURL(ctx context.Context, u *url.URL) (time.Time, error) {
	questionID, err := parseStackOverflowQuestionPath(u.Path)
	if err != nil {

		return time.Time{}, errors.New("invalid stackoverflow question url")
	}

	reqURL := fmt.Sprintf("%s/questions/%d?site=stackoverflow", c.baseURL, questionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {

		return time.Time{}, fmt.Errorf("build stackoverflow request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {

		return time.Time{}, fmt.Errorf("do stackoverflow request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		return time.Time{}, fmt.Errorf("stackoverflow non-2xx status: %d", resp.StatusCode)
	}

	var payload dto.StackOverflowResponse

	decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
	if decodeErr != nil {

		return time.Time{}, fmt.Errorf("decode stackoverflow response: %w", decodeErr)
	}

	if len(payload.Items) == 0 || payload.Items[0].LastActivityDate == 0 {

		return time.Time{}, errors.New("stackoverflow response missing last_activity_date")
	}

	return time.Unix(payload.Items[0].LastActivityDate, 0).UTC(), nil
}
