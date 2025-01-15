package main

import (
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"net/http"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/router"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
)

func main() {
	if err := RunServer(); err != nil {
		panic(err)
	}
}

// RunServer Запуск сервера.
func RunServer() error {
	cfg := config.InitConfig()

	URLs := storage.InitStorage()
	mux := router.Router(URLs)

	return http.ListenAndServe(cfg.ServerAddress, mux)
}
