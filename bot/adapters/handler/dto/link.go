package dto

import (
	"errors"
	"strings"
)

var ErrInvalidLinkUpdate = errors.New("url and tgChatIds are required")

type LinkUpdate struct {
	ID          int64   `json:"id"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	TgChatIDs   []int64 `json:"tgChatIds"`
}

func (u LinkUpdate) Validate() error {
	if strings.TrimSpace(u.URL) == "" || len(u.TgChatIDs) == 0 {
		return ErrInvalidLinkUpdate
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
