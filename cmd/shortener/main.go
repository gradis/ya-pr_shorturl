package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gradis/ya-pr_shorturl/internal/handler"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"github.com/gradis/ya-pr_shorturl/internal/service"
)

func main() {
	repo := repository.NewMemoryRepository()
	urlService := service.NewURLService(repo)
	urlHandler := handler.NewURLHandler(urlService)

	router := gin.Default()

	router.HandleMethodNotAllowed = true

	urlHandler.RegisterRoutes(router)

	router.NoRoute(func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad request")
	})

	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad request")
	})

	err := router.Run(":8080")
	if err != nil {
		panic(err)
	}
}
