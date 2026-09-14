package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"go.uber.org/zap"
)

// Claims Описывает набор JWT-утверждений,
// используемых для аутентификации пользователя.
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// BuildJWTToken Создание JWT-токена.
func BuildJWTToken(secretKey string, tokenExp time.Duration) (string, error) {
	// создаем экземпляр структуры, которую будем записывать в токен
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserID: uuid.New().String(),
	}

	// создаем токен с claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// подписываем секретным ключом и возвращаем токен
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GetUserID Получение ID пользователя с помощью распарсивания JWT-токена.
func GetUserID(tokenString string, secretKey string) (string, error) {
	// создаем пустой экземпляр Claims, куда будем распарсивать токен
	claims := &Claims{}

	// распарсиваем токен, проверяя на метод подписи
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return "", fmt.Errorf("неверный метод подписи: %v", t.Header["alg"])
		}

		return []byte(secretKey), nil
	})

	if err != nil {
		return "", fmt.Errorf("ошибка парсинга токена: %v", err)
	}

	// проверяем токен на валидность
	if !token.Valid {
		return "", fmt.Errorf("токен недействителен")
	}

	// возвращаем значение UserID из экземпляра структуры
	return claims.UserID, nil
}

// SetUserIDToContext Извлечение userID из JWT-токена и добавление его в контекст запроса.
// В случае ошибки устанавливает статус Unauthorized.
func SetUserIDToContext(jwtToken string, secretKey string, w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request, bool) {
	userID, err := GetUserID(jwtToken, secretKey)
	if err != nil || userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return w, r, false
	}
	r = r.WithContext(context.WithValue(r.Context(), contextkeys.UserID, userID))
	return w, r, true
}

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
				jwtToken, err = BuildJWTToken(secretKey, tokenExp)
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
			w, r, ok := SetUserIDToContext(jwtToken, secretKey, w, r)
			if !ok {
				// SetUserIDToContext уже отправил 401.
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
