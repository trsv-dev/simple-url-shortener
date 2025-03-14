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

	if longURL == "123" {
		return "", fmt.Errorf("ошибка при сохранении URL: %v", http.StatusInternalServerError)
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

// Структура, которая имитирует хранилище, выдающее ошибку при обращении к эндпоинту "/ping".
type PingError struct {
	MockStorage // для имплементации интерфейса storage.Storage, чтобы не описывать абстрактные методы
	pingErr     error
}

func (p *PingError) Ping() error {
	return p.pingErr
}

func TestGetHandler(t *testing.T) {
	// для Get хендлера передаем структуру в виде "короткая ссылка: длинная ссылка"
	mockStorage := &MockStorage{
		mockData: map[string]string{"shortURL": "https://example.com"},
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
		{name: "Valid shortURL", requestMethod: http.MethodGet, shortURL: "/shortURL", expectedStatus: http.StatusTemporaryRedirect, expectedLocation: "https://example.com"},
		{name: "Invalid Method (POST) with valid shortURL", requestMethod: http.MethodPost, shortURL: "/shortURL", expectedStatus: http.StatusBadRequest},
		{name: "Invalid Method (DELETE) with valid shortURL", requestMethod: http.MethodDelete, shortURL: "/shortURL", expectedStatus: http.StatusBadRequest},
		{name: "Invalid Method (DELETE) with valid shortURL", requestMethod: http.MethodPut, shortURL: "/shortURL", expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.requestMethod, tt.shortURL, nil)
			w := httptest.NewRecorder()

			h := NewAppHandler(mockStorage)
			h.GetHandler(w, r)

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
	// для Post хендлера передаем структуру в виде "длинная ссылка: короткая ссылка"
	mockStorage := &MockStorage{
		mockData: map[string]string{"https://example.com": "shortURL"},
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
		{name: "Valid longURL", requestMethod: http.MethodPost, longURL: "https://example.com", URL: "/", expectedBody: "http://localhost:8080/shortURL", expectedStatus: http.StatusCreated},
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

			h := NewAppHandler(mockStorage)
			h.PostHandler(w, r)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.expectedStatus, res.StatusCode)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestAPIShorten(t *testing.T) {
	// для APIShorten хендлера передаем структуру в виде "длинная ссылка: короткая ссылка"
	mockStorage := &MockStorage{
		mockData: map[string]string{"https://example.com": "shortURL"},
	}

	testTable := []struct {
		name                string
		requestMethod       string
		URL                 string
		requestBody         string
		expectedStatus      int
		expectedBody        string
		expectedContentType string
	}{
		{name: "Valid longURL in JSON", requestMethod: http.MethodPost, URL: "/api/shorten", requestBody: `{"url": "https://example.com"}`, expectedStatus: http.StatusCreated, expectedBody: `{"result":"http://localhost:8080/shortURL"}`, expectedContentType: "application/json"},
		{name: "Invalid longURL in JSON", requestMethod: http.MethodPost, URL: "/api/shorten", requestBody: `{"url": "https://invalid-example.com"}`, expectedStatus: http.StatusBadRequest},
		{name: "Invalid body in JSON", requestMethod: http.MethodPost, URL: "/api/shorten", requestBody: "", expectedStatus: http.StatusInternalServerError},
		{name: "Not URL in JSON", requestMethod: http.MethodPost, URL: "/api/shorten", requestBody: `{"url": "123"}`, expectedStatus: http.StatusInternalServerError},
		{name: "Invalid JSON", requestMethod: http.MethodPost, URL: "/api/shorten", requestBody: `{"url": "https://invalid-example.com"`, expectedStatus: http.StatusBadRequest},
		{name: "Valid longURL in JSON and wrong method (GET)", requestMethod: http.MethodGet, URL: "/api/shorten", requestBody: `{"url": "https://example.com"}`, expectedStatus: http.StatusMethodNotAllowed},
		{name: "Valid longURL in JSON and wrong method (PUT)", requestMethod: http.MethodPut, URL: "/api/shorten", requestBody: `{"url": "https://example.com"}`, expectedStatus: http.StatusMethodNotAllowed},
		{name: "Valid longURL in JSON and wrong method (DELETE)", requestMethod: http.MethodDelete, URL: "/api/shorten", requestBody: `{"url": "https://example.com"}`, expectedStatus: http.StatusMethodNotAllowed},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {

			// Если в тесте надо проверить 500 ошибку, то подсовываем тело с ошибкой.
			var body io.Reader
			if tt.expectedStatus == http.StatusInternalServerError && tt.requestBody == "" {
				body = &ErrorReader{}
				// Для всех других тестов создаем нормальное тело запроса.
			} else {
				body = strings.NewReader(tt.requestBody)
			}

			r := httptest.NewRequest(tt.requestMethod, tt.URL, body)
			w := httptest.NewRecorder()

			h := NewAppHandler(mockStorage)
			h.APIShorten(w, r)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.expectedStatus, res.StatusCode)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}

			if tt.expectedContentType != "" {
				assert.Equal(t, tt.expectedContentType, res.Header.Get("Content-Type"))
			}
		})
	}
}

func TestPing(t *testing.T) {
	testTable := []struct {
		name           string
		method         string
		storageError   error
		expectedStatus int
	}{
		{name: "method GET - status OK", method: http.MethodGet, storageError: nil, expectedStatus: http.StatusOK},
		{name: "method GET - storage error", method: http.MethodGet, storageError: errors.New("db error"), expectedStatus: http.StatusInternalServerError},
		{name: "method POST - method not allowed", method: http.MethodPost, storageError: nil, expectedStatus: http.StatusMethodNotAllowed},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			pingError := &PingError{pingErr: tt.storageError}
			handler := AppHandler{storage: pingError}

			r := httptest.NewRequest(tt.method, "/ping", nil)
			w := httptest.NewRecorder()

			handler.Ping(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
