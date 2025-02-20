package storage

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

// Storage интерфейс определяет методы для управления хранилищем URL.
// Он предоставляет операции для сохранения длинного URL и получения
// соответствующего длинного URL по короткому. Реализации этого
// интерфейса должны обеспечивать необходимую логику для хранения
// и поиска URL, а также обрабатывать возможные ошибки.
type Storage interface {
	SetInStorage(longURL string) (string, error)
	GetFromStorage(shortURL string) (string, error)
}

// InitStorage Инициализация хранилища
func InitStorage() *URLStorage {
	URLs := URLStorage{
		id:          0,
		longToShort: make(map[string]string),
		shortToLong: make(map[string]string),
	}

	if err := URLs.RestoreFromFileDB(); err != nil {
		logger.Log.Error("Ошибка загрузки данных из файла БД:", zap.String("error", err.Error()))
	}

	return &URLs
}

// URLStorage Структура, имплементирующая интерфейс Storage, предназначена для хранения двух мап:
// - longToShort: сопоставление "длинного адреса" с "коротким адресом".
// - shortToLong: сопоставление "короткого адреса" с "длинным адресом".
// Используется для обеспечения быстрого поиска соответствий с временной сложностью O(1).
type URLStorage struct {
	id          int
	longToShort map[string]string
	shortToLong map[string]string
}

// SetInStorage Проверка корректности переданного URL, создание соответствия переданного URL и короткого URL,
// возврат короткого URL.
func (u *URLStorage) SetInStorage(longURL string) (string, error) {
	if utils.IsValidURL(longURL) {
		if value, ok := u.longToShort[longURL]; ok {
			return value, nil
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
			if _, err = u.GetFromStorage(shortURL); err == nil {
				continue
			}
			break
		}
		u.longToShort[longURL], u.shortToLong[shortURL] = shortURL, longURL
		u.id++

		if err = u.SetEntryToFileDB(strconv.Itoa(u.id), shortURL, longURL); err != nil {
			return "", fmt.Errorf("ошибка записи в файл %v: %v", config.GetServerConfig().FileDB, err)
		}

		return u.longToShort[longURL], nil
	}
	return "", fmt.Errorf("передан невалидный URL")
}

// GetFromStorage Поиск и возврат найденного длинного URL по переданному короткому URL.
func (u *URLStorage) GetFromStorage(shortURL string) (string, error) {
	if value, ok := u.shortToLong[shortURL]; ok {
		return value, nil
	}

	return "", fmt.Errorf("`%v` не найден в базе данных", shortURL)
}

// IsFileDBExist Проверка на существование в конфигурации сервера пути до файла БД.
func (u *URLStorage) isFileStorageEnabled() bool {
	if config.GetServerConfig().FileDB != "" {
		logger.Log.Info("Для хранения данных используется файл", zap.String("file path", config.GetServerConfig().FileDB))
		return true
	}

	return false
}

// SetEntryToFileDB Запись переданного длинного URL и сгенерированного короткого URL в файл БД.
func (u *URLStorage) SetEntryToFileDB(id, shortURL, longURL string) error {
	// если в конфигурации сервера пустое значение FileDB - выходим из функции
	if !u.isFileStorageEnabled() {
		return nil
	}

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

	filename := config.GetServerConfig().FileDB
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	// чтобы использовать буферизацию при записи на диск используем буферизированный Writer
	//_, err = file.Write(append(jsonData, '\n'))
	writer := bufio.NewWriter(file)
	// Записываем данные в буфер и сразу проверяем ошибку
	if _, err = writer.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("ошибка записи в файл: %v", err)
	}
	// Сбрасываем буфер в файл
	if err = writer.Flush(); err != nil {
		return fmt.Errorf("ошибка сброса буфера в файл: %v", err)
	}

	return nil
}

// RestoreFromFileDB Восстановление данных из файла БД при перезапуске сервера.
func (u *URLStorage) RestoreFromFileDB() error {
	// если в конфигурации сервера пустое значение FileDB - выходим из функции
	if !u.isFileStorageEnabled() {
		return nil
	}

	// если в конфигурации сервера не пустое значение FileDB - продолжаем
	logger.Log.Info("Восстанавливаем данные из файла БД", zap.String("file path", config.GetServerConfig().FileDB))
	filename := config.GetServerConfig().FileDB
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("ошибка чтения файла: %v", err)
	}
	defer file.Close()

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

		// записываем полученные переменные как данные структуры хранения длинных и коротких ссылок (URLStorage)
		u.longToShort[longURL], u.shortToLong[shortURL] = shortURL, longURL

		if id > u.id {
			u.id = id
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения файла: %v", err)
	}

	logger.Log.Info("Восстановление данных из файла БД завершено")
	return nil
}
