package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
)

const defaultBaseURL = "http://localhost:8080"

var (
	ErrUrlNotFound = errors.New("url not found")
	ErrInvalidURL  = errors.New("invalid URL")
)

type URLRepository interface {
	Save(id string, originalURL string) error
	GetByID(id string) (string, bool)
	Exists(id string) bool
}

type URLService struct {
	repo    URLRepository
	baseURL string
}

func NewURLService(repo URLRepository, baseURL string) *URLService {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	baseURL = strings.TrimRight(baseURL, "/")

	return &URLService{
		repo:    repo,
		baseURL: baseURL,
	}
}

func (s *URLService) AddUrl(originalURL string) (string, error) {
	if !isValidURL(originalURL) {
		return "", ErrInvalidURL
	}

	id, err := s.generateUniqueID()
	if err != nil {
		return "", err
	}

	err = s.repo.Save(id, originalURL)
	if err != nil {
		return "", err
	}

	shortURL := fmt.Sprintf("%s/%s", s.baseURL, id)

	return shortURL, nil
}

func (s *URLService) GetUrlByID(id string) (string, error) {
	originalURL, ok := s.repo.GetByID(id)
	if !ok {
		return "", ErrUrlNotFound
	}

	return originalURL, nil
}

func (s *URLService) generateUniqueID() (string, error) {
	for {
		id, err := generateID(8)
		if err != nil {
			return "", err
		}

		if !s.repo.Exists(id) {
			return id, nil
		}
	}
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
