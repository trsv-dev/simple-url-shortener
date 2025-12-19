package memory

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
)

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

// Возвращает MemoryStorage для бенчмарка.
func getMemoryStorage() *MemoryStorage {
	memStorage := InitStorage()
	return memStorage
}

// Создает и возвращает батч URL.
func makeBatch(amount int) []models.BatchItem {
	batch := make([]models.BatchItem, 0, amount)

	for i := 0; i < amount; i++ {
		batch = append(batch, models.BatchItem{
			CorrelationID: strconv.Itoa(i),
			OriginalURL:   fmt.Sprintf("https://example.com/%d", i),
		})
	}

	return batch
}

// BenchmarkMemoryStorage_SetInStorage Замеряет скорость работы SetInStorage в MemoryStorage.
func BenchmarkMemoryStorage_SetInStorage(b *testing.B) {
	// Задаём конфиг явно, чтобы не было логов
	config.ServerConfig = getBenchServerConfig()

	storage := getMemoryStorage()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		ctx := context.WithValue(context.Background(), contextkeys.UserID, strconv.Itoa(i))
		url := fmt.Sprintf("https://example.com/page_%d", i)

		b.StartTimer()

		_, err := storage.SetInStorage(ctx, url)
		if err != nil {
			b.Fatal("unexpected error", err)
		}
	}
}

// BenchmarkMemoryStorage_SetBatchInStorage Замеряет скорость работы SetBatchInStorage.
func BenchmarkMemoryStorage_SetBatchInStorage(b *testing.B) {
	// Задаём конфиг явно, чтобы не было логов
	config.ServerConfig = getBenchServerConfig()

	storage := getMemoryStorage()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		ctx := context.WithValue(context.Background(), contextkeys.UserID, strconv.Itoa(i))
		batchAmount := 1000
		batch := makeBatch(batchAmount)

		b.StartTimer()

		err := storage.SetBatchInStorage(ctx, batch)
		if err != nil {
			b.Fatal("unexpected error", err)
		}
	}
}

// BenchmarkMemoryStorage_GetFromStorage Замеряет скорость работы GetFromStorage.
func BenchmarkMemoryStorage_GetFromStorage(b *testing.B) {
	config.ServerConfig = getBenchServerConfig()

	storage := getMemoryStorage()

	ctx := context.WithValue(context.Background(), contextkeys.UserID, "1")

	// Подготовка данных
	items := 10000
	shortIDs := make([]string, items)

	for i := 0; i < items; i++ {
		id, err := storage.SetInStorage(ctx, fmt.Sprintf("https://example.com/page_%d", i))
		if err != nil {
			b.Fatal(err)
		}
		shortIDs[i] = id
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// ограничим диапазон индексов (i%items) чтобы не вылететь за размер слайса
		shortID := shortIDs[i%items]

		_, err := storage.GetFromStorage(ctx, shortID)

		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryStorage_GetUserURLs Замеряет скорость работы GetUserURLs.
func BenchmarkMemoryStorage_GetUserURLs(b *testing.B) {
	config.ServerConfig = getBenchServerConfig()

	storage := getMemoryStorage()

	ctx := context.WithValue(context.Background(), contextkeys.UserID, "1")

	longURLssAmount := 10000

	for i := 0; i < longURLssAmount; i++ {
		_, err := storage.SetInStorage(ctx, fmt.Sprintf("https://example.com/page_%d", i))
		if err != nil {
			b.Fatalf("unexpected error: %s", err.Error())
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := storage.GetUserURLs(ctx)

		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryStorage_MarkAsDelete Замеряет скорость работы MarkAsDelete.
func BenchmarkMemoryStorage_MarkAsDelete(b *testing.B) {
	config.ServerConfig = getBenchServerConfig()

	storage := getMemoryStorage()

	ctx := context.WithValue(context.Background(), contextkeys.UserID, "1")

	totalURLs := 100

	originalURLs := make([]string, totalURLs)

	for i := 0; i < totalURLs; i++ {
		shortURL, err := storage.SetInStorage(ctx, fmt.Sprintf("https://example.com/page_%d", i))
		if err != nil {
			b.Fatal(err)
		}
		originalURLs[i] = shortURL
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		// восстанавливаем storage после каждой итерации
		for k, item := range storage.Item {
			item.IsDeleted = false
			storage.Item[k] = item
		}

		b.StartTimer()

		err := storage.MarkAsDelete(ctx, originalURLs[:len(originalURLs)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryStorage_CleanUp(b *testing.B) {
	storage := getMemoryStorage()
	ctx := context.WithValue(context.Background(), contextkeys.UserID, "1")

	totalURLs := 10000

	// подготовка: создаём все URL
	originalItemMap := make(map[string]models.Item, totalURLs)

	for i := 0; i < totalURLs; i++ {
		shortURL, err := storage.SetInStorage(ctx, fmt.Sprintf("https://example.com/page_%d", i))
		if err != nil {
			b.Fatal(err)
		}

		item := storage.Item[shortURL]

		// помечаем половину как удалённые
		if i%2 == 0 {
			item.IsDeleted = true
		}

		// рабочая map с половиной данных, помеченных как isDeleted (будет перезаписываться)
		storage.Item[shortURL] = item
		// эталонная map с половиной данных, помеченных как isDeleted (никогда не меняется)
		originalItemMap[shortURL] = item
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// восстанавливаем map из originalItemMap перед каждой итерацией
		storage.mu.Lock()
		storage.Item = make(map[string]models.Item, len(originalItemMap))
		for k, v := range originalItemMap {
			storage.Item[k] = v
		}
		storage.mu.Unlock()
		b.StartTimer()

		err := storage.CleanUp(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
