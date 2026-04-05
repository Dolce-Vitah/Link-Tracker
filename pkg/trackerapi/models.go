package trackerapi

import (
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/handlerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/linkdto"
)

var (
	ErrInvalidLinkUpdate       = linkdto.ErrInvalidLinkUpdate
	ErrInvalidLinkUpdateDetail = linkdto.ErrInvalidLinkUpdateDetail
	ErrInvalidFailureReport    = linkdto.ErrInvalidFailureReport
)

const (
	EventKindGitHubIssue       = linkdto.EventKindGitHubIssue
	EventKindGitHubPullRequest = linkdto.EventKindGitHubPullRequest
	EventKindStackAnswer       = linkdto.EventKindStackAnswer
	EventKindStackComment      = linkdto.EventKindStackComment
	PreviewMaxRunes            = linkdto.PreviewMaxRunes
)

type ErrorResponse = handlerapi.ErrorResponse

type LinkUpdate = linkdto.LinkUpdate
type ProcessingFailureReport = linkdto.ProcessingFailureReport
type LinkResponse = linkdto.LinkResponse
type AddLinkRequest = linkdto.AddLinkRequest
type RemoveLinkRequest = linkdto.RemoveLinkRequest
type ListLinksResponse = linkdto.ListLinksResponse

func NewLinkUpdate(
	id int64,
	rawURL string,
	chatIDs []int64,
	eventKind, title, author string,
	createdAt time.Time,
	preview string,
) LinkUpdate {
	return linkdto.NewLinkUpdate(id, rawURL, chatIDs, eventKind, title, author, createdAt, preview)
}
