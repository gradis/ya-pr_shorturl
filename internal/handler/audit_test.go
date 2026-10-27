package handler

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/audit"
	"github.com/gradis/ya-pr_shorturl/internal/auth"
	"github.com/gradis/ya-pr_shorturl/internal/service"
)

type auditRecorder struct{ events []audit.Event }

func (r *auditRecorder) Notify(_ context.Context, e audit.Event) { r.events = append(r.events, e) }

func TestHandlerAudit(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body, action string
		serviceErr                       error
		status                           int
	}{
		{"text", "POST", "/", "https://example.com", "shorten", nil, 201},
		{"json", "POST", "/api/shorten", `{"url":"https://example.com"}`, "shorten", nil, 201},
		{"follow", "GET", "/abc", "", "follow", nil, 307},
		{"bad body", "POST", "/", "", "", nil, 400},
		{"bad json", "POST", "/api/shorten", `{`, "", nil, 400},
		{"conflict text", "POST", "/", "https://example.com", "", service.ErrURLAlreadyExists, 409},
		{"conflict json", "POST", "/api/shorten", `{"url":"https://example.com"}`, "", service.ErrURLAlreadyExists, 409},
		{"missing", "GET", "/abc", "", "", service.ErrURLNotFound, 404},
		{"deleted", "GET", "/abc", "", "", service.ErrURLDeleted, 410},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := &auditRecorder{}
			svc := &mockURLService{addURLFunc: func(context.Context, string) (string, error) { return "http://localhost/abc", tc.serviceErr }, getURLByIDFunc: func(context.Context, string) (string, error) { return "https://example.com", tc.serviceErr }}
			router := gin.New()
			NewURLHandler(svc, nil, recorder).RegisterRoutes(router)
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req = req.WithContext(auth.WithUserID(req.Context(), "user-123"))
			req.Header.Set("Content-Type", "application/json")
			before := time.Now().Unix()
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("status = %d", response.Code)
			}
			if tc.action == "" {
				if len(recorder.events) != 0 {
					t.Fatal("unexpected audit event")
				}
				return
			}
			if len(recorder.events) != 1 {
				t.Fatalf("events = %d", len(recorder.events))
			}
			e := recorder.events[0]
			if e.Action != tc.action || e.URL != "https://example.com" || e.UserID != "user-123" || e.TS < before || e.TS > time.Now().Unix() {
				t.Fatalf("event = %+v", e)
			}
		})
	}
}
