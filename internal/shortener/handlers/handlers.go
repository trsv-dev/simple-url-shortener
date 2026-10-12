package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/trsv-dev/simple-url-shortener/internal/shortener/config"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/errs"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/logger"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/models"
	"github.com/trsv-dev/simple-url-shortener/internal/shortener/storage"
	"go.uber.org/zap"
)

// AppHandler Структура для работы с зависимостями HTTP хендлеров.
type AppHandler struct {
	storage   storage.Storage
	SecretKey string
	TokenExp  time.Duration
}

// NewAppHandler Конструктор AppHandler.
func NewAppHandler(s storage.Storage, secretKey string, tokenExp time.Duration) *AppHandler {
	return &AppHandler{storage: s, SecretKey: secretKey, TokenExp: tokenExp}
}

// PostHandler Функция обработки запроса к "/".
func (h *AppHandler) PostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	shortURL, err := h.storage.SetInStorage(ctx, longURL)
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
	ctx := r.Context()

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")

	value, err := h.storage.GetFromStorage(ctx, id)
	var errURLDeleted *errs.ErrURLDeleted

	if err != nil {
		switch {
		case errors.As(err, &errURLDeleted):
			w.WriteHeader(http.StatusGone)
			return
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Location", value)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// APIShorten Функция обработки запроса к "/api/shorten"
func (h *AppHandler) APIShorten(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
	shortURL, err := h.storage.SetInStorage(ctx, request.URL)
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
	ctx := r.Context()

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := h.storage.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Batch Функция обработки пакетных POST-запросов в формате JSON к /api/shorten/batch.
func (h *AppHandler) Batch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	err = h.storage.SetBatchInStorage(ctx, batch)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	data, err := json.Marshal(&batch)
	if err != nil {
		http.Error(w, "Ошибка получения структуры ответа", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

// UserURLs Функция обработки GET-запроса к /api/user/urls на получение всех сокращенных пользователем URL.
func (h *AppHandler) UserURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	urls, err := h.storage.GetUserURLs(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(urls)
	if err != nil {
		logger.Log.Debug("Ошибка маршалинга URL-ов пользователя", zap.String("rows error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// Delete Функция обработки DELETE-запроса к /api/user/urls, которая в теле запроса
// принимает список идентификаторов сокращённых URL для асинхронного удаления.
func (h *AppHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения запроса", http.StatusInternalServerError)
		return
	}

	// из хендлера возвращается JSON со списком коротких URL для удаления
	var URLs []string

	err = json.Unmarshal(body, &URLs)
	if err != nil {
		http.Error(w, "Ошибка декодирования json", http.StatusInternalServerError)
		return
	}

	err = h.storage.MarkAsDelete(r.Context(), URLs)

	if err != nil {
		logger.Log.Info("Ошибка при удалении URL", zap.String("delete url error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// Stats Функция обработки GET-запроса к /api/internal/stats для получения статистики
// (общее кол-во пользователей и общее кол-во сохраненных ссылок).
// Доступ только из доверенной подсети, задающейся через переменную окружения TRUSTED_SUBNET или флаг -t,
// например, 127.0.0.0/8 или -t 192.168.1.0/24.
// При пустом значении доступ запрещён для любого входящего запроса.
func (h *AppHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// из конфига сервера получаем доверенные маски подсети
	trustedSubnet := config.GetServerConfig().TrustedSubnet
	if trustedSubnet == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// получаем X-Real-IP из заголовка запроса
	ipStr := r.Header.Get("X-Real-IP")

	// парсим его
	ip := net.ParseIP(ipStr)
	if ip == nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// парсим доверенные маски
	_, subnet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		logger.Log.Error("Некорректная доверенная подсеть", zap.String("subnet", trustedSubnet), zap.Error(err))
		http.Error(w, "Ошибка конфигурации", http.StatusInternalServerError)
		return
	}

	// проверяем, входит ли ip в доверенную подсеть
	if !subnet.Contains(ip) {
		logger.Log.Debug("Попытка доступа к статистике с недопустимого адреса", zap.String("remote address", ip.String()))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	urlsStat, err := h.storage.GetURLsStat(r.Context())
	if err != nil {
		http.Error(w, "Ошибка получения статистики по URL-ам", http.StatusInternalServerError)
		return
	}

	usersStat, err := h.storage.GetUsersStat(r.Context())
	if err != nil {
		http.Error(w, "Ошибка получения статистики по пользователям", http.StatusInternalServerError)
		return
	}

	stat := models.Stats{URLs: urlsStat, Users: usersStat}

	data, err := json.Marshal(&stat)
	if err != nil {
		logger.Log.Debug("Ошибка маршалинга статистики", zap.String("rows error", err.Error()))
		http.Error(w, "Ошибка маршалинга статистики", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
