package handler

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dialog"

type SessionStore interface {
	Get(chatID int64) (dialog.Session, bool)
	Set(chatID int64, session dialog.Session)
	Clear(chatID int64)
}
