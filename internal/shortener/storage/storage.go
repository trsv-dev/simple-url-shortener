package storage

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
