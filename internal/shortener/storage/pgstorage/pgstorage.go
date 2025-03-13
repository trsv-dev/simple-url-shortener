package pgstorage

import (
	"context"
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"go.uber.org/zap"
	"time"
)

type PgStorage struct {
	DB *sql.DB
}

func InitStorage() *PgStorage {

	logger.Log.Info("Для хранения данных используется PostgreSQL")

	postgres, err := sql.Open("pgx", config.GetServerConfig().DatabaseDSN)
	if err != nil {
		panic(err)
	}

	return &PgStorage{
		DB: postgres,
	}
}

// SetInStorage Абстрактный метод для удовлетворения интерфейсу Storage.
// TODO реализовать метод для PostgreSQL
func (pg *PgStorage) SetInStorage(longURL string) (string, error) {
	return "", nil
}

// GetFromStorage Абстрактный метод для удовлетворения интерфейсу Storage.
// TODO реализовать метод для PostgreSQL
func (pg *PgStorage) GetFromStorage(shortURL string) (string, error) {
	return "", nil
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
