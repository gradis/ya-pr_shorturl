package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
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

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
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
