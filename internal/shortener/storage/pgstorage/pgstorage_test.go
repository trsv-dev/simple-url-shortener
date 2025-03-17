package pgstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPgStorage_Ping_WithMock(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true)) // Включаем мониторинг Ping()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPing().WillReturnError(nil) // эмулируем успешный ping

	pg := &PgStorage{DB: db}

	err = pg.Ping()
	assert.NoError(t, err, "Ожидалось, что Ping() не вернет ошибку")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидаемые вызовы должны быть выполнены")
}

func TestPgStorage_Ping_Fail_WithMock(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true)) // Включаем мониторинг Ping()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPing().WillReturnError(context.DeadlineExceeded) // Ожидаем ошибку пинга

	pg := &PgStorage{DB: db}

	err = pg.Ping()
	assert.Error(t, err, "Ожидалась ошибка из-за тайм-аута базы данных")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидаемые вызовы должны быть выполнены")
}

func TestPgStorage_IsLongURLExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	pg := &PgStorage{DB: db}
	query := `SELECT short_url FROM urls WHERE long_url = \$1`

	t.Run("URL in DB", func(t *testing.T) {
		mock.ExpectQuery(query).WithArgs("https://example.com").
			WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("abcd1234"))

		shortURL, err := pg.IsLongURLExists("https://example.com")
		assert.NoError(t, err)
		assert.Equal(t, "abcd1234", shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("URL is not in DB", func(t *testing.T) {
		mock.ExpectQuery(query).WithArgs("https://notfound.com").WillReturnError(sql.ErrNoRows)

		shortURL, err := pg.IsLongURLExists("https://notfound.com")
		assert.NoError(t, err)
		assert.Empty(t, shortURL)

		// Проверяем, что все ожидаемые запросы выполнены
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPgStorage_GetFromStorage(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	pg := &PgStorage{DB: db}
	query := `SELECT long_url FROM urls WHERE short_url = \$1`

	t.Run("LongURL in DB", func(t *testing.T) {
		mock.ExpectQuery(query).WithArgs("abcd1234").
			WillReturnRows(sqlmock.NewRows([]string{"long_url"}).AddRow("https://example.com"))

		longURL, err := pg.GetFromStorage("abcd1234")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", longURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("LongURL is not in DB", func(t *testing.T) {
		mock.ExpectQuery(query).WithArgs("abc123").WillReturnError(sql.ErrNoRows)

		longURL, err := pg.GetFromStorage("abc123")
		assert.Error(t, err)
		assert.Empty(t, longURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Invalid shortURL", func(t *testing.T) {
		mock.ExpectQuery(query).WithArgs("errorURL").
			WillReturnError(errors.New("короткий адрес не найден в базе данных"))

		longURL, err := pg.GetFromStorage("errorURL")
		assert.Error(t, err)
		assert.ErrorContains(t, err, "короткий адрес не найден в базе данных")
		assert.Empty(t, longURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPgStorage_SetInStorage(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	pg := &PgStorage{DB: db}

	t.Run("Invalid URL", func(t *testing.T) {
		shortURL, err := pg.SetInStorage("invalid-url")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "передан невалидный URL")
		assert.Empty(t, shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("URL already in DB", func(t *testing.T) {
		mock.ExpectQuery(`SELECT short_url FROM urls WHERE long_url = \$1`).WithArgs("https://example.com").
			WillReturnRows(sqlmock.NewRows([]string{"shortURL"}).AddRow("abcd1234"))

		shortURL, err := pg.SetInStorage("https://example.com")
		assert.NoError(t, err)
		assert.Equal(t, "abcd1234", shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DB insert error", func(t *testing.T) {
		longURL := "https://example.com"

		// Мокируем проверку существования URL (он отсутствует)
		mock.ExpectQuery(`SELECT short_url FROM urls WHERE long_url = \$1`).
			WithArgs("https://example.com").
			WillReturnRows(sqlmock.NewRows([]string{"short_url"})) // Пустой результат

		// Ожидаем, что любая попытка вставки вызовет ошибку (имитируем падение БД)
		mock.ExpectExec(`INSERT INTO urls`).WithArgs(longURL, sqlmock.AnyArg()).
			WillReturnError(fmt.Errorf("произошла ошибка вставки в БД"))

		shortURL, err := pg.SetInStorage(longURL)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка вставки в БД")
		assert.Empty(t, shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Return shortURL from storage", func(t *testing.T) {
		mock.ExpectQuery(`SELECT short_url FROM urls WHERE long_url = \$1`).
			WithArgs("https://example.com").
			WillReturnRows(sqlmock.NewRows([]string{"short_url"}))

		mock.ExpectExec(`INSERT INTO urls`).WithArgs("https://example.com", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		shortURL, err := pg.SetInStorage("https://example.com")
		assert.NoError(t, err)
		assert.Equal(t, length, len(shortURL))

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Return error if error in SetInStorage()", func(t *testing.T) {
		mock.ExpectQuery(`SELECT short_url FROM urls WHERE long_url = \$1`).WithArgs("https://error.com").
			WillReturnError(fmt.Errorf("ошибка базы данных"))

		_, err := pg.SetInStorage("https://error.com")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
