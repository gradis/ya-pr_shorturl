package repository

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
)

type FileRepository struct {
	mu          sync.RWMutex
	filePath    string
	urls        map[string]URLRecord
	originalIDs map[string]string
}

type FileRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
}

func NewFileRepository(filePath string) (*FileRepository, error) {
	repo := &FileRepository{
		filePath:    filePath,
		urls:        make(map[string]URLRecord),
		originalIDs: make(map[string]string),
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *FileRepository) Ping(context.Context) error {
	return nil
}

func (r *FileRepository) Close() {}

func (r *FileRepository) SaveURL(ctx context.Context, id string, originalURL string) (URLSaveResult, error) {
	if err := ctx.Err(); err != nil {
		return URLSaveResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existingID, exists := r.originalIDs[originalURL]; exists {
		return URLSaveResult{
			ID:        existingID,
			Duplicate: true,
		}, nil
	}

	if _, exists := r.urls[id]; exists {
		return URLSaveResult{}, ErrURLIDConflict
	}

	userID, _ := auth.UserIDFromContext(ctx)
	r.urls[id] = URLRecord{
		ID:          id,
		OriginalURL: originalURL,
		UserID:      userID,
	}
	r.originalIDs[originalURL] = id

	if err := r.flush(); err != nil {
		delete(r.urls, id)
		delete(r.originalIDs, originalURL)
		return URLSaveResult{}, err
	}

	return URLSaveResult{ID: id}, nil
}

func (r *FileRepository) SaveBatch(ctx context.Context, records []URLRecord) ([]URLRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]URLRecord, 0, len(records))
	urlsRollback := make(map[string]URLRecord, len(records))
	originalIDsRollback := make(map[string]string, len(records))

	for _, record := range records {
		if existingID, exists := r.originalIDs[record.OriginalURL]; exists {
			result = append(result, URLRecord{
				ID:          existingID,
				OriginalURL: record.OriginalURL,
			})
			continue
		}

		if current, exists := r.urls[record.ID]; exists {
			urlsRollback[record.ID] = current
			return nil, ErrURLIDConflict
		}

		if record.UserID == "" {
			record.UserID, _ = auth.UserIDFromContext(ctx)
		}
		r.urls[record.ID] = record
		r.originalIDs[record.OriginalURL] = record.ID
		originalIDsRollback[record.OriginalURL] = ""
		result = append(result, record)
	}

	if err := r.flush(); err != nil {
		for _, record := range records {
			if previous, exists := urlsRollback[record.ID]; exists {
				r.urls[record.ID] = previous
				continue
			}

			delete(r.urls, record.ID)

			if _, exists := originalIDsRollback[record.OriginalURL]; exists {
				delete(r.originalIDs, record.OriginalURL)
			}
		}

		return nil, err
	}

	return result, nil
}

func (r *FileRepository) GetByID(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	record, exists := r.urls[id]
	if !exists {
		return "", ErrURLNotFound
	}

	return record.OriginalURL, nil
}

func (r *FileRepository) GetByUserID(ctx context.Context, userID string) ([]URLRecord, error) {
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

func (r *FileRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.urls[id]
	return ok
}

func (r *FileRepository) load() error {
	if r.filePath == "" {
		return nil
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	if len(data) == 0 {
		return nil
	}

	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
		return r.loadJSONRecords(data)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var record FileRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return err
		}

		r.urls[record.ShortURL] = URLRecord{
			ID:          record.ShortURL,
			OriginalURL: record.OriginalURL,
			UserID:      record.UserID,
		}
		r.originalIDs[record.OriginalURL] = record.ShortURL
	}

	return scanner.Err()
}

func (r *FileRepository) loadJSONRecords(data []byte) error {
	var records []FileRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	for _, record := range records {
		r.urls[record.ShortURL] = URLRecord{
			ID:          record.ShortURL,
			OriginalURL: record.OriginalURL,
			UserID:      record.UserID,
		}
		r.originalIDs[record.OriginalURL] = record.ShortURL
	}

	return nil
}

func (r *FileRepository) flush() error {
	if r.filePath == "" {
		return nil
	}

	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	records := make([]FileRecord, 0, len(r.urls))

	i := 1
	for shortID, record := range r.urls {
		records = append(records, FileRecord{
			UUID:        strconv.Itoa(i),
			ShortURL:    shortID,
			OriginalURL: record.OriginalURL,
			UserID:      record.UserID,
		})
		i++
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)

	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return err
		}
	}

	tempFile, err := os.CreateTemp(dir, ".url-storage-*.tmp")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if err := tempFile.Chmod(0644); err != nil {
		return err
	}

	if _, err := tempFile.Write(buffer.Bytes()); err != nil {
		return err
	}

	if err := tempFile.Sync(); err != nil {
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, r.filePath); err != nil {
		return err
	}

	return nil
}
