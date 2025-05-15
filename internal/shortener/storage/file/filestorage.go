package file

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/generator"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/utils"
	"go.uber.org/zap"
	"os"
	"sync"
)

// Длина генерируемой последовательности символов.
const length = 8

// FileStorage Файловое хранилище.
// Структура файлового хранилища, имплементирующая интерфейс Storage.
type FileStorage struct {
	Item      map[string]models.Item
	Generator generator.URLGenerator
	mu        sync.RWMutex
}

// InitStorage Инициализация хранилища.
func InitStorage() *FileStorage {
	URLs := &FileStorage{
		Item:      make(map[string]models.Item),
		Generator: &generator.DefaultGenerator{},
	}

	logger.Log.Info("Для хранения данных используется файл", zap.String("file path", config.GetServerConfig().FileDB))

	if err := URLs.RestoreFromFileDB(); err != nil {
		logger.Log.Error("Ошибка загрузки данных из файла БД:", zap.String("error", err.Error()))
	}

	return URLs
}

// SetInStorage сохраняет переданный длинный URL в файловую БД и in-memory хранилище.
// Возвращает сгенерированный shortURL или ошибку при любом сбое.
func (fs *FileStorage) SetInStorage(ctx context.Context, longURL string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// если длинный URL валидный
	if utils.IsValidURL(longURL) {
		if shortURL, err := fs.GetShortFromLong(longURL); err == nil {
			conflictErr := errs.NewErrLongURLConflict(longURL, shortURL, errors.New("URL уже существует в БД"))
			return "", conflictErr
		}

		var shortURL string

		// если нет - генерируем новый shortURL, если коллизии и не получается за три попытки, то пропускаем,
		// shortURL при этом будет пустым
		for attempt := 0; attempt < 3; attempt++ {
			// генерируем короткий URL
			generatedShortURL, err := fs.Generator.MakeShortURL(length)
			if err != nil {
				return "", fmt.Errorf("ошибка при генерации короткого URL: %v", err)
			}
			if _, err = fs.GetFromStorage(ctx, generatedShortURL); err == nil {
				logger.Log.Info("Коллизия shortURL, пробуем заново", zap.String("shortURL", generatedShortURL))
				continue
			}

			// если не произошло ошибки при генерации и generatedShortURL не был найден в БД,
			// присваиваем его значение shortURL и выходим из цикла
			shortURL = generatedShortURL
			break
		}

		if shortURL == "" {
			logger.Log.Error("невозможно сгенерировать уникальный shortURL, возможны коллизии")
			return "", fmt.Errorf("невозможно сгенерировать уникальный shortURL, возможны коллизии")
		}

		userID, ok := ctx.Value(contextkeys.UserID).(string)
		if !ok || userID == "" {
			return "", fmt.Errorf("ID пользователя не найдено в контексте запроса")
		}

		// генерируем новый uuid, который запишем для item и в in-memory представление, и в файловую БД
		UUID := uuid.New().String()

		item := models.Item{
			UUID:      UUID,
			ShortURL:  shortURL,
			LongURL:   longURL,
			UserID:    userID,
			IsDeleted: false,
		}

		// записываем в файл
		if fs.IsFileStorageEnabled() {
			filePath := config.GetServerConfig().FileDB
			if err := fs.SetEntryToFileDB(filePath, &item); err != nil {
				return "", fmt.Errorf("ошибка записи в файл %v: %v", config.GetServerConfig().FileDB, err)
			}
		}

		// добавляем item в in-memory представление
		fs.Item[shortURL] = item

		return shortURL, nil
	}
	return "", fmt.Errorf("передан невалидный URL")
}

// SetBatchInStorage Сохранение батча переданных URL-ов в файловую БД.
func (fs *FileStorage) SetBatchInStorage(ctx context.Context, batch []models.BatchItem) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	baseURL := config.GetServerConfig().BaseURL

	// для каждого длинного URL в батче проверяем
	for i := range batch {
		// если длинный URL валидный
		if !utils.IsValidURL(batch[i].OriginalURL) {
			continue
		}

		// проверяем, есть ли уже длинный (Original URL) URL в хранилище
		if existingShortURL, err := fs.GetLongFromShort(batch[i].OriginalURL); err == nil {
			// если есть - просто присваиваем соответствующему полю в структуре:
			// дополняем полученный из хендлера batch коротким URL с базовым URL чтобы в хендлере потом
			// маршаллить его в ответ сервера
			batch[i].ShortURL = baseURL + "/" + existingShortURL
			continue
		}

		var (
			shortURL string
			err      error
		)

		// если нет - генерируем новый shortURL, если коллизии и не получается за три попытки, то пропускаем,
		// shortURL при этом будет пустым
		for attempt := 0; attempt < 3; attempt++ {
			// генерируем короткий URL
			shortURL, err = fs.Generator.MakeShortURL(length)
			if err != nil {
				return fmt.Errorf("ошибка при генерации короткого URL: %v", err)
			}

			// проверяем, нет ли коллизии по shortURL
			if _, err := fs.GetFromStorage(ctx, shortURL); err == nil {
				logger.Log.Info("Коллизия shortURL, пробуем заново", zap.String("shortURL", shortURL))
				continue
			}

			// если коллизии нет, то вносим в хранилище и в возвратную структуру батча
			ctxUserID, ok := ctx.Value(contextkeys.UserID).(string)
			if !ok || ctxUserID == "" {
				return fmt.Errorf("ID пользователя не найдено в контексте запроса")
			}

			// дополняем полученный из хендлера batch коротким URL с базовым URL чтобы в хендлере потом
			// маршаллить его в ответ сервера
			batch[i].ShortURL = baseURL + "/" + shortURL

			// формируем структуру для записи в файл
			item := models.Item{
				UUID:      uuid.New().String(),
				ShortURL:  shortURL,
				LongURL:   batch[i].OriginalURL,
				UserID:    ctxUserID,
				IsDeleted: false,
			}

			// записываем в файл
			if fs.IsFileStorageEnabled() {
				filePath := config.GetServerConfig().FileDB
				if err := fs.SetEntryToFileDB(filePath, &item); err != nil {
					return fmt.Errorf("ошибка записи в файл %s: %w", config.GetServerConfig().FileDB, err)
				}
			}

			// добавляем item в in-memory представление
			fs.Item[shortURL] = item

			break
		}
	}

	return nil
}

// GetFromStorage Поиск и возврат найденного длинного URL по переданному короткому URL.
// ВНИМАНИЕ: небезопасна для конкурентного доступа (unsafe).
// Требует внешнего захвата блокировки: fs.mu.RLock() перед вызовом!
func (fs *FileStorage) GetFromStorage(ctx context.Context, shortURL string) (string, error) {
	if item, ok := fs.Item[shortURL]; ok {
		if item.IsDeleted {
			urlDeletedError := errs.NewErrURLDeleted(shortURL, true, errors.New("URL был помечен как удаленный"))
			logger.Log.Debug("Попытка доступа к удаленному URL", zap.Error(urlDeletedError))
			return "", urlDeletedError
		}

		return item.LongURL, nil
	}

	return "", fmt.Errorf("`%v` не найден в базе данных", shortURL)
}

// IsFileStorageEnabled IsFileDBExist Проверка на существование в конфигурации сервера пути до файла БД.
func (fs *FileStorage) IsFileStorageEnabled() bool {
	return config.GetServerConfig().FileDB != ""
}

// SetEntryToFileDB записывает переданную структуру FileEntry в указанный файл БД.
// ВНИМАНИЕ: небезопасна для конкурентного доступа (unsafe).
// Требует внешнего захвата блокировки: fs.mu.Lock() перед вызовом!
func (fs *FileStorage) SetEntryToFileDB(filePath string, item *models.Item) error {
	// если в конфигурации сервера пустое значение FileDB - выходим из функции
	if !fs.IsFileStorageEnabled() {
		return fmt.Errorf("файловое хранилище не задано в конфигурации")
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	// если в конфигурации сервера не пустое значение FileDB и файл существует - продолжаем
	jsonData, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("ошибка маршаллинга JSON: %v", err)
	}

	// чтобы использовать буферизацию при записи на диск используем буферизированный Writer
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
func (fs *FileStorage) RestoreFromFileDB() error {
	// если в конфигурации сервера пустое значение FileDB - выходим из функции
	if !fs.IsFileStorageEnabled() {
		return fmt.Errorf("файловое хранилище не инициализировано")
	}

	// если в конфигурации сервера не пустое значение FileDB - продолжаем
	logger.Log.Info("Восстанавливаем данные из файла БД", zap.String("file path", config.GetServerConfig().FileDB))

	fs.mu.Lock()
	defer fs.mu.Unlock()

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
		// каждую строку анмаршалим в модель item
		var item models.Item
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			return fmt.Errorf("ошибка парсинга JSON: %v", err)
		}

		// в качестве ключа shortURL для fs.Item[shortURL] возьмем item.ShortURL
		fs.Item[item.ShortURL] = item
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка сканирования файла: %v", err)
	}

	return nil
}

// GetUserURLs возвращает список всех URL-ов, ранее добавленных пользователем в файловую БД.
func (fs *FileStorage) GetUserURLs(ctx context.Context) ([]models.BatchItem, error) {
	baseURL := config.GetServerConfig().BaseURL

	// получаем UserID через контекст и приводим его к string
	ctxUserID, ok := ctx.Value(contextkeys.UserID).(string)
	if !ok || ctxUserID == "" {
		return nil, fmt.Errorf("ID пользователя не найдено в контексте запроса")
	}

	// инициализируем слайс batch для вывода результата
	var batch []models.BatchItem

	// берем блокировку на чтение (RLock) для защиты доступа к fs.Item
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Перебираем все элементы в хранилище (ключи игнорируем, значения используем) Item
	for _, item := range fs.Item {
		// фильтруем: только URL-ы пользователя с текущим UserID и не помеченные как удаленные
		if item.UserID == ctxUserID && !item.IsDeleted {
			// формируем объект BatchItem (т.е. batch) с оригинальным и коротким URL (с добавлением базового URL)
			batch = append(batch, models.BatchItem{
				OriginalURL: item.LongURL,
				ShortURL:    baseURL + "/" + item.ShortURL,
			})
		}
	}

	// возвращаем сформированный список URL (т.е. batch)
	return batch, nil
}

// MarkAsDelete Пометка записи в файловой БД как удаленной.
func (fs *FileStorage) MarkAsDelete(ctx context.Context, urls []string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	ctxUserID, ok := ctx.Value(contextkeys.UserID).(string)
	if !ok || ctxUserID == "" {
		return fmt.Errorf("userID отсутствует в контексте")
	}

	// будем считать shortURL-ы, которые не удалось удалить из-за того, что они не принадлежат
	// пользователю или уже были помечены как удаленные
	var undeletedCount int

	for _, shortURL := range urls {
		item, ok := fs.Item[shortURL]
		// если запись не существует или пользователь, переданный через контекст не является владельцем url,
		// или url отмечен как удаленный -
		// переходим к следующей итерации
		if !ok || item.IsDeleted || item.UserID != ctxUserID {
			undeletedCount++
			continue
		}

		item.IsDeleted = true
		fs.Item[shortURL] = item
	}

	if undeletedCount > 0 {
		logger.Log.Info("Некоторые URL не были помечены как удаленные — возможно, они не принадлежат пользователю или отсутствуют в БД",
			zap.Int("передано", len(urls)),
			zap.Int("обновлено", len(urls)-undeletedCount),
		)
	}

	return nil
}

// CleanUp Удаление записей, которые пользователь пометил на удаление.
func (fs *FileStorage) CleanUp(ctx context.Context) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	currentFile, err := os.Open(config.GetServerConfig().FileDB)
	if err != nil {
		// если файла не существует возвращаем ошибку
		if os.IsNotExist(err) {
			return fmt.Errorf("файл не существует: %w", err)
		}
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}

	defer currentFile.Close()

	mapLength := len(fs.Item)
	latestURLs := fs.GetActualURLs()

	newFile, err := os.CreateTemp(os.TempDir(), "temp_file_db_*.json")
	if err != nil {
		return fmt.Errorf("ошибка создания нового файла БД: %w", err)
	}

	defer os.Remove(newFile.Name())
	defer newFile.Close()

	// считаем количество удаленных записей
	deletedCount := mapLength - len(latestURLs)

	for _, fileEntry := range latestURLs {
		err := fs.SetEntryToFileDB(newFile.Name(), &fileEntry)
		if err != nil {
			return fmt.Errorf("ошибка записи в файл %v: %v", config.GetServerConfig().FileDB, err)
		}
	}

	// Закрываем файлы перед заменой
	currentFile.Close()
	newFile.Close()

	// Удаляем старый файл и переименовываем временный
	if err := os.Remove(currentFile.Name()); err != nil {
		return fmt.Errorf("ошибка удаления старого файла БД: %w", err)
	}

	if err := os.Rename(newFile.Name(), currentFile.Name()); err != nil {
		return fmt.Errorf("ошибка переименования временного файла БД: %w", err)
	}

	logger.Log.Debug("Количество удаленных строк:", zap.Int("deleted urls", deletedCount))

	return nil
}

// Ping Абстрактный метод для удовлетворения интерфейсу Storage.
func (fs *FileStorage) Ping(ctx context.Context) error {
	return nil // В памяти нет внешнего ресурса, поэтому всегда "живой"
}

// Close Абстрактный метод для удовлетворения интерфейсу Storage.
func (fs *FileStorage) Close() error {
	return nil // Закрывать нечего, но метод нужен для соответствия интерфейсу
}

// GetShortFromLong Поиск короткого URL по переданному оригинальному URL.
// ВНИМАНИЕ: небезопасна для конкурентного доступа (unsafe).
// Требует внешнего захвата блокировки: fs.mu.RLock() перед вызовом!
func (fs *FileStorage) GetShortFromLong(longURL string) (string, error) {
	// Перебираем все элементы в мапе fs.Item
	for shortURL, item := range fs.Item {
		if item.LongURL == longURL && !item.IsDeleted {
			return shortURL, nil
		}
	}

	return "", fmt.Errorf("не был найден короткий URL, который соответствовал бы переданному длинному")
}

// GetLongFromShort Поиск длинного URL по переданному короткому.
// ВНИМАНИЕ: небезопасна для конкурентного доступа (unsafe).
// Требует внешнего захвата блокировки: fs.mu.RLock() перед вызовом!
func (fs *FileStorage) GetLongFromShort(shortURL string) (string, error) {
	if item, ok := fs.Item[shortURL]; ok && !item.IsDeleted {
		// если короткий URL найден в мапе и он не помечен как удалённый — возвращаем длинный URL
		return item.LongURL, nil
	}

	// если URL не найден или помечен как удалённый — возвращаем ошибку
	return "", fmt.Errorf("не был найден длинный URL, который соответствовал бы переданному короткому")
}

// GetActualURLs Возврат актуального списка URL из in-memory представления.
// ВНИМАНИЕ: небезопасна для конкурентного доступа (unsafe).
// Требует внешнего захвата блокировки: fs.mu.RLock() перед вызовом!
func (fs *FileStorage) GetActualURLs() map[string]models.Item {
	latest := make(map[string]models.Item)

	for shortURL, item := range fs.Item {
		// если item.IsDelete = true, то пропускаем его и не добавляем в мапу latest, где находятся "живые" item
		if item.IsDeleted {
			continue
		}

		latest[shortURL] = item
	}

	return latest
}
