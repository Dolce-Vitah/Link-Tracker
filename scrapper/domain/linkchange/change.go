package linkchange

import (
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/linkdto"
)

type Change struct {
	Kind      string 
	Title     string 
	Author    string
	CreatedAt time.Time
	Preview   string 
}

const (
	KindGitHubIssue       = linkdto.EventKindGitHubIssue
	KindGitHubPullRequest = linkdto.EventKindGitHubPullRequest
	KindStackAnswer       = linkdto.EventKindStackAnswer
	KindStackComment      = linkdto.EventKindStackComment
)
