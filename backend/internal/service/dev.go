package service

import (
	"sync"

	"voxcanvas/backend/internal/llm"
)

type DevStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewDevStore() *DevStore {
	return &DevStore{data: make(map[string]string)}
}

func (d *DevStore) Get(sessionID string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.data[sessionID]
}

func (d *DevStore) Set(sessionID, value string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if value == "" {
		delete(d.data, sessionID)
		return
	}
	d.data[sessionID] = value
}

func (d *DevStore) Append(sessionID, newSentence string, refiner llm.Refiner) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	refined, err := refiner.Refine(d.data[sessionID], newSentence)
	if err != nil {
		return "", err
	}
	d.data[sessionID] = refined
	return refined, nil
}
