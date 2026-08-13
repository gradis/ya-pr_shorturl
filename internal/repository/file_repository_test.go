package repository

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
)

func TestFileRepository_SaveAndLoad(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "storage.json")

	repo, err := NewFileRepository(path)
	if err != nil {
		t.Fatal(err)
	}

	result, err := repo.SaveURL(ctx, "abc123", "https://practicum.yandex.ru")
	if err != nil {
		t.Fatal(err)
	}

	if result.ID != "abc123" || result.Duplicate {
		t.Fatal("expected url to be saved")
	}

	repo2, err := NewFileRepository(path)
	if err != nil {
		t.Fatal(err)
	}

	got, err := repo2.GetByID(ctx, "abc123")
	if err != nil {
		t.Fatal("expected url to be restored")
	}

	want := "https://practicum.yandex.ru"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestFileRepository_DeletePersistsAfterReload(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"storage.json",
	)

	repo, err := NewFileRepository(path)
	if err != nil {
		t.Fatalf(
			"NewFileRepository returned error: %v",
			err,
		)
	}

	saveCtx := auth.WithUserID(
		context.Background(),
		"user-123",
	)

	_, err = repo.SaveURL(
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
		t.Fatalf(
			"DeleteBatch returned error: %v",
			err,
		)
	}

	reloadedRepo, err := NewFileRepository(path)
	if err != nil {
		t.Fatalf(
			"reload repository: %v",
			err,
		)
	}

	_, err = reloadedRepo.GetByID(
		context.Background(),
		"abc123",
	)

	if !errors.Is(err, ErrURLDeleted) {
		t.Fatalf(
			"expected ErrURLDeleted after reload, got %v",
			err,
		)
	}
}
