package handlers

import (
	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
)

// GRPCHandler Поддерживает все необходимые методы сервера.
type GRPCHandler struct {
	pb.UnimplementedURLShortenerServiceServer
	Storage       storage.Storage
	SecretKey     string
	ServerAddress string
	GRPCPort      string
}

// NewGRPCHandler Конструктор GRPCHandler.
func NewGRPCHandler(storage storage.Storage, secretKey, serverAddress, grpcPort string) *GRPCHandler {
	return &GRPCHandler{
		Storage:       storage,
		SecretKey:     secretKey,
		ServerAddress: serverAddress,
		GRPCPort:      grpcPort,
	}
}
