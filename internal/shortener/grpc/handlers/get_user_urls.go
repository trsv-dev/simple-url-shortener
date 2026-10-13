package handlers

import (
	"context"

	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetUserURLs Функция обработки gRPC запроса (GET) к /api/user/urls на получение всех сокращенных пользователем URL.
func (g *GRPCHandler) GetUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.GetUserURLsResponse, error) {
	urls, err := g.Storage.GetUserURLs(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	out := make([]*pb.UserURL, 0, len(urls))

	for _, item := range urls {
		out = append(out, &pb.UserURL{
			OriginalUrl: item.OriginalURL,
			ShortUrl:    item.ShortURL,
		})
	}

	return &pb.GetUserURLsResponse{UserUrls: out}, nil
}
