package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"go.uber.org/zap"
)

const defaultBaseURL = "http://localhost:8080"

var (
	// ErrURLNotFound указывает, что короткий ID отсутствует в хранилище.
	ErrURLNotFound = errors.New("url not found")
	// ErrURLDeleted указывает, что ссылка помечена как удалённая.
	ErrURLDeleted = errors.New("url deleted")
	// ErrInvalidURL указывает, что оригинальный URL не содержит корректные схему и хост.
	ErrInvalidURL = errors.New("invalid URL")
	// ErrInvalidURLIDs указывает, что после очистки не осталось ID для удаления.
	ErrInvalidURLIDs = errors.New("invalid URL ids")
	// ErrURLAlreadyExists сигнализирует о повторном сокращении оригинального URL.
	ErrURLAlreadyExists = errors.New("URL already exists")
	// ErrUnauthorized указывает на отсутствие пользователя в контексте.
	ErrUnauthorized = errors.New("user is not authenticated")
)

// URLRepository задаёт контракт хранения ссылок для URLService.
// SaveBatch возвращает записи в порядке входного пакета с фактически сохранёнными ID.
type URLRepository interface {
	// SaveURL сохраняет URL или возвращает ID существующей записи.
	SaveURL(ctx context.Context, id string, originalURL string) (repository.URLSaveResult, error)
	// SaveBatch сохраняет пакет и возвращает записи с фактическими ID.
	SaveBatch(ctx context.Context, records []repository.URLRecord) ([]repository.URLRecord, error)
	// GetByID находит оригинальный URL по короткому ID.
	GetByID(ctx context.Context, id string) (string, error)
	// GetByUserID возвращает записи указанного пользователя.
	GetByUserID(ctx context.Context, userID string) ([]repository.URLRecord, error)
	// DeleteBatch помечает ссылки как удалённые с проверкой владельца.
	DeleteBatch(ctx context.Context, records []repository.URLDeleteRecord) error
}

// URLService реализует сокращение URL и управление ссылками пользователя.
// Конструктор запускает фоновое удаление; после работы необходимо вызвать Close.
type URLService struct {
	repo    URLRepository
	baseURL string
	logg    *zap.Logger

	deleteQueue chan deleteRequest
	deleteWG    sync.WaitGroup
	closeOnce   sync.Once
}

// BatchURL содержит оригинальный URL и идентификатор сопоставления в пакете.
type BatchURL struct {
	// CorrelationID — идентификатор сопоставления запроса с результатом.
	CorrelationID string
	// OriginalURL — оригинальный URL.
	OriginalURL string
}

// BatchURLResult связывает идентификатор запроса с полученным коротким URL.
type BatchURLResult struct {
	// CorrelationID — идентификатор сопоставления запроса с результатом.
	CorrelationID string
	// ShortURL — полный короткий URL, включая базовый адрес.
	ShortURL string
}

// UserURL представляет пару короткого и оригинального URL пользователя.
type UserURL struct {
	// ShortURL — полный короткий URL, включая базовый адрес.
	ShortURL string
	// OriginalURL — оригинальный URL.
	OriginalURL string
}

// NewURLService создаёт сервис и запускает обработчик очереди удаления.
// Пустой baseURL заменяется http://localhost:8080, завершающие слеши удаляются.
// При nil logg используется журнал без вывода. repo должен быть ненулевым.
func NewURLService(
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

// AddURL проверяет и сохраняет оригинальный URL, возвращая короткую ссылку.
// При повторном URL возвращает существующую ссылку и ErrURLAlreadyExists.
// Владелец новой записи берётся из контекста, если пользователь определён.
func (s *URLService) AddURL(ctx context.Context, originalURL string) (string, error) {
	if !isValidURL(originalURL) {
		return "", ErrInvalidURL
	}

	id, duplicate, err := s.saveWithUniqueID(ctx, originalURL)
	if err != nil {
		return "", fmt.Errorf("save shortened URL: %w", err)
	}

	shortURL := s.baseURL + "/" + id
	if duplicate {
		return shortURL, ErrURLAlreadyExists
	}

	return shortURL, nil
}

// AddBatchURLs сокращает пакет и возвращает результаты в исходном порядке.
// Повторные URL получают существующие ID; некорректный URL даёт ErrInvalidURL.
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
		})
	}

	savedRecords, err := s.repo.SaveBatch(ctx, records)
	if err != nil {
		return nil, fmt.Errorf("save shortened URL batch: %w", err)
	}

	for i, record := range savedRecords {
		results[i].ShortURL = s.baseURL + "/" + record.ID
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

// GetURLByID возвращает оригинальный URL по короткому ID.
// Отсутствующий ID даёт ErrURLNotFound, удалённая ссылка — ErrURLDeleted.
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

// GetUserURLs возвращает ссылки пользователя из контекста.
// При отсутствии пользователя возвращает ErrUnauthorized; порядок зависит от хранилища.
func (s *URLService) GetUserURLs(ctx context.Context) ([]UserURL, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, ErrUnauthorized
	}

	records, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user URLs: %w", err)
	}

	urls := make([]UserURL, 0, len(records))
	for _, record := range records {
		urls = append(urls, UserURL{
			ShortURL:    s.baseURL + "/" + record.ID,
			OriginalURL: record.OriginalURL,
		})
	}

	return urls, nil
}

// DeleteUserURLs удаляет пустые и повторные ID и ставит запрос в очередь.
// Требует пользователя в контексте. Успешный возврат означает принятие запроса,
// а не завершение удаления. При отмене контекста может вернуть его ошибку.
// Не следует вызывать после Close или одновременно с ним.
func (s *URLService) DeleteUserURLs(
	ctx context.Context,
	ids []string,
) error {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return ErrUnauthorized
	}

	cleanIDs := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		if _, exists := seen[id]; exists {
			continue
		}

		seen[id] = struct{}{}
		cleanIDs = append(cleanIDs, id)
	}

	if len(cleanIDs) == 0 {
		return ErrInvalidURLIDs
	}

	request := deleteRequest{
		UserID: userID,
		IDs:    cleanIDs,
	}

	select {
	case s.deleteQueue <- request:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func generateID(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// Reject the upper eight values so every alphabet symbol has exactly
	// four possible byte values. Modulo without rejection would bias IDs.
	const limit = 256 - 256%len(alphabet)
	result := make([]byte, length)
	var randomBytes [32]byte
	for written := 0; written < len(result); {
		if _, err := rand.Read(randomBytes[:]); err != nil {
			return "", err
		}
		for _, value := range randomBytes {
			if int(value) >= limit {
				continue
			}
			result[written] = alphabet[int(value)%len(alphabet)]
			written++
			if written == len(result) {
				break
			}
		}
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
