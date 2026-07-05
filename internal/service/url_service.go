package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"

	"github.com/gradis/ya-pr_shorturl/internal/repository"
)

const defaultBaseURL = "http://localhost:8080"

var (
	ErrURLNotFound = errors.New("url not found")
	ErrInvalidURL  = errors.New("invalid URL")
)

type URLService struct {
	repo    repository.URLRepository
	baseURL string
}

type BatchURL struct {
	CorrelationID string
	OriginalURL   string
}

type BatchURLResult struct {
	CorrelationID string
	ShortURL      string
}

func NewURLService(repo repository.URLRepository, baseURL string) *URLService {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	baseURL = strings.TrimRight(baseURL, "/")

	return &URLService{
		repo:    repo,
		baseURL: baseURL,
	}
}

func (s *URLService) AddURL(ctx context.Context, originalURL string) (string, error) {
	if !isValidURL(originalURL) {
		return "", ErrInvalidURL
	}

	id, err := s.saveWithUniqueID(ctx, originalURL)
	if err != nil {
		return "", fmt.Errorf("save shortened URL: %w", err)
	}

	shortURL := fmt.Sprintf("%s/%s", s.baseURL, id)

	return shortURL, nil
}

func (s *URLService) AddBatchURLs(ctx context.Context, urls []BatchURL) ([]BatchURLResult, error) {
	records := make([]repository.URLRecord, 0, len(urls))
	results := make([]BatchURLResult, 0, len(urls))

	for _, item := range urls {
		if !isValidURL(item.OriginalURL) {
			return nil, ErrInvalidURL
		}

		id, err := generateID(8)
		if err != nil {
			return nil, err
		}

		records = append(records, repository.URLRecord{
			ID:          id,
			OriginalURL: item.OriginalURL,
		})

		results = append(results, BatchURLResult{
			CorrelationID: item.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", s.baseURL, id),
		})
	}

	if err := s.repo.SaveBatch(ctx, records); err != nil {
		return nil, fmt.Errorf("save shortened URL batch: %w", err)
	}

	return results, nil
}

func (s *URLService) GetURLByID(ctx context.Context, id string) (string, error) {
	originalURL, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrURLNotFound) {
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("get URL by ID: %w", err)
	}

	return originalURL, nil
}

func (s *URLService) saveWithUniqueID(ctx context.Context, originalURL string) (string, error) {
	const maxAttempts = 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		id, err := generateID(8)
		if err != nil {
			return "", err
		}

		saved, err := s.repo.SaveIfNotExist(ctx, id, originalURL)
		if err != nil {
			return "", fmt.Errorf("save URL: %w", err)
		}

		if saved {
			return id, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique id after %d attempts", maxAttempts)
}

func generateID(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	result := make([]byte, length)

	for i := range result {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}

		result[i] = alphabet[randomIndex.Int64()]
	}

	return string(result), nil
}

func isValidURL(value string) bool {
	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return false
	}

	return true
}
