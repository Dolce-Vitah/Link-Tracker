package trackerapi

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/common"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/linkdto"
)

var ErrInvalidLinkUpdate = linkdto.ErrInvalidLinkUpdate

type ErrorResponse = common.ErrorResponse

type LinkUpdate = linkdto.LinkUpdate
type LinkResponse = linkdto.LinkResponse
type AddLinkRequest = linkdto.AddLinkRequest
type RemoveLinkRequest = linkdto.RemoveLinkRequest
type ListLinksResponse = linkdto.ListLinksResponse
