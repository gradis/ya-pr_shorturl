package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/auth"
	"github.com/gradis/ya-pr_shorturl/internal/middleware"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"github.com/gradis/ya-pr_shorturl/internal/service"
)

type mockURLService struct {
	addURLFunc func(
		ctx context.Context,
		originalURL string,
	) (string, error)

	getURLByIDFunc func(
		ctx context.Context,
		id string,
	) (string, error)

	addBatchURLsFunc func(
		ctx context.Context,
		urls []service.BatchURL,
	) ([]service.BatchURLResult, error)

	getUserURLsFunc func(
		ctx context.Context,
	) ([]service.UserURL, error)

	deleteUserURLsFunc func(
		ctx context.Context,
		ids []string,
	) error
}

var _ URLService = (*mockURLService)(nil)

func (m *mockURLService) AddURL(
	ctx context.Context,
	originalURL string,
) (string, error) {
	if m.addURLFunc != nil {
		return m.addURLFunc(ctx, originalURL)
	}

	return "", nil
}

func (m *mockURLService) GetURLByID(
	ctx context.Context,
	id string,
) (string, error) {
	if m.getURLByIDFunc != nil {
		return m.getURLByIDFunc(ctx, id)
	}

	return "", nil
}

func (m *mockURLService) AddBatchURLs(
	ctx context.Context,
	urls []service.BatchURL,
) ([]service.BatchURLResult, error) {
	if m.addBatchURLsFunc != nil {
		return m.addBatchURLsFunc(ctx, urls)
	}

	return nil, nil
}

func (m *mockURLService) GetUserURLs(
	ctx context.Context,
) ([]service.UserURL, error) {
	if m.getUserURLsFunc != nil {
		return m.getUserURLsFunc(ctx)
	}

	return nil, nil
}

func (m *mockURLService) DeleteUserURLs(
	ctx context.Context,
	ids []string,
) error {
	if m.deleteUserURLsFunc != nil {
		return m.deleteUserURLsFunc(ctx, ids)
	}

	return nil
}

func newTestRouter(s URLService) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.HandleMethodNotAllowed = true

	h := NewURLHandler(s)
	h.RegisterRoutes(router)

	router.NoRoute(func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad request")
	})

	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad request")
	})

	return router
}

func newAuthenticatedTestRouter(
	s URLService,
	userID string,
) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.Use(func(c *gin.Context) {
		ctx := auth.WithUserID(
			c.Request.Context(),
			userID,
		)

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	h := NewURLHandler(s)
	h.RegisterRoutes(router)

	return router
}

func TestURLHandler_PostSuccess(t *testing.T) {
	serviceCalled := false

	s := &mockURLService{
		addURLFunc: func(
			ctx context.Context,
			originalURL string,
		) (string, error) {
			serviceCalled = true

			if ctx == nil {
				t.Fatal("expected non-nil context")
			}

			want := "https://example.com"
			if originalURL != want {
				t.Fatalf(
					"expected originalURL %q, got %q",
					want,
					originalURL,
				)
			}

			return "http://localhost:8080/abc123", nil
		},
	}

	router := newTestRouter(s)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("  https://example.com  "),
	)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	if !serviceCalled {
		t.Fatal("expected service.AddURL to be called")
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Fatalf(
			"expected Content-Type text/plain, got %q",
			contentType,
		)
	}

	wantBody := "http://localhost:8080/abc123"
	if got := rec.Body.String(); got != wantBody {
		t.Fatalf("expected body %q, got %q", wantBody, got)
	}
}

func TestURLHandler_PostBadPath(t *testing.T) {
	s := &mockURLService{
		addURLFunc: func(
			ctx context.Context,
			originalURL string,
		) (string, error) {
			t.Fatal("service.AddURL should not be called")
			return "", nil
		},
	}

	router := newTestRouter(s)

	req := httptest.NewRequest(
		http.MethodPost,
		"/abc",
		strings.NewReader("https://example.com"),
	)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestURLHandler_PostEmptyBody(t *testing.T) {
	s := &mockURLService{
		addURLFunc: func(
			ctx context.Context,
			originalURL string,
		) (string, error) {
			t.Fatal("service.AddURL should not be called")
			return "", nil
		},
	}

	router := newTestRouter(s)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("   \n\t  "),
	)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestURLHandler_PostServiceError(t *testing.T) {
	s := &mockURLService{
		addURLFunc: func(
			ctx context.Context,
			originalURL string,
		) (string, error) {
			want := "https://example.com"
			if originalURL != want {
				t.Fatalf(
					"expected originalURL %q, got %q",
					want,
					originalURL,
				)
			}

			return "", errors.New("service error")
		},
	}

	router := newTestRouter(s)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("https://example.com"),
	)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestURLHandler_PostInvalidURL(t *testing.T) {
	s := &mockURLService{
		addURLFunc: func(
			ctx context.Context,
			originalURL string,
		) (string, error) {
			return "", service.ErrInvalidURL
		},
	}

	router := newTestRouter(s)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("invalid-url"),
	)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestURLHandler_PostConflict(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(
		repo,
		"http://localhost:8080",
	)
	router := newTestRouter(svc)

	body := "https://example.com"

	firstReq := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(body),
	)
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)

	if firstRec.Code != http.StatusCreated {
		t.Fatalf(
			"expected first status %d, got %d",
			http.StatusCreated,
			firstRec.Code,
		)
	}

	secondReq := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(body),
	)
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusConflict {
		t.Fatalf(
			"expected second status %d, got %d",
			http.StatusConflict,
			secondRec.Code,
		)
	}

	if secondRec.Body.String() != firstRec.Body.String() {
		t.Fatalf(
			"expected existing short URL %q, got %q",
			firstRec.Body.String(),
			secondRec.Body.String(),
		)
	}
}

func TestURLHandler_GetSuccess(t *testing.T) {
	serviceCalled := false

	s := &mockURLService{
		getURLByIDFunc: func(
			ctx context.Context,
			id string,
		) (string, error) {
			serviceCalled = true

			if ctx == nil {
				t.Fatal("expected non-nil context")
			}

			if id != "abc123" {
				t.Fatalf(
					"expected id %q, got %q",
					"abc123",
					id,
				)
			}

			return "https://example.com", nil
		},
	}

	router := newTestRouter(s)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusTemporaryRedirect,
			rec.Code,
		)
	}

	if !serviceCalled {
		t.Fatal("expected service.GetURLByID to be called")
	}

	if got := rec.Header().Get("Location"); got != "https://example.com" {
		t.Fatalf(
			"expected Location %q, got %q",
			"https://example.com",
			got,
		)
	}
}

func TestURLHandler_GetRootPath(t *testing.T) {
	s := &mockURLService{
		getURLByIDFunc: func(
			ctx context.Context,
			id string,
		) (string, error) {
			t.Fatal("service.GetURLByID should not be called")
			return "", nil
		},
	}

	router := newTestRouter(s)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestURLHandler_GetNotFound(t *testing.T) {
	s := &mockURLService{
		getURLByIDFunc: func(
			ctx context.Context,
			id string,
		) (string, error) {
			return "", service.ErrURLNotFound
		},
	}

	router := newTestRouter(s)

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			rec.Code,
		)
	}
}

func TestURLHandler_GetServiceError(t *testing.T) {
	s := &mockURLService{
		getURLByIDFunc: func(
			ctx context.Context,
			id string,
		) (string, error) {
			if id != "abc123" {
				t.Fatalf(
					"expected id %q, got %q",
					"abc123",
					id,
				)
			}

			return "", errors.New("service error")
		},
	}

	router := newTestRouter(s)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestURLHandler_BadMethod(t *testing.T) {
	router := newTestRouter(&mockURLService{})

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandleShortenJSON(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(
		repo,
		"http://localhost:8080",
	)
	router := newTestRouter(svc)

	body := `{"url":"https://practicum.yandex.ru"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Fatalf(
			"expected Content-Type application/json, got %q",
			contentType,
		)
	}

	var response shortenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Result == "" {
		t.Fatal("expected non-empty result")
	}

	if !strings.HasPrefix(
		response.Result,
		"http://localhost:8080/",
	) {
		t.Fatalf("unexpected short URL: %q", response.Result)
	}
}

func TestHandleShortenJSON_BadRequest(t *testing.T) {
	router := newTestRouter(&mockURLService{})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandleShortenJSON_ServiceError(t *testing.T) {
	s := &mockURLService{
		addURLFunc: func(
			ctx context.Context,
			originalURL string,
		) (string, error) {
			return "", errors.New("service error")
		},
	}

	router := newTestRouter(s)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(
			`{"url":"https://practicum.yandex.ru"}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestHandleShortenJSON_Conflict(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(
		repo,
		"http://localhost:8080",
	)
	router := newTestRouter(svc)

	body := `{"url":"https://practicum.yandex.ru"}`

	firstReq := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(body),
	)
	firstReq.Header.Set("Content-Type", "application/json")
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)

	if firstRec.Code != http.StatusCreated {
		t.Fatalf(
			"expected first status %d, got %d",
			http.StatusCreated,
			firstRec.Code,
		)
	}

	var firstResponse shortenResponse
	if err := json.Unmarshal(firstRec.Body.Bytes(), &firstResponse); err != nil {
		t.Fatalf("failed to unmarshal first response: %v", err)
	}

	secondReq := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(body),
	)
	secondReq.Header.Set("Content-Type", "application/json")
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusConflict {
		t.Fatalf(
			"expected second status %d, got %d",
			http.StatusConflict,
			secondRec.Code,
		)
	}

	var secondResponse shortenResponse
	if err := json.Unmarshal(secondRec.Body.Bytes(), &secondResponse); err != nil {
		t.Fatalf("failed to unmarshal second response: %v", err)
	}

	if secondResponse.Result != firstResponse.Result {
		t.Fatalf(
			"expected existing short URL %q, got %q",
			firstResponse.Result,
			secondResponse.Result,
		)
	}
}

func TestHandleShortenBatch(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(
		repo,
		"http://localhost:8080",
	)
	router := newTestRouter(svc)

	body := `[
		{"correlation_id":"first","original_url":"https://practicum.yandex.ru"},
		{"correlation_id":"second","original_url":"https://example.com"}
	]`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten/batch",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	var response []batchShortenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 response items, got %d", len(response))
	}

	if response[0].CorrelationID != "first" {
		t.Fatalf(
			"expected first correlation_id, got %q",
			response[0].CorrelationID,
		)
	}

	for _, item := range response {
		if !strings.HasPrefix(item.ShortURL, "http://localhost:8080/") {
			t.Fatalf("unexpected short URL: %q", item.ShortURL)
		}
	}
}

func TestHandleShortenBatch_Empty(t *testing.T) {
	router := newTestRouter(&mockURLService{})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten/batch",
		strings.NewReader(`[]`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandleUserURLs_Unauthorized(t *testing.T) {
	router := newTestRouter(&mockURLService{})

	request := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	request.AddCookie(&http.Cookie{
		Name:  auth.CookieName,
		Value: "cookie-without-authenticated-user",
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			response.Code,
		)
	}
}

func TestHandleUserURLs_InvalidCookieIsReplacedAndUnauthorized(t *testing.T) {
	const secret = "test-signing-secret"

	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(repo, "http://localhost:8080")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.Authentication(secret))
	NewURLHandler(svc).RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	request.AddCookie(&http.Cookie{
		Name:  auth.CookieName,
		Value: "forged-token",
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			response.Code,
		)
	}

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected a replacement cookie, got %d", len(cookies))
	}
	if _, err := auth.Verify(cookies[0].Value, []byte(secret)); err != nil {
		t.Fatalf("expected a valid replacement cookie: %v", err)
	}
}

func TestHandleUserURLs_ReturnsOnlyAuthenticatedUserURLs(t *testing.T) {
	const secret = "test-signing-secret"

	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(repo, "http://localhost:8080")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.Authentication(secret))
	NewURLHandler(svc).RegisterRoutes(router)

	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":"https://example.com/user-one"}`),
	)
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, createResponse.Code)
	}

	cookies := createResponse.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected authentication cookie, got %d cookies", len(cookies))
	}

	otherUserRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":"https://example.com/user-two"}`),
	)
	otherUserRequest.Header.Set("Content-Type", "application/json")
	otherUserResponse := httptest.NewRecorder()
	router.ServeHTTP(otherUserResponse, otherUserRequest)

	listRequest := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	listRequest.AddCookie(cookies[0])
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listResponse.Code)
	}

	var response []userURLResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("expected one user URL, got %d", len(response))
	}
	if response[0].OriginalURL != "https://example.com/user-one" {
		t.Fatalf("unexpected original URL %q", response[0].OriginalURL)
	}
	if !strings.HasPrefix(response[0].ShortURL, "http://localhost:8080/") {
		t.Fatalf("unexpected short URL %q", response[0].ShortURL)
	}
}

func TestHandleUserURLs_NoContent(t *testing.T) {
	const secret = "test-signing-secret"

	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(repo, "http://localhost:8080")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.Authentication(secret))
	NewURLHandler(svc).RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			response.Code,
		)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("expected empty response body, got %q", response.Body.String())
	}
}

func TestGzipResponse(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(
		repo,
		"http://localhost:8080",
	)

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.Gzip())

	h := NewURLHandler(svc)
	h.RegisterRoutes(router)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(
			`{"url":"https://practicum.yandex.ru"}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("expected gzip encoding, got %q", got)
	}

	gzipReader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	body, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("failed to read gzip response: %v", err)
	}

	var response shortenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestGzipRequest(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(
		repo,
		"http://localhost:8080",
	)

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.Gzip())

	h := NewURLHandler(svc)
	h.RegisterRoutes(router)

	var buffer bytes.Buffer

	gzipWriter := gzip.NewWriter(&buffer)

	if _, err := gzipWriter.Write(
		[]byte(`{"url":"https://practicum.yandex.ru"}`),
	); err != nil {
		t.Fatalf("failed to write gzip body: %v", err)
	}

	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		&buffer,
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}
}

func TestURLHandler_DeleteUserURLsAccepted(t *testing.T) {
	const userID = "user-123"

	serviceCalled := false

	s := &mockURLService{
		deleteUserURLsFunc: func(
			ctx context.Context,
			ids []string,
		) error {
			serviceCalled = true

			gotUserID, ok := auth.UserIDFromContext(ctx)
			if !ok {
				t.Fatal("expected authenticated user in context")
			}

			if gotUserID != userID {
				t.Fatalf(
					"expected user ID %q, got %q",
					userID,
					gotUserID,
				)
			}

			wantIDs := []string{
				"6qxTVvsy",
				"RTfd56hn",
				"Jlfd67ds",
			}

			if !reflect.DeepEqual(ids, wantIDs) {
				t.Fatalf(
					"expected ids %#v, got %#v",
					wantIDs,
					ids,
				)
			}

			return nil
		},
	}

	router := newAuthenticatedTestRouter(s, userID)

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/user/urls",
		strings.NewReader(
			`["6qxTVvsy","RTfd56hn","Jlfd67ds"]`,
		),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusAccepted,
			response.Code,
		)
	}

	if !serviceCalled {
		t.Fatal("expected DeleteUserURLs to be called")
	}
}

func TestURLHandler_DeleteUserURLsBadJSON(t *testing.T) {
	s := &mockURLService{
		deleteUserURLsFunc: func(
			ctx context.Context,
			ids []string,
		) error {
			t.Fatal("DeleteUserURLs should not be called")
			return nil
		},
	}

	router := newAuthenticatedTestRouter(s, "user-123")

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/user/urls",
		strings.NewReader(`["abc123"`),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.Code,
		)
	}
}

func TestURLHandler_DeleteUserURLsEmptyList(t *testing.T) {
	s := &mockURLService{
		deleteUserURLsFunc: func(
			ctx context.Context,
			ids []string,
		) error {
			t.Fatal("DeleteUserURLs should not be called")
			return nil
		},
	}

	router := newAuthenticatedTestRouter(s, "user-123")

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/user/urls",
		strings.NewReader(`[]`),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.Code,
		)
	}
}

func TestURLHandler_DeleteUserURLsUnauthorized(t *testing.T) {
	s := &mockURLService{
		deleteUserURLsFunc: func(
			ctx context.Context,
			ids []string,
		) error {
			t.Fatal("DeleteUserURLs should not be called")
			return nil
		},
	}

	router := newTestRouter(s)

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/user/urls",
		strings.NewReader(`["abc123"]`),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			response.Code,
		)
	}
}

func TestURLHandler_DeleteUserURLsServiceError(t *testing.T) {
	s := &mockURLService{
		deleteUserURLsFunc: func(
			ctx context.Context,
			ids []string,
		) error {
			return errors.New("delete service error")
		},
	}

	router := newAuthenticatedTestRouter(s, "user-123")

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/user/urls",
		strings.NewReader(`["abc123"]`),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			response.Code,
		)
	}
}

func TestURLHandler_GetDeletedURL(t *testing.T) {
	s := &mockURLService{
		getURLByIDFunc: func(
			ctx context.Context,
			id string,
		) (string, error) {
			if id != "deleted-id" {
				t.Fatalf(
					"expected id %q, got %q",
					"deleted-id",
					id,
				)
			}

			return "", service.ErrURLDeleted
		},
	}

	router := newTestRouter(s)

	request := httptest.NewRequest(
		http.MethodGet,
		"/deleted-id",
		nil,
	)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusGone {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusGone,
			response.Code,
		)
	}
}
