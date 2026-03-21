package dto

import "time"

type GitHubRepoResponse struct {
	UpdatedAt time.Time `json:"updated_at"`
}

type StackOverflowResponse struct {
	Items []StackOverflowQuestion `json:"items"`
}

type StackOverflowQuestion struct {
	LastActivityDate int64 `json:"last_activity_date"`
}
