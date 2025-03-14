package memory

import (
	"fmt"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/utils"
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
}

// InitStorage Инициализация хранилища
func InitStorage() *MemoryStorage {

	logger.Log.Info("Для хранения данных используется память")

	return &MemoryStorage{
		ID:          0,
		LongToShort: make(map[string]string),
		ShortToLong: make(map[string]string),
	}
}

func (ms *MemoryStorage) SetInStorage(longURL string) (string, error) {
	if utils.IsValidURL(longURL) {
		if shortURL, ok := ms.LongToShort[longURL]; ok {
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
			if _, err = ms.GetFromStorage(shortURL); err == nil {
				continue
			}
			break
		}
		ms.LongToShort[longURL], ms.ShortToLong[shortURL] = shortURL, longURL
		ms.ID++

		return ms.LongToShort[longURL], nil
	}
	return "", fmt.Errorf("передан невалидный URL")
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
