package gzipper

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipWriter — структура-обертка над http.ResponseWriter,
// переопределяющая метод Write для записи сжатых данных в поток.
// Используется в GzipCompressorMiddleware для автоматического сжатия ответов сервера.
type gzipWriter struct {
	http.ResponseWriter           // Встраивание стандартного ResponseWriter для сохранения его методов
	Writer              io.Writer // Поток для записи сжатых данных (gzip.Writer)
}

// Write записывает сжатые данные в поток gzip.
func (gz gzipWriter) Write(b []byte) (int, error) {
	return gz.Writer.Write(b)
}

// GzipCompressorMiddleware Middleware для сжатия ответа сервера, если клиент поддерживает сжатие ("Accept-Encoding" = "gzip").
func GzipCompressorMiddleware(h http.Handler) http.Handler {
	gzipCompressed := func(w http.ResponseWriter, r *http.Request) {
		// если клиент не поддерживает сжатие - обрабатываем запрос как обычно
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.ServeHTTP(w, r)
			return
		}
		// если Content-Type не совпадает с нужным - обрабатываем запрос как обычно
		if !(r.Header.Get("Content-Type") == "application/json" ||
			r.Header.Get("Content-Type") == "application/gzip" ||
			r.Header.Get("Content-Type") == "text/html") {
			h.ServeHTTP(w, r)
			return
		}
		// если клиент поддерживает приём сжатого ответа сервера и Content-Type в запросе совпадает с нужным -
		// сжимаем ответ и обрабатываем его с помощью переопределенного Writer-а, который наследует
		// методы http.ResponseWriter, кроме метода Write
		gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gz.Close()

		// Устанавливаем заголовок "Content-Encoding: gzip", чтобы гарантировать, что
		// он будет установлен до любого вызова WriteHeader() в хендлерах,
		// тем самым предотвращая возможные ошибки при обработке ответов.
		w.Header().Set("Content-Encoding", "gzip")
		// подменяем 'w' на 'gzipWriter{ResponseWriter: w, Writer: gz}'
		h.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	}

	return http.HandlerFunc(gzipCompressed)
}

// GzipDecompressorMiddleware распаковывает gzip-сжатые запросы.
// Если клиент отправил сжатый запрос, Middleware декомпрессирует его перед обработкой.
func GzipDecompressorMiddleware(h http.Handler) http.Handler {
	gzipDecompressed := func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			h.ServeHTTP(w, r)
			return
		}

		gzdec, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gzdec.Close()

		r.Body = gzdec

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(gzipDecompressed)
}
