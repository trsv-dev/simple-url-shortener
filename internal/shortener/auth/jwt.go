package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
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
