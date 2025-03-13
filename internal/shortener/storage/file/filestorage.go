package file

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/utils"
	"go.uber.org/zap"
	"os"
	"strconv"
)

// Длина генерируемой последовательности символов.
const length = 8

// FileStorage Файловое хранилище.
// Структура, имплементирующая интерфейс Storage, предназначена для хранения в памяти двух мап:
// - longToShort: сопоставление "длинного адреса" с "коротким адресом".
// - shortToLong: сопоставление "короткого адреса" с "длинным адресом".
// Используется для обеспечения быстрого поиска соответствий с временной сложностью O(1).
type FileStorage struct {
	ID          int
	LongToShort map[string]string
	ShortToLong map[string]string
}

// InitStorage Инициализация хранилища.
func InitStorage() *FileStorage {
	URLs := &FileStorage{
		ID:          0,
		LongToShort: make(map[string]string),
		ShortToLong: make(map[string]string),
	}

	logger.Log.Info("Для хранения данных используется файл", zap.String("file path", config.GetServerConfig().FileDB))

	if err := URLs.RestoreFromFileDB(); err != nil {
		logger.Log.Error("Ошибка загрузки данных из файла БД:", zap.String("error", err.Error()))
	}

	return URLs
}

func (fs *FileStorage) SetInStorage(longURL string) (string, error) {
	if utils.IsValidURL(longURL) {
		if shortURL, ok := fs.LongToShort[longURL]; ok {
			return shortURL, nil
		}

		var shortURL string
		var err error
		// Проверяем уникальность сгенерированного короткого URL.
		// Если короткий URL существует, цикл повторяется для генерации нового.
		for {
			shortURL, err = utils.MakeShortURL(length)
			if err != nil {
				return "", fmt.Errorf("ошибка при генерации короткого URL: %v", err)
			}
			if _, err = fs.GetFromStorage(shortURL); err == nil {
				continue
			}
			break
		}
		fs.LongToShort[longURL], fs.ShortToLong[shortURL] = shortURL, longURL
		fs.ID++

		if fs.IsFileStorageEnabled() {
			if err := fs.SetEntryToFileDB(strconv.Itoa(fs.ID), shortURL, longURL); err != nil {
				return "", fmt.Errorf("ошибка записи в файл %v: %v", config.GetServerConfig().FileDB, err)
			}
		}

		return fs.LongToShort[longURL], nil
	}
	return "", fmt.Errorf("передан невалидный URL")
}

// GetFromStorage Поиск и возврат найденного длинного URL по переданному короткому URL.
func (fs *FileStorage) GetFromStorage(shortURL string) (string, error) {
	if value, ok := fs.ShortToLong[shortURL]; ok {
		return value, nil
	}

	return "", fmt.Errorf("`%v` не найден в базе данных", shortURL)
}

// IsFileStorageEnabled IsFileDBExist Проверка на существование в конфигурации сервера пути до файла БД.
func (fs *FileStorage) IsFileStorageEnabled() bool {
	return config.GetServerConfig().FileDB != ""
}

// SetEntryToFileDB Запись переданного длинного URL и сгенерированного короткого URL в файл БД.
func (fs *FileStorage) SetEntryToFileDB(id, shortURL, longURL string) error {
	// если в конфигурации сервера пустое значение FileDB - выходим из функции
	if !fs.IsFileStorageEnabled() {
		return fmt.Errorf("файловое хранилище не задано в конфигурации")
	}

	filename := config.GetServerConfig().FileDB
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	// если в конфигурации сервера не пустое значение FileDB - продолжаем
	entry := struct {
		UUID     string `json:"uuid"`
		ShortURL string `json:"short_url"`
		LongURL  string `json:"long_url"`
	}{
		UUID:     id,
		ShortURL: shortURL,
		LongURL:  longURL,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("ошибка маршаллинга JSON: %v", err)
	}

	// чтобы использовать буферизацию при записи на диск используем буферизированный Writer
	//_, err = file.Write(append(jsonData, '\n'))
	writer := bufio.NewWriter(file)
	// Записываем данные в буфер и сразу проверяем ошибку
	if _, err = writer.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("ошибка записи в файл: %v", err)
	}

	logger.Log.Info("Запись сохранена в файловую БД:", zap.String("ID", id), zap.String("URL", longURL))

	// Сбрасываем буфер в файл
	if err = writer.Flush(); err != nil {
		return fmt.Errorf("ошибка сброса буфера в файл: %v", err)
	}

	return nil
}

// RestoreFromFileDB Восстановление данных из файла БД при перезапуске сервера.
func (fs *FileStorage) RestoreFromFileDB() error {
	// если в конфигурации сервера пустое значение FileDB - выходим из функции
	if !fs.IsFileStorageEnabled() {
		return fmt.Errorf("файловое хранилище не инициализировано")
	}

	// если в конфигурации сервера не пустое значение FileDB - продолжаем
	logger.Log.Info("Восстанавливаем данные из файла БД", zap.String("file path", config.GetServerConfig().FileDB))
	filename := config.GetServerConfig().FileDB
	file, err := os.Open(filename)

	if err != nil {
		// если файла не существует
		if os.IsNotExist(err) {
			return fmt.Errorf("файл не существует: %v", err)
		}

		return fmt.Errorf("ошибка чтения файла: %v", err)
	}
	defer file.Close()

	// если файл существует, но пустой
	if fileStat, err := file.Stat(); err == nil && fileStat.Size() == 0 {
		return nil
	}

	scanner := bufio.NewScanner(file)

	// сканируем открытый файл построчно
	for scanner.Scan() {
		// каждую строку анмаршалим в мапу
		var entry map[string]string
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return fmt.Errorf("ошибка парсинга JSON: %v", err)
		}

		// разбираем полученную мапу на переменные
		id, err := strconv.Atoi(entry["uuid"])
		if err != nil {
			return fmt.Errorf("ошибка конвертации uuid в число: %v", err)
		}
		shortURL := entry["short_url"]
		longURL := entry["long_url"]

		// записываем полученные переменные как данные структуры хранения длинных и коротких ссылок (FileStorage)
		fs.LongToShort[longURL], fs.ShortToLong[shortURL] = shortURL, longURL

		if id > fs.ID {
			fs.ID = id
		}
	}

	return nil
}

// Ping Абстрактный метод для удовлетворения интерфейсу Storage.
func (fs *FileStorage) Ping() error {
	return nil // В памяти нет внешнего ресурса, поэтому всегда "живой"
}

// Close Абстрактный метод для удовлетворения интерфейсу Storage.
func (fs *FileStorage) Close() error {
	return nil // Закрывать нечего, но метод нужен для соответствия интерфейсу
}
