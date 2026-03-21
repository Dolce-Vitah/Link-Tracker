package dto

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"

type RegisterChatRequest struct {
	ChatID int64 `json:"chatId"`
}

type RegisterChatResponse struct{}

type DeleteChatRequest struct {
	ChatID int64 `json:"chatId"`
}

type DeleteChatResponse struct{}

type AddLinkGRPCRequest struct {
	ChatID int64              `json:"chatId"`
	Body   api.AddLinkRequest `json:"body"`
}

type AddLinkGRPCResponse struct {
	Link api.LinkResponse `json:"link"`
}

type RemoveLinkGRPCRequest struct {
	ChatID int64                 `json:"chatId"`
	Body   api.RemoveLinkRequest `json:"body"`
}

type RemoveLinkGRPCResponse struct {
	Link api.LinkResponse `json:"link"`
}

type ListLinksRequest struct {
	ChatID int64 `json:"chatId"`
}

type ListLinksResponse struct {
	Body api.ListLinksResponse `json:"body"`
}
