package interceptors

import (
	"context"
	"strconv"
	"time"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func LoggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {

	start := time.Now()

	resp, err := handler(ctx, req)

	var size int

	if msg, ok := resp.(proto.Message); ok {
		size = proto.Size(msg)
	}

	logger.Log.Debug(
		"Got incoming gRPC request",
		zap.String("method", info.FullMethod),
		zap.String("status", status.Code(err).String()),
		zap.String("duration", time.Since(start).String()),
		zap.String("size", strconv.Itoa(size)),
	)

	return resp, err
}
