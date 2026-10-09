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
	grpcserver "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/server"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/server"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
	"go.uber.org/zap"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// Выводит значение переменной, и если оно пустое, присваивает значение по умолчанию (`N/A`).
func valueOrNA(s string) string {
	if s == "" {
		return "N/A"
	}

	return s
}

// Выводит в Stdout инфо о сборке.
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", valueOrNA(buildVersion))
	fmt.Printf("Build date: %s\n", valueOrNA(buildDate))
	fmt.Printf("Build commit: %s\n", valueOrNA(buildCommit))
}

func main() {

	printBuildInfo()

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
		syscall.SIGINT,
		syscall.SIGQUIT,
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
	//grpcHandler := grpc_handlers.NewGRPCServer(URLStorage)

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

	go func() {
		if err := grpcserver.RunGRPCServer(ctx, URLStorage); err != nil {
			logger.Log.Error("Ошибка gRPC сервера", zap.Error(err))
		}
	}()

	err := server.RunServer(ctx, cfg.ServerAddress, cfg.EnableHTTPS, appHandler)
	if err != nil {
		panic(err)
	}
}
