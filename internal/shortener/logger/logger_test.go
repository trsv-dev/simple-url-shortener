package logger

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

func TestIsValidLogLevel(t *testing.T) {
	supportedLogLevel := "Debug"
	unsupportedLogLevel := "Alarm"

	assert.Equal(t, true, IsValidLogLevel(supportedLogLevel))
	assert.Equal(t, false, IsValidLogLevel(unsupportedLogLevel))
}

func TestInitLogger(t *testing.T) {
	testTable := []struct {
		name        string
		logLevel    string
		expectError bool
	}{
		{name: "Initial logger with debug log level", logLevel: "Debug", expectError: false},
		{name: "Initial logger with invalid log level", logLevel: "Invalid", expectError: true},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			err := InitLogger(tt.logLevel)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestRequestAndResponseLogger(t *testing.T) {
	// Создаём буфер для записи логов
	var buf bytes.Buffer

	mockCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf), // Подключаем буфер как writer
		zap.DebugLevel,
	)

	// Подменяем глобальный логгер
	mockLogger := zap.New(mockCore)
	originalLogger := Log
	Log = mockLogger
	defer func() {
		Log = originalLogger
	}()

	// Создаём тестовый хендлер
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Создаём запрос и записываем ответ через наш middleware
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	middleware := RequestAndResponseLogger(handler)
	middleware.ServeHTTP(w, r)

	// Проверяем, что логгер записал ожидаемые данные
	logOutput := buf.String()
	require.NotEmpty(t, logOutput)

	// Пример проверки ключевых данных в логах
	assert.Contains(t, logOutput, `"method":"GET"`)
	assert.Contains(t, logOutput, `"uri":"/test"`)
	assert.Contains(t, logOutput, `"status":"200"`)
	assert.Contains(t, logOutput, `"size":"13"`) // Длина ответа "test response"
}
