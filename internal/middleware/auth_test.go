package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/auth"
)

func TestAuthenticationIssuesAndAcceptsCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "test-signing-secret"

	router := gin.New()
	router.Use(Authentication(secret))
	router.GET("/", func(c *gin.Context) {
		userID, ok := auth.UserIDFromContext(c.Request.Context())
		if !ok {
			t.Fatal("expected authenticated user in request context")
		}
		c.String(http.StatusOK, userID)
	})

	firstRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, firstRequest)

	cookies := firstResponse.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one authentication cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != auth.CookieName {
		t.Fatalf("expected cookie %q, got %q", auth.CookieName, cookie.Name)
	}

	userID, err := auth.Verify(cookie.Value, []byte(secret))
	if err != nil {
		t.Fatalf("expected a valid signed cookie: %v", err)
	}
	if userID != firstResponse.Body.String() {
		t.Fatalf("expected cookie user ID %q, got %q", firstResponse.Body.String(), userID)
	}

	secondRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	secondRequest.AddCookie(cookie)
	secondResponse := httptest.NewRecorder()
	router.ServeHTTP(secondResponse, secondRequest)

	if got := secondResponse.Header().Get("Set-Cookie"); got != "" {
		t.Fatalf("did not expect a replacement cookie, got %q", got)
	}
	if secondResponse.Body.String() != userID {
		t.Fatalf("expected the same user ID %q, got %q", userID, secondResponse.Body.String())
	}
}

func TestAuthenticationReplacesInvalidCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "test-signing-secret"

	router := gin.New()
	router.Use(Authentication(secret))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{
		Name:  auth.CookieName,
		Value: "forged-token",
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected a replacement cookie, got %d cookies", len(cookies))
	}
	if _, err := auth.Verify(cookies[0].Value, []byte(secret)); err != nil {
		t.Fatalf("expected a valid replacement cookie: %v", err)
	}
}
