package api

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/common"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/linkdto"
)

type ErrorResponse = common.ErrorResponse

type LinkUpdate = linkdto.LinkUpdate
type LinkResponse = linkdto.LinkResponse
type AddLinkRequest = linkdto.AddLinkRequest
type RemoveLinkRequest = linkdto.RemoveLinkRequest
type ListLinksResponse = linkdto.ListLinksResponse

var ErrInvalidLinkUpdate = linkdto.ErrInvalidLinkUpdate
