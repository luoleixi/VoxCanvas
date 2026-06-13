package service

import "sync"

type GeneratedResult struct {
	ImageID int64
	Text    string
	Image   string
}

type GeneratedStore struct {
	mu   sync.RWMutex
	data map[string]GeneratedResult
}

func NewGeneratedStore() *GeneratedStore {
	return &GeneratedStore{data: make(map[string]GeneratedResult)}
}

func (s *GeneratedStore) Get(sessionID string) (GeneratedResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result, ok := s.data[sessionID]
	return result, ok
}

func (s *GeneratedStore) Set(sessionID string, result GeneratedResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[sessionID] = result
}

func (s *GeneratedStore) Clear(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, sessionID)
}
