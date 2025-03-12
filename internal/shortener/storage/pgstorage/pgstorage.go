package pgstorage

import (
	"context"
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
	"go.uber.org/zap"
	"time"
)

type PgStorage struct {
	storage.Storage
	db *sql.DB
}

func InitStorage() *PgStorage {

	logger.Log.Info("Для хранения данных используется PostgreSQL")

	postgres, err := sql.Open("pgx", config.GetServerConfig().DatabaseDSN)
	if err != nil {
		panic(err)
	}

	return &PgStorage{
		db: postgres,
	}
}

func (pg *PgStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := pg.db.PingContext(ctx); err != nil {
		logger.Log.Error("База данных Postgres не отвечает", zap.String("error", err.Error()))
		return err
	}

	return nil
}

func (pg *PgStorage) Close() error {
	return pg.db.Close()
}
