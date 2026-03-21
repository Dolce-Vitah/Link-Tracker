package external

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type LastUpdatedClient interface {
	GetLastUpdated(ctx context.Context, rawURL string) (time.Time, error)
}

type HTTPClient struct {
	githubClient        *GitHubClient
	stackOverflowClient *StackOverflowClient
}

const (
	githubHost         = "github.com"
	githubWWWHost      = "www.github.com"
	stackOverflowHost  = "stackoverflow.com"
	stackOverflowWHost = "www.stackoverflow.com"
)

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		githubClient:        NewGitHubClient(timeout),
		stackOverflowClient: NewStackOverflowClient(timeout),
	}
}

func (c *HTTPClient) GetLastUpdated(ctx context.Context, rawURL string) (time.Time, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse url: %w", err)
	}

	switch {
	case isGitHubHost(u.Host):
		return c.githubClient.getLastUpdatedFromURL(ctx, u)
	case isStackOverflowHost(u.Host):
		return c.stackOverflowClient.getLastUpdatedFromURL(ctx, u)
	default:
		return time.Time{}, fmt.Errorf("unsupported host: %s", u.Host)
	}
}
