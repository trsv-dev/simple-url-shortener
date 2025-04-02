package memory

import (
	"errors"
	"fmt"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/generator"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/utils"
	"go.uber.org/zap"
	"sync"
)

// Длина генерируемой последовательности символов.
const length = 8

// MemoryStorage Хранилище в памяти.
// Структура, имплементирующая интерфейс Storage, предназначена для хранения в памяти двух мап:
// - longToShort: сопоставление "длинного адреса" с "коротким адресом".
// - shortToLong: сопоставление "короткого адреса" с "длинным адресом".
// Используется для обеспечения быстрого поиска соответствий с временной сложностью O(1).
type MemoryStorage struct {
	ID          int
	LongToShort map[string]string
	ShortToLong map[string]string
	Generator   generator.URLGenerator
	mu          sync.Mutex
}

// InitStorage Инициализация хранилища
func InitStorage() *MemoryStorage {

	logger.Log.Info("Для хранения данных используется память")

	return &MemoryStorage{
		ID:          0,
		LongToShort: make(map[string]string),
		ShortToLong: make(map[string]string),
		Generator:   &generator.DefaultGenerator{},
	}
}

func (ms *MemoryStorage) SetInStorage(longURL string) (string, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// если длинный URL валидный
	if utils.IsValidURL(longURL) {
		// проверяем, есть ли он уже в хранилище
		if shortURL, ok := ms.LongToShort[longURL]; ok {
			//return shortURL, nil
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
			if _, err = ms.GetFromStorage(generatedShortURL); err == nil {
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

		ms.LongToShort[longURL], ms.ShortToLong[shortURL] = shortURL, longURL
		ms.ID++

		return ms.LongToShort[longURL], nil
	}
	return "", fmt.Errorf("передан невалидный URL")
}

func (ms *MemoryStorage) SetBatchInStorage(batch []models.BatchItem) error {
	baseURL := config.GetServerConfig().BaseURL

	ms.mu.Lock()
	// для каждого длинного URL в батче проверяем
	for i := range batch {
		// если длинный URL валидный
		if !utils.IsValidURL(batch[i].OriginalURL) {
			continue
		}

		// проверяем, есть ли уже длинный (Original URL) URL в хранилище
		if existingShortURL, ok := ms.LongToShort[batch[i].OriginalURL]; ok {
			// если есть - просто присваиваем соответствующему полю в структуре
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
			if _, err := ms.GetFromStorage(shortURL); err == nil {
				logger.Log.Info("Коллизия shortURL, пробуем заново", zap.String("shortURL", shortURL))
				continue
			}

			// если коллизии нет, то вносим в хранилище и в возвратную структуру батча
			ms.LongToShort[batch[i].OriginalURL], ms.ShortToLong[shortURL] = shortURL, batch[i].OriginalURL
			ms.ID++

			batch[i].ShortURL = baseURL + "/" + shortURL
			break
		}
	}

	ms.mu.Unlock()
	return nil
}

// GetFromStorage Поиск и возврат найденного длинного URL по переданному короткому URL.
func (ms *MemoryStorage) GetFromStorage(shortURL string) (string, error) {
	if value, ok := ms.ShortToLong[shortURL]; ok {
		return value, nil
	}

	return "", fmt.Errorf("`%v` не найден в базе данных", shortURL)
}

// Ping Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MemoryStorage) Ping() error {
	return nil // В памяти нет внешнего ресурса, поэтому всегда "живой"
}

// Close Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MemoryStorage) Close() error {
	return nil // Закрывать нечего, но метод нужен для соответствия интерфейсу
}
