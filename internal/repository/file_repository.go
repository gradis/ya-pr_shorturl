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

func (r *FileRepository) SaveIfNotExist(ctx context.Context, id string, originalURL string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.urls[id]; exists {
		return false, nil
	}

	r.urls[id] = originalURL

	if err := r.flush(); err != nil {
		delete(r.urls, id)
		return false, err
	}

	return true, nil
}

func (r *FileRepository) GetByID(ctx context.Context, id string) (string, error) {
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

		r.urls[record.ShortURL] = record.OriginalURL
	}

	return scanner.Err()
}

func (r *FileRepository) loadJSONRecords(data []byte) error {
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

var _ URLRepository = (*FileRepository)(nil)
