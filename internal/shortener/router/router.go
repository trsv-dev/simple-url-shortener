package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/middleware"
)

// Router Обработка маршрутов.
func Router(h *handlers.AppHandler) *chi.Mux {

	router := chi.NewRouter()

	router.Use(
		logger.RequestAndResponseLogger,
		middleware.AuthMiddleware(h.SecretKey, h.TokenExp),
		middleware.GzipDecompressorMiddleware,
		middleware.GzipCompressorMiddleware,
	)

	// Обработка POST-запросов для создания короткого URL.
	router.Post("/", h.PostHandler)

	// Обработка GET-запросов для получения оригинального URL.
	router.Get("/{short_url}", h.GetHandler)

	// Обработка POST-запросов в формате JSON для создания короткого URL.
	router.Post("/api/shorten", h.APIShorten)

	// Обработка Get-запроса, проверяющего соединение с базой данных.
	router.Get("/ping", h.Ping)

	// Обработка POST-запросов в формате JSON, содержащих список URL для создания коротких URL.
	router.Post("/api/shorten/batch", h.Batch)

	// Обработка GET-запроса для получения всех URL, сокращенных пользователем.
	router.Get("/api/user/urls", h.UserURLs)

	// Обработка DELETE-запроса для асинхронного удаления списка сокращенных URL, переданных пользователем.
	router.Delete("/api/user/urls", h.Delete)

	// Обработка GET-запроса для получения статистики по количеству пользователей сервера и количеству сохраненных URL.
	router.Get("/api/internal/stats", h.Stats)

	return router
}
