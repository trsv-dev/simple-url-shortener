package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/router"
	"go.uber.org/zap"
)

// NewServer Инициализирует HTTP-сервер приложения:
// загружает конфигурацию, настраивает логирование,
// выбирает хранилище и собирает HTTP-роутер.
//
// Возвращает сконфигурированный http.Server,
// используемое хранилище и ошибку инициализации, если она возникла.
func NewServer(serverAddress string, appHandler *handlers.AppHandler) (*http.Server, error) {
	mux := router.Router(appHandler)

	return &http.Server{
			Addr:    serverAddress,
			Handler: mux,
		},
		nil
}

// RunServer Запуск сервера.
func RunServer(ctx context.Context, serverAddress string, appHandler *handlers.AppHandler) error {
	server, err := NewServer(serverAddress, appHandler)
	if err != nil {
		return err
	}

	// запускаем ListenAndServe() в горутине, чтобы он не блокировал управление на себя и была возможность
	// отменить его через передаваемый контекст
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("Ошибка запуска сервера", zap.Error(err))
		}
	}()

	// ожидаем отмену контекста
	// тут произойдет блокировка, т.к. происходит чтение из пустого канала
	// канал остается пустым, пока контекст не передаст в него сигнал завершения работы
	<-ctx.Done()

	err = server.Shutdown(ctx)
	if err != nil {
		logger.Log.Error("Ошибка остановки сервера", zap.Error(err))
		return err
	}

	logger.Log.Info("Сервер остановлен")
	return nil
}
