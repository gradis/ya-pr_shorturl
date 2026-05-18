package main

import (
	"net/http"

	"github.com/gradis/ya-pr_shorturl/internal/handler"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"github.com/gradis/ya-pr_shorturl/internal/service"
)

func main() {
	repo := repository.NewMemoryRepository()
	urlService := service.NewURLService(repo)
	urlHandler := handler.NewURLHandler(urlService)

	mux := http.NewServeMux()

	mux.HandleFunc("/", urlHandler.Handle)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}
