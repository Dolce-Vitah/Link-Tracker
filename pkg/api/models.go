package api

type ErrorResponse struct {
	Description      string   `json:"description"`
	Code             string   `json:"code"`
	ExceptionName    string   `json:"exceptionName,omitempty"`
	ExceptionMessage string   `json:"exceptionMessage,omitempty"`
	Stacktrace       []string `json:"stacktrace,omitempty"`
}

type LinkUpdate struct {
	ID          int64   `json:"id"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	TgChatIDs   []int64 `json:"tgChatIds"`
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
