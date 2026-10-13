package handlers

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// PingDB При gRPC запросе (GET) к "/ping" проверяет соединение с базой данных
func (g *GRPCHandler) PingDB(ctx context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	if err := g.Storage.Ping(ctx); err != nil {
		//w.WriteHeader(http.StatusInternalServerError)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}
