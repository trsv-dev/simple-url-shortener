package handlers

import (
	"fmt"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
	"io"
	"net/http"
	"strings"
)

const serverHost = "127.0.0.1:8080"

// PostHandler Функция обработки запроса к "/".
func PostHandler(storage storage.Storage, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer r.Body.Close()
	rawURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "ошибка чтения тела запроса", http.StatusInternalServerError)
		return
	}

	longURL := string(rawURL)
	shortURL, err := storage.SetInStorage(longURL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	answer := fmt.Sprintf("http://%s/%s", serverHost, shortURL)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(answer))
}

// GetHandler Функция обработки запроса к "/{id}".
func GetHandler(storage storage.Storage, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")

	value, err := storage.GetFromStorage(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", value)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
