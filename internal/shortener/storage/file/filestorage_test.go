package file

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"io"
	"os"
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
	FileDB, err := os.CreateTemp("", "test-short-url-pgstorage.json_*.json")
	require.NoError(t, err)
	defer os.Remove(FileDB.Name())

	storage, err := initTestEnvironment(FileDB)
	assert.NoError(t, err)

	t.Run("Write entry to file DB", func(t *testing.T) {
		// кладем в файловую БД некую запись
		err = storage.SetEntryToFileDB("1", "shortURL", "https://example.com")
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
		err = storage.SetEntryToFileDB("1", "shortURL", "https://example.com")

		// Функция должна вернуть ошибку
		require.Error(t, err, "Ожидалась ошибка при записи в недоступный файл")
	})
}

func TestFileStorage_SetInStorage(t *testing.T) {
	FileDB, err := os.CreateTemp("", "test-short-url-pgstorage.json_*.json")
	require.NoError(t, err)
	defer os.Remove(FileDB.Name())

	storage, err := initTestEnvironment(FileDB)
	assert.NoError(t, err)

	t.Run("Set valid long URL", func(t *testing.T) {
		shortURL, err := storage.SetInStorage("https://example.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, shortURL)
	})

	t.Run("Set invalid long URL", func(t *testing.T) {
		shortURL, err := storage.SetInStorage("ht://example")
		assert.Error(t, err)
		assert.Empty(t, shortURL)
	})

	t.Run("Set duplicated long URL", func(t *testing.T) {
		shortURL1, err := storage.SetInStorage("https://example.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, shortURL1)
		shortURL2, err := storage.SetInStorage("https://example.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, shortURL2)

		assert.Equal(t, shortURL1, shortURL2)
	})

	// Проверка записи в файловую БД через SetInStorage()
	t.Run("Checking the created record in the fileDB", func(t *testing.T) {
		shortURL, err := storage.SetInStorage("https://example.com")
		assert.NoError(t, err)

		file, err := os.Open(config.GetServerConfig().FileDB)
		assert.NoError(t, err)

		data, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Contains(t, string(data), `"uuid":"1"`)
		require.Contains(t, string(data), fmt.Sprintf(`"short_url":"%v"`, shortURL))
		require.Contains(t, string(data), `"long_url":"https://example.com"`)
	})
}

func TestMemoryStorage_GetFromStorage(t *testing.T) {
	FileDB, err := os.CreateTemp("", "test-short-url-pgstorage.json_*.json")
	require.NoError(t, err)
	defer os.Remove(FileDB.Name())

	storage, err := initTestEnvironment(FileDB)
	assert.NoError(t, err)

	t.Run("Get valid shortURL", func(t *testing.T) {
		shortURL, err := storage.SetInStorage("https://example.com")
		assert.NoError(t, err)
		longURL, err := storage.GetFromStorage(shortURL)
		assert.NoError(t, err)
		assert.NotEmpty(t, longURL)
		assert.Equal(t, "https://example.com", longURL)
	})

	t.Run("Get invalid shortURL", func(t *testing.T) {
		longURL, err := storage.GetFromStorage("invalidShortURL")
		assert.Error(t, err)
		assert.Empty(t, longURL)
	})
}

func TestFileStorage_RestoreFromFileDB(t *testing.T) {
	FileDB, err := os.CreateTemp("", "test-short-url-pgstorage.json_*.json")
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

		err := emptyStorage.SetEntryToFileDB("1", "shortURL", "https://example.com")

		// Ошибки не должно быть, так как мы просто пропускаем запись в файл
		assert.Error(t, err)

		// Проверяем, что файл не создался
		_, err = os.Stat(config.ServerConfig.FileDB)
		assert.Error(t, err)               // Должна быть ошибка
		assert.True(t, os.IsNotExist(err)) // Ошибка должна быть "файл не найден"
	})

	t.Run("FileDB does not exist", func(t *testing.T) {
		// Создаём временное имя файла, но сам файл не создаём.
		FileDB, err := os.CreateTemp("", "test-short-url-pgstorage.json_*.json")
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
		FileDB, err := os.CreateTemp("", "test-short-url-pgstorage.json_*")
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
