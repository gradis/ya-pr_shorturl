package repository

import "sync"

type MemoryRepository struct {
	mu   sync.RWMutex
	urls map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		urls: make(map[string]string),
	}
}

func (r *MemoryRepository) Save(id string, originalURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.urls[id] = originalURL
	return nil
}

func (r *MemoryRepository) GetByID(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	originalURL, ok := r.urls[id]

	return originalURL, ok
}

func (r *MemoryRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.urls[id]
	return ok
}
