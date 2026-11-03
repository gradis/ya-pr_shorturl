package repository

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestGetByUserIDFiltersAndReturnsCopies(t *testing.T) {
	type userRepository interface {
		SaveBatch(context.Context, []URLRecord) ([]URLRecord, error)
		GetByUserID(context.Context, string) ([]URLRecord, error)
	}
	for _, name := range []string{"memory", "file"} {
		t.Run(name, func(t *testing.T) {
			var repo userRepository
			if name == "memory" {
				repo = NewMemoryRepository()
			} else {
				fileRepo, err := NewFileRepository(filepath.Join(t.TempDir(), "urls.jsonl"))
				if err != nil {
					t.Fatal(err)
				}
				repo = fileRepo
			}
			ctx := context.Background()
			records := []URLRecord{
				{ID: "one", OriginalURL: "https://example.com/one", UserID: "owner"},
				{ID: "two", OriginalURL: "https://example.com/two", UserID: "other"},
				{ID: "three", OriginalURL: "https://example.com/three", UserID: "owner"},
			}
			if _, err := repo.SaveBatch(ctx, records); err != nil {
				t.Fatal(err)
			}
			got, err := repo.GetByUserID(ctx, "owner")
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 2 {
				t.Fatalf("got %d records", len(got))
			}
			seen := make(map[string]bool)
			for _, record := range got {
				if record.UserID != "owner" {
					t.Fatalf("other user's record: %+v", record)
				}
				seen[record.ID] = true
			}
			if !seen["one"] || !seen["three"] {
				t.Fatalf("IDs = %v", seen)
			}
			got[0].OriginalURL = "modified"
			again, err := repo.GetByUserID(ctx, "owner")
			if err != nil {
				t.Fatal(err)
			}
			for _, record := range again {
				if record.OriginalURL == "modified" {
					t.Fatal("caller modified stored record")
				}
			}
			empty, err := repo.GetByUserID(ctx, "missing")
			if err != nil || len(empty) != 0 {
				t.Fatalf("missing user = %+v, %v", empty, err)
			}
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := repo.GetByUserID(canceled, "owner"); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation = %v", err)
			}
		})
	}
}
