package service

import (
	"context"
	"strings"
	"testing"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
)

func TestGenerateID(t *testing.T) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, length := range []int{0, 1, 8, 32, 65} {
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id, err := generateID(length)
			if err != nil {
				t.Fatal(err)
			}
			if len(id) != length {
				t.Fatalf("length = %d, want %d", len(id), length)
			}
			if strings.Trim(id, alphabet) != "" {
				t.Fatalf("invalid ID %q", id)
			}
			if length >= 8 && seen[id] {
				t.Fatalf("duplicate ID %q", id)
			}
			seen[id] = true
		}
	}
}

func TestBatchUsesSavedIDs(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := auth.WithUserID(context.Background(), "user")
	const existing = "https://example.com/existing"
	if _, err := repo.SaveURL(ctx, "existing", existing); err != nil {
		t.Fatal(err)
	}
	svc := NewURLService(repo, "http://localhost:8080/", nil)
	defer svc.Close()
	results, err := svc.AddBatchURLs(ctx, []BatchURL{{CorrelationID: "old", OriginalURL: existing}, {CorrelationID: "new", OriginalURL: "https://example.com/new"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].CorrelationID != "old" || results[0].ShortURL != "http://localhost:8080/existing" || results[1].CorrelationID != "new" {
		t.Fatalf("results = %+v", results)
	}
	id := strings.TrimPrefix(results[1].ShortURL, "http://localhost:8080/")
	if len(id) != 8 {
		t.Fatalf("new ID = %q", id)
	}
	got, err := svc.GetURLByID(ctx, id)
	if err != nil || got != "https://example.com/new" {
		t.Fatalf("follow = %q, %v", got, err)
	}
	urls, err := svc.GetUserURLs(ctx)
	if err != nil || len(urls) != 2 {
		t.Fatalf("user URLs = %+v, %v", urls, err)
	}
	for _, url := range urls {
		if !strings.HasPrefix(url.ShortURL, "http://localhost:8080/") {
			t.Fatalf("invalid short URL %q", url.ShortURL)
		}
	}
}
