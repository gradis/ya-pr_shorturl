package repository

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkMemoryRepository(b *testing.B) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	records := make([]URLRecord, 1000)
	for i := range records {
		records[i] = URLRecord{ID: fmt.Sprint(i), OriginalURL: fmt.Sprintf("https://example.com/articles/%d", i), UserID: fmt.Sprint(i % 10)}
	}
	if _, err := repo.SaveBatch(ctx, records); err != nil {
		b.Fatal(err)
	}
	b.Run("GetByID", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := repo.GetByID(ctx, "123"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("GetByUserID", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := repo.GetByUserID(ctx, "1"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("SaveExisting", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := repo.SaveURL(ctx, "123", records[123].OriginalURL); err != nil {
				b.Fatal(err)
			}
		}
	})
}
