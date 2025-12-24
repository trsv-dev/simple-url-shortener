package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/server"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
	"go.uber.org/zap"
)

func main() {
	// recover для логирования паник в main
	defer func() {
		if r := recover(); r != nil {
			log.Println("Паника в main:", fmt.Sprintf("%v", r))
		}
	}()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	// пытаемся прочитать файл .env
	errEnv := godotenv.Load("../../.env")
	if errEnv != nil {
		log.Println("Не удалось загрузить .env:", errEnv)
	}

	// получаем конфигурацию сервера
	cfg := config.InitConfig()

	// если логирование включено в конфигурации и уровень логирования валидный - запускаем логгер.
	if cfg.Logging && logger.IsValidLogLevel(cfg.LogLevel) {
		if err := logger.InitLogger(cfg.LogLevel); err != nil {
			panic(err)
		}
	}

	// в зависимости от конфигурации устанавливаем тип хранилища.
	URLStorage := storage.SelectStorage(cfg)

	// откладываем закрытие БД через метод Close(), полученный из хранилища, на момент
	// когда RunServer() вернет управление по той или иной причине
	defer func() {
		if URLStorage != nil {
			if err := URLStorage.Close(); err != nil {
				logger.Log.Error("Ошибка закрытия хранилища (БД)", zap.Error(err))
			}
		}

		logger.Log.Info("Хранилище успешно закрыто")
	}()

	logger.Log.Info("Сервер запущен",
		zap.String("address", cfg.ServerAddress),
		zap.String("log_level", cfg.LogLevel),
	)

	appHandler := handlers.NewAppHandler(URLStorage, cfg.SecretKey, cfg.TokenExp)

	// запускаем горутину, которая через временные промежутки делает удаление помеченных (is_delete=True) URL
	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := URLStorage.CleanUp(ctx)
				if err != nil {
					logger.Log.Error("Ошибка удаления URL", zap.Error(err))
				}

			case <-ctx.Done():
				logger.Log.Info("Горутина для фонового удаления URL завершена")
				return
			}
		}
	}()

	err := server.RunServer(ctx, cfg.ServerAddress, appHandler)
	if err != nil {
		panic(err)
	}
}
