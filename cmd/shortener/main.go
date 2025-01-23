package main

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
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

	// если логирование включено и уровень логирования валидный - запускаем логгер.
	if cfg.Logging && logger.IsValidLogLevel(cfg.LogLevel) {
		if err := logger.InitLogger(cfg.LogLevel); err != nil {
			return err
		}
	}

	URLs := storage.InitStorage()
	mux := router.Router(URLs)

	// Логируем запуск сервера.
	logger.Log.Info("Server is running",
		zap.String("address", cfg.ServerAddress),
		zap.String("log_level", cfg.LogLevel),
	)

	return http.ListenAndServe(cfg.ServerAddress, logger.RequestAndResponseLogger(mux))
}
