package errs

import "fmt"

type ErrLongURLConflict struct {
	Err      error
	LongURL  string
	ShortURL string
}

func (ec *ErrLongURLConflict) Error() string {
	return fmt.Sprintf("такой URL уже существует в БД: %v, ошибка: %v", ec.LongURL, ec.Err)
}

func (ec *ErrLongURLConflict) Unwrap() error {
	return ec.Err
}

func NewErrLongURLConflict(longURL string, shortURL string, err error) error {
	return &ErrLongURLConflict{
		Err:      err,
		LongURL:  longURL,
		ShortURL: shortURL,
	}
}
