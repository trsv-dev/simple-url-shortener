package server

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage/memory"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestNewServer(t *testing.T) {
	// сбрасываем флаги
	resetFlags()

	// получаем конфигурацию сервера по умолчанию
	cfg := config.GetServerConfig()

	// создаём минимальное тестовое хранилище
	URLStorage := memory.InitStorage()
	appHandler := handlers.NewAppHandler(URLStorage, "secret", 1*time.Hour)

	server, err := NewServer(cfg.ServerAddress, appHandler, cfg.EnableHTTPS)
	assert.NoError(t, err)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Ошибка при запуске сервера: %v", err)
		}
	}()

	// Даем серверу немного времени на запуск
	time.Sleep(10 * time.Millisecond)

	// Пробуем отправить простой запрос, чтобы убедиться, что сервер работает
	resp, err := http.Get("http://127.0.0.1:8080")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// Штатно завершаем сервер
	err = server.Close()
	assert.NoError(t, err)
}

func TestRunServer(t *testing.T) {
	// сбрасываем флаги
	resetFlags()

	// получаем конфигурацию сервера по умолчанию
	cfg := config.GetServerConfig()

	// создаём минимальное тестовое хранилище
	URLStorage := memory.InitStorage()
	appHandler := handlers.NewAppHandler(URLStorage, "secret", 1*time.Hour)

	ctx, cancel := context.WithCancel(context.Background())

	// Запускаем сервер в горутине
	go func() {
		err := RunServer(ctx, cfg.ServerAddress, cfg.EnableHTTPS, appHandler)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Ошибка при запуске сервера: %v", err)
		}
	}()

	// Даем серверу немного времени на запуск
	time.Sleep(500 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:8080/ping")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// останавливаем сервер через контекст
	cancel()
	time.Sleep(10 * time.Millisecond)
}
