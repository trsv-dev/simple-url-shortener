package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage/memory"
)

// Возвращает AppHandler для бенчамрка.
func getBenchmarkAppHandler() *AppHandler {
	memStore := memory.InitStorage()
	handler := NewAppHandler(memStore)

	return handler
}

// Возвращает конфиг сервера с отключенным логгированием.
func getBenchServerConfig() *config.Config {
	return &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
		Logging:       false,
		LogLevel:      "Info",
		FileDB:        "",
		DatabaseDSN:   "",
	}
}

func makeBatchBytes(amount int) []byte {
	batch := make([]models.BatchItem, 0, amount)

	for i := 0; i < amount; i++ {
		batch = append(batch, models.BatchItem{
			CorrelationID: strconv.Itoa(i),
			OriginalURL:   fmt.Sprintf("https://example.com/%d", i),
		})
	}

	data, _ := json.Marshal(batch)
	return data
}

// BenchmarkPostHandler Замеряет скорость работы PostHandler.
func BenchmarkPostHandler(b *testing.B) {
	// Задаём конфиг явно, чтобы не было логов
	config.ServerConfig = getBenchServerConfig()

	// Используем in-memory хранилище, чтобы не зависеть от файлов/БД
	handler := getBenchmarkAppHandler()

	// Подготовка http.Request
	ctx := context.WithValue(context.Background(), contextkeys.UserID, "1")

	b.ResetTimer() // сброс таймера перед измерением
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// тело запроса
		url := fmt.Sprintf("https://example.com/page_%d", i)
		// каждый раз передаем новый longURL из сгенерированного слайса longURLs
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(url)))
		r = r.WithContext(ctx)
		// каждый раз создаём новый Recorder, чтобы не смешивать ответы
		w := httptest.NewRecorder()

		handler.PostHandler(w, r)

		if w.Code != http.StatusCreated {
			b.Fatalf("unexpected status: %d", w.Code)
		}
	}
}

// BenchmarkGetHandler Замеряет скорость работы GetHandler.
func BenchmarkGetHandler(b *testing.B) {
	// Инициализируем конфиг, чтобы не было логов
	config.ServerConfig = getBenchServerConfig()
	handler := getBenchmarkAppHandler()

	ctx := context.WithValue(context.Background(), contextkeys.UserID, "1")

	shortID, err := handler.storage.SetInStorage(ctx, "https://example.com")
	if err != nil {
		b.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.GetHandler(w, r)

		if w.Code != http.StatusTemporaryRedirect {
			b.Fatalf("unexpected status: %d", w.Code)
		}
	}
}

// BenchmarkAPIShorten Замеряет скорость работы APIShorten.
func BenchmarkAPIShorten(b *testing.B) {
	config.ServerConfig = getBenchServerConfig()
	handler := getBenchmarkAppHandler()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx := context.WithValue(context.Background(), contextkeys.UserID, "1")
		url := fmt.Sprintf("https://example.com/%d", i) // уникальный URL для каждой итерации
		body := []byte(fmt.Sprintf(`{"url":"%s"}`, url))

		r := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
		r = r.WithContext(ctx)

		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.APIShorten(w, r)

		if w.Code != http.StatusCreated {
			b.Fatalf("unexpected status: %d", w.Code)
		}
	}
}

// BenchmarkBatch Замеряет скорость работы Batch.
func BenchmarkBatch(b *testing.B) {
	config.ServerConfig = getBenchServerConfig()
	handler := getBenchmarkAppHandler()

	batchCapacity := 100

	batch := makeBatchBytes(batchCapacity)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// cоздаём request с контекстом, где есть UserID
		b.StopTimer()
		ctx := context.WithValue(context.Background(), contextkeys.UserID, strconv.Itoa(i))

		r := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(batch))
		r = r.WithContext(ctx)

		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		b.StartTimer()

		handler.Batch(w, r)

		if w.Code != http.StatusCreated {
			b.Fatalf("unexpected status: %d", w.Code)
		}
	}
}

// BenchmarkUserURLs Замеряет скорость работы UserURLs.
func BenchmarkUserURLs(b *testing.B) {
	config.ServerConfig = getBenchServerConfig()
	handler := getBenchmarkAppHandler()

	ctx := context.WithValue(context.Background(), contextkeys.UserID, "1")

	longURLssAmount := 100

	for i := 0; i < longURLssAmount; i++ {
		_, err := handler.storage.SetInStorage(ctx, fmt.Sprintf("https://example.com/page_%d", i))
		if err != nil {
			b.Fatalf("unexpected error: %s", err.Error())
		}
	}

	r := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	r = r.WithContext(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()

		handler.UserURLs(w, r)

		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", w.Code)
		}
	}
}

// BenchmarkDelete Замеряет скорость работы Delete.
// JSON marshal и подготовка запроса вынесены за таймер.
func BenchmarkDelete(b *testing.B) {
	config.ServerConfig = getBenchServerConfig()
	handler := getBenchmarkAppHandler()

	ctx := context.WithValue(context.Background(), contextkeys.UserID, "user-1")

	totalURLs := 10000
	batchSize := 10

	shortURLs := make([]string, totalURLs)

	for i := 0; i < totalURLs; i++ {
		shortURL, err := handler.storage.SetInStorage(ctx, fmt.Sprintf("https://example.com/page_%d", i))
		if err != nil {
			b.Fatal(err)
		}
		shortURLs[i] = shortURL
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		start := (i * batchSize) % totalURLs
		end := start + batchSize

		if end > totalURLs {
			end = totalURLs
		}

		data, _ := json.Marshal(shortURLs[start:end])

		r := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(data))
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()

		b.StartTimer()

		handler.Delete(w, r)

		if w.Code != http.StatusAccepted {
			b.Fatalf("unexpected status: %d", w.Code)
		}
	}
}
