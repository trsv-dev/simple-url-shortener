package router

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockStorage Структура, которая имитирует работу хранилища URL.
type MockStorage struct {
	mockData map[string]string
}

func (ms *MockStorage) SetInStorage(longURL string) (string, error) {
	if longURL == "https://invalid-example.com" || longURL == "" {
		return "", errors.New("передан невалидный url")
	}
	return ms.mockData[longURL], nil
}

func (ms *MockStorage) GetFromStorage(shortURL string) (string, error) {
	value, ok := ms.mockData[shortURL]
	if !ok {
		return "", fmt.Errorf("`%v` не найден в базе данных", shortURL)
	}

	return value, nil
}

// Функция, выполняющая запрос и в GET, и в POST методах.
func executeRequest(t *testing.T, mux http.Handler, method, url string, body io.Reader, expectedStatusCode int) {
	r := httptest.NewRequest(method, url, body)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)
	res := w.Result()
	defer res.Body.Close()

	if method == http.MethodGet && expectedStatusCode == http.StatusTemporaryRedirect {
		assert.Equal(t, "https://example.com", res.Header.Get("Location"))
	}

	assert.Equal(t, expectedStatusCode, res.StatusCode)
}

func TestRouter(t *testing.T) {
	tests := []struct {
		name               string
		longURL            string
		shortURL           string
		URL                string
		method             string
		expectedStatusCode int
	}{
		{name: "POST to `/` should call PostHandler", longURL: "https://example.com", shortURL: "shortURL", URL: "/", method: http.MethodPost, expectedStatusCode: http.StatusCreated},
		{name: "POST to `/invalid-link` should receive 400", longURL: "https://invalid-example.com", URL: "/", method: http.MethodPost, expectedStatusCode: http.StatusBadRequest},
		{name: "GET to `/{id}` should call GetHandler", longURL: "https://example.com", URL: "/shortURL", method: http.MethodGet, expectedStatusCode: http.StatusTemporaryRedirect},
		{name: "GET to `/invalid-id` should receive 400", longURL: "https://example.com", URL: "/invalidID", method: http.MethodGet, expectedStatusCode: http.StatusBadRequest},
	}

	for _, tt := range tests {
		mockStorage := &MockStorage{
			mockData: map[string]string{"shortURL": "https://example.com"},
		}
		mux := Router(mockStorage)

		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.method == http.MethodPost {
				body = strings.NewReader(tt.longURL)
			}
			executeRequest(t, mux, tt.method, tt.URL, body, tt.expectedStatusCode)
		})
	}
}
