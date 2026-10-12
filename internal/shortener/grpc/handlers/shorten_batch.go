package handlers

import (
	"context"

	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ShortenBatch Функция обработки пакетного gRPC запроса (POST) в формате JSON к /api/shorten/batch.
func (s *GRPCHandler) ShortenBatch(ctx context.Context, in *pb.ShortenBatchRequest) (*pb.ShortenBatchResponse, error) {
	if len(in.Urls) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Батчи не могут быть пустыми")
	}

	// создаем слайс, который "понимает" хранилище
	batch := make([]models.BatchItem, 0, len(in.Urls))

	// заполняем его данными из in.Urls
	for _, item := range in.Urls {
		batch = append(batch, models.BatchItem{
			CorrelationID: item.CorrelationId,
			OriginalURL:   item.OriginalUrl,
		})
	}

	// отправляем батч в хранилище, там он "дозаполняется" сокращенными URL
	err := s.Storage.SetBatchInStorage(ctx, batch)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// создаем слайс для ответа сервера
	out := make([]*pb.ShortenBatchResponse_URL, 0, len(batch))
	for _, item := range batch {
		out = append(out, &pb.ShortenBatchResponse_URL{
			CorrelationId: item.CorrelationID,
			OriginalUrl:   item.OriginalURL,
			ShortUrl:      item.ShortURL,
		})
	}

	return &pb.ShortenBatchResponse{Urls: out}, nil
}
