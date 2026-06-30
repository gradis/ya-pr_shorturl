package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DatabasePinger interface {
	Ping(ctx context.Context) error
}

type PingHandler struct {
	db DatabasePinger
}

func NewPingHandler(db DatabasePinger) *PingHandler {
	return &PingHandler{db: db}
}

func (h *PingHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/ping", h.handlePing)
}

func (h *PingHandler) handlePing(c *gin.Context) {
	if h.db == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	c.Status(http.StatusOK)
}
