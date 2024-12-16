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

const latin = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const length = 8
const serverAddress = "127.0.0.1"
const serverPort = "8080"

var URLs urlStorage
var random *rand.Rand

// Инициализируем структуру для хранения мапы соответствия "длинных адресов" и "коротких адресов"
func init() {
	source := rand.NewSource(time.Now().UnixNano())
	random = rand.New(source)

	URLs = urlStorage{addresses: make(map[string]string)}
}

// Структура для хранения мапы соответствия "длинных адресов" и "коротких адресов"
type urlStorage struct {
	addresses map[string]string
}

func main() {
	if err := runServer(); err != nil {
		panic(err)
	}
}

// Проверка корректности переданного URL, создание соответствия переданного URL и короткого URL,
// возврат короткого URL
func (u *urlStorage) createAndGetShortURL(address []byte) (string, error) {
	if isValidURL(address) {
		if value, ok := u.addresses[string(address)]; ok {
			return value, nil
		}
		u.addresses[string(address)] = makeShortURL(length)
		return u.addresses[string(address)], nil
	}
	return "", fmt.Errorf("передан невалидный url")
}

// Проверка переданного URL на валидность
func isValidURL(address []byte) bool {
	pattern := `^https?://([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(/[^\s]*)?$`
	parsedURL := string(address)

	matched, _ := regexp.MatchString(pattern, parsedURL)
	return matched
}

// Создание рандомной строки заданной длины из константного набора символов
func makeShortURL(length int) string {
	var str string
	for i := 0; i < length; i++ {
		str += string(latin[random.Intn(len(latin))])
	}
	return str
}

// Обработка маршрутов и запуск сервера
func runServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", postHandler)
	mux.HandleFunc("/{id}", getHandler)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
	return nil
}

// Функция обработки запроса к "/"
func postHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	address, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения тела запроса", http.StatusInternalServerError)
		return
	}

	shortURL, err := URLs.createAndGetShortURL(address)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	answer := fmt.Sprintf("http://%s:%s/%s", serverAddress, serverPort, shortURL)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(answer))
}

// Функция обработки запроса к "/{id}"
func getHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")

	for k, v := range URLs.addresses {
		if v == id {
			w.Header().Set("Location", k)
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
	}

	w.WriteHeader(http.StatusBadRequest)
	return
}
