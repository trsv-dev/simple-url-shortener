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

// TokenExp Определяет время жизни JWT-токена.
// Значение должно задаваться через конфигурацию или переменные окружения.
const TokenExp = time.Hour * 24

// SecretKey Используется для подписи и проверки JWT-токенов.
// Значение должно задаваться через конфигурацию или переменные окружения.
const SecretKey = "SuperSecretKey"

// BuildJWTToken Создание JWT-токена.
func BuildJWTToken() (string, error) {
	// создаем экземпляр структуры, которую будем записывать в токен
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		UserID: uuid.New().String(),
	}

	// создаем токен с claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// подписываем секретным ключом и возвращаем токен
	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GetUserID Получение ID пользователя с помощью распарсивания JWT-токена.
func GetUserID(tokenString string) (string, error) {
	// создаем пустой экземпляр Claims, куда будем распарсивать токен
	claims := &Claims{}

	// распарсиваем токен, проверяя на метод подписи
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return "", fmt.Errorf("неверный метод подписи: %v", t.Header["alg"])
		}

		return []byte(SecretKey), nil
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
func SetUserIDToContext(jwtToken string, w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	userID, err := GetUserID(jwtToken)
	if err != nil || userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return w, r
	}
	r = r.WithContext(context.WithValue(r.Context(), contextkeys.UserID, userID))
	return w, r
}

// AuthMiddleware Проверка наличия JWT-куки. При отсутствии — создание новой куки с токеном.
// Извлечение userID из токена и добавление его в контекст запроса.
func AuthMiddleware(h http.Handler) http.Handler {
	auth := func(w http.ResponseWriter, r *http.Request) {

		// ищем куку с именем "JWT" в запросе для получения токена пользователя
		cookieToken, err := r.Cookie("JWT")
		// если кука отсутствует — создаём новый токен и устанавливаем куку
		if err != nil && errors.Is(err, http.ErrNoCookie) {
			logger.Log.Debug("Cookie отсутствует", zap.Error(err))
			// создаем JWT-токен и прописываем его в куку
			jwtToken, err := BuildJWTToken()
			if err != nil {
				logger.Log.Debug("Невозможно сгенерировать токен", zap.Error(err))
				http.Error(w, fmt.Sprintf("Невозможно сгенерировать токен: %v", err), http.StatusInternalServerError)
				return
			}

			cookie := &http.Cookie{
				Name:    "JWT",
				Value:   jwtToken,
				Expires: time.Now().Add(TokenExp),
				Path:    "/",
			}

			// устанавливаем куку
			http.SetCookie(w, cookie)

			// получаем UserID и добавляем его в контекст запроса
			w, r = SetUserIDToContext(jwtToken, w, r)

			// если кука с именем "JWT" есть в запросе, то получаем из нее токен,
			// с помощью которого получаем UserID и добавляем его в контекст запроса
		} else {
			w, r = SetUserIDToContext(cookieToken.Value, w, r)
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(auth)
}
