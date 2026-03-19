package dto

import (
	"errors"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var ErrInvalidUpdate = errors.New("update or message is nil")

type DialogRequest struct {
	Update *tgbotapi.Update
}

func (r DialogRequest) Validate() error {
	if r.Update == nil || r.Update.Message == nil {
		return ErrInvalidUpdate
	}
	return nil
}

func (r DialogRequest) ChatID() int64 {
	return r.Update.Message.Chat.ID
}

func (r DialogRequest) Text() string {
	return strings.TrimSpace(r.Update.Message.Text)
}

func (r DialogRequest) IsCommand() bool {
	return r.Update.Message.IsCommand()
}

func (r DialogRequest) Command() string {
	return r.Update.Message.Command()
}
