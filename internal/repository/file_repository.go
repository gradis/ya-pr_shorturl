package repository

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type FileRepository struct {
	mu       sync.RWMutex
	filePath string
	urls     map[string]string
}

type FileRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func NewFileRepository(filePath string) (*FileRepository, error) {
	repo := &FileRepository{
		filePath: filePath,
		urls:     make(map[string]string),
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *FileRepository) SaveIfNotExists(id string, originalURL string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.urls[id]; ok {
		return false, nil
	}

	r.urls[id] = originalURL

	if err := r.flush(); err != nil {
		delete(r.urls, id)
		return false, err
	}

	return true, nil
}

func (r *FileRepository) GetByID(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	originalURL, ok := r.urls[id]
	return originalURL, ok
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

	var records []FileRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	for _, record := range records {
		r.urls[record.ShortURL] = record.OriginalURL
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
	for shortID, originalURL := range r.urls {
		records = append(records, FileRecord{
			UUID:        strconv.Itoa(i),
			ShortURL:    shortID,
			OriginalURL: originalURL,
		})
		i++
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
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

	if _, err := tempFile.Write(data); err != nil {
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
