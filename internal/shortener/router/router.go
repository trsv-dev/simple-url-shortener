package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
)

// Router Обработка маршрутов.
func Router(storage storage.Storage) *chi.Mux {

	router := chi.NewRouter()

	// Обработка POST-запросов для создания короткого URL.
	router.Post("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.PostHandler(storage, w, r)
	})

	// Обработка GET-запросов для получения оригинального URL.
	router.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetHandler(storage, w, r)
	})

	return router
}
