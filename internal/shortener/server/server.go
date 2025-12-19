package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/router"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
	"go.uber.org/zap"
)

func NewServer() (*http.Server, storage.Storage, error) {
	cfg := config.InitConfig()

	// если логирование включено и уровень логирования валидный - запускаем логгер.
	if cfg.Logging && logger.IsValidLogLevel(cfg.LogLevel) {
		if err := logger.InitLogger(cfg.LogLevel); err != nil {
			return nil, nil, err
		}
	}

	// в зависимости от конфигурации устанавливаем тип хранилища.
	URLs := storage.SelectStorage(cfg)

	h := handlers.NewAppHandler(URLs)
	mux := router.Router(h)

	// Логируем запуск сервера.
	logger.Log.Info("Server is running",
		zap.String("address", cfg.ServerAddress),
		zap.String("log_level", cfg.LogLevel),
	)

	return &http.Server{
			Addr:    cfg.ServerAddress,
			Handler: mux,
		},
		URLs,
		nil
}

// RunServer Запуск сервера.
func RunServer(ctx context.Context) (storage.Storage, error) {
	server, storage, err := NewServer()
	if err != nil {
		return nil, err
	}

	// запускаем ListenAndServe() в горутине, чтобы он не блокировал управление на себя и была возможность
	// отменить его через передаваемый контекст
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("Ошибка запуска сервера", zap.Error(err))
		}
	}()

	// запускаем горутину, которая через временные промежутки делает удаление помеченных (is_delete=True) URL
	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err = storage.CleanUp(ctx)
				if err != nil {
					logger.Log.Error("Ошибка удаления URL", zap.Error(err))
				}

			case <-ctx.Done():
				logger.Log.Info("Горутина для фонового удаления URL завершена")
				return
			}
		}
	}()

	// ожидаем отмену контекста
	// тут произойдет блокировка, т.к. происходит чтение из пустого канала
	// канал остается пустым, пока контекст не передаст в него сигнал завершения работы
	<-ctx.Done()

	err = server.Shutdown(ctx)
	if err != nil {
		logger.Log.Error("Ошибка остановки сервера", zap.Error(err))
		return nil, err
	}

	logger.Log.Info("Сервер остановлен")
	return storage, nil
}
