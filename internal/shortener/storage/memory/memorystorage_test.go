package memory

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/mocks"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"strings"
	"testing"
)

func TestMemoryStorage_SetInStorage(t *testing.T) {
	memoryStorage := InitStorage()

	ctx := context.Background()
	userID := uuid.New().String()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("Set valid long URL", func(t *testing.T) {
		shortURL, err := memoryStorage.SetInStorage(ctx, "https://example.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, shortURL)
	})

	t.Run("Set invalid long URL", func(t *testing.T) {
		shortURL, err := memoryStorage.SetInStorage(ctx, "ht://example")
		assert.Error(t, err)
		assert.Empty(t, shortURL)
	})

	t.Run("Set duplicated long URL", func(t *testing.T) {
		shortURL1, err := memoryStorage.SetInStorage(ctx, "https://unique.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, shortURL1)

		var conflictErr *errs.ErrLongURLConflict

		shortURL2, err := memoryStorage.SetInStorage(ctx, "https://unique.com")
		if errors.As(err, &conflictErr) {
			assert.Equal(t, shortURL1, conflictErr.ShortURL)
		}
		assert.Error(t, err)
		assert.Empty(t, shortURL2)
	})
}

func TestMemoryStorage_GetFromStorage(t *testing.T) {
	memoryStorage := InitStorage()

	ctx := context.Background()
	userID := uuid.New().String()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("Get valid shortURL", func(t *testing.T) {
		shortURL, err := memoryStorage.SetInStorage(ctx, "https://example.com")
		assert.NoError(t, err)
		longURL, err := memoryStorage.GetFromStorage(ctx, shortURL)
		assert.NoError(t, err)
		assert.NotEmpty(t, longURL)
		assert.Equal(t, "https://example.com", longURL)
	})

	t.Run("Get invalid shortURL", func(t *testing.T) {
		longURL, err := memoryStorage.GetFromStorage(ctx, "invalidShortURL")
		assert.Error(t, err)
		assert.Empty(t, longURL)
	})
}

func TestMemoryStorage_SetBatchInStorage(t *testing.T) {
	memoryStorage := InitStorage()

	ctx := context.Background()
	userID := uuid.New().String()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("Set valid batch", func(t *testing.T) {
		batch := []models.BatchItem{
			{
				CorrelationID: "abcd1234",
				OriginalURL:   "https://example.com",
			},
			{
				CorrelationID: "efgh5678",
				OriginalURL:   "https://example2.com",
			},
		}

		err := memoryStorage.SetBatchInStorage(ctx, batch)
		assert.NoError(t, err)

		correlationID1 := "abcd1234"
		originalURL1 := "https://example.com"
		correlationID2 := "efgh5678"
		originalURL2 := "https://example2.com"

		shortURL := strings.TrimPrefix(batch[0].ShortURL, "http://localhost:8080/")

		// проверяем получение оригинального URL с помощью полученного из батча короткого URL
		result, err := memoryStorage.GetFromStorage(ctx, shortURL)
		assert.NoError(t, err)
		assert.Equal(t, originalURL1, result)

		// проверяем, что в батче после получения короткого URL остальные элементы без изменений
		assert.Equal(t, correlationID1, batch[0].CorrelationID)
		assert.Equal(t, originalURL1, batch[0].OriginalURL)
		assert.Equal(t, correlationID2, batch[1].CorrelationID)
		assert.Equal(t, originalURL2, batch[1].OriginalURL)

		// проверяем, что приходит короткий URL длиной 8 символов
		assert.Equal(t, length, len(shortURL))
		// проверяем, что генерируются уникальные короткие URL
		assert.NotEqual(t, batch[0].ShortURL, batch[1].ShortURL)
	})
}

func TestMemoryStorage_GetUserURLs(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("Get user urls", func(t *testing.T) {
		memoryStorage := InitStorage()
		memoryStorage.Generator = &mocks.MockGenerator{Urls: []string{"abcd1234", "efgh5678"}}

		expectedBatch := []models.BatchItem{
			{
				OriginalURL: "https://example.com",
				ShortURL:    "http://localhost:8080/abcd1234",
			},
			{
				OriginalURL: "https://example2.com",
				ShortURL:    "http://localhost:8080/efgh5678",
			},
		}

		_, err := memoryStorage.SetInStorage(ctx, "https://example.com")
		assert.NoError(t, err)

		_, err = memoryStorage.SetInStorage(ctx, "https://example2.com")
		assert.NoError(t, err)

		batch, err := memoryStorage.GetUserURLs(ctx)
		assert.NoError(t, err)
		assert.ElementsMatch(t, expectedBatch, batch)
	})

	t.Run("No UserID in r.Context()", func(t *testing.T) {
		memoryStorage := InitStorage()
		ctx := context.Background()

		batch, err := memoryStorage.GetUserURLs(ctx)
		assert.Error(t, err)
		assert.Nil(t, batch)
	})
}

func TestMemoryStorage_GetActualURLs(t *testing.T) {
	ctx := context.Background()
	userID := "123456aa-1234-12a3-a1b2-1234567890a1"
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("Get actual urls", func(t *testing.T) {
		memoryStorage := InitStorage()
		memoryStorage.Generator = &mocks.MockGenerator{Urls: []string{"abcd1234", "efgh5678"}}

		// Ожидаемая структура
		expectedItem := map[string]models.Item{
			"abcd1234": {
				ShortURL:  "abcd1234",
				LongURL:   "https://example.com",
				UserID:    userID,
				IsDeleted: false,
			},
			"efgh5678": {
				ShortURL:  "efgh5678",
				LongURL:   "https://example2.com",
				UserID:    userID,
				IsDeleted: false,
			},
		}

		_, err := memoryStorage.SetInStorage(ctx, "https://example.com")
		assert.NoError(t, err)

		_, err = memoryStorage.SetInStorage(ctx, "https://example2.com")
		assert.NoError(t, err)

		latest := memoryStorage.GetActualURLs()
		for shortURL, actualItem := range latest {
			expectedItem, ok := expectedItem[shortURL]
			require.True(t, ok, "неожиданный shortURL %s", shortURL)

			assert.Equal(t, expectedItem.ShortURL, actualItem.ShortURL)
			assert.Equal(t, expectedItem.LongURL, actualItem.LongURL)
			assert.Equal(t, expectedItem.UserID, actualItem.UserID)
			assert.Equal(t, expectedItem.IsDeleted, actualItem.IsDeleted)
		}
	})
}

func TestMemoryStorage_GetLongFromShort(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("Get long url from valid short url", func(t *testing.T) {
		memoryStorage := InitStorage()
		memoryStorage.Generator = &mocks.MockGenerator{Urls: []string{"abcd1234"}}
		expectedLongURL := "https://example.com"

		_, err := memoryStorage.SetInStorage(ctx, "https://example.com")
		assert.NoError(t, err)

		longURL, err := memoryStorage.GetLongFromShort("abcd1234")
		assert.NoError(t, err)
		assert.Equal(t, expectedLongURL, longURL)
	})

	t.Run("Get long url from invalid short url", func(t *testing.T) {
		memoryStorage := InitStorage()
		memoryStorage.Generator = &mocks.MockGenerator{Urls: []string{"abcd1234"}}

		_, err := memoryStorage.SetInStorage(ctx, "https://example.com")
		assert.NoError(t, err, "ошибка при добавлении URL в хранилище")

		_, err = memoryStorage.GetLongFromShort("error123")
		assert.Error(t, err, "ожидалась ошибка при запросе с несуществующим коротким URL")
	})
}

func TestMemoryStorage_CleanUp(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("Mark short URL as deleted and do cleanup", func(t *testing.T) {
		memoryStorage := InitStorage()
		memoryStorage.Generator = &mocks.MockGenerator{Urls: []string{"abcd1234", "efgh5678"}}

		// ожидаемая структура после очистки
		expectedItem := map[string]models.Item{
			"efgh5678": {
				ShortURL:  "efgh5678",
				LongURL:   "https://example2.com",
				UserID:    userID,
				IsDeleted: false,
			},
		}

		// добавляем два URL
		_, err := memoryStorage.SetInStorage(ctx, "https://example.com")
		assert.NoError(t, err)
		_, err = memoryStorage.SetInStorage(ctx, "https://example2.com")
		assert.NoError(t, err)

		// помечаем один из них как удалённый
		err = memoryStorage.MarkAsDelete(ctx, []string{"abcd1234"})
		assert.NoError(t, err)

		// вызываем очистку удаленных URL
		err = memoryStorage.CleanUp(ctx)
		assert.NoError(t, err)

		latest := memoryStorage.GetActualURLs()

		// проверяем, что остался только один, нужный
		require.Len(t, latest, 1, "ожидалась одна не удалённая запись")

		// сверяем с ожидаемым (в цикле и без проверки UUID)
		for shortURL, actualItem := range latest {
			expectedItem, ok := expectedItem[shortURL]
			require.True(t, ok, "неожиданный shortURL %s", shortURL)

			assert.Equal(t, expectedItem.ShortURL, actualItem.ShortURL)
			assert.Equal(t, expectedItem.LongURL, actualItem.LongURL)
			assert.Equal(t, expectedItem.UserID, actualItem.UserID)
			assert.Equal(t, expectedItem.IsDeleted, actualItem.IsDeleted)
		}
	})
}

func TestMemoryStorage_MarkAsDelete(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("Mark short URL as deleted", func(t *testing.T) {
		memoryStorage := InitStorage()
		memoryStorage.Generator = &mocks.MockGenerator{Urls: []string{"abcd1234"}}

		// ожидаемая структура после очистки
		expectedItem := map[string]models.Item{
			"abcd1234": {
				ShortURL:  "abcd1234",
				LongURL:   "https://example.com",
				UserID:    userID,
				IsDeleted: true,
			},
		}

		// добавляем URL
		_, err := memoryStorage.SetInStorage(ctx, "https://example.com")
		assert.NoError(t, err)

		// помечаем как удалённый
		err = memoryStorage.MarkAsDelete(ctx, []string{"abcd1234"})
		assert.NoError(t, err)

		// сверяем с ожидаемым (в цикле и без проверки UUID)
		for shortURL, item := range memoryStorage.Item {
			expectedItem, ok := expectedItem[shortURL]
			require.True(t, ok, "неожиданный shortURL %s", shortURL)

			assert.Equal(t, expectedItem.ShortURL, item.ShortURL)
			assert.Equal(t, expectedItem.LongURL, item.LongURL)
			assert.Equal(t, expectedItem.UserID, item.UserID)
			assert.Equal(t, expectedItem.IsDeleted, item.IsDeleted)
		}
	})
}
