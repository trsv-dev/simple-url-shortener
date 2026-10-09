package handlers

import (
	"context"
	"net"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	pb "github.com/trsv-dev/simple-url-shortener/internal/shortener/grpc/proto"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Stats Функция обработки gRPC запроса (GET) к /api/internal/stats для получения статистики
// (общее кол-во пользователей и общее кол-во сохраненных ссылок).
// Доступ только из доверенной подсети, задающейся через переменную окружения TRUSTED_SUBNET или флаг -t,
// например, 127.0.0.0/8 или -t 192.168.1.0/24.
// При пустом значении доступ запрещён для любого входящего запроса.
func (s *GRPCServer) Stats(ctx context.Context, in *emptypb.Empty) (*pb.GetStatsResponse, error) {
	// из конфига сервера получаем доверенные маски подсети
	trustedSubnet := config.GetServerConfig().TrustedSubnet
	if trustedSubnet == "" {
		return nil, status.Error(codes.PermissionDenied, "доступ запрещён")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "metadata отсутствует")
	}

	ipValues := md.Get("x-real-ip")
	if len(ipValues) == 0 {
		return nil, status.Error(codes.InvalidArgument, "X-Real-IP отсутствует")
	}

	ipStr := ipValues[0]

	// парсим его
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, status.Error(codes.PermissionDenied, "доступ запрещён")
	}

	// парсим доверенные маски
	_, subnet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		logger.Log.Error("Некорректная доверенная подсеть", zap.String("subnet", trustedSubnet), zap.Error(err))
		return nil, status.Error(codes.Internal, "доступ запрещён")
	}

	// проверяем, входит ли ip в доверенную подсеть
	if !subnet.Contains(ip) {
		logger.Log.Debug("Попытка доступа к статистике с недопустимого адреса", zap.String("remote address", ip.String()))
		return nil, status.Error(codes.PermissionDenied, "доступ запрещён")
	}

	urlsStat, err := s.Storage.GetURLsStat(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	usersStat, err := s.Storage.GetUsersStat(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	stat := models.Stats{URLs: urlsStat, Users: usersStat}

	return &pb.GetStatsResponse{
		UsersAmount: uint32(stat.Users),
		UrlsAmount:  uint32(stat.URLs),
	}, nil
}
