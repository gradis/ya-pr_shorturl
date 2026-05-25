package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockURLService struct {
	addUrlFunc     func(originalURL string) (string, error)
	getUrlByIDFunc func(id string) (string, error)
}

var _ URLService = (*mockURLService)(nil)

func (m *mockURLService) AddUrl(originalURL string) (string, error) {
	if m.addUrlFunc != nil {
		return m.addUrlFunc(originalURL)
	}

	return "", nil
}

func (m *mockURLService) GetUrlByID(id string) (string, error) {
	if m.getUrlByIDFunc != nil {
		return m.getUrlByIDFunc(id)
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

func TestURLHandler_handlePost_Success(t *testing.T) {
	serviceCalled := false

	service := &mockURLService{
		addUrlFunc: func(originalURL string) (string, error) {
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

func TestURLHandler_handlePost_BadPath(t *testing.T) {
	service := &mockURLService{
		addUrlFunc: func(originalURL string) (string, error) {
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

func TestURLHandler_handlePost_EmptyBody(t *testing.T) {
	service := &mockURLService{
		addUrlFunc: func(originalURL string) (string, error) {
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

func TestURLHandler_handlePost_ServiceError(t *testing.T) {
	service := &mockURLService{
		addUrlFunc: func(originalURL string) (string, error) {
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

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

type errorReader struct{}

func (errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

func TestURLHandler_handlePost_ReadBodyError(t *testing.T) {
	service := &mockURLService{
		addUrlFunc: func(originalURL string) (string, error) {
			t.Fatal("service.AddUrl should not be called")
			return "", nil
		},
	}

	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(errorReader{}))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestURLHandler_handleGet_Success(t *testing.T) {
	serviceCalled := false

	service := &mockURLService{
		getUrlByIDFunc: func(id string) (string, error) {
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

func TestURLHandler_handleGet_RootPath(t *testing.T) {
	service := &mockURLService{
		getUrlByIDFunc: func(id string) (string, error) {
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

func TestURLHandler_handleGet_ServiceError(t *testing.T) {
	service := &mockURLService{
		getUrlByIDFunc: func(id string) (string, error) {
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

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
