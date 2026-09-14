package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/generator"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/utils"
	"go.uber.org/zap"
)

// Длина генерируемой последовательности символов.
const length = 8

// MemoryStorage Хранилище в памяти.
// Структура хранилища в памяти, имплементирующая интерфейс Storage.
type MemoryStorage struct {
	Item        map[string]models.Item // short → item
	LongToShort map[string]string      // long → short
	Generator   generator.URLGenerator
	mu          sync.RWMutex
}

// InitStorage Инициализация хранилища
func InitStorage() *MemoryStorage {

	logger.Log.Info("Для хранения данных используется память")

	return &MemoryStorage{
		Item:        make(map[string]models.Item),
		LongToShort: make(map[string]string),
		Generator:   generator.NewDefaultGenerator(),
	}
}

// SetInStorage сохраняет переданный длинный URL в in-memory хранилище.
// Возвращает сгенерированный shortURL или ошибку при любом сбое.
func (ms *MemoryStorage) SetInStorage(ctx context.Context, longURL string) (string, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// если длинный URL валидный
	if utils.IsValidURL(longURL) {
		if shortURL, ok := ms.LongToShort[longURL]; ok {
			conflictErr := errs.NewErrLongURLConflict(longURL, shortURL, errors.New("URL уже существует в БД"))
			return "", conflictErr
		}

		var shortURL string

		// если нет - генерируем новый shortURL, если коллизии и не получается за три попытки, то пропускаем,
		// shortURL при этом будет пустым
		for attempt := 0; attempt < 3; attempt++ {
			// генерируем короткий URL
			generatedShortURL, err := ms.Generator.MakeShortURL(length)
			if err != nil {
				return "", fmt.Errorf("ошибка при генерации короткого URL: %v", err)
			}
			if _, err = ms.GetFromStorage(ctx, generatedShortURL); err == nil {
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

		// добавляем item в in-memory представление
		ms.Item[shortURL] = item
		ms.LongToShort[longURL] = shortURL

		return shortURL, nil
	}
	return "", fmt.Errorf("передан невалидный URL")
}

// SetBatchInStorage Сохранение батча переданных URL-ов в in-memory хранилище.
func (ms *MemoryStorage) SetBatchInStorage(ctx context.Context, batch []models.BatchItem) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	baseURL := config.GetServerConfig().BaseURL

	// для каждого длинного URL в батче проверяем
	for i := range batch {
		// если длинный URL валидный
		if !utils.IsValidURL(batch[i].OriginalURL) {
			continue
		}

		// проверяем, есть ли уже длинный (Original URL) URL в хранилище
		if existingShortURL, ok := ms.LongToShort[batch[i].OriginalURL]; ok {
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
			shortURL, err = ms.Generator.MakeShortURL(length)
			if err != nil {
				return fmt.Errorf("ошибка при генерации короткого URL: %v", err)
			}

			// проверяем, нет ли коллизии по shortURL
			if _, err := ms.GetFromStorage(ctx, shortURL); err == nil {
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

			// добавляем item в in-memory представление
			ms.Item[shortURL] = item
			ms.LongToShort[item.LongURL] = shortURL

			break
		}
	}

	return nil
}

// GetFromStorage Поиск и возврат найденного длинного URL по переданному короткому URL.
// ВНИМАНИЕ: небезопасна для конкурентного доступа (unsafe).
// Требует внешнего захвата блокировки: fs.mu.RLock() перед вызовом!
func (ms *MemoryStorage) GetFromStorage(ctx context.Context, shortURL string) (string, error) {
	if item, ok := ms.Item[shortURL]; ok {
		if item.IsDeleted {
			urlDeletedError := errs.NewErrURLDeleted(shortURL, true, errors.New("URL был помечен как удаленный"))
			logger.Log.Debug("Попытка доступа к удаленному URL", zap.Error(urlDeletedError))
			return "", urlDeletedError
		}

		return item.LongURL, nil
	}

	return "", fmt.Errorf("`%v` не найден в базе данных", shortURL)
}

// GetUserURLs Возвращает список всех URL-ов, ранее добавленных пользователем в in-memory хранилище.
func (ms *MemoryStorage) GetUserURLs(ctx context.Context) ([]models.BatchItem, error) {
	baseURL := config.GetServerConfig().BaseURL

	// получаем UserID через контекст и приводим его к string
	ctxUserID, ok := ctx.Value(contextkeys.UserID).(string)
	if !ok || ctxUserID == "" {
		return nil, fmt.Errorf("ID пользователя не найдено в контексте запроса")
	}

	// инициализируем слайс batch для вывода результата
	var batch []models.BatchItem

	// берем блокировку на чтение (RLock) для защиты доступа к ms.Item
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Перебираем все элементы в хранилище (ключи игнорируем, значения используем) Item
	for _, item := range ms.Item {
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

// GetURLsStat Возвращает общее количество ссылок, хранящееся в in-memory хранилище.
func (ms *MemoryStorage) GetURLsStat(ctx context.Context) (int, error) {

	// получаем UserID через контекст и приводим его к string
	ctxUserID, ok := ctx.Value(contextkeys.UserID).(string)
	if !ok || ctxUserID == "" {
		return -1, fmt.Errorf("ID пользователя не найдено в контексте запроса")
	}

	var numURLs int

	// берем блокировку на чтение (RLock) для защиты доступа к ms.Item
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Перебираем все элементы в хранилище (ключи игнорируем, значения используем) Item
	for _, item := range ms.Item {
		// фильтруем: только URL-ы не помеченные как удаленные
		if !item.IsDeleted {
			numURLs++
		}
	}

	// возвращаем количество ссылок, хранящееся в in-memory хранилище.
	return numURLs, nil
}

// GetUsersStat Возвращает общее количество пользователей, ссылки которых хранятся в in-memory хранилище.
func (ms *MemoryStorage) GetUsersStat(ctx context.Context) (int, error) {

	// получаем UserID через контекст и приводим его к string
	ctxUserID, ok := ctx.Value(contextkeys.UserID).(string)
	if !ok || ctxUserID == "" {
		return -1, fmt.Errorf("ID пользователя не найдено в контексте запроса")
	}

	var users = make(map[string]int)

	// берем блокировку на чтение (RLock) для защиты доступа к ms.Item
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Перебираем все элементы в хранилище, не помеченные как удаленные (ключи игнорируем, значения используем) Item
	for _, item := range ms.Item {
		if !item.IsDeleted {
			if users[item.UserID] == 0 {
				users[item.UserID]++
			}
		}
	}

	// возвращаем количество пользователей in-memory хранилища.
	return len(users), nil
}

// MarkAsDelete Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MemoryStorage) MarkAsDelete(ctx context.Context, urls []string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ctxUserID, ok := ctx.Value(contextkeys.UserID).(string)
	if !ok || ctxUserID == "" {
		return fmt.Errorf("userID отсутствует в контексте")
	}

	// будем считать shortURL-ы, которые не удалось удалить из-за того, что они не принадлежат
	// пользователю или уже были помечены как удаленные
	var undeletedCount int

	for _, shortURL := range urls {
		item, ok := ms.Item[shortURL]
		// если запись не существует или пользователь, переданный через контекст не является владельцем url,
		// или url отмечен как удаленный -
		// переходим к следующей итерации
		if !ok || item.IsDeleted || item.UserID != ctxUserID {
			undeletedCount++
			continue
		}

		item.IsDeleted = true
		ms.Item[shortURL] = item
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
func (ms *MemoryStorage) CleanUp(ctx context.Context) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	originalMapLength := len(ms.Item)

	for shortURL, item := range ms.Item {
		// если item.IsDelete = true, то удаляем его из мапы
		if item.IsDeleted {
			delete(ms.Item, shortURL)
		}
	}

	// считаем количество удаленных записей
	deletedCount := originalMapLength - len(ms.Item)

	logger.Log.Debug("Количество удаленных строк:", zap.Int("deleted urls", deletedCount))

	return nil
}

// Ping Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MemoryStorage) Ping(ctx context.Context) error {
	return nil // В памяти нет внешнего ресурса, поэтому всегда "живой"
}

// Close Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MemoryStorage) Close() error {
	return nil // Закрывать нечего, но метод нужен для соответствия интерфейсу
}
