package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRequestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zap.InfoLevel)
	logg := zap.New(core)

	router := gin.New()
	router.Use(RequestLogger(logg))
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expect status code %d, but got %d", http.StatusOK, rec.Code)
	}
	logs := recorded.All()

	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, but got %d", len(logs))
	}

	fields := logs[0].ContextMap()

	if fields["method"] != http.MethodGet {
		t.Fatalf("expected method %s, but got %s", http.MethodGet, fields["method"])
	}

	if fields["uri"] != "/ping" {
		t.Fatalf("expected uri %s, but got %s", "/ping", fields["uri"])
	}

	if fields["status"] != int64(http.StatusOK) && fields["status"] != http.StatusOK {
		t.Fatalf("expected status code %d, but got %d", http.StatusOK, fields["status"])
	}

	if _, ok := fields["duration"]; !ok {
		t.Fatal("expected duration field in log")
	}

	if _, ok := fields["size"]; !ok {
		t.Fatal("expected size field in log")
	}
}
