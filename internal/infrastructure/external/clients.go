package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type LastUpdatedClient interface {
	GetLastUpdated(ctx context.Context, rawURL string) (time.Time, error)
}

type HTTPClient struct {
	httpClient    *http.Client
	githubBaseURL string
	stackBaseURL  string
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		httpClient:    &http.Client{Timeout: timeout},
		githubBaseURL: "https://api.github.com",
		stackBaseURL:  "https://api.stackexchange.com/2.3",
	}
}

func (c *HTTPClient) GetLastUpdated(ctx context.Context, rawURL string) (time.Time, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse url: %w", err)
	}

	switch {
	case strings.Contains(u.Host, "github.com"):
		return c.githubLastUpdated(ctx, u)
	case strings.Contains(u.Host, "stackoverflow.com"):
		return c.stackOverflowLastUpdated(ctx, u)
	default:
		return time.Time{}, fmt.Errorf("unsupported host: %s", u.Host)
	}
}

type githubRepoResponse struct {
	UpdatedAt time.Time `json:"updated_at"`
}

func (c *HTTPClient) githubLastUpdated(ctx context.Context, u *url.URL) (time.Time, error) {
	const requiredGitHubPathParts = 2

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < requiredGitHubPathParts {
		return time.Time{}, errors.New("invalid github repository url")
	}
	reqURL := fmt.Sprintf("%s/repos/%s/%s", strings.TrimRight(c.githubBaseURL, "/"), parts[0], parts[1])
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("build github request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return time.Time{}, fmt.Errorf("do github request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return time.Time{}, fmt.Errorf("github non-2xx status: %d", resp.StatusCode)
	}

	var payload githubRepoResponse
	decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
	if decodeErr != nil {
		return time.Time{}, fmt.Errorf("decode github response: %w", decodeErr)
	}
	if payload.UpdatedAt.IsZero() {
		return time.Time{}, errors.New("github response missing updated_at")
	}
	return payload.UpdatedAt, nil
}

type stackOverflowResponse struct {
	Items []struct {
		LastActivityDate int64 `json:"last_activity_date"`
	} `json:"items"`
}

func (c *HTTPClient) stackOverflowLastUpdated(ctx context.Context, u *url.URL) (time.Time, error) {
	const requiredStackPathParts = 2

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < requiredStackPathParts || parts[0] != "questions" {
		return time.Time{}, errors.New("invalid stackoverflow question url")
	}
	questionID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse stackoverflow question id: %w", err)
	}

	reqURL := fmt.Sprintf("%s/questions/%d?site=stackoverflow", strings.TrimRight(c.stackBaseURL, "/"), questionID)
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

	var payload stackOverflowResponse
	decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
	if decodeErr != nil {
		return time.Time{}, fmt.Errorf("decode stackoverflow response: %w", decodeErr)
	}
	if len(payload.Items) == 0 || payload.Items[0].LastActivityDate == 0 {
		return time.Time{}, errors.New("stackoverflow response missing last_activity_date")
	}
	return time.Unix(payload.Items[0].LastActivityDate, 0).UTC(), nil
}
