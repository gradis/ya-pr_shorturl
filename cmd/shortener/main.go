package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/config"
	database "github.com/gradis/ya-pr_shorturl/internal/config/db"
	"github.com/gradis/ya-pr_shorturl/internal/handler"
	"github.com/gradis/ya-pr_shorturl/internal/logger"
	"github.com/gradis/ya-pr_shorturl/internal/middleware"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
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

	db, err := database.New(context.Background(), cfg.DatabaseDSN)
	if err != nil {
		logg.Fatal(
			"Failed to initialize database",
			zap.Error(err),
		)
	}
	defer db.Close()

	repo, err := repository.NewFileRepository(cfg.FileStoragePath)
	if err != nil {
		logg.Fatal("failed to create file repository",
			zap.String("file_storage_path", cfg.FileStoragePath),
			zap.Error(err))
	}

	urlService := service.NewURLService(repo, cfg.BaseURL)
	urlHandler := handler.NewURLHandler(urlService)
	pingHandler := handler.NewPingHandler(db.Pool)

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
