package server

import (
	"context"
	"net"
	"strings"

	srv "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/handlers"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/interceptors"
	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// RunGRPCServer Запуск gRPC сервера.
func RunGRPCServer(ctx context.Context, grpcHandler *srv.GRPCHandler) error {
	serverAddress := strings.Split(grpcHandler.ServerAddress, `:`)[0]
	grpcAddress := serverAddress + `:` + grpcHandler.GRPCPort

	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		logger.Log.Error("Ошибка создания listener при запуске gRPC сервера", zap.Error(err))
		return err
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.LoggingInterceptor,
			interceptors.AuthInterceptor(grpcHandler.SecretKey),
		),
	)

	pb.RegisterURLShortenerServiceServer(server, grpcHandler)

	go func() {
		logger.Log.Info("Сервер gRPC запущен", zap.String("address", grpcAddress))

		if SrvErr := server.Serve(lis); err != nil {
			logger.Log.Error("Ошибка запуска gRPC сервера", zap.Error(SrvErr))
		}
	}()

	<-ctx.Done()

	logger.Log.Info("Остановка gRPC сервера")

	server.GracefulStop()

	logger.Log.Info("Сервер gRPC остановлен")

	return nil
}
