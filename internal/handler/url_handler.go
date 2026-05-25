package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

func (h *URLHandler) RegisterRoutes(router gin.IRouter) {
	router.POST("/", h.handlePost)
	router.GET("/:id", h.handleGet)
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
		c.String(http.StatusBadRequest, "bad request")
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
		c.String(http.StatusBadRequest, "bad request")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, originalURL)
}
