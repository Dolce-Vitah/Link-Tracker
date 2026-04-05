package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/adapters/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/domain/linkchange"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/textutil"
)

func (c *StackOverflowClient) FetchQuestionChanges(ctx context.Context, questionID int64, since time.Time) ([]linkchange.Change, time.Time, error) {
	title, lastAct, err := c.fetchQuestionMeta(ctx, questionID)
	if err != nil {
		return nil, time.Time{}, err
	}
	lastActTime := time.Unix(lastAct, 0).UTC()
	if since.IsZero() {
		return nil, lastActTime, nil
	}
	sinceUnix := since.UTC().Unix()
	timeline, err := c.fetchTimeline(ctx, questionID)
	if err != nil {
		return nil, time.Time{}, err
	}
	var candidates []dto.StackTimelineItem
	for _, ev := range timeline {
		if ev.CreationDate <= sinceUnix {
			continue
		}
		switch ev.TimelineType {
		case "answer", "comment":
			candidates = append(candidates, ev)
		default:
			continue
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].CreationDate < candidates[j].CreationDate
	})
	userIDs := make(map[int64]struct{})
	answerIDs := make(map[int64]struct{})
	for _, ev := range candidates {
		if ev.UserID != 0 {
			userIDs[ev.UserID] = struct{}{}
		}
		if ev.TimelineType == "answer" && ev.PostID != 0 {
			answerIDs[ev.PostID] = struct{}{}
		}
	}
	users, err := c.fetchUsersMap(ctx, userIDs)
	if err != nil {
		return nil, time.Time{}, err
	}
	answers, err := c.fetchAnswersMap(ctx, answerIDs)
	if err != nil {
		return nil, time.Time{}, err
	}
	wm := since.UTC()
	var out []linkchange.Change
	for _, ev := range candidates {
		created := time.Unix(ev.CreationDate, 0).UTC()
		author := userName(users, ev.UserID)
		var kind string
		var preview string
		switch ev.TimelineType {
		case "answer":
			kind = linkchange.KindStackAnswer
			preview = textutil.TruncateRunes(strings.TrimSpace(stripHTML(answers[ev.PostID])), 200)
		case "comment":
			kind = linkchange.KindStackComment
			preview = textutil.TruncateRunes(strings.TrimSpace(ev.Detail), 200)
		}
		out = append(out, linkchange.Change{
			Kind:      kind,
			Title:     title,
			Author:    author,
			CreatedAt: created,
			Preview:   preview,
		})
		if created.After(wm) {
			wm = created
		}
	}
	if len(out) == 0 {
		return nil, since.UTC(), nil
	}
	return out, wm, nil
}

func userName(users map[int64]string, id int64) string {
	if n, ok := users[id]; ok && strings.TrimSpace(n) != "" {
		return strings.TrimSpace(n)
	}
	if id == 0 {
		return "unknown"
	}
	return "user:" + strconv.FormatInt(id, 10)
}

func stripHTML(s string) string {
	out := s
	for {
		i := strings.Index(out, "<")
		if i < 0 {
			break
		}
		j := strings.Index(out[i:], ">")
		if j < 0 {
			break
		}
		out = out[:i] + " " + out[i+j+1:]
	}
	return strings.Join(strings.Fields(out), " ")
}

func (c *StackOverflowClient) fetchQuestionMeta(ctx context.Context, questionID int64) (title string, lastActivityUnix int64, err error) {
	reqURL := fmt.Sprintf("%s/questions/%d?site=stackoverflow", c.baseURL, questionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("build stackoverflow question request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("do stackoverflow question request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("stackoverflow question non-2xx: %d: %s", resp.StatusCode, string(b))
	}
	var payload dto.StackOverflowResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", 0, fmt.Errorf("decode stackoverflow question: %w", err)
	}
	if len(payload.Items) == 0 || payload.Items[0].LastActivityDate == 0 {
		return "", 0, errors.New("stackoverflow response missing question data")
	}
	return strings.TrimSpace(payload.Items[0].Title), payload.Items[0].LastActivityDate, nil
}

func (c *StackOverflowClient) fetchTimeline(ctx context.Context, questionID int64) ([]dto.StackTimelineItem, error) {
	reqURL := fmt.Sprintf("%s/questions/%d/timeline?site=stackoverflow", c.baseURL, questionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build stackoverflow timeline request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do stackoverflow timeline request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stackoverflow timeline non-2xx: %d: %s", resp.StatusCode, string(b))
	}
	var payload dto.StackTimelineResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode stackoverflow timeline: %w", err)
	}
	return payload.Items, nil
}

func (c *StackOverflowClient) fetchUsersMap(ctx context.Context, ids map[int64]struct{}) (map[int64]string, error) {
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}
	parts := make([]string, 0, len(ids))
	for id := range ids {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	sort.Strings(parts)
	reqURL := fmt.Sprintf("%s/users/%s?site=stackoverflow", c.baseURL, strings.Join(parts, ";"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build stackoverflow users request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do stackoverflow users request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stackoverflow users non-2xx: %d: %s", resp.StatusCode, string(b))
	}
	var payload dto.StackUsersResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode stackoverflow users: %w", err)
	}
	out := make(map[int64]string, len(payload.Items))
	for _, u := range payload.Items {
		out[u.UserID] = u.DisplayName
	}
	return out, nil
}

func (c *StackOverflowClient) fetchAnswersMap(ctx context.Context, ids map[int64]struct{}) (map[int64]string, error) {
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}
	parts := make([]string, 0, len(ids))
	for id := range ids {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	sort.Strings(parts)
	reqURL := fmt.Sprintf("%s/answers/%s?site=stackoverflow&filter=withbody", c.baseURL, strings.Join(parts, ";"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build stackoverflow answers request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do stackoverflow answers request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stackoverflow answers non-2xx: %d: %s", resp.StatusCode, string(b))
	}
	var payload dto.StackAnswersResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode stackoverflow answers: %w", err)
	}
	out := make(map[int64]string, len(payload.Items))
	for _, a := range payload.Items {
		out[a.AnswerID] = a.Body
	}
	return out, nil
}
