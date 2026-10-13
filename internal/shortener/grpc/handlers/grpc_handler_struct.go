package handlers

import (
	"time"

	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
)

// GRPCHandler Поддерживает все необходимые методы сервера.
type GRPCHandler struct {
	pb.UnimplementedURLShortenerServiceServer
	Storage       storage.Storage
	SecretKey     string
	TokenExp      time.Duration
	ServerAddress string
	GRPCPort      string
}

// NewGRPCHandler Конструктор GRPCHandler.
func NewGRPCHandler(storage storage.Storage, grpcConfig GRPCConfig) *GRPCHandler {
	return &GRPCHandler{
		Storage:       storage,
		ServerAddress: grpcConfig.ServerAddress,
		GRPCPort:      grpcConfig.GRPCPort,
		SecretKey:     grpcConfig.SecretKey,
		TokenExp:      grpcConfig.TokenExp,
	}
}

// GRPCConfig Структура для передачи зависимостей GRPC сервера
type GRPCConfig struct {
	ServerAddress string
	GRPCPort      string
	SecretKey     string
	TokenExp      time.Duration
}
