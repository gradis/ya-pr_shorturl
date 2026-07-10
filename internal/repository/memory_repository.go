package repository

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	urls        map[string]string
	originalIDs map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		urls:        make(map[string]string),
		originalIDs: make(map[string]string),
	}
}

func (r *MemoryRepository) Ping(context.Context) error {
	return nil
}

func (r *MemoryRepository) Close() {}

func (r *MemoryRepository) SaveURL(ctx context.Context, id string, originalURL string) (URLSaveResult, error) {
	if err := ctx.Err(); err != nil {
		return URLSaveResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existingID, ok := r.originalIDs[originalURL]; ok {
		return URLSaveResult{
			ID:        existingID,
			Duplicate: true,
		}, nil
	}

	if _, ok := r.urls[id]; ok {
		return URLSaveResult{}, ErrURLIDConflict
	}

	r.urls[id] = originalURL
	r.originalIDs[originalURL] = id

	return URLSaveResult{ID: id}, nil
}

func (r *MemoryRepository) SaveBatch(ctx context.Context, records []URLRecord) ([]URLRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]URLRecord, 0, len(records))
	for _, record := range records {
		if existingID, ok := r.originalIDs[record.OriginalURL]; ok {
			result = append(result, URLRecord{
				ID:          existingID,
				OriginalURL: record.OriginalURL,
			})
			continue
		}

		if _, ok := r.urls[record.ID]; ok {
			return nil, ErrURLIDConflict
		}

		r.urls[record.ID] = record.OriginalURL
		r.originalIDs[record.OriginalURL] = record.ID
		result = append(result, record)
	}

	return result, nil
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
