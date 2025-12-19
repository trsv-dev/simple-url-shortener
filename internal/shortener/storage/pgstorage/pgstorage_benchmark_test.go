package pgstorage

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
		DatabaseDSN:   "host=localhost port=5432 user=urlsbench password=urlsbench dbname=urlsbench sslmode=disable",
	}
}

// Возвращает PgStorage для бенчмарка.
func getPgStorage() *PgStorage {
	pgStorage, _ := InitStorage()
	return pgStorage
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

// BenchmarkPgStorage_SetInStorage Замеряет скорость работы SetInStorage в PgStorage.
func BenchmarkPgStorage_SetInStorage(b *testing.B) {
	// Задаём конфиг явно, чтобы не было логов
	config.ServerConfig = getBenchServerConfig()

	storage := getPgStorage()

	// очищаем таблицу
	_, err := storage.DB.Exec("TRUNCATE TABLE urls")
	if err != nil {
		b.Fatal(err)
	}

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

// BenchmarkPgStorage_SetBatchInStorage Замеряет скорость работы SetBatchInStorage.
func BenchmarkPgStorage_SetBatchInStorage(b *testing.B) {
	// Задаём конфиг явно, чтобы не было логов
	config.ServerConfig = getBenchServerConfig()

	storage := getPgStorage()

	// очищаем таблицу
	_, err := storage.DB.Exec("TRUNCATE TABLE urls")
	if err != nil {
		b.Fatal(err)
	}

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

// BenchmarkPgStorage_GetFromStorage Замеряет скорость работы GetFromStorage.
func BenchmarkPgStorage_GetFromStorage(b *testing.B) {
	config.ServerConfig = getBenchServerConfig()

	storage := getPgStorage()

	// очищаем таблицу
	_, err := storage.DB.Exec("TRUNCATE TABLE urls")
	if err != nil {
		b.Fatal(err)
	}

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

// BenchmarkPgStorage_GetUserURLs Замеряет скорость работы GetUserURLs.
func BenchmarkPgStorage_GetUserURLs(b *testing.B) {
	config.ServerConfig = getBenchServerConfig()

	storage := getPgStorage()

	// очищаем таблицу
	_, err := storage.DB.Exec("TRUNCATE TABLE urls")
	if err != nil {
		b.Fatal(err)
	}

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

// BenchmarkPgStorage_MarkAsDelete Замеряет скорость работы MarkAsDelete.
func BenchmarkPgStorage_MarkAsDelete(b *testing.B) {
	config.ServerConfig = getBenchServerConfig()

	storage := getPgStorage()

	// очищаем таблицу
	_, err := storage.DB.Exec("TRUNCATE TABLE urls")
	if err != nil {
		b.Fatal(err)
	}

	totalURLs := 100
	ctx := context.WithValue(context.Background(), contextkeys.UserID, "1")

	// Подготавливаем массив коротких URL
	originalURLs := make([]string, totalURLs)

	for i := 0; i < totalURLs; i++ {
		url := fmt.Sprintf("https://example.com/page_%d", i)
		shortURL, err := storage.SetInStorage(ctx, url)
		if err != nil {
			b.Fatal(err)
		}
		originalURLs[i] = shortURL
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		// очищаем таблицу и заново вставляем оригинальные URL
		_, err := storage.DB.Exec("TRUNCATE TABLE urls")
		if err != nil {
			b.Fatal(err)
		}
		for j := 0; j < totalURLs; j++ {
			_, err := storage.SetInStorage(ctx, fmt.Sprintf("https://example.com/page_%d", j))
			if err != nil {
				b.Fatal(err)
			}
		}

		b.StartTimer()
		// вызываем MarkAsDelete
		err = storage.MarkAsDelete(ctx, originalURLs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPgStorage_CleanUp Замеряет скорость работы CleanUp.
func BenchmarkPgStorage_CleanUp(b *testing.B) {
	config.ServerConfig = getBenchServerConfig()
	storage := getPgStorage()

	totalURLs := 10000
	ctx := context.WithValue(context.Background(), contextkeys.UserID, "1")

	// Подготовка таблицы один раз
	_, err := storage.DB.Exec("TRUNCATE TABLE urls")
	if err != nil {
		b.Fatal(err)
	}

	shortURLs := make([]string, totalURLs)
	for i := 0; i < totalURLs; i++ {
		shortURL, err := storage.SetInStorage(ctx, fmt.Sprintf("https://example.com/page_%d", i))
		if err != nil {
			b.Fatal(err)
		}
		shortURLs[i] = shortURL
	}

	// Помечаем половину как удаленные
	for i := 0; i < totalURLs; i += 2 {
		_, err := storage.DB.Exec(`UPDATE urls SET is_deleted = TRUE WHERE short_url = $1`, shortURLs[i])
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// начинаем транзакцию
		tx, err := storage.DB.BeginTx(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}

		// обёртка CleanUp внутри транзакции
		err = storage.CleanUp(ctx)
		if err != nil {
			tx.Rollback()
			b.Fatal(err)
		}

		// откатываем, чтобы таблица вернулась в исходное состояние
		if err := tx.Rollback(); err != nil {
			b.Fatal(err)
		}
	}
}
