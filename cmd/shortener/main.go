package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/config"
	"github.com/gradis/ya-pr_shorturl/internal/handler"
	"github.com/gradis/ya-pr_shorturl/internal/logger"
	"github.com/gradis/ya-pr_shorturl/internal/middleware"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"github.com/gradis/ya-pr_shorturl/internal/service"
)

func main() {
	cfg := config.Parse()

	repo, err := repository.NewFileRepository(cfg.FileStoragePath)
	if err != nil {
		log.Fatal(err)
	}

	urlService := service.NewURLService(repo, cfg.BaseURL)
	urlHandler := handler.NewURLHandler(urlService)

	logg, err := logger.New()
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		_ = logg.Sync()
	}()

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(logg))
	router.Use(middleware.Gzip())

	router.HandleMethodNotAllowed = true

	urlHandler.RegisterRoutes(router)

	router.NoRoute(func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad request")
	})

	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad request")
	})

	if err := router.Run(cfg.ServerAddress); err != nil {
		log.Fatal(err)
	}
}
