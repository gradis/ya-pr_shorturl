// Package audit distributes request events to registered observers.
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Event struct {
	TS     int64  `json:"ts"`
	Action string `json:"action"`
	UserID string `json:"user_id,omitempty"`
	URL    string `json:"url"`
}

type Observer interface {
	Update(context.Context, Event) error
}

// Publisher keeps a fixed set of observers registered at startup.
// A failed observer does not prevent delivery to the remaining observers.
type Publisher struct {
	observers []Observer
	logg      *zap.Logger
}

func NewPublisher(logg *zap.Logger, observers ...Observer) *Publisher {
	if logg == nil {
		logg = zap.NewNop()
	}
	return &Publisher{observers: append([]Observer(nil), observers...), logg: logg}
}

func (p *Publisher) Notify(ctx context.Context, event Event) {
	for _, observer := range p.observers {
		if err := observer.Update(ctx, event); err != nil {
			p.logg.Error("audit delivery failed", zap.Error(err))
		}
	}
}

type FileObserver struct {
	mu   sync.Mutex
	file *os.File
}

func NewFileObserver(path string) (*FileObserver, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("open audit file: %w", err)
	}
	return &FileObserver{file: file}, nil
}

func (o *FileObserver) Update(_ context.Context, event Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return json.NewEncoder(o.file).Encode(event)
}

func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.file.Close()
}

type HTTPObserver struct {
	url    string
	client *http.Client
}

func NewHTTPObserver(endpoint string) (*HTTPObserver, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("audit URL must be an absolute HTTP or HTTPS URL")
	}
	return &HTTPObserver{url: endpoint, client: &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func (o *HTTPObserver) Update(ctx context.Context, event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("audit server returned status %d", resp.StatusCode)
	}
	return nil
}
