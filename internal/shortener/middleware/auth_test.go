package middleware

import (
	"github.com/stretchr/testify/assert"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthMiddleware(t *testing.T) {
	t.Run("Empty cookie", func(t *testing.T) {
		// мок-хендлер, который возвращает 200 если его вызвали
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// если дошли досюда без User_ID в куке - возвращаем ошибку
			userID := r.Context().Value(contextkeys.UserID)
			if userID == nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		// оборачиваем хендлер с помощью middleware
		handler := AuthMiddleware(mockHandler)

		r := httptest.NewRequest("GET", "/api/user/urls", nil)
		w := httptest.NewRecorder()

		// выполняем запрос
		handler.ServeHTTP(w, r)

		// смотрим ответ
		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Valid cookie", func(t *testing.T) {

		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// если дошли досюда без User_ID в куке - возвращаем ошибку
			userID := r.Context().Value(contextkeys.UserID)
			if userID == nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)

		})

		handler := AuthMiddleware(mockHandler)

		r := httptest.NewRequest("GET", "/api/user/urls", nil)

		JWTtoken, err := BuildJWTToken()
		assert.NoError(t, err)
		cookie := &http.Cookie{
			Name:    "JWT",
			Value:   JWTtoken,
			Expires: time.Now().Add(time.Minute * 2),
			Path:    "/",
		}
		r.AddCookie(cookie)

		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestBuildJWTToken(t *testing.T) {
	t.Run("Build JWT token", func(t *testing.T) {
		token, err := BuildJWTToken()
		tokenParts := strings.Split(token, ".")

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, 3, len(tokenParts))
	})
}

func TestGetUserID(t *testing.T) {
	t.Run("Get UserID", func(t *testing.T) {
		token, err := BuildJWTToken()
		assert.NoError(t, err)

		userID, err := GetUserID(token)
		assert.NoError(t, err)
		assert.NotEmpty(t, userID)
		assert.Equal(t, 36, len(userID))
	})
}
