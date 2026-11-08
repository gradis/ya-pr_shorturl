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

// Event описывает успешное создание ссылки (shorten) или переход по ней (follow).
// TS хранит Unix-время в секундах, URL — оригинальный адрес.
// Пустой UserID опускается при сериализации JSON.
type Event struct {
	TS     int64  `json:"ts"`
	Action string `json:"action"`
	UserID string `json:"user_id,omitempty"`
	URL    string `json:"url"`
}

// Observer принимает событие аудита; ошибка доставки возвращается издателю.
// Реализация должна поддерживать конкурентные вызовы Update.
type Observer interface {
	// Update принимает событие аудита с контекстом доставки.
	Update(context.Context, Event) error
}

// Publisher keeps a fixed set of observers registered at startup.
// A failed observer does not prevent delivery to the remaining observers.
type Publisher struct {
	observers []Observer
	logg      *zap.Logger
}

// NewPublisher фиксирует набор наблюдателей и журнал ошибок доставки.
// При nil logg используется журнал без вывода; пустой набор допустим.
func NewPublisher(logg *zap.Logger, observers ...Observer) *Publisher {
	if logg == nil {
		logg = zap.NewNop()
	}
	return &Publisher{observers: append([]Observer(nil), observers...), logg: logg}
}

// Notify синхронно передаёт событие каждому наблюдателю.
// Ошибки записываются в журнал и не прерывают доставку другим наблюдателям.
func (p *Publisher) Notify(ctx context.Context, event Event) {
	for _, observer := range p.observers {
		if err := observer.Update(ctx, event); err != nil {
			p.logg.Error("audit delivery failed", zap.Error(err))
		}
	}
}

// FileObserver последовательно добавляет JSON-события в файл аудита.
// Запись и закрытие защищены мьютексом.
type FileObserver struct {
	mu   sync.Mutex
	file *os.File
}

// NewFileObserver открывает или создаёт файл в режиме добавления.
// Новый файл получает права 0600. После работы необходимо вызвать Close.
func NewFileObserver(path string) (*FileObserver, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("open audit file: %w", err)
	}
	return &FileObserver{file: file}, nil
}

// Update добавляет событие в конец файла отдельной строкой JSON.
// Контекст не используется для отмены файловой записи.
func (o *FileObserver) Update(_ context.Context, event Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return json.NewEncoder(o.file).Encode(event)
}

// Close закрывает файл аудита. Вызывается после завершения отправки событий.
func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.file.Close()
}

// HTTPObserver отправляет события JSON-запросами POST удалённому приёмнику.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver создаёт приёмник для абсолютного HTTP(S) URL.
// Таймаут запроса — пять секунд; перенаправления и повторные отправки отключены.
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

// Update отправляет событие методом POST с Content-Type application/json.
// Учитывает отмену контекста; статус вне диапазона 2xx считается ошибкой.
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
