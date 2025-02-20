package storage

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestInitStorage(t *testing.T) {
	// создаем временный файл для файловой БД, чтобы подсунуть его в конфигурацию вместо настоящего файла БД
	testFileDB, err := os.CreateTemp("", "test-short-url-db.json_*.json")
	require.NoError(t, err)
	defer os.Remove(testFileDB.Name())

	// подсовываем временный файл вместо настоящего, чтобы при прохождении тестов в файле не оставалось данных
	// и другие тесты не использовали их (и не падали)
	config.InitConfig().FileDB = testFileDB.Name()

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

			tempDir := t.TempDir()
			config.GetServerConfig().FileDB = filepath.Join(tempDir, "test-short-url-db.json")
			defer os.Remove(config.GetServerConfig().FileDB)

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
		tempDir := t.TempDir()
		config.GetServerConfig().FileDB = filepath.Join(tempDir, "test-short-url-db.json")
		defer os.Remove(config.GetServerConfig().FileDB)

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

func TestURLStorage_SetEntryToFileDB(t *testing.T) {
	t.Run("Set entry to file DB", func(t *testing.T) {
		storage := InitStorage()

		tempDir := t.TempDir()
		config.GetServerConfig().FileDB = filepath.Join(tempDir, "test-short-url-db.json")
		defer os.Remove(config.GetServerConfig().FileDB)

		// кладем в файловую БД некую запись
		err := storage.SetEntryToFileDB("1", "shortURL", "http://example.com")
		require.NoError(t, err)

		//открываем файл БД
		file, err := os.Open(config.GetServerConfig().FileDB)
		require.NoError(t, err)
		defer file.Close()

		// читаем данные из файла
		data, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Contains(t, string(data), `"uuid":"1"`)
		require.Contains(t, string(data), `"short_url":"shortURL"`)
		require.Contains(t, string(data), `"long_url":"http://example.com"`)
	})
}

func TestURLStorage_RestoreFromFileDB(t *testing.T) {
	t.Run("Restore from file DB", func(t *testing.T) {
		storage := InitStorage()
		entryJSON := `{"uuid":"1", "short_url":"shortestURL", "long_url":"http://mock-example.com"}`
		expectedStorage := URLStorage{
			id: 1,
			longToShort: map[string]string{
				"http://mock-example.com": "shortestURL",
			},
			shortToLong: map[string]string{
				"shortestURL": "http://mock-example.com",
			},
		}

		// присваиваем полю из конфигурации значение тестового файла БД
		tempDir := t.TempDir()
		config.GetServerConfig().FileDB = filepath.Join(tempDir, "test-short-url-db.json")
		defer os.Remove(config.GetServerConfig().FileDB)

		// открываем/создаем файл, из которого собираемся восстанавливать хранилище
		file, _ := os.OpenFile(config.GetServerConfig().FileDB, os.O_CREATE|os.O_RDWR, 0666)
		_, err := file.Write([]byte(entryJSON))
		require.NoError(t, err)

		// Закрываем файл перед чтением в RestoreFromFileDB
		file.Close()

		// восстанавливаем данные из файла БД
		err = storage.RestoreFromFileDB()
		require.NoError(t, err)

		require.Equal(t, expectedStorage.id, storage.id)
		require.Equal(t, expectedStorage.longToShort, storage.longToShort)
		require.Equal(t, expectedStorage.shortToLong, storage.shortToLong)
	})
}
