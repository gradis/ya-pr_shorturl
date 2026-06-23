package repository

import (
	"path/filepath"
	"testing"
)

func TestFileRepository_SaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "storage.json")

	repo, err := NewFileRepository(path)
	if err != nil {
		t.Fatal(err)
	}

	saved, err := repo.SaveIfNotExists("abc123", "https://practicum.yandex.ru")
	if err != nil {
		t.Fatal(err)
	}

	if !saved {
		t.Fatal("expected url to be saved")
	}

	repo2, err := NewFileRepository(path)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := repo2.GetByID("abc123")
	if !ok {
		t.Fatal("expected url to be restored")
	}

	want := "https://practicum.yandex.ru"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
