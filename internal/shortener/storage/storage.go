package storage

import (
	"fmt"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/utils"
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

func InitStorage() *URLStorage {
	URLs := URLStorage{
		id:          0,
		longToShort: make(map[string]string),
		shortToLong: make(map[string]string),
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
		// Проверяем уникальность сгенерированного короткого URL.
		// Если короткий URL существует, цикл повторяется для генерации нового.
		for {
			shortURL = utils.MakeShortURL(length)
			if _, err := u.GetFromStorage(shortURL); err == nil {
				continue
			}
			break
		}
		u.longToShort[longURL], u.shortToLong[shortURL] = shortURL, longURL
		u.id++

		return u.longToShort[longURL], nil
	}
	return "", fmt.Errorf("передан невалидный url")
}

// GetFromStorage Поиск и возврат найденного длинного URL по переданному короткому URL.
func (u *URLStorage) GetFromStorage(shortURL string) (string, error) {
	if value, ok := u.shortToLong[shortURL]; ok {
		return value, nil
	}

	return "", fmt.Errorf("`%v` не найден в базе данных", shortURL)
}
