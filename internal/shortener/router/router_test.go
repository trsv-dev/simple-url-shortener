package router

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/handlers"
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

// Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MockStorage) Ping() error {
	return nil
}

// Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MockStorage) Close() error {
	return nil
}

func (ms *MockStorage) SetInStorage(longURL string) (string, error) {
	if longURL == "https://invalid-example.com" || longURL == "" {
		return "", errors.New("передан невалидный URL")
	}

	// в оригинальном хранилище тут происходит проверка длинного URL
	// на соответствие требованиям и генерация короткого URL. Так как это mock хранилища, то
	// просто пропишем якобы полученный shortURL руками
	shortURL := "shortURL"
	ms.mockData[shortURL] = longURL

	return shortURL, nil
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
	testTable := []struct {
		name               string
		longURL            string
		shortURL           string
		URL                string
		method             string
		expectedStatusCode int
	}{
		{name: "POST to `/` should call PostHandler", longURL: "https://example.com", shortURL: "shortURL", URL: "/", method: http.MethodPost, expectedStatusCode: http.StatusCreated},
		{name: "POST to `/invalid-link` should receive 400", longURL: "https://invalid-example.com", URL: "/", method: http.MethodPost, expectedStatusCode: http.StatusBadRequest},
		{name: "GET to `/{id}` should call GetHandler", URL: "/shortURL", method: http.MethodGet, expectedStatusCode: http.StatusTemporaryRedirect},
		{name: "GET to `/invalid-id` should receive 400", URL: "/invalidID", method: http.MethodGet, expectedStatusCode: http.StatusBadRequest},
	}

	for _, tt := range testTable {
		mockStorage := &MockStorage{
			mockData: map[string]string{"shortURL": "https://example.com"},
		}

		h := handlers.NewAppHandler(mockStorage)
		mux := Router(h)

		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.method == http.MethodPost {
				body = strings.NewReader(tt.longURL)
			}
			executeRequest(t, mux, tt.method, tt.URL, body, tt.expectedStatusCode)
		})
	}
}

func TestRouterIntegration(t *testing.T) {

	testTable := []struct {
		name               string
		URL                string
		method             string
		longURL            string
		shortURL           string
		want               string
		expectedStatusCode int
	}{
		{name: "Get shortURL from longURL", URL: "", method: http.MethodPost, longURL: "https://example.com", want: "shortURL", expectedStatusCode: http.StatusCreated},
		{name: "Get longURL from shortURL (string)", URL: "shortURL", method: http.MethodGet, want: "https://example.com", expectedStatusCode: http.StatusTemporaryRedirect},
		{name: "Get longURL from shortURL (json)", URL: "api/shorten", method: http.MethodPost, longURL: `{"url": "https://example.com"}`, want: `{"result":"http://localhost:8080/shortURL"}`, expectedStatusCode: http.StatusCreated},
		{name: "Invalid shortURL", URL: "invalid", method: http.MethodGet, expectedStatusCode: http.StatusBadRequest},
		{name: "Bad path", URL: "/bad/path", method: http.MethodGet, expectedStatusCode: http.StatusNotFound},
		{name: "Invalid longURL (string)", URL: "", method: http.MethodPost, longURL: "https://invalid-example.com", expectedStatusCode: http.StatusBadRequest},
		{name: "Invalid longURL (json)", URL: "api/shorten", method: http.MethodPost, longURL: `{"url": "https://invalid-example.com"}`, expectedStatusCode: http.StatusBadRequest},
	}

	mockStorage := &MockStorage{
		mockData: map[string]string{"shortURL": "https://example.com"},
	}

	h := handlers.NewAppHandler(mockStorage)
	testServer := httptest.NewServer(Router(h))
	defer testServer.Close()

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(tt.method, testServer.URL+"/"+tt.URL, strings.NewReader(tt.longURL))
			require.NoError(t, err)

			client := testServer.Client()

			// Не следуем по редиректам, а просто возвращаем ответ с помощью http.ErrUseLastResponse
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}

			response, err := client.Do(request)
			require.NoError(t, err)
			defer response.Body.Close()

			if tt.method == http.MethodGet && tt.expectedStatusCode == http.StatusTemporaryRedirect {
				assert.Equal(t, tt.want, response.Header.Get("Location"))
				return
			}

			if len(tt.want) > 0 {
				body, err := io.ReadAll(response.Body)
				require.NoError(t, err)
				assert.Equal(t, tt.want, strings.TrimPrefix(string(body), config.GetServerConfig().BaseURL+"/"))
				return
			}

			assert.Equal(t, tt.expectedStatusCode, response.StatusCode)
		})
	}
}
