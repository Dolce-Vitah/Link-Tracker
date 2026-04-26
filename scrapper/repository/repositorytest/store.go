package repositorytest

import (
	"sync"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

type memoryLink struct {
	tracked repository.TrackedLink
}

type Store struct {
	mu             sync.RWMutex
	nextID         int64
	nextTagID      int64
	chats          map[int64]struct{}
	linksByURL     map[string]*memoryLink
	linksByChat    map[int64]map[string]struct{}
	tagsByChat     map[int64]map[string]repository.Tag
	tagsByChatLink map[int64]map[string]map[string]struct{}
	outbox         []repository.OutboxMessage
	nextOutboxID   int64
}

func NewStore() *Store {
	return &Store{
		nextID:         1,
		nextTagID:      1,
		chats:          make(map[int64]struct{}),
		linksByURL:     make(map[string]*memoryLink),
		linksByChat:    make(map[int64]map[string]struct{}),
		tagsByChat:     make(map[int64]map[string]repository.Tag),
		tagsByChatLink: make(map[int64]map[string]map[string]struct{}),
		nextOutboxID:   1,
	}
}
