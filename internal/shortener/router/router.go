package router

import (
	"net/http"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
)

// Router Обработка маршрутов и запуск сервера.
func Router(storage storage.Storage) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.PostHandler(storage, w, r)
	})
	mux.HandleFunc("/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetHandler(storage, w, r)
	})

	return mux
}
