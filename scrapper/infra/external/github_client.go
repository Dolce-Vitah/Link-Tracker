package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/adapters/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/domain/linkchange"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/textutil"
)

type GitHubClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewGitHubClient(timeout time.Duration, baseURL string) *GitHubClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.github.com"
	}
	return &GitHubClient{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(baseURL, "/"),
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

	reqURL := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)

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

func (c *GitHubClient) FetchRepoChanges(ctx context.Context, owner, repo string, since time.Time) ([]linkchange.Change, time.Time, error) {
	if since.IsZero() {
		return c.githubBaselineWatermark(ctx, owner, repo)
	}
	return c.githubDeltaChanges(ctx, owner, repo, since)
}

func (c *GitHubClient) githubBaselineWatermark(ctx context.Context, owner, repo string) ([]linkchange.Change, time.Time, error) {
	path := fmt.Sprintf("%s/repos/%s/%s/issues?state=all&per_page=1&sort=created&direction=desc",
		c.baseURL, owner, repo)
	items, err := c.fetchIssuesPath(ctx, path)
	if err != nil {
		return nil, time.Time{}, err
	}
	if len(items) == 0 {
		return nil, time.Now().UTC(), nil
	}
	return nil, items[0].CreatedAt.UTC(), nil
}

func (c *GitHubClient) githubDeltaChanges(ctx context.Context, owner, repo string, since time.Time) ([]linkchange.Change, time.Time, error) {
	sinceRFC := since.UTC().Format(time.RFC3339)
	page := 1
	var out []linkchange.Change
	wm := since.UTC()
	for {
		path := fmt.Sprintf("%s/repos/%s/%s/issues?state=all&since=%s&per_page=100&page=%d&sort=created&direction=asc",
			c.baseURL, owner, repo, url.QueryEscape(sinceRFC), page)
		items, err := c.fetchIssuesPath(ctx, path)
		if err != nil {
			return nil, time.Time{}, err
		}
		if len(items) == 0 {
			break
		}
		for _, it := range items {
			created := it.CreatedAt.UTC()
			if !created.After(since.UTC()) {
				continue
			}
			kind := linkchange.KindGitHubIssue
			if it.PullRequest != nil {
				kind = linkchange.KindGitHubPullRequest
			}
			author := strings.TrimSpace(it.User.Login)
			if author == "" {
				author = "unknown"
			}
			out = append(out, linkchange.Change{
				Kind:      kind,
				Title:     strings.TrimSpace(it.Title),
				Author:    author,
				CreatedAt: created,
				Preview:   textutil.TruncateRunes(strings.TrimSpace(it.Body), 200),
			})
			if created.After(wm) {
				wm = created
			}
		}
		if len(items) < 100 {
			break
		}
		page++
	}
	return out, wm, nil
}

func (c *GitHubClient) fetchIssuesPath(ctx context.Context, reqURL string) ([]dto.GitHubIssueItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build github issues request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do github issues request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github issues non-2xx status: %d: %s", resp.StatusCode, string(body))
	}
	var items []dto.GitHubIssueItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode github issues: %w", err)
	}
	return items, nil
}
