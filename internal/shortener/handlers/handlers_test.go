package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/contextkeys"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
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

// GetUserURLs Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MockStorage) GetUserURLs(ctx context.Context) ([]models.BatchItem, error) {

	if ctx.Value("User_ID") != "" {
		var batch []models.BatchItem

		item := models.BatchItem{
			OriginalURL: "https://example.com",
			ShortURL:    "abcd1234",
		}

		batch = append(batch, item)

		return batch, nil
	}

	return nil, fmt.Errorf("ошибка получения url")
}

// Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MockStorage) MarkAsDelete(ctx context.Context, urls []string) error {
	return nil
}

// Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MockStorage) CleanUp(ctx context.Context) error {
	return nil
}

// Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MockStorage) Ping(ctx context.Context) error {
	return nil
}

// Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MockStorage) Close() error {
	return nil
}

// Абстрактный метод для удовлетворения интерфейсу Storage.
func (ms *MockStorage) SetBatchInStorage(ctx context.Context, batch []models.BatchItem) error {
	// Имитируем процесс сокращения URL и добавления его в структуру batch
	for i := range batch {
		if batch[i].OriginalURL != "" {
			// Имитируем сокращение
			shortURL := "http://127.0.0.1:8080/" + "shortURL" + strconv.Itoa(i+1) // Генерация короткого URL
			batch[i].ShortURL = shortURL
		}
	}

	return nil // Возвращаем nil, чтобы тест проходил успешно
}

func (ms *MockStorage) SetInStorage(ctx context.Context, longURL string) (string, error) {
	if longURL == "https://invalid-example.com" || longURL == "" {
		return "", errors.New("передан невалидный URL")
	}

	if longURL == "123" {
		return "", fmt.Errorf("ошибка при сохранении URL: %v", http.StatusInternalServerError)
	}

	return ms.mockData[longURL], nil
}

func (ms *MockStorage) GetFromStorage(ctx context.Context, shortURL string) (string, error) {
	value, ok := ms.mockData[shortURL]
	if !ok && shortURL != "deletedShortURL" && shortURL != "badShortUrl" {
		return "", fmt.Errorf("`%v` не найден в базе данных", shortURL)
	}

	switch {
	case shortURL == "deletedShortURL":
		errURLDeleted := errs.NewErrURLDeleted("deletedShortURL", true, errors.New("URL был помечен как удаленный"))
		return "", errURLDeleted
	case shortURL == "badShortUrl":
		return "", fmt.Errorf("невалидный URL")
	default:
		return value, nil
	}
}

// Структура, которая имитирует хранилище, выдающее ошибку при обращении к эндпоинту "/ping".
type PingError struct {
	MockStorage // для имплементации интерфейса storage.Storage, чтобы не описывать абстрактные методы
	pingErr     error
}

func (p *PingError) Ping(ctx context.Context) error {
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
		{name: "Mark as deleted shortURL", requestMethod: http.MethodGet, shortURL: "/deletedShortURL", expectedStatus: http.StatusGone},
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

func TestAppHandler_Batch(t *testing.T) {
	// для APIShorten хендлера передаем структуру в виде "длинная ссылка: короткая ссылка"
	mockStorage := &MockStorage{
		mockData: map[string]string{"https://example.com": "shortURL"},
	}
	URL := "/api/shorten/batch"

	testTable := []struct {
		name           string
		method         string
		requestBody    string
		expectedData   string
		expectedStatus int
	}{
		{
			name:   "Empty batch request",
			method: http.MethodPost, requestBody: `[]`,
			expectedData:   "Батчи не могут быть пустыми\n",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid batch request",
			method:         http.MethodPost,
			requestBody:    `[{"correlation_id": "abcd1234", "original_url": "https://example.com"}]`,
			expectedData:   `[{"correlation_id":"abcd1234","original_url":"https://example.com","short_url":"http://127.0.0.1:8080/shortURL1"}]`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Wrong method (GET)",
			method:         http.MethodGet,
			requestBody:    `[{"correlation_id": "abcd1234", "original_url": "https://example.com"}]`,
			expectedData:   "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid JSON",
			method:         http.MethodPost,
			requestBody:    `[{correlation_id": "abcd1234", "original_url": "https://example.com"}]`,
			expectedData:   "Невалидная структура запроса\n",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAppHandler(mockStorage)
			body := strings.NewReader(tt.requestBody)
			r := httptest.NewRequest(tt.method, URL, body)
			w := httptest.NewRecorder()

			h.Batch(w, r)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.expectedStatus, res.StatusCode)
			assert.Equal(t, tt.expectedData, w.Body.String())
		})
	}
}

func TestAppHandler_UserURLs(t *testing.T) {
	mockStorage := &MockStorage{
		mockData: map[string]string{"https://example.com": "shortURL"},
	}

	h := NewAppHandler(mockStorage)
	URL := "/api/user/urls"

	testTable := []struct {
		name           string
		method         string
		userID         any // Можно string или nil
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Get user urls",
			method:         http.MethodGet,
			userID:         "123", // значение не проверяется, но наличие важно
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"original_url":"https://example.com","short_url":"abcd1234"}]`,
		},
		{
			name:           "Get no error when user_id is empty",
			method:         http.MethodGet,
			userID:         "", // имитируем пустого пользователя
			expectedStatus: http.StatusOK,
			expectedBody:   "", // тело не важно
		},
		{
			name:           "Invalid method",
			method:         http.MethodPost,
			userID:         "123",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, URL, nil)

			// добавляем контекст, если явно указан userID
			if tt.userID != nil {
				ctx := context.WithValue(req.Context(), contextkeys.UserID, tt.userID)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			h.UserURLs(w, req)

			res := w.Result()
			defer res.Body.Close()

			body, _ := io.ReadAll(res.Body)

			assert.Equal(t, tt.expectedStatus, res.StatusCode)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, string(body))
			}
		})
	}
}

func TestAppHandler_Delete(t *testing.T) {
	mockStorage := &MockStorage{
		mockData: map[string]string{"https://example.com": "shortURL"},
	}

	h := NewAppHandler(mockStorage)
	URL := "/api/user/urls"

	testTable := []struct {
		name           string
		method         string
		shortURL       string
		expectedStatus int
	}{
		{name: "Normal deletion shortURL", method: "DELETE", shortURL: `["shortURL"]`, expectedStatus: http.StatusAccepted},
		{name: "Error shortURL (not json-formatted)", method: "DELETE", shortURL: `[shortURL]`, expectedStatus: http.StatusInternalServerError},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {

			r := httptest.NewRequest(tt.method, URL, strings.NewReader(tt.shortURL))
			r.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			h.Delete(w, r)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.expectedStatus, res.StatusCode)
		})
	}
}
