package dto

import (
	"errors"
	"strings"
)

var (
	ErrEmptyText     = errors.New("text is empty")
	ErrInvalidChatID = errors.New("chat id must be positive")
)

type CommandRequest struct {
	Text   string
	ChatID int64
}

func (r CommandRequest) Validate() error {
	if r.ChatID <= 0 {
		return ErrInvalidChatID
	}

	if strings.TrimSpace(r.Text) == "" {
		return ErrEmptyText
	}
	return nil
}
