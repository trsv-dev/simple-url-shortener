package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/utils"
)

// PostHandler Функция обработки запроса к "/".
func PostHandler(storage storage.Storage, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rawURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения тела запроса", http.StatusInternalServerError)
		return
	}

	longURL := string(rawURL)
	shortURL, err := storage.SetInStorage(longURL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	host, port := utils.GetHostAndPort(w, r)
	answer := fmt.Sprintf("http://%s:%s/%s", host, port, shortURL)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(answer))
}

// GetHandler Функция обработки запроса к "/{id}".
func GetHandler(storage storage.Storage, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
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
