package server

import (
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/router"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
	"go.uber.org/zap"
	"net/http"
)

func NewServer() (*http.Server, error) {
	cfg := config.InitConfig()

	// если логирование включено и уровень логирования валидный - запускаем логгер.
	if cfg.Logging && logger.IsValidLogLevel(cfg.LogLevel) {
		if err := logger.InitLogger(cfg.LogLevel); err != nil {
			return nil, err
		}
	}

	URLs := storage.InitStorage()
	h := handlers.NewAppHandler(URLs)
	mux := router.Router(h)

	// Логируем запуск сервера.
	logger.Log.Info("Server is running",
		zap.String("address", cfg.ServerAddress),
		zap.String("log_level", cfg.LogLevel),
	)

	return &http.Server{Addr: cfg.ServerAddress, Handler: logger.RequestAndResponseLogger(mux)}, nil
}

// RunServer Запуск сервера.
func RunServer() error {
	server, err := NewServer()
	if err != nil {
		return err
	}

	if err = server.ListenAndServe(); err != nil {
		return err
	}

	return nil
}
