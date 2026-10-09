package handlers

import (
	"context"
	"errors"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetOriginalURL Функция обработки gRPC запроса (GET) к "/{id}".
func (s *GRPCServer) GetOriginalURL(ctx context.Context, in *pb.GetOriginalURLRequest) (*pb.GetOriginalURLResponse, error) {
	id := in.ShortUrl

	value, err := s.Storage.GetFromStorage(ctx, id)
	var errURLDeleted *errs.ErrURLDeleted

	if err != nil {
		switch {
		case errors.As(err, &errURLDeleted):
			return nil, status.Error(codes.NotFound, err.Error())
		default:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	return &pb.GetOriginalURLResponse{
		OriginalUrl: value,
	}, nil
}
