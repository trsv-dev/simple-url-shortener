package server

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestRunServer(t *testing.T) {
	server, err := NewServer()
	assert.NoError(t, err)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Ошибка при запуске сервера: %v", err)
		}
	}()

	// Даем серверу немного времени на запуск
	time.Sleep(2 * time.Millisecond)

	// Пробуем отправить простой запрос, чтобы убедиться, что сервер работает
	resp, err := http.Get("http://127.0.0.1:8080")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// Завершаем сервер
	err = server.Close()
	assert.NoError(t, err)
}
