package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
)

// Router Обработка маршрутов.
func Router(h *handlers.AppHandler) *chi.Mux {

	router := chi.NewRouter()

	// Обработка POST-запросов для создания короткого URL.
	router.Post("/", h.PostHandler)

	// Обработка GET-запросов для получения оригинального URL.
	router.Get("/{id}", h.GetHandler)

	// Обработка POST-запросов в формате JSON для создания короткого URL.
	router.Post("/api/shorten", h.APIShorten)

	return router
}
