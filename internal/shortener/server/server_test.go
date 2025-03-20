package server

import (
	"bytes"
	"context"
	"flag"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"testing"
	"time"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestNewServer(t *testing.T) {
	// сбрасываем флаги
	resetFlags()

	server, _, err := NewServer()
	assert.NoError(t, err)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Ошибка при запуске сервера: %v", err)
		}
	}()

	// Даем серверу немного времени на запуск
	time.Sleep(5 * time.Millisecond)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем сервер в горутине
	go func() {
		_, err := RunServer(ctx)
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Ошибка при запуске сервера: %v", err)
		}
	}()

	// Даем серверу немного времени на запуск
	time.Sleep(2 * time.Millisecond)

	// формируем тело запроса
	body := bytes.NewBuffer([]byte("http://yandex.ru"))
	// пробуем отправить простой запрос, чтобы убедиться, что сервер работает
	resp, err := http.Post("http://127.0.0.1:8080", "text/html", body)
	if err != nil {
		t.Errorf("Ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Завершаем сервер с помощью контекста
	client := http.Client{}
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:8080", nil)
	if err != nil {
		t.Errorf("Ошибка при создании запроса: %v", err)
		return
	}

	resp, err = client.Do(req)
	if err != nil {
		t.Errorf("Ошибка при отправке запроса на завершение: %v", err)
		return
	}
	defer resp.Body.Close()
	defer os.Remove("/tmp/short-url-storage.json")
	assert.NoError(t, err)
}
