package handlers

import (
	"context"

	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// DeleteURLs Функция обработки gRPC запроса (DELETE) к /api/user/urls, которая в теле запроса.
func (s *GRPCHandler) DeleteURLs(ctx context.Context, in *pb.DeleteURLsRequest) (*emptypb.Empty, error) {
	err := s.Storage.MarkAsDelete(ctx, in.ShortUrls)

	if err != nil {
		logger.Log.Info("Ошибка при удалении URL", zap.String("delete url error", err.Error()))

		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}
