package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Shorten Функция обработки gRPC запроса (POST) к "/api/shorten"
func (g *GRPCHandler) Shorten(ctx context.Context, in *pb.ShortenRequest) (*pb.ShortenResponse, error) {
	baseURL := config.GetServerConfig().BaseURL
	shortURL, err := g.Storage.SetInStorage(ctx, in.OriginalUrl)
	var conflictErr *errs.ErrLongURLConflict
	if err != nil {
		switch {
		case errors.As(err, &conflictErr):
			// если URL уже существует в базе - возвращаем через структуру кастомной ошибки
			// shortURL, который соответствует этому URL
			answer := baseURL + "/" + conflictErr.ShortURL
			return nil, status.Error(codes.AlreadyExists, answer)
		case err.Error() == "передан невалидный URL":
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	// формируем URL, добавляем к полученному короткому URL адрес сервера
	answer := fmt.Sprint(baseURL + "/" + shortURL)

	return &pb.ShortenResponse{ShortenedUrl: answer}, nil
}
