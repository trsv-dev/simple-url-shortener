package pgstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/mocks"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"regexp"
	"testing"
)

func TestPgStorage_Ping_WithMock(t *testing.T) {
	ctx := context.TODO()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true)) // Включаем мониторинг Ping()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPing().WillReturnError(nil) // эмулируем успешный ping

	pg := &PgStorage{DB: db}

	err = pg.Ping(ctx)
	assert.NoError(t, err, "Ожидалось, что Ping() не вернет ошибку")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидаемые вызовы должны быть выполнены")
}

func TestPgStorage_Ping_Fail_WithMock(t *testing.T) {
	ctx := context.TODO()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true)) // Включаем мониторинг Ping()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPing().WillReturnError(context.DeadlineExceeded) // Ожидаем ошибку пинга

	pg := &PgStorage{DB: db}

	err = pg.Ping(ctx)
	assert.Error(t, err, "Ожидалась ошибка из-за тайм-аута базы данных")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидаемые вызовы должны быть выполнены")
}

func TestPgStorage_IsLongURLExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	pg := &PgStorage{DB: db}
	query := `SELECT short_url FROM urls WHERE long_url = $1`

	t.Run("URL in DB", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs("https://example.com").
			WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("abcd1234"))

		shortURL, err := pg.IsLongURLExists("https://example.com")
		assert.NoError(t, err)
		assert.Equal(t, "abcd1234", shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("URL is not in DB", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs("https://notfound.com").WillReturnError(sql.ErrNoRows)

		shortURL, err := pg.IsLongURLExists("https://notfound.com")
		assert.NoError(t, err)
		assert.Empty(t, shortURL)

		// Проверяем, что все ожидаемые запросы выполнены
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPgStorage_GetFromStorage(t *testing.T) {
	ctx := context.TODO()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	pg := &PgStorage{DB: db}
	query := `SELECT long_url FROM urls WHERE short_url = $1`

	t.Run("LongURL in DB", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs("abcd1234").
			WillReturnRows(sqlmock.NewRows([]string{"long_url"}).AddRow("https://example.com"))

		longURL, err := pg.GetFromStorage(ctx, "abcd1234")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", longURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("LongURL is not in DB", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs("abc123").WillReturnError(sql.ErrNoRows)

		longURL, err := pg.GetFromStorage(ctx, "abc123")
		assert.Error(t, err)
		assert.Empty(t, longURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Invalid shortURL", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs("errorURL").
			WillReturnError(errors.New("короткий адрес не найден в базе данных"))

		longURL, err := pg.GetFromStorage(ctx, "errorURL")
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

	pg := &PgStorage{
		DB:        db,
		Generator: &mocks.MockGenerator{}, // моковый генератор коротких URL
	}

	userID := uuid.New().String()
	ctx := context.TODO()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("Invalid URL", func(t *testing.T) {
		ctx := context.TODO()
		shortURL, err := pg.SetInStorage(ctx, "invalid-url")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "передан невалидный URL")
		assert.Empty(t, shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("URL already in DB", func(t *testing.T) {
		query := `SELECT short_url FROM urls WHERE long_url=$1`

		// Мокаем поведение: SELECT возвращает существующий shortURL
		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs("https://example.com").
			WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("abcd1234"))

		_, err := pg.SetInStorage(ctx, "https://example.com")

		var conflictErr *errs.ErrLongURLConflict
		if assert.ErrorAs(t, err, &conflictErr) {
			assert.Equal(t, "abcd1234", conflictErr.ShortURL)
		} else {
			t.Errorf("ожидалась ошибка ErrLongURLConflict, но получили: %v", err)
		}

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Normal URL insertion", func(t *testing.T) {
		selectQuery := `SELECT short_url FROM urls WHERE long_url=$1`
		insertQuery := `INSERT INTO urls (long_url, short_url, user_id) VALUES ($1, $2, $3) RETURNING short_url`

		// Мокаем SELECT — вернёт sql.ErrNoRows, значит URL ещё нет в БД
		mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs("https://example.com").
			WillReturnError(sql.ErrNoRows)

		// Предположим, что сгенерированный shortURL будет "abc12345"
		expectedShortURL := "abcd1234"
		pg.Generator = &mocks.MockGenerator{ShortURL: "abcd1234"}

		// Мокаем INSERT с возвращаемым значением
		mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
			WithArgs("https://example.com", expectedShortURL, userID).
			WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow(expectedShortURL))

		shortURL, err := pg.SetInStorage(ctx, "https://example.com")
		assert.NoError(t, err)
		assert.Equal(t, expectedShortURL, shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Return error if error in SetInStorage()", func(t *testing.T) {
		selectQuery := `SELECT short_url FROM urls WHERE long_url=$1`

		// Мокаем SELECT — вернёт sql.ErrNoRows, значит URL ещё нет в БД
		mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs("https://example.com").
			WillReturnError(sql.ErrNoRows)

		// Мок-генератор, который сразу возвращает ошибку
		pg.Generator = &mocks.MockGenerator{Err: sql.ErrConnDone}

		// Вызов SetInStorage
		shortURL, err := pg.SetInStorage(ctx, "https://example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка при генерации короткого URL")
		assert.Empty(t, shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Return error if SELECT query fails", func(t *testing.T) {
		selectQuery := `SELECT short_url FROM urls WHERE long_url=$1`

		// Мокаем SELECT — вернёт sql.ErrNoRows, значит URL ещё нет в БД
		mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs("https://example.com").
			WillReturnError(sql.ErrConnDone)

		// Вызов SetInStorage
		ctx := context.TODO()
		shortURL, err := pg.SetInStorage(ctx, "https://example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка при проверке существующего URL")
		assert.Empty(t, shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Return sql.ErrNoRows error if INSERT query fails", func(t *testing.T) {
		selectQuery := `SELECT short_url FROM urls WHERE long_url=$1`
		insertQuery := `INSERT INTO urls (long_url, short_url, user_id) VALUES ($1, $2, $3) RETURNING short_url`

		// Мокаем SELECT — вернёт sql.ErrNoRows, значит URL ещё нет в БД
		mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs("https://example.com").
			WillReturnError(sql.ErrNoRows)

		// Предположим, что сгенерированный shortURL будет "abc12345"
		expectedShortURL := "abcd1234"
		pg.Generator = &mocks.MockGenerator{ShortURL: "abcd1234"}

		// Мокаем INSERT с возвращаемым значением
		mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
			WithArgs("https://example.com", expectedShortURL, userID).
			WillReturnError(sql.ErrNoRows)

		shortURL, err := pg.SetInStorage(ctx, "https://example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "неожиданная ошибка: запись не была возвращена")
		assert.Empty(t, shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Handle shortURL collision (code 23505)", func(t *testing.T) {
		selectQuery := `SELECT short_url FROM urls WHERE long_url=$1`
		insertQuery := `INSERT INTO urls (long_url, short_url, user_id) VALUES ($1, $2, $3) RETURNING short_url`

		// Мокаем SELECT — вернёт sql.ErrNoRows, значит URL ещё нет в БД
		mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs("https://example.com").
			WillReturnError(sql.ErrNoRows)

		pgErr := &pgconn.PgError{Code: "23505"}

		// Первая попытка вставки должна завершиться коллизией
		mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
			WithArgs("https://example.com", "abc12345", userID).
			WillReturnError(pgErr)

		// После коллизии должна быть вторая попытка вставки
		mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
			WithArgs("https://example.com", "xyz98765", userID).
			WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("xyz98765"))

		pg.Generator = &mocks.MockGenerator{Urls: []string{"abc12345", "xyz98765"}}

		shortURL, err := pg.SetInStorage(ctx, "https://example.com")

		assert.NoError(t, err)
		assert.Equal(t, "xyz98765", shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Return error if INSERT fails with unknown error", func(t *testing.T) {
		selectQuery := `SELECT short_url FROM urls WHERE long_url=$1`
		insertQuery := `INSERT INTO urls (long_url, short_url, user_id) VALUES ($1, $2, $3) RETURNING short_url`

		mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs("https://example.com").
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
			WithArgs("https://example.com", "abc12345", userID).
			WillReturnError(sql.ErrTxDone)

		pg.Generator = &mocks.MockGenerator{ShortURL: "abc12345"}

		shortURL, err := pg.SetInStorage(ctx, "https://example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка вставки в БД")
		assert.Empty(t, shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Return from `SetInStorage` with collision (code 23505)", func(t *testing.T) {
		selectQuery := `SELECT short_url FROM urls WHERE long_url=$1`
		insertQuery := `INSERT INTO urls (long_url, short_url, user_id) VALUES ($1, $2, $3) RETURNING short_url`

		// Мокаем SELECT — вернёт sql.ErrNoRows, значит URL ещё нет в БД
		mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs("https://example.com").
			WillReturnError(sql.ErrNoRows)

		pgErr := &pgconn.PgError{Code: "23505"}

		// Первая попытка вставки должна завершиться коллизией
		mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
			WithArgs("https://example.com", "abc12345", userID).
			WillReturnError(pgErr)

		// Вторая попытка вставки должна завершиться коллизией
		mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
			WithArgs("https://example.com", "xyz98765", userID).
			WillReturnError(pgErr)

		// Третья попытка вставки должна завершиться коллизией
		mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
			WithArgs("https://example.com", "qwer1234", userID).
			WillReturnError(pgErr)

		pg.Generator = &mocks.MockGenerator{Urls: []string{"abc12345", "xyz98765", "qwer1234", "asdf5678"}}

		shortURL, err := pg.SetInStorage(ctx, "https://example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "невозможно сгенерировать уникальный shortURL, возможны коллизии")
		assert.Empty(t, shortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPgStorage_SetBatchInStorage(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	pg := &PgStorage{
		DB:        db,
		Generator: &mocks.MockGenerator{}, // моковый генератор коротких URL
	}

	userID := uuid.New().String()
	ctx := context.TODO()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("All valid URLs inserted successfully", func(t *testing.T) {
		mock.ExpectBegin()

		insertQuery := `INSERT INTO urls (long_url, short_url, user_id) VALUES($1, $2, $3) 
                    ON CONFLICT (long_url) DO UPDATE SET short_url=urls.short_url RETURNING short_url`

		mock.ExpectPrepare(regexp.QuoteMeta(insertQuery))

		mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
			WithArgs("https://example.com/1", "abcd1234", userID).
			WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("abcd1234"))

		mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
			WithArgs("https://example.com/2", "efgh5678", userID).
			WillReturnRows(sqlmock.NewRows([]string{"abcd1234"}).AddRow("efgh5678"))

		mock.ExpectCommit()

		pg.Generator = &mocks.MockGenerator{Urls: []string{"abcd1234", "efgh5678"}}

		batch := []models.BatchItem{
			{OriginalURL: "https://example.com/1"},
			{OriginalURL: "https://example.com/2"},
		}

		err := pg.SetBatchInStorage(ctx, batch)
		assert.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/abcd1234", batch[0].ShortURL)
		assert.Equal(t, "http://localhost:8080/efgh5678", batch[1].ShortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("All valid URLs inserted successfully, invalid don't insert", func(t *testing.T) {
		mock.ExpectBegin()

		insertQuery := `INSERT INTO urls (long_url, short_url, user_id) VALUES($1, $2, $3) 
                    ON CONFLICT (long_url) DO UPDATE SET short_url=urls.short_url RETURNING short_url`

		mock.ExpectPrepare(regexp.QuoteMeta(insertQuery))

		mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
			WithArgs("https://example.com/1", "abcd1234", userID).
			WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("abcd1234"))

		mock.ExpectCommit()

		pg.Generator = &mocks.MockGenerator{Urls: []string{"abcd1234"}}

		batch := []models.BatchItem{
			{OriginalURL: "https://example.com/1"},
			{OriginalURL: "example/2"},
		}

		err := pg.SetBatchInStorage(ctx, batch)
		assert.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/abcd1234", batch[0].ShortURL)
		assert.Empty(t, batch[1].ShortURL)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPgStorage_GetUserURLs(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	pg := &PgStorage{
		DB:        db,
		Generator: &mocks.MockGenerator{},
	}

	userID := uuid.New().String()
	ctx := context.TODO()
	ctx = context.WithValue(ctx, contextkeys.UserID, userID)

	t.Run("Get users urls", func(t *testing.T) {
		selectQuery := `SELECT long_url, short_url FROM urls WHERE user_id = $1`

		// Ожидаем, что результат будет содержать один элемент.
		expected := []models.BatchItem{
			{
				OriginalURL: "https://example.com",
				ShortURL:    "http://localhost:8080/abcd1234",
			},
		}

		mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{"long_url", "short_url"}).
			AddRow("https://example.com", "abcd1234"))

		urls, err := pg.GetUserURLs(ctx)
		assert.NoError(t, err)

		assert.Equal(t, expected, urls)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Error when fetching user URLs from the database", func(t *testing.T) {
		selectQuery := `SELECT long_url, short_url FROM urls WHERE user_id = $1`

		mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(userID).WillReturnError(fmt.Errorf("rows error"))

		urls, err := pg.GetUserURLs(ctx)
		assert.Error(t, err)

		assert.Nil(t, urls)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
