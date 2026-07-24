package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/config"
	"github.com/gradis/ya-pr_shorturl/internal/handler"
	"github.com/gradis/ya-pr_shorturl/internal/logger"
	"github.com/gradis/ya-pr_shorturl/internal/middleware"
	"github.com/gradis/ya-pr_shorturl/internal/service"
	"go.uber.org/zap"
)

func main() {
	logg, err := logger.New()
	if err != nil {
		panic(err)
	}

	defer func() {
		_ = logg.Sync()
	}()

	cfg := config.Parse()

	storage, err := createStorage(
		context.Background(),
		*cfg,
	)
	if err != nil {
		logg.Fatal(
			"failed to initialize storage",
			zap.Error(err),
		)
	}
	defer storage.Close()

	urlService := service.NewURLService(storage, cfg.BaseURL)
	urlHandler := handler.NewURLHandler(urlService)
	pingHandler := handler.NewPingHandler(storage, logg)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(logg))
	router.Use(middleware.Gzip())

	router.HandleMethodNotAllowed = true

	urlHandler.RegisterRoutes(router)
	pingHandler.RegisterRoutes(router)

	router.NoRoute(func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad request")
	})

	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad request")
	})

	if err := router.Run(cfg.ServerAddress); err != nil {
		logg.Fatal(
			"failed to run HTTP server",
			zap.String("server_address", cfg.ServerAddress),
			zap.Error(err),
		)
	}
}
