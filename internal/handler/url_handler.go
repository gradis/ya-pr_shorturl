package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/auth"
	"github.com/gradis/ya-pr_shorturl/internal/service"
	"go.uber.org/zap"
)

type URLService interface {
	AddURL(ctx context.Context, originalURL string) (string, error)
	AddBatchURLs(ctx context.Context, urls []service.BatchURL) ([]service.BatchURLResult, error)
	GetURLByID(ctx context.Context, id string) (string, error)
	GetUserURLs(ctx context.Context) ([]service.UserURL, error)
	DeleteUserURLs(ctx context.Context, ids []string) error
}

type URLHandler struct {
	service URLService
	logg    *zap.Logger
}

func NewURLHandler(service URLService, logg *zap.Logger) *URLHandler {
	if logg == nil {
		logg = zap.NewNop()
	}

	return &URLHandler{
		service: service,
		logg:    logg,
	}
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Result string `json:"result"`
}

type batchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type batchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type userURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (h *URLHandler) RegisterRoutes(router gin.IRouter) {
	router.POST("/", h.handlePost)
	router.GET("/:id", h.handleGet)
	router.POST("/api/shorten", h.handleShortenJSON)
	router.POST("/api/shorten/batch", h.handleShortenBatch)
	router.GET("/api/user/urls", h.handleUserURLs)
	router.DELETE("/api/user/urls", h.handleDeleteUserURLs)
}

func (h *URLHandler) handleUserURLs(c *gin.Context) {
	if _, ok := auth.UserIDFromContext(c.Request.Context()); !ok {
		c.Status(http.StatusUnauthorized)
		return
	}

	urls, err := h.service.GetUserURLs(c.Request.Context())
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			c.Status(http.StatusUnauthorized)
			return
		}

		h.handleJSONServiceError(c, err)
		return
	}

	if len(urls) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	response := make([]userURLResponse, 0, len(urls))
	for _, item := range urls {
		response = append(response, userURLResponse{
			ShortURL:    item.ShortURL,
			OriginalURL: item.OriginalURL,
		})
	}

	c.JSON(http.StatusOK, response)
}

func (h *URLHandler) handlePost(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.String(http.StatusBadRequest, "bad request")
		return
	}

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		c.String(http.StatusBadRequest, "bad request")
		return
	}

	shortURL, err := h.service.AddURL(c.Request.Context(), originalURL)
	if err != nil {
		if errors.Is(err, service.ErrURLAlreadyExists) {
			c.Data(http.StatusConflict, "text/plain", []byte(shortURL))
			return
		}

		h.handleTextServiceError(c, err)
		return
	}

	c.Data(http.StatusCreated, "text/plain", []byte(shortURL))
}

func (h *URLHandler) handleGet(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.String(http.StatusBadRequest, "bad request")
		return
	}

	originalURL, err := h.service.GetURLByID(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrURLNotFound):
			c.String(http.StatusNotFound, "url not found")

		case errors.Is(err, service.ErrURLDeleted):
			c.Status(http.StatusGone)

		default:
			h.logInternalError(c, err)
			c.String(
				http.StatusInternalServerError,
				http.StatusText(http.StatusInternalServerError),
			)
		}

		return
	}

	c.Redirect(http.StatusTemporaryRedirect, originalURL)
}

func (h *URLHandler) handleShortenJSON(c *gin.Context) {
	var req shortenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	shortURL, err := h.service.AddURL(c.Request.Context(), req.URL)
	if err != nil {
		if errors.Is(err, service.ErrURLAlreadyExists) {
			c.JSON(http.StatusConflict, shortenResponse{
				Result: shortURL,
			})
			return
		}

		h.handleJSONServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, shortenResponse{
		Result: shortURL,
	})
}

func (h *URLHandler) handleShortenBatch(c *gin.Context) {
	var req []batchShortenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	if len(req) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	urls := make([]service.BatchURL, 0, len(req))
	for _, item := range req {
		originalURL := strings.TrimSpace(item.OriginalURL)
		if originalURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		urls = append(urls, service.BatchURL{
			CorrelationID: item.CorrelationID,
			OriginalURL:   originalURL,
		})
	}

	results, err := h.service.AddBatchURLs(c.Request.Context(), urls)
	if err != nil {
		h.handleJSONServiceError(c, err)
		return
	}

	response := make([]batchShortenResponse, 0, len(results))
	for _, result := range results {
		response = append(response, batchShortenResponse{
			CorrelationID: result.CorrelationID,
			ShortURL:      result.ShortURL,
		})
	}

	c.JSON(http.StatusCreated, response)
}

func (h *URLHandler) handleTextServiceError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrInvalidURL) {
		c.String(http.StatusBadRequest, "bad request")
		return
	}

	h.logInternalError(c, err)
	c.String(
		http.StatusInternalServerError,
		http.StatusText(http.StatusInternalServerError),
	)
}

func (h *URLHandler) handleJSONServiceError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrInvalidURL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	h.logInternalError(c, err)
	c.JSON(
		http.StatusInternalServerError,
		gin.H{"error": http.StatusText(http.StatusInternalServerError)},
	)
}

func (h *URLHandler) handleDeleteUserURLs(c *gin.Context) {
	ctx := c.Request.Context()

	if _, ok := auth.UserIDFromContext(ctx); !ok {
		c.Status(http.StatusUnauthorized)
		return
	}

	var ids []string

	if err := c.ShouldBindJSON(&ids); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "bad request"},
		)
		return
	}

	if len(ids) == 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "bad request"},
		)
		return
	}

	if err := h.service.DeleteUserURLs(ctx, ids); err != nil {
		switch {
		case errors.Is(err, service.ErrUnauthorized):
			c.Status(http.StatusUnauthorized)

		case errors.Is(err, service.ErrInvalidURLIDs):
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "bad request"},
			)

		default:
			h.handleJSONServiceError(c, err)
		}

		return
	}

	c.Status(http.StatusAccepted)
}

func (h *URLHandler) logInternalError(c *gin.Context, err error) {
	h.logg.Error(
		"URL handler request failed",
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.Error(err),
	)
}
