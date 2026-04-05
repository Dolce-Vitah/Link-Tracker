package linkdto

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/textutil"
)

const PreviewMaxRunes = 200

var (
	ErrInvalidLinkUpdate   = errors.New("url and tgChatIds are required")
	ErrInvalidLinkUpdateDetail = errors.New("eventKind, title, author and createdAt are required")
	ErrInvalidFailureReport = errors.New("tgChatId and urls are required")
)

const (
	EventKindGitHubIssue       = "github_issue"
	EventKindGitHubPullRequest = "github_pull_request"
	EventKindStackAnswer       = "stackoverflow_answer"
	EventKindStackComment      = "stackoverflow_comment"
)

type LinkUpdate struct {
	ID          int64     `json:"id"`
	URL         string    `json:"url"`
	TgChatIDs   []int64   `json:"tgChatIds"`
	EventKind   string    `json:"eventKind"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	CreatedAt   time.Time `json:"createdAt"`
	Preview     string    `json:"preview"`
	Description string    `json:"description,omitempty"`
}

func (u LinkUpdate) Validate() error {
	if strings.TrimSpace(u.URL) == "" || len(u.TgChatIDs) == 0 {
		return ErrInvalidLinkUpdate
	}
	if strings.TrimSpace(u.EventKind) == "" ||
		strings.TrimSpace(u.Title) == "" ||
		strings.TrimSpace(u.Author) == "" ||
		u.CreatedAt.IsZero() {
		return ErrInvalidLinkUpdateDetail
	}
	if utf8.RuneCountInString(u.Preview) > PreviewMaxRunes {
		return errors.New("preview exceeds 200 runes")
	}
	return nil
}

func NewLinkUpdate(
	id int64,
	rawURL string,
	chatIDs []int64,
	eventKind, title, author string,
	createdAt time.Time,
	preview string,
) LinkUpdate {
	return LinkUpdate{
		ID:        id,
		URL:       rawURL,
		TgChatIDs: chatIDs,
		EventKind: eventKind,
		Title:     strings.TrimSpace(title),
		Author:    strings.TrimSpace(author),
		CreatedAt: createdAt.UTC(),
		Preview:   textutil.TruncateRunes(strings.TrimSpace(preview), PreviewMaxRunes),
	}
}

type ProcessingFailureReport struct {
	TgChatID int64    `json:"tgChatId"`
	URLs     []string `json:"urls"`
	Detail   string   `json:"detail,omitempty"`
}

func (r ProcessingFailureReport) Validate() error {
	if r.TgChatID == 0 || len(r.URLs) == 0 {
		return ErrInvalidFailureReport
	}
	return nil
}

type LinkResponse struct {
	ID      int64    `json:"id"`
	URL     string   `json:"url"`
	Tags    []string `json:"tags,omitempty"`
	Filters []string `json:"filters,omitempty"`
}

type AddLinkRequest struct {
	Link    string   `json:"link"`
	Tags    []string `json:"tags,omitempty"`
	Filters []string `json:"filters,omitempty"`
}

type RemoveLinkRequest struct {
	Link string `json:"link"`
}

type ListLinksResponse struct {
	Links []LinkResponse `json:"links"`
	Size  int32          `json:"size"`
}
