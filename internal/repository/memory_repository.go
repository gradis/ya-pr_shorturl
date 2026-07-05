package repository

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu   sync.RWMutex
	urls map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		urls: make(map[string]string),
	}
}

func (r *MemoryRepository) SaveIfNotExist(ctx context.Context, id string, originalURL string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.urls[id]; ok {
		return false, nil
	}

	r.urls[id] = originalURL
	return true, nil
}

func (r *MemoryRepository) GetByID(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	originalURL, exists := r.urls[id]
	if !exists {
		return "", ErrURLNotFound
	}
	return originalURL, nil
}

func (r *MemoryRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.urls[id]
	return ok
}

var _ URLRepository = (*MemoryRepository)(nil)
