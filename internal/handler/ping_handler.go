package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DatabasePinger interface {
	Ping(ctx context.Context) error
}

type PingHandler struct {
	db  DatabasePinger
	log *zap.Logger
}

func NewPingHandler(db DatabasePinger, log *zap.Logger) *PingHandler {
	return &PingHandler{db: db, log: log}
}

func (h *PingHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/ping", h.handlePing)
}

func (h *PingHandler) handlePing(c *gin.Context) {
	if h.db == nil {
		h.log.Error("database ping failed", zap.String("reason", "db pinger is not configured"))
		c.Status(http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		h.log.Error("database ping failed",
			zap.Error(err),
		)

		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}
