package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockURLService struct {
	addUrlFunc     func(originalURL string) (string, error)
	getUrlByIDFunc func(id string) (string, error)
}

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

	handler := NewURLHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("  https://example.com  "))
	rec := httptest.NewRecorder()

	handler.handlePost(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, res.StatusCode)
	}

	if !serviceCalled {
		t.Fatal("expected service.AddUrl to be called")
	}

	if got := res.Header.Get("Content-Type"); got != "text/plain" {
		t.Fatalf("expected Content-Type %q, got %q", "text/plain", got)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if got := string(body); got != "http://localhost:8080/abc123" {
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

	handler := NewURLHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/abc", strings.NewReader("https://example.com"))
	rec := httptest.NewRecorder()

	handler.handlePost(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}
}

func TestURLHandler_handlePost_EmptyBody(t *testing.T) {
	service := &mockURLService{
		addUrlFunc: func(originalURL string) (string, error) {
			t.Fatal("service.AddUrl should not be called")
			return "", nil
		},
	}

	handler := NewURLHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("   \n\t  "))
	rec := httptest.NewRecorder()

	handler.handlePost(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
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

	handler := NewURLHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	rec := httptest.NewRecorder()

	handler.handlePost(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
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

	handler := NewURLHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/", errorReader{})
	rec := httptest.NewRecorder()

	handler.handlePost(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}
}

func TestURLHandler_handleGet_Success(t *testing.T) {
	service := &mockURLService{
		getUrlByIDFunc: func(id string) (string, error) {
			if id != "abc123" {
				t.Fatalf("expected id %q, got %q", "abc123", id)
			}

			return "https://example.com", nil
		},
	}

	h := NewURLHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rec := httptest.NewRecorder()

	h.handleGet(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected status %d, got %d", http.StatusTemporaryRedirect, res.StatusCode)
	}

	if got := res.Header.Get("Location"); got != "https://example.com" {
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

	h := NewURLHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.handleGet(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
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

	h := NewURLHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rec := httptest.NewRecorder()

	h.handleGet(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}
}
