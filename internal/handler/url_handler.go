package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type URLService interface {
	AddUrl(originalURL string) (string, error)
	GetUrlByID(id string) (string, error)
}

type URLHandler struct {
	service URLService
}

func NewURLHandler(service URLService) *URLHandler {
	return &URLHandler{
		service: service,
	}
}

func (h *URLHandler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (h *URLHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.AddUrl(originalURL)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(shortURL))
}

func (h *URLHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")
	id = strings.TrimSpace(id)

	if id == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	originalURL, err := h.service.GetUrlByID(id)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
	}

	fmt.Println("aaa")
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
