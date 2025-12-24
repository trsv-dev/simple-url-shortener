package logger

import (
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// Log Никакой код, кроме функции InitLogger, не должен модифицировать эту переменную.
// По умолчанию установлен no-op-логгер, который не выводит никаких сообщений.
var Log *zap.Logger = zap.NewNop()

// InitLogger Инициализирует синглтон логгера с необходимым уровнем логирования.
func InitLogger(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	// создаём новую конфигурацию логгера
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	// Можно включить / выключить трейсбек
	cfg.DisableStacktrace = true

	// создаём логгер на основе конфигурации
	logger, err := cfg.Build()
	if err != nil {
		return err
	}

	// устанавливаем синглтон
	Log = logger
	return nil
}

// responseData хранит метаданные HTTP-ответа,
// необходимые для логирования.
type responseData struct {
	status int
	size   int
}

// loggingResponseWriter — обёртка над http.ResponseWriter,
// позволяющая перехватывать код статуса и размер ответа
// для последующего логирования.
type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

// Write Проксирует запись тела ответа в оригинальный ResponseWriter
// и накапливает количество записанных байт.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	// захватываем размер
	r.responseData.size += size

	return size, err
}

// WriteHeader проксирует установку HTTP-статуса
// и сохраняет его для логирования.
//
// Если WriteHeader не был вызван явно,
// стандартный http.Server использует статус 200.
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWrite
	r.ResponseWriter.WriteHeader(statusCode)
	// захватываем код статуса
	r.responseData.status = statusCode
}

// RequestAndResponseLogger Логирование запросов и ответов.
func RequestAndResponseLogger(h http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {

		data := &responseData{
			status: 0,
			size:   0,
		}

		lw := loggingResponseWriter{
			// встраиваем оригинальный http.ResponseWriter
			ResponseWriter: w,
			responseData:   data,
		}

		start := time.Now()

		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		Log.Debug("Got incoming HTTP request",
			zap.String("uri", r.RequestURI),
			zap.String("method", r.Method),
			zap.String("status", strconv.Itoa(data.status)),
			zap.String("duration", duration.String()),
			zap.String("size", strconv.Itoa(data.size)),
		)
	}

	return http.HandlerFunc(f)
}

// IsValidLogLevel Функция, которая проверяет валидность введенного уровня логирования.
func IsValidLogLevel(logLevel string) bool {
	level := map[string]bool{
		"Debug":  true,
		"Info":   true,
		"Warn":   true,
		"Error":  true,
		"DPanic": true,
		"Panic":  true,
		"Fatal":  true,
	}

	return level[logLevel]
}
