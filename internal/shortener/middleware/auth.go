package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/auth"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"go.uber.org/zap"
)

// AuthMiddleware Фабрика AuthMiddleware.
// Проверка наличия JWT-куки. При отсутствии — создание новой куки с токеном.
// Извлечение userID из токена и добавление его в контекст запроса.
func AuthMiddleware(secretKey string, tokenExp time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ищем куку с именем "JWT" в запросе
			cookieToken, err := r.Cookie("JWT")
			var jwtToken string

			switch {
			// Если кука отсутствует — создаём новый токен
			case errors.Is(err, http.ErrNoCookie):
				logger.Log.Debug("Cookie отсутствует", zap.Error(err))
				// создаем JWT-токен и прописываем его в куку
				jwtToken, err = auth.BuildJWTToken(secretKey, tokenExp)
				if err != nil {
					logger.Log.Debug("Невозможно сгенерировать токен", zap.Error(err))
					http.Error(w, fmt.Sprintf("Невозможно сгенерировать токен: %v", err), http.StatusInternalServerError)
					return
				}

				cookie := &http.Cookie{
					Name:    "JWT",
					Value:   jwtToken,
					Expires: time.Now().Add(tokenExp),
					Path:    "/",
				}

				// устанавливаем куку
				http.SetCookie(w, cookie)
			// Ошибка при получении cookie.
			case err != nil:
				w.WriteHeader(http.StatusUnauthorized)
				return
			// JWT уже есть — берём токен из cookie.
			default:
				jwtToken = cookieToken.Value
			}

			// Получаем UserID и добавляем его в контекст запроса.
			w, r, ok := auth.SetUserIDToContext(jwtToken, secretKey, w, r)
			if !ok {
				// SetUserIDToContext уже отправил 401.
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
