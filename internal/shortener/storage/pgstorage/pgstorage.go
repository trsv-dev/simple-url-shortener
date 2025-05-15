package pgstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lib/pq"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"go.uber.org/zap"
	"strings"
	"time"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/generator"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/utils"
)

const length = 8

type PgStorage struct {
	DB        *sql.DB
	Generator generator.URLGenerator
}

func InitStorage() (*PgStorage, error) {

	logger.Log.Info("Для хранения данных используется PostgreSQL")

	postgres, err := sql.Open("pgx", config.GetServerConfig().DatabaseDSN)
	if err != nil {
		logger.Log.Error("Ошибка подключения к БД:", zap.Error(err))
		return nil, err
	}

	if err = postgres.Ping(); err != nil {
		logger.Log.Error("Ошибка при попытке подключения к базе данных", zap.Error(err))
		return nil, err
	}

	pgstore := &PgStorage{
		DB:        postgres,
		Generator: &generator.DefaultGenerator{},
	}

	query := `CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			long_url TEXT NOT NULL UNIQUE,
			short_url VARCHAR(250) NOT NULL UNIQUE,
            user_id VARCHAR(36),
			is_deleted BOOLEAN NOT NULL DEFAULT FALSE
		  );`

	_, err = postgres.Exec(query)
	if err != nil {
		logger.Log.Error("Ошибка создания таблицы:", zap.Error(err))
		postgres.Close() // Закрываем соединение, если возникла ошибка
		logger.Log.Info("Закрываем соединение с PostgreSQL")
		return nil, err
	}

	return pgstore, nil
}

func (pg *PgStorage) IsLongURLExists(longURL string) (string, error) {
	var shortURL string

	row := pg.DB.QueryRow(`SELECT short_url FROM urls WHERE long_url = $1`, longURL)
	err := row.Scan(&shortURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // URL не найден, не ошибка
		} else {
			return "", fmt.Errorf("ошибка получения URL: %v", err)
		}
	}

	return shortURL, nil
}

func (pg *PgStorage) SetInStorage(ctx context.Context, longURL string) (string, error) {
	if !utils.IsValidURL(longURL) {
		return "", fmt.Errorf("передан невалидный URL")
	}

	// сначала проверяем — есть ли longURL в базе
	var existingShortURL string
	err := pg.DB.QueryRowContext(ctx, `SELECT short_url FROM urls WHERE long_url=$1`, longURL).Scan(&existingShortURL)
	if err == nil {
		// URL уже существует
		conflictErr := errs.NewErrLongURLConflict(longURL, existingShortURL, errors.New("URL уже существует в БД"))
		return "", conflictErr
	}
	if !errors.Is(err, sql.ErrNoRows) {
		// Ошибка запроса (не отсутствие строки)
		return "", fmt.Errorf("ошибка при проверке существующего URL: %v", err)
	}

	// значит err == sql.ErrNoRows — URL не найден запрос вернулся пустым,
	// идём дальше и пытаемся сгенерировать для него shortURL и сделать вставку пары в БД
	for attempts := 3; attempts > 0; attempts-- {
		// генерируем короткий URL
		generatedShortURL, err := pg.Generator.MakeShortURL(length)
		if err != nil {
			return "", fmt.Errorf("ошибка при генерации короткого URL: %v", err)
		}

		row := pg.DB.QueryRowContext(ctx, `INSERT INTO urls (long_url, short_url, user_id) VALUES ($1, $2, $3)
									 RETURNING short_url`, longURL, generatedShortURL, ctx.Value(contextkeys.UserID))

		var returnedShortURL string
		var pgErr *pgconn.PgError

		err = row.Scan(&returnedShortURL)

		if err != nil {
			switch {
			case errors.Is(err, sql.ErrNoRows):
				// Ошибки отсутствия строки тут быть не должно, но на всякий случай обработаем
				return "", fmt.Errorf("неожиданная ошибка: запись не была возвращена, %v", err)
			case errors.As(err, &pgErr) && pgErr.Code == "23505":
				logger.Log.Info("Коллизия shortURL", zap.String("shortURL", generatedShortURL))
				continue
			default:
				// если ошибка вставки - возвращаем ошибку
				logger.Log.Info("Ошибка вставки в БД", zap.Error(err))
				return "", fmt.Errorf("ошибка вставки в БД: %v", err)
			}
		}

		return generatedShortURL, nil
	}

	// выход после попытки устранения коллизий и невозможности сгенерировать неповторяющийся shortURL
	logger.Log.Error("невозможно сгенерировать уникальный shortURL, возможны коллизии")
	return "", fmt.Errorf("невозможно сгенерировать уникальный shortURL, возможны коллизии")
}

func (pg *PgStorage) SetBatchInStorage(ctx context.Context, batch []models.BatchItem) error {
	baseURL := config.GetServerConfig().BaseURL

	tx, err := pg.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания транзакции %v ", err)
	}
	defer tx.Rollback()

	// если встречаем повторяющийся long_url, то делаем фейковый апдейт его short_url этим же short_url
	// и возвращаем значение short_url.
	// Если все идет штатно, то просто вставляем long_url и сгенерированный short_url.
	// Если происходит коллизия short_url - откатываем транзакцию.
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO urls (long_url, short_url, user_id) VALUES($1, $2, $3) 
                                   ON CONFLICT (long_url) DO UPDATE SET short_url=urls.short_url RETURNING short_url`)
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса %v ", err)
	}
	defer stmt.Close()

	// для каждого длинного URL в батче проверяем
	for i := range batch {
		// если передан невалидный URL - берем следующий
		if !utils.IsValidURL(batch[i].OriginalURL) {
			logger.Log.Debug("в батче присутствует невалидный URL", zap.String("original_url", batch[i].OriginalURL))
			continue
		}

		var shortURL string

		// генерируем короткий URL
		generatedShortURL, err := pg.Generator.MakeShortURL(length)
		if err != nil {
			return fmt.Errorf("ошибка генерации короткого URL: %v", err)
		}

		row := stmt.QueryRowContext(ctx, batch[i].OriginalURL, generatedShortURL, ctx.Value(contextkeys.UserID))
		err = row.Scan(&shortURL)
		if err != nil {
			switch {
			// Ошибки отсутствия строки тут быть не должно, но на всякий случай обработаем
			case errors.Is(err, sql.ErrNoRows):
				return fmt.Errorf("неожиданная ошибка: запись не была возвращена, %v", err)
			// если коллизия - продолжаем
			case strings.Contains(err.Error(), "23505"):
				logger.Log.Info("Коллизия shortURL", zap.String("shortURL", generatedShortURL))
				continue
			// если ошибка вставки - откатываем транзакцию
			default:
				logger.Log.Info("Ошибка вставки в БД", zap.Error(err))
				return fmt.Errorf("ошибка вставки в БД: %v", err)
			}
		}

		// Успешная вставка или обновление
		batch[i].ShortURL = baseURL + "/" + shortURL
		continue
	}

	// коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка при коммите транзакции: %v", err)
	}

	return nil
}

func (pg *PgStorage) GetFromStorage(ctx context.Context, shortURL string) (string, error) {
	//var longURL string

	// структура, в которую мы будем получать не только longURL, но и isDeleted
	URL := struct {
		longURL   string
		isDeleted bool
	}{}

	row := pg.DB.QueryRowContext(ctx, `SELECT long_url, is_deleted FROM urls WHERE short_url = $1`, shortURL)
	err := row.Scan(&URL.longURL, &URL.isDeleted)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			logger.Log.Error("короткий адрес не найден в базе данных", zap.String("shortURL", shortURL), zap.String("error", err.Error()))
			return "", fmt.Errorf("`%v` короткий адрес не найден в базе данных", shortURL)
		default:
			return "", fmt.Errorf("`%v` ошибка получения оригинального URL", err)
		}
	}

	// если URL был помечен на удаление (is_deleted = true), но его почему-то запросили - надо дать знать в хендлер,
	// он недоступен (в хендлере отдаем http.StatusGone)
	if URL.isDeleted {
		urlDeletedError := errs.NewErrURLDeleted(shortURL, URL.isDeleted, errors.New("URL был помечен как удаленный"))
		logger.Log.Debug("Попытка доступа к удаленному URL", zap.Error(urlDeletedError))
		return "", urlDeletedError
	}

	// если is_deleted = false, то отдаем длинный URL
	return URL.longURL, nil
}

func (pg *PgStorage) GetUserURLs(ctx context.Context) ([]models.BatchItem, error) {
	var urls []models.BatchItem
	baseURL := config.GetServerConfig().BaseURL

	rows, err := pg.DB.QueryContext(ctx, `SELECT long_url, short_url FROM urls WHERE user_id = $1 AND is_deleted = FALSE`, ctx.Value(contextkeys.UserID))
	if err != nil {
		logger.Log.Debug("ошибка запроса на получение URL-ов пользователя", zap.String("rows error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var url models.BatchItem

		if err := rows.Scan(&url.OriginalURL, &url.ShortURL); err != nil {
			logger.Log.Debug("ошибка парсинга запроса на получение URL-ов пользователя", zap.String("rows error", err.Error()))
			return nil, err
		}

		// добавляем к сокращенному URL базовый URL
		url.ShortURL = baseURL + "/" + url.ShortURL
		urls = append(urls, url)
	}

	err = rows.Err()
	if err != nil {
		logger.Log.Error("Ошибка при обработке строк на получение URL-ов пользователя:", zap.Error(err))
	}

	return urls, nil
}

func (pg *PgStorage) MarkAsDelete(ctx context.Context, URLs []string) error {
	// извлекаем userID из контекста
	ctxUserID, ok := ctx.Value(contextkeys.UserID).(string)
	if !ok || ctxUserID == "" {
		return fmt.Errorf("userID отсутствует в контексте")
	}

	// начинаем транзакцию
	tx, err := pg.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания транзакции %v ", err)
	}
	defer tx.Rollback()

	Result, err := tx.ExecContext(ctx, `UPDATE urls SET is_deleted = TRUE WHERE short_url = ANY($1) AND user_id = $2`, pq.Array(URLs), ctxUserID)
	if err != nil {
		logger.Log.Warn("Ошибка при выполнении запроса на обновление", zap.Error(err))
	}

	affectedRows, err := Result.RowsAffected()
	if err != nil {
		logger.Log.Debug("Не удалось получить количество обновленных строк", zap.Error(err))
	}
	if err == nil && affectedRows < int64(len(URLs)) {
		logger.Log.Debug("Некоторые URL не были помечены как удаленные — возможно, они не принадлежат пользователю или отсутствуют в БД",
			zap.Int("передано", len(URLs)),
			zap.Int64("обновлено", affectedRows),
		)
	}

	// коммитим транзакцию
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("ошибка при коммите транзакции: %v", err)
	}

	return nil
}

func (pg *PgStorage) CleanUp(ctx context.Context) error {
	Result, err := pg.DB.ExecContext(ctx, `DELETE FROM urls WHERE is_deleted = TRUE`)
	if err != nil {
		return fmt.Errorf("ошибка при удалении URL %w", err)
	}

	affectedRows, err := Result.RowsAffected()
	if err != nil {
		logger.Log.Debug("Не удалось получить количество удаленных строк", zap.Error(err))
	}
	if affectedRows > int64(0) {
		logger.Log.Debug("Количество удаленных строк:", zap.Int64("deleted urls", affectedRows))
	}

	return nil
}

func (pg *PgStorage) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := pg.DB.PingContext(ctx); err != nil {
		logger.Log.Error("База данных Postgres не отвечает", zap.String("error", err.Error()))
		return err
	}

	return nil
}

func (pg *PgStorage) Close() error {
	return pg.DB.Close()
}
