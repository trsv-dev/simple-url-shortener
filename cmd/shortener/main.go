package main

import (
	"context"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/server"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage, err := server.RunServer(ctx)
	if err != nil {
		panic(err)
	}

	// откладываем закрытие БД через метод Close(), полученный из хранилища, на момент
	// когда RunServer() вернет управление по той или иной причине
	defer func() {
		if storage != nil {
			if err = storage.Close(); err != nil {
				logger.Log.Error("Ошибка закрытия хранилища (БД)", zap.Error(err))
			}
		}
	}()
}
