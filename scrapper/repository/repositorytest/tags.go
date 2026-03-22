package repositorytest

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"

func (s *Store) CreateTag(chatID int64, tag string) (repository.Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; !ok {
		return repository.Tag{}, repository.ErrChatNotFound
	}
	normalized := normalizeTag(tag)
	if normalized == "" {
		return repository.Tag{}, repository.ErrInvalidTag
	}
	if _, ok := s.tagsByChat[chatID]; !ok {
		s.tagsByChat[chatID] = make(map[string]repository.Tag)
	}
	if existing, exists := s.tagsByChat[chatID][normalized]; exists {
		return existing, repository.ErrTagExists
	}
	created := repository.Tag{ID: s.nextTagID, ChatID: chatID, Name: normalized}
	s.nextTagID++
	s.tagsByChat[chatID][normalized] = created
	return created, nil
}

func (s *Store) ListTags(chatID int64, limit int, offset int) ([]repository.Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.chats[chatID]; !ok {
		return nil, repository.ErrChatNotFound
	}
	chatTags := s.tagsByChat[chatID]
	out := make([]repository.Tag, 0, len(chatTags))
	for _, tag := range chatTags {
		out = append(out, tag)
	}
	if limit <= 0 {
		limit = len(out)
	}
	if offset >= len(out) {
		return []repository.Tag{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (s *Store) UpdateTag(chatID int64, oldTag string, newTag string) (repository.Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; !ok {
		return repository.Tag{}, repository.ErrChatNotFound
	}
	oldName := normalizeTag(oldTag)
	newName := normalizeTag(newTag)
	if oldName == "" || newName == "" {
		return repository.Tag{}, repository.ErrInvalidTag
	}
	chatTags := s.tagsByChat[chatID]
	existing, ok := chatTags[oldName]
	if !ok {
		return repository.Tag{}, repository.ErrTagNotFound
	}
	if _, exists := chatTags[newName]; exists && oldName != newName {
		return repository.Tag{}, repository.ErrTagExists
	}
	delete(chatTags, oldName)
	existing.Name = newName
	chatTags[newName] = existing

	if byLink, okByLink := s.tagsByChatLink[chatID]; okByLink {
		for _, linkTags := range byLink {
			if _, has := linkTags[oldName]; has {
				delete(linkTags, oldName)
				linkTags[newName] = struct{}{}
			}
		}
	}
	return existing, nil
}

func (s *Store) DeleteTag(chatID int64, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; !ok {
		return repository.ErrChatNotFound
	}
	name := normalizeTag(tag)
	if name == "" {
		return repository.ErrTagNotFound
	}
	chatTags := s.tagsByChat[chatID]
	if _, ok := chatTags[name]; !ok {
		return repository.ErrTagNotFound
	}
	delete(chatTags, name)
	if byLink, okByLink := s.tagsByChatLink[chatID]; okByLink {
		for _, linkTags := range byLink {
			delete(linkTags, name)
		}
	}
	return nil
}
