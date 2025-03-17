package pgstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/utils"
	"go.uber.org/zap"
	"strings"
	"time"
)

const length = 8

type PgStorage struct {
	DB *sql.DB
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
			long_url TEXT NOT NULL,
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
		DB: postgres,
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

	existingShortURL, err := pg.IsLongURLExists(longURL)
	if err != nil {
		return "", err
	} else if existingShortURL != "" {
		return existingShortURL, nil
	}

	for {
		shortURL, err := utils.MakeShortURL(length)
		if err != nil {
			return "", fmt.Errorf("ошибка при генерации короткого URL: %v", err)
		}

		if _, err = pg.DB.Exec(`INSERT INTO urls (long_url, short_url) VALUES ($1, $2)`, longURL, shortURL); err == nil {
			return shortURL, nil
		}

		// Проверяем, является ли ошибка нарушением уникальности поля short_url
		if !strings.Contains(err.Error(), "duplicate key value") {
			return "", fmt.Errorf("ошибка вставки в БД: %v", err)
		}
	}
}

func (pg *PgStorage) GetFromStorage(shortURL string) (string, error) {
	var longURL string

	row := pg.DB.QueryRow(`SELECT long_url FROM urls WHERE short_url = $1`, shortURL)
	err := row.Scan(&longURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Log.Error("короткий адрес не найден в базе данных", zap.String("shortURL", shortURL), zap.String("error", err.Error()))
			return "", fmt.Errorf("`%v` короткий адрес не найден в базе данных", shortURL)
		} else {
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
