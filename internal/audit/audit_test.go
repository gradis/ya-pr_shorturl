package audit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type observerFunc func(context.Context, Event) error

func (f observerFunc) Update(ctx context.Context, e Event) error { return f(ctx, e) }

func TestPublisherContinuesAfterFailure(t *testing.T) {
	want := Event{TS: 123, Action: "shorten", URL: "https://example.com", UserID: "user"}
	called := false
	p := NewPublisher(nil, observerFunc(func(context.Context, Event) error { return errors.New("failed") }), observerFunc(func(_ context.Context, e Event) error {
		called = true
		if e != want {
			t.Errorf("event = %+v", e)
		}
		return nil
	}))
	p.Notify(context.Background(), want)
	if !called {
		t.Fatal("second observer was not notified")
	}
	NewPublisher(nil).Notify(context.Background(), want)
}

func TestFileObserverAppendConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	original := Event{TS: 1, Action: "follow", URL: "https://example.com"}
	data, _ := json.Marshal(original)
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	observer, err := NewFileObserver(path)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := observer.Update(context.Background(), original); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if err := observer.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	for i := 0; i < 31; i++ {
		var e Event
		if err := decoder.Decode(&e); err != nil {
			t.Fatal(err)
		}
		if e != original {
			t.Fatalf("event = %+v", e)
		}
	}
}

func TestHTTPObserver(t *testing.T) {
	want := Event{TS: 123, Action: "follow", URL: "https://example.com", UserID: "u"}
	for _, status := range []int{http.StatusNoContent, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
					t.Error("invalid HTTP request")
				}
				var got Event
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Error(err)
				}
				if got != want {
					t.Errorf("event = %+v", got)
				}
				w.WriteHeader(status)
			}))
			defer server.Close()
			observer, err := NewHTTPObserver(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			err = observer.Update(context.Background(), want)
			if (err != nil) != (status >= 300) {
				t.Fatalf("unexpected delivery error: %v", err)
			}
		})
	}
	for _, endpoint := range []string{"relative", "ftp://example.com", "http://"} {
		if _, err := NewHTTPObserver(endpoint); err == nil {
			t.Errorf("accepted %q", endpoint)
		}
	}
}
