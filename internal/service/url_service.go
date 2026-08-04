package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"sync"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"go.uber.org/zap"
)

const defaultBaseURL = "http://localhost:8080"

var (
	ErrURLNotFound      = errors.New("url not found")
	ErrURLDeleted       = errors.New("url deleted")
	ErrInvalidURL       = errors.New("invalid URL")
	ErrInvalidURLIDs    = errors.New("invalid URL ids")
	ErrURLAlreadyExists = errors.New("URL already exists")
	ErrUnauthorized     = errors.New("user is not authenticated")
)

type URLRepository interface {
	SaveURL(ctx context.Context, id string, originalURL string) (repository.URLSaveResult, error)
	SaveBatch(ctx context.Context, records []repository.URLRecord) ([]repository.URLRecord, error)
	GetByID(ctx context.Context, id string) (string, error)
	DeleteBatch(ctx context.Context, records []repository.URLDeleteRecord) error
}

type UserURLRepository interface {
	GetByUserID(ctx context.Context, userID string) ([]repository.URLRecord, error)
}

type URLService struct {
	repo    URLRepository
	baseURL string
	logg    *zap.Logger

	deleteQueue chan deleteRequest
	deleteWG    sync.WaitGroup
	closeOnce   sync.Once
}

type BatchURL struct {
	CorrelationID string
	OriginalURL   string
}

type BatchURLResult struct {
	CorrelationID string
	ShortURL      string
}

type UserURL struct {
	ShortURL    string
	OriginalURL string
}

func NewURLService(repo URLRepository, baseURL string) *URLService {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	baseURL = strings.TrimRight(baseURL, "/")

	return NewURLServiceWithLogger(
		repo,
		baseURL,
		zap.NewNop(),
	)
}

func NewURLServiceWithLogger(
	repo URLRepository,
	baseURL string,
	logg *zap.Logger,
) *URLService {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	if logg == nil {
		logg = zap.NewNop()
	}

	service := &URLService{
		repo:        repo,
		baseURL:     strings.TrimRight(baseURL, "/"),
		logg:        logg,
		deleteQueue: make(chan deleteRequest, deleteQueueSize),
	}

	service.startDeleteWorker()

	return service
}

func (s *URLService) AddURL(ctx context.Context, originalURL string) (string, error) {
	if !isValidURL(originalURL) {
		return "", ErrInvalidURL
	}

	id, duplicate, err := s.saveWithUniqueID(ctx, originalURL)
	if err != nil {
		return "", fmt.Errorf("save shortened URL: %w", err)
	}

	shortURL := fmt.Sprintf("%s/%s", s.baseURL, id)
	if duplicate {
		return shortURL, ErrURLAlreadyExists
	}

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

	savedRecords, err := s.repo.SaveBatch(ctx, records)
	if err != nil {
		return nil, fmt.Errorf("save shortened URL batch: %w", err)
	}

	for i, record := range savedRecords {
		results[i].ShortURL = fmt.Sprintf("%s/%s", s.baseURL, record.ID)
	}

	return results, nil
}

func (s *URLService) saveWithUniqueID(ctx context.Context, originalURL string) (string, bool, error) {
	const maxAttempts = 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		id, err := generateID(8)
		if err != nil {
			return "", false, err
		}

		result, err := s.repo.SaveURL(ctx, id, originalURL)
		if err != nil {
			if errors.Is(err, repository.ErrURLIDConflict) {
				continue
			}

			return "", false, fmt.Errorf("save URL: %w", err)
		}

		return result.ID, result.Duplicate, nil
	}

	return "", false, fmt.Errorf("failed to generate unique id after %d attempts", maxAttempts)
}

func (s *URLService) GetURLByID(ctx context.Context, id string) (string, error) {
	originalURL, err := s.repo.GetByID(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrURLNotFound):
			return "", ErrURLNotFound

		case errors.Is(err, repository.ErrURLDeleted):
			return "", ErrURLDeleted

		default:
			return "", fmt.Errorf("get URL by ID: %w", err)
		}
	}

	return originalURL, nil
}

func (s *URLService) GetUserURLs(ctx context.Context) ([]UserURL, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, ErrUnauthorized
	}

	userRepo, ok := s.repo.(UserURLRepository)
	if !ok {
		return nil, errors.New("repository does not support user URLs")
	}

	records, err := userRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user URLs: %w", err)
	}

	urls := make([]UserURL, 0, len(records))
	for _, record := range records {
		urls = append(urls, UserURL{
			ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, record.ID),
			OriginalURL: record.OriginalURL,
		})
	}

	return urls, nil
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
