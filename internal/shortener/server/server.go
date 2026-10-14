package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/router"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

// NewServer создает HTTP-сервер и настраивает маршрутизацию и TLS.
//
// Возвращает сконфигурированный http.Server,
// используемое хранилище и ошибку инициализации, если она возникла.
func NewServer(serverAddress string, appHandler *handlers.AppHandler, enableHTTPS bool) (*http.Server, error) {
	mux := router.Router(appHandler)

	manager := &autocert.Manager{
		Cache:      autocert.DirCache("cache-dir"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("shortener.trsv-dev.ru", "www.shortener.trsv-dev.ru"),
	}

	if enableHTTPS {
		return &http.Server{
				Addr:      serverAddress,
				Handler:   mux,
				TLSConfig: manager.TLSConfig(),
			},
			nil
	}

	return &http.Server{
			Addr:    serverAddress,
			Handler: mux,
		},
		nil
}

// RunServer Запуск сервера.
func RunServer(ctx context.Context, serverAddress string, enableHTTPS bool, appHandler *handlers.AppHandler) error {
	server, err := NewServer(serverAddress, appHandler, enableHTTPS)
	if err != nil {
		return err
	}

	// запускаем ListenAndServe() или ListenAndServeTLS() в горутине, чтобы он не блокировал управление на себя и была возможность
	// отменить его через передаваемый контекст
	go func() {
		var err error

		logger.Log.Info("Сервер HTTP запущен", zap.String("address", serverAddress))

		if enableHTTPS {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("Ошибка запуска HTTP сервера", zap.Error(err))
		}
	}()

	// ожидаем отмену контекста
	// тут произойдет блокировка, т.к. происходит чтение из пустого канала
	// канал остается пустым, пока контекст не передаст в него сигнал завершения работы
	<-ctx.Done()

	logger.Log.Info("Остановка HTTP сервера")
	err = server.Shutdown(ctx)
	if err != nil {
		logger.Log.Error("Ошибка остановки HTTP сервера", zap.Error(err))
		return err
	}

	logger.Log.Info("Сервер HTTP остановлен")
	return nil
}
