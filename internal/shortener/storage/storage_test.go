package storage

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitStorage(t *testing.T) {
	mockStorage := &URLStorage{
		id:          0,
		longToShort: make(map[string]string),
		shortToLong: make(map[string]string),
	}
	testTable := []struct {
		name            string
		expectedStorage *URLStorage
	}{
		{name: "Init Storage", expectedStorage: mockStorage},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			actualStorage := InitStorage()
			assert.NotNil(t, actualStorage, "Хранилище не должно быть nil")
			assert.Equal(t, mockStorage, actualStorage)
		})
	}
}

func TestURLStorage_SetInStorage(t *testing.T) {
	testTable := []struct {
		name      string
		longURL   string
		wantError bool
		sameURL   bool
	}{
		{name: "Set URL to storage and get short URL", longURL: "https://example.com"},
		{name: "Set duplicate URL to storage and get same short URL", longURL: "https://example.com", sameURL: true},
		{name: "Invalid URL", longURL: "https://example", wantError: true},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			storage := InitStorage()

			result, err := storage.SetInStorage(tt.longURL)

			if tt.wantError {
				assert.Error(t, err, "Ожидалась ошибка добавления URL")
				assert.Empty(t, result)
				return
			}

			if tt.sameURL {
				result2, err2 := storage.SetInStorage(tt.longURL)
				assert.NoError(t, err2, "Ошибка добавления URL в хранилище")
				assert.NotEmpty(t, result)
				assert.Equal(t, result2, result)
			}
		})
	}
}

func TestURLStorage_GetFromStorage(t *testing.T) {
	testTable := []struct {
		name          string
		longURL       string
		shortURL      string
		expectedError bool
	}{
		{name: "Get URL by valid shortURL", longURL: "https://example.com", expectedError: false},
		{name: "Get URL by invalid shortURL", shortURL: "shortURL", expectedError: true},
	}

	for _, tt := range testTable {
		storage := InitStorage()

		if tt.expectedError {
			longURL, err := storage.GetFromStorage(tt.shortURL)
			assert.Error(t, err, "Ожидалась ошибка для несуществующего shortURL")
			assert.Empty(t, longURL)
		}

		if !tt.expectedError {
			shortURL, err := storage.SetInStorage(tt.longURL)
			assert.NoError(t, err, "Ошибка при добавлении URL в хранилище")
			assert.NotEmpty(t, shortURL, "Короткий URL не должен быть пустым")
			longURL, err := storage.GetFromStorage(shortURL)
			assert.NoError(t, err, "Ошибка при извлечении URL из хранилища по короткой ссылке")
			assert.Equal(t, tt.longURL, longURL, "Извлеченный URL не совпадает с исходным")
		}
	}
}
