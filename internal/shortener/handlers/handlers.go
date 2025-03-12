package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
)

// AppHandler Структура для работы с зависимостями хендлеров.
type AppHandler struct {
	storage storage.Storage
}

// NewAppHandler Конструктор AppHandler
func NewAppHandler(s storage.Storage) *AppHandler {
	return &AppHandler{storage: s}
}

// PostHandler Функция обработки запроса к "/".
func (h *AppHandler) PostHandler(w http.ResponseWriter, r *http.Request) {
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
	shortURL, err := h.storage.SetInStorage(longURL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	baseURL := config.GetServerConfig().BaseURL
	answer := fmt.Sprint(baseURL + "/" + shortURL)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(answer))
}

// GetHandler Функция обработки запроса к "/{id}".
func (h *AppHandler) GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")

	value, err := h.storage.GetFromStorage(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", value)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// APIShorten Функция обработки запроса к "/api/shorten"
func (h *AppHandler) APIShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	request := models.Request{}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения запроса", http.StatusInternalServerError)
		return
	}

	// анмаршалим тело запроса в структуру модели запроса
	err = json.Unmarshal(body, &request)
	if err != nil {
		http.Error(w, "Невалидная структура запроса", http.StatusBadRequest)
		return
	}

	// помещаем полученный в запросе длинный URL в хранилище
	shortURL, err := h.storage.SetInStorage(request.URL)
	if err != nil {
		if err.Error() == "передан невалидный URL" {
			http.Error(w, "Некорректный формат URL", http.StatusBadRequest)
		} else {
			http.Error(w, "Ошибка обработки запроса", http.StatusInternalServerError)
		}

		return
	}

	// формируем URL, добавляем к полученному короткому URL адрес сервера
	baseURL := config.GetServerConfig().BaseURL
	answer := fmt.Sprint(baseURL + "/" + shortURL)

	response := models.Response{Result: answer}

	// маршалим полученный URL (с адресом сервера) в структуру модели ответа
	result, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Ошибка кодирования ответа", http.StatusInternalServerError)
		return
	}

	// устанавливаем заголовки, возвращаем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(result)
}

// Ping При запросе проверяет соединение с базой данных
func (h *AppHandler) Ping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := h.storage.Ping(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
