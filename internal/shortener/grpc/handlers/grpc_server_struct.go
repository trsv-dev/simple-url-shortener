package handlers

import (
	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
)

// GRPCServer Поддерживает все необходимые методы сервера.
type GRPCServer struct {
	pb.UnimplementedURLShortenerServiceServer
	Storage storage.Storage
}

// NewGRPCServer Конструктор GRPCServer.
func NewGRPCServer(storage storage.Storage) *GRPCServer {
	return &GRPCServer{Storage: storage}
}
