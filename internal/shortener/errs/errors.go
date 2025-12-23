package errs

import "fmt"

// ErrLongURLConflict Кастомная ошибка для случая, когда отправляемый URL уже существует в БД.
type ErrLongURLConflict struct {
	Err      error
	LongURL  string
	ShortURL string
}

// Error Описывает конфликт при сохранении длинного URL.
func (ec *ErrLongURLConflict) Error() string {
	return fmt.Sprintf("такой URL уже существует в БД: %v, ошибка: %v", ec.LongURL, ec.Err)
}

// Unwrap Возвращает исходную ошибку, чтобы ErrLongURLConflict
// можно было корректно обрабатывать через errors.Is / errors.As.
func (ec *ErrLongURLConflict) Unwrap() error {
	return ec.Err
}

// NewErrLongURLConflict Создает ErrLongURLConflict,
// оборачивая исходную ошибку сохранения URL.
func NewErrLongURLConflict(longURL string, shortURL string, err error) error {
	return &ErrLongURLConflict{
		Err:      err,
		LongURL:  longURL,
		ShortURL: shortURL,
	}
}

// ErrURLDeleted Кастомная ошибка для случая, когда запрашиваемый URL еще не был удален фактически,
// но был помечен как удаленный в БД.
type ErrURLDeleted struct {
	Err       error
	ShortURL  string
	IsDeleted bool
}

// Error Возвращает описание ошибки доступа к удалённому URL.
func (ed *ErrURLDeleted) Error() string {
	return fmt.Sprintf("доступ к `%v`: %v", ed.ShortURL, ed.Err)
}

// Unwrap Возвращает исходную ошибку для поддержки
// стандартной цепочки ошибок (errors.Is / errors.As)
func (ed *ErrURLDeleted) Unwrap() error {
	return ed.Err
}

// NewErrURLDeleted Создает ErrURLDeleted,
// оборачивая исходную ошибку доступа к URL.
func NewErrURLDeleted(shortURL string, isDeleted bool, err error) *ErrURLDeleted {
	return &ErrURLDeleted{
		Err:       err,
		ShortURL:  shortURL,
		IsDeleted: isDeleted,
	}
}
