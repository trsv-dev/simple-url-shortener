package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
	"io"
	"net/http"
	"strings"
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
	baseURL := config.GetServerConfig().BaseURL

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
	var conflictErr *errs.ErrLongURLConflict
	if err != nil {
		switch {
		case errors.As(err, &conflictErr):
			answer := baseURL + "/" + conflictErr.ShortURL
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(answer))
			return
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
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
	baseURL := config.GetServerConfig().BaseURL

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
	var conflictErr *errs.ErrLongURLConflict
	if err != nil {
		switch {
		case errors.As(err, &conflictErr):
			// если URL уже существует в базе - возвращаем через структуру кастомной ошибки
			// shortURL, который соответствует этому URL
			answer := baseURL + "/" + conflictErr.ShortURL
			response := models.Response{Result: answer}
			result, err := json.Marshal(response)
			if err != nil {
				http.Error(w, "Ошибка кодирования ответа", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			w.Write(result)
			return
		case err.Error() == "передан невалидный URL":
			http.Error(w, "Некорректный формат URL", http.StatusBadRequest)
			w.WriteHeader(http.StatusBadRequest)
			return
		default:
			http.Error(w, "Ошибка обработки запроса", http.StatusInternalServerError)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// формируем URL, добавляем к полученному короткому URL адрес сервера
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

// Ping При GET-запросе к "/ping" проверяет соединение с базой данных
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

// Batch Функция обработки пакетных POST-запросов в формате JSON к /api/shorten/batch.
func (h *AppHandler) Batch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// создаем слайс структур batch
	batch := make([]models.BatchItem, 0)

	// читаем тело запроса
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения запроса", http.StatusInternalServerError)
		return
	}

	// анмаршаллим тело запроса прямо в слайс структур
	err = json.Unmarshal(body, &batch)
	if err != nil {
		http.Error(w, "Невалидная структура запроса", http.StatusBadRequest)
		return
	}

	// проверяем элементы batch на пустые поля
	if len(batch) == 0 {
		http.Error(w, "Батчи не могут быть пустыми", http.StatusBadRequest)
		return
	}

	err = h.storage.SetBatchInStorage(batch)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	data, err := json.Marshal(&batch)
	if err != nil {
		http.Error(w, "ошибка получения структуры ответа", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}
