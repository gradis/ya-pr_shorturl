package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/auth"
	"github.com/gradis/ya-pr_shorturl/internal/handler"
	"github.com/gradis/ya-pr_shorturl/internal/middleware"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"github.com/gradis/ya-pr_shorturl/internal/service"
	"go.uber.org/zap"
)

const exampleSecret = "example-signing-secret"

// exampleApp uses the real HTTP handlers, service and in-memory repository.
// A signed cookie keeps the same user across requests without starting a server.
type exampleApp struct {
	router     *gin.Engine
	service    *service.URLService
	repository *repository.MemoryRepository
	cookie     *http.Cookie
}

func newExampleApp() *exampleApp {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryRepository()
	svc := service.NewURLService(repo, "http://short.example", zap.NewNop())
	router := gin.New()
	router.Use(middleware.Authentication(exampleSecret))
	handler.NewURLHandler(svc, zap.NewNop()).RegisterRoutes(router)
	handler.NewPingHandler(repo, zap.NewNop()).RegisterRoutes(router)
	token, err := auth.Sign("example-user", []byte(exampleSecret))
	if err != nil {
		panic(err)
	}
	return &exampleApp{router: router, service: svc, repository: repo, cookie: &http.Cookie{Name: auth.CookieName, Value: token}}
}

func (app *exampleApp) request(method, path, body, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	req.AddCookie(app.cookie)
	response := httptest.NewRecorder()
	app.router.ServeHTTP(response, req)
	return response
}

func (app *exampleApp) seed(id, originalURL string) {
	ctx := auth.WithUserID(context.Background(), "example-user")
	if _, err := app.repository.SaveURL(ctx, id, originalURL); err != nil {
		panic(err)
	}
}

// ExampleURLHandler_RegisterRoutes_shortenText demonstrates POST / with a plain
// URL body, followed by a duplicate request returning the existing short URL.
func ExampleURLHandler_RegisterRoutes_shortenText() {
	app := newExampleApp()
	defer app.service.Close()

	response := app.request(http.MethodPost, "/", "https://example.com/articles/42", "text/plain")
	fmt.Println(response.Code)
	fmt.Println(strings.HasPrefix(response.Body.String(), "http://short.example/"))

	duplicate := app.request(http.MethodPost, "/", "https://example.com/articles/42", "text/plain")
	fmt.Println(duplicate.Code)
	fmt.Println(duplicate.Body.String() == response.Body.String())
	// Output:
	// 201
	// true
	// 409
	// true
}

// ExampleURLHandler_RegisterRoutes_shortenJSON demonstrates POST /api/shorten
// and decoding the JSON result without depending on a randomly generated ID.
func ExampleURLHandler_RegisterRoutes_shortenJSON() {
	app := newExampleApp()
	defer app.service.Close()

	response := app.request(http.MethodPost, "/api/shorten", `{"url":"https://example.com/articles/42"}`, "application/json")
	var body struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		panic(err)
	}
	fmt.Println(response.Code)
	fmt.Println(strings.HasPrefix(response.Header().Get("Content-Type"), "application/json"))
	fmt.Println(strings.HasPrefix(body.Result, "http://short.example/"))
	// Output:
	// 201
	// true
	// true
}

// ExampleURLHandler_RegisterRoutes_follow demonstrates GET /{id} and the
// Location response header. The recorder does not follow redirects.
func ExampleURLHandler_RegisterRoutes_follow() {
	app := newExampleApp()
	defer app.service.Close()
	app.seed("article42", "https://example.com/articles/42")

	response := app.request(http.MethodGet, "/article42", "", "")
	fmt.Println(response.Code)
	fmt.Println(response.Header().Get("Location"))

	missing := app.request(http.MethodGet, "/missing", "", "")
	fmt.Println(missing.Code)
	// Output:
	// 307
	// https://example.com/articles/42
	// 404
}

// ExampleURLHandler_RegisterRoutes_batch demonstrates POST /api/shorten/batch.
// Correlation IDs link results to input items, including an existing URL.
func ExampleURLHandler_RegisterRoutes_batch() {
	app := newExampleApp()
	defer app.service.Close()
	app.seed("article42", "https://example.com/articles/42")

	response := app.request(http.MethodPost, "/api/shorten/batch", `[
 {"correlation_id":"first","original_url":"https://example.com/articles/42"},
 {"correlation_id":"second","original_url":"https://example.com/articles/43"}
 ]`, "application/json")
	var results []struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &results); err != nil {
		panic(err)
	}
	fmt.Println(response.Code)
	for _, result := range results {
		fmt.Println(result.CorrelationID, strings.HasPrefix(result.ShortURL, "http://short.example/"))
	}
	// Output:
	// 201
	// first true
	// second true
}

// ExampleURLHandler_RegisterRoutes_userURLs demonstrates GET /api/user/urls
// with a signed cookie, both before and after a user has saved a URL.
func ExampleURLHandler_RegisterRoutes_userURLs() {
	app := newExampleApp()
	defer app.service.Close()
	empty := app.request(http.MethodGet, "/api/user/urls", "", "")
	fmt.Println(empty.Code)

	app.seed("article42", "https://example.com/articles/42")
	response := app.request(http.MethodGet, "/api/user/urls", "", "")
	var urls []struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &urls); err != nil {
		panic(err)
	}
	fmt.Println(response.Code)
	for _, url := range urls {
		fmt.Println(url.ShortURL, url.OriginalURL)
	}
	// Output:
	// 204
	// 200
	// http://short.example/article42 https://example.com/articles/42
}

// ExampleURLHandler_RegisterRoutes_delete demonstrates DELETE /api/user/urls.
// HTTP 202 accepts the request asynchronously. Close drains the queue so the
// subsequent GET deterministically observes the deleted URL, without sleeps.
func ExampleURLHandler_RegisterRoutes_delete() {
	app := newExampleApp()
	defer app.service.Close()
	app.seed("article42", "https://example.com/articles/42")

	response := app.request(http.MethodDelete, "/api/user/urls", `["article42"]`, "application/json")
	fmt.Println(response.Code)
	app.service.Close()
	deleted := app.request(http.MethodGet, "/article42", "", "")
	fmt.Println(deleted.Code)
	// Output:
	// 202
	// 410
}

// ExamplePingHandler_RegisterRoutes demonstrates GET /ping. The in-memory
// repository has no external dependency and always passes its health check.
func ExamplePingHandler_RegisterRoutes() {
	app := newExampleApp()
	defer app.service.Close()

	response := app.request(http.MethodGet, "/ping", "", "")
	fmt.Println(response.Code)
	// Output:
	// 200
}
