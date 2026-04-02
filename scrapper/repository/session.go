package repository

import "sync"

type State string

const (
	StateIdle         State = "idle"
	StateAwaitingURL  State = "awaiting_url"
	StateAwaitingTags State = "awaiting_tags"
)

type Session struct {
	State      State
	PendingURL string
}

type SessionRepository interface {
	Get(chatID int64) (Session, bool)
	Set(chatID int64, session Session)
	Clear(chatID int64)
}

type InMemorySessionRepository struct {
	mu       sync.RWMutex
	sessions map[int64]Session
}

func NewInMemorySessionRepository() *InMemorySessionRepository {
	return &InMemorySessionRepository{
		sessions: make(map[int64]Session),
	}
}

func (r *InMemorySessionRepository) Get(chatID int64) (Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[chatID]

	return session, ok
}

func (r *InMemorySessionRepository) Set(chatID int64, session Session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[chatID] = session
}

func (r *InMemorySessionRepository) Clear(chatID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, chatID)
}
