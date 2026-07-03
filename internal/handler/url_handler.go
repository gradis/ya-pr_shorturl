package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/service"
)

type URLService interface {
	AddURL(originalURL string) (string, error)
	GetURLByID(id string) (string, error)
}

type URLHandler struct {
	service URLService
}

func NewURLHandler(service URLService) *URLHandler {
	return &URLHandler{
		service: service,
	}
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Result string `json:"result"`
}

func (h *URLHandler) RegisterRoutes(router gin.IRouter) {
	router.POST("/", h.handlePost)
	router.GET("/:id", h.handleGet)
	router.POST("/api/shorten", h.handleShortenJSON)
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

	shortURL, err := h.service.AddURL(originalURL)
	if err != nil {
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

	originalURL, err := h.service.GetURLByID(id)
	if err != nil {
		if errors.Is(err, service.ErrURLNotFound) {
			c.String(http.StatusNotFound, "url not found")
			return
		}

		c.String(http.StatusInternalServerError, "internal server error")
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

	shortURL, err := h.service.AddURL(req.URL)
	if err != nil {
		h.handleJSONServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, shortenResponse{
		Result: shortURL,
	})
}

func (h *URLHandler) handleTextServiceError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrInvalidURL) {
		c.String(http.StatusBadRequest, "bad request")
		return
	}

	c.String(http.StatusInternalServerError, "internal server error")
}

func (h *URLHandler) handleJSONServiceError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrInvalidURL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	c.JSON(
		http.StatusInternalServerError,
		gin.H{"error": "internal server error"},
	)
}
