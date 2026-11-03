package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"github.com/gradis/ya-pr_shorturl/internal/service"
)

func BenchmarkFollowHandler(b *testing.B) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryRepository()
	if _, err := repo.SaveURL(context.Background(), "abc12345", "https://example.com/articles/123?source=benchmark"); err != nil {
		b.Fatal(err)
	}
	svc := service.NewURLService(repo, "http://localhost:8080", nil)
	b.Cleanup(svc.Close)
	router := gin.New()
	NewURLHandler(svc, nil).RegisterRoutes(router)
	request := httptest.NewRequest("GET", "/abc12345", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != 307 {
			b.Fatal(response.Code)
		}
	}
}
