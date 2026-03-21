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

type GitHubClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewGitHubClient(timeout time.Duration) *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    "https://api.github.com",
	}
}

func (c *GitHubClient) GetLastUpdated(ctx context.Context, rawURL string) (time.Time, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse url: %w", err)
	}
	if !isGitHubHost(u.Host) {
		return time.Time{}, fmt.Errorf("unsupported host: %s", u.Host)
	}
	return c.getLastUpdatedFromURL(ctx, u)
}

func (c *GitHubClient) getLastUpdatedFromURL(ctx context.Context, u *url.URL) (time.Time, error) {
	owner, repo, err := parseGitHubRepoPath(u.Path)
	if err != nil {
		return time.Time{}, errors.New("invalid github repository url")
	}
	reqURL := fmt.Sprintf("%s/repos/%s/%s", strings.TrimRight(c.baseURL, "/"), owner, repo)
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

	var payload dto.GitHubRepoResponse
	decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
	if decodeErr != nil {
		return time.Time{}, fmt.Errorf("decode github response: %w", decodeErr)
	}
	if payload.UpdatedAt.IsZero() {
		return time.Time{}, errors.New("github response missing updated_at")
	}
	return payload.UpdatedAt, nil
}
