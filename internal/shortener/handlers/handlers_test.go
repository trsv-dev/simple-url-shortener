package handlers

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

// Структура, которая имитирует ошибку при чтении. Когда обработчик получает её,
// он должен корректно обрабатывать ситуацию, которая приводит к внутренней ошибке сервера.
type ErrorReader struct{}

func (e *ErrorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("mock read error")
}

// Структура, которая имитирует работу хранилища URL.
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

func TestGetHandler(t *testing.T) {
	mockStorage := &MockStorage{
		mockData: map[string]string{"ShortURL": "https://example.com"},
	}

	testTable := []struct {
		name             string
		requestMethod    string
		shortURL         string
		expectedStatus   int
		expectedLocation string
	}{
		{name: "Invalid (empty) shortURL", requestMethod: http.MethodGet, shortURL: "/", expectedStatus: http.StatusBadRequest},
		{name: "Invalid (bad) shortURL", requestMethod: http.MethodGet, shortURL: "/badShortUrl", expectedStatus: http.StatusBadRequest},
		{name: "Valid shortURL", requestMethod: http.MethodGet, shortURL: "/ShortURL", expectedStatus: http.StatusTemporaryRedirect, expectedLocation: "https://example.com"},
		{name: "Invalid Method (POST) with valid shortURL", requestMethod: http.MethodPost, shortURL: "/ShortURL", expectedStatus: http.StatusBadRequest},
		{name: "Invalid Method (DELETE) with valid shortURL", requestMethod: http.MethodDelete, shortURL: "/ShortURL", expectedStatus: http.StatusBadRequest},
		{name: "Invalid Method (DELETE) with valid shortURL", requestMethod: http.MethodPut, shortURL: "/ShortURL", expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.requestMethod, tt.shortURL, nil)
			w := httptest.NewRecorder()

			GetHandler(mockStorage, w, r)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.expectedStatus, res.StatusCode)

			if tt.expectedLocation != "" {
				assert.Equal(t, tt.expectedLocation, res.Header.Get("Location"))
			}
		})
	}
}

func TestPostHandler(t *testing.T) {
	mockStorage := &MockStorage{
		mockData: map[string]string{"https://example.com": "ShortURL"},
	}

	testTable := []struct {
		name           string
		requestMethod  string
		URL            string
		longURL        string
		expectedBody   string
		expectedStatus int
	}{
		{name: "Invalid longURL", requestMethod: http.MethodPost, longURL: "https://invalid-example.com", URL: "/", expectedBody: "", expectedStatus: http.StatusBadRequest},
		{name: "Empty longURL", requestMethod: http.MethodPost, longURL: "", URL: "/", expectedBody: "", expectedStatus: http.StatusBadRequest},
		{name: "Valid longURL", requestMethod: http.MethodPost, longURL: "https://example.com", URL: "/", expectedBody: fmt.Sprintf("http://%v/ShortURL", serverHost), expectedStatus: http.StatusCreated},
		{name: "Valid longURL with invalid URL", requestMethod: http.MethodPost, longURL: "https://example.com", URL: "/invalid", expectedStatus: http.StatusBadRequest},
		{name: "Valid longURL and wrong method (GET)", requestMethod: http.MethodGet, longURL: "https://example.com", URL: "/", expectedStatus: http.StatusBadRequest},
		{name: "Valid longURL and wrong method (PUT)", requestMethod: http.MethodPut, longURL: "https://example.com", URL: "/", expectedStatus: http.StatusBadRequest},
		{name: "Valid longURL and wrong method (DELETE)", requestMethod: http.MethodDelete, longURL: "https://example.com", URL: "/", expectedStatus: http.StatusBadRequest},
		{name: "Internal Server Error", requestMethod: http.MethodPost, longURL: "bodyWithError", URL: "/", expectedStatus: http.StatusInternalServerError},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {

			// Если в тесте надо проверить 500 ошибку, то подсовываем тело с ошибкой.
			var body io.Reader
			if tt.expectedStatus == http.StatusInternalServerError {
				body = &ErrorReader{}
				// Для всех других тестов создаем нормальное тело запроса.
			} else {
				body = strings.NewReader(tt.longURL)
			}

			r := httptest.NewRequest(tt.requestMethod, tt.URL, body)
			w := httptest.NewRecorder()

			PostHandler(mockStorage, w, r)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.expectedStatus, res.StatusCode)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}
