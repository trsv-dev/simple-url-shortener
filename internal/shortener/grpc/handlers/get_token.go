package handlers

import (
	"context"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/auth"
	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (g *GRPCHandler) GetToken(ctx context.Context, _ *emptypb.Empty) (*pb.GetTokenResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)

	// если в запросе уже присутствует authorization - повторно токен не выдаём
	if ok && len(md.Get("authorization")) > 0 {
		return nil, status.Error(codes.AlreadyExists, "Токен уже передан в запросе")
	}

	// иначе создаем токен и отдаем пользователю
	jwtToken, err := auth.BuildJWTToken(g.SecretKey, g.TokenExp)
	if err != nil {
		logger.Log.Debug("Невозможно сгенерировать токен", zap.Error(err))
		return nil, status.Error(codes.Internal, "Невозможно сгенерировать токен")
	}

	return &pb.GetTokenResponse{Token: jwtToken}, nil
}
