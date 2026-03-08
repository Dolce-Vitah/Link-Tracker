package dialog

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

type Store struct {
	mu       sync.RWMutex
	sessions map[int64]Session
}

func NewStore() *Store {
	return &Store{
		sessions: make(map[int64]Session),
	}
}

func (s *Store) Get(chatID int64) (Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[chatID]
	return session, ok
}

func (s *Store) Set(chatID int64, session Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[chatID] = session
}

func (s *Store) Clear(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, chatID)
}
