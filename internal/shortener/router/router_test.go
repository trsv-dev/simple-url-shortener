package router

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		return "", errors.New("передан невалидный URL")
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
		{name: "Get shortURL from longURL", URL: "/", method: http.MethodPost, longURL: "https://example.com", want: "shortURL", expectedStatusCode: http.StatusCreated},
		{name: "Get longURL from shortURL", URL: "/shortURL", method: http.MethodGet, want: "https://example.com", expectedStatusCode: http.StatusTemporaryRedirect},
		{name: "Invalid shortURL", URL: "/invalid", method: http.MethodGet, expectedStatusCode: http.StatusBadRequest},
		{name: "Bad path", URL: "/bad/path", method: http.MethodGet, expectedStatusCode: http.StatusNotFound},
		{name: "Invalid longURL", URL: "/", method: http.MethodPost, longURL: "https://invalid-example.com", expectedStatusCode: http.StatusBadRequest},
	}

	mockStorage := &MockStorage{
		mockData: map[string]string{"shortURL": "https://example.com"},
	}

	testServer := httptest.NewServer(Router(mockStorage))
	defer testServer.Close()

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {

			request, err := http.NewRequest(tt.method, testServer.URL+tt.URL, strings.NewReader(tt.longURL))
			require.NoError(t, err)

			client := testServer.Client()

			// Не следуем по редиректам, а просто возвращаем ответ с помощью http.ErrUseLastResponse
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}

			response, err := client.Do(request)
			require.NoError(t, err)
			defer response.Body.Close()

			fmt.Println(mockStorage)

			if tt.method == http.MethodGet && tt.expectedStatusCode == http.StatusTemporaryRedirect {
				assert.Equal(t, tt.want, response.Header.Get("Location"))
			}
			assert.Equal(t, tt.expectedStatusCode, response.StatusCode)
		})
	}
}
