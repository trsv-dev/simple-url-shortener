package file

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/mocks"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"io"
	"os"
	"strings"
	"testing"
)

// Тестовая конфигурация сервера, в которую мы передаем временный файл БД.
func initTestConfig(fileDBPath string) {
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
		Logging:       true,
		LogLevel:      "Info",
		FileDB:        fileDBPath,
	}

	// подменяем "боевую" конфигурацию тестовой
	config.ServerConfig = cfg
}

// Инициализация тестового окружения с тестовой конфигурацией и тестовым файлом БД.
func initTestEnvironment(fileDB *os.File) (*FileStorage, error) {
	// подсовываем временный файл вместо настоящего, чтобы при прохождении тестов в файле не оставалось данных
	// и другие тесты не использовали их (и не падали)
	initTestConfig(fileDB.Name()) // подставляем временный файл в тестовый конфиг

	// инициализируем хранилище только после переопределения конфигурации сервера на тестовую
	storage := InitStorage()

	return storage, nil
}

// Проверка записи в файловую БД через SetEntryToFileDB()
func TestFileStorage_SetEntryToFileDB(t *testing.T) {
	userID := uuid.New().String()

	FileDB, err := os.CreateTemp("", "test-short-url-storage.json_*.json")
	require.NoError(t, err)
	defer os.Remove(FileDB.Name())

	storage, err := initTestEnvironment(FileDB)
	assert.NoError(t, err)

	t.Run("Write entry to file DB", func(t *testing.T) {
		// кладем в файловую БД некую запись
		err = storage.SetEntryToFileDB("1", userID, "shortURL", "https://example.com")
		require.NoError(t, err)

		//открываем файл БД (потому что он был закрыт в предыдущем тесте)
		file, err := os.Open(config.GetServerConfig().FileDB)
		require.NoError(t, err)
		defer file.Close()

		// читаем данные из файла
		data, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Contains(t, string(data), `"uuid":"1"`)
		require.Contains(t, string(data), `"short_url":"shortURL"`)
		require.Contains(t, string(data), `"long_url":"https://example.com"`)
	})

	t.Run("FileDB write error", func(t *testing.T) {
		// Закрываем и удаляем файл
		require.NoError(t, FileDB.Close())
		require.NoError(t, os.Remove(FileDB.Name()))

		// Создаём папку вместо удалённого файла
		require.NoError(t, os.Mkdir(FileDB.Name(), 0755))

		// Пытаемся выполнить запись
		err = storage.SetEntryToFileDB("1", userID, "shortURL", "https://example.com")

		// Функция должна вернуть ошибку
		require.Error(t, err, "Ожидалась ошибка при записи в недоступный файл")
	})
}

func TestFileStorage_SetInStorage(t *testing.T) {
	userID := uuid.New().String()
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	FileDB, err := os.CreateTemp("", "test-short-url-storage.json_*.json")
	require.NoError(t, err)
	defer os.Remove(FileDB.Name())

	storage, err := initTestEnvironment(FileDB)
	assert.NoError(t, err)

	t.Run("Set valid long URL", func(t *testing.T) {
		shortURL, err := storage.SetInStorage(ctx, "https://example.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, shortURL)
	})

	t.Run("Set invalid long URL", func(t *testing.T) {
		shortURL, err := storage.SetInStorage(ctx, "ht://example")
		assert.Error(t, err)
		assert.Empty(t, shortURL)
	})

	t.Run("Set duplicated long URL", func(t *testing.T) {
		// сохраняем первый URL, ошибок быть не должно
		shortURL1, err := storage.SetInStorage(ctx, "https://unique.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, shortURL1)
		// сохраняем точно такой же второй URL, должна быть ошибка и пустое значение shortURL2
		shortURL2, err := storage.SetInStorage(ctx, "https://unique.com")

		// Проверяем, что вернулась ошибка конфликта URL
		var conflictErr *errs.ErrLongURLConflict

		if assert.ErrorAs(t, err, &conflictErr) {
			assert.Equal(t, shortURL1, conflictErr.ShortURL) // Проверяем, что ошибка содержит верный shortURL
		} else {
			t.Errorf("ожидалась ошибка ErrLongURLConflict, но получили: %v", err)
		}

		assert.Error(t, err)
		assert.Empty(t, shortURL2)

		assert.Equal(t, shortURL1, conflictErr.ShortURL)
	})

	// Проверка записи в файловую БД через SetInStorage()
	t.Run("Checking the created record in the fileDB", func(t *testing.T) {
		shortURL, err := storage.SetInStorage(ctx, "https://test.com")
		assert.NoError(t, err)

		file, err := os.Open(config.GetServerConfig().FileDB)
		assert.NoError(t, err)

		data, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Contains(t, string(data), `"uuid":"1"`)
		require.Contains(t, string(data), fmt.Sprintf(`"short_url":"%v"`, shortURL))
		require.Contains(t, string(data), `"long_url":"https://test.com"`)
	})
}

func TestMemoryStorage_GetFromStorage(t *testing.T) {
	userID := uuid.New().String()
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	FileDB, err := os.CreateTemp("", "test-short-url-storage.json_*.json")
	require.NoError(t, err)
	defer os.Remove(FileDB.Name())

	storage, err := initTestEnvironment(FileDB)
	assert.NoError(t, err)

	t.Run("Get valid shortURL", func(t *testing.T) {
		shortURL, err := storage.SetInStorage(ctx, "https://example.com")
		assert.NoError(t, err)
		longURL, err := storage.GetFromStorage(ctx, shortURL)
		assert.NoError(t, err)
		assert.NotEmpty(t, longURL)
		assert.Equal(t, "https://example.com", longURL)
	})

	t.Run("Get invalid shortURL", func(t *testing.T) {
		longURL, err := storage.GetFromStorage(ctx, "invalidShortURL")
		assert.Error(t, err)
		assert.Empty(t, longURL)
	})
}

func TestFileStorage_RestoreFromFileDB(t *testing.T) {
	userID := uuid.New().String()

	FileDB, err := os.CreateTemp("", "test-short-url-storage.json_*.json")
	require.NoError(t, err)
	defer os.Remove(FileDB.Name())

	_, err = FileDB.Write([]byte(`{"uuid":"1","short_url":"shortURL","long_url":"https://example.com"}`))
	assert.NoError(t, err)

	require.NoError(t, FileDB.Close())
	storage, err := initTestEnvironment(FileDB)
	assert.NoError(t, err)

	t.Run("Recover data from file DB to empty memory storage", func(t *testing.T) {
		assert.Equal(t, "https://example.com", storage.ShortToLong["shortURL"])
		assert.Equal(t, "shortURL", storage.LongToShort["https://example.com"])
		assert.Equal(t, 1, storage.ID)
	})

	t.Run("Empty path to DB file", func(t *testing.T) {
		// если строка с путем до файла БД пустая, то мы пропускаем запись в файл
		config.ServerConfig.FileDB = ""

		emptyStorage := &FileStorage{
			ID:          0,
			ShortToLong: make(map[string]string),
			LongToShort: make(map[string]string),
		}

		err := emptyStorage.SetEntryToFileDB("1", userID, "shortURL", "https://example.com")

		// Ошибки не должно быть, так как мы просто пропускаем запись в файл
		assert.Error(t, err)

		// Проверяем, что файл не создался
		_, err = os.Stat(config.ServerConfig.FileDB)
		assert.Error(t, err)               // Должна быть ошибка
		assert.True(t, os.IsNotExist(err)) // Ошибка должна быть "файл не найден"
	})

	t.Run("FileDB does not exist", func(t *testing.T) {
		// Создаём временное имя файла, но сам файл не создаём.
		FileDB, err := os.CreateTemp("", "test-short-url-storage.json_*.json")
		require.NoError(t, err)
		os.Remove(FileDB.Name()) // Удаляем файл сразу, чтобы симулировать его отсутствие.

		storage := &FileStorage{
			ID:          0,
			ShortToLong: make(map[string]string),
			LongToShort: make(map[string]string),
		}

		// в конфигурации указываем как БД уже не существующий файл
		config.ServerConfig.FileDB = FileDB.Name()
		// пытаемся восстановить записи из несуществующего файла
		err = storage.RestoreFromFileDB()

		assert.Error(t, err)                 // Функция не должна возвращать ошибку
		assert.Empty(t, storage.ShortToLong) // Хранилище должно остаться пустым
		assert.Empty(t, storage.LongToShort)
		assert.Equal(t, 0, storage.ID) // ID не должен меняться
	})

	t.Run("FileDB read error", func(t *testing.T) {
		// Создаём временное имя файла, но сам файл не создаём.
		FileDB, err := os.CreateTemp("", "test-short-url-storage.json_*")
		require.NoError(t, err)

		// Закрываем и удаляем его
		require.NoError(t, FileDB.Close())
		require.NoError(t, os.Remove(FileDB.Name()))

		storage := &FileStorage{
			ID:          0,
			ShortToLong: make(map[string]string),
			LongToShort: make(map[string]string),
		}

		// в конфигурации указываем как БД уже не существующий файл
		config.ServerConfig.FileDB = FileDB.Name()
		// пытаемся восстановить записи из несуществующего файла
		err = storage.RestoreFromFileDB()

		// Функция должна вернуть ошибку
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "файл не существует")
	})
}

func TestFileStorage_SetBatchInStorage(t *testing.T) {
	userID := uuid.New().String()
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	FileDB, err := os.CreateTemp("", "test-short-url-storage.json_*.json")
	require.NoError(t, err)
	defer os.Remove(FileDB.Name())

	storage, err := initTestEnvironment(FileDB)
	assert.NoError(t, err)

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

		err := storage.SetBatchInStorage(ctx, batch)
		assert.NoError(t, err)

		correlationID1 := "abcd1234"
		originalURL1 := "https://example.com"
		correlationID2 := "efgh5678"
		originalURL2 := "https://example2.com"

		shortURL := strings.TrimPrefix(batch[0].ShortURL, "http://localhost:8080/")

		// проверяем получение оригинального URL с помощью полученного из батча короткого URL
		result, err := storage.GetFromStorage(ctx, shortURL)
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

func TestFileStorage_GetUserURLs(t *testing.T) {
	memoryStorage := InitStorage()

	ctx := context.Background()
	userID := uuid.New().String()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("Get user urls", func(t *testing.T) {
		memoryStorage.Generator = &mocks.MockGenerator{Urls: []string{"abcd1234", "efgh5678"}}

		expectedBatch := []models.BatchItem{
			{
				OriginalURL: "https://example3.com",
				ShortURL:    "http://localhost:8080/abcd1234",
			},
			{
				OriginalURL: "https://example4.com",
				ShortURL:    "http://localhost:8080/efgh5678",
			},
		}

		_, err := memoryStorage.SetInStorage(ctx, "https://example3.com")
		assert.NoError(t, err)

		_, err = memoryStorage.SetInStorage(ctx, "https://example4.com")
		assert.NoError(t, err)

		batch, err := memoryStorage.GetUserURLs(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedBatch, batch)
	})

	t.Run("No UserID in r.Context()", func(t *testing.T) {
		ctx := context.Background()

		batch, err := memoryStorage.GetUserURLs(ctx)
		assert.Error(t, err)
		assert.Nil(t, batch)
	})
}
