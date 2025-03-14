package middleware

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
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

	if longURL == "123" {
		return "", fmt.Errorf("ошибка при сохранении URL: %v", http.StatusInternalServerError)
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

func TestGzipCompressorMiddleware(t *testing.T) {
	storage := MockStorage{mockData: map[string]string{"shortURL": "https://example.com"}}
	testApp := handlers.NewAppHandler(&storage)

	mockHandler := GzipCompressorMiddleware(http.HandlerFunc(testApp.PostHandler))
	testSrv := httptest.NewServer(mockHandler)
	defer testSrv.Close()

	testTable := []struct {
		name             string
		contentType      string
		acceptEncoding   string
		requestBody      string
		expectedStatus   int
		expectedEncoding string
		expectedResponse string
	}{
		{name: "Accepts gzip and right Content-Type", contentType: "text/html", acceptEncoding: "gzip",
			requestBody: "https://example.com", expectedStatus: http.StatusCreated,
			expectedEncoding: "gzip", expectedResponse: "shortURL",
		},
		{name: "Accepts gzip, but Content-Type not text/html or application/json",
			contentType: "application/x-www-form-urlencoded", acceptEncoding: "gzip",
			requestBody: "https://example.com", expectedStatus: http.StatusCreated,
			expectedEncoding: "", expectedResponse: "shortURL",
		},
		{name: "Accepts gzip, but Accept-Encoding not gzip",
			contentType: "text/html", acceptEncoding: "",
			requestBody: "https://example.com", expectedStatus: http.StatusCreated,
			expectedEncoding: "", expectedResponse: "shortURL",
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.Buffer{}
			if tt.acceptEncoding == "gzip" {
				gzw := gzip.NewWriter(&buf)
				_, err := gzw.Write([]byte(tt.requestBody))
				require.NoError(t, err)
				err = gzw.Close()
				require.NoError(t, err)
			} else {
				_, err := io.WriteString(&buf, tt.requestBody)
				require.NoError(t, err)
			}

			request := httptest.NewRequest("POST", testSrv.URL, &buf)
			request.RequestURI = ""
			request.Header.Set("Content-Type", tt.contentType)
			request.Header.Set("Accept-Encoding", tt.acceptEncoding)

			response, err := http.DefaultClient.Do(request)
			require.NoError(t, err)
			require.Equal(t, tt.expectedStatus, response.StatusCode)

			if tt.expectedEncoding != "" {
				require.Equal(t, tt.expectedEncoding, response.Header.Get("Content-Encoding"))
			} else {
				require.Empty(t, response.Header.Get("Content-Encoding"))
			}

			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			if tt.expectedEncoding == "gzip" {
				gzr, err := gzip.NewReader(bytes.NewReader(body))
				require.NoError(t, err)
				body, err = io.ReadAll(gzr)
				require.NoError(t, err)
			}

			require.Equal(t, tt.expectedResponse, strings.TrimPrefix(string(body), "http://localhost:8080/"))
		})
	}
}

func TestGzipDecompressorMiddleware(t *testing.T) {
	storage := MockStorage{mockData: map[string]string{"shortURL": "https://example.com"}}
	testApp := handlers.NewAppHandler(&storage)

	mockHandler := GzipDecompressorMiddleware(http.HandlerFunc(testApp.PostHandler))
	testSrv := httptest.NewServer(mockHandler)
	defer testSrv.Close()

	testTable := []struct {
		name             string
		contentType      string
		acceptEncoding   string
		requestBody      string
		expectedStatus   int
		expectedEncoding string
		expectedResponse string
	}{
		{name: "Sends gzip and Content-Encoding is gzip", contentType: "text/html",
			acceptEncoding: "gzip", requestBody: "https://example.com",
			expectedStatus: http.StatusCreated, expectedEncoding: "gzip",
			expectedResponse: "shortURL",
		},
		{name: "Sends gzip, but Content-Encoding is not gzip", contentType: "text/html",
			acceptEncoding: "", requestBody: "https://example.com",
			expectedStatus: http.StatusCreated, expectedEncoding: "",
			expectedResponse: "shortURL",
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.Buffer{}

			if tt.expectedEncoding == "gzip" {
				gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
				require.NoError(t, err)
				_, err = gz.Write([]byte(tt.requestBody))
				require.NoError(t, err)
				err = gz.Close()
				require.NoError(t, err)
			} else {
				_, err := io.WriteString(&buf, tt.requestBody)
				require.NoError(t, err)
			}

			request := httptest.NewRequest("POST", testSrv.URL, &buf)
			request.RequestURI = ""
			request.Header.Set("Content-Encoding", tt.expectedEncoding)
			request.Header.Set("Content-Type", tt.contentType)

			response, err := http.DefaultClient.Do(request)
			require.NoError(t, err)
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			require.Equal(t, tt.expectedResponse, strings.TrimPrefix(string(body), "http://localhost:8080/"))
			require.Equal(t, http.StatusCreated, response.StatusCode)
		})
	}
}
