package interceptors

import (
	"context"
	"strings"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/auth"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthInterceptor(secretKey string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {

		// если пользователь обращается за токеном - пропускаем
		if info.FullMethod == "/grpc_server.URLShortenerService/GetToken" {
			return handler(ctx, req)
		}

		// получаем метаданные входящего gRPC-запроса.
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(
				codes.Unauthenticated,
				"Отсутствуют метаданные аутентификации",
			)
		}

		// ищем заголовок authorization
		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "Отсутствует хедер авторизации")
		}

		// ожидаемый формат: Bearer <JWT>
		parts := strings.Fields(values[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return nil, status.Error(codes.Unauthenticated, "Неверный хедер авторизации")
		}

		// используем существующую проверку JWT
		userID, err := auth.GetUserID(parts[1], secretKey)
		if err != nil || userID == "" {
			return nil, status.Error(codes.Unauthenticated, "Невалидный токен")
		}

		// кладём userID в контекст, как в HTTP middleware
		ctx = context.WithValue(ctx, contextkeys.UserID, userID)

		// продолжаем выполнение RPC-обработчика
		return handler(ctx, req)
	}
}
