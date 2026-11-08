package repository

import (
	"context"
	"sync"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
)

// MemoryRepository хранит ссылки и индекс оригинальных URL в памяти.
// Методы защищены мьютексом и допускают конкурентные запросы.
type MemoryRepository struct {
	mu          sync.RWMutex
	urls        map[string]URLRecord
	originalIDs map[string]string
}

// NewMemoryRepository создаёт пустое хранилище в памяти.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		urls:        make(map[string]URLRecord),
		originalIDs: make(map[string]string),
	}
}

// Ping всегда возвращает nil: локальному хранилищу не нужно соединение с БД.
func (r *MemoryRepository) Ping(context.Context) error {
	return nil
}

// Close не выполняет действий: хранилище не держит открытых файлов или соединений.
func (r *MemoryRepository) Close() {}

// SaveURL сохраняет ссылку с владельцем из контекста.
// Для повторного оригинального URL возвращает существующий ID и Duplicate=true.
// При занятом ID другой ссылки возвращает ErrURLIDConflict.
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

// SaveBatch сохраняет пакет ссылок и возвращает фактические ID в исходном порядке.
// Повторные оригинальные URL используют существующие записи. Для новых записей
// пустой UserID заполняется из контекста.
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

// GetByID возвращает оригинальный URL, ErrURLNotFound или ErrURLDeleted.
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

// GetByUserID возвращает записи пользователя, включая помеченные как удалённые.
// Порядок записей не определён.
func (r *MemoryRepository) GetByUserID(ctx context.Context, userID string) ([]URLRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Count under the same read lock to allocate the exact result size.
	// This avoids allocating intermediate slices during repeated growth.
	count := 0
	for _, record := range r.urls {
		if record.UserID == userID {
			count++
		}
	}
	records := make([]URLRecord, 0, count)
	for _, record := range r.urls {
		if record.UserID == userID {
			records = append(records, record)
		}
	}

	return records, nil
}

// DeleteBatch помечает ссылки как удалённые только при совпадении владельца.
// Несуществующие ID и записи других пользователей пропускаются.
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

// Exists сообщает, занят ли короткий ID, включая помеченные как удалённые записи.
func (r *MemoryRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.urls[id]
	return ok
}
