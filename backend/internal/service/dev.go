package service

import (
	"sync"

	"voxcanvas/backend/internal/llm"
)

type DevStore struct {
	mu  sync.RWMutex
	dev string
}

func NewDevStore() *DevStore {
	return &DevStore{}
}

func (d *DevStore) Get() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.dev
}

func (d *DevStore) Set(s string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dev = s
}

func (d *DevStore) Append(newSentence string, refiner llm.Refiner) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	refined, err := refiner.Refine(d.dev, newSentence)
	if err != nil {
		return "", err
	}
	d.dev = refined
	return refined, nil
}
