package trackerapi

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/dto"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/adapters/handler/handlerapi"
)

var ErrInvalidLinkUpdate = dto.ErrInvalidLinkUpdate

type ErrorResponse = handlerapi.ErrorResponse

type LinkUpdate = dto.LinkUpdate
type LinkResponse = dto.LinkResponse
type AddLinkRequest = dto.AddLinkRequest
type RemoveLinkRequest = dto.RemoveLinkRequest
type ListLinksResponse = dto.ListLinksResponse
