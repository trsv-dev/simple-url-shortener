package memory

import (
	"github.com/stretchr/testify/assert"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"strings"
	"testing"
)

func TestMemoryStorage_SetInStorage(t *testing.T) {
	memoryStorage := InitStorage()

	t.Run("Set valid long URL", func(t *testing.T) {
		shortURL, err := memoryStorage.SetInStorage("https://example.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, shortURL)
	})

	t.Run("Set invalid long URL", func(t *testing.T) {
		shortURL, err := memoryStorage.SetInStorage("ht://example")
		assert.Error(t, err)
		assert.Empty(t, shortURL)
	})

	t.Run("Set duplicated long URL", func(t *testing.T) {
		shortURL1, err := memoryStorage.SetInStorage("https://example.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, shortURL1)
		shortURL2, err := memoryStorage.SetInStorage("https://example.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, shortURL2)

		assert.Equal(t, shortURL1, shortURL2)
	})
}

func TestMemoryStorage_GetFromStorage(t *testing.T) {
	memoryStorage := InitStorage()

	t.Run("Get valid shortURL", func(t *testing.T) {
		shortURL, err := memoryStorage.SetInStorage("https://example.com")
		assert.NoError(t, err)
		longURL, err := memoryStorage.GetFromStorage(shortURL)
		assert.NoError(t, err)
		assert.NotEmpty(t, longURL)
		assert.Equal(t, "https://example.com", longURL)
	})

	t.Run("Get invalid shortURL", func(t *testing.T) {
		longURL, err := memoryStorage.GetFromStorage("invalidShortURL")
		assert.Error(t, err)
		assert.Empty(t, longURL)
	})
}

func TestMemoryStorage_SetBatchInStorage(t *testing.T) {
	memoryStorage := InitStorage()

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

		err := memoryStorage.SetBatchInStorage(batch)
		assert.NoError(t, err)

		correlationID1 := "abcd1234"
		originalURL1 := "https://example.com"
		correlationID2 := "efgh5678"
		originalURL2 := "https://example2.com"

		shortURL := strings.TrimPrefix(batch[0].ShortURL, "http://localhost:8080/")

		// проверяем получение оригинального URL с помощью полученного из батча короткого URL
		result, err := memoryStorage.GetFromStorage(shortURL)
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
