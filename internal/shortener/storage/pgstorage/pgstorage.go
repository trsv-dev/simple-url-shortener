package pgstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/generator"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/utils"
	"go.uber.org/zap"
	"strings"
	"time"
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

	query := `CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			long_url TEXT NOT NULL UNIQUE,
			short_url VARCHAR(250) NOT NULL UNIQUE
		  );`

	_, err = postgres.Exec(query)
	if err != nil {
		logger.Log.Error("Ошибка создания таблицы:", zap.Error(err))
		postgres.Close() // Закрываем соединение, если возникла ошибка
		logger.Log.Info("Закрываем соединение с PostgreSQL")
		return nil, err
	}

	return &PgStorage{
		DB:        postgres,
		Generator: &generator.DefaultGenerator{},
	}, nil
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

func (pg *PgStorage) SetInStorage(longURL string) (string, error) {
	if !utils.IsValidURL(longURL) {
		return "", fmt.Errorf("передан невалидный URL")
	}

	// сначала проверяем — есть ли longURL в базе
	var existingShortURL string
	err := pg.DB.QueryRow(`SELECT short_url FROM urls WHERE long_url=$1`, longURL).Scan(&existingShortURL)
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

		row := pg.DB.QueryRow(`INSERT INTO urls (long_url, short_url) VALUES ($1, $2)
									 RETURNING short_url`, longURL, generatedShortURL)

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

func (pg *PgStorage) SetBatchInStorage(batch []models.BatchItem) error {
	baseURL := config.GetServerConfig().BaseURL

	tx, err := pg.DB.Begin()
	if err != nil {
		return fmt.Errorf("ошибка создания транзакции %v ", err)
	}
	defer tx.Rollback()

	// если встречаем повторяющийся long_url, то делаем фейковый апдейт его short_url этим же short_url
	// и возвращаем значение short_url.
	// Если все идет штатно, то просто вставляем long_url и сгенерированный short_url.
	// Если происходит коллизия short_url - откатываем транзакцию.
	stmt, err := tx.Prepare(`INSERT INTO urls (long_url, short_url) VALUES($1, $2) 
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
			//return fmt.Errorf("в батче присутствует невалидный URL: %v", batch[i].OriginalURL)
			continue
		}

		var shortURL string

		// генерируем короткий URL
		generatedShortURL, err := pg.Generator.MakeShortURL(length)
		if err != nil {
			return fmt.Errorf("ошибка генерации короткого URL: %v", err)
		}

		row := stmt.QueryRow(batch[i].OriginalURL, generatedShortURL)
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

func (pg *PgStorage) GetFromStorage(shortURL string) (string, error) {
	var longURL string

	row := pg.DB.QueryRow(`SELECT long_url FROM urls WHERE short_url = $1`, shortURL)
	err := row.Scan(&longURL)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			logger.Log.Error("короткий адрес не найден в базе данных", zap.String("shortURL", shortURL), zap.String("error", err.Error()))
			return "", fmt.Errorf("`%v` короткий адрес не найден в базе данных", shortURL)
		default:
			return "", fmt.Errorf("`%v` ошибка получения оригинального URL", err)
		}
	}

	return longURL, nil
}

func (pg *PgStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
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
