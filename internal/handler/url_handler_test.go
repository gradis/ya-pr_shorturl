package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/middleware"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"github.com/gradis/ya-pr_shorturl/internal/service"
)

type mockURLService struct {
	addURLFunc     func(originalURL string) (string, error)
	getURLByIDFunc func(id string) (string, error)
}

var _ URLService = (*mockURLService)(nil)

func (m *mockURLService) AddURL(originalURL string) (string, error) {
	if m.addURLFunc != nil {
		return m.addURLFunc(originalURL)
	}

	return "", nil
}

func (m *mockURLService) GetURLByID(id string) (string, error) {
	if m.getURLByIDFunc != nil {
		return m.getURLByIDFunc(id)
	}

	return "", nil
}

func newTestRouter(service URLService) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.HandleMethodNotAllowed = true

	h := NewURLHandler(service)
	h.RegisterRoutes(router)

	router.NoRoute(func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad request")
	})

	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad request")
	})

	return router
}

func TestURLHandler_PostSuccess(t *testing.T) {
	serviceCalled := false

	service := &mockURLService{
		addURLFunc: func(originalURL string) (string, error) {
			serviceCalled = true

			if originalURL != "https://example.com" {
				t.Fatalf("expected originalURL %q, got %q", "https://example.com", originalURL)
			}

			return "http://localhost:8080/abc123", nil
		},
	}

	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("  https://example.com  "))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	if !serviceCalled {
		t.Fatal("expected service.AddUrl to be called")
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("expected Content-Type text/plain, got %q", got)
	}

	if got := rec.Body.String(); got != "http://localhost:8080/abc123" {
		t.Fatalf("expected body %q, got %q", "http://localhost:8080/abc123", got)
	}
}

func TestURLHandler_PostBadPath(t *testing.T) {
	service := &mockURLService{
		addURLFunc: func(originalURL string) (string, error) {
			t.Fatal("service.AddUrl should not be called")
			return "", nil
		},
	}

	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodPost, "/abc", strings.NewReader("https://example.com"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestURLHandler_PostEmptyBody(t *testing.T) {
	service := &mockURLService{
		addURLFunc: func(originalURL string) (string, error) {
			t.Fatal("service.AddUrl should not be called")
			return "", nil
		},
	}

	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("   \n\t  "))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestURLHandler_PostServiceError(t *testing.T) {
	service := &mockURLService{
		addURLFunc: func(originalURL string) (string, error) {
			if originalURL != "https://example.com" {
				t.Fatalf("expected originalURL %q, got %q", "https://example.com", originalURL)
			}

			return "", errors.New("service error")
		},
	}

	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestURLHandler_GetSuccess(t *testing.T) {
	serviceCalled := false

	service := &mockURLService{
		getURLByIDFunc: func(id string) (string, error) {
			serviceCalled = true

			if id != "abc123" {
				t.Fatalf("expected id %q, got %q", "abc123", id)
			}

			return "https://example.com", nil
		},
	}

	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected status %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}

	if !serviceCalled {
		t.Fatal("expected service.GetUrlByID to be called")
	}

	if got := rec.Header().Get("Location"); got != "https://example.com" {
		t.Fatalf("expected Location %q, got %q", "https://example.com", got)
	}
}

func TestURLHandler_GetRootPath(t *testing.T) {
	service := &mockURLService{
		getURLByIDFunc: func(id string) (string, error) {
			t.Fatal("service.GetUrlByID should not be called")
			return "", nil
		},
	}

	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestURLHandler_GetServiceError(t *testing.T) {
	service := &mockURLService{
		getURLByIDFunc: func(id string) (string, error) {
			if id != "abc123" {
				t.Fatalf("expected id %q, got %q", "abc123", id)
			}

			return "", errors.New("service error")
		},
	}

	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestURLHandler_BadMethod(t *testing.T) {
	service := &mockURLService{}

	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleShortenJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(repo, "http://localhost:8080")
	h := NewURLHandler(svc)

	r := gin.New()
	h.RegisterRoutes(r)

	body := `{"url":"https://practicum.yandex.ru"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected Content-Type application/json, got %q", contentType)
	}

	var resp struct {
		Result string `json:"result"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Result == "" {
		t.Fatal("expected non-empty result")
	}

	if !strings.HasPrefix(resp.Result, "http://localhost:8080/") {
		t.Fatalf("unexpected short url: %q", resp.Result)
	}
}

func TestHandleShortenJSON_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(repo, "http://localhost:8080")
	h := NewURLHandler(svc)

	r := gin.New()
	h.RegisterRoutes(r)

	body := `{"url":`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGzipResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(repo, "http://localhost:8080")
	h := NewURLHandler(svc)

	r := gin.New()
	r.Use(middleware.Gzip())
	h.RegisterRoutes(r)

	reqBody := `{"url":"https://practicum.yandex.ru"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip encoding, got %q", rec.Header().Get("Content-Encoding"))
	}

	gzReader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	body, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("failed to read gzip response: %v", err)
	}

	var resp struct {
		Result string `json:"result"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestGzipRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(repo, "http://localhost:8080")
	h := NewURLHandler(svc)

	r := gin.New()
	r.Use(middleware.Gzip())
	h.RegisterRoutes(r)

	var buf bytes.Buffer

	gzWriter := gzip.NewWriter(&buf)
	_, err := gzWriter.Write([]byte(`{"url":"https://practicum.yandex.ru"}`))
	if err != nil {
		t.Fatalf("failed to write gzip body: %v", err)
	}

	if err := gzWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}
