package service

import "sync"

type GeneratedResult struct {
	ImageID int64
	Text    string
	Image   string
}

type GeneratedStore struct {
	mu           sync.RWMutex
	data         map[string][]GeneratedResult
	currentIndex map[string]int
	undoIndex    map[string]int
}

func NewGeneratedStore() *GeneratedStore {
	return &GeneratedStore{
		data:         make(map[string][]GeneratedResult),
		currentIndex: make(map[string]int),
		undoIndex:    make(map[string]int),
	}
}

func (s *GeneratedStore) Get(sessionID string) (GeneratedResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := s.data[sessionID]
	if len(results) == 0 {
		return GeneratedResult{}, false
	}
	index, ok := s.currentIndex[sessionID]
	if ok && index < 0 {
		return GeneratedResult{}, false
	}
	if !ok || index >= len(results) {
		index = len(results) - 1
	}
	return results[index], true
}

func (s *GeneratedStore) Set(sessionID string, result GeneratedResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[sessionID] = append(s.data[sessionID], result)
	index := len(s.data[sessionID]) - 1
	s.currentIndex[sessionID] = index
	s.undoIndex[sessionID] = index
}

func (s *GeneratedStore) UndoPrevious(sessionID string) (GeneratedResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	results := s.data[sessionID]
	if len(results) == 0 {
		return GeneratedResult{}, false
	}
	index, ok := s.undoIndex[sessionID]
	if !ok {
		index = len(results) - 1
	}
	if index < 0 || index >= len(results) {
		return GeneratedResult{}, false
	}
	s.currentIndex[sessionID] = index
	s.undoIndex[sessionID] = index - 1
	return results[index], true
}

func (s *GeneratedStore) Clear(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	results := s.data[sessionID]
	if len(results) == 0 {
		delete(s.data, sessionID)
		delete(s.currentIndex, sessionID)
		delete(s.undoIndex, sessionID)
		return
	}
	s.currentIndex[sessionID] = -1
	if _, ok := s.undoIndex[sessionID]; !ok {
		s.undoIndex[sessionID] = len(results) - 1
	}
}
