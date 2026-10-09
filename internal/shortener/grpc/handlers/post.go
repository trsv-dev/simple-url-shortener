package handlers

import (
	"context"
	"errors"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Post Функция обработки gRPC запроса (POST) к "/".
func (s *GRPCServer) Post(ctx context.Context, in *pb.PostRequest) (*pb.PostResponse, error) {
	shortURL, err := s.Storage.SetInStorage(ctx, in.OriginalUrl)

	var conflictErr *errs.ErrLongURLConflict

	if err != nil {
		if errors.As(err, &conflictErr) {
			return &pb.PostResponse{
				ShortenedUrl: config.GetServerConfig().BaseURL + "/" + conflictErr.ShortURL,
			}, status.Error(codes.AlreadyExists, conflictErr.Error())
		}

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pb.PostResponse{
		ShortenedUrl: config.GetServerConfig().BaseURL + "/" + shortURL,
	}, nil
}
