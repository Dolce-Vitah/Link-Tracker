package external

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/domain/linkchange"
)

const (
	githubHost         = "github.com"
	githubWWWHost      = "www.github.com"
	stackOverflowHost  = "stackoverflow.com"
	stackOverflowWHost = "www.stackoverflow.com"
)

type LastUpdatedClient interface {
	GetLastUpdated(ctx context.Context, rawURL string) (time.Time, error)
}

type HTTPClient struct {
	githubClient        *GitHubClient
	stackOverflowClient *StackOverflowClient
}

func NewHTTPClient(timeout time.Duration, githubAPIBase, stackExchangeAPIBase string) *HTTPClient {
	return &HTTPClient{
		githubClient:        NewGitHubClient(timeout, githubAPIBase),
		stackOverflowClient: NewStackOverflowClient(timeout, stackExchangeAPIBase),
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

func (c *HTTPClient) FetchChangesSince(ctx context.Context, rawURL string, since time.Time) ([]linkchange.Change, time.Time, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("parse url: %w", err)
	}
	switch {
	case isGitHubHost(u.Host):
		owner, repo, pathErr := parseGitHubRepoPath(u.Path)
		if pathErr != nil {
			return nil, time.Time{}, fmt.Errorf("invalid github repository url: %w", pathErr)
		}
		return c.githubClient.FetchRepoChanges(ctx, owner, repo, since)
	case isStackOverflowHost(u.Host):
		qid, qerr := parseStackOverflowQuestionPath(u.Path)
		if qerr != nil {
			return nil, time.Time{}, errors.New("invalid stackoverflow question url")
		}
		return c.stackOverflowClient.FetchQuestionChanges(ctx, qid, since)
	default:
		return nil, time.Time{}, fmt.Errorf("unsupported host: %s", u.Host)
	}
}
