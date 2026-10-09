package server

import (
	"context"
	"net"

	srv "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/interceptors"
	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// RunGRPCServer Запуск gRPC сервера.
func RunGRPCServer(ctx context.Context, storage storage.Storage) error {
	lis, err := net.Listen("tcp", ":3200")
	if err != nil {
		logger.Log.Error("Ошибка создания listener при запуске gRPC сервера", zap.Error(err))
		return err
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.LoggingInterceptor,
		),
	)

	pb.RegisterURLShortenerServiceServer(server, srv.NewGRPCServer(storage))

	go func() {
		logger.Log.Info("gRPC сервер запущен", zap.String("address", ":3200"))

		if SrvErr := server.Serve(lis); err != nil {
			logger.Log.Error("Ошибка запуска gRPC сервера", zap.Error(SrvErr))
		}
	}()

	<-ctx.Done()

	logger.Log.Info("Остановка gRPC сервера")

	server.GracefulStop()

	logger.Log.Info("Сервер gRPC сервер остановлен")

	return nil
}
