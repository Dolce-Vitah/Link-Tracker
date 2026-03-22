package repositorytest

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"

func (s *Store) RegisterChat(chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; ok {
		return repository.ErrChatExists
	}
	s.chats[chatID] = struct{}{}
	return nil
}

func (s *Store) DeleteChat(chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; !ok {
		return repository.ErrChatNotFound
	}
	delete(s.chats, chatID)
	if linkURLs, ok := s.linksByChat[chatID]; ok {
		for linkURL := range linkURLs {
			tracked := s.linksByURL[linkURL]
			delete(tracked.tracked.ChatIDs, chatID)
			if len(tracked.tracked.ChatIDs) == 0 {
				delete(s.linksByURL, linkURL)
			}
		}
		delete(s.linksByChat, chatID)
	}
	delete(s.tagsByChat, chatID)
	delete(s.tagsByChatLink, chatID)
	return nil
}
