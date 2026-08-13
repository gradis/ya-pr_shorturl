package service

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"go.uber.org/zap"
)

type deleteRepositoryMock struct {
	mu      sync.Mutex
	deleted []repository.URLDeleteRecord
}

func (m *deleteRepositoryMock) SaveURL(
	ctx context.Context,
	id string,
	originalURL string,
) (repository.URLSaveResult, error) {
	return repository.URLSaveResult{
		ID: id,
	}, nil
}

func (m *deleteRepositoryMock) SaveBatch(
	ctx context.Context,
	records []repository.URLRecord,
) ([]repository.URLRecord, error) {
	return records, nil
}

func (m *deleteRepositoryMock) GetByID(
	ctx context.Context,
	id string,
) (string, error) {
	return "", repository.ErrURLNotFound
}

func (m *deleteRepositoryMock) GetByUserID(
	ctx context.Context,
	userID string,
) ([]repository.URLRecord, error) {
	return nil, nil
}

func (m *deleteRepositoryMock) DeleteBatch(
	ctx context.Context,
	records []repository.URLDeleteRecord,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleted = append(
		m.deleted,
		records...,
	)

	return nil
}

func (m *deleteRepositoryMock) deletedRecords() []repository.URLDeleteRecord {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append(
		[]repository.URLDeleteRecord(nil),
		m.deleted...,
	)
}

func TestURLService_DeleteUserURLsUnauthorized(t *testing.T) {
	repo := &deleteRepositoryMock{}

	svc := NewURLService(
		repo,
		"http://localhost:8080",
		zap.NewNop(),
	)
	defer svc.Close()

	err := svc.DeleteUserURLs(
		context.Background(),
		[]string{"abc123"},
	)

	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf(
			"expected ErrUnauthorized, got %v",
			err,
		)
	}
}

func TestURLService_DeleteUserURLsInvalidIDs(t *testing.T) {
	repo := &deleteRepositoryMock{}

	svc := NewURLService(
		repo,
		"http://localhost:8080",
		zap.NewNop(),
	)
	defer svc.Close()

	ctx := auth.WithUserID(
		context.Background(),
		"user-123",
	)

	err := svc.DeleteUserURLs(
		ctx,
		[]string{"", " ", "\t"},
	)

	if !errors.Is(err, ErrInvalidURLIDs) {
		t.Fatalf(
			"expected ErrInvalidURLIDs, got %v",
			err,
		)
	}
}

func TestURLService_DeleteUserURLsQueuesRecords(t *testing.T) {
	repo := &deleteRepositoryMock{}

	svc := NewURLService(
		repo,
		"http://localhost:8080",
		zap.NewNop(),
	)

	ctx := auth.WithUserID(
		context.Background(),
		"user-123",
	)

	err := svc.DeleteUserURLs(
		ctx,
		[]string{
			" first-id ",
			"",
			"second-id",
			"first-id",
		},
	)
	if err != nil {
		t.Fatalf(
			"DeleteUserURLs returned error: %v",
			err,
		)
	}

	svc.Close()

	got := repo.deletedRecords()

	want := []repository.URLDeleteRecord{
		{
			ID:     "first-id",
			UserID: "user-123",
		},
		{
			ID:     "second-id",
			UserID: "user-123",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"expected records %#v, got %#v",
			want,
			got,
		)
	}
}
