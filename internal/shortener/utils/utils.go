package utils

import (
	"errors"
	"math/rand"
	"regexp"
	"time"
)

var latin = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890"
var random *rand.Rand

// Инициализируем структуру для хранения мап соответствия адресов.
func init() {
	source := rand.NewSource(time.Now().UnixNano())
	random = rand.New(source)
}

// MakeShortURL Создание рандомной строки заданной длины из константного набора символов.
func MakeShortURL(length int) (string, error) {
	if length < 5 {
		return "", errors.New("переданная длина слишком мала для генерации URL")
	}
	var str string
	for i := 0; i < length; i++ {
		str += string(latin[random.Intn(len(latin))])
	}
	return str, nil
}

// IsValidURL Проверка переданного URL на валидность (URL не пустой и соответствует требованиям к URL).
func IsValidURL(longURL string) bool {
	pattern := `^https?://([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(/[^\s]*)?$`
	matched, _ := regexp.MatchString(pattern, longURL)
	return matched
}
