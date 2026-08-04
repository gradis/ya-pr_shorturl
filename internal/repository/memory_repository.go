package repository

import (
	"context"
	"sync"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	urls        map[string]URLRecord
	originalIDs map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		urls:        make(map[string]URLRecord),
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

	userID, _ := auth.UserIDFromContext(ctx)
	r.urls[id] = URLRecord{
		ID:          id,
		OriginalURL: originalURL,
		UserID:      userID,
	}
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

		if record.UserID == "" {
			record.UserID, _ = auth.UserIDFromContext(ctx)
		}
		r.urls[record.ID] = record
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

	record, exists := r.urls[id]
	if !exists {
		return "", ErrURLNotFound
	}

	if record.IsDeleted {
		return "", ErrURLDeleted
	}

	return record.OriginalURL, nil
}

func (r *MemoryRepository) GetByUserID(ctx context.Context, userID string) ([]URLRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	records := make([]URLRecord, 0)
	for _, record := range r.urls {
		if record.UserID == userID {
			records = append(records, record)
		}
	}

	return records, nil
}

func (r *MemoryRepository) DeleteBatch(ctx context.Context, records []URLDeleteRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, deleteRecord := range records {
		record, exists := r.urls[deleteRecord.ID]
		if !exists {
			continue
		}

		if record.UserID != deleteRecord.UserID {
			continue
		}

		record.IsDeleted = true
		r.urls[deleteRecord.ID] = record
	}

	return nil
}

func (r *MemoryRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.urls[id]
	return ok
}
