package dto

import "time"

type GitHubRepoResponse struct {
	UpdatedAt time.Time `json:"updated_at"`
}

type StackOverflowResponse struct {
	Items []StackOverflowQuestion `json:"items"`
}

type StackOverflowQuestion struct {
	QuestionID       int64  `json:"question_id"`
	Title            string `json:"title"`
	LastActivityDate int64  `json:"last_activity_date"`
}

type GitHubIssueItem struct {
	HTMLURL     string    `json:"html_url"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
	User        struct {
		Login string `json:"login"`
	} `json:"user"`
	PullRequest *struct {
		URL string `json:"url"`
	} `json:"pull_request"`
}

type StackTimelineResponse struct {
	Items []StackTimelineItem `json:"items"`
}

type StackTimelineItem struct {
	TimelineType string `json:"timeline_type"`
	CreationDate int64  `json:"creation_date"`
	PostID       int64  `json:"post_id"`
	UserID       int64  `json:"user_id"`
	Detail       string  `json:"detail"`
}

type StackAnswersResponse struct {
	Items []StackAnswerItem `json:"items"`
}

type StackAnswerItem struct {
	AnswerID int64  `json:"answer_id"`
	Body     string `json:"body"`
}

type StackUsersResponse struct {
	Items []StackUserItem `json:"items"`
}

type StackUserItem struct {
	UserID          int64  `json:"user_id"`
	DisplayName     string `json:"display_name"`
}
