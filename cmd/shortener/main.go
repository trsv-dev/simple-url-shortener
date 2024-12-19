package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const latin = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890"
const length = 8
const serverAddress = "127.0.0.1"
const serverPort = "8080"

var URLs urlStorage
var random *rand.Rand

// Инициализируем структуру для хранения мап соответствия адресов.
func init() {
	source := rand.NewSource(time.Now().UnixNano())
	random = rand.New(source)
}

// Storage интерфейс определяет методы для управления хранилищем URL.
// Он предоставляет операции для сохранения длинного URL и получения
// соответствующего длинного URL по короткому. Реализации этого
// интерфейса должны обеспечивать необходимую логику для хранения
// и поиска URL, а также обрабатывать возможные ошибки.
type Storage interface {
	setInStorage(longURL string) (string, error)
	getFromStorage(shortURL string) (string, error)
}

func main() {

	URLs = urlStorage{
		id:          0,
		longToShort: make(map[string]string),
		shortToLong: make(map[string]string),
	}

	if err := runServer(&URLs); err != nil {
		panic(err)
	}
}

// Структура для хранения двух мап:
// - longToShort: сопоставление "длинного адреса" с "коротким адресом".
// - shortToLong: сопоставление "короткого адреса" с "длинным адресом".
// Используется для обеспечения быстрого поиска соответствий с временной сложностью O(1).

type urlStorage struct {
	id          int
	longToShort map[string]string
	shortToLong map[string]string
}

// Проверка корректности переданного URL, создание соответствия переданного URL и короткого URL,
// возврат короткого URL.
func (u *urlStorage) setInStorage(longURL string) (string, error) {
	if isValidURL(longURL) {
		if value, ok := u.longToShort[longURL]; ok {
			return value, nil
		}

		shortURL := makeShortURL(u, length)
		u.longToShort[longURL], u.shortToLong[shortURL] = shortURL, longURL
		u.id++

		return u.longToShort[longURL], nil
	}
	return "", fmt.Errorf("передан невалидный url")
}

// Поиск и возврат найденного длинного URL по переданному короткому URL.
func (u *urlStorage) getFromStorage(shortURL string) (string, error) {
	if value, ok := u.shortToLong[shortURL]; ok {
		return value, nil
	}

	return "", fmt.Errorf("`%v` не найден в базе данных", shortURL)
}

// Создание рандомной строки заданной длины из константного набора символов.
func makeShortURL(storage Storage, length int) string {
	for {
		var str string
		for i := 0; i < length; i++ {
			str += string(latin[random.Intn(len(latin))])
		}
		// проверяем, что сгенерированная последовательность еще не использовалась.
		if _, err := storage.getFromStorage(str); err != nil {
			return str
		}
	}
}

// Проверка переданного URL на валидность.
func isValidURL(longURL string) bool {
	pattern := `^https?://([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(/[^\s]*)?$`
	matched, _ := regexp.MatchString(pattern, longURL)
	return matched
}

// Обработка маршрутов и запуск сервера.
func runServer(storage Storage) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		postHandler(storage, w, r)
	})
	mux.HandleFunc("/{id}", func(w http.ResponseWriter, r *http.Request) {
		getHandler(storage, w, r)
	})

	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
	return nil
}

// Функция обработки запроса к "/".
func postHandler(storage Storage, w http.ResponseWriter, r *http.Request) {
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
	shortURL, err := storage.setInStorage(longURL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	answer := fmt.Sprintf("http://%s:%s/%s", serverAddress, serverPort, shortURL)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(answer))
}

// Функция обработки запроса к "/{id}".
func getHandler(storage Storage, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")

	value, err := storage.getFromStorage(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	w.Header().Set("Location", value)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
