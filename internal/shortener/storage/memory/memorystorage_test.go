package memory

import (
	"github.com/stretchr/testify/assert"
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
