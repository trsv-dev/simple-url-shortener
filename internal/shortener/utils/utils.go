package utils

import (
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"time"
)

const latin = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890"

var random *rand.Rand

// Инициализируем структуру для хранения мап соответствия адресов.
func init() {
	source := rand.NewSource(time.Now().UnixNano())
	random = rand.New(source)
}

// MakeShortURL Создание рандомной строки заданной длины из константного набора символов.
func MakeShortURL(length int) string {
	var str string
	for i := 0; i < length; i++ {
		str += string(latin[random.Intn(len(latin))])
	}
	return str
}

// IsValidURL Проверка переданного URL на валидность.
func IsValidURL(longURL string) bool {
	pattern := `^https?://([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(/[^\s]*)?$`
	matched, _ := regexp.MatchString(pattern, longURL)
	return matched
}

// GetHostAndPort Получение хоста и порта.
func GetHostAndPort(w http.ResponseWriter, r *http.Request) (string, string) {
	host, port, _ := net.SplitHostPort(r.Host)
	return host, port
}
