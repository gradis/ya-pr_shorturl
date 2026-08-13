package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
)

func TestMemoryRepository_DeleteBatch(t *testing.T) {
	repo := NewMemoryRepository()

	saveCtx := auth.WithUserID(
		context.Background(),
		"user-123",
	)

	_, err := repo.SaveURL(
		saveCtx,
		"abc123",
		"https://example.com",
	)
	if err != nil {
		t.Fatalf("SaveURL returned error: %v", err)
	}

	err = repo.DeleteBatch(
		context.Background(),
		[]URLDeleteRecord{
			{
				ID:     "abc123",
				UserID: "user-123",
			},
		},
	)
	if err != nil {
		t.Fatalf("DeleteBatch returned error: %v", err)
	}

	_, err = repo.GetByID(
		context.Background(),
		"abc123",
	)

	if !errors.Is(err, ErrURLDeleted) {
		t.Fatalf(
			"expected ErrURLDeleted, got %v",
			err,
		)
	}
}

func TestMemoryRepository_DeleteBatchDoesNotDeleteForeignURL(
	t *testing.T,
) {
	repo := NewMemoryRepository()

	saveCtx := auth.WithUserID(
		context.Background(),
		"owner-user",
	)

	_, err := repo.SaveURL(
		saveCtx,
		"abc123",
		"https://example.com",
	)
	if err != nil {
		t.Fatalf("SaveURL returned error: %v", err)
	}

	err = repo.DeleteBatch(
		context.Background(),
		[]URLDeleteRecord{
			{
				ID:     "abc123",
				UserID: "other-user",
			},
		},
	)
	if err != nil {
		t.Fatalf("DeleteBatch returned error: %v", err)
	}

	got, err := repo.GetByID(
		context.Background(),
		"abc123",
	)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}

	const want = "https://example.com"

	if got != want {
		t.Fatalf(
			"expected URL %q, got %q",
			want,
			got,
		)
	}
}
