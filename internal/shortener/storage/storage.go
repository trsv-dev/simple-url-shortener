package storage

import (
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage/file"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage/memory"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage/pgstorage"
	"go.uber.org/zap"
)

// Storage интерфейс определяет методы для управления хранилищем URL.
// Он предоставляет операции для сохранения длинного URL и получения
// соответствующего длинного URL по короткому. Реализации этого
// интерфейса должны обеспечивать необходимую логику для хранения
// и поиска URL, а также обрабатывать возможные ошибки.
type Storage interface {
	SetInStorage(longURL string) (string, error)
	GetFromStorage(shortURL string) (string, error)
	Ping() error
	Close() error
}

// SelectStorage выбирает хранилище в порядке приоритета: PostgreSQL → файл → память.
// Функция возвращает интерфейс Storage, хотя InitStorage() разных хранилищ возвращает указатели
// на структуры (*PgStorage, *FileStorage, *MemoryStorage).
// Это работает, потому что в Go можно присваивать интерфейсу указатель на структуру,
// если она реализует все методы этого интерфейса.
// Это позволяет использовать однотипный код для работы с хранилищем без привязки к конкретной реализации.
func SelectStorage(cfg *config.Config) Storage {
	if cfg.DatabaseDSN != "" {
		dbStorage, err := pgstorage.InitStorage()
		if err != nil {
			logger.Log.Error("Ошибка при инициализации PostgreSQL", zap.Error(err))
		} else {
			return dbStorage
		}
	}

	if cfg.FileDB != "" {
		return file.InitStorage()
	}

	logger.Log.Warn("Не заданы конфигурации для PostgreSQL и файлового хранилища, используем хранилище в памяти.")
	return memory.InitStorage()
}
