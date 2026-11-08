package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
)

func benchmarkService(b *testing.B) (*URLService, context.Context, []BatchURL) {
	b.Helper()
	repo := repository.NewMemoryRepository()
	ctx := auth.WithUserID(context.Background(), "benchmark-user")
	records := make([]repository.URLRecord, 1000)
	batch := make([]BatchURL, 10)
	for i := range records {
		user := "other-user"
		if i%10 == 0 {
			user = "benchmark-user"
		}
		records[i] = repository.URLRecord{ID: fmt.Sprintf("id%06d", i), OriginalURL: fmt.Sprintf("https://example.com/articles/%d?source=benchmark", i), UserID: user}
		if i < len(batch) {
			batch[i] = BatchURL{CorrelationID: fmt.Sprint(i), OriginalURL: records[i].OriginalURL}
		}
	}
	if _, err := repo.SaveBatch(ctx, records); err != nil {
		b.Fatal(err)
	}
	svc := NewURLService(repo, "http://localhost:8080", nil)
	b.Cleanup(svc.Close)
	return svc, ctx, batch
}

func BenchmarkGenerateID(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := generateID(8); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkURLService(b *testing.B) {
	b.Run("ShortenExisting", func(b *testing.B) {
		svc, ctx, batch := benchmarkService(b)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if _, err := svc.AddURL(ctx, batch[0].OriginalURL); !errors.Is(err, ErrURLAlreadyExists) {
				b.Fatal(err)
			}
		}
	})
	b.Run("Batch10", func(b *testing.B) {
		svc, ctx, batch := benchmarkService(b)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if _, err := svc.AddBatchURLs(ctx, batch); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Follow", func(b *testing.B) {
		svc, ctx, _ := benchmarkService(b)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if _, err := svc.GetURLByID(ctx, "id000000"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("List100", func(b *testing.B) {
		svc, ctx, _ := benchmarkService(b)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if _, err := svc.GetUserURLs(ctx); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Fixed mix: one duplicate shorten, a batch of 10, a follow, and a list of 100.
// The seeded repository stays bounded, making before/after profiles comparable.
func BenchmarkProfileWorkload(b *testing.B) {
	svc, ctx, batch := benchmarkService(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := svc.AddURL(ctx, batch[0].OriginalURL); !errors.Is(err, ErrURLAlreadyExists) {
			b.Fatal(err)
		}
		if _, err := svc.AddBatchURLs(ctx, batch); err != nil {
			b.Fatal(err)
		}
		if _, err := svc.GetURLByID(ctx, "id000000"); err != nil {
			b.Fatal(err)
		}
		if _, err := svc.GetUserURLs(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
